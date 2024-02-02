#!/bin/env bash

set -eou pipefail

create_if_does_not_exist() {
  local resource_type=$1
  local resource_name=$2
  kubectl create  "${resource_type}" "${resource_name}" --dry-run=client -o yaml | kubectl apply -f-
}

KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME_E2E:-so-e2e}
NO_KIND_CLEANUP=${NO_KIND_CLEANUP:-}
CI_MODE=${CI_MODE:-}

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

kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml --wait
echo "Wait until otel operator pod is in ready state..."
kubectl wait --namespace opentelemetry-operator-system --for=condition=available deployment/opentelemetry-operator-controller-manager --timeout=300s

# Create subscription operator resources
(cd .. && make manifests generate install)

# Use example
kubectl apply -f ../docs/examples/simple-demo

if [[ -z "${CI_MODE}" ]]; then
  $(cd .. && timeout 5m make run &)
else
  kind load docker-image controller:latest --name "${KIND_CLUSTER_NAME}"
  cd .. && make deploy && cd -
fi

# Create log-generator
helm install --wait \
  --create-namespace \
  --namespace example-tenant-ns \
  --generate-name oci://ghcr.io/kube-logging/helm-charts/log-generator \
  --debug \
  --set app.count=0

LOG_GENERATOR_POD=$(kubectl get pods -A -o custom-columns=':metadata.name' | grep log-generator)
kubectl port-forward -n example-tenant-ns "pod/${LOG_GENERATOR_POD}" 11000:11000 &
sleep 5
JSON_PAYLOAD='{ "type": "web", "format": "apache", "count": 100 }'
curl --location --request POST '127.0.0.1:11000/loggen' --header 'Content-Type: application/json' --data-raw "${JSON_PAYLOAD}"

# Make sure log generator only generates n log messages
EXPECTED_NUMBER_OF_LOGS=$(echo "$JSON_PAYLOAD" | jq .count)

EXPECTED_NUMBER_OF_LOGS_CHECKPOINT=15
# Check for received messages - subscription-sample
while
  echo "Checking for subscription-sample-1 in deployments/receiver-collector logs, expected: ${EXPECTED_NUMBER_OF_LOGS}"
  NUM_OF_LOGS=$(kubectl logs --namespace example-tenant-ns deployments/receiver-collector | grep -c "subscription-sample-1")
  echo "Found logs: ${NUM_OF_LOGS}"

  if [[ $NUM_OF_LOGS -eq $EXPECTED_NUMBER_OF_LOGS_CHECKPOINT ]]; then
    # Kill the telemetry controller to assert persist works
    TELEMETRY_CONTROLLER_POD=$(kubectl get pods -A -o custom-columns=':metadata.name' | grep  subscription-operator-controller-manager)
    kubectl delete pod --namespace subscription-operator-system "${TELEMETRY_CONTROLLER_POD}"  
  fi

  
  [[ $NUM_OF_LOGS -ne $EXPECTED_NUMBER_OF_LOGS ]]
do true; done

# Check for received messages - subscription-sample-2
while
  echo "Checking for subscription-sample-2 in deployments/receiver-collector logs, expected: ${EXPECTED_NUMBER_OF_LOGS}"
  NUM_OF_LOGS=$(kubectl logs --namespace example-tenant-ns deployments/receiver-collector | grep -c "subscription-sample-2")

  [[ $NUM_OF_LOGS -ne $EXPECTED_NUMBER_OF_LOGS ]]
do true; done

echo "E2E test: PASSED"


# Check if cluster should be removed, ctx restored
if [[ -z "${NO_KIND_CLEANUP}" ]]; then
  kind delete cluster --name "${KIND_CLUSTER_NAME}"
fi
