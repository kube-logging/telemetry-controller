#!/usr/bin/env bash

set -eou pipefail
set -o xtrace

function main()
{
    setup

    test_one_tenant_two_subscriptions
    test_tenants_with_bridges
    test_filestorage_receiver_failure
    test_filestorage_collector_failure
    test_problem_tracking

    echo "E2E (helm) test: PASSED"
}

function setup()
{
    kubectl get namespaces telemetry-controller-system || kubectl create namespace telemetry-controller-system
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
    check_logs_in_workload_with_regex "telemetry-controller-system" "deployments" "receiver-collector" "subscription-sample-1"
    check_logs_in_workload_with_regex "telemetry-controller-system" "deployments" "receiver-collector" "subscription-sample-2"

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
    check_logs_in_workload_with_regex "telemetry-controller-system" "deployments" "receiver-collector" "subscription: Str\(database\)"
    check_logs_in_workload_with_regex "telemetry-controller-system" "deployments" "receiver-collector" "subscription: Str\(web\)"

    helm_uninstall_log_generator_from_ns "shared"
    undeploy_test_assets "e2e/testdata/tenants_with_bridges/"
}

function test_filestorage_receiver_failure()
{
    helm_install_log_generator_to_ns "example-tenant-ns" "app.count=0" "app.eventPerSec=100"

    deploy_test_assets "e2e/testdata/filestorage/"

    kubectl port-forward --namespace "example-tenant-ns" "deployments/log-generator" 11000:11000 &

    kubectl wait --namespace "telemetry-controller-system" --for=condition=available "deployments/receiver-collector" --timeout=300s

    JSON_PAYLOAD='{ "type": "web", "format": "apache", "count": 10000 }'
    curl --location --request POST '127.0.0.1:11000/loggen' --header 'Content-Type: application/json' --data-raw "${JSON_PAYLOAD}"
    EXPECTED_NUMBER_OF_LOGS=$(echo "$JSON_PAYLOAD" | jq ".count")
    kill $(lsof -t -i:11000)

    sleep 3

    POD_NAME=$(kubectl get pods -A -o custom-columns=':metadata.name' | grep "otelcollector-example-collector")
    kubectl wait --namespace "collector" --for=condition=ready "pods/$POD_NAME" --timeout=300s

    # Wait until 10% of the logs are processed
    check_logs_until_expected_number_is_reached "$((EXPECTED_NUMBER_OF_LOGS / 10))"

    # Stop the receiver-collector deployment to see if the logs are stored in the file storage
    kubectl scale deployments --namespace "telemetry-controller-system" "receiver-collector" --replicas=0
    check_logs_in_workload_with_regex "collector" "daemonsets" "otelcollector-example-collector" "Exporting failed. Will retry the request after interval."
    kubectl scale deployments --namespace "telemetry-controller-system" "receiver-collector" --replicas=1

    sleep 5

    check_logs_until_expected_number_is_reached "$EXPECTED_NUMBER_OF_LOGS"
    echo "SUCCESS: All logs have been processed."

    rm -rd /tmp/otelcol-contrib
    helm_uninstall_log_generator_from_ns "example-tenant-ns"
    undeploy_test_assets "e2e/testdata/filestorage/"
}

function test_filestorage_collector_failure()
{
    helm_install_log_generator_to_ns "example-tenant-ns" "app.count=0" "app.eventPerSec=100"

    deploy_test_assets "e2e/testdata/filestorage/"

    kubectl port-forward --namespace "example-tenant-ns" "deployments/log-generator" 11000:11000 &

    kubectl wait --namespace "telemetry-controller-system" --for=condition=available "deployments/receiver-collector" --timeout=300s

    JSON_PAYLOAD='{ "type": "web", "format": "apache", "count": 10000 }'
    curl --location --request POST '127.0.0.1:11000/loggen' --header 'Content-Type: application/json' --data-raw "${JSON_PAYLOAD}"
    EXPECTED_NUMBER_OF_LOGS=$(echo "$JSON_PAYLOAD" | jq ".count")
    kill $(lsof -t -i:11000)

    sleep 3

    POD_NAME=$(kubectl get pods -A -o custom-columns=':metadata.name' | grep "otelcollector-example-collector")
    kubectl wait --namespace "collector" --for=condition=ready "pods/$POD_NAME" --timeout=300s

    # Wait until 10% of the logs are processed
    check_logs_until_expected_number_is_reached "$((EXPECTED_NUMBER_OF_LOGS / 10))"

    kubectl delete pods --namespace "collector" $POD_NAME

    sleep 5

    check_logs_until_expected_number_is_reached "$EXPECTED_NUMBER_OF_LOGS"
    echo "SUCCESS: All logs have been processed."

    rm -rd /tmp/otelcol-contrib
    helm_uninstall_log_generator_from_ns "example-tenant-ns"
    undeploy_test_assets "e2e/testdata/filestorage/"
}

