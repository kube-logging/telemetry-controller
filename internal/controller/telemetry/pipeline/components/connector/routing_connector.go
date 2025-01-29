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

import (
	"fmt"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

type RoutingConnectorTableItem struct {
	Condition string   `json:"condition,omitempty"`
	Pipelines []string `json:"pipelines,omitempty"`
}

type RoutingConnector struct {
	Name             string                      `json:"-"`
	DefaultPipelines []string                    `json:"default_pipelines,omitempty"`
	ErrorMode        components.ErrorMode        `json:"error_mode,omitempty"`
	Table            []RoutingConnectorTableItem `json:"table"`
}

func (rc *RoutingConnector) populateRoutingConnectorTable(seenConditionsPipelineMap map[string][]string) {
	for condition, pipelines := range seenConditionsPipelineMap {
		tableItem := RoutingConnectorTableItem{
			Condition: condition,
			Pipelines: pipelines,
		}
		rc.Table = append(rc.Table, tableItem)
	}
}

func newRoutingConnector(name string, tenantRouteConfig v1alpha1.RouteConfig) RoutingConnector {
	return RoutingConnector{
		Name:             name,
		DefaultPipelines: tenantRouteConfig.DefaultPipelines,
		ErrorMode:        components.ErrorMode(tenantRouteConfig.ErrorMode),
	}
}

func GenerateRoutingConnectorForTenantsSubscriptions(tenantName string, tenantRouteConfig v1alpha1.RouteConfig, subscriptionNames []v1alpha1.NamespacedName, subscriptions map[v1alpha1.NamespacedName]v1alpha1.Subscription) RoutingConnector {
	rc := newRoutingConnector(fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName), tenantRouteConfig)
	components.SortNamespacedNames(subscriptionNames)
	seenConditionsPipelineMap := make(map[string][]string)
	for _, subscriptionRef := range subscriptionNames {
		subscription, ok := subscriptions[subscriptionRef]
		if ok {
			pipelineName := fmt.Sprintf("logs/tenant_%s_subscription_%s_%s", tenantName, subscription.Namespace, subscription.Name)
			if _, ok := seenConditionsPipelineMap[subscription.Spec.Condition]; !ok {
				seenConditionsPipelineMap[subscription.Spec.Condition] = []string{pipelineName}
			} else {
				seenConditionsPipelineMap[subscription.Spec.Condition] = append(seenConditionsPipelineMap[subscription.Spec.Condition], pipelineName)
			}
		}
	}
	rc.populateRoutingConnectorTable(seenConditionsPipelineMap)

	return rc
}

func GenerateRoutingConnectorForSubscriptionsOutputs(subscriptionRef v1alpha1.NamespacedName, outputNames []v1alpha1.NamespacedName) RoutingConnector {
	rc := newRoutingConnector(fmt.Sprintf("routing/subscription_%s_%s_outputs", subscriptionRef.Namespace, subscriptionRef.Name), v1alpha1.RouteConfig{})
	components.SortNamespacedNames(outputNames)
	pipelines := []string{}
	for _, outputRef := range outputNames {
		pipelines = append(pipelines, fmt.Sprintf("logs/output_%s_%s_%s_%s", subscriptionRef.Namespace, subscriptionRef.Name, outputRef.Namespace, outputRef.Name))
	}

	tableItem := RoutingConnectorTableItem{
		Condition: "true",
		Pipelines: pipelines,
	}
	rc.Table = append(rc.Table, tableItem)

	return rc
}

func GenerateRoutingConnectorForBridge(bridge v1alpha1.Bridge) RoutingConnector {
	rc := newRoutingConnector(fmt.Sprintf("routing/bridge_%s", bridge.Name), v1alpha1.RouteConfig{})
	tableItem := RoutingConnectorTableItem{
		Condition: bridge.Spec.Condition,
		Pipelines: []string{fmt.Sprintf("logs/tenant_%s", bridge.Spec.TargetTenant)},
	}
	rc.Table = append(rc.Table, tableItem)

	return rc
}

func hasPipelineReceiverOrExporter(pipeline *otelv1beta1.Pipeline, receiverName string) bool {
	for _, receiver := range pipeline.Receivers {
		if receiver == receiverName {
			return true
		}
	}

	for _, exporter := range pipeline.Exporters {
		if exporter == receiverName {
			return true
		}
	}

	return false
}

func addConnectorToPipeline(pipeline *otelv1beta1.Pipeline, connectorName string, needsReceiver, needsExporter bool) {
	if !hasPipelineReceiverOrExporter(pipeline, connectorName) {
		if needsReceiver {
			pipeline.Receivers = append(pipeline.Receivers, connectorName)
		}
		if needsExporter {
			pipeline.Exporters = append(pipeline.Exporters, connectorName)
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

func GenerateRoutingConnectorForBridgesTenantPipeline(tenantName string, pipeline *otelv1beta1.Pipeline, bridges []v1alpha1.Bridge) {
	for _, bridge := range bridges {
		needsReceiver, needsExporter, bridgeName := checkBridgeConnectorForTenant(tenantName, bridge)
		connectorName := fmt.Sprintf("routing/bridge_%s", bridgeName)
		addConnectorToPipeline(pipeline, connectorName, needsReceiver, needsExporter)
	}
}
