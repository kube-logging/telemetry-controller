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
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

type ElasticsearchWrapper struct {
	*v1alpha1.Elasticsearch

	// Shadows Elasticsearch.QueueConfig to use wrapper type with defaults.
	QueueConfig *queueWrapper `json:"sending_queue,omitempty"`
	// Shadows Elasticsearch.Retry to apply enabled=true default.
	Retry *v1alpha1.ElasticsearchRetry `json:"retry,omitempty"`
	// Auth extension reference, set from OutputSpec.Authentication.
	Auth *v1alpha1.Authentication `json:"auth,omitempty"`
}

func (w *ElasticsearchWrapper) mapToElasticsearchWrapper(apiConfig *v1alpha1.Elasticsearch) {
	w.Elasticsearch = apiConfig

	w.Retry = setDefaultElasticsearchRetry(apiConfig.Retry)
	w.QueueConfig = &queueWrapper{}
	w.QueueConfig.setDefaultQueueSettings(apiConfig.QueueConfig)
}

func setDefaultElasticsearchRetry(apiRetry *v1alpha1.ElasticsearchRetry) *v1alpha1.ElasticsearchRetry {
	retry := &v1alpha1.ElasticsearchRetry{
		Enabled: utils.ToPtr(true),
	}

	if apiRetry != nil {
		if apiRetry.Enabled != nil {
			retry.Enabled = apiRetry.Enabled
		}
		retry.MaxRetries = apiRetry.MaxRetries
		retry.InitialInterval = apiRetry.InitialInterval
		retry.MaxInterval = apiRetry.MaxInterval
		retry.RetryOnStatus = apiRetry.RetryOnStatus
	}

	return retry
}

func GenerateElasticsearchExporters(ctx context.Context, resourceRelations components.ResourceRelations) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)
	for _, output := range resourceRelations.OutputsWithSecretData {
		if output.Output.Spec.Elasticsearch != nil {
			internalConfig := ElasticsearchWrapper{}
			internalConfig.mapToElasticsearchWrapper(output.Output.Spec.Elasticsearch)
			if _, err := resourceRelations.FindTenantForOutput(output.Output.NamespacedName()); err != nil {
				logger.Error(err, "failed to find tenant for output, skipping", "output", output.Output.NamespacedName().String())
				continue
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
	}

	return result
}
