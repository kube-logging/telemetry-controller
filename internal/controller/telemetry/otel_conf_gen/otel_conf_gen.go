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

package otel_conf_gen

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/connector"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/exporter"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/extension"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/extension/storage"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/processor"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/receiver"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"golang.org/x/exp/maps"
)

var ErrNoResources = errors.New("there are no resources deployed that the collector(s) can use")

type OtelColConfigInput struct {
	components.ResourceRelations
	MemoryLimiter v1alpha1.MemoryLimiter
	Debug         bool
}

func (cfgInput *OtelColConfigInput) IsEmpty() bool {
	if cfgInput == nil {
		return true
	}

	v := reflect.ValueOf(*cfgInput)
	for i := range v.NumField() {
		field := v.Field(i)
		if v.Field(i).Kind() == reflect.Struct {
			if !reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()) {
				return false
			}
		}
	}

	return true
}

func (cfgInput *OtelColConfigInput) generateExporters(ctx context.Context) map[string]any {
	exporters := map[string]any{}
	maps.Copy(exporters, exporter.GenerateMetricsExporters())
	maps.Copy(exporters, exporter.GenerateOTLPGRPCExporters(ctx, cfgInput.ResourceRelations))
	maps.Copy(exporters, exporter.GenerateOTLPHTTPExporters(ctx, cfgInput.ResourceRelations))
	maps.Copy(exporters, exporter.GenerateFluentforwardExporters(ctx, cfgInput.ResourceRelations))
	if cfgInput.Debug {
		maps.Copy(exporters, exporter.GenerateDebugExporters())
	}

	return exporters
}

func (cfgInput *OtelColConfigInput) generateProcessors() map[string]any {
	processors := make(map[string]any)
	processors["k8sattributes"] = processor.GenerateDefaultKubernetesProcessor()
	processors["memory_limiter"] = processor.GenerateProcessorMemoryLimiter(cfgInput.MemoryLimiter)
	maps.Copy(processors, processor.GenerateMetricsProcessors())

	for _, tenant := range cfgInput.Tenants {
		processors[fmt.Sprintf("attributes/tenant_%s", tenant.Name)] = processor.GenerateTenantAttributeProcessor(tenant.Name)

		// Add a transform processor if the tenant has one
		if tenant.Spec.Transform.Name != "" {
			processors[fmt.Sprintf("transform/%s", tenant.Spec.Transform.Name)] = processor.GenerateTransformProcessorForTenant(tenant)
		}
	}

	for _, subscription := range cfgInput.Subscriptions {
		processors[fmt.Sprintf("attributes/subscription_%s", subscription.Name)] = processor.GenerateSubscriptionAttributeProcessor(subscription.Name)
	}

	for _, output := range cfgInput.OutputsWithSecretData {
		processors[fmt.Sprintf("attributes/exporter_name_%s", output.Output.Name)] = processor.GenerateOutputExporterNameProcessor(components.GetExporterNameForOutput(output.Output))

		// Add a batch processor if the output has one
		if output.Output.Spec.Batch != nil {
			processors[fmt.Sprintf("batch/%s", output.Output.Name)] = processor.GenerateBatchProcessorForOutput(*output.Output.Spec.Batch)
		}
	}

	return processors
}

