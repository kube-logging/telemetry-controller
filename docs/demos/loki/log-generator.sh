#!/bin/bash

helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n example-tenant-ns
helm upgrade --install --wait log-generator oci://ghcr.io/kube-logging/helm-charts/log-generator -n another-tenant-ns
