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

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configopaque"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

const (
	defaultBasicAuthUsernameField = "username"
	defaultBasicAuthPasswordField = "password"
	defaultBearerAuthTokenField   = "token"
)

type OtelColConfigInput struct {
	// These must only include resources that are selected by the collector, tenant labelselectors, and listed outputs in the subscriptions
	Tenants               []v1alpha1.Tenant
	Subscriptions         map[v1alpha1.NamespacedName]v1alpha1.Subscription
	Bridges               []v1alpha1.Bridge
	OutputsWithSecretData []OutputWithSecretData
	MemoryLimiter         v1alpha1.MemoryLimiter

	// Subscriptions map, where the key is the Tenants' name, value is a slice of subscriptions' namespaced name
	TenantSubscriptionMap map[string][]v1alpha1.NamespacedName
	SubscriptionOutputMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
	Debug                 bool
}

type OutputWithSecretData struct {
	Output v1alpha1.Output
	Secret corev1.Secret
}

type RoutingConnectorTableItem struct {
	Statement string   `json:"statement"`
	Pipelines []string `json:"pipelines"`
}

type RoutingConnector struct {
	Name             string                      `json:"-"`
	DefaultPipelines []string                    `json:"default_pipelines,omitempty"`
	Table            []RoutingConnectorTableItem `json:"table"`
}

type AttributesProcessor struct {
	Actions []AttributesProcessorAction `json:"actions"`
}

type AttributesProcessorAction struct {
	Action        string `json:"action"`
	Key           string `json:"key"`
	Value         string `json:"value,omitempty"`
	FromAttribute string `json:"from_attribute,omitempty"`
	FromContext   string `json:"from_context,omitempty"`
}

type CountConnectorAttributeConfig struct {
	Key          string `json:"key,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
}

type CountConnectorMetricInfo struct {
	Description        string                          `json:"description,omitempty"`
	Conditions         []string                        `json:"conditions,omitempty"`
	Attributes         []CountConnectorAttributeConfig `json:"attributes,omitempty"`
	ResourceAttributes []CountConnectorAttributeConfig `json:"resource_attributes,omitempty"`
}

type DeltatoCumulativeConfig struct {
	MaxStale   time.Duration `json:"max_stale,omitempty"`
	MaxStreams int           `json:"max_streams,omitempty"`
}

type PrometheusExporterConfig struct {
	HTTPServerConfig `json:",inline"`

	// Namespace if set, exports metrics under the provided value.
	Namespace string `json:"namespace,omitempty"`

	// ConstLabels are values that are applied for every exported metric.
	ConstLabels prometheus.Labels `json:"const_labels,omitempty"`

	// SendTimestamps will send the underlying scrape timestamp with the export
	SendTimestamps bool `json:"send_timestamps,omitempty"`

	// MetricExpiration defines how long metrics are kept without updates
	MetricExpiration time.Duration `json:"metric_expiration,omitempty"`

	// ResourceToTelemetrySettings defines configuration for converting resource attributes to metric labels.
	ResourceToTelemetrySettings ResourceToTelemetrySettings `json:"resource_to_telemetry_conversion,omitempty"`

	// EnableOpenMetrics enables the use of the OpenMetrics encoding option for the prometheus exporter.
	EnableOpenMetrics bool `json:"enable_open_metrics,omitempty"`

	// AddMetricSuffixes controls whether suffixes are added to metric names. Defaults to true.
	AddMetricSuffixes bool `json:"add_metric_suffixes,omitempty"`
}

type ResourceToTelemetrySettings struct {
	// Enabled indicates whether to convert resource attributes to telemetry attributes. Default is `false`.
	Enabled bool `json:"enabled,omitempty"`
}

type HTTPServerConfig struct {
	// Endpoint configures the listening address for the server.
	Endpoint string `json:"endpoint,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *TLSServerConfig `json:"tls,omitempty"`

	// CORS configures the server for HTTP cross-origin resource sharing (CORS).
	CORS *CORSConfig `json:"cors,omitempty"`

	// Auth for this receiver
	Auth *configauth.Authentication `json:"auth,omitempty"`

	// MaxRequestBodySize sets the maximum request body size in bytes
	MaxRequestBodySize int64 `json:"max_request_body_size,omitempty"`

	// IncludeMetadata propagates the client metadata from the incoming requests to the downstream consumers
	// Experimental: *NOTE* this option is subject to change or removal in the future.
	IncludeMetadata bool `json:"include_metadata,omitempty"`

	// Additional headers attached to each HTTP response sent to the client.
	// Header values are opaque since they may be sensitive.
	ResponseHeaders map[string]configopaque.String `json:"response_headers,omitempty"`
}

