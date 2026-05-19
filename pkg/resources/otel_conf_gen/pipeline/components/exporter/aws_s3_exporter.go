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
	"encoding/json"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
)

// AWSS3Wrapper shadows v1alpha1.AWSS3 so we can layer defaults onto the queue
// and retry configurations the same way the other exporters do, and so the
// storage extension can be injected when tenant persistence is enabled.
type AWSS3Wrapper struct {
	*v1alpha1.AWSS3

	// Shadows AWSS3.QueueConfig to use the wrapper type with defaults.
	QueueConfig *queueWrapper `json:"sending_queue,omitempty"`
	// Shadows AWSS3.RetryConfig to use the wrapper type with defaults.
	RetryConfig *backOffWrapper `json:"retry_on_failure,omitempty"`
}

func (w *AWSS3Wrapper) mapToAWSS3Wrapper(apiConfig *v1alpha1.AWSS3) {
	w.AWSS3 = apiConfig

	w.QueueConfig = &queueWrapper{}
	w.QueueConfig.setDefaultQueueSettings(apiConfig.QueueConfig)

	w.RetryConfig = &backOffWrapper{}
	w.RetryConfig.setDefaultBackOffConfig(apiConfig.RetryConfig)
}

func GenerateAWSS3Exporters(ctx context.Context, resourceRelations components.ResourceRelations) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)
	for _, output := range resourceRelations.OutputsWithSecretData {
		if output.Output.Spec.AWSS3 == nil {
			continue
		}

		internalConfig := AWSS3Wrapper{}
		internalConfig.mapToAWSS3Wrapper(output.Output.Spec.AWSS3)

		tenant, err := resourceRelations.FindTenantForOutput(output.Output.NamespacedName())
		if err != nil {
			logger.Error(err, "failed to find tenant for output, skipping", "output", output.Output.NamespacedName().String())
			continue
		}
		if tenant.Spec.PersistenceConfig.EnableFileStorage {
			internalConfig.QueueConfig.Storage = new(fmt.Sprintf("file_storage/%s", tenant.Name))
		}

		valuesMarshaled, err := json.Marshal(internalConfig)
		if err != nil {
			logger.Error(err, "failed to marshal config for output", "output", output.Output.NamespacedName().String())
			continue
		}
		var values map[string]any
		if err := json.Unmarshal(valuesMarshaled, &values); err != nil {
			logger.Error(err, "failed to unmarshal config for output", "output", output.Output.NamespacedName().String())
			continue
		}

		result[components.GetExporterNameForOutput(output.Output)] = values
	}

	return result
}
