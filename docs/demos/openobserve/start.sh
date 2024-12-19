#!/usr/bin/env bash

set -euo pipefail

KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-kind}

# Install OpenObserve
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: openobserve
---
apiVersion: v1
kind: Service
metadata:
  name: openobserve
  namespace: openobserve
spec:
  clusterIP: None
  selector:
    app: openobserve
  ports:
  - name: http
    port: 5080
    targetPort: 5080
---
apiVersion: v1
kind: Service
metadata:
  name: openobserve-otlp-grpc
  namespace: openobserve
spec:
  clusterIP: None
  selector:
    app: openobserve
  ports:
  - name: otlp-grpc
    port: 5081
    targetPort: 5081
---

# create statefulset
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: openobserve
  namespace: openobserve
  labels:
    name: openobserve
spec:
  serviceName: openobserve
  replicas: 1
  selector:
    matchLabels:
      name: openobserve
      app: openobserve
  template:
    metadata:
      labels:
        name: openobserve
        app: openobserve
    spec:
      securityContext:
        fsGroup: 2000
        runAsUser: 10000
        runAsGroup: 3000
        runAsNonRoot: true
      # terminationGracePeriodSeconds: 0
      containers:
        - name: openobserve
          image: public.ecr.aws/zinclabs/openobserve:v0.14.0
          env:
            - name: ZO_ROOT_USER_EMAIL
              value: root@example.com
            - name: ZO_ROOT_USER_PASSWORD
              value: Complexpass#123
            - name: ZO_DATA_DIR
              value: /data
          # command: ["/bin/bash", "-c", "while true; do sleep 1; done"]
          imagePullPolicy: Always
          resources:
            limits:
              cpu: 4096m
              memory: 2048Mi
            requests:
              cpu: 256m
              memory: 50Mi
          ports:
            - containerPort: 5080
              name: http
            - containerPort: 50801
              name: otlp-grpc
          volumeMounts:
          - name: data
            mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes:
        - ReadWriteOnce
      # storageClassName: default
      # NOTE: You can increase the storage size
      resources:
        requests:
          storage: 10Gi
EOF

# Check for readiness
while
  echo "Waiting for OpenObserve pod to be running"
  kubectl get pods -n openobserve | grep -qi "running"
  
  [[ $? -ne 0 ]]
do sleep 3; done

# Grab organization password

# OpenObserve UI
kubectl -n openobserve port-forward svc/openobserve 5080:5080 &
# OpenObserve OTLP GRPC
#kubectl -n openobserve port-forward svc/openobserve-otlp-grpc 5081:5081 &
sleep 5

OO_USER="root@example.com"
OO_PWD="Complexpass#123"
OO_PASSCODE=$(curl --silent localhost:5080/api/default/passcode -v --user $OO_USER:$OO_PWD | jq .data.passcode -r)
OO_TOKEN=$(echo "$OO_USER:$OO_PASSCODE" | tr -d '\n' | base64)
yq eval '(.spec.otlp.headers.Authorization = "Basic '${OO_TOKEN}'") as $auth | select(.kind == "Output") | $auth' -i ./demo.yaml


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


(cd ../../.. && make manifests generate install)

cd ../../../ && make docker-build
kind load docker-image controller:local --name "${KIND_CLUSTER_NAME}"
make deploy && cd -

kubectl apply -f ./demo.yaml


# Create log-generator
helm install --wait --create-namespace --namespace example-tenant-ns --generate-name oci://ghcr.io/kube-logging/helm-charts/log-generator
