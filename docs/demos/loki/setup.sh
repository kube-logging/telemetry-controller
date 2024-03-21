#!/bin/bash

helm upgrade --install --namespace loki --create-namespace loki --repo https://grafana.github.io/helm-charts loki --values loki_values.yaml --version 5.38.0
helm upgrade --install --namespace=loki --repo https://grafana.github.io/helm-charts loki-grafana grafana --version 7.0.8

helm upgrade --install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true --version v1.13.3
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.93.0/opentelemetry-operator.yaml

(cd ../../.. && make install)

kubectl apply -f manifests.yaml

helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n tenant-loki-1
helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n tenant-loki-2

kubectl get secret -n loki loki-grafana -o jsonpath="{.data.admin-password}" | base64 --decode
echo ""

(cd ../../.. && make run)
