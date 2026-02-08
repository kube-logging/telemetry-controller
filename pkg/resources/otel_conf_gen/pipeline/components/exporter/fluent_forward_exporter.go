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

type FluentForwardWrapper struct {
	v1alpha1.TCPClientSettings `json:",inline"`

	// RequireAck enables the acknowledgement feature.
	RequireAck *bool `json:"require_ack,omitempty"`

	// The Fluent tag parameter used for routing
	Tag *string `json:"tag,omitempty"`

	// CompressGzip enables gzip compression for the payload.
	CompressGzip *bool `json:"compress_gzip,omitempty"`

	// DefaultLabelsEnabled is a map of default attributes to be added to each log record.
	DefaultLabelsEnabled *map[string]bool `json:"default_labels_enabled,omitempty"`

	QueueConfig *queueWrapper                `json:"sending_queue,omitempty"`
	RetryConfig *backOffWrapper              `json:"retry_on_failure,omitempty"`
	Kubernetes  *v1alpha1.KubernetesMetadata `json:"kubernetes_metadata,omitempty"`
}

func (w *FluentForwardWrapper) mapToFluentForwardWrapper(userConfig *v1alpha1.Fluentforward) {
	w.QueueConfig = &queueWrapper{}
	w.RetryConfig = &backOffWrapper{}
	w.QueueConfig.setDefaultQueueSettings(userConfig.QueueConfig)
	w.RetryConfig.setDefaultBackOffConfig(userConfig.RetryConfig)

	if userConfig.RequireAck != nil {
		w.RequireAck = userConfig.RequireAck
	}
	if userConfig.Tag != nil {
		w.Tag = userConfig.Tag
	}
	if userConfig.CompressGzip != nil {
		w.CompressGzip = userConfig.CompressGzip
	}
	if userConfig.DefaultLabelsEnabled != nil {
		w.DefaultLabelsEnabled = userConfig.DefaultLabelsEnabled
	}
	if userConfig.Kubernetes != nil {
		w.Kubernetes = userConfig.Kubernetes
	}
	w.TCPClientSettings = userConfig.TCPClientSettings
}

func GenerateFluentforwardExporters(ctx context.Context, resourceRelations components.ResourceRelations) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)
	for _, output := range resourceRelations.OutputsWithSecretData {
		if output.Output.Spec.Fluentforward != nil {
			internalConfig := FluentForwardWrapper{}
			internalConfig.mapToFluentForwardWrapper(output.Output.Spec.Fluentforward)
			tenant, err := resourceRelations.FindTenantForOutput(output.Output.NamespacedName())
			if err != nil {
				logger.Error(err, "failed to find tenant for output, skipping", "output", output.Output.NamespacedName().String())
				continue
			}
			if tenant.Spec.PersistenceConfig.EnableFileStorage {
				internalConfig.QueueConfig.Storage = utils.ToPtr(fmt.Sprintf("file_storage/%s", tenant.Name))
			}

			fluentForwardMarshaled, err := json.Marshal(internalConfig)
			if err != nil {
				logger.Error(err, "failed to marshal config for output", "output", output.Output.NamespacedName().String())
				continue
			}
			var fluentForwardValues map[string]any
			if err := json.Unmarshal(fluentForwardMarshaled, &fluentForwardValues); err != nil {
				logger.Error(err, "failed to unmarshal config for output", "output", output.Output.NamespacedName().String())
				continue
			}

			result[fmt.Sprintf("fluentforwardexporter/%s_%s", output.Output.Namespace, output.Output.Name)] = fluentForwardValues
		}
	}

	return result
}
