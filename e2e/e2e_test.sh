#!/bin/env bash

create_if_does_not_exist() {
  local resource_type=$1
  local resource_name=$2
  kubectl create  "${resource_type}" "${resource_name}" --dry-run=client -o yaml | kubectl apply -f-
}

KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME_E2E:-so-e2e}
# Backup current kubernetes context
CURRENT_K8S_CTX=$(kubectl config view | grep "current" | cut -f 2 -d : | xargs)

# Prepare env
kind create cluster --name ${KIND_CLUSTER_NAME} --wait 5m
kubectl config set-context kind-${KIND_CLUSTER_NAME}

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

# Create receivers
create_if_does_not_exist "namespace" "otlp-receiver-1"
create_if_does_not_exist "namespace" "otlp-receiver-2"

kubectl apply -f otlp_receiver.yaml


# Create subscription operator resources
cd .. && make manifests generate install && cd -

create_if_does_not_exist "namespace" "outputs"
kubectl apply -f telemetry_v1alpha1_oteloutputs.yaml

create_if_does_not_exist "namespace" "subscriptions"
kubectl label namespace subscriptions subscription=subscription-sample
kubectl apply -f telemetry_v1alpha1_subscriptions.yaml

kubectl apply -f telemetry_v1alpha1_tenant.yaml
kubectl apply -f telemetry_v1alpha1_collector.yaml


# TODO: Run subscription operator in the script
# cd .. && nohup make run & 
# cd -

# Create log-generators
helm install --wait --create-namespace --namespace log-generator-1 --generate-name oci://ghcr.io/kube-logging/helm-charts/log-generator
LOG_GENERATOR_POD_1="$(kubectl get pods --namespace log-generator-1 --no-headers -o custom-columns=':metadata.name' | grep generator)"
kubectl label --namespace log-generator-1 pod ${LOG_GENERATOR_POD_1} tenant=tenant-sample
kubectl label --namespace log-generator-1 pod ${LOG_GENERATOR_POD_1} subscription=subscription-sample-1

helm install --wait --create-namespace --namespace log-generator-2 --generate-name oci://ghcr.io/kube-logging/helm-charts/log-generator
LOG_GENERATOR_POD_2="$(kubectl get pods --namespace log-generator-2 --no-headers -o custom-columns=':metadata.name' | grep generator)"
kubectl label --namespace log-generator-2 pod ${LOG_GENERATOR_POD_2} tenant=tenant-sample
kubectl label --namespace log-generator-2 pod ${LOG_GENERATOR_POD_2} subscription=subscription-sample-2

# Check for received messages - subscription-sample-1
while
  echo "Checking for subscription-sample-1 in deployments/test-otlp-receiver-1-collector logs"
  kubectl logs --namespace otlp-receiver-1 deployments/test-otlp-receiver-1-collector | grep -q "subscription-sample-1"
  [[ $? -ne 0 ]]
do true; done

# Check for received messages - subscription-sample-2
while
  echo "Checking for subscription-sample-2 in deployments/test-otlp-receiver-2-collector logs"
  kubectl logs --namespace otlp-receiver-2 deployments/test-otlp-receiver-2-collector | grep -q "subscription-sample-2"

  [[ $? -ne 0 ]]
do true; done

echo "E2E test: PASSED"


# Check if cluster should be removed, ctx restored
if [[ -z "${NO_KIND_CLEANUP}" ]]; then
  kind delete cluster --name ${KIND_CLUSTER_NAME}
fi

if [[ "${CURRENT_K8S_CTX}" != "" ]]; then
  kubectl config get-contexts -o name | grep -q "${CURRENT_K8S_CTX}" && kubectl config set-context "${CURRENT_K8S_CTX}"
fi
