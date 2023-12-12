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

// CollectorSpec defines the desired state of Collector
type CollectorSpec struct {
	TenantSelectors []metav1.LabelSelector `json:"tenantSelectors,omitempty"`
}

// CollectorStatus defines the observed state of Collector
type CollectorStatus struct {
	Tenants []string `json:"tenants"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Tenants",type=string,JSONPath=`.status.tenants`

// Collector is the Schema for the collectors API
type Collector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollectorSpec   `json:"spec,omitempty"`
	Status CollectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CollectorList contains a list of Collector
type CollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Collector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Collector{}, &CollectorList{})
}
