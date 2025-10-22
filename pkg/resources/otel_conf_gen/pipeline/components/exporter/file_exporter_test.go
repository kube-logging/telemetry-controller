// Copyright © 2025 Kube logging authors
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

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

func TestGenerateFileExporter(t *testing.T) {
	tests := []struct {
		name              string
		resourceRelations components.ResourceRelations
		expectedResult    map[string]any
	}{
		{
			name: "Basic file exporter with path only",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output1",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path: "/tmp/logs.json",
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/default_output1": map[string]any{
					"path": "/tmp/logs.json",
				},
			},
		},
		{
			name: "File exporter with append mode",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output2",
								Namespace: "test-ns",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path:   "/var/log/telemetry.log",
									Append: true,
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/test-ns_output2": map[string]any{
					"path":   "/var/log/telemetry.log",
					"append": true,
				},
			},
		},
		{
			name: "File exporter with rotation settings",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output3",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path: "/tmp/rotating.log",
									Rotation: &v1alpha1.Rotation{
										MaxMegabytes: 100,
										MaxDays:      7,
										MaxBackups:   5,
										LocalTime:    utils.ToPtr(true),
									},
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/default_output3": map[string]any{
					"path": "/tmp/rotating.log",
					"rotation": map[string]any{
						"max_megabytes": float64(100),
						"max_days":      float64(7),
						"max_backups":   float64(5),
						"localtime":     true,
					},
				},
			},
		},
		{
			name: "File exporter with format type and compression",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output4",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path:        "/tmp/logs.proto",
									FormatType:  "proto",
									Compression: "zstd",
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/default_output4": map[string]any{
					"path":        "/tmp/logs.proto",
					"format":      "proto",
					"compression": "zstd",
				},
			},
		},
		{
			name: "File exporter with flush interval",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output5",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path:          "/tmp/logs.json",
									FlushInterval: 5 * time.Second,
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/default_output5": map[string]any{
					"path":           "/tmp/logs.json",
					"flush_interval": float64(5 * time.Second),
				},
			},
		},
		{
			name: "File exporter with group by settings",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output6",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path: "/tmp/logs/*.json",
									GroupBy: &v1alpha1.GroupBy{
										Enabled:           true,
										ResourceAttribute: "service.name",
										MaxOpenFiles:      50,
									},
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/default_output6": map[string]any{
					"path": "/tmp/logs/*.json",
					"group_by": map[string]any{
						"enabled":            true,
						"resource_attribute": "service.name",
						"max_open_files":     float64(50),
					},
				},
			},
		},
		{
			name: "File exporter with custom encoding",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output7",
								Namespace: "default",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path:     "/tmp/logs.txt",
									Encoding: utils.ToPtr(component.MustNewID("json")),
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/default_output7": map[string]any{
					"path":     "/tmp/logs.txt",
					"encoding": "json",
				},
			},
		},
		{
			name: "File exporter with all options",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output-all",
								Namespace: "prod",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path:          "/var/log/telemetry/output.json",
									Append:        true,
									FormatType:    "json",
									Compression:   "zstd",
									FlushInterval: 10 * time.Second,
									Rotation: &v1alpha1.Rotation{
										MaxMegabytes: 200,
										MaxDays:      14,
										MaxBackups:   10,
										LocalTime:    utils.ToPtr(false),
									},
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/prod_output-all": map[string]any{
					"path":           "/var/log/telemetry/output.json",
					"append":         true,
					"format":         "json",
					"compression":    "zstd",
					"flush_interval": float64(10 * time.Second),
					"rotation": map[string]any{
						"max_megabytes": float64(200),
						"max_days":      float64(14),
						"max_backups":   float64(10),
						"localtime":     false,
					},
				},
			},
		},
		{
			name: "Multiple file exporters",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output1",
								Namespace: "ns1",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path: "/tmp/output1.json",
								},
							},
						},
					},
					{
						Output: v1alpha1.Output{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "output2",
								Namespace: "ns2",
							},
							Spec: v1alpha1.OutputSpec{
								File: &v1alpha1.File{
									Path:   "/tmp/output2.log",
									Append: true,
								},
							},
						},
					},
				},
			},
			expectedResult: map[string]any{
				"file/ns1_output1": map[string]any{
					"path": "/tmp/output1.json",
				},
				"file/ns2_output2": map[string]any{
					"path":   "/tmp/output2.log",
					"append": true,
				},
			},
		},
		{
			name: "No file exporters",
			resourceRelations: components.ResourceRelations{
				OutputsWithSecretData: []components.OutputWithSecretData{
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
							},
						},
					},
				},
			},
			expectedResult: map[string]any{},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			assert.Equal(t, ttp.expectedResult, GenerateFileExporter(context.TODO(), ttp.resourceRelations))
		})
	}
}
