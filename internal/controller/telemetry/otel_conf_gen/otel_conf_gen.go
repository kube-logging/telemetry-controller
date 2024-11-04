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
	"fmt"
	"slices"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"golang.org/x/exp/maps"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/connector"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/exporter"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/extension"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/processor"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/receiver"
)

type OtelColConfigInput struct {
	// These must only include resources that are selected by the collector, tenant labelselectors, and listed outputs in the subscriptions
	Tenants               []v1alpha1.Tenant
	Subscriptions         map[v1alpha1.NamespacedName]v1alpha1.Subscription
	Bridges               []v1alpha1.Bridge
	OutputsWithSecretData []components.OutputWithSecretData
	MemoryLimiter         v1alpha1.MemoryLimiter

	// Subscriptions map, where the key is the Tenants' name, value is a slice of subscriptions' namespaced name
	TenantSubscriptionMap map[string][]v1alpha1.NamespacedName
	SubscriptionOutputMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
	Debug                 bool
}

func (cfgInput *OtelColConfigInput) generateExporters(ctx context.Context) map[string]any {
	exporters := map[string]any{}
	maps.Copy(exporters, exporter.GenerateMetricsExporters())
	maps.Copy(exporters, exporter.GenerateOTLPGRPCExporters(ctx, cfgInput.OutputsWithSecretData))
	maps.Copy(exporters, exporter.GenerateOTLPHTTPExporters(ctx, cfgInput.OutputsWithSecretData))
	maps.Copy(exporters, exporter.GenerateFluentforwardExporters(ctx, cfgInput.OutputsWithSecretData))
	maps.Copy(exporters, exporter.GenerateDebugExporters())

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
	}

	return processors
}

func (cfgInput *OtelColConfigInput) generateExtensions() map[string]any {
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
			receivers[k8sReceiverName] = receiver.GenerateDefaultKubernetesReceiver(namespaces)
		}
	}

	return receivers
}

func (cfgInput *OtelColConfigInput) generateConnectors() map[string]any {
	connectors := make(map[string]any)
	maps.Copy(connectors, connector.GenerateCountConnectors())

	for _, tenant := range cfgInput.Tenants {
		rc := connector.GenerateRoutingConnectorForTenantsSubscriptions(tenant.Name, cfgInput.TenantSubscriptionMap[tenant.Name], cfgInput.Subscriptions)
		connectors[rc.Name] = rc
	}

	for _, subscription := range cfgInput.Subscriptions {
		rc := connector.GenerateRoutingConnectorForSubscriptionsOutputs(subscription.NamespacedName(), cfgInput.SubscriptionOutputMap[subscription.NamespacedName()])
		connectors[rc.Name] = rc
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
		namedPipelines[fmt.Sprintf("logs/tenant_%s", tenant)] = pipeline.GenerateRootPipeline(tenant)
		tenants = append(tenants, tenant)
	}

	maps.Copy(namedPipelines, pipeline.GenerateMetricsPipelines())

	for _, tenant := range tenants {
		// Generate a pipeline for the tenant
		tenantRootPipeline := fmt.Sprintf("logs/tenant_%s", tenant)
		namedPipelines[tenantRootPipeline] = pipeline.GenerateRootPipeline(tenant)

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
