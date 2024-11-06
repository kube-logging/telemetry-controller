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
	"testing"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateRoutingConnectorForSubscriptionsOutputs(t *testing.T) {
	tests := []struct {
		name            string
		subscriptionRef v1alpha1.NamespacedName
		outputNames     []v1alpha1.NamespacedName
		expectedRC      RoutingConnector
	}{
		{
			name: "Single output",
			subscriptionRef: v1alpha1.NamespacedName{
				Namespace: "ns1",
				Name:      "sub1",
			},
			outputNames: []v1alpha1.NamespacedName{
				{Namespace: "ns1", Name: "out1"},
			},
			expectedRC: RoutingConnector{
				Name: "routing/subscription_ns1_sub1_outputs",
				Table: []RoutingConnectorTableItem{
					{
						Condition: "true",
						Pipelines: []string{"logs/output_ns1_sub1_ns1_out1"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedRC, GenerateRoutingConnectorForSubscriptionsOutputs(ttp.subscriptionRef, ttp.outputNames))
		})
	}
}

func TestGenerateRoutingConnectorForBridge(t *testing.T) {
	tests := []struct {
		name       string
		bridge     v1alpha1.Bridge
		expectedRC RoutingConnector
	}{
		{
			name: "Valid bridge",
			bridge: v1alpha1.Bridge{
				Spec: v1alpha1.BridgeSpec{
					Condition:    `attributes["X-Tenant"] == "telemetry-controller-system"`,
					TargetTenant: "tenant1",
				},
			},
			expectedRC: RoutingConnector{
				Name: "routing/bridge_",
				Table: []RoutingConnectorTableItem{
					{
						Condition: `attributes["X-Tenant"] == "telemetry-controller-system"`,
						Pipelines: []string{"logs/tenant_tenant1"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedRC, GenerateRoutingConnectorForBridge(ttp.bridge))
		})
	}
}

func TestGenerateRoutingConnectorForBridgesTenantPipeline(t *testing.T) {
	tests := []struct {
		name             string
		tenantName       string
		pipeline         *otelv1beta1.Pipeline
		bridges          []v1alpha1.Bridge
		expectedPipeline *otelv1beta1.Pipeline
	}{
		{
			name:       "Empty bridges list",
			tenantName: "tenant1",
			pipeline: &otelv1beta1.Pipeline{
				Receivers: []string{"already-existing-receiver"},
				Exporters: []string{"already-existing-exporter"},
			},
			bridges: []v1alpha1.Bridge{},
			expectedPipeline: &otelv1beta1.Pipeline{
				Receivers: []string{"already-existing-receiver"},
				Exporters: []string{"already-existing-exporter"},
			},
		},
		{
			name:       "Bridge requiring exporter only",
			tenantName: "tenant",
			pipeline: &otelv1beta1.Pipeline{
				Receivers: []string{"already-existing-receiver"},
				Exporters: []string{"already-existing-exporter"},
			},
			bridges: []v1alpha1.Bridge{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bridge1",
					},
					Spec: v1alpha1.BridgeSpec{
						SourceTenant: "tenant",
					},
				},
			},
			expectedPipeline: &otelv1beta1.Pipeline{
				Receivers: []string{"already-existing-receiver"},
				Exporters: []string{"already-existing-exporter", "routing/bridge_bridge1"},
			},
		},
		{
			name:       "Bridge requiring receiver only",
			tenantName: "tenant",
			pipeline: &otelv1beta1.Pipeline{
				Receivers: []string{"already-existing-receiver"},
				Exporters: []string{"already-existing-exporter"},
			},
			bridges: []v1alpha1.Bridge{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "bridge2",
					},
					Spec: v1alpha1.BridgeSpec{
						TargetTenant: "tenant",
					},
				},
			},
			expectedPipeline: &otelv1beta1.Pipeline{
				Receivers: []string{"already-existing-receiver", "routing/bridge_bridge2"},
				Exporters: []string{"already-existing-exporter"},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(tt.name, func(t *testing.T) {
			GenerateRoutingConnectorForBridgesTenantPipeline(ttp.tenantName, ttp.pipeline, ttp.bridges)
			assert.Equal(t, ttp.expectedPipeline, ttp.pipeline)
		})
	}
}
