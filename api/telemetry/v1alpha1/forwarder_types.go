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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ForwarderSpec defines the desired state of Forwarder
type ForwarderSpec struct {
	SourceTenant string `json:"sourceTenant,omitempty"`
	TargetTenant string `json:"targetTenant,omitempty"`
	// The OTTL condition which must be satisfied in order to forward telemetry
	// from the source tenant to the target tenant
	OTTL string `json:"ottl,omitempty"`
}

// ForwarderStatus defines the observed state of Forwarder
type ForwarderStatus struct {
	State string `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,categories=telemetry-all
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// Forwarder is the Schema for the forwarders API
type Forwarder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ForwarderSpec   `json:"spec,omitempty"`
	Status ForwarderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ForwarderList contains a list of Forwarder
type ForwarderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Forwarder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Forwarder{}, &ForwarderList{})
}
