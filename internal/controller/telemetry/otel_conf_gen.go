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
	"time"

	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configopaque"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

type OtelColConfigInput struct {
	// These must only include resources that are selected by the collector, tenant labelselectors, and listed outputs in the subscriptions
	Tenants       []v1alpha1.Tenant
	Subscriptions map[v1alpha1.NamespacedName]v1alpha1.Subscription
	Outputs       []v1alpha1.OtelOutput
	MemoryLimiter v1alpha1.MemoryLimiter

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

type CountConnectorAttributeConfig struct {
	Key          string `yaml:"key,omitempty"`
	DefaultValue string `yaml:"default_value,omitempty"`
}

type CountConnectorMetricInfo struct {
	Description        string                          `yaml:"description,omitempty"`
	Conditions         []string                        `yaml:"conditions,omitempty"`
	Attributes         []CountConnectorAttributeConfig `yaml:"attributes,omitempty"`
	ResourceAttributes []CountConnectorAttributeConfig `yaml:"resource_attributes,omitempty"`
}

type DeltatoCumulativeConfig struct {
	MaxStale   time.Duration `yaml:"max_stale,omitempty"`
	MaxStreams int           `yaml:"max_streams,omitempty"`
}

type PrometheusExporterConfig struct {
	HTTPServerConfig `yaml:",inline"`

	// Namespace if set, exports metrics under the provided value.
	Namespace string `yaml:"namespace,omitempty"`

	// ConstLabels are values that are applied for every exported metric.
	ConstLabels prometheus.Labels `yaml:"const_labels,omitempty"`

	// SendTimestamps will send the underlying scrape timestamp with the export
	SendTimestamps bool `yaml:"send_timestamps,omitempty"`

	// MetricExpiration defines how long metrics are kept without updates
	MetricExpiration time.Duration `yaml:"metric_expiration,omitempty"`

	// ResourceToTelemetrySettings defines configuration for converting resource attributes to metric labels.
	ResourceToTelemetrySettings resourcetotelemetry.Settings `yaml:"resource_to_telemetry_conversion,omitempty"`

	// EnableOpenMetrics enables the use of the OpenMetrics encoding option for the prometheus exporter.
	EnableOpenMetrics bool `yaml:"enable_open_metrics,omitempty"`

	// AddMetricSuffixes controls whether suffixes are added to metric names. Defaults to true.
	AddMetricSuffixes bool `yaml:"add_metric_suffixes,omitempty"`
}

type HTTPServerConfig struct {
	// Endpoint configures the listening address for the server.
	Endpoint string `yaml:"endpoint,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *TLSServerConfig `yaml:"tls,omitempty"`

	// CORS configures the server for HTTP cross-origin resource sharing (CORS).
	CORS *CORSConfig `yaml:"cors,omitempty"`

	// Auth for this receiver
	Auth *configauth.Authentication `yaml:"auth,omitempty"`

	// MaxRequestBodySize sets the maximum request body size in bytes
	MaxRequestBodySize int64 `yaml:"max_request_body_size,omitempty"`

	// IncludeMetadata propagates the client metadata from the incoming requests to the downstream consumers
	// Experimental: *NOTE* this option is subject to change or removal in the future.
	IncludeMetadata bool `yaml:"include_metadata,omitempty"`

	// Additional headers attached to each HTTP response sent to the client.
	// Header values are opaque since they may be sensitive.
	ResponseHeaders map[string]configopaque.String `yaml:"response_headers,omitempty"`
}

type TLSServerConfig struct {
	// squash ensures fields are correctly decoded in embedded struct.
	v1alpha1.TLSSetting `yaml:",squash"`

	// These are config options specific to server connections.

	// Path to the TLS cert to use by the server to verify a client certificate. (optional)
	// This sets the ClientCAs and ClientAuth to RequireAndVerifyClientCert in the TLSConfig. Please refer to
	// https://godoc.org/crypto/tls#Config for more information. (optional)
	ClientCAFile string `yaml:"client_ca_file"`

	// Reload the ClientCAs file when it is modified
	// (optional, default false)
	ReloadClientCAFile bool `yaml:"client_ca_file_reload"`
}

// CORSConfig configures a receiver for HTTP cross-origin resource sharing (CORS).
// See the underlying https://github.com/rs/cors package for details.
type CORSConfig struct {
	// AllowedOrigins sets the allowed values of the Origin header for
	// HTTP/JSON requests to an OTLP receiver. An origin may contain a
	// wildcard (*) to replace 0 or more characters (e.g.,
	// "http://*.domain.com", or "*" to allow any origin).
	AllowedOrigins []string `yaml:"allowed_origins"`

	// AllowedHeaders sets what headers will be allowed in CORS requests.
	// The Accept, Accept-Language, Content-Type, and Content-Language
	// headers are implicitly allowed. If no headers are listed,
	// X-Requested-With will also be accepted by default. Include "*" to
	// allow any request header.
	AllowedHeaders []string `yaml:"allowed_headers"`

	// MaxAge sets the value of the Access-Control-Max-Age response header.
	// Set it to the number of seconds that browsers should cache a CORS
	// preflight response for.
	MaxAge int `yaml:"max_age"`
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

type TransformProcessor struct {
	LogStatements []Statement `yaml:"log_statements,omitempty"`
}

type Statement struct {
	Context    string   `yaml:"context"`
	Statements []string `yaml:"statements"`
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

	maps.Copy(exporters, generateMetricsExporters())
	maps.Copy(exporters, cfgInput.generateOTLPExporters(ctx))
	maps.Copy(exporters, cfgInput.generateLokiExporters(ctx))
	maps.Copy(exporters, cfgInput.generateFluentforwardExporters(ctx))
	exporters["logging/debug"] = map[string]any{
		"verbosity": "detailed",
	}
	return exporters
}

func generateMetricsExporters() map[string]any {
	metricsExporters := make(map[string]any)

	defaultPrometheusExporterConfig := PrometheusExporterConfig{
		HTTPServerConfig: HTTPServerConfig{
			Endpoint: "0.0.0.0:9999",
		},
	}

	metricsExporters["prometheus/message_metrics_exporter"] = defaultPrometheusExporterConfig

	return metricsExporters
}

func (cfgInput *OtelColConfigInput) generateOTLPExporters(ctx context.Context) map[string]any {
	logger := log.FromContext(ctx)
	result := make(map[string]any)

	for _, output := range cfgInput.Outputs {
		if output.Spec.OTLP != nil {
			name := GetExporterNameForOtelOutput(output)
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

	result := make(map[string]any)

	for _, output := range cfgInput.Outputs {
		if output.Spec.Loki != nil {

			name := GetExporterNameForOtelOutput(output)
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

func (cfgInput *OtelColConfigInput) generateFluentforwardExporters(ctx context.Context) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)

	for _, output := range cfgInput.Outputs {
		if output.Spec.Fluentforward != nil {
			// TODO: add proper error handling
			name := fmt.Sprintf("fluentforwardexporter/%s_%s", output.Namespace, output.Name)
			fluentForwardMarshaled, err := yaml.Marshal(output.Spec.Fluentforward)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.NamespacedName().String())
			}
			var fluetForwardValues map[string]any
			if err := yaml.Unmarshal(fluentForwardMarshaled, &fluetForwardValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.NamespacedName().String())
			}

			result[name] = fluetForwardValues
		}
	}

	return result
}

func generatePipeline(receivers, processors, exporters []string) Pipeline {
	result := Pipeline{}

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
	connectors := make(map[string]any)

	countConnectors := generateCountConnectors()
	maps.Copy(connectors, countConnectors)

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
	memoryLimiter := make(map[string]any)

	memoryLimiter["check_interval"] = cfgInput.MemoryLimiter.CheckInterval.String()
	if cfgInput.MemoryLimiter.MemoryLimitMiB != 0 {
		memoryLimiter["limit_mib"] = cfgInput.MemoryLimiter.MemoryLimitMiB
	}
	if cfgInput.MemoryLimiter.MemorySpikeLimitMiB != 0 {
		memoryLimiter["spike_limit_mib"] = cfgInput.MemoryLimiter.MemorySpikeLimitMiB
	}
	if cfgInput.MemoryLimiter.MemoryLimitPercentage != 0 {
		memoryLimiter["limit_percentage"] = cfgInput.MemoryLimiter.MemoryLimitPercentage
	}
	if cfgInput.MemoryLimiter.MemorySpikePercentage != 0 {
		memoryLimiter["spike_limit_percentage"] = cfgInput.MemoryLimiter.MemorySpikePercentage
	}
	return memoryLimiter
}

func generateCountConnectors() map[string]any {
	countConnectors := make(map[string]any)

	countConnectors["count/tenant_metrics"] = map[string]any{
		"logs": map[string]CountConnectorMetricInfo{
			"telemetry_controller_tenant_log_count": {
				Description: "The number of logs from each tenant pipeline.",
				Attributes: []CountConnectorAttributeConfig{{
					Key: "tenant",
				}},
				ResourceAttributes: []CountConnectorAttributeConfig{
					{
						Key: "k8s.namespace.name",
					},
					{
						Key: "k8s.node.name",
					},
					{
						Key: "k8s.container.name",
					},
					{
						Key: "k8s.pod.name",
					},
					{
						Key: "k8s.pod.labels.app.kubernetes.io/name",
					},
				},
			},
		},
	}

	countConnectors["count/output_metrics"] = map[string]any{
		"logs": map[string]CountConnectorMetricInfo{
			"telemetry_controller_output_log_count": {
				Description: "The number of logs sent out from each exporter.",
				Attributes: []CountConnectorAttributeConfig{
					{
						Key: "tenant",
					}, {
						Key: "subscription",
					}, {
						Key: "exporter",
					}},
				ResourceAttributes: []CountConnectorAttributeConfig{
					{
						Key: "k8s.namespace.name",
					},
					{
						Key: "k8s.node.name",
					},
					{
						Key: "k8s.container.name",
					},
					{
						Key: "k8s.pod.name",
					},
					{
						Key: "k8s.pod.labels.app.kubernetes.io/name",
					},
				},
			},
		},
	}

	return countConnectors
}

func (cfgInput *OtelColConfigInput) generateProcessors() map[string]any {
	processors := make(map[string]any)

	k8sProcessorName := "k8sattributes"
	processors[k8sProcessorName] = cfgInput.generateDefaultKubernetesProcessor()

	processors["memory_limiter"] = cfgInput.generateProcessorMemoryLimiter()
	metricsProcessors := generateMetricsProcessors()

	maps.Copy(processors, metricsProcessors)

	for _, tenant := range cfgInput.Tenants {
		processors[fmt.Sprintf("attributes/tenant_%s", tenant.Name)] = generateTenantAttributeProcessor(tenant)
	}

	for _, subscription := range cfgInput.Subscriptions {
		processors[fmt.Sprintf("attributes/subscription_%s", subscription.Name)] = generateSubscriptionAttributeProcessor(subscription)
	}

	for _, output := range cfgInput.Outputs {
		processors[fmt.Sprintf("attributes/exporter_name_%s", output.Name)] = generateOutputExporterNameProcessor(output)
		if output.Spec.Loki != nil {
			processors[fmt.Sprintf("attributes/loki_exporter_%s", output.Name)] = generateLokiExporterAttributeProcessor()
			processors[fmt.Sprintf("resource/loki_exporter_%s", output.Name)] = generateLokiExporterResourceProcessor()
		}
	}

	return processors
}

func generateOutputExporterNameProcessor(output v1alpha1.OtelOutput) AttributesProcessor {
	processor := AttributesProcessor{
		Actions: []AttributesProcessorAction{{
			Action: "insert",
			Key:    "exporter",
			Value:  GetExporterNameForOtelOutput(output),
		}},
	}

	return processor
}

func generateMetricsProcessors() map[string]any {
	metricsProcessors := make(map[string]any)

	metricsProcessors["deltatocumulative"] = DeltatoCumulativeConfig{}

	metricsProcessors["attributes/metricattributes"] = AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action:        "insert",
				Key:           "app",
				FromAttribute: "k8s.pod.labels.app.kubernetes.io/name",
			},
			{
				Action:        "insert",
				Key:           "host",
				FromAttribute: "k8s.node.name",
			},
			{
				Action:        "insert",
				Key:           "namespace",
				FromAttribute: "k8s.namespace.name",
			},
			{
				Action:        "insert",
				Key:           "container",
				FromAttribute: "k8s.container.name",
			},
			{
				Action:        "insert",
				Key:           "pod",
				FromAttribute: "k8s.pod.name",
			},
		},
	}

	return metricsProcessors
}

func generateTenantAttributeProcessor(tenant v1alpha1.Tenant) AttributesProcessor {
	processor := AttributesProcessor{
		Actions: []AttributesProcessorAction{
			{
				Action: "insert",
				Key:    "tenant",
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
				Key:    "subscription",
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
				FromAttribute: "tenant",
			},
			{
				Action: "insert",
				Key:    "loki.attribute.labels",
				Value:  "tenant",
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

func generateRootPipeline(tenantName string) Pipeline {
	tenantCountConnectorName := "count/tenant_metrics"
	receiverName := fmt.Sprintf("filelog/%s", tenantName)
	exporterName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName)
	return generatePipeline([]string{receiverName}, []string{"k8sattributes", fmt.Sprintf("attributes/tenant_%s", tenantName)}, []string{exporterName, tenantCountConnectorName})
}

func (cfgInput *OtelColConfigInput) generateNamedPipelines() map[string]Pipeline {
	outputCountConnectorName := "count/output_metrics"
	var namedPipelines = make(map[string]Pipeline)

	tenants := []string{}
	for tenant := range cfgInput.TenantSubscriptionMap {
		tenantRootPipeline := fmt.Sprintf("logs/tenant_%s", tenant)
		namedPipelines[tenantRootPipeline] = generateRootPipeline(tenant)
		tenants = append(tenants, tenant)
	}

	metricsPipelines := generateMetricsPipelines()
	maps.Copy(namedPipelines, metricsPipelines)

	for _, tenant := range tenants {
		// Generate a pipeline for the tenant
		tenantRootPipeline := fmt.Sprintf("logs/tenant_%s", tenant)
		tenantRoutingName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenant)
		namedPipelines[tenantRootPipeline] = generateRootPipeline(tenant)

		// Generate pipelines for the subscriptions for the tenant
		for _, subscription := range cfgInput.TenantSubscriptionMap[tenant] {
			tenantSubscriptionPipelineName := fmt.Sprintf("%s_subscription_%s_%s", tenantRootPipeline, subscription.Namespace, subscription.Name)

			namedPipelines[tenantSubscriptionPipelineName] = generatePipeline([]string{tenantRoutingName}, []string{fmt.Sprintf("attributes/subscription_%s", subscription.Name)}, []string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)})

			for _, outputRef := range cfgInput.SubscriptionOutputMap[subscription] {
				outputPipelineName := fmt.Sprintf("logs/output_%s_%s_%s_%s", subscription.Namespace, subscription.Name, outputRef.Namespace, outputRef.Name)

				idx := slices.IndexFunc(cfgInput.Outputs, func(elem v1alpha1.OtelOutput) bool {
					return outputRef == elem.NamespacedName()
				})

				if idx != -1 {
					output := cfgInput.Outputs[idx]

					receivers := []string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)}
					processors := []string{fmt.Sprintf("attributes/exporter_name_%s", output.Name)}
					var exporters []string

					if output.Spec.Loki != nil {
						processors = append(processors,
							fmt.Sprintf("attributes/loki_exporter_%s", output.Name),
							fmt.Sprintf("resource/loki_exporter_%s", output.Name))
						exporters = []string{GetExporterNameForOtelOutput(output), outputCountConnectorName}
					}

					if output.Spec.OTLP != nil {
						exporters = []string{GetExporterNameForOtelOutput(output), outputCountConnectorName}
					}

					if output.Spec.Fluentforward != nil {
						exporters = []string{GetExporterNameForOtelOutput(output), outputCountConnectorName}
					}
					if cfgInput.Debug {
						exporters = append(exporters, "logging/debug")
					}

					namedPipelines[outputPipelineName] = generatePipeline(receivers, processors, exporters)
				}
			}
		}

	}

	return namedPipelines
}

func generateMetricsPipelines() map[string]Pipeline {
	metricsPipelines := make(map[string]Pipeline)

	metricsPipelines["metrics/tenant"] = Pipeline{
		Receivers:  []string{"count/tenant_metrics"},
		Processors: []string{"deltatocumulative", "attributes/metricattributes"},
		Exporters:  []string{"prometheus/message_metrics_exporter"},
	}

	metricsPipelines["metrics/output"] = Pipeline{
		Receivers:  []string{"count/output_metrics"},
		Processors: []string{"deltatocumulative", "attributes/metricattributes"},
		Exporters:  []string{"prometheus/message_metrics_exporter"},
	}

	return metricsPipelines
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

	defaultPodAssociation := []map[string]any{
		{"sources": defaultSources},
	}

	defaultMetadata := []string{
		"k8s.pod.name",
		"k8s.pod.uid",
		"k8s.deployment.name",
		"k8s.namespace.name",
		"k8s.node.name",
		"k8s.pod.start_time",
	}

	defaultLabels := []map[string]string{
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

func (cfgInput *OtelColConfigInput) generateDefaultKubernetesReceiver(namespaces []string) map[string]any {

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

	includeList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		include := fmt.Sprintf("/var/log/pods/%s_*/*/*.log", ns)
		includeList = append(includeList, include)
	}

	k8sReceiver := map[string]any{
		"include":           includeList,
		"exclude":           []string{"/var/log/pods/*/otc-container/*.log"},
		"start_at":          "end",
		"include_file_path": true,
		"include_file_name": false,
		"operators":         operators,
		"retry_on_failure": map[string]any{
			"enabled":          true,
			"max_elapsed_time": 0,
		},
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
	for tenantName := range cfgInput.TenantSubscriptionMap {
		if tenantIdx := slices.IndexFunc(cfgInput.Tenants, func(t v1alpha1.Tenant) bool {
			return tenantName == t.Name
		}); tenantIdx != -1 {
			k8sReceiverName := fmt.Sprintf("filelog/%s", tenantName)
			namespaces := cfgInput.Tenants[tenantIdx].Status.LogSourceNamespaces
			result.Receivers[k8sReceiverName] = cfgInput.generateDefaultKubernetesReceiver(namespaces)
		}
	}

	result.Connectors = cfgInput.generateConnectors()
	result.Services.Pipelines.NamedPipelines = make(map[string]Pipeline)

	result.Services.Pipelines.NamedPipelines = cfgInput.generateNamedPipelines()
	result.Services.Telemetry = make(map[string]any)

	result.Services.Telemetry = make(map[string]any)

	result.Services.Telemetry["metrics"] = map[string]string{
		"level": "detailed",
	}

	if cfgInput.Debug {
		result.Services.Telemetry["logs"] = map[string]string{
			"level": "debug",
		}
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