func (cfgInput *OtelColConfigInput) generateExtensions() (map[string]any, []string) {
	extensions := make(map[string]any)
	for _, output := range cfgInput.OutputsWithSecretData {
		if output.Output.Spec.Authentication != nil {
			if output.Output.Spec.Authentication.BasicAuth != nil {
				extName := fmt.Sprintf("basicauth/%s_%s", output.Output.Namespace, output.Output.Name)
				extensions[extName] = extension.GenerateBasicAuthExtensionsForOutput(output)
			}
			if output.Output.Spec.Authentication.BearerAuth != nil {
				extName := fmt.Sprintf("bearertokenauth/%s_%s", output.Output.Namespace, output.Output.Name)
				extensions[extName] = extension.GenerateBearerAuthExtensionsForOutput(output)
			}
		}
	}

	for _, tenant := range cfgInput.Tenants {
		if tenant.Spec.PersistenceConfig.EnableFileStorage {
			extensions[fmt.Sprintf("file_storage/%s", tenant.Name)] = storage.GenerateFileStorageExtensionForTenant(tenant.Spec.PersistenceConfig.Directory, tenant.Name)
		}
	}

	var extensionNames []string
	if len(extensions) > 0 {
		extensionNames = make([]string, 0, len(extensions))
		for k := range extensions {
			extensionNames = append(extensionNames, k)
		}
	} else {
		extensionNames = nil
	}
	sort.Strings(extensionNames)

	return extensions, extensionNames
}

func (cfgInput *OtelColConfigInput) generateReceivers() map[string]any {
	receivers := make(map[string]any)
	for tenantName := range cfgInput.TenantSubscriptionMap {
		if tenantIdx := slices.IndexFunc(cfgInput.Tenants, func(t v1alpha1.Tenant) bool {
			return tenantName == t.Name
		}); tenantIdx != -1 {
			namespaces := cfgInput.Tenants[tenantIdx].Status.LogSourceNamespaces
			if len(namespaces) > 0 || cfgInput.Tenants[tenantIdx].Spec.SelectFromAllNamespaces {
				receivers[fmt.Sprintf("filelog/%s", tenantName)] = receiver.GenerateDefaultKubernetesReceiver(namespaces, cfgInput.Tenants[tenantIdx])
			}
		}
	}

	return receivers
}

func (cfgInput *OtelColConfigInput) generateConnectors() map[string]any {
	connectors := make(map[string]any)
	maps.Copy(connectors, connector.GenerateCountConnectors())

	for _, tenant := range cfgInput.Tenants {
		// Generate routing connector for the tenant's subscription if it has any
		if len(cfgInput.TenantSubscriptionMap[tenant.Name]) > 0 {
			rc := connector.GenerateRoutingConnectorForTenantsSubscriptions(tenant.Name, tenant.Spec.RouteConfig, cfgInput.TenantSubscriptionMap[tenant.Name], cfgInput.Subscriptions)
			connectors[rc.Name] = rc
		}
	}

	for _, subscription := range cfgInput.Subscriptions {
		// Generate routing connector for the subscription's outputs if it has any
		if len(cfgInput.SubscriptionOutputMap[subscription.NamespacedName()]) > 0 {
			rc := connector.GenerateRoutingConnectorForSubscriptionsOutputs(subscription.NamespacedName(), cfgInput.SubscriptionOutputMap[subscription.NamespacedName()])
			connectors[rc.Name] = rc
		}
	}

	for _, bridge := range cfgInput.Bridges {
		rc := connector.GenerateRoutingConnectorForBridge(bridge)
		connectors[rc.Name] = rc
	}

	return connectors
}

