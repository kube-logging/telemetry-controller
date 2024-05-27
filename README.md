# Telemetry Controller

Telemetry Controller collects, routes and forwards telemetry data (logs, metrics and traces) from Kubernetes clusters
supporting multi-tenancy out of the box.

## Description

Telemetry-controller can be configured using Custom Resources to set up an opinionated Opentelemetry Collector configuration to route log messages based on rules defined as a Tenant -> Subscription relation map.

## Getting Started

To get started with the Telemetry Controller, complete the following steps. Alternatively, see our [Telemetry Controller overview and quickstart blog post](https://axoflow.com/reinvent-kubernetes-logging-with-telemetry-controller/).

### Prerequisites
- go version v1.22+
- docker version 24+
- kubectl version v1.26+
- kubernetes v1.26+ with *containerd* as the container runtime

### Optional: create a cluster locally

We recommend using kind or minikube for local experimentation and development.

Kind uses containerd by default, but for minikube you have to start the cluster using the `--container-runtime=containerd` flag.

```sh
kind create cluster
# or
minikube start --container-runtime=containerd
```

### Deployment steps for users

Install dependencies (cert-manager and opentelemetry-operator):
```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.4/cert-manager.yaml
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.98.0/opentelemetry-operator.yaml
```

Deploy latest telemetry-controller:
```sh
kubectl apply -k github.com/kube-logging/telemetry-controller/config/default --server-side
```

### Deployment steps for devs

#### Install deps, CRDs and RBAC

```sh
# Install dependencies (cert-manager and opentelemtry-operator):
make install-deps

# Install the CRDs and RBAC into the cluster:
make install
```

#### Run

```sh
# Option 1 (faster): Run the operator from you local machine (uses cluster-admin rights)
make run

# Option 2 (safer): Build and run the operator inside the cluster (uses proper RBAC)
make docker-build IMG=telemetry-controller:latest

kind load docker-image telemetry-controller:latest
# or
minikube image load telemetry-controller:latest

make deploy IMG=telemetry-controller:latest
```

### Example setup

You can deploy the example configuration provided as part of the docs. This will deploy a demo pipeline with one tenant, two subscriptions, and an OpenObserve instance.
Deploying Openobserve is an optional, but recommended step, logs can be forwarded to any OTLP endpoint. Openobserve provides a UI to visualize the ingested logstream.

```sh
# Deploy Openobserve
kubectl apply -f docs/examples/simple-demo/openobserve.yaml

# Set up portforwarding for Openobserve UI
kubectl -n openobserve port-forward svc/openobserve 5080:5080 &
```

Open the UI at `localhost:5080`, navigate to the `Ingestion/OTEL Collector` tab, and copy the authorization token as seen on the screenshot.
![Openobserve auth](docs/assets/openobserve-auth.png)

Paste this token to the example manifests:
```sh
sed -i '' -e "s/\<TOKEN\>/INSERT YOUR COPIED TOKEN HERE/" docs/examples/simple-demo/one_tenant_two_subscriptions.yaml
```
```sh
# Deploy the pipeline definition
kubectl apply -f docs/examples/simple-demo/one_tenant_two_subscriptions.yaml
```

**Create a workload, which will generate logs for the pipeline:**
```sh
helm install --wait --create-namespace --namespace example-tenant-ns --generate-name oci://ghcr.io/kube-logging/helm-charts/log-generator
```

**Open the Openobserve UI and inspect the generated log messages**

Set up portforwarding for Openobserve UI
```sh
kubectl -n openobserve port-forward svc/openobserve 5080:5080
```
![Openobserve logs](docs/assets/openobserve-logs.png)

### Sending logs to logging-operator (example)

Install dependencies (cert-manager and opentelemetry-operator):

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.4/cert-manager.yaml
```

```sh
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.98.0/opentelemetry-operator.yaml
# Wait for the opentelemtry-operator to be running
kubectl wait --namespace opentelemetry-operator-system --for=condition=available deployment/opentelemetry-operator-controller-manager --timeout=300s
```

Deploy latest telemetry-controller:

```sh
kubectl apply -k github.com/kube-logging/telemetry-controller/config/default --server-side
```

Install logging-operator

```sh
helm upgrade --install logging-operator oci://ghcr.io/kube-logging/helm-charts/logging-operator --version=4.6.0 -n logging-operator --create-namespace
```

Install log-generator

```sh
helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n log-generator --create-namespace
```

Apply the provided example resource for logging-operator: [logging-operator.yaml](./docs/examples/fluent-forward/logging-operator.yaml)

```sh
kubectl apply -f logging-operator.yaml
```

Apply the provided example resource for telemetry-controller: [telemetry-controller.yaml](./docs/examples/fluent-forward/telemetry-controller.yaml)

```sh
kubectl apply -f telemetry-controller.yaml
```


## Contributing

If you find this project useful, help us:

- Support the development of this project and star this repo! :star:
- Help new users with issues they may encounter :muscle:
- Send a pull request with your new features and bug fixes :rocket: 

Please read the [Organisation's Code of Conduct](https://github.com/kube-logging/.github/blob/main/CODE_OF_CONDUCT.md)!

*For more information, read our organization's [contribution guidelines](https://github.com/kube-logging/.github/blob/main/CONTRIBUTING.md)*.

## License

Copyright © 2024 Kube logging authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
