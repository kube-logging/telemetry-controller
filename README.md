# telemetry-controller
The Telemetry Controller is a multi-tenancy focused solution, that facilitates collection of telemetry data from Kubernetes workloads, without any need for changes to the running software.
## Description
Telemetry-controller can be configured using Custom Resources to set up an opinionated Opentelemetry Collector configuration to route log messages based on rules defined as a Tenant -> Subscription relation map.
## Getting Started

### Prerequisites
- go version v1.20.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Deployment steps for users

**Install cert-manager, and opentelemtry-operator:**
```sh
helm upgrade --install --repo https://charts.jetstack.io cert-manager cert-manager --namespace cert-manager --create-namespace --version v1.13.3 --set installCRDs=true --wait

kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml --wait
```

**Deploy telemetry-controller:**
```sh
kubectl apply -k 'github.com/kube-logging/telemetry-controller/config/default'
```
**Remove the controller and CRDs from the cluster:**
```sh
kubectl delete -k 'github.com/kube-logging/telemetry-controller/config/default'
```

### Deployment steps for contributors
**Install cert-manager, and opentelemtry-operator:**
```sh
helm upgrade --install --repo https://charts.jetstack.io cert-manager cert-manager --namespace cert-manager --create-namespace --version v1.13.3 --set installCRDs=true --wait

kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml --wait
```

**Install the CRDs into the cluster:**

```sh
make install
```

**Build your image and load it to your KinD nodes**
```sh
make docker-build IMG=telemetry-controller:latest
kind load docker-image telemetry-controller:latest
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=telemetry-controller:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -f docs/examples/simple-demo/
```

**Remove the controller from the cluster:**

```sh
make undeploy
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```
### Example setup
**Create instances of your solution**
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
![Openobserve logs](docs/assets/openobserve-logs.png)


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

