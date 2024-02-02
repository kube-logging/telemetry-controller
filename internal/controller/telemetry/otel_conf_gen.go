// Copyright © 2023 Kube logging authors
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

package telemetry

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"gopkg.in/yaml.v3"
)

// TODO move this to its appropriate place
type OtelColConfigInput struct {
	Tenants       []v1alpha1.Tenant
	Subscriptions []v1alpha1.Subscription
	Outputs       []v1alpha1.OtelOutput

	// Subscriptions map, where the key is the Tenants' namespaced name, value is a slice of subscriptions' namespaced name
	TenantSubscriptionMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
	SubscriptionOutputMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
}

type RoutingConnectorTableItem struct {
	Statement string   `yaml:"statement"`
	Pipelines []string `yaml:"pipelines,flow"`
}

type RoutingConnector struct {
	Name             string                      `yaml:"-"`
	DefaultPipelines []string                    `yaml:"default_pipelines,flow,omitempty"`
	Table            []RoutingConnectorTableItem `yaml:"table"`
}

type AttributesProcessor struct {
	Actions []AttributesProcessorAction `yaml:"actions"`
}

type AttributesProcessorAction struct {
	Action string `yaml:"action"`
	Key    string `yaml:"key"`
	Value  string `yaml:"value"`
}

type Pipeline struct {
	Receivers  []string `yaml:"receivers,omitempty,flow"`
	Processors []string `yaml:"processors,omitempty,flow"`
	Exporters  []string `yaml:"exporters,omitempty,flow"`
}

type Pipelines struct {
	Traces         Pipeline            `yaml:"traces,omitempty"`
	Metrics        Pipeline            `yaml:"metrics,omitempty"`
	Logs           Pipeline            `yaml:"logs,omitempty"`
	NamedPipelines map[string]Pipeline `yaml:",inline,omitempty"`
}

type Services struct {
	Extensions map[string]any `yaml:"extensions,omitempty"`
	Pipelines  Pipelines      `yaml:"pipelines,omitempty"`
	Telemetry  map[string]any `yaml:"telemetry,omitempty"`
}

type OtelColConfigIR struct {
	Receivers  map[string]any `yaml:"receivers,omitempty"`
	Exporters  map[string]any `yaml:"exporters,omitempty"`
	Processors map[string]any `yaml:"processors,omitempty"`
	Connectors map[string]any `yaml:"connectors,omitempty"`
	Services   Services       `yaml:"service,omitempty"`
}

func (cfgInput *OtelColConfigInput) generateExporters() map[string]any {
	exporters := cfgInput.generateOTLPExporters()
	exporters["logging/debug"] = map[string]any{
		"verbosity": "detailed",
	}
	return exporters
}

func (cfgInput *OtelColConfigInput) generateOTLPExporters() map[string]any {
	var result = make(map[string]any)

	for _, output := range cfgInput.Outputs {
		name := fmt.Sprintf("otlp/%s_%s", output.Namespace, output.Name)
		otlpGrpcValuesMarshaled, err := yaml.Marshal(output.Spec.OTLP)
		if err != nil {
			return result
		}
		var otlpGrpcValues map[string]any
		if err := yaml.Unmarshal(otlpGrpcValuesMarshaled, &otlpGrpcValues); err != nil {
			return result
		}

		result[name] = otlpGrpcValues
	}

	return result
}

func generatePipeline(receivers, processors, exporters []string) Pipeline {
	var result = Pipeline{}

	result.Receivers = receivers
	result.Processors = processors
	result.Exporters = exporters

	return result
}

func (rc *RoutingConnector) AddRoutingConnectorTableElem(newTableItem RoutingConnectorTableItem) {
	rc.Table = append(rc.Table, newTableItem)
}

func newRoutingConnector(name string, defaultPipelines []string) RoutingConnector {
	result := RoutingConnector{}

	result.DefaultPipelines = defaultPipelines
	result.Name = name

	return result
}

