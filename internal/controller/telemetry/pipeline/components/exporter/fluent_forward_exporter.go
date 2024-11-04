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
	"errors"
	"fmt"

	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func GenerateFluentforwardExporters(ctx context.Context, outputsWithSecretData []components.OutputWithSecretData) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)
	for _, output := range outputsWithSecretData {
		if output.Output.Spec.Fluentforward != nil {
			// TODO: add proper error handling
			name := fmt.Sprintf("fluentforwardexporter/%s_%s", output.Output.Namespace, output.Output.Name)
			fluentForwardMarshaled, err := yaml.Marshal(output.Output.Spec.Fluentforward)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}
			var fluetForwardValues map[string]any
			if err := yaml.Unmarshal(fluentForwardMarshaled, &fluetForwardValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
			}

			result[name] = fluetForwardValues
		}
	}

	return result
}
