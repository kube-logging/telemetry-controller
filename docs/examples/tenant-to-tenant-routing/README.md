# Tenant-to-Tenant Log Routing

This guide demonstrates how to implement cross-tenant log routing using the `Telemetry Controller` in a Kubernetes cluster. You'll learn how to route specific logs from a shared tenant to other tenants based on HTTP verbs.

## Prerequisites

- `kubectl` installed and configured
- One of the following:
  - `kind`
  - `minikube` with containerd runtime
- `helm`

## Architecture Overview

This setup implements the following architecture:

- Three distinct tenants: `shared`, `database`, and `web`
- Each tenant collects logs from its corresponding namespace
- The logs from the shared tenant will get routed following these rules:
  - HTTP `GET` requests → `database` tenant
  - HTTP `PUT` requests → `web` tenant
- All tenants forward logs to a central `OpenObserve` instance

## Step-by-Step Installation

### 1. Create a local cluster

You can create a Kubernetes cluster using either `kind` or `minikube`.

```bash
kind create cluster
# OR
minikube start --container-runtime=containerd
```

### 2. Set Up namespaces

Create the required namespaces:

```bash
kubectl apply -f namespaces.yaml
```

### 3. Deploy OpenObserve

Deploy OpenObserve for log aggregation:

```bash
# Deploy the core OpenObserve components
kubectl apply -f https://raw.githubusercontent.com/zinclabs/openobserve/main/deploy/k8s/statefulset.yaml

# Deploy the OpenObserve service
kubectl apply -f openobserve-svc.yaml
```

### 4. Create resources

**Before proceeding, make sure to update the authentication tokens in the `pipeline.yaml` file.**

_NOTE: You can check out this [example](../../../README.md#example-setup) on how to retrieve your token._

Apply the pipeline configuration:

```bash
kubectl apply -f pipeline.yaml
```

This creates:

- A Collector that will collect logs.
- Three tenant configurations (`shared`, `database`, `web`)
- Tenant-specific log collection rules via the `transform-processor`
- Routing achieved via the [Bridge](../../../config/crd/bases/telemetry.kube-logging.dev_bridges.yaml) custom-resource, based on HTTP verbs.

### 5. Deploy the log generator

Deploy the log generator in the shared namespace using Helm:

```bash
helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n shared
```

### 6. Access the Logs

Set up port-forwarding to access OpenObserve:

```bash
kubectl port-forward -n openobserve svc/openobserve 8080:5080 > /dev/null &
```

### 7. Verification

Access the logs by navigating to the OpenObersve UI:

<http://localhost:8080>

Every log message that had the `GET` verb in them was routed to the `database` tenant, while the ones that had `PUT` got routed to `web`.

## Cleanup

Simply teardown the cluster:

```bash
kind delete cluster
# OR 
minikube delete
```
