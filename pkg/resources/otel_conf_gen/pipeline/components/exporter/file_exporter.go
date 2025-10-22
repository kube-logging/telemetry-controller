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
	"encoding/json"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
)

func GenerateFileExporter(ctx context.Context, resourceRelations components.ResourceRelations) map[string]any {
	logger := log.FromContext(ctx)

	result := make(map[string]any)
	for _, output := range resourceRelations.OutputsWithSecretData {
		if output.Output.Spec.File != nil {
			fileValuesMarshaled, err := json.Marshal(output.Output.Spec.File)
			if err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
				continue
			}
			var fileValues map[string]any
			if err := json.Unmarshal(fileValuesMarshaled, &fileValues); err != nil {
				logger.Error(errors.New("failed to compile config for output"), "failed to compile config for output %q", output.Output.NamespacedName().String())
				continue
			}

			result[fmt.Sprintf("file/%s_%s", output.Output.Namespace, output.Output.Name)] = fileValues
		}
	}

	return result
}
