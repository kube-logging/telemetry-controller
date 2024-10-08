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
	"fmt"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

func GetExporterNameForOutput(output v1alpha1.Output) string {
	var exporterName string
	if output.Spec.OTLPGRPC != nil {
		exporterName = fmt.Sprintf("otlp/%s_%s", output.Namespace, output.Name)
	} else if output.Spec.OTLPHTTP != nil {
		exporterName = fmt.Sprintf("otlphttp/%s_%s", output.Namespace, output.Name)
	} else if output.Spec.Fluentforward != nil {
		exporterName = fmt.Sprintf("fluentforwardexporter/%s_%s", output.Namespace, output.Name)
	}

	return exporterName
}
