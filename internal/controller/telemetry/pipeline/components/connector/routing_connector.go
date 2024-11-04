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
	"strings"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/utils"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

type RoutingConnectorTableItem struct {
	Statement string   `json:"statement"`
	Pipelines []string `json:"pipelines"`
}

type RoutingConnector struct {
	Name             string                      `json:"-"`
	DefaultPipelines []string                    `json:"default_pipelines,omitempty"`
	ErrorMode        components.ErrorMode        `json:"error_mode,omitempty"`
	MatchOnce        bool                        `json:"match_once,omitempty"`
	Table            []RoutingConnectorTableItem `json:"table"`
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
	newItem := RoutingConnectorTableItem{
		Statement: fmt.Sprintf("%s%s", subscription.Spec.OTTL, strings.Repeat(" ", index)),
		Pipelines: []string{pipelineName},
	}

	return newItem
}

func GenerateRoutingConnectorForTenantsSubscriptions(tenantName string, subscriptionNames []v1alpha1.NamespacedName, subscriptions map[v1alpha1.NamespacedName]v1alpha1.Subscription) RoutingConnector {
	rc := newRoutingConnector(fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName))
	utils.SortNamespacedNames(subscriptionNames)
	for index, subscriptionRef := range subscriptionNames {
		subscription := subscriptions[subscriptionRef]
		tableItem := buildRoutingTableItemForSubscription(tenantName, subscription, index)
		rc.AddRoutingConnectorTableElem(tableItem)
	}

	return rc
}

func GenerateRoutingConnectorForSubscriptionsOutputs(subscriptionRef v1alpha1.NamespacedName, outputNames []v1alpha1.NamespacedName) RoutingConnector {
	rc := newRoutingConnector(fmt.Sprintf("routing/subscription_%s_%s_outputs", subscriptionRef.Namespace, subscriptionRef.Name))
	utils.SortNamespacedNames(outputNames)
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

func GenerateRoutingConnectorForBridge(bridge v1alpha1.Bridge) RoutingConnector {
	rc := newRoutingConnector(fmt.Sprintf("routing/bridge_%s", bridge.Name))

	tableItem := RoutingConnectorTableItem{
		Statement: bridge.Spec.OTTL,
		Pipelines: []string{fmt.Sprintf("logs/tenant_%s", bridge.Spec.TargetTenant)},
	}
	rc.AddRoutingConnectorTableElem(tableItem)

	return rc
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

		if needsReceiver {
			pipeline.Receivers = append(pipeline.Receivers, fmt.Sprintf("routing/bridge_%s", bridgeName))
		}

		if needsExporter {
			pipeline.Exporters = append(pipeline.Exporters, fmt.Sprintf("routing/bridge_%s", bridgeName))
		}
	}
}
