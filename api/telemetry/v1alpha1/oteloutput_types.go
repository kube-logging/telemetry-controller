// Copyright © 2023 Kube logging authors
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

package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OtelOutputSpec defines the desired state of OtelOutput
type OtelOutputSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of OtelOutput. Edit oteloutput_types.go to remove/update
	OTLP          *OTLP          `json:"otlp,omitempty"`
	Loki          *Loki          `json:"loki,omitempty"`
	Fluentforward *Fluentforward `json:"fluentforward,omitempty"`
}

// OTLP grpc exporter config ref: https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlpexporter/config.go
type OTLP struct {
	QueueConfig      QueueSettings `json:"sending_queue,omitempty" yaml:"sending_queue,omitempty"`
	RetryConfig      BackOffConfig `json:"retry_on_failure,omitempty" yaml:"retry_on_failure,omitempty"`
	TimeoutSettings  `json:",inline" yaml:",inline"`
	GRPCClientConfig `json:",inline" yaml:",inline"`
}

type Loki struct {
	QueueConfig      QueueSettings `json:"sending_queue,omitempty" yaml:"sending_queue,omitempty"`
	RetryConfig      BackOffConfig `json:"retry_on_failure,omitempty" yaml:"retry_on_failure,omitempty"`
	HTTPClientConfig `json:",inline" yaml:",inline"`
}

type Fluentforward struct {
	TCPClientSettings `json:",inline" yaml:",inline"` // squash ensures fields are correctly decoded in embedded struct.

	// RequireAck enables the acknowledgement feature.
	RequireAck bool `json:"require_ack,omitempty" yaml:"require_ack,omitempty"`

	// The Fluent tag parameter used for routing
	Tag string `json:"tag,omitempty" yaml:"tag,omitempty"`

	// CompressGzip enables gzip compression for the payload.
	CompressGzip bool `json:"compress_gzip,omitempty" yaml:"compress_gzip,omitempty"`

	// DefaultLabelsEnabled is a map of default attributes to be added to each log record.
	DefaultLabelsEnabled map[string]bool `json:"default_labels_enabled,omitempty" yaml:"default_labels_enabled,omitempty"`

	QueueConfig QueueSettings `json:"sending_queue,omitempty" yaml:"sending_queue,omitempty"`
	RetryConfig BackOffConfig `json:"retry_on_failure,omitempty" yaml:"retry_on_failure,omitempty"`
}

type TCPClientSettings struct {
	// The target endpoint URI to send data to (e.g.: some.url:24224).
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

	// Connection Timeout parameter configures `net.Dialer`.
	ConnectionTimeout time.Duration `json:"connection_timeout,omitempty" yaml:"connection_timeout,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting TLSClientSetting `json:"tls,omitempty" yaml:"tls,omitempty"`

	// SharedKey is used for authorization with the server that knows it.
	SharedKey string `json:"shared_key,omitempty" yaml:"shared_key,omitempty"`
}

// OtelOutputStatus defines the observed state of OtelOutput
type OtelOutputStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=telemetry-all

// OtelOutput is the Schema for the oteloutputs API
type OtelOutput struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OtelOutputSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status OtelOutputStatus `json:"status,omitempty" `
}

// +kubebuilder:object:root=true

// OtelOutputList contains a list of OtelOutput
type OtelOutputList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OtelOutput `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OtelOutput{}, &OtelOutputList{})
}

func (o *OtelOutput) NamespacedName() NamespacedName {
	return NamespacedName{Namespace: o.Namespace, Name: o.Name}
}