func buildRoutingTableItemForTenant(tenant v1alpha1.Tenant) RoutingConnectorTableItem {
	conditions := make([]string, len(tenant.Status.LogSourceNamespaces))

	for i, namespace := range tenant.Status.LogSourceNamespaces {
		conditions[i] = fmt.Sprintf(`IsMatch(attributes["k8s.namespace.name"], %q)`, namespace)
	}

	conditionString := strings.Join(conditions, " or ")

	newItem := RoutingConnectorTableItem{
		//Statement: fmt.Sprintf(`set(attributes["tenant"], %q) where %s`, tenant.Name, conditionString),
		Statement: fmt.Sprintf(`route() where %s`, conditionString),
		Pipelines: []string{fmt.Sprintf("logs/tenant_%s", tenant.Name)},
	}

	return newItem
}

func generateRootRoutingConnector(tenants []v1alpha1.Tenant) RoutingConnector {
	// Generate routing table's first hop that will sort it's input by tenant name
	defaultRc := newRoutingConnector("routing/tenants", []string{})
	for _, tenant := range tenants {
		tableItem := buildRoutingTableItemForTenant(tenant)
		defaultRc.AddRoutingConnectorTableElem(tableItem)
	}
	return defaultRc
}

func buildRoutingTableItemForSubscription(tenantName string, subscription v1alpha1.Subscription, index int) RoutingConnectorTableItem {

	pipelineName := fmt.Sprintf("logs/tenant_%s_subscription_%s", tenantName, subscription.Name)

	appendedSpaces := strings.Repeat(" ", index)

	newItem := RoutingConnectorTableItem{
		Statement: fmt.Sprintf("%s%s", subscription.Spec.OTTL, appendedSpaces),
		Pipelines: []string{pipelineName},
	}

	return newItem
}

func (cfgInput *OtelColConfigInput) generateRoutingConnectorForTenantsSubscription(tenantName string, subscriptionNames []v1alpha1.NamespacedName) RoutingConnector {
	rcName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName)
	rc := newRoutingConnector(rcName, []string{})

	for index, subscriptionRef := range subscriptionNames {

		subscriptionIdx := slices.IndexFunc(cfgInput.Subscriptions, func(output v1alpha1.Subscription) bool {
			return output.Name == subscriptionRef.Name && output.Namespace == subscriptionRef.Namespace
		})

		if subscriptionIdx == -1 {
			continue
		}

		subscription := cfgInput.Subscriptions[subscriptionIdx]

		tableItem := buildRoutingTableItemForSubscription(tenantName, subscription, index)
		rc.AddRoutingConnectorTableElem(tableItem)
	}

	return rc
}

func (cfgInput *OtelColConfigInput) generateConnectors() map[string]any {
	var connectors = make(map[string]any)

	rootRoutingConnector := generateRootRoutingConnector(cfgInput.Tenants)
	connectors[rootRoutingConnector.Name] = rootRoutingConnector

	for _, tenant := range cfgInput.Tenants {
		rc := cfgInput.generateRoutingConnectorForTenantsSubscription(tenant.Name, cfgInput.TenantSubscriptionMap[tenant.NamespacedName()])
		connectors[rc.Name] = rc
	}

	return connectors

}

func (cfgInput *OtelColConfigInput) generateProcessors() map[string]any {
	var processors = make(map[string]any)

	k8sProcessorName := "k8sattributes"
	processors[k8sProcessorName] = cfgInput.generateDefaultKubernetesProcessor()

	for _, tenant := range cfgInput.Tenants {
		processors[fmt.Sprintf("attributes/tenant_%s", tenant.Name)] = generateTenantAttributeProcessor(tenant)
	}

	for _, subscription := range cfgInput.Subscriptions {
		processors[fmt.Sprintf("attributes/subscription_%s", subscription.Name)] = generateSubscriptionAttributeProcessor(subscription)
	}
	return processors

}

func generateTenantAttributeProcessor(tenant v1alpha1.Tenant) AttributesProcessor {
	processor := AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action: "insert",
				Key:    "tenant_name",
				Value:  tenant.Name,
			},
		},
	}
	return processor

}

