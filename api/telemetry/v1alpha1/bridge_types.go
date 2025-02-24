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

// BridgeSpec defines the desired state of Bridge
type BridgeSpec struct {
	// +kubebuilder:validation:Required

	// The source tenant from which telemetry will be forwarded.
	SourceTenant string `json:"sourceTenant"`

	// +kubebuilder:validation:Required

	// The target tenant to which telemetry will be forwarded.
	TargetTenant string `json:"targetTenant"`

	// +kubebuilder:validation:Required

	// The condition which must be satisfied in order to forward telemetry
	// from the source tenant to the target tenant.
	Condition string `json:"condition"`
}

// BridgeStatus defines the observed state of Bridge
type BridgeStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,categories=telemetry-all
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Source Tenant",type=string,JSONPath=`.spec.sourceTenant`
//+kubebuilder:printcolumn:name="Target Tenant",type=string,JSONPath=`.spec.targetTenant`

// Bridge is the Schema for the Bridges API
type Bridge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BridgeSpec   `json:"spec,omitempty"`
	Status BridgeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BridgeList contains a list of Bridge
type BridgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bridge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bridge{}, &BridgeList{})
}
