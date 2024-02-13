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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OtelOutputSpec defines the desired state of OtelOutput
type OtelOutputSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of OtelOutput. Edit oteloutput_types.go to remove/update
	OTLP OTLPgrpc `json:"otlp,omitempty"`
}

// OTLP grpc exporter config ref: https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlpexporter/config.go
type OTLPgrpc struct {
	QueueConfig     QueueSettings `json:"sending_queue,omitempty" yaml:"sending_queue,omitempty"`
	RetryConfig     BackOffConfig `json:"retry_on_failure,omitempty" yaml:"retry_on_failure,omitempty"`
	TimeoutSettings `json:",inline" yaml:",inline"`
	ClientConfig    `json:",inline" yaml:",inline"`
}

// OtelOutputStatus defines the observed state of OtelOutput
type OtelOutputStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OtelOutput is the Schema for the oteloutputs API
type OtelOutput struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OtelOutputSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status OtelOutputStatus `json:"status,omitempty" `
}

//+kubebuilder:object:root=true

// OtelOutputList contains a list of OtelOutput
type OtelOutputList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OtelOutput `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OtelOutput{}, &OtelOutputList{})
}
