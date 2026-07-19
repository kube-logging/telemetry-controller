// Copyright © 2026 Kube logging authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package receiver

import (
	"fmt"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

// GenerateKubernetesEventsReceiver assembles the configuration for a k8s_events receiver
// (k8seventsreceiver), which collects Kubernetes events and emits them as log records.
// When the tenant selects from all namespaces, the namespaces field is omitted,
// which makes the receiver collect events from all namespaces.
// ref: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/k8seventsreceiver
func GenerateKubernetesEventsReceiver(namespaces []string, dryRunMode bool, tenant v1alpha1.Tenant, eventsToLogs v1alpha1.EventsToLogs) map[string]any {
	k8sEventsReceiver := map[string]any{}
	if !tenant.Spec.SelectFromAllNamespaces && len(namespaces) > 0 {
		k8sEventsReceiver["namespaces"] = namespaces
	}
	if eventsToLogs.KubeAPIQPS != nil {
		k8sEventsReceiver["kube_api_qps"] = *eventsToLogs.KubeAPIQPS
	}
	if eventsToLogs.KubeAPIBurst != nil {
		k8sEventsReceiver["kube_api_burst"] = *eventsToLogs.KubeAPIBurst
	}
	if eventsToLogs.DedupInterval != nil {
		k8sEventsReceiver["dedup_interval"] = *eventsToLogs.DedupInterval
	}
	if !dryRunMode && tenant.Spec.PersistenceConfig.EnableFileStorage {
		k8sEventsReceiver["storage"] = fmt.Sprintf("file_storage/%s", tenant.Name)
	}

	return k8sEventsReceiver
}
