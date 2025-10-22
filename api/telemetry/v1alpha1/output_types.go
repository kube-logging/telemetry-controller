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

	"go.opentelemetry.io/collector/component"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/pkg/resources/problem"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"
)

// Batch processor configuration.
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
	File           *File          `json:"file,omitempty"`
	Authentication *OutputAuth    `json:"authentication,omitempty"`
	Batch          *Batch         `json:"batch,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.basicAuth) && has(self.bearerAuth)) == false",message="Only one authentication method can be specified: either basicAuth or bearerAuth, not both"

// Output Authentication configuration.
type OutputAuth struct {
	BasicAuth  *BasicAuthConfig  `json:"basicAuth,omitempty"`
	BearerAuth *BearerAuthConfig `json:"bearerAuth,omitempty"`
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

// Configuration for the OTLP gRPC exporter.
// ref: https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlpexporter/config.go
type OTLPGRPC struct {
	QueueConfig      *QueueSettings `json:"sending_queue,omitempty"`
	RetryConfig      *BackOffConfig `json:"retry_on_failure,omitempty"`
	TimeoutSettings  `json:",inline"`
	GRPCClientConfig `json:",inline"`
}

// Configuration for the OTLP HTTP exporter.
type OTLPHTTP struct {
	QueueConfig      *QueueSettings `json:"sending_queue,omitempty"`
	RetryConfig      *BackOffConfig `json:"retry_on_failure,omitempty"`
	HTTPClientConfig `json:",inline"`

	// +kubebuilder:validation:Enum:=proto;json

	// The encoding to export telemetry (default: "proto")
	Encoding *string `json:"encoding,omitempty"`
}

type formatType string

const (
	FormatTypeJSON  formatType = "json"
	FormatTypeProto formatType = "proto"
)

type compression string

const (
	CompressionZstd compression = "zstd"
)

// File defines configuration for the file exporter.
type File struct {
	// Path of the file to write to. Path is relative to current directory.
	Path string `json:"path,omitempty"`

	// Mode defines whether the exporter should append to the file.
	// Options:
	// - false[default]:  truncates the file
	// - true:  appends to the file.
	Append bool `json:"append,omitempty"`

	// Rotation defines an option about rotation of telemetry files. Ignored
	// when GroupByAttribute is used.
	Rotation *Rotation `json:"rotation,omitempty"`

	// +kubebuilder:validation:Enum:=proto;json

	// FormatType define the data format of encoded telemetry data
	// Options:
	// - json[default]:  OTLP json bytes.
	// - proto:  OTLP binary protobuf bytes.

	FormatType formatType `json:"format,omitempty"`

	// Encoding defines the encoding of the telemetry data.
	// If specified, it overrides `FormatType` and applies an encoding extension.
	Encoding *component.ID `json:"encoding,omitempty"`

	// +kubebuilder:validation:Enum:=zstd

	// Compression Codec used to export telemetry data
	// Supported compression algorithms:`zstd`
	Compression compression `json:"compression,omitempty"`

	// FlushInterval is the duration between flushes.
	// See time.ParseDuration for valid values.
	FlushInterval time.Duration `json:"flush_interval,omitempty"`

	// GroupBy enables writing to separate files based on a resource attribute.
	GroupBy *GroupBy `json:"group_by,omitempty"`
}

// Rotation an option to rolling log files
type Rotation struct {
	// MaxMegabytes is the maximum size in megabytes of the file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxMegabytes int `json:"max_megabytes,omitempty"`

	// MaxDays is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxDays int `json:"max_days,omitempty"`

	// MaxBackups is the maximum number of old log files to retain. The default
	// is to 100 files.
	MaxBackups int `json:"max_backups,omitempty"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime *bool `json:"localtime,omitempty"`
}