function test_problem_tracking()
{
    helm_install_log_generator_to_ns "problem-test-ns"

    deploy_test_assets "e2e/testdata/problem_tracking/problem"
    
    # Verify initial problem states
    check_resource_status "output" "problem-test-ns" "output-without-tenant" "tenant" "problem-test-tenant"
    check_resource_status "subscription" "problem-test-ns" "valid-subscription" "state" "ready"
    
    check_resource_status "subscription" "problem-test-ns" "subscription-missing-output" "state" "failed"
    check_resource_problems "subscription" "problem-test-ns" "subscription-missing-output" "non-existent-output" 1
    
    check_resource_status "subscription" "problem-test-ns" "cross-tenant-subscription" "state" "failed"
    check_resource_problems "subscription" "problem-test-ns" "cross-tenant-subscription" "tenant" 1
    
    check_resource_status "tenant" "" "problem-test-tenant" "state" "failed"
    # there should be atleast 1 problem
    check_resource_problems_min "tenant" "" "problem-test-tenant" "subscription-missing-output" 1
    
    # Apply fixes
    deploy_test_assets "e2e/testdata/problem_tracking/fix"
    
    # Verify fixed states
    check_resource_status "subscription" "problem-test-ns" "subscription-missing-output" "state" "ready"
    check_resource_problems "subscription" "problem-test-ns" "subscription-missing-output" "" 0
    
    check_resource_status "subscription" "problem-test-ns" "cross-tenant-subscription" "state" "ready"
    check_resource_problems "subscription" "problem-test-ns" "cross-tenant-subscription" "" 0
    
    check_resource_status "tenant" "" "problem-test-tenant" "state" "ready"
    check_resource_problems "tenant" "" "problem-test-tenant" "" 0
    
    # Verify logs are flowing
    check_logs_in_workload_with_regex "telemetry-controller-system" "deployments" "receiver-collector" "valid-subscription"
    
    helm_uninstall_log_generator_from_ns "problem-test-ns"
    undeploy_test_assets "e2e/testdata/problem_tracking/problem"
}

function helm_install_log_generator_to_ns() {
    local namespace="$1"
    shift

    local set_args=()
    while (( "$#" )); do
        set_args+=("--set" "$1")
        shift
    done

    helm install \
        --wait \
        --create-namespace \
        --namespace "$namespace" \
        log-generator \
        oci://ghcr.io/kube-logging/helm-charts/log-generator \
        "${set_args[@]}"
}

function helm_uninstall_log_generator_from_ns()
{
    local namespace="$1"

    helm uninstall --namespace "$namespace" log-generator
}

function deploy_test_assets()
{
    for manifest in "$@"; do
        kubectl apply -f "${manifest}"
    done

    sleep 5
}

function undeploy_test_assets()
{
    for manifest in "$@"; do
        kubectl delete -f "${manifest}"
    done

    sleep 5
}

