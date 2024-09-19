#!/bin/bash

helm upgrade --install --namespace loki --create-namespace loki --repo https://grafana.github.io/helm-charts loki --values loki_values.yaml --version 6.12.0
helm upgrade --install --namespace loki --repo https://grafana.github.io/helm-charts loki-grafana grafana --version 8.5.1

helm upgrade --install --wait --create-namespace --namespace telemetry-controller-system telemetry-controller oci://ghcr.io/kube-logging/helm-charts/telemetry-controller


kubectl apply -f manifests.yaml

helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n tenant-demo-1
helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n tenant-demo-2

kubectl get secret -n loki loki-grafana -o jsonpath="{.data.admin-password}" | base64 --decode
echo ""