func generateSubscriptionAttributeProcessor(subscription v1alpha1.Subscription) AttributesProcessor {
	processor := AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action: "insert",
				Key:    "subscription_name",
				Value:  subscription.Name,
			},
		},
	}
	return processor

}

func generateRootPipeline() Pipeline {
	return generatePipeline([]string{"filelog/kubernetes"}, []string{"k8sattributes"}, []string{"routing/tenants"})
}

func (cfgInput *OtelColConfigInput) generateNamedPipelines() map[string]Pipeline {
	var namedPipelines = make(map[string]Pipeline)

	// Default pipeline
	//namedPipelines["logs/default"] = generatePipeline([]string{"routing/tenants"}, []string{}, []string{"debug/default"})

	namedPipelines["logs/all"] = generateRootPipeline()

	tenants := []v1alpha1.NamespacedName{}
	for tenant := range cfgInput.TenantSubscriptionMap {
		tenants = append(tenants, tenant)
	}

	for _, tenant := range tenants {
		// Generate a pipeline for the tenant
		tenantPipelineName := fmt.Sprintf("logs/tenant_%s", tenant.Name)
		tenantRoutingName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenant.Name)
		namedPipelines[tenantPipelineName] = generatePipeline([]string{"routing/tenants"}, []string{fmt.Sprintf("attributes/tenant_%s", tenant.Name)}, []string{tenantRoutingName})

		// Generate pipelines for the subscriptions for the tenant
		for _, subscription := range cfgInput.TenantSubscriptionMap[tenant] {
			tenantSubscriptionPipelineName := fmt.Sprintf("%s_subscription_%s", tenantPipelineName, subscription.Name)

			targetOutputNames := []string{}
			for _, outputRef := range cfgInput.SubscriptionOutputMap[subscription] {
				outputIdx := slices.IndexFunc(cfgInput.Outputs, func(output v1alpha1.OtelOutput) bool {
					return output.Name == outputRef.Name && output.Namespace == outputRef.Namespace
				})

				if outputIdx == -1 {
					continue
				}

				targetOutputName := fmt.Sprintf("otlp/%s_%s", outputRef.Namespace, outputRef.Name)

				targetOutputNames = append(targetOutputNames, targetOutputName)

			}
			namedPipelines[tenantSubscriptionPipelineName] = generatePipeline([]string{tenantRoutingName}, []string{fmt.Sprintf("attributes/subscription_%s", subscription.Name)}, targetOutputNames)

		}

		// Add default (catch all) pipelines
		//tenantDefaultCatchAllPipelineName := fmt.Sprintf("%s_default", tenantPipelineName)
		//tenantDefaultCatchAllPipelineExporterName := fmt.Sprintf("debug/tenant_%s_default", tenant.Name)
		//namedPipelines[tenantDefaultCatchAllPipelineName] = generatePipeline([]string{tenantRoutingName}, []string{}, []string{})

	}

	return namedPipelines

}

func (cfgInput *OtelColConfigInput) generateDefaultKubernetesProcessor() map[string]any {
	type Source struct {
		Name string `yaml:"name,omitempty"`
		From string `yaml:"from,omitempty"`
	}

	defaultSources := []Source{
		{
			Name: "k8s.namespace.name",
			From: "resource_attribute",
		},
		{
			Name: "k8s.pod.name",
			From: "resource_attribute",
		},
	}

	var defaultPodAssociation = []map[string]any{
		{"sources": defaultSources},
	}

	var defaultMetadata = []string{
		"k8s.pod.name",
		"k8s.pod.uid",
		"k8s.deployment.name",
		"k8s.namespace.name",
		"k8s.node.name",
		"k8s.pod.start_time",
	}

	k8sProcessor := map[string]any{
		"auth_type":   "serviceAccount",
		"passthrough": false,
		"extract": map[string]any{
			"metadata": defaultMetadata,
		},
		"pod_association": defaultPodAssociation,
	}

	return k8sProcessor
}

