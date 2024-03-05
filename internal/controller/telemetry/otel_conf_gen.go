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
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type OtelColConfigInput struct {
	// These must only include resources that are selected by the collector, tenant labelselectors, and listed outputs in the subscriptions
	Tenants       []v1alpha1.Tenant
	Subscriptions map[v1alpha1.NamespacedName]v1alpha1.Subscription
	Outputs       []v1alpha1.OtelOutput
	MemoryLimit   *corev1.ResourceRequirements

	// Subscriptions map, where the key is the Tenants' name, value is a slice of subscriptions' namespaced name
	TenantSubscriptionMap map[string][]v1alpha1.NamespacedName
	SubscriptionOutputMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
	Debug                 bool
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
	Action        string `yaml:"action"`
	Key           string `yaml:"key"`
	Value         string `yaml:"value,omitempty"`
	FromAttribute string `yaml:"from_attribute,omitempty"`
	FromContext   string `yaml:"from_context,omitempty"`
}

type ResourceProcessor struct {
	Actions []ResourceProcessorAction `yaml:"attributes"`
}

type ResourceProcessorAction struct {
	Action        string `yaml:"action"`
	Key           string `yaml:"key"`
	Value         string `yaml:"value,omitempty"`
	FromAttribute string `yaml:"from_attribute,omitempty"`
	FromContext   string `yaml:"from_context,omitempty"`
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

func (cfgInput *OtelColConfigInput) generateExporters(ctx context.Context) map[string]any {
	exporters := map[string]any{}
	// TODO: add proper error handling

	maps.Copy(exporters, cfgInput.generateOTLPExporters(ctx))
	maps.Copy(exporters, cfgInput.generateLokiExporters(ctx))
	exporters["logging/debug"] = map[string]any{
		"verbosity": "detailed",
	}
	return exporters
}

func (cfgInput *OtelColConfigInput) generateOTLPExporters(ctx context.Context) map[string]any {
	logger := log.FromContext(ctx)
	var result = make(map[string]any)

	for _, output := range cfgInput.Outputs {
		if output.Spec.OTLP != nil {
			name := fmt.Sprintf("otlp/%s_%s", output.Namespace, output.Name)
			otlpGrpcValuesMarshaled, err := yaml.Marshal(output.Spec.OTLP)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.NamespacedName().String())
			}
			var otlpGrpcValues map[string]any
			if err := yaml.Unmarshal(otlpGrpcValuesMarshaled, &otlpGrpcValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.NamespacedName().String())
			}

			result[name] = otlpGrpcValues
		}
	}

	return result
}
func (cfgInput *OtelColConfigInput) generateLokiExporters(ctx context.Context) map[string]any {
	logger := log.FromContext(ctx)

	var result = make(map[string]any)

	for _, output := range cfgInput.Outputs {
		if output.Spec.Loki != nil {

			// TODO: add proper error handling
			name := fmt.Sprintf("loki/%s_%s", output.Namespace, output.Name)
			lokiHTTPValuesMarshaled, err := yaml.Marshal(output.Spec.Loki)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.NamespacedName().String())
			}
			var lokiHTTPValues map[string]any
			if err := yaml.Unmarshal(lokiHTTPValuesMarshaled, &lokiHTTPValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.NamespacedName().String())
			}

			result[name] = lokiHTTPValues
		}
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

func newRoutingConnector(name string) RoutingConnector {
	result := RoutingConnector{}

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
		Statement: fmt.Sprintf(`route() where %s`, conditionString),
		Pipelines: []string{fmt.Sprintf("logs/tenant_%s", tenant.Name)},
	}

	return newItem
}

func generateRootRoutingConnector(tenants []v1alpha1.Tenant) RoutingConnector {
	// Generate routing table's first hop that will sort it's input by tenant name
	defaultRc := newRoutingConnector("routing/tenants")
	for _, tenant := range tenants {
		tableItem := buildRoutingTableItemForTenant(tenant)
		defaultRc.AddRoutingConnectorTableElem(tableItem)
	}
	return defaultRc
}

func buildRoutingTableItemForSubscription(tenantName string, subscription v1alpha1.Subscription, index int) RoutingConnectorTableItem {

	pipelineName := fmt.Sprintf("logs/tenant_%s_subscription_%s_%s", tenantName, subscription.Namespace, subscription.Name)

	appendedSpaces := strings.Repeat(" ", index)

	newItem := RoutingConnectorTableItem{
		Statement: fmt.Sprintf("%s%s", subscription.Spec.OTTL, appendedSpaces),
		Pipelines: []string{pipelineName},
	}

	return newItem
}

