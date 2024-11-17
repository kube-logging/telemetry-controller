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
	"testing"
	"time"

	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configopaque"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/utils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateOTLPHTTPExporters(t *testing.T) {
	tests := []struct {
		name                  string
		outputsWithSecretData []components.OutputWithSecretData
		expectedResult        map[string]any
	}{
		{
			name: "Basic auth",
			outputsWithSecretData: []components.OutputWithSecretData{
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
			expectedResult: map[string]any{
				"otlphttp/default_output1": map[string]any{
					"endpoint": "http://example.com",
					"auth": map[string]any{
						"authenticator": "basicauth/default_output1",
					},
				},
			},
		},
		{
			name: "Bearer auth",
			outputsWithSecretData: []components.OutputWithSecretData{
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
			expectedResult: map[string]any{
				"otlphttp/default_output2": map[string]any{
					"endpoint": "http://example.com",
					"auth": map[string]any{
						"authenticator": "bearertokenauth/default_output2",
					},
				},
			},
		},
		{
			name: "No auth",
			outputsWithSecretData: []components.OutputWithSecretData{
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
			expectedResult: map[string]any{
				"otlphttp/default_output3": map[string]any{
					"endpoint": "http://example.com",
				},
			},
		},
		{
			name: "All fields set",
			outputsWithSecretData: []components.OutputWithSecretData{
				{
					Output: v1alpha1.Output{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "output4",
							Namespace: "default",
						},
						Spec: v1alpha1.OutputSpec{
							OTLPHTTP: &v1alpha1.OTLPHTTP{
								QueueConfig: &v1alpha1.QueueSettings{
									Enabled:      true,
									NumConsumers: 10,
									QueueSize:    100,
									StorageID:    "storage-id",
								},
								RetryConfig: &v1alpha1.BackOffConfig{
									Enabled:             true,
									InitialInterval:     5 * time.Second,
									RandomizationFactor: "0.1",
									Multiplier:          "2.0",
									MaxInterval:         10 * time.Second,
									MaxElapsedTime:      60 * time.Second,
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
						"storage":       "storage-id",
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
			assert.Equal(t, ttp.expectedResult, GenerateOTLPHTTPExporters(context.TODO(), ttp.outputsWithSecretData))
		})
	}
}
