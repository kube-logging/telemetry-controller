#!/usr/bin/env bash

set -eou pipefail
set -o xtrace

function main()
{
    setup

    test_one_tenant_two_subscriptions
    test_tenants_with_bridges

    echo "E2E (helm) test: PASSED"
}

function setup()
{
    kubectl get namespace telemetry-controller-system || kubectl create namespace telemetry-controller-system
    kubectl config set-context --current --namespace=telemetry-controller-system

    load_images

    helm_install_telemetry_controller
}

function load_images()
{
    local images=("controller:local")
    for image in "${images[@]}"; do
        kind load docker-image "${image}"
    done
}

function helm_install_telemetry_controller()
{
    local chart_dir="charts/telemetry-controller"
    helm dependency update "${chart_dir}"
    helm upgrade --install \
        --debug \
        --wait \
        --create-namespace \
        -f "e2e/values.yaml" \
        telemetry-controller \
        "${chart_dir}"
}

function test_one_tenant_two_subscriptions()
{
    helm_install_log_generator_to_ns "example-tenant-ns"

    deploy_test_assets "e2e/testdata/one_tenant_two_subscriptions/"

    # Check for received messages - subscription-sample-1
    # NOTE: We should not use grep -q, because it causes a SIGPIPE for kubectl and we have -o pipefail
    check_logs_in_workload_with_regex "example-tenant-ns" "receiver-collector" "subscription-sample-1"
    check_logs_in_workload_with_regex "example-tenant-ns" "receiver-collector" "subscription-sample-2"

    helm_uninstall_log_generator_from_ns "example-tenant-ns"
    undeploy_test_assets "e2e/testdata/one_tenant_two_subscriptions/"
}

function test_tenants_with_bridges()
{
    helm_install_log_generator_to_ns "shared"

    deploy_test_assets "e2e/testdata/tenants_with_bridges/"

    # NOTE: Since both database and web tenant is parsing logs from the shared tenant
    # if we see logs having Attribute "subscription" with value "Str(database)" or "Str(web)"
    # then it means the logs are being parsed by the respective tenants and the bridges are working as expected.
    check_logs_in_workload_with_regex "telemetry-controller-system" "receiver-collector" "subscription: Str\(database\)"
    check_logs_in_workload_with_regex "telemetry-controller-system" "receiver-collector" "subscription: Str\(web\)"

    helm_uninstall_log_generator_from_ns "shared"
    undeploy_test_assets "e2e/testdata/tenants_with_bridges/"
}

function helm_install_log_generator_to_ns()
{
    local namespace="$1"

    helm install \
        --wait \
        --create-namespace \
        --namespace "$namespace" \
        log-generator \
        oci://ghcr.io/kube-logging/helm-charts/log-generator
}

function helm_uninstall_log_generator_from_ns()
{
    local namespace="$1"

    helm uninstall --namespace "$namespace" log-generator
}

function deploy_test_assets()
{
    local manifests="$1"

    kubectl apply -f "${manifests}"

    sleep 5
}

function undeploy_test_assets()
{
    local manifests="$1"

    kubectl delete -f "${manifests}"

    sleep 5
}

function check_logs_in_workload_with_regex()
{
    local namespace="$1"
    local deployment="$2"
    local regex="$3"
    local max_duration=300
    local start_time=$(date +%s)

    echo "Checking for logs in $namespace/$deployment with regex: $regex"

    while true; do
        if kubectl logs --namespace "$namespace" "deployments/$deployment" | grep -E "$regex"; then
            echo "Logs with regex: $regex found in $namespace/$deployment."
            return 0
        fi

        sleep 1

        local current_time=$(date +%s)
        local elapsed_time=$((current_time - start_time))

        if [ "$elapsed_time" -ge "$max_duration" ]; then
            echo "ERROR: Logs with regex: $regex not found in $namespace/$deployment after $max_duration seconds."
            return 1
        fi
    done
}

main "$@"