function check_logs_in_workload_with_regex()
{
    local namespace="$1"
    local workload_type="$2"
    local deployment="$3"
    local regex="$4"
    local max_duration=300
    local start_time=$(date +%s)

    echo "Checking for logs in $namespace/$deployment with regex: $regex"

    while true; do
        if kubectl logs --namespace "$namespace" "$workload_type/$deployment" | grep -E "$regex"; then
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

function check_logs_until_expected_number_is_reached() {
    local expected_number_of_logs="$1"
    local log_path="/tmp/otelcol-contrib/e2e.log"
    local max_duration=300
    local start_time=$(date +%s)

    mkdir -p /tmp/otelcol-contrib

    while true; do
        LOG_PATH=$(docker exec kind-control-plane find /tmp -name "e2e.log" 2>/dev/null)
        if [ -n "$LOG_PATH" ]; then
            docker exec kind-control-plane cat "$LOG_PATH" > "$log_path"
            if [ $? -eq 0 ] && [ -s "$log_path" ]; then
                NUM_OF_LOGS=$(cat "$log_path" | jq -r '.resourceLogs|map(.scopeLogs)[0][0].logRecords|map(.body.stringValue)' | wc -l)
                if [[ "$NUM_OF_LOGS" -ge "$expected_number_of_logs" ]]; then
                    echo "Expected number of logs processed."
                    return 0
                fi
            fi
        fi
        
        sleep 5
        
        local current_time=$(date +%s)
        local elapsed_time=$((current_time - start_time))
        if [ "$elapsed_time" -ge "$max_duration" ]; then
            echo "ERROR: Logs not found after $max_duration seconds."
            return 1
        fi
    done
}

function check_resource_status()
{
    local resource_type="$1"
    local namespace="$2"
    local resource_name="$3"
    local field="$4"
    local expected_value="$5"
    
    local ns_flag=""
    if [ -n "$namespace" ]; then
        ns_flag="-n $namespace"
    fi
    
    local actual_value=$(kubectl get "$resource_type" $ns_flag "$resource_name" -o jsonpath="{.status.$field}")
    
    if [ "$actual_value" != "$expected_value" ]; then
        echo "ERROR: $resource_type $resource_name should have $field '$expected_value', but has '$actual_value'"
        kubectl get "$resource_type" $ns_flag "$resource_name" -o yaml
        return 1
    fi
}

function check_resource_problems()
{
    local resource_type="$1"
    local namespace="$2"
    local resource_name="$3"
    local expected_problem_text="$4"
    local expected_count="$5"
    
    local ns_flag=""
    if [ -n "$namespace" ]; then
        ns_flag="-n $namespace"
    fi
    
    local problems=$(kubectl get "$resource_type" $ns_flag "$resource_name" -o jsonpath='{.status.problems}')
    local problems_count=$(kubectl get "$resource_type" $ns_flag "$resource_name" -o jsonpath='{.status.problemsCount}')
    
    # Check if problems should be empty
    if [ -z "$expected_problem_text" ]; then
        if ([ -n "$problems" ] && [ "$problems" != "[]" ]) || [ "$problems_count" -ne "$expected_count" ]; then
            echo "ERROR: $resource_type $resource_name should have no problems"
            echo "Problems: $problems"
            return 1
        fi
    else
        # Check if problems contain expected text
        if [[ ! "$problems" == *"$expected_problem_text"* ]] || [ "$problems_count" -ne "$expected_count" ]; then
            echo "ERROR: $resource_type $resource_name should have problem mentioning '$expected_problem_text' and count $expected_count"
            echo "Problems: $problems (count: $problems_count)"
            return 1
        fi
    fi
}

function check_resource_problems_min()
{
    local resource_type="$1"
    local namespace="$2"
    local resource_name="$3"
    local expected_problem_text="$4"
    local min_count="$5"
    
    local ns_flag=""
    if [ -n "$namespace" ]; then
        ns_flag="-n $namespace"
    fi
    
    local problems=$(kubectl get "$resource_type" $ns_flag "$resource_name" -o jsonpath='{.status.problems}')
    local problems_count=$(kubectl get "$resource_type" $ns_flag "$resource_name" -o jsonpath='{.status.problemsCount}')
    
    # Check if problems contain expected text and count is at least min_count
    if [[ ! "$problems" == *"$expected_problem_text"* ]] || [ "$problems_count" -lt "$min_count" ]; then
        echo "ERROR: $resource_type $resource_name should have problem mentioning '$expected_problem_text' and count >= $min_count"
        echo "Problems: $problems (count: $problems_count)"
        return 1
    fi
}

main "$@"
