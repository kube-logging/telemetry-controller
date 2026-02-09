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
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

type OTLPGRPCWrapper struct {
	QueueConfig               *queueWrapper   `json:"sending_queue,omitempty"`
	RetryConfig               *backOffWrapper `json:"retry_on_failure,omitempty"`
	v1alpha1.TimeoutSettings  `json:",inline"`
	v1alpha1.GRPCClientConfig `json:",inline"`
}

func (w *OTLPGRPCWrapper) mapToOTLPGRPCWrapper(apiConfig *v1alpha1.OTLPGRPC) {
	w.QueueConfig = &queueWrapper{}
	w.RetryConfig = &backOffWrapper{}
	w.QueueConfig.setDefaultQueueSettings(apiConfig.QueueConfig)
	w.RetryConfig.setDefaultBackOffConfig(apiConfig.RetryConfig)

	if apiConfig.Timeout != nil {
		w.Timeout = apiConfig.Timeout
	}
	w.GRPCClientConfig = apiConfig.GRPCClientConfig
}

func GenerateOTLPGRPCExporters(ctx context.Context, resourceRelations components.ResourceRelations) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)
	for _, output := range resourceRelations.OutputsWithSecretData {
		if output.Output.Spec.OTLPGRPC != nil {
			internalConfig := OTLPGRPCWrapper{}
			internalConfig.mapToOTLPGRPCWrapper(output.Output.Spec.OTLPGRPC)
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
			otlpGrpcValuesMarshaled, err := json.Marshal(internalConfig)
			if err != nil {
				logger.Error(err, "failed to marshal config for output", "output", output.Output.NamespacedName().String())
				continue
			}
			var otlpGRPCValues map[string]any
			if err := json.Unmarshal(otlpGrpcValuesMarshaled, &otlpGRPCValues); err != nil {
				logger.Error(err, "failed to unmarshal config for output", "output", output.Output.NamespacedName().String())
				continue
			}

			result[components.GetExporterNameForOutput(output.Output)] = otlpGRPCValues
		}
	}

	return result
}
