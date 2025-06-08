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

package telemetry

import (
	"context"
	"fmt"
	"testing"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

func TestCollectorReconciler_populateSecretForOutput(t *testing.T) {
	tests := []struct {
		name                  string
		output                *v1alpha1.Output
		existingSecrets       []runtime.Object
		expectedProblemsCount int
		expectedProblems      []string
		expectedState         state.State
	}{
		{
			name: "no authentication - no problems",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec:       v1alpha1.OutputSpec{},
				Status:     v1alpha1.OutputStatus{},
			},
			existingSecrets:       []runtime.Object{},
			expectedProblemsCount: 0,
			expectedProblems:      []string{},
			expectedState:         "",
		},
		{
			name: "basic auth with existing secret - no problems",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec: v1alpha1.OutputSpec{
					Authentication: &v1alpha1.OutputAuth{
						BasicAuth: &v1alpha1.BasicAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "basic-secret",
								Namespace: "default",
							},
						},
					},
				},
				Status: v1alpha1.OutputStatus{},
			},
			existingSecrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "basic-secret", Namespace: "default"},
					Data:       map[string][]byte{"username": []byte("user"), "password": []byte("pass")},
				},
			},
			expectedProblemsCount: 0,
			expectedProblems:      []string{},
			expectedState:         "",
		},
		{
			name: "basic auth with missing secret - one problem",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec: v1alpha1.OutputSpec{
					Authentication: &v1alpha1.OutputAuth{
						BasicAuth: &v1alpha1.BasicAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "missing-secret",
								Namespace: "default",
							},
						},
					},
				},
				Status: v1alpha1.OutputStatus{},
			},
			existingSecrets:       []runtime.Object{},
			expectedProblemsCount: 1,
			expectedProblems:      []string{"failed getting secrets for output test-output: secrets \"missing-secret\" not found"},
			expectedState:         state.StateFailed,
		},
		{
			name: "bearer auth with existing secret - no problems",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec: v1alpha1.OutputSpec{
					Authentication: &v1alpha1.OutputAuth{
						BearerAuth: &v1alpha1.BearerAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "bearer-secret",
								Namespace: "default",
							},
						},
					},
				},
				Status: v1alpha1.OutputStatus{},
			},
			existingSecrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "bearer-secret", Namespace: "default"},
					Data:       map[string][]byte{"token": []byte("bearer-token")},
				},
			},
			expectedProblemsCount: 0,
			expectedProblems:      []string{},
			expectedState:         "",
		},
		{
			name: "bearer auth with missing secret - one problem",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec: v1alpha1.OutputSpec{
					Authentication: &v1alpha1.OutputAuth{
						BearerAuth: &v1alpha1.BearerAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "missing-bearer-secret",
								Namespace: "default",
							},
						},
					},
				},
				Status: v1alpha1.OutputStatus{},
			},
			existingSecrets:       []runtime.Object{},
			expectedProblemsCount: 1,
			expectedProblems:      []string{"failed getting secrets for output test-output: secrets \"missing-bearer-secret\" not found"},
			expectedState:         state.StateFailed,
		},
		{
			name: "both auth types with both secrets missing - two problems",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec: v1alpha1.OutputSpec{
					Authentication: &v1alpha1.OutputAuth{
						BasicAuth: &v1alpha1.BasicAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "missing-secret",
								Namespace: "default",
							},
						},
						BearerAuth: &v1alpha1.BearerAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "missing-bearer-secret",
								Namespace: "default",
							},
						},
					},
				},
				Status: v1alpha1.OutputStatus{},
			},
			existingSecrets:       []runtime.Object{},
			expectedProblemsCount: 2,
			expectedProblems: []string{
				"failed getting secrets for output test-output: secrets \"missing-secret\" not found",
				"failed getting secrets for output test-output: secrets \"missing-bearer-secret\" not found",
			},
			expectedState: state.StateFailed,
		},
		{
			name: "both auth types with one secret missing - one problem",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec: v1alpha1.OutputSpec{
					Authentication: &v1alpha1.OutputAuth{
						BasicAuth: &v1alpha1.BasicAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "existing-basic-secret",
								Namespace: "default",
							},
						},
						BearerAuth: &v1alpha1.BearerAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "missing-bearer-secret",
								Namespace: "default",
							},
						},
					},
				},
				Status: v1alpha1.OutputStatus{},
			},
			existingSecrets: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "existing-basic-secret", Namespace: "default"},
					Data:       map[string][]byte{"username": []byte("user"), "password": []byte("pass")},
				},
			},
			expectedProblemsCount: 1,
			expectedProblems:      []string{"failed getting secrets for output test-output: secrets \"missing-bearer-secret\" not found"},
			expectedState:         state.StateFailed,
		},
		{
			name: "output with existing problems - adds new problem",
			output: &v1alpha1.Output{
				ObjectMeta: metav1.ObjectMeta{Name: "test-output", Namespace: "default"},
				Spec: v1alpha1.OutputSpec{
					Authentication: &v1alpha1.OutputAuth{
						BasicAuth: &v1alpha1.BasicAuthConfig{
							SecretRef: &corev1.SecretReference{
								Name:      "missing-secret",
								Namespace: "default",
							},
						},
					},
				},
				Status: v1alpha1.OutputStatus{
					Problems:      []string{"existing problem"},
					ProblemsCount: 1,
				},
			},
			existingSecrets:       []runtime.Object{},
			expectedProblemsCount: 2,
			expectedProblems: []string{
				"existing problem",
				"failed getting secrets for output test-output: secrets \"missing-secret\" not found",
			},
			expectedState: state.StateFailed,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			require.NoError(t, v1alpha1.AddToScheme(scheme))

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ttp.output).
				WithRuntimeObjects(ttp.existingSecrets...).
				Build()

			r := &CollectorReconciler{
				fakeClient,
				scheme,
			}

			outputWithSecret := &components.OutputWithSecretData{
				Output: *ttp.output,
			}
			_ = r.populateSecretForOutput(context.Background(), ttp.output, outputWithSecret)

			assert.Equal(t, ttp.expectedProblemsCount, ttp.output.Status.ProblemsCount,
				"ProblemsCount should match expected value")

			assert.Equal(t, len(ttp.expectedProblems), len(ttp.output.Status.Problems),
				"Number of problems should match expected")

			for i, expectedProblem := range ttp.expectedProblems {
				if i < len(ttp.output.Status.Problems) {
					assert.Equal(t, expectedProblem, ttp.output.Status.Problems[i],
						fmt.Sprintf("Problem %d should match expected", i))
				}
			}

			if ttp.expectedState != "" {
				assert.Equal(t, ttp.expectedState, ttp.output.Status.State,
					"State should match expected value")
			}
		})
	}
}