type GroupBy struct {
	// Enables group_by. When group_by is enabled, rotation setting is ignored.  Default is false.
	Enabled bool `json:"enabled,omitempty"`

	// ResourceAttribute specifies the name of the resource attribute that
	// contains the path segment of the file to write to. The final path will be
	// the Path config value, with the * replaced with the value of this resource
	// attribute. Default is "fileexporter.path_segment".
	ResourceAttribute string `json:"resource_attribute,omitempty"`

	// MaxOpenFiles specifies the maximum number of open file descriptors for the output files.
	// The default is 100.
	MaxOpenFiles int `json:"max_open_files,omitempty"`
}

type Endpoint struct {
	// TCPAddr is the address of the server to connect to.
	TCPAddr *string `json:"tcp_addr"`
	// Controls whether to validate the tcp address.
	// Turning this ON may result in the collector failing to start if it came up faster then the endpoint.
	// default is false.
	ValidateTCPResolution bool `json:"validate_tcp_resolution,omitempty"`
}

type KubernetesMetadata struct {
	Key              string `json:"key"`
	IncludePodLabels bool   `json:"include_pod_labels"`
}

// TCPClientSettings defines common settings for a TCP client.
type TCPClientSettings struct {
	// +kubebuilder:validation:Required

	// Endpoint to send logs to.
	*Endpoint `json:"endpoint"`

	// +kubebuilder:validation:Format=duration

	// Connection Timeout parameter configures `net.Dialer`.
	ConnectionTimeout *string `json:"connection_timeout,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *TLSClientSetting `json:"tls,omitempty"`

	// SharedKey is used for authorization with the server that knows it.
	SharedKey *string `json:"shared_key,omitempty"`
}

// Configuration for the fluentforward exporter.
type Fluentforward struct {
	TCPClientSettings `json:",inline"`

	// RequireAck enables the acknowledgement feature.
	RequireAck *bool `json:"require_ack,omitempty"`

	// The Fluent tag parameter used for routing
	Tag *string `json:"tag,omitempty"`

	// CompressGzip enables gzip compression for the payload.
	CompressGzip *bool `json:"compress_gzip,omitempty"`

	// DefaultLabelsEnabled is a map of default attributes to be added to each log record.
	DefaultLabelsEnabled *map[string]bool `json:"default_labels_enabled,omitempty"`

	QueueConfig *QueueSettings      `json:"sending_queue,omitempty"`
	RetryConfig *BackOffConfig      `json:"retry_on_failure,omitempty"`
	Kubernetes  *KubernetesMetadata `json:"kubernetes_metadata,omitempty"`
}

// OutputStatus defines the observed state of Output
type OutputStatus struct {
	Tenant        string      `json:"tenant,omitempty"`
	State         state.State `json:"state,omitempty"`
	Problems      []string    `json:"problems,omitempty"`
	ProblemsCount int         `json:"problemsCount,omitempty"`
}

func (o *Output) GetTenant() string {
	return o.Status.Tenant
}

func (o *Output) SetTenant(tenant string) {
	o.Status.Tenant = tenant
}

func (o *Output) GetState() state.State {
	return o.Status.State
}

func (o *Output) SetState(state state.State) {
	o.Status.State = state
}

func (o *Output) GetProblems() []string {
	return o.Status.Problems
}

func (o *Output) SetProblems(problems []string) {
	o.Status.Problems = problems
	o.Status.ProblemsCount = len(problems)
}

func (o *Output) AddProblem(probs ...string) {
	problem.Add(o, probs...)
}

func (o *Output) ClearProblems() {
	o.SetProblems([]string{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.status.tenant`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:resource:categories=telemetry-all

// Output is the Schema for the outputs API
type Output struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OutputSpec   `json:"spec,omitempty"`
	Status OutputStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OutputList contains a list of Output
type OutputList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Output `json:"items"`
}

func (l *OutputList) GetItems() []model.ResourceOwnedByTenant {
	items := make([]model.ResourceOwnedByTenant, len(l.Items))
	for i := range l.Items {
		items[i] = &l.Items[i]
	}
	return items
}

func init() {
	SchemeBuilder.Register(&Output{}, &OutputList{})
}

func (o *Output) NamespacedName() NamespacedName {
	return NamespacedName{Namespace: o.Namespace, Name: o.Name}
}
