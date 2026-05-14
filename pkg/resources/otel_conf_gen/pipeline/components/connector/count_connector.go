// Copyright © 2024 Kube logging authors
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

package connector

type CountConnectorAttributeConfig struct {
	Key          *string `json:"key,omitempty"`
	DefaultValue *string `json:"default_value,omitempty"`
}

type CountConnectorMetricInfo struct {
	Description *string                         `json:"description,omitempty"`
	Conditions  []string                        `json:"conditions,omitempty"`
	Attributes  []CountConnectorAttributeConfig `json:"attributes,omitempty"`
}

func GenerateCountConnectors() map[string]any {
	countConnectors := make(map[string]any)
	countConnectors["count/tenant_metrics"] = map[string]any{
		"logs": map[string]CountConnectorMetricInfo{
			"telemetry_controller_tenant_log_count": {
				Description: new("The number of logs from each tenant pipeline."),
				Attributes: []CountConnectorAttributeConfig{
					{
						Key: new("tenant"),
					},
					{
						Key: new("k8s.namespace.name"),
					},
					{
						Key: new("k8s.node.name"),
					},
					{
						Key: new("k8s.container.name"),
					},
					{
						Key: new("k8s.pod.name"),
					},
					{
						Key: new("k8s.pod.labels.app.kubernetes.io/name"),
					},
					{
						Key: new("k8s.pod.labels.app"),
					},
				},
			},
		},
	}

	countConnectors["count/output_metrics"] = map[string]any{
		"logs": map[string]CountConnectorMetricInfo{
			"telemetry_controller_output_log_count": {
				Description: new("The number of logs sent out from each exporter."),
				Attributes: []CountConnectorAttributeConfig{
					{
						Key: new("tenant"),
					},
					{
						Key: new("subscription"),
					},
					{
						Key: new("exporter"),
					},
					{
						Key: new("k8s.namespace.name"),
					},
					{
						Key: new("k8s.node.name"),
					},
					{
						Key: new("k8s.container.name"),
					},
					{
						Key: new("k8s.pod.name"),
					},
					{
						Key: new("k8s.pod.labels.app.kubernetes.io/name"),
					},
					{
						Key: new("k8s.pod.labels.app"),
					},
				},
			},
		},
	}

	return countConnectors
}
