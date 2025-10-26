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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configopaque"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

const (
	DefaultPrometheusExporterID = "prometheus/message_metrics_exporter"
)

type TLSServerConfig struct {
	// squash ensures fields are correctly decoded in embedded struct.
	v1alpha1.TLSSetting `json:",inline"`

	// These are config options specific to server connections.

	// Path to the TLS cert to use by the server to verify a client certificate. (optional)
	// This sets the ClientCAs and ClientAuth to RequireAndVerifyClientCert in the TLSConfig. Please refer to
	// https://godoc.org/crypto/tls#Config for more information. (optional)
	ClientCAFile string `json:"client_ca_file"`

	// Reload the ClientCAs file when it is modified
	// (optional, default false)
	ReloadClientCAFile bool `json:"client_ca_file_reload"`
}

// CORSConfig configures a receiver for HTTP cross-origin resource sharing (CORS).
// See the underlying https://github.com/rs/cors package for details.
type CORSConfig struct {
	// AllowedOrigins sets the allowed values of the Origin header for
	// HTTP/JSON requests to an OTLP receiver. An origin may contain a
	// wildcard (*) to replace 0 or more characters (e.g.,
	// "http://*.domain.com", or "*" to allow any origin).
	AllowedOrigins []string `json:"allowed_origins"`

	// AllowedHeaders sets what headers will be allowed in CORS requests.
	// The Accept, Accept-Language, Content-Type, and Content-Language
	// headers are implicitly allowed. If no headers are listed,
	// X-Requested-With will also be accepted by default. Include "*" to
	// allow any request header.
	AllowedHeaders []string `json:"allowed_headers"`

	// MaxAge sets the value of the Access-Control-Max-Age response header.
	// Set it to the number of seconds that browsers should cache a CORS
	// preflight response for.
	MaxAge int `json:"max_age"`
}

type HTTPServerConfig struct {
	// Endpoint configures the listening address for the server.
	Endpoint string `json:"endpoint,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *TLSServerConfig `json:"tls,omitempty"`

	// CORS configures the server for HTTP cross-origin resource sharing (CORS).
	CORS *CORSConfig `json:"cors,omitempty"`

	// Auth for this receiver
	Auth *configauth.Config `json:"auth,omitempty"`

	// MaxRequestBodySize sets the maximum request body size in bytes
	MaxRequestBodySize int64 `json:"max_request_body_size,omitempty"`

	// IncludeMetadata propagates the client metadata from the incoming requests to the downstream consumers
	// Experimental: *NOTE* this option is subject to change or removal in the future.
	IncludeMetadata bool `json:"include_metadata,omitempty"`

	// Additional headers attached to each HTTP response sent to the client.
	// Header values are opaque since they may be sensitive.
	ResponseHeaders map[string]configopaque.String `json:"response_headers,omitempty"`
}

type ResourceToTelemetrySettings struct {
	// Enabled indicates whether to convert resource attributes to telemetry attributes. Default is `false`.
	Enabled bool `json:"enabled,omitempty"`
}

type PrometheusExporterConfig struct {
	HTTPServerConfig `json:",inline"`

	// Namespace if set, exports metrics under the provided value.
	Namespace string `json:"namespace,omitempty"`

	// ConstLabels are values that are applied for every exported metric.
	ConstLabels prometheus.Labels `json:"const_labels,omitempty"`

	// SendTimestamps will send the underlying scrape timestamp with the export
	SendTimestamps bool `json:"send_timestamps,omitempty"`

	// MetricExpiration defines how long metrics are kept without updates
	MetricExpiration time.Duration `json:"metric_expiration,omitempty"`

	// ResourceToTelemetrySettings defines configuration for converting resource attributes to metric labels.
	ResourceToTelemetrySettings *ResourceToTelemetrySettings `json:"resource_to_telemetry_conversion,omitempty"`

	// EnableOpenMetrics enables the use of the OpenMetrics encoding option for the prometheus exporter.
	EnableOpenMetrics bool `json:"enable_open_metrics,omitempty"`

	// AddMetricSuffixes controls whether suffixes are added to metric names. Defaults to true.
	AddMetricSuffixes bool `json:"add_metric_suffixes,omitempty"`
}

func GenerateMetricsExporters() map[string]any {
	defaultPrometheusExporterConfig := PrometheusExporterConfig{
		HTTPServerConfig: HTTPServerConfig{
			Endpoint: ":9999",
		},
	}

	metricsExporters := make(map[string]any)
	metricsExporters[DefaultPrometheusExporterID] = defaultPrometheusExporterConfig

	return metricsExporters
}
