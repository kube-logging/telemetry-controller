#!/usr/bin/env bash

set -eou pipefail
set -o xtrace

function main()
{
    kubectl get namespace telemetry-controller-system || kubectl create namespace telemetry-controller-system
    kubectl config set-context --current --namespace=telemetry-controller-system

    load_images

    helm_deploy_telemetry_controller

    helm_deploy_log_generator

    deploy_test_assets

    # Check for received messages - subscription-sample-1
    # NOTE: We should not use grep -q, because it causes a SIGPIPE for kubectl and we have -o pipefail
    check_subscription_logs "subscription-sample-1"
    check_subscription_logs "subscription-sample-2"

    echo "E2E (helm) test: PASSED"
}

function load_images()
{
    local images=("controller:local")
    for image in "${images[@]}"; do
        kind load docker-image "${image}"
    done
}

function helm_deploy_telemetry_controller()
{
    helm upgrade --install \
        --debug \
        --wait \
        --create-namespace \
        -f e2e/values.yaml \
        telemetry-controller \
        "charts/telemetry-controller/"
}

function helm_deploy_log_generator()
{
    helm install \
        --wait \
        --create-namespace \
        --namespace example-tenant-ns \
        --generate-name \
        oci://ghcr.io/kube-logging/helm-charts/log-generator
}

function deploy_test_assets()
{
    kubectl apply -f e2e/testdata/one_tenant_two_subscriptions/

    sleep 5

    # Wait for the deployment to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/receiver-collector --namespace example-tenant-ns
}

function check_subscription_logs()
{
    local subscription="$1"
    local namespace="${2:-example-tenant-ns}"
    local deployment="${3:-receiver-collector}"
    local max_duration=300
    local start_time=$(date +%s)

    echo "Checking for $subscription in $namespace/$deployment logs"

    while true; do
        if kubectl logs --namespace "$namespace" "deployments/$deployment" | grep -E "$subscription"; then
            echo "Found $subscription in logs"
            return 0
        fi

        sleep 1

        local current_time=$(date +%s)
        local elapsed_time=$((current_time - start_time))

        if [ "$elapsed_time" -ge "$max_duration" ]; then
            echo "ERROR: Subscription $subscription not found in logs after $max_duration seconds"
            return 1
        fi
    done
}

main "$@"
