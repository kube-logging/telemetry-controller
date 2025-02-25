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
	"encoding/json"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

type OTLPHTTPWrapper struct {
	QueueConfig               *queueWrapper   `json:"sending_queue,omitempty"`
	RetryConfig               *backOffWrapper `json:"retry_on_failure,omitempty"`
	v1alpha1.HTTPClientConfig `json:",inline"`
}

func (w *OTLPHTTPWrapper) mapToOTLPHTTPWrapper(apiConfig *v1alpha1.OTLPHTTP) {
	w.QueueConfig = &queueWrapper{}
	w.RetryConfig = &backOffWrapper{}
	w.QueueConfig.setDefaultQueueSettings(apiConfig.QueueConfig)
	w.RetryConfig.setDefaultBackOffConfig(apiConfig.RetryConfig)
	w.HTTPClientConfig = apiConfig.HTTPClientConfig
}

func GenerateOTLPHTTPExporters(ctx context.Context, resourceRelations components.ResourceRelations) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)
	for _, output := range resourceRelations.OutputsWithSecretData {
		if output.Output.Spec.OTLPHTTP != nil {
			internalConfig := OTLPHTTPWrapper{}
			internalConfig.mapToOTLPHTTPWrapper(output.Output.Spec.OTLPHTTP)
			tenant, err := resourceRelations.FindTenantForOutput(output.Output.NamespacedName())
			if err != nil {
				logger.Error(err, "failed to find tenant for output, skipping", "output", output.Output.NamespacedName().String())
				continue
			}
			if tenant.Spec.PersistenceConfig.EnableFileStorage {
				internalConfig.QueueConfig.Storage = utils.ToPtr(fmt.Sprintf("file_storage/%s", tenant.Name))
			}

			if output.Output.Spec.Authentication != nil {
				if output.Output.Spec.Authentication.BasicAuth != nil {
					internalConfig.Auth = &v1alpha1.Authentication{
						AuthenticatorID: utils.ToPtr(fmt.Sprintf("basicauth/%s_%s", output.Output.Namespace, output.Output.Name)),
					}
				} else if output.Output.Spec.Authentication.BearerAuth != nil {
					internalConfig.Auth = &v1alpha1.Authentication{
						AuthenticatorID: utils.ToPtr(fmt.Sprintf("bearertokenauth/%s_%s", output.Output.Namespace, output.Output.Name)),
					}
				}
			}
			otlpHttpValuesMarshaled, err := json.Marshal(internalConfig)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}
			var otlpHttpValues map[string]any
			if err := json.Unmarshal(otlpHttpValuesMarshaled, &otlpHttpValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}

			result[components.GetExporterNameForOutput(output.Output)] = otlpHttpValues
		}
	}

	return result
}