type TLSServerConfig struct {
	// squash ensures fields are correctly decoded in embedded struct.
	v1alpha1.TLSSetting `json:",inline"`

	// These are config options specific to server connections.

	// Path to the TLS cert to use by the server to verify a client certificate. (optional)
	// This sets the ClientCAs and ClientAuth to RequireAndVerifyClientCert in the TLSConfig. Please refer to
	// https://godoc.org/crypto/tls#Config for more information. (optional)
	ClientCAFile string `json:"client_ca_file"`

	// Reload the ClientCAs file when it is modified
	// (optional, default false)
	ReloadClientCAFile bool `json:"client_ca_file_reload"`
}

// CORSConfig configures a receiver for HTTP cross-origin resource sharing (CORS).
// See the underlying https://github.com/rs/cors package for details.
type CORSConfig struct {
	// AllowedOrigins sets the allowed values of the Origin header for
	// HTTP/JSON requests to an OTLP receiver. An origin may contain a
	// wildcard (*) to replace 0 or more characters (e.g.,
	// "http://*.domain.com", or "*" to allow any origin).
	AllowedOrigins []string `json:"allowed_origins"`

	// AllowedHeaders sets what headers will be allowed in CORS requests.
	// The Accept, Accept-Language, Content-Type, and Content-Language
	// headers are implicitly allowed. If no headers are listed,
	// X-Requested-With will also be accepted by default. Include "*" to
	// allow any request header.
	AllowedHeaders []string `json:"allowed_headers"`

	// MaxAge sets the value of the Access-Control-Max-Age response header.
	// Set it to the number of seconds that browsers should cache a CORS
	// preflight response for.
	MaxAge int `json:"max_age"`
}

type ResourceProcessor struct {
	Actions []ResourceProcessorAction `json:"attributes"`
}

type ResourceProcessorAction struct {
	Action        string `json:"action"`
	Key           string `json:"key"`
	Value         string `json:"value,omitempty"`
	FromAttribute string `json:"from_attribute,omitempty"`
	FromContext   string `json:"from_context,omitempty"`
}

type ErrorMode string

const (
	ErrorModeIgnore    ErrorMode = "ignore"
	ErrorModeSilent    ErrorMode = "silent"
	ErrorModePropagate ErrorMode = "propagate"
)

type TransformProcessor struct {
	ErrorMode        ErrorMode   `json:"error_mode,omitempty"`
	TraceStatements  []Statement `json:"trace_statements,omitempty"`
	MetricStatements []Statement `json:"metric_statements,omitempty"`
	LogStatements    []Statement `json:"log_statements,omitempty"`
}

type Statement struct {
	Context    string   `json:"context"`
	Conditions []string `json:"conditions"`
	Statements []string `json:"statements"`
}

type Pipeline struct {
	Receivers  []string `json:"receivers,omitempty"`
	Processors []string `json:"processors,omitempty"`
	Exporters  []string `json:"exporters,omitempty"`
}

type Pipelines struct {
	Traces         Pipeline            `json:"traces,omitempty"`
	Metrics        Pipeline            `json:"metrics,omitempty"`
	Logs           Pipeline            `json:"logs,omitempty"`
	NamedPipelines map[string]Pipeline `json:",inline,omitempty"`
}

type Services struct {
	Extensions map[string]any `json:"extensions,omitempty"`
	Pipelines  Pipelines      `json:"pipelines,omitempty"`
	Telemetry  map[string]any `json:"telemetry,omitempty"`
}

type OtelColConfigIR struct {
	Receivers  map[string]any `json:"receivers,omitempty"`
	Exporters  map[string]any `json:"exporters,omitempty"`
	Processors map[string]any `json:"processors,omitempty"`
	Connectors map[string]any `json:"connectors,omitempty"`
	Services   Services       `json:"service,omitempty"`
}