func (cfgInput *OtelColConfigInput) generateNamedPipelines() map[string]*otelv1beta1.Pipeline {
	const outputCountConnectorName = "count/output_metrics"

	var namedPipelines = make(map[string]*otelv1beta1.Pipeline)
	tenants := []string{}
	for tenant := range cfgInput.TenantSubscriptionMap {
		namedPipelines[fmt.Sprintf("logs/tenant_%s", tenant)] = pipeline.GenerateRootPipeline(cfgInput.Tenants, tenant)
		tenants = append(tenants, tenant)
	}

	maps.Copy(namedPipelines, pipeline.GenerateMetricsPipelines())

	for _, tenant := range tenants {
		// Generate a pipeline for the tenant
		tenantRootPipeline := fmt.Sprintf("logs/tenant_%s", tenant)
		namedPipelines[tenantRootPipeline] = pipeline.GenerateRootPipeline(cfgInput.Tenants, tenant)

		connector.GenerateRoutingConnectorForBridgesTenantPipeline(tenant, namedPipelines[tenantRootPipeline], cfgInput.Bridges)
		processor.GenerateTransformProcessorForTenantPipeline(tenant, namedPipelines[tenantRootPipeline], cfgInput.Tenants)

		// Generate pipelines for the subscriptions for the tenant
		for _, subscription := range cfgInput.TenantSubscriptionMap[tenant] {
			tenantSubscriptionPipelineName := fmt.Sprintf("%s_subscription_%s_%s", tenantRootPipeline, subscription.Namespace, subscription.Name)
			namedPipelines[tenantSubscriptionPipelineName] = pipeline.GeneratePipeline([]string{fmt.Sprintf("routing/tenant_%s_subscriptions", tenant)}, []string{fmt.Sprintf("attributes/subscription_%s", subscription.Name)}, []string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)})

			for _, outputRef := range cfgInput.SubscriptionOutputMap[subscription] {
				outputPipelineName := fmt.Sprintf("logs/output_%s_%s_%s_%s", subscription.Namespace, subscription.Name, outputRef.Namespace, outputRef.Name)

				idx := slices.IndexFunc(cfgInput.OutputsWithSecretData, func(elem components.OutputWithSecretData) bool {
					return outputRef == elem.Output.NamespacedName()
				})
				if idx != -1 {
					output := cfgInput.OutputsWithSecretData[idx]

					receivers := []string{fmt.Sprintf("routing/subscription_%s_%s_outputs", subscription.Namespace, subscription.Name)}
					processors := []string{fmt.Sprintf("attributes/exporter_name_%s", output.Output.Name)}

					// NOTE: The order of the processors is important.
					// The batch processor should be defined in the pipeline after the memory_limiter as well as any sampling processors.
					// This is because batching should happen after any data drops such as sampling.
					// ref: https://github.com/open-telemetry/opentelemetry-collector/tree/main/processor#recommended-processors
					if output.Output.Spec.Batch != nil {
						processors = append(processors, fmt.Sprintf("batch/%s", output.Output.Name))
					}

					var exporters []string

					if output.Output.Spec.OTLPGRPC != nil {
						exporters = []string{components.GetExporterNameForOutput(output.Output), outputCountConnectorName}
					}

					if output.Output.Spec.OTLPHTTP != nil {
						exporters = []string{components.GetExporterNameForOutput(output.Output), outputCountConnectorName}
					}

					if output.Output.Spec.Fluentforward != nil {
						exporters = []string{components.GetExporterNameForOutput(output.Output), outputCountConnectorName}
					}
					if cfgInput.Debug {
						exporters = append(exporters, "debug")
					}

					namedPipelines[outputPipelineName] = pipeline.GeneratePipeline(receivers, processors, exporters)
				}
			}
		}

	}

	return namedPipelines
}

func (cfgInput *OtelColConfigInput) generateTelemetry() map[string]any {
	telemetry := map[string]interface{}{
		"metrics": map[string]interface{}{
			"level": "detailed",
			"readers": []map[string]interface{}{
				{
					"pull": map[string]interface{}{
						"exporter": map[string]interface{}{
							"prometheus": map[string]interface{}{
								"host": "",
								"port": 8888,
							},
						},
					},
				},
			},
		},
	}

	if cfgInput.Debug {
		telemetry["logs"] = map[string]string{
			"level": "debug",
		}
	}

	return telemetry
}

