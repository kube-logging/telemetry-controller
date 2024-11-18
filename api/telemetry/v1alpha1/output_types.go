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

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Batch struct {
	// From 	go.opentelemetry.io/collector/processor/batchprocessor

	// +kubebuilder:validation:Format=duration

	// Timeout sets the time after which a batch will be sent regardless of size.
	// When this is set to zero, batched data will be sent immediately.
	Timeout string `json:"timeout,omitempty"`

	// SendBatchSize is the size of a batch which after hit, will trigger it to be sent.
	// When this is set to zero, the batch size is ignored and data will be sent immediately
	// subject to only send_batch_max_size.
	SendBatchSize uint32 `json:"send_batch_size,omitempty"`

	// SendBatchMaxSize is the maximum size of a batch. It must be larger than SendBatchSize.
	// Larger batches are split into smaller units.
	// Default value is 0, that means no maximum size.
	SendBatchMaxSize uint32 `json:"send_batch_max_size,omitempty"`

	// MetadataKeys is a list of client.Metadata keys that will be
	// used to form distinct batchers.  If this setting is empty,
	// a single batcher instance will be used.  When this setting
	// is not empty, one batcher will be used per distinct
	// combination of values for the listed metadata keys.
	//
	// Empty value and unset metadata are treated as distinct cases.
	//
	// Entries are case-insensitive.  Duplicated entries will
	// trigger a validation error.
	MetadataKeys []string `json:"metadata_keys,omitempty"`

	// MetadataCardinalityLimit indicates the maximum number of
	// batcher instances that will be created through a distinct
	// combination of MetadataKeys.
	MetadataCardinalityLimit uint32 `json:"metadata_cardinality_limit,omitempty"`
}

// OutputSpec defines the desired state of Output
type OutputSpec struct {
	OTLPGRPC       *OTLPGRPC      `json:"otlp,omitempty"`
	Fluentforward  *Fluentforward `json:"fluentforward,omitempty"`
	OTLPHTTP       *OTLPHTTP      `json:"otlphttp,omitempty"`
	Authentication *OutputAuth    `json:"authentication,omitempty"`
	Batch          *Batch         `json:"batch,omitempty"`
}

type OutputAuth struct {
	BasicAuth  *BasicAuthConfig  `json:"basicauth,omitempty"`
	BearerAuth *BearerAuthConfig `json:"bearerauth,omitempty"`
}

type BasicAuthConfig struct {
	SecretRef     *corev1.SecretReference `json:"secretRef,omitempty"`
	UsernameField string                  `json:"usernameField,omitempty"`
	PasswordField string                  `json:"passwordField,omitempty"`
}

type BearerAuthConfig struct {
	SecretRef  *corev1.SecretReference `json:"secretRef,omitempty"`
	TokenField string                  `json:"tokenField,omitempty"`
}

// OTLP grpc exporter config ref: https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlpexporter/config.go
type OTLPGRPC struct {
	QueueConfig      QueueSettings `json:"sending_queue,omitempty" yaml:"sending_queue,omitempty"`
	RetryConfig      BackOffConfig `json:"retry_on_failure,omitempty" yaml:"retry_on_failure,omitempty"`
	TimeoutSettings  `json:",inline" yaml:",inline"`
	GRPCClientConfig `json:",inline" yaml:",inline"`
}

type OTLPHTTP struct {
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

	Kubernetes *KubernetesMetadata `json:"kubernetes_metadata,omitempty" yaml:"kubernetes_metadata,omitempty"`
}

type KubernetesMetadata struct {
	Key              string `json:"key" yaml:"key,omitempty"`
	IncludePodLabels bool   `json:"include_pod_labels" yaml:"include_pod_labels,omitempty"`
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

// OutputStatus defines the observed state of Output
type OutputStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=telemetry-all

// Output is the Schema for the outputs API
type Output struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OutputSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status OutputStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OutputList contains a list of Output
type OutputList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Output `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Output{}, &OutputList{})
}

func (o *Output) NamespacedName() NamespacedName {
	return NamespacedName{Namespace: o.Namespace, Name: o.Name}
}
