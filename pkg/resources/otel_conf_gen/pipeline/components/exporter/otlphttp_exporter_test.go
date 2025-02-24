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
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configopaque"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

func TestGenerateOTLPHTTPExporters(t *testing.T) {
	tests := []struct {
		name              string
		resourceRelations components.ResourceRelations
		expectedResult    map[string]any
	}{
		{
			name: "Basic auth",
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
								OTLPHTTP: &v1alpha1.OTLPHTTP{
									HTTPClientConfig: v1alpha1.HTTPClientConfig{
										Endpoint: utils.ToPtr("http://example.com"),
									},
								},
								Authentication: &v1alpha1.OutputAuth{
									BasicAuth: &v1alpha1.BasicAuthConfig{
										SecretRef: &corev1.SecretReference{
											Name:      "secret-name",
											Namespace: "secret-ns",
										},
										UsernameField: "username",
										PasswordField: "password",
									},
								},
							},
						},
						Secret: corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "secret-name",
								Namespace: "secret-ns",
							},
							Data: map[string][]byte{
								"username": []byte("user"),
								"password": []byte("pass"),
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
				"otlphttp/default_output1": map[string]any{
					"endpoint": "http://example.com",
					"auth": map[string]any{
						"authenticator": "basicauth/default_output1",
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
			name: "Bearer auth",
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
								OTLPHTTP: &v1alpha1.OTLPHTTP{
									HTTPClientConfig: v1alpha1.HTTPClientConfig{
										Endpoint: utils.ToPtr("http://example.com"),
									},
								},
								Authentication: &v1alpha1.OutputAuth{
									BearerAuth: &v1alpha1.BearerAuthConfig{
										SecretRef: &corev1.SecretReference{
											Name:      "secret-name",
											Namespace: "secret-ns",
										},
										TokenField: "token",
									},
								},
							},
						},
						Secret: corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "secret-name",
								Namespace: "secret-ns",
							},
							Data: map[string][]byte{
								"token": []byte("token-value"),
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
							Name:      "output2",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"otlphttp/default_output2": map[string]any{
					"endpoint": "http://example.com",
					"auth": map[string]any{
						"authenticator": "bearertokenauth/default_output2",
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
			name: "No auth",
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
									Name:      "output3",
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
								Name:      "output3",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								OTLPHTTP: &v1alpha1.OTLPHTTP{
									HTTPClientConfig: v1alpha1.HTTPClientConfig{
										Endpoint: utils.ToPtr("http://example.com"),
									},
								},
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
							Name:      "output3",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"otlphttp/default_output3": map[string]any{
					"endpoint": "http://example.com",
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
							Name:      "sub4",
							Namespace: "default",
						},
						Spec: v1alpha1.SubscriptionSpec{
							Outputs: []v1alpha1.NamespacedName{
								{
									Name:      "output4",
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
								Name:      "output4",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								OTLPHTTP: &v1alpha1.OTLPHTTP{
									QueueConfig: &v1alpha1.QueueSettings{
										NumConsumers: utils.ToPtr(10),
										QueueSize:    utils.ToPtr(100),
									},
									RetryConfig: &v1alpha1.BackOffConfig{
										InitialInterval:     utils.ToPtr(5 * time.Second),
										RandomizationFactor: utils.ToPtr("0.1"),
										Multiplier:          utils.ToPtr("2.0"),
										MaxInterval:         utils.ToPtr(10 * time.Second),
										MaxElapsedTime:      utils.ToPtr(60 * time.Second),
									},
									HTTPClientConfig: v1alpha1.HTTPClientConfig{
										Endpoint: utils.ToPtr("http://example.com"),
										ProxyURL: utils.ToPtr("http://proxy.example.com"),
										TLSSetting: &v1alpha1.TLSClientSetting{
											Insecure:           true,
											InsecureSkipVerify: true,
											ServerName:         "server-name",
										},
										ReadBufferSize:  utils.ToPtr(1024),
										WriteBufferSize: utils.ToPtr(1024),
										Timeout:         utils.ToPtr(5 * time.Second),
										Headers: &map[string]configopaque.String{
											"header1": configopaque.String("value1"),
										},
										Compression:          utils.ToPtr(configcompression.Type("gzip")),
										MaxIdleConns:         utils.ToPtr(10),
										MaxIdleConnsPerHost:  utils.ToPtr(10),
										MaxConnsPerHost:      utils.ToPtr(10),
										IdleConnTimeout:      utils.ToPtr(5 * time.Second),
										DisableKeepAlives:    utils.ToPtr(true),
										HTTP2ReadIdleTimeout: utils.ToPtr(5 * time.Second),
										HTTP2PingTimeout:     utils.ToPtr(5 * time.Second),
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
							Name:      "output4",
							Namespace: "default",
						},
					},
				},
			},
			expectedResult: map[string]any{
				"otlphttp/default_output4": map[string]any{
					"endpoint":  "http://example.com",
					"proxy_url": "http://proxy.example.com",
					"tls": map[string]any{
						"insecure":             true,
						"insecure_skip_verify": true,
						"server_name_override": "server-name",
					},
					"read_buffer_size":  float64(1024),
					"write_buffer_size": float64(1024),
					"timeout":           float64(5 * time.Second),
					"headers": map[string]any{
						"header1": configopaque.String("value1").String(),
					},
					"compression":             "gzip",
					"max_idle_conns":          float64(10),
					"max_idle_conns_per_host": float64(10),
					"max_conns_per_host":      float64(10),
					"idle_conn_timeout":       float64(5 * time.Second),
					"disable_keep_alives":     true,
					"http2_read_idle_timeout": float64(5 * time.Second),
					"http2_ping_timeout":      float64(5 * time.Second),
					"sending_queue": map[string]any{
						"enabled":       true,
						"num_consumers": float64(10),
						"queue_size":    float64(100),
						"storage":       fmt.Sprintf("file_storage/%s", testTenantName),
					},
					"retry_on_failure": map[string]any{
						"enabled":              true,
						"initial_interval":     float64(5 * time.Second),
						"randomization_factor": "0.1",
						"multiplier":           "2.0",
						"max_interval":         float64(10 * time.Second),
						"max_elapsed_time":     float64(60 * time.Second),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedResult, GenerateOTLPHTTPExporters(context.TODO(), ttp.resourceRelations))
		})
	}
}