func (cfgInput *OtelColConfigInput) AssembleConfig(ctx context.Context) (otelv1beta1.Config, map[string]string) {
	exporters := cfgInput.generateExporters(ctx)

	processors := cfgInput.generateProcessors()

	extensions, extensionNames := cfgInput.generateExtensions()

	receivers := cfgInput.generateReceivers()

	connectors := cfgInput.generateConnectors()

	pipelines := cfgInput.generateNamedPipelines()

	telemetry := cfgInput.generateTelemetry()

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

	otelConfig := otelv1beta1.Config{
		Receivers:  otelv1beta1.AnyConfig{Object: receivers},
		Exporters:  otelv1beta1.AnyConfig{Object: exporters},
		Processors: &otelv1beta1.AnyConfig{Object: processors},
		Connectors: &otelv1beta1.AnyConfig{Object: connectors},
		Extensions: &otelv1beta1.AnyConfig{Object: extensions},
		Service: otelv1beta1.Service{
			Extensions: extensionNames,
			Telemetry:  &otelv1beta1.AnyConfig{Object: telemetry},
			Pipelines:  pipelines,
		},
	}

	return otelConfig, assembleAdditionalArgs(&otelConfig)
}

func assembleAdditionalArgs(otelConfig *otelv1beta1.Config) map[string]string {
	const (
		featureGatesKey             = "feature-gates"
		flattenLogsFeatureGateValue = "transform.flatten.logs"

		transformProcessorID = "transform"
		flattenDataKey       = "flatten_data"
	)

	args := make(map[string]string)
	for processorName, processorConfig := range otelConfig.Processors.Object {
		if strings.Contains(processorName, transformProcessorID) && processorConfig.(processor.TransformProcessor).FlattenData {
			args[featureGatesKey] = flattenLogsFeatureGateValue
			break
		}
	}

	return args
}

func validateTenants(tenants *[]v1alpha1.Tenant) error {
	var result *multierror.Error

	if len(*tenants) == 0 {
		return errors.New("no tenants provided, at least one tenant must be provided")
	}

	for _, tenant := range *tenants {
		if tenant.Spec.LogSourceNamespaceSelectors != nil && tenant.Spec.SelectFromAllNamespaces {
			result = multierror.Append(result, fmt.Errorf("tenant %s has both log source namespace selectors and select from all namespaces enabled", tenant.Name))
		}
	}

	return result.ErrorOrNil()
}

func validateSubscriptionsAndBridges(tenants *[]v1alpha1.Tenant, subscriptions *map[v1alpha1.NamespacedName]v1alpha1.Subscription, bridges *[]v1alpha1.Bridge) error {
	var result *multierror.Error

	hasSubs := len(*subscriptions) > 0
	hasBridges := len(*bridges) > 0
	if !hasSubs && !hasBridges {
		return errors.New("no subscriptions or bridges provided, at least one subscription or bridge must be provided")
	}

	if hasSubs {
		for _, subscription := range *subscriptions {
			if len(subscription.Spec.Outputs) == 0 {
				result = multierror.Append(result, fmt.Errorf("subscription %s has no outputs", subscription.Name))
			}
		}
	}

	if hasBridges {
		tenantMap := make(map[string]struct{})
		for _, tenant := range *tenants {
			tenantMap[tenant.Name] = struct{}{}
		}

		for _, bridge := range *bridges {
			if _, sourceFound := tenantMap[bridge.Spec.SourceTenant]; !sourceFound {
				result = multierror.Append(result, fmt.Errorf("bridge: %s has a source tenant: %s that does not exist", bridge.Name, bridge.Spec.SourceTenant))
			}
			if _, targetFound := tenantMap[bridge.Spec.TargetTenant]; !targetFound {
				result = multierror.Append(result, fmt.Errorf("bridge: %s has a target tenant: %s that does not exist", bridge.Name, bridge.Spec.TargetTenant))
			}
		}
	}

	return result.ErrorOrNil()
}

func (cfgInput *OtelColConfigInput) ValidateConfig() error {
	if cfgInput.IsEmpty() {
		return ErrNoResources
	}

	var result *multierror.Error
	if err := validateTenants(&cfgInput.Tenants); err != nil {
		result = multierror.Append(result, err)
	}
	if err := validateSubscriptionsAndBridges(&cfgInput.Tenants, &cfgInput.Subscriptions, &cfgInput.Bridges); err != nil {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}
