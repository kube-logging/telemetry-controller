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

package connector

import "github.com/kube-logging/telemetry-controller/pkg/sdk/utils"

type BytesConnectorAttributes struct {
	Key          *string `json:"key,omitempty"`
	DefaultValue *string `json:"default_value,omitempty"`
}

type BytesConnector struct {
	Description string                     `json:"description,omitempty"`
	Attributes  []BytesConnectorAttributes `json:"attributes,omitempty"`
}

func GenerateBytesConnectors() map[string]any {
	bytesConnectors := make(map[string]any)

	bytesConnectors["bytes/exporter"] = map[string]any{
		"logs": map[string]BytesConnector{
			"otelcol_exporter_sent_log_records_bytes": {
				Description: "Bytes of log records successfully sent to destination",
				Attributes: []BytesConnectorAttributes{{
					Key: utils.ToPtr("exporter"),
				}},
			},
		},
	}

	return bytesConnectors
}