func (cfgInput *OtelColConfigInput) generateRoutingConnectorForTenantsSubscriptions(tenantName string, subscriptionNames []v1alpha1.NamespacedName) RoutingConnector {
	rcName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName)
	rc := newRoutingConnector(rcName)

	slices.SortFunc(subscriptionNames, func(a, b v1alpha1.NamespacedName) int {
		return strings.Compare(a.String(), b.String())
	})

	for index, subscriptionRef := range subscriptionNames {

		subscription := cfgInput.Subscriptions[subscriptionRef]

		tableItem := buildRoutingTableItemForSubscription(tenantName, subscription, index)
		rc.AddRoutingConnectorTableElem(tableItem)
	}

	return rc
}

func (cfgInput *OtelColConfigInput) generateRoutingConnectorForSubscriptionsOutputs(subscriptionRef v1alpha1.NamespacedName, outputNames []v1alpha1.NamespacedName) RoutingConnector {
	rcName := fmt.Sprintf("routing/subscription_%s_%s_outputs", subscriptionRef.Namespace, subscriptionRef.Name)
	rc := newRoutingConnector(rcName)

	slices.SortFunc(outputNames, func(a, b v1alpha1.NamespacedName) int {
		return strings.Compare(a.String(), b.String())
	})

	pipelines := []string{}

	for _, outputRef := range outputNames {
		pipelines = append(pipelines, fmt.Sprintf("logs/output_%s_%s_%s_%s", subscriptionRef.Namespace, subscriptionRef.Name, outputRef.Namespace, outputRef.Name))
	}

	tableItem := RoutingConnectorTableItem{
		Statement: "route()",
		Pipelines: pipelines,
	}

	rc.AddRoutingConnectorTableElem(tableItem)

	return rc
}

func (cfgInput *OtelColConfigInput) generateConnectors() map[string]any {
	var connectors = make(map[string]any)

	rootRoutingConnector := generateRootRoutingConnector(cfgInput.Tenants)
	connectors[rootRoutingConnector.Name] = rootRoutingConnector

	for _, tenant := range cfgInput.Tenants {
		rc := cfgInput.generateRoutingConnectorForTenantsSubscriptions(tenant.Name, cfgInput.TenantSubscriptionMap[tenant.Name])
		connectors[rc.Name] = rc
	}

	for _, subscription := range cfgInput.Subscriptions {
		rc := cfgInput.generateRoutingConnectorForSubscriptionsOutputs(subscription.NamespacedName(), cfgInput.SubscriptionOutputMap[subscription.NamespacedName()])
		connectors[rc.Name] = rc
	}

	return connectors

}
func (cfgInput *OtelColConfigInput) generateProcessorMemoryLimiter() map[string]any {
	var memoryLimiter = make(map[string]any)

	if memLimit := cfgInput.MemoryLimit.Limits[corev1.ResourceLimitsMemory]; !memLimit.IsZero() {
		memoryLimiter["check_interval"] = "1s"
		// From memorylimiterprocessor's README
		// > Note that typically the total memory usage of process will be about 50MiB higher than this value.
		// Because of this, 50MiB will be subtracted from memLimit
		limit := memLimit
		memoryOverhead := resource.MustParse("50Mi")
		limit.Sub(memoryOverhead)

		memoryLimiter["limit_mib"] = limit.Value() / 1024 / 1024
	}

	return memoryLimiter
}
func (cfgInput *OtelColConfigInput) generateProcessors() map[string]any {
	var processors = make(map[string]any)

	k8sProcessorName := "k8sattributes"
	processors[k8sProcessorName] = cfgInput.generateDefaultKubernetesProcessor()

	if cfgInput.MemoryLimit != nil {
		processors["memory_limiter"] = cfgInput.generateProcessorMemoryLimiter()
	}

	for _, tenant := range cfgInput.Tenants {
		processors[fmt.Sprintf("attributes/tenant_%s", tenant.Name)] = generateTenantAttributeProcessor(tenant)
	}

	for _, subscription := range cfgInput.Subscriptions {
		processors[fmt.Sprintf("attributes/subscription_%s", subscription.Name)] = generateSubscriptionAttributeProcessor(subscription)
	}

	for _, output := range cfgInput.Outputs {
		if output.Spec.Loki != nil {
			processors[fmt.Sprintf("attributes/loki_exporter_%s", output.Name)] = generateLokiExporterAttributeProcessor()
			processors[fmt.Sprintf("resource/loki_exporter_%s", output.Name)] = generateLokiExporterResourceProcessor()
		}
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

func generateLokiExporterAttributeProcessor() AttributesProcessor {
	processor := AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action:        "insert",
				Key:           "loki.tenant",
				FromAttribute: "tenant_name",
			},
			{
				Action: "insert",
				Key:    "loki.attribute.labels",
				Value:  "tenant_name",
			},
		},
	}
	return processor
}

