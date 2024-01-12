#!/bin/env bash

create_namespace() {
  local namespace_name=$1
  kubectl get namespace ${namespace_name} > /dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    kubectl create namespace ${namespace_name}
  fi
}

KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME_E2E:-so-e2e}
# Backup current kubernetes context
CURRENT_K8S_CTX=$(kubectl config view | grep "current" | cut -f 2 -d : | xargs)

# Prepare env
kind create cluster --name ${KIND_CLUSTER_NAME} --wait 5m
kubectl config set-context kind-${KIND_CLUSTER_NAME}

# Install prerequisites
helm repo add jetstack https://charts.jetstack.io
helm repo update

helm upgrade \
  --install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.13.3 \
  --set installCRDs=true \
  --wait

kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml --wait
echo "Wait until otel operator pod is in ready state..."
OTEL_COL_OPERATOR_POD="$(kubectl get pods -A --no-headers -o custom-columns=':metadata.name' | grep opentelemetry-operator)"
kubectl wait --namespace opentelemetry-operator-system --for=condition=Ready pod/${OTEL_COL_OPERATOR_POD}

# Create receivers
create_namespace "otlp-receiver-1"
create_namespace "otlp-receiver-2"

kubectl apply -f otlp_receiver.yaml


# Create subscription operator resources
cd .. && make manifests generate install && cd -

create_namespace "outputs"
kubectl apply -f telemetry_v1alpha1_oteloutputs.yaml

create_namespace "subscriptions"
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
OTLP_RECEIVER_POD_1="$(kubectl get pods -A --no-headers -o custom-columns=':metadata.name' | grep receiver-1)"
while
  echo "Checking for subscription-sample-1 in ${OTLP_RECEIVER_POD_1} pod logs"
  kubectl logs --namespace otlp-receiver-1 pods/"${OTLP_RECEIVER_POD_1}" | grep -q "subscription-sample-1"
  [[ $? -ne 0 ]]
do true; done

# Check for received messages - subscription-sample-2
OTLP_RECEIVER_POD_2="$(kubectl get pods -A --no-headers -o custom-columns=':metadata.name' | grep receiver-2)"
while
  echo "Checking for subscription-sample-2 in ${OTLP_RECEIVER_POD_2} pod logs"
  kubectl logs --namespace otlp-receiver-2 pods/"${OTLP_RECEIVER_POD_2}" | grep -q "subscription-sample-2"
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