func (cfgInput *OtelColConfigInput) generateExporters(ctx context.Context) map[string]any {
	exporters := map[string]any{}

	maps.Copy(exporters, generateMetricsExporters())
	maps.Copy(exporters, cfgInput.generateOTLPGRPCExporters(ctx))
	maps.Copy(exporters, cfgInput.generateOTLPHTTPExporters(ctx))
	maps.Copy(exporters, cfgInput.generateFluentforwardExporters(ctx))
	exporters["debug"] = map[string]any{
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

func (cfgInput *OtelColConfigInput) generateOTLPGRPCExporters(ctx context.Context) map[string]any {
	logger := log.FromContext(ctx)
	result := make(map[string]any)

	for _, output := range cfgInput.OutputsWithSecretData {
		if output.Output.Spec.OTLPGRPC != nil {
			name := GetExporterNameForOutput(output.Output)

			if output.Output.Spec.Authentication != nil {
				if output.Output.Spec.Authentication.BasicAuth != nil {
					output.Output.Spec.OTLPGRPC.Auth = &v1alpha1.Authentication{
						AuthenticatorID: fmt.Sprintf("basicauth/%s_%s", output.Output.Namespace, output.Output.Name)}
				} else if output.Output.Spec.Authentication.BearerAuth != nil {
					output.Output.Spec.OTLPGRPC.Auth = &v1alpha1.Authentication{
						AuthenticatorID: fmt.Sprintf("bearertokenauth/%s_%s", output.Output.Namespace, output.Output.Name)}
				}
			}
			otlpGrpcValuesMarshaled, err := yaml.Marshal(output.Output.Spec.OTLPGRPC)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}
			var otlpGrpcValues map[string]any
			if err := yaml.Unmarshal(otlpGrpcValuesMarshaled, &otlpGrpcValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}

			result[name] = otlpGrpcValues
		}
	}

	return result
}

func (cfgInput *OtelColConfigInput) generateOTLPHTTPExporters(ctx context.Context) map[string]any {
	logger := log.FromContext(ctx)
	result := make(map[string]any)

	for _, output := range cfgInput.OutputsWithSecretData {
		if output.Output.Spec.OTLPHTTP != nil {
			name := GetExporterNameForOutput(output.Output)

			if output.Output.Spec.Authentication != nil {
				if output.Output.Spec.Authentication.BasicAuth != nil {
					output.Output.Spec.OTLPHTTP.Auth.AuthenticatorID = fmt.Sprintf("basicauth/%s_%s", output.Output.Namespace, output.Output.Name)
				} else if output.Output.Spec.Authentication.BearerAuth != nil {
					output.Output.Spec.OTLPHTTP.Auth.AuthenticatorID = fmt.Sprintf("bearertokenauth/%s_%s", output.Output.Namespace, output.Output.Name)
				}
			}
			otlpHttpValuesMarshaled, err := yaml.Marshal(output.Output.Spec.OTLPHTTP)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}
			var otlpHttpValues map[string]any
			if err := yaml.Unmarshal(otlpHttpValuesMarshaled, &otlpHttpValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}

			result[name] = otlpHttpValues
		}
	}

	return result
}

func (cfgInput *OtelColConfigInput) generateFluentforwardExporters(ctx context.Context) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)

	for _, output := range cfgInput.OutputsWithSecretData {
		if output.Output.Spec.Fluentforward != nil {
			// TODO: add proper error handling
			name := fmt.Sprintf("fluentforwardexporter/%s_%s", output.Output.Namespace, output.Output.Name)
			fluentForwardMarshaled, err := yaml.Marshal(output.Output.Spec.Fluentforward)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}
			var fluetForwardValues map[string]any
			if err := yaml.Unmarshal(fluentForwardMarshaled, &fluetForwardValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}

			result[name] = fluetForwardValues
		}
	}

	return result
}

