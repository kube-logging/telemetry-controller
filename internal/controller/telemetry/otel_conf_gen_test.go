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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed otel_col_conf_test_fixtures/complex.yaml
var otelColTargetYaml string

func TestOtelColConfComplex(t *testing.T) {
	// Required inputs
	var subscriptions = []v1alpha1.Subscription{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-1",
				Namespace: "example-tenant-ns",
			},
			Spec: v1alpha1.SubscriptionSpec{
				OTTL: "route()",
				Outputs: []v1alpha1.NamespacedName{
					{
						Name:      "otlp-test-output",
						Namespace: "collector",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-2",
				Namespace: "example-tenant-ns",
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
	}
	inputCfg := OtelColConfigInput{
		Subscriptions: subscriptions,
		Tenants: []v1alpha1.Tenant{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-tenant",
				},
				Spec: v1alpha1.TenantSpec{
					SubscriptionNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant",
							},
						},
					},
					LogSourceNamespaceSelectors: []metav1.LabelSelector{
						{
							MatchLabels: map[string]string{
								"nsSelector": "example-tenant",
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
					OTLP: v1alpha1.OTLPgrpc{
						ClientConfig: v1alpha1.ClientConfig{
							Endpoint: "receiver-collector.example-tenant-ns.svc.cluster.local:4317",
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
					OTLP: v1alpha1.OTLPgrpc{
						ClientConfig: v1alpha1.ClientConfig{
							Endpoint: "receiver-collector.example-tenant-ns.svc.cluster.local:4317",
							TLSSetting: v1alpha1.TLSClientSetting{
								Insecure: true,
							},
						},
					},
				},
			},
		},
	}

	// TODO extract this logic

	subscriptionOutputMap := map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{}
	for _, subscription := range inputCfg.Subscriptions {
		subscriptionOutputMap[subscription.NamespacedName()] = subscription.Spec.Outputs
	}
	inputCfg.SubscriptionOutputMap = subscriptionOutputMap

	inputCfg.TenantSubscriptionMap = map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{}
	tenant := inputCfg.Tenants[0]
	inputCfg.TenantSubscriptionMap[tenant.NamespacedName()] = getSubscriptionNamesFromSubscription(inputCfg.Subscriptions)

	// IR
	generatedIR := inputCfg.ToIntermediateRepresentation()

	// Final YAML
	_, err := generatedIR.ToYAML()
	if err != nil {
		t.Fatalf("YAML formatting failed, err=%v", err)
	}

	actualYAMLBytes, err := generatedIR.ToYAMLRepresentation()
	if err != nil {
		t.Fatalf("error %v", err)
	}
	actualYAML, err := generatedIR.ToYAML()
	if err != nil {
		t.Fatalf("error %v", err)
	}

	var actualUniversalMap map[string]any
	if err := yaml.Unmarshal(actualYAMLBytes, &actualUniversalMap); err != nil {
		t.Fatalf("error: %v", err)
	}

	var expectedUniversalMap map[string]any
	if err := yaml.Unmarshal([]byte(otelColTargetYaml), &expectedUniversalMap); err != nil {
		t.Fatalf("error: %v", err)
	}

	// use dyff for YAML comparison
	if diff := cmp.Diff(expectedUniversalMap, actualUniversalMap); diff != "" {
		t.Logf("mismatch:\n---%s\n---\n", diff)
	}

	if !reflect.DeepEqual(actualUniversalMap, expectedUniversalMap) {
		t.Logf(`yaml mismatch:
expected=
---
%s
---
actual=
---
%s
---`, otelColTargetYaml, actualYAML)
		t.Fatalf(`yaml marshaling failed
expected=
---
%v
---,
actual=
---
%v
---`,
			expectedUniversalMap, actualUniversalMap)
	}
}

func Test_generateRootRoutingConnector(t *testing.T) {
	type args struct {
		tenants []v1alpha1.Tenant
	}
	tests := []struct {
		name string
		args args
		want RoutingConnector
	}{
		{
			name: "two_tenants",
			args: args{
				tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenantA",
							Namespace: "nsA",
						},
						Status: v1alpha1.TenantStatus{
							LogSourceNamespaces: []string{"a", "b"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "tenantB",
							Namespace: "nsB",
						},
						Status: v1alpha1.TenantStatus{
							LogSourceNamespaces: []string{"c", "d"},
						},
					},
				},
			},
			want: RoutingConnector{
				Name:             "routing/tenants",
				DefaultPipelines: []string{},
				Table: []RoutingConnectorTableItem{
					{
						Statement: `route() where IsMatch(attributes["k8s.namespace.name"], "a") or IsMatch(attributes["k8s.namespace.name"], "b")`,
						Pipelines: []string{"logs/tenant_tenantA"},
					},
					{
						Statement: `route() where IsMatch(attributes["k8s.namespace.name"], "c") or IsMatch(attributes["k8s.namespace.name"], "d")`,
						Pipelines: []string{"logs/tenant_tenantB"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateRootRoutingConnector(tt.args.tenants)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestOtelColConfigInput_generateRoutingConnectorForTenantsSubscription(t *testing.T) {
	type fields struct {
		Tenants               []v1alpha1.Tenant
		Subscriptions         []v1alpha1.Subscription
		Outputs               []v1alpha1.OtelOutput
		TenantSubscriptionMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
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
				Tenants: []v1alpha1.Tenant{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenantA",
						Namespace: "nsA",
					},
					Spec: v1alpha1.TenantSpec{},
					Status: v1alpha1.TenantStatus{
						LogSourceNamespaces: []string{"a", "b"},
					},
				}},
				Subscriptions: []v1alpha1.Subscription{
					{
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
				TenantSubscriptionMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{
						Namespace: "nsA",
						Name:      "tenantA",
					}: {
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
				Name:             "routing/tenant_tenantA_subscriptions",
				DefaultPipelines: []string{},
				Table: []RoutingConnectorTableItem{
					{
						Statement: `set(attributes["subscription"], "subscriptionA")`,
						Pipelines: []string{"logs/tenant_tenantA_subscription_subsA"},
					},
					{
						Statement: `set(attributes["subscription"], "subscriptionB") `,
						Pipelines: []string{"logs/tenant_tenantA_subscription_subsB"},
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
			got := cfgInput.generateRoutingConnectorForTenantsSubscription(tt.args.tenantName, tt.args.subscriptionNames)
			assert.Equal(t, got, tt.want)
		})
	}
}
