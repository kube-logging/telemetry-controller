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

package pipeline

import (
	"fmt"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components/exporter"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components/processor"
)

func GeneratePipeline(receivers, processors, exporters []string) *otelv1beta1.Pipeline {
	return &otelv1beta1.Pipeline{
		Receivers:  filterEmptyPipelines(receivers),
		Processors: filterEmptyPipelines(processors),
		Exporters:  filterEmptyPipelines(exporters),
	}
}

func GenerateRootPipeline(tenants []v1alpha1.Tenant, tenantName string) *otelv1beta1.Pipeline {
	tenantCountConnectorName := "count/tenant_metrics"
	var receiverName string
	var exporterName string
	for _, tenant := range tenants {
		if tenant.Name == tenantName {
			// Add filelog receiver to tenant's pipeline if it has any logsource namespace selectors
			// or if it selects from all namespaces
			if tenant.Status.LogSourceNamespaces != nil || tenant.Spec.SelectFromAllNamespaces {
				receiverName = fmt.Sprintf("filelog/%s", tenantName)
			}
			// Add routing connector to tenant's pipeline if it has any subscription namespace selectors
			if tenant.Status.Subscriptions != nil {
				exporterName = fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName)
			}
		}
	}

	return GeneratePipeline([]string{receiverName}, []string{"k8sattributes", fmt.Sprintf("attributes/tenant_%s", tenantName), "filter/exclude"}, []string{exporterName, tenantCountConnectorName})
}

func GenerateMetricsPipelines() map[string]*otelv1beta1.Pipeline {
	metricsPipelines := make(map[string]*otelv1beta1.Pipeline)
	metricsPipelines["metrics/tenant"] = &otelv1beta1.Pipeline{
		Receivers:  []string{"count/tenant_metrics"},
		Processors: []string{processor.DefaultDeltaToCumulativeProcessorID, "attributes/metricattributes"},
		Exporters:  []string{exporter.DefaultPrometheusExporterID},
	}
	metricsPipelines["metrics/output"] = &otelv1beta1.Pipeline{
		Receivers:  []string{"count/output_metrics"},
		Processors: []string{processor.DefaultDeltaToCumulativeProcessorID, "attributes/metricattributes"},
		Exporters:  []string{exporter.DefaultPrometheusExporterID},
	}

	metricsPipelines["metrics/output_bytes"] = &otelv1beta1.Pipeline{
		Receivers:  []string{"bytes/exporter"},
		Processors: []string{processor.DefaultDeltaToCumulativeProcessorID, "attributes/metricattributes"},
		Exporters:  []string{exporter.DefaultPrometheusExporterID},
	}

	return metricsPipelines
}

func filterEmptyPipelines(items []string) []string {
	var result []string
	for _, item := range items {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
