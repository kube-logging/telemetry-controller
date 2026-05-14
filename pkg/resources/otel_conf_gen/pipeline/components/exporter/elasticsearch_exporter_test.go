// Copyright © 2026 Kube logging authors
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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
)

func TestGenerateElasticsearchExporters(t *testing.T) {
	tests := []struct {
		name              string
		resourceRelations components.ResourceRelations
		expectedResult    map[string]any
	}{
		{
			name: "Basic endpoint",
			resourceRelations: components.ResourceRelations{
				Tenants: []v1alpha1.Tenant{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: testTenantName,
						},
						Spec: v1alpha1.TenantSpec{
							PersistenceConfig: v1alpha1.PersistenceConfig{
								EnableFileStorage: false,
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
									Name:      "es-output1",
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
								Name:      "es-output1",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								Elasticsearch: &v1alpha1.Elasticsearch{
									Endpoint: new("https://elasticsearch:9200"),
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
							Name:      "es-output1",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"elasticsearch/default_es-output1": map[string]any{
					"endpoint": "https://elasticsearch:9200",
					"retry": map[string]any{
						"enabled": true,
					},
					"sending_queue": map[string]any{
						"enabled":    true,
						"queue_size": float64(1000),
					},
				},
			},
		},
		{
			name: "With basic auth",
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
							Name:      "sub2",
							Namespace: "default",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{
									Name:      "es-output2",
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
								Name:      "es-output2",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								Elasticsearch: &v1alpha1.Elasticsearch{
									Endpoint: new("https://elasticsearch:9200"),
								},
								Authentication: &v1alpha1.OutputAuth{
									BasicAuth: &v1alpha1.BasicAuthConfig{
										SecretRef: &corev1.SecretReference{
											Name:      "es-secret",
											Namespace: "default",
										},
										UsernameField: "username",
										PasswordField: "password",
									},
								},
							},
						},
						Secret: corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "es-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("elastic"),
								"password": []byte("changeme"),
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {
						{
							Name:      "sub2",
							Namespace: "default",
						},
					},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{
						Name:      "sub2",
						Namespace: "default",
					}: {
						{
							Name:      "es-output2",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"elasticsearch/default_es-output2": map[string]any{
					"endpoint": "https://elasticsearch:9200",
					"auth": map[string]any{
						"authenticator": "basicauth/default_es-output2",
					},
					"retry": map[string]any{
						"enabled": true,
					},
					"sending_queue": map[string]any{
						"enabled":    true,
						"queue_size": float64(1000),
					},
				},
			},
		},
		{
			name: "With bearer auth",
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
							Name:      "sub3",
							Namespace: "default",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{
									Name:      "es-output3",
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
								Name:      "es-output3",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								Elasticsearch: &v1alpha1.Elasticsearch{
									Endpoint: new("https://elasticsearch:9200"),
								},
								Authentication: &v1alpha1.OutputAuth{
									BearerAuth: &v1alpha1.BearerAuthConfig{
										SecretRef: &corev1.SecretReference{
											Name:      "es-bearer-secret",
											Namespace: "default",
										},
										TokenField: "token",
									},
								},
							},
						},
						Secret: corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "es-bearer-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"token": []byte("my-bearer-token"),
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {
						{
							Name:      "sub3",
							Namespace: "default",
						},
					},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{
						Name:      "sub3",
						Namespace: "default",
					}: {
						{
							Name:      "es-output3",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"elasticsearch/default_es-output3": map[string]any{
					"endpoint": "https://elasticsearch:9200",
					"auth": map[string]any{
						"authenticator": "bearertokenauth/default_es-output3",
					},
					"retry": map[string]any{
						"enabled": true,
					},
					"sending_queue": map[string]any{
						"enabled":    true,
						"queue_size": float64(1000),
					},
				},
			},
		},
		{
			name: "With persistence",
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
							Name:      "sub4",
							Namespace: "default",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{
									Name:      "es-output4",
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
								Name:      "es-output4",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								Elasticsearch: &v1alpha1.Elasticsearch{
									Endpoint: new("https://elasticsearch:9200"),
									QueueConfig: &v1alpha1.QueueSettings{
										NumConsumers: new(5),
										QueueSize:    new(500),
									},
								},
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {
						{
							Name:      "sub4",
							Namespace: "default",
						},
					},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{
						Name:      "sub4",
						Namespace: "default",
					}: {
						{
							Name:      "es-output4",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"elasticsearch/default_es-output4": map[string]any{
					"endpoint": "https://elasticsearch:9200",
					"retry": map[string]any{
						"enabled": true,
					},
					"sending_queue": map[string]any{
						"enabled":       true,
						"num_consumers": float64(5),
						"queue_size":    float64(500),
					},
				},
			},
		},
		{
			name: "All fields set",
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
							Name:      "sub5",
							Namespace: "default",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{
									Name:      "es-output5",
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
								Name:      "es-output5",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								Elasticsearch: &v1alpha1.Elasticsearch{
									Endpoints:    []string{"https://es1:9200", "https://es2:9200"},
									CloudID:      new("my-cloud-id"),
									User:         new("elastic"),
									Password:     new("changeme"),
									APIKey:       new("base64encodedkey"),
									LogsIndex:    new("logs-otel"),
									MetricsIndex: new("metrics-otel"),
									TracesIndex:  new("traces-otel"),
									NumWorkers:   new(4),
									Pipeline:     new("my-ingest-pipeline"),
									Mapping: &v1alpha1.ElasticsearchMapping{
										Mode: new("otel"),
									},
									Retry: &v1alpha1.ElasticsearchRetry{
										Enabled:         new(true),
										MaxRetries:      new(3),
										InitialInterval: new("100ms"),
										MaxInterval:     new("1m"),
										RetryOnStatus:   []int{429, 502},
									},
									Flush: &v1alpha1.ElasticsearchFlush{
										Bytes:    new(5000000),
										Interval: new("5s"),
									},
									Discover: &v1alpha1.ElasticsearchDiscover{
										OnStart:  new(true),
										Interval: new("30s"),
									},
									QueueConfig: &v1alpha1.QueueSettings{
										NumConsumers: new(10),
										QueueSize:    new(2000),
									},
									TLSSetting: &v1alpha1.TLSClientSetting{
										Insecure:           true,
										InsecureSkipVerify: true,
										ServerName:         "es-server",
									},
								},
								Authentication: &v1alpha1.OutputAuth{
									BasicAuth: &v1alpha1.BasicAuthConfig{
										SecretRef: &corev1.SecretReference{
											Name:      "es-secret",
											Namespace: "default",
										},
										UsernameField: "username",
										PasswordField: "password",
									},
								},
							},
						},
						Secret: corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "es-secret",
								Namespace: "default",
							},
							Data: map[string][]byte{
								"username": []byte("elastic"),
								"password": []byte("changeme"),
							},
						},
					},
				},
				TenantSubscriptionMap: map[string][]v1alpha1.NamespacedName{
					testTenantName: {
						{
							Name:      "sub5",
							Namespace: "default",
						},
					},
				},
				SubscriptionOutputMap: map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{
					{
						Name:      "sub5",
						Namespace: "default",
					}: {
						{
							Name:      "es-output5",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"elasticsearch/default_es-output5": map[string]any{
					"endpoints":     []any{"https://es1:9200", "https://es2:9200"},
					"cloudid":       "my-cloud-id",
					"user":          "elastic",
					"password":      "changeme",
					"api_key":       "base64encodedkey",
					"logs_index":    "logs-otel",
					"metrics_index": "metrics-otel",
					"traces_index":  "traces-otel",
					"num_workers":   float64(4),
					"pipeline":      "my-ingest-pipeline",
					"mapping": map[string]any{
						"mode": "otel",
					},
					"retry": map[string]any{
						"enabled":          true,
						"max_retries":      float64(3),
						"initial_interval": "100ms",
						"max_interval":     "1m",
						"retry_on_status":  []any{float64(429), float64(502)},
					},
					"flush": map[string]any{
						"bytes":    float64(5000000),
						"interval": "5s",
					},
					"discover": map[string]any{
						"on_start": true,
						"interval": "30s",
					},
					"tls": map[string]any{
						"insecure":             true,
						"insecure_skip_verify": true,
						"server_name_override": "es-server",
					},
					"auth": map[string]any{
						"authenticator": "basicauth/default_es-output5",
					},
					"sending_queue": map[string]any{
						"enabled":       true,
						"num_consumers": float64(10),
						"queue_size":    float64(2000),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedResult, GenerateElasticsearchExporters(context.TODO(), ttp.resourceRelations))
		})
	}
}
