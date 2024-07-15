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
	_ "embed"
	"encoding/json"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/google/go-cmp/cmp"
	"github.com/siliconbrain/go-mapseqs/mapseqs"
	"github.com/siliconbrain/go-seqs/seqs"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

//go:embed otel_col_conf_test_fixtures/complex.json
var otelColTargetJSON string

func TestOtelColConfComplex(t *testing.T) {
	// Required inputs
	var subscriptions = map[v1alpha1.NamespacedName]v1alpha1.Subscription{
		{Name: "subscription-example-1", Namespace: "example-tenant-a-ns"}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-1",
				Namespace: "example-tenant-a-ns",
			},
			Spec: v1alpha1.SubscriptionSpec{
				OTTL: "route()",
				Outputs: []v1alpha1.NamespacedName{
					{
						Name:      "otlp-test-output",
						Namespace: "collector",
					},
					{
						Name:      "loki-test-output",
						Namespace: "collector",
					},
				},
			},
		},
		{Name: "subscription-example-2", Namespace: "example-tenant-a-ns"}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-2",
				Namespace: "example-tenant-a-ns",
			},
			Spec: v1alpha1.SubscriptionSpec{
				OTTL: "route()",
				Outputs: []v1alpha1.NamespacedName{
					{
						Name:      "otlp-test-output-2",
						Namespace: "collector",
					},
				},
			},
		},
		{Name: "subscription-example-3", Namespace: "example-tenant-b-ns"}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-3",
				Namespace: "example-tenant-b-ns",
			},
			Spec: v1alpha1.SubscriptionSpec{
				OTTL: "route()",
				Outputs: []v1alpha1.NamespacedName{
					{
						Name:      "otlp-test-output-2",
						Namespace: "collector",
					},
					{
						Name:      "fluentforward-test-output",
						Namespace: "collector",
					},
				},
			},
		},
	}
	inputCfg := OtelColConfigInput{
		Subscriptions: subscriptions,
		Tenants: []v1alpha1.Tenant{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-tenant-a",
				},
				Spec: v1alpha1.TenantSpec{
					SubscriptionNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant-a",
							},
						},
					},
					LogSourceNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant-a",
							},
						},
					},
				},
				Status: v1alpha1.TenantStatus{
					LogSourceNamespaces: []string{
						"example-tenant-a",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-tenant-b",
				},
				Spec: v1alpha1.TenantSpec{
					SubscriptionNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant-b",
							},
						},
					},
					LogSourceNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant-b",
							},
						},
					},
				},
				Status: v1alpha1.TenantStatus{
					LogSourceNamespaces: []string{
						"example-tenant-b",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-tenant-a",
				},
				Spec: v1alpha1.TenantSpec{
					SubscriptionNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant-a",
							},
						},
					},
					LogSourceNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant-a",
							},
						},
					},
				},
			},
		},
		Outputs: []v1alpha1.OtelOutput{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "otlp-test-output",
					Namespace: "collector",
				},
				Spec: v1alpha1.OtelOutputSpec{
					OTLP: &v1alpha1.OTLP{
						GRPCClientConfig: v1alpha1.GRPCClientConfig{
							Endpoint: "receiver-collector.example-tenant-a-ns.svc.cluster.local:4317",
							TLSSetting: v1alpha1.TLSClientSetting{
								Insecure: true,
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "otlp-test-output-2",
					Namespace: "collector",
				},
				Spec: v1alpha1.OtelOutputSpec{
					OTLP: &v1alpha1.OTLP{
						GRPCClientConfig: v1alpha1.GRPCClientConfig{
							Endpoint: "receiver-collector.example-tenant-a-ns.svc.cluster.local:4317",
							TLSSetting: v1alpha1.TLSClientSetting{
								Insecure: true,
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "loki-test-output",
					Namespace: "collector",
				},
				Spec: v1alpha1.OtelOutputSpec{
					Loki: &v1alpha1.Loki{
						HTTPClientConfig: v1alpha1.HTTPClientConfig{
							Endpoint: "loki.example-tenant-a-ns.svc.cluster.local:4317",
							TLSSetting: v1alpha1.TLSClientSetting{
								Insecure: true,
							},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fluentforward-test-output",
					Namespace: "collector",
				},
				Spec: v1alpha1.OtelOutputSpec{
					Fluentforward: &v1alpha1.Fluentforward{
						TCPClientSettings: v1alpha1.TCPClientSettings{
							Endpoint: "fluentforward.example-tenant-ns.svc.cluster.local:8888",
							TLSSetting: v1alpha1.TLSClientSetting{
								Insecure: true,
							},
						},
					},
				},
			},
		},
		MemoryLimiter: v1alpha1.MemoryLimiter{
			CheckInterval:         1 * time.Second,
			MemoryLimitPercentage: 75,
			MemorySpikePercentage: 25,
		},
	}

	// TODO extract this logic

	subscriptionOutputMap := map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{}
	for _, subscription := range inputCfg.Subscriptions {
		subscriptionOutputMap[subscription.NamespacedName()] = subscription.Spec.Outputs
	}
	inputCfg.SubscriptionOutputMap = subscriptionOutputMap

	inputCfg.TenantSubscriptionMap = map[string][]v1alpha1.NamespacedName{}
	tenantA := inputCfg.Tenants[0]
	tenantASubscriptions := make(map[v1alpha1.NamespacedName]v1alpha1.Subscription)
	for subName, sub := range inputCfg.Subscriptions {
		if subName.Namespace != "example-tenant-b-ns" {
			tenantASubscriptions[subName] = sub
		}
	}
	inputCfg.TenantSubscriptionMap[tenantA.Name] = seqs.ToSlice(mapseqs.KeysOf(tenantASubscriptions))

	tenantB := inputCfg.Tenants[1]
	inputCfg.TenantSubscriptionMap[tenantB.Name] = []v1alpha1.NamespacedName{
		{
			Name:      "subscription-example-3",
			Namespace: "example-tenant-b-ns",
		},
	}

	// Config
	// Hack is needed here, to actually be able to compare the expected and generated config, because of underlying JSON/YAML tagged structs

	var expectedMap map[string]any
	if err := json.Unmarshal([]byte(otelColTargetJSON), &expectedMap); err != nil {
		t.Fatalf("error: %v", err)
	}

	generatedConfig := inputCfg.AssembleConfig(ctx)

	var generatedMap map[string]any
	if err := mapstructure.Decode(generatedConfig, &generatedMap); err != nil {
		t.Fatalf("error: %v", err)
	}
	generatedJSON, _ := json.MarshalIndent(generatedMap, "", "  ")

	var unmarshaledMap map[string]any
	if err := json.Unmarshal(generatedJSON, &unmarshaledMap); err != nil {
		t.Fatalf("error: %v", err)
	}
	if diff := cmp.Diff(expectedMap, unmarshaledMap); diff != "" {
		t.Errorf("mismatch:\n---%s\n---\n", diff)
	}
}
func TestOtelColConfigInput_generateRoutingConnectorForTenantsSubscription(t *testing.T) {
	type fields struct {
		Tenants               []v1alpha1.Tenant
		Subscriptions         map[v1alpha1.NamespacedName]v1alpha1.Subscription
		Outputs               []v1alpha1.OtelOutput
		TenantSubscriptionMap map[string][]v1alpha1.NamespacedName
		SubscriptionOutputMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
	}
	type args struct {
		tenantName        string
		subscriptionNames []v1alpha1.NamespacedName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   RoutingConnector
	}{
		{
			name: "two_subscriptions",
			fields: fields{
				Tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tenantA",
						},
						Spec: v1alpha1.TenantSpec{},
						Status: v1alpha1.TenantStatus{
							LogSourceNamespaces: []string{"a", "b"},
						},
					}},
				Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
					{
						Name:      "subsA",
						Namespace: "nsA",
					}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "subsA",
							Namespace: "nsA",
						},
						Spec: v1alpha1.SubscriptionSpec{Outputs: []v1alpha1.NamespacedName{
							{
								Namespace: "xy",
								Name:      "zq",
							},
						}, OTTL: `set(attributes["subscription"], "subscriptionA")`},
					},
					{
						Name:      "subsB",
						Namespace: "nsA",
					}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "subsB",
							Namespace: "nsA",
						},
						Spec: v1alpha1.SubscriptionSpec{Outputs: []v1alpha1.NamespacedName{
							{
								Namespace: "xy",
								Name:      "zq",
							},
						}, OTTL: `set(attributes["subscription"], "subscriptionB")`},
					},
				},
				Outputs: []v1alpha1.OtelOutput{},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					"tenantA": {
						{
							Namespace: "nsA",
							Name:      "subsA",
						},
						{
							Namespace: "nsA",
							Name:      "subsB",
						},
					},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{},
			},
			args: args{
				tenantName: "tenantA",
				subscriptionNames: []v1alpha1.NamespacedName{
					{
						Namespace: "nsA",
						Name:      "subsA",
					},
					{
						Namespace: "nsA",
						Name:      "subsB",
					},
				},
			},
			want: RoutingConnector{
				Name: "routing/tenant_tenantA_subscriptions",
				Table: []RoutingConnectorTableItem{
					{
						Statement: `set(attributes["subscription"], "subscriptionA")`,
						Pipelines: []string{"logs/tenant_tenantA_subscription_nsA_subsA"},
					},
					{
						Statement: `set(attributes["subscription"], "subscriptionB") `,
						Pipelines: []string{"logs/tenant_tenantA_subscription_nsA_subsB"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgInput := &OtelColConfigInput{
				Tenants:               tt.fields.Tenants,
				Subscriptions:         tt.fields.Subscriptions,
				Outputs:               tt.fields.Outputs,
				TenantSubscriptionMap: tt.fields.TenantSubscriptionMap,
				SubscriptionOutputMap: tt.fields.SubscriptionOutputMap,
			}
			got := cfgInput.generateRoutingConnectorForTenantsSubscriptions(tt.args.tenantName, tt.args.subscriptionNames)
			assert.Equal(t, got, tt.want)
		})
	}
}
