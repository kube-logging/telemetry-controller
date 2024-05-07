#!/usr/bin/env bash

set -eou pipefail

create_if_does_not_exist() {
  local resource_type=$1
  local resource_name=$2
  kubectl create  "${resource_type}" "${resource_name}" --dry-run=client -o yaml | kubectl apply -f-
}

KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME_E2E:-so-e2e}
NO_KIND_CLEANUP=${NO_KIND_CLEANUP:-}
CI_MODE=${CI_MODE:-}
  # Backup current kubernetes context
CURRENT_K8S_CTX=$(kubectl config view | grep "current" | cut -f 2 -d : | xargs)

if GOOS="darwin"
then
  TIMEOUT_CMD=gtimeout
else
  TIMEOUT_CMD=timeout
fi



# Prepare env
kind create cluster --name "${KIND_CLUSTER_NAME}" --wait 5m
kubectl config set-context kind-"${KIND_CLUSTER_NAME}"

# Install prerequisites

helm upgrade \
  --install \
  --repo https://charts.jetstack.io \
  cert-manager cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.13.3 \
  --set installCRDs=true \
  --wait

kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.96.0/opentelemetry-operator.yaml --wait
echo "Wait until otel operator pod is in ready state..."
kubectl wait --namespace opentelemetry-operator-system --for=condition=available deployment/opentelemetry-operator-controller-manager --timeout=300s

# Create subscription operator resources
(cd .. && make manifests generate install)

# Use example
kubectl apply -f ../e2e/testdata/one_tenant_two_subscriptions

if [[ -z "${CI_MODE}" ]]; then
  $(cd .. && $(TIMEOUT_CMD) 5m make run &)
else
  kind load docker-image controller:latest --name "${KIND_CLUSTER_NAME}"
  cd .. && make deploy && cd -
fi

# Create log-generator
helm install --wait --create-namespace --namespace example-tenant-ns --generate-name oci://ghcr.io/kube-logging/helm-charts/log-generator


# Check for received messages - subscription-sample
while
  echo "Checking for subscription-sample-1 in deployments/receiver-collector logs"
  kubectl logs --namespace example-tenant-ns deployments/receiver-collector | grep -q "subscription-sample-1"
  
  [[ $? -ne 0 ]]
do true; done

# Check for received messages - subscription-sample-2
while
  echo "Checking for subscription-sample-2 in deployments/receiver-collector logs"
  kubectl logs --namespace example-tenant-ns deployments/receiver-collector | grep -q "subscription-sample-2"

  [[ $? -ne 0 ]]
do true; done

echo "E2E test: PASSED"


# Check if cluster should be removed, ctx restored
if [[ -z "${NO_KIND_CLEANUP}" ]]; then
  kind delete cluster --name "${KIND_CLUSTER_NAME}"
fi