func generatePipeline(receivers, processors, exporters []string) *otelv1beta1.Pipeline {
	result := otelv1beta1.Pipeline{}

	result.Receivers = receivers
	result.Processors = processors
	result.Exporters = exporters

	return &result
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

func (cfgInput *OtelColConfigInput) generateRoutingConnectorForBridge(bridge v1alpha1.Bridge) RoutingConnector {
	rcName := fmt.Sprintf("routing/bridge_%s", bridge.Name)
	rc := newRoutingConnector(rcName)

	tableItem := RoutingConnectorTableItem{
		Statement: bridge.Spec.OTTL,
		Pipelines: []string{fmt.Sprintf("logs/tenant_%s", bridge.Spec.TargetTenant)},
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

	for _, bridge := range cfgInput.Bridges {
		rc := cfgInput.generateRoutingConnectorForBridge(bridge)
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
					{
						Key: "k8s.pod.labels.app",
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
					{
						Key: "k8s.pod.labels.app",
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

		// Add a transform processor if the tenant has one
		if tenant.Spec.Transform.Name != "" {
			processors[fmt.Sprintf("transform/%s", tenant.Spec.Transform.Name)] = generateTransformProcessorForTenant(tenant)
		}
	}

	for _, subscription := range cfgInput.Subscriptions {
		processors[fmt.Sprintf("attributes/subscription_%s", subscription.Name)] = generateSubscriptionAttributeProcessor(subscription)
	}

	for _, output := range cfgInput.OutputsWithSecretData {
		processors[fmt.Sprintf("attributes/exporter_name_%s", output.Output.Name)] = generateOutputExporterNameProcessor(output.Output)
	}

	return processors
}

func generateOutputExporterNameProcessor(output v1alpha1.Output) AttributesProcessor {
	processor := AttributesProcessor{
		Actions: []AttributesProcessorAction{{
			Action: "insert",
			Key:    "exporter",
			Value:  GetExporterNameForOutput(output),
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
				FromAttribute: "k8s.pod.labels.app",
			}, {
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

func generateTransformProcessorForTenant(tenant v1alpha1.Tenant) TransformProcessor {
	return TransformProcessor{
		ErrorMode:        ErrorMode(tenant.Spec.Transform.ErrorMode),
		TraceStatements:  convertAPIStatements(tenant.Spec.Transform.TraceStatements),
		MetricStatements: convertAPIStatements(tenant.Spec.Transform.MetricStatements),
		LogStatements:    convertAPIStatements(tenant.Spec.Transform.LogStatements),
	}
}

func convertAPIStatements(APIStatements []v1alpha1.Statement) []Statement {
	statements := make([]Statement, len(APIStatements))
	for i, statement := range APIStatements {
		statements[i] = Statement{
			Context:    statement.Context,
			Conditions: statement.Conditions,
			Statements: statement.Statements,
		}
	}
	return statements
}

func generateRootPipeline(tenantName string) *otelv1beta1.Pipeline {
	tenantCountConnectorName := "count/tenant_metrics"
	receiverName := fmt.Sprintf("filelog/%s", tenantName)
	exporterName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName)
	return generatePipeline([]string{receiverName}, []string{"k8sattributes", fmt.Sprintf("attributes/tenant_%s", tenantName)}, []string{exporterName, tenantCountConnectorName})
}

func (cfgInput *OtelColConfigInput) generateNamedPipelines() map[string]*otelv1beta1.Pipeline {
	outputCountConnectorName := "count/output_metrics"
	var namedPipelines = make(map[string]*otelv1beta1.Pipeline)

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

		cfgInput.addBridgeConnectorToTenantPipeline(tenant, namedPipelines[tenantRootPipeline], cfgInput.Bridges)
		cfgInput.addTransformProcessorToTenantPipeline(tenant, namedPipelines[tenantRootPipeline])

		// Generate pipelines for the subscriptions for the tenant
		for _, subscription := range cfgInput.TenantSubscriptionMap[tenant] {
			tenantSubscriptionPipelineName := fmt.Sprintf("%s_subscription_%s_%s", tenantRootPipeline, subscription.Namespace, subscription.Name)

			namedPipelines[tenantSubscriptionPipelineName] = generatePipeline([]string{tenantRoutingName}, []string{fmt.Sprintf("attributes/subscription_%s", subscription.Name)}, []string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)})

			for _, outputRef := range cfgInput.SubscriptionOutputMap[subscription] {
				outputPipelineName := fmt.Sprintf("logs/output_%s_%s_%s_%s", subscription.Namespace, subscription.Name, outputRef.Namespace, outputRef.Name)

				idx := slices.IndexFunc(cfgInput.OutputsWithSecretData, func(elem OutputWithSecretData) bool {
					return outputRef == elem.Output.NamespacedName()
				})

				if idx != -1 {
					output := cfgInput.OutputsWithSecretData[idx]

					receivers := []string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)}
					processors := []string{fmt.Sprintf("attributes/exporter_name_%s", output.Output.Name)}
					var exporters []string

					if output.Output.Spec.OTLPGRPC != nil {
						exporters = []string{GetExporterNameForOutput(output.Output), outputCountConnectorName}
					}

					if output.Output.Spec.OTLPHTTP != nil {
						exporters = []string{GetExporterNameForOutput(output.Output), outputCountConnectorName}
					}

					if output.Output.Spec.Fluentforward != nil {
						exporters = []string{GetExporterNameForOutput(output.Output), outputCountConnectorName}
					}
					if cfgInput.Debug {
						exporters = append(exporters, "debug")
					}

					namedPipelines[outputPipelineName] = generatePipeline(receivers, processors, exporters)
				}
			}
		}

	}

	return namedPipelines
}

func generateMetricsPipelines() map[string]*otelv1beta1.Pipeline {
	metricsPipelines := make(map[string]*otelv1beta1.Pipeline)

	metricsPipelines["metrics/tenant"] = &otelv1beta1.Pipeline{
		Receivers:  []string{"count/tenant_metrics"},
		Processors: []string{"deltatocumulative", "attributes/metricattributes"},
		Exporters:  []string{"prometheus/message_metrics_exporter"},
	}

	metricsPipelines["metrics/output"] = &otelv1beta1.Pipeline{
		Receivers:  []string{"count/output_metrics"},
		Processors: []string{"deltatocumulative", "attributes/metricattributes"},
		Exporters:  []string{"prometheus/message_metrics_exporter"},
	}

	return metricsPipelines
}

func (cfgInput *OtelColConfigInput) addTransformProcessorToTenantPipeline(tenantName string, pipeline *otelv1beta1.Pipeline) {
	for _, tenant := range cfgInput.Tenants {
		if tenant.Name == tenantName && tenant.Spec.Transform.Name != "" {
			pipeline.Processors = append(pipeline.Processors, fmt.Sprintf("transform/%s", tenant.Spec.Transform.Name))
		}
	}
}

func checkBridgeConnectorForTenant(tenantName string, bridge v1alpha1.Bridge) (needsReceiver bool, needsExporter bool, bridgeName string) {
	if bridge.Spec.SourceTenant == tenantName {
		needsExporter = true
	}
	if bridge.Spec.TargetTenant == tenantName {
		needsReceiver = true
	}
	bridgeName = bridge.Name

	return
}

func (cfgInput *OtelColConfigInput) addBridgeConnectorToTenantPipeline(tenantName string, pipeline *otelv1beta1.Pipeline, bridges []v1alpha1.Bridge) {
	for _, bridge := range bridges {
		needsReceiver, needsExporter, bridgeName := checkBridgeConnectorForTenant(tenantName, bridge)

		if needsReceiver {
			pipeline.Receivers = append(pipeline.Receivers, fmt.Sprintf("routing/bridge_%s", bridgeName))
		}

		if needsExporter {
			pipeline.Exporters = append(pipeline.Exporters, fmt.Sprintf("routing/bridge_%s", bridgeName))
		}
	}
}

func (cfgInput *OtelColConfigInput) generateDefaultKubernetesProcessor() map[string]any {
	type Source struct {
		Name string `json:"name,omitempty"`
		From string `json:"from,omitempty"`
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

type BasicAuthExtensionConfig struct {
	ClientAuth BasicClientAuthConfig `json:"client_auth,omitempty"`
}

type BasicClientAuthConfig struct {
	// Username holds the username to use for client authentication.
	Username string `json:"username"`
	// Password holds the password to use for client authentication.
	Password string `json:"password"`
}

func generateBasicAuthExtensionsForOutput(output OutputWithSecretData) BasicAuthExtensionConfig {
	var effectiveUsernameField, effectivePasswordField string

	if output.Output.Spec.Authentication.BasicAuth.UsernameField != "" {
		effectiveUsernameField = output.Output.Spec.Authentication.BasicAuth.UsernameField
	} else {
		effectiveUsernameField = defaultBasicAuthUsernameField
	}

	if output.Output.Spec.Authentication.BasicAuth.PasswordField != "" {
		effectivePasswordField = output.Output.Spec.Authentication.BasicAuth.PasswordField
	} else {
		effectivePasswordField = defaultBasicAuthPasswordField
	}

	config := BasicAuthExtensionConfig{}

	if u, ok := output.Secret.Data[effectiveUsernameField]; ok {
		config.ClientAuth.Username = string(u)
	}

	if p, ok := output.Secret.Data[effectivePasswordField]; ok {
		config.ClientAuth.Password = string(p)
	}

	return config
}

type BearerTokenAuthExtensionConfig struct {
	BearerToken string `json:"token,omitempty"`
}

func generateBearerAuthExtensionsForOutput(output OutputWithSecretData) BearerTokenAuthExtensionConfig {
	var effectiveTokenField string

	if output.Output.Spec.Authentication.BearerAuth.TokenField != "" {
		effectiveTokenField = output.Output.Spec.Authentication.BasicAuth.UsernameField
	} else {
		effectiveTokenField = defaultBearerAuthTokenField
	}

	config := BearerTokenAuthExtensionConfig{}

	if t, ok := output.Secret.Data[effectiveTokenField]; ok {
		config.BearerToken = string(t)
	}

	return config
}

func (cfgInput *OtelColConfigInput) generateExtensions() map[string]any {
	extensions := make(map[string]any)

	for _, output := range cfgInput.OutputsWithSecretData {
		if output.Output.Spec.Authentication != nil {
			if output.Output.Spec.Authentication.BasicAuth != nil {
				extName := fmt.Sprintf("basicauth/%s_%s", output.Output.Namespace, output.Output.Name)
				extensions[extName] = generateBasicAuthExtensionsForOutput(output)
			}
			if output.Output.Spec.Authentication.BearerAuth != nil {
				extName := fmt.Sprintf("bearertokenauth/%s_%s", output.Output.Namespace, output.Output.Name)
				extensions[extName] = generateBearerAuthExtensionsForOutput(output)
			}
		}
	}

	return extensions
}

func (cfgInput *OtelColConfigInput) generateReceivers() map[string]any {
	receivers := make(map[string]any)

	for tenantName := range cfgInput.TenantSubscriptionMap {
		if tenantIdx := slices.IndexFunc(cfgInput.Tenants, func(t v1alpha1.Tenant) bool {
			return tenantName == t.Name
		}); tenantIdx != -1 {
			k8sReceiverName := fmt.Sprintf("filelog/%s", tenantName)
			namespaces := cfgInput.Tenants[tenantIdx].Status.LogSourceNamespaces
			receivers[k8sReceiverName] = cfgInput.generateDefaultKubernetesReceiver(namespaces)
		}
	}

	return receivers
}

func (cfgInput *OtelColConfigInput) AssembleConfig(ctx context.Context) otelv1beta1.Config {
	exporters := cfgInput.generateExporters(ctx)

	processors := cfgInput.generateProcessors()

	extensions := cfgInput.generateExtensions()

	receivers := cfgInput.generateReceivers()

	connectors := cfgInput.generateConnectors()

	pipelines := cfgInput.generateNamedPipelines()

	telemetry := make(map[string]any)

	telemetry["metrics"] = map[string]string{
		"level": "detailed",
	}

	if cfgInput.Debug {
		telemetry["logs"] = map[string]string{
			"level": "debug",
		}
	}

	if _, ok := processors["memory_limiter"]; ok {
		for name, pipeline := range pipelines {
			// From memorylimiterprocessor's README:
			// > For the memory_limiter processor, the best practice is to add it as the first processor in a pipeline.
			memProcessors := []string{"memory_limiter"}
			memProcessors = append(memProcessors, pipeline.Processors...)
			pipeline.Processors = memProcessors
			pipelines[name] = pipeline
		}
	}

	extensionNames := make([]string, 0, len(extensions))
	for k := range extensions {
		extensionNames = append(extensionNames, k)
	}

	return otelv1beta1.Config{
		Receivers:  otelv1beta1.AnyConfig{Object: receivers},
		Exporters:  otelv1beta1.AnyConfig{Object: exporters},
		Processors: &otelv1beta1.AnyConfig{Object: processors},
		Connectors: &otelv1beta1.AnyConfig{Object: connectors},
		Extensions: &otelv1beta1.AnyConfig{Object: extensions},
		Service: otelv1beta1.Service{
			Extensions: &extensionNames,
			Telemetry:  &otelv1beta1.AnyConfig{Object: telemetry},
			Pipelines:  pipelines,
		},
	}
}
