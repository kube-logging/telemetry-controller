# telemetry-controller

![type: application](https://img.shields.io/badge/type-application-informational?style=flat-square) ![kube version: >=1.26.0](https://img.shields.io/badge/kube%20version->=1.26.0-informational?style=flat-square) [![artifact hub](https://img.shields.io/badge/artifact%20hub-telemetry--controller-informational?style=flat-square)](https://artifacthub.io/packages/helm/kube-logging/telemetry-controller)

A Helm chart for deploying telemetry-controller

## TL;DR;

```bash
helm install --generate-name --wait oci://ghcr.io/kube-logging/helm-charts/telemetry-controller
```

or to install with a specific version:

```bash
helm install --generate-name --wait oci://ghcr.io/kube-logging/helm-charts/telemetry-controller --version $VERSION
```

## Introduction

This chart bootstraps a [Telemetry controller](https://github.com/kube-logging/telemetry-controller) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.19+

## Installing CRDs

Use `createCustomResource=false` with Helm v3 to avoid trying to create CRDs from the `crds` folder and from templates at the same time.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| replicaCount | int | `1` |  |
| image.repository | string | `"ghcr.io/kube-logging/telemetry-controller"` |  |
| image.tag | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| extraArgs[0] | string | `"--leader-elect=true"` |  |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| namespaceOverride | string | `""` |  |
| fullnameOverride | string | `""` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.name | string | `""` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| resources | object | See values.yaml | Resource requests and limits for the manager container. |
| nodeSelector | object | `{}` | Node selector for pod scheduling. |
| tolerations | list | `[]` | Tolerations for pod scheduling. |
| affinity | object | `{}` | Affinity rules for pod scheduling. |
| topologySpreadConstraints | list | `[]` | Topology spread constraints for pod scheduling. |
| priorityClassName | string | `""` | Priority class name for the pod. |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| service.type | string | `"ClusterIP"` |  |
| service.https.name | string | `"https"` |  |
| service.https.port | int | `8443` |  |
| service.https.protocol | string | `"TCP"` |  |
| service.https.targetPort | int | `8443` |  |
| service.http.name | string | `"http"` |  |
| service.http.port | int | `8080` |  |
| service.http.protocol | string | `"TCP"` |  |
| service.http.targetPort | int | `8080` |  |
| opentelemetry-operator | object | See values.yaml | Configuration for the opentelemetry-operator sub-chart. |
| monitoring.secure | bool | `true` |  |
| monitoring.serviceMonitor.enabled | bool | `false` | Create a Prometheus Operator ServiceMonitor object. |
| monitoring.serviceMonitor.additionalLabels | object | `{}` |  |
| monitoring.serviceMonitor.metricRelabelings | list | `[]` |  |
| monitoring.serviceMonitor.relabelings | list | `[]` |  |
