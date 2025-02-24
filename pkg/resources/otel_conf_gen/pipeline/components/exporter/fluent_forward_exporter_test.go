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

package exporter

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

const testTenantName = "tenant1"

func TestGenerateFluentforwardExporters(t *testing.T) {
	tests := []struct {
		name              string
		resourceRelations components.ResourceRelations
		expectedResult    map[string]any
	}{
		{
			name: "Valid config",
			resourceRelations: components.ResourceRelations{
				Tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: testTenantName,
						},
						Spec: v1alpha1.TenantSpec{
							PersistenceConfig: v1alpha1.PersistenceConfig{
								EnableFileStorage: true,
							},
						},
					},
				},
				Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
					{
						Name:      "default",
						Namespace: "default",
					}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "sub1",
							Namespace: "default",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{
									Name:      "output1",
									Namespace: "default",
								},
							},
						},
					},
				},
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output1",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								Fluentforward: &v1alpha1.Fluentforward{
									TCPClientSettings: v1alpha1.TCPClientSettings{
										Endpoint: &v1alpha1.Endpoint{
											TCPAddr: utils.ToPtr("http://example.com"),
										},
									},
								},
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {
						{
							Name:      "sub1",
							Namespace: "default",
						},
					},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{
						Name:      "sub1",
						Namespace: "default",
					}: {
						{
							Name:      "output1",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"fluentforwardexporter/default_output1": map[string]any{
					"endpoint": map[string]any{
						"tcp_addr": "http://example.com",
					},
					"sending_queue": map[string]any{
						"enabled":    true,
						"queue_size": float64(100),
						"storage":    fmt.Sprintf("file_storage/%s", testTenantName),
					},
					"retry_on_failure": map[string]any{
						"enabled":          true,
						"max_elapsed_time": float64(0),
					},
				},
			},
		},
		{
			name: "All fields set, tls settings omitted",
			resourceRelations: components.ResourceRelations{
				Tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: testTenantName,
						},
						Spec: v1alpha1.TenantSpec{
							PersistenceConfig: v1alpha1.PersistenceConfig{
								EnableFileStorage: true,
							},
						},
					},
				},
				Subscriptions: map[v1alpha1.NamespacedName]v1alpha1.Subscription{
					{
						Name:      "default",
						Namespace: "default",
					}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "sub1",
							Namespace: "default",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{
									Name:      "output2",
									Namespace: "default",
								},
							},
						},
					},
				},
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output2",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								Fluentforward: &v1alpha1.Fluentforward{
									TCPClientSettings: v1alpha1.TCPClientSettings{
										Endpoint: &v1alpha1.Endpoint{
											TCPAddr: utils.ToPtr("http://example.com"),
										},
										ConnectionTimeout: utils.ToPtr("30s"),
										SharedKey:         utils.ToPtr("shared-key"),
									},
									RequireAck:           utils.ToPtr(true),
									Tag:                  utils.ToPtr("tag"),
									CompressGzip:         utils.ToPtr(true),
									DefaultLabelsEnabled: &map[string]bool{"label1": true},
									QueueConfig:          &v1alpha1.QueueSettings{},
									RetryConfig:          &v1alpha1.BackOffConfig{},
									Kubernetes:           &v1alpha1.KubernetesMetadata{Key: "key", IncludePodLabels: true},
								},
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {
						{
							Name:      "sub1",
							Namespace: "default",
						},
					},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{
						Name:      "sub1",
						Namespace: "default",
					}: {
						{
							Name:      "output2",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"fluentforwardexporter/default_output2": map[string]any{
					"endpoint": map[string]any{
						"tcp_addr": "http://example.com",
					},
					"connection_timeout":     "30s",
					"shared_key":             "shared-key",
					"require_ack":            true,
					"tag":                    "tag",
					"compress_gzip":          true,
					"default_labels_enabled": map[string]any{"label1": true},
					"sending_queue": map[string]any{
						"enabled":    true,
						"queue_size": float64(100),
						"storage":    fmt.Sprintf("file_storage/%s", testTenantName),
					},
					"retry_on_failure": map[string]any{
						"enabled":          true,
						"max_elapsed_time": float64(0),
					},
					"kubernetes_metadata": map[string]any{
						"key":                "key",
						"include_pod_labels": true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedResult, GenerateFluentforwardExporters(context.TODO(), ttp.resourceRelations))
		})
	}
}
