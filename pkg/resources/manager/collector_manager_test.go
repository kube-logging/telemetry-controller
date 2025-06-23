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

package manager

import (
	"testing"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSetOtelCommonFieldsDefaults(t *testing.T) {
	tests := []struct {
		name                string
		initialCommonFields *otelv1beta1.OpenTelemetryCommonFields
		additionalArgs      map[string]string
		saName              string
		expectedResult      *otelv1beta1.OpenTelemetryCommonFields
		expectedError       error
	}{
		{
			name: "Basic Initialization",
			initialCommonFields: &otelv1beta1.OpenTelemetryCommonFields{
				Args: map[string]string{},
			},
			additionalArgs: map[string]string{
				"key1": "value1",
			},
			saName: "test-sa",
			expectedResult: &otelv1beta1.OpenTelemetryCommonFields{
				Image:          axoflowOtelCollectorImageRef,
				ServiceAccount: "test-sa",
				Args: map[string]string{
					"key1": "value1",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "varlog",
						ReadOnly:  true,
						MountPath: "/var/log",
					},
					{
						Name:      "varlibdockercontainers",
						ReadOnly:  true,
						MountPath: "/var/lib/docker/containers",
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "varlog",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/log",
							},
						},
					},
					{
						Name: "varlibdockercontainers",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/lib/docker/containers",
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Override Existing Args",
			initialCommonFields: &otelv1beta1.OpenTelemetryCommonFields{
				Args: map[string]string{
					"existingKey": "existingValue",
				},
			},
			additionalArgs: map[string]string{
				"existingKey": "value1",
			},
			saName: "test-sa",
			expectedResult: &otelv1beta1.OpenTelemetryCommonFields{
				Image:          axoflowOtelCollectorImageRef,
				ServiceAccount: "test-sa",
				Args: map[string]string{
					"existingKey": "value1",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "varlog",
						ReadOnly:  true,
						MountPath: "/var/log",
					},
					{
						Name:      "varlibdockercontainers",
						ReadOnly:  true,
						MountPath: "/var/lib/docker/containers",
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "varlog",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/log",
							},
						},
					},
					{
						Name: "varlibdockercontainers",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/lib/docker/containers",
							},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name:                "Nil Initial Fields",
			initialCommonFields: nil,
			additionalArgs: map[string]string{
				"key1": "value1",
			},
			saName:         "test-sa",
			expectedResult: nil,
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			var commonFieldsCopy *otelv1beta1.OpenTelemetryCommonFields
			if ttp.initialCommonFields != nil {
				commonFieldsCopy = ttp.initialCommonFields.DeepCopy()
			}
			setOtelCommonFieldsDefaults(commonFieldsCopy, ttp.additionalArgs, ttp.saName)

			assert.Equal(t, ttp.expectedResult, commonFieldsCopy)
		})
	}
}