func generateLokiExporterResourceProcessor() ResourceProcessor {
	processor := ResourceProcessor{
		Actions: []ResourceProcessorAction{
			{
				Action: "insert",
				Key:    "loki.resource.labels",
				Value:  "k8s.pod.name, k8s.namespace.name",
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
	namedPipelines["logs/all"] = generateRootPipeline()

	tenants := []string{}
	for tenant := range cfgInput.TenantSubscriptionMap {
		tenants = append(tenants, tenant)
	}

	for _, tenant := range tenants {
		// Generate a pipeline for the tenant
		tenantPipelineName := fmt.Sprintf("logs/tenant_%s", tenant)
		tenantRoutingName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenant)
		namedPipelines[tenantPipelineName] = generatePipeline([]string{"routing/tenants"}, []string{fmt.Sprintf("attributes/tenant_%s", tenant)}, []string{tenantRoutingName})

		// Generate pipelines for the subscriptions for the tenant
		for _, subscription := range cfgInput.TenantSubscriptionMap[tenant] {
			tenantSubscriptionPipelineName := fmt.Sprintf("%s_subscription_%s_%s", tenantPipelineName, subscription.Namespace, subscription.Name)

			namedPipelines[tenantSubscriptionPipelineName] = generatePipeline([]string{tenantRoutingName}, []string{fmt.Sprintf("attributes/subscription_%s", subscription.Name)}, []string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)})

			for _, outputRef := range cfgInput.SubscriptionOutputMap[subscription] {
				outputPipelineName := fmt.Sprintf("logs/output_%s_%s_%s_%s", subscription.Namespace, subscription.Name, outputRef.Namespace, outputRef.Name)

				idx := slices.IndexFunc(cfgInput.Outputs, func(elem v1alpha1.OtelOutput) bool {
					return outputRef == elem.NamespacedName()
				})

				if idx != -1 {
					output := cfgInput.Outputs[idx]

					if output.Spec.Loki != nil {
						namedPipelines[outputPipelineName] = generatePipeline([]string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)}, []string{fmt.Sprintf("attributes/loki_exporter_%s", output.Name), fmt.Sprintf("resource/loki_exporter_%s", output.Name)}, []string{fmt.Sprintf("loki/%s_%s", output.Namespace, output.Name)})
					}

					if output.Spec.OTLP != nil {
						namedPipelines[outputPipelineName] = generatePipeline([]string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)}, []string{}, []string{fmt.Sprintf("otlp/%s_%s", output.Namespace, output.Name)})
					}
				}
			}
		}

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

	var defaultLabels = []map[string]string{
		{
			"from":      "pod",
			"tag_name":  "all_labels",
			"key_regex": ".*",
		},
	}

	k8sProcessor := map[string]any{
		"auth_type":   "serviceAccount",
		"passthrough": false,
		"extract": map[string]any{
			"metadata": defaultMetadata,
			"labels":   defaultLabels,
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

func (cfgInput *OtelColConfigInput) ToIntermediateRepresentation(ctx context.Context) *OtelColConfigIR {
	result := OtelColConfigIR{}

	// Get  outputs based tenant names
	result.Exporters = cfgInput.generateExporters(ctx)

	// Add processors
	result.Processors = cfgInput.generateProcessors()

	result.Receivers = make(map[string]any)
	k8sReceiverName := "filelog/kubernetes" // only one instance for now
	result.Receivers[k8sReceiverName] = cfgInput.generateDefaultKubernetesReceiver()

	result.Connectors = cfgInput.generateConnectors()
	result.Services.Pipelines.NamedPipelines = make(map[string]Pipeline)

	result.Services.Pipelines.NamedPipelines = cfgInput.generateNamedPipelines()

	if cfgInput.Debug {
		result.Services.Telemetry = map[string]any{
			"logs": map[string]string{"level": "debug"},
		}
	} else {
		result.Services.Telemetry = map[string]any{}
	}
	if _, ok := result.Processors["memory_limiter"]; ok {
		for name, namedPipeline := range result.Services.Pipelines.NamedPipelines {
			// From memorylimiterprocessor's README:
			// > For the memory_limiter processor, the best practice is to add it as the first processor in a pipeline.
			processors := []string{"memory_limiter"}
			processors = append(processors, namedPipeline.Processors...)
			namedPipeline.Processors = processors
			result.Services.Pipelines.NamedPipelines[name] = namedPipeline
		}
	}

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
