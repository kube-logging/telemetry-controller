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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/utils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateOTLPGRPCExporters(t *testing.T) {
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
							OTLPGRPC: &v1alpha1.OTLPGRPC{
								GRPCClientConfig: v1alpha1.GRPCClientConfig{
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
				"otlp/default_output1": map[string]any{
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
							OTLPGRPC: &v1alpha1.OTLPGRPC{
								GRPCClientConfig: v1alpha1.GRPCClientConfig{
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
				"otlp/default_output2": map[string]any{
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
							OTLPGRPC: &v1alpha1.OTLPGRPC{
								GRPCClientConfig: v1alpha1.GRPCClientConfig{
									Endpoint: utils.ToPtr("http://example.com"),
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"otlp/default_output3": map[string]any{
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
							OTLPGRPC: &v1alpha1.OTLPGRPC{
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
								TimeoutSettings: v1alpha1.TimeoutSettings{
									Timeout: utils.ToPtr(5 * time.Second),
								},
								GRPCClientConfig: v1alpha1.GRPCClientConfig{
									Endpoint:    utils.ToPtr("http://example.com"),
									Compression: utils.ToPtr(configcompression.Type("gzip")),
									TLSSetting: &v1alpha1.TLSClientSetting{
										Insecure:           true,
										InsecureSkipVerify: true,
										ServerName:         "server-name",
									},
									Keepalive: &v1alpha1.KeepaliveClientConfig{
										Time:                5 * time.Second,
										Timeout:             5 * time.Second,
										PermitWithoutStream: true,
									},
									ReadBufferSize:  utils.ToPtr(1024),
									WriteBufferSize: utils.ToPtr(1024),
									WaitForReady:    utils.ToPtr(true),
									Headers: &map[string]string{
										"header1": "value1",
									},
									BalancerName: utils.ToPtr("round_robin"),
									Authority:    utils.ToPtr("authority"),
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"otlp/default_output4": map[string]any{
					"endpoint": "http://example.com",
					"tls": map[string]any{
						"insecure":             true,
						"insecure_skip_verify": true,
						"server_name_override": "server-name",
					},
					"compression": "gzip",
					"keepalive": map[string]any{
						"time":                  float64(5 * time.Second),
						"timeout":               float64(5 * time.Second),
						"permit_without_stream": true,
					},
					"read_buffer_size":  float64(1024),
					"write_buffer_size": float64(1024),
					"wait_for_ready":    true,
					"headers": map[string]any{
						"header1": "value1",
					},
					"balancer_name": "round_robin",
					"authority":     "authority",
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
					"timeout": float64(5 * time.Second),
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedResult, GenerateOTLPGRPCExporters(context.TODO(), ttp.outputsWithSecretData))
		})
	}
}
