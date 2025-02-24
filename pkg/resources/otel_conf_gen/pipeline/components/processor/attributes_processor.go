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

package processor

type AttributesProcessor struct {
	Actions []AttributesProcessorAction `json:"actions"`
}

type Action string

const (
	ActionInsert  Action = "insert"
	ActionUpdate  Action = "update"
	ActionUpsert  Action = "upsert"
	ActionDelete  Action = "delete"
	ActionHash    Action = "hash"
	ActionExtract Action = "extract"
	ActionConvert Action = "convert"
)

type AttributesProcessorAction struct {
	Action        Action `json:"action"`
	Key           string `json:"key"`
	Value         string `json:"value,omitempty"`
	FromAttribute string `json:"from_attribute,omitempty"`
	FromContext   string `json:"from_context,omitempty"`
}

func GenerateTenantAttributeProcessor(tenantName string) AttributesProcessor {
	return AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action: ActionInsert,
				Key:    "tenant",
				Value:  tenantName,
			},
		},
	}
}

func GenerateSubscriptionAttributeProcessor(subscriptionName string) AttributesProcessor {
	return AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action: ActionInsert,
				Key:    "subscription",
				Value:  subscriptionName,
			},
		},
	}
}

func GenerateOutputExporterNameProcessor(outputName string) AttributesProcessor {
	return AttributesProcessor{
		Actions: []AttributesProcessorAction{{
			Action: ActionInsert,
			Key:    "exporter",
			Value:  outputName,
		}},
	}
}

func GenerateMetricsProcessors() map[string]any {
	metricsProcessors := make(map[string]any)
	metricsProcessors["deltatocumulative"] = DeltaToCumulativeConfig{}
	metricsProcessors["attributes/metricattributes"] = AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action:        ActionInsert,
				Key:           "app",
				FromAttribute: "k8s.pod.labels.app",
			},
			{
				Action:        ActionInsert,
				Key:           "host",
				FromAttribute: "k8s.node.name",
			},
			{
				Action:        ActionInsert,
				Key:           "namespace",
				FromAttribute: "k8s.namespace.name",
			},
			{
				Action:        ActionInsert,
				Key:           "container",
				FromAttribute: "k8s.container.name",
			},
			{
				Action:        ActionInsert,
				Key:           "pod",
				FromAttribute: "k8s.pod.name",
			},
		},
	}

	return metricsProcessors
}
