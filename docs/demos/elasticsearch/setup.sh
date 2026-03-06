#!/bin/bash

helm repo add elastic https://helm.elastic.co 2>/dev/null
helm repo update

helm upgrade --install --wait --namespace elasticsearch --create-namespace elasticsearch elastic/elasticsearch \
  --set replicas=1 \
  --set minimumMasterNodes=1 \
  --set resources.requests.cpu=100m \
  --set resources.requests.memory=512Mi \
  --set persistence.enabled=false

helm upgrade --install --wait --create-namespace --namespace telemetry-controller-system telemetry-controller oci://ghcr.io/kube-logging/helm-charts/telemetry-controller

kubectl apply -f manifests.yaml

helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n tenant-es-demo