func (cfgInput *OtelColConfigInput) generateDefaultKubernetesReceiver() map[string]any {

	// TODO: fix parser-crio
	operators := []map[string]any{
		{
			"type": "router",
			"id":   "get-format",
			"routes": []map[string]string{
				{
					"output": "parser-docker",
					"expr":   `body matches "^\\{"`,
				},
				{
					"output": "parser-containerd",
					"expr":   `body matches "^[^ Z]+Z"`,
				},
			},
		},
		{
			"type":   "regex_parser",
			"id":     "parser-containerd",
			"regex":  `^(?P<time>[^ ^Z]+Z) (?P<stream>stdout|stderr) (?P<logtag>[^ ]*) ?(?P<log>.*)$`,
			"output": "extract_metadata_from_filepath",
			"timestamp": map[string]string{
				"parse_from": "attributes.time",
				"layout":     "%Y-%m-%dT%H:%M:%S.%LZ",
			},
		},
		{
			"type":   "json_parser",
			"id":     "parser-docker",
			"output": "extract_metadata_from_filepath",
			"timestamp": map[string]string{
				"parse_from": "attributes.time",
				"layout":     "%Y-%m-%dT%H:%M:%S.%LZ",
			},
		},
		{
			"type":       "regex_parser",
			"id":         "extract_metadata_from_filepath",
			"regex":      `^.*\/(?P<namespace>[^_]+)_(?P<pod_name>[^_]+)_(?P<uid>[a-f0-9-]+)\/(?P<container_name>[^\/]+)\/(?P<restart_count>\d+)\.log$`,
			"parse_from": `attributes["log.file.path"]`,
			"cache": map[string]int{
				"size": 128,
			},
		},
		{
			"type": "move",
			"from": "attributes.log",
			"to":   "body",
		},
		{
			"type": "move",
			"from": "attributes.stream",
			"to":   `attributes["log.iostream"]`,
		},
		{
			"type": "move",
			"from": "attributes.container_name",
			"to":   `resource["k8s.container.name"]`,
		},
		{
			"type": "move",
			"from": "attributes.namespace",
			"to":   `resource["k8s.namespace.name"]`,
		},
		{
			"type": "move",
			"from": "attributes.pod_name",
			"to":   `resource["k8s.pod.name"]`,
		},
		{
			"type": "move",
			"from": "attributes.restart_count",
			"to":   `resource["k8s.container.restart_count"]`,
		},
		{
			"type": "move",
			"from": "attributes.uid",
			"to":   `resource["k8s.pod.uid"]`,
		},
	}

	k8sReceiver := map[string]any{
		"include":           []string{"/var/log/pods/*/*/*.log"},
		"exclude":           []string{"/var/log/pods/*/otc-container/*.log"},
		"start_at":          "end",
		"include_file_path": true,
		"include_file_name": false,
		"operators":         operators,
	}

	return k8sReceiver

}

func (cfgInput *OtelColConfigInput) ToIntermediateRepresentation() *OtelColConfigIR {
	result := OtelColConfigIR{}

	// Get  outputs based tenant names
	result.Exporters = cfgInput.generateExporters()

	// Add processors
	result.Processors = cfgInput.generateProcessors()

	result.Receivers = make(map[string]any)
	k8sReceiverName := "filelog/kubernetes" //only one instance for now
	result.Receivers[k8sReceiverName] = cfgInput.generateDefaultKubernetesReceiver()

	result.Connectors = cfgInput.generateConnectors()
	result.Services.Pipelines.NamedPipelines = make(map[string]Pipeline)

	result.Services.Pipelines.NamedPipelines = cfgInput.generateNamedPipelines()

	result.Services.Telemetry = make(map[string]any)

	return &result
}

func (cfg *OtelColConfigIR) ToYAML() (string, error) {
	bytes, err := cfg.ToYAMLRepresentation()
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (cfg *OtelColConfigIR) ToYAMLRepresentation() ([]byte, error) {
	if cfg != nil {
		bytes, err := yaml.Marshal(cfg)
		if err != nil {
			return []byte{}, err
		}
		return bytes, nil
	}
	return []byte{}, nil
}
