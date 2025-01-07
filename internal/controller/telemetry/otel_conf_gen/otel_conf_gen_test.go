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
	_ "embed"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components/connector"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/utils"
)

//go:embed otel_col_conf_test_fixtures/complex.yaml
var otelColTargetYAML string

func TestOtelColConfComplex(t *testing.T) {
	// Required inputs
	var subscriptions = map[v1alpha1.NamespacedName]v1alpha1.Subscription{
		{Name: "subscription-example-1", Namespace: "example-tenant-a-ns"}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-1",
				Namespace: "example-tenant-a-ns",
			},
			Spec: v1alpha1.SubscriptionSpec{
				Condition: "true",
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
			Status: v1alpha1.SubscriptionStatus{
				Tenant: "example-tenant-a",
			},
		},
		{Name: "subscription-example-2", Namespace: "example-tenant-a-ns"}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-2",
				Namespace: "example-tenant-a-ns",
			},
			Spec: v1alpha1.SubscriptionSpec{
				Condition: "true",
				Outputs: []v1alpha1.NamespacedName{
					{
						Name:      "otlp-test-output-2",
						Namespace: "collector",
					},
				},
			},
			Status: v1alpha1.SubscriptionStatus{
				Tenant: "example-tenant-a",
			},
		},
		{Name: "subscription-example-3", Namespace: "example-tenant-b-ns"}: {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "subscription-example-3",
				Namespace: "example-tenant-b-ns",
			},
			Spec: v1alpha1.SubscriptionSpec{
				Condition: "true",
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
			Status: v1alpha1.SubscriptionStatus{
				Tenant: "example-tenant-b",
			},
		},
	}
	inputCfg := OtelColConfigInput{
		ResourceRelations: components.ResourceRelations{
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
						PersistenceConfig: v1alpha1.PersistenceConfig{
							EnableFileStorage: true,
						},
					},
					Status: v1alpha1.TenantStatus{
						LogSourceNamespaces: []string{
							"example-tenant-a",
						},
						Subscriptions: []v1alpha1.NamespacedName{
							{
								Namespace: "example-tenant-a-ns",
								Name:      "subscription-example-1",
							},
							{
								Namespace: "example-tenant-a-ns",
								Name:      "subscription-example-2",
							},
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
						PersistenceConfig: v1alpha1.PersistenceConfig{
							EnableFileStorage: true,
						},
					},
					Status: v1alpha1.TenantStatus{
						LogSourceNamespaces: []string{
							"example-tenant-b",
						},
						Subscriptions: []v1alpha1.NamespacedName{
							{
								Namespace: "example-tenant-b-ns",
								Name:      "subscription-example-3",
							},
						},
					},
				},
			},
			OutputsWithSecretData: []components.OutputWithSecretData{
				{
					Secret: corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "bearer-test-secret",
							Namespace: "collector",
						},
						Data: map[string][]byte{
							"token": []byte("testtoken"),
						},
						Type: "Opaque",
					},
					Output: v1alpha1.Output{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "otlp-test-output",
							Namespace: "collector",
						},
						Spec: v1alpha1.OutputSpec{
							OTLPGRPC: &v1alpha1.OTLPGRPC{
								GRPCClientConfig: v1alpha1.GRPCClientConfig{
									Endpoint: utils.ToPtr("receiver-collector.example-tenant-a-ns.svc.cluster.local:4317"),
									TLSSetting: &v1alpha1.TLSClientSetting{
										Insecure: true,
									},
								},
							},
							Authentication: &v1alpha1.OutputAuth{
								BearerAuth: &v1alpha1.BearerAuthConfig{
									SecretRef: &corev1.SecretReference{
										Name:      "bearer-test-secret",
										Namespace: "collector",
									},
								},
							},
							Batch: &v1alpha1.Batch{
								Timeout:                  "5s",
								SendBatchSize:            512,
								SendBatchMaxSize:         4096,
								MetadataKeys:             []string{"key1", "key2"},
								MetadataCardinalityLimit: 100,
							},
						},
					},
				},
				{
					Output: v1alpha1.Output{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "otlp-test-output-2",
							Namespace: "collector",
						},
						Spec: v1alpha1.OutputSpec{
							OTLPGRPC: &v1alpha1.OTLPGRPC{
								GRPCClientConfig: v1alpha1.GRPCClientConfig{
									Endpoint: utils.ToPtr("receiver-collector.example-tenant-a-ns.svc.cluster.local:4317"),
									TLSSetting: &v1alpha1.TLSClientSetting{
										Insecure: true,
									},
								},
							},
						},
					},
				},
				{
					Output: v1alpha1.Output{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "loki-test-output",
							Namespace: "collector",
						},
						Spec: v1alpha1.OutputSpec{
							OTLPHTTP: &v1alpha1.OTLPHTTP{
								HTTPClientConfig: v1alpha1.HTTPClientConfig{
									Endpoint: utils.ToPtr("loki.example-tenant-a-ns.svc.cluster.local:4317"),
									TLSSetting: &v1alpha1.TLSClientSetting{
										Insecure: true,
									},
								},
							},
						},
					},
				},
				{
					Output: v1alpha1.Output{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fluentforward-test-output",
							Namespace: "collector",
						},
						Spec: v1alpha1.OutputSpec{
							Fluentforward: &v1alpha1.Fluentforward{
								TCPClientSettings: v1alpha1.TCPClientSettings{
									Endpoint: &v1alpha1.Endpoint{
										TCPAddr: utils.ToPtr("fluentd.example-tenant-b-ns.svc.cluster.local:24224"),
									},
									TLSSetting: &v1alpha1.TLSClientSetting{
										Insecure: true,
									},
								},
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

	inputCfg.SubscriptionOutputMap = make(map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName)
	for _, subscription := range inputCfg.Subscriptions {
		inputCfg.SubscriptionOutputMap[subscription.NamespacedName()] = subscription.Spec.Outputs
	}

	inputCfg.TenantSubscriptionMap = make(map[string][]v1alpha1.NamespacedName)
	for _, tenant := range inputCfg.Tenants {
		for _, subscription := range inputCfg.Subscriptions {
			if subscription.Status.Tenant == tenant.Name {
				inputCfg.TenantSubscriptionMap[tenant.Name] = append(inputCfg.TenantSubscriptionMap[tenant.Name], subscription.NamespacedName())
			}
		}
	}

	// Config

	// The receiver and exporter entries are not serialized because of tags on the underlying data structure. The tests won't contain them, this is a known issue.
	generatedConfig, _ := inputCfg.AssembleConfig(context.TODO())
	actualYAMLBytes, err := yaml.Marshal(generatedConfig)
	if err != nil {
		t.Fatalf("error %v", err)
	}

	var actualUniversalMap map[string]any
	if err := yaml.Unmarshal(actualYAMLBytes, &actualUniversalMap); err != nil {
		t.Fatalf("error: %v", err)
	}

	var expectedUniversalMap map[string]any
	if err := yaml.Unmarshal([]byte(otelColTargetYAML), &expectedUniversalMap); err != nil {
		t.Fatalf("error: %v", err)
	}

	// use dyff for YAML comparison
	if diff := cmp.Diff(expectedUniversalMap, actualUniversalMap); diff != "" {
		t.Logf("mismatch:\n---%s\n---\n", diff)
	}

	if !reflect.DeepEqual(actualUniversalMap, expectedUniversalMap) {
		t.Fatalf(`yaml mismatch:
expected=
---
%s
---
actual=
---
%s
---`, otelColTargetYAML, string(actualYAMLBytes))
	}
}

func TestOtelColConfigInput_generateRoutingConnectorForTenantsSubscription(t *testing.T) {
	type fields struct {
		Tenants               []v1alpha1.Tenant
		Subscriptions         map[v1alpha1.NamespacedName]v1alpha1.Subscription
		OutputsWithSecretData []components.OutputWithSecretData
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
		want   connector.RoutingConnector
	}{
		{
			name: "two_subscriptions",
			fields: fields{
				Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
					{
						Name:      "subsA",
						Namespace: "nsA",
					}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "subsA",
							Namespace: "nsA",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Condition: "true",
							Outputs: []v1alpha1.NamespacedName{
								{
									Namespace: "xy",
									Name:      "zq",
								},
							},
						},
					},
					{
						Name:      "subsB",
						Namespace: "nsA",
					}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "subsB",
							Namespace: "nsA",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Condition: "true",
							Outputs: []v1alpha1.NamespacedName{
								{
									Namespace: "xy",
									Name:      "zq",
								},
							},
						},
					},
				},
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
			want: connector.RoutingConnector{
				Name: "routing/tenant_tenantA_subscriptions",
				Table: []connector.RoutingConnectorTableItem{
					{
						Condition: "true",
						Pipelines: []string{"logs/tenant_tenantA_subscription_nsA_subsA", "logs/tenant_tenantA_subscription_nsA_subsB"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgInput := &OtelColConfigInput{
				ResourceRelations: components.ResourceRelations{
					Subscriptions: tt.fields.Subscriptions,
				},
			}
			got := connector.GenerateRoutingConnectorForTenantsSubscriptions(tt.args.tenantName, v1alpha1.RouteConfig{}, tt.args.subscriptionNames, cfgInput.Subscriptions)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestOtelColConfigInput_generateNamedPipelines(t *testing.T) {
	tests := []struct {
		name              string
		cfgInput          OtelColConfigInput
		expectedPipelines map[string]*otelv1beta1.Pipeline
	}{
		{
			name: "Single tenant with no subscriptions",
			cfgInput: OtelColConfigInput{
				ResourceRelations: components.ResourceRelations{
					Bridges:               nil,
					OutputsWithSecretData: nil,
					TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
						"tenant1": {
							{
								Namespace: "ns1",
								Name:      "sub1",
							},
						},
					},
					SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
						{
							Namespace: "ns1",
							Name:      "sub1",
						}: {},
					},
				},
			},
			expectedPipelines: map[string]*otelv1beta1.Pipeline{
				"logs/tenant_tenant1": pipeline.GenerateRootPipeline([]v1alpha1.Tenant{}, "tenant1"),
				"logs/tenant_tenant1_subscription_ns1_sub1": pipeline.GeneratePipeline(
					[]string{"routing/tenant_tenant1_subscriptions"},
					[]string{"attributes/subscription_sub1"},
					[]string{"routing/subscription_ns1_sub1_outputs"},
				),
				"metrics/output": pipeline.GeneratePipeline(
					[]string{"count/output_metrics"},
					[]string{"deltatocumulative", "attributes/metricattributes"},
					[]string{"prometheus/message_metrics_exporter"},
				),
				"metrics/tenant": pipeline.GeneratePipeline(
					[]string{"count/tenant_metrics"},
					[]string{"deltatocumulative", "attributes/metricattributes"},
					[]string{"prometheus/message_metrics_exporter"},
				),
			},
		},
		{
			name: "Three tenants two bridges",
			cfgInput: OtelColConfigInput{
				ResourceRelations: components.ResourceRelations{
					Tenants: []v1alpha1.Tenant{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "tenant1",
							},
							Spec: v1alpha1.TenantSpec{
								LogSourceNamespaceSelectors: []metav1.LabelSelector{
									{
										MatchLabels: map[string]string{
											"nsSelector": "ns1",
										},
									},
								},
							},
							Status: v1alpha1.TenantStatus{
								LogSourceNamespaces: []string{"ns1"},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "tenant2",
							},
							Spec: v1alpha1.TenantSpec{
								SubscriptionNamespaceSelectors: []metav1.LabelSelector{
									{
										MatchLabels: map[string]string{
											"nsSelector": "ns2",
										},
									},
								},
							},
							Status: v1alpha1.TenantStatus{
								Subscriptions: []v1alpha1.NamespacedName{
									{
										Namespace: "ns2",
										Name:      "sub2",
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "tenant3",
							},
							Spec: v1alpha1.TenantSpec{
								SubscriptionNamespaceSelectors: []metav1.LabelSelector{
									{
										MatchLabels: map[string]string{
											"nsSelector": "ns3",
										},
									},
								},
							},
							Status: v1alpha1.TenantStatus{
								Subscriptions: []v1alpha1.NamespacedName{
									{
										Namespace: "ns3",
										Name:      "sub3",
									},
								},
							},
						},
					},
					Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
						{
							Namespace: "ns2",
							Name:      "sub2",
						}: {
							ObjectMeta: metav1.ObjectMeta{
								Name:      "sub2",
								Namespace: "ns2",
							},
							Spec: v1alpha1.SubscriptionSpec{
								Condition: "true",
								Outputs: []v1alpha1.NamespacedName{
									{
										Namespace: "xy",
										Name:      "zq",
									},
								},
							},
							Status: v1alpha1.SubscriptionStatus{
								Tenant:  "tenant2",
								Outputs: []v1alpha1.NamespacedName{{Namespace: "xy", Name: "zq"}},
							},
						},
						{
							Namespace: "ns3",
							Name:      "sub3",
						}: {
							ObjectMeta: metav1.ObjectMeta{
								Name:      "sub3",
								Namespace: "ns3",
							},
							Spec: v1alpha1.SubscriptionSpec{
								Condition: "true",
								Outputs: []v1alpha1.NamespacedName{
									{
										Namespace: "xy",
										Name:      "zq",
									},
								},
							},
							Status: v1alpha1.SubscriptionStatus{
								Tenant:  "tenant3",
								Outputs: []v1alpha1.NamespacedName{{Namespace: "xy", Name: "zq"}},
							},
						},
					},
					Bridges: []v1alpha1.Bridge{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "bridge1",
							},
							Spec: v1alpha1.BridgeSpec{
								SourceTenant: "tenant1",
								TargetTenant: "tenant2",
								Condition:    `attributes["parsed"]["method"] == "GET"`,
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "bridge2",
							},
							Spec: v1alpha1.BridgeSpec{
								SourceTenant: "tenant1",
								TargetTenant: "tenant3",
								Condition:    `attributes["parsed"]["method"] == "PUT"`,
							},
						},
					},
					TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
						"tenant1": {},
						"tenant2": {
							{
								Namespace: "ns2",
								Name:      "sub2",
							},
						},
						"tenant3": {
							{
								Namespace: "ns3",
								Name:      "sub3",
							},
						},
					},
					SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
						{
							Namespace: "ns2",
							Name:      "sub2",
						}: {
							{
								Namespace: "xy",
								Name:      "zq",
							},
						},
						{
							Namespace: "ns3",
							Name:      "sub3",
						}: {
							{
								Namespace: "xy",
								Name:      "zq",
							},
						},
					},
				},
				Debug: false,
			},
			expectedPipelines: map[string]*otelv1beta1.Pipeline{
				"logs/tenant_tenant1": {
					Receivers:  []string{"filelog/tenant1"},
					Processors: []string{"k8sattributes", "attributes/tenant_tenant1", "filter/exclude"},
					Exporters:  []string{"count/tenant_metrics", "routing/bridge_bridge1", "routing/bridge_bridge2"},
				},
				"logs/tenant_tenant2": {
					Receivers:  []string{"routing/bridge_bridge1"},
					Processors: []string{"k8sattributes", "attributes/tenant_tenant2", "filter/exclude"},
					Exporters:  []string{"routing/tenant_tenant2_subscriptions", "count/tenant_metrics"},
				},
				"logs/tenant_tenant2_subscription_ns2_sub2": {
					Receivers:  []string{"routing/tenant_tenant2_subscriptions"},
					Processors: []string{"attributes/subscription_sub2"},
					Exporters:  []string{"routing/subscription_ns2_sub2_outputs"},
				},
				"logs/tenant_tenant3": {
					Receivers:  []string{"routing/bridge_bridge2"},
					Processors: []string{"k8sattributes", "attributes/tenant_tenant3", "filter/exclude"},
					Exporters:  []string{"routing/tenant_tenant3_subscriptions", "count/tenant_metrics"},
				},
				"logs/tenant_tenant3_subscription_ns3_sub3": {
					Receivers:  []string{"routing/tenant_tenant3_subscriptions"},
					Processors: []string{"attributes/subscription_sub3"},
					Exporters:  []string{"routing/subscription_ns3_sub3_outputs"},
				},
				"metrics/output": {
					Receivers:  []string{"count/output_metrics"},
					Processors: []string{"deltatocumulative", "attributes/metricattributes"},
					Exporters:  []string{"prometheus/message_metrics_exporter"},
				},
				"metrics/tenant": {
					Receivers:  []string{"count/tenant_metrics"},
					Processors: []string{"deltatocumulative", "attributes/metricattributes"},
					Exporters:  []string{"prometheus/message_metrics_exporter"},
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedPipelines, ttp.cfgInput.generateNamedPipelines())
		})
	}
}
