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
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/resources"
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/resources/state"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SubscriptionSpec defines the desired state of Subscription
type SubscriptionSpec struct {
	// +kubebuilder:validation:Required

	// The condition which must be satisfied in order to forward telemetry to the outputs.
	Condition string `json:"condition"`

	// +kubebuilder:validation:Required

	// The outputs to which the logs will be routed if the condition evaluates to true.
	Outputs []NamespacedName `json:"outputs"`
}

// SubscriptionStatus defines the observed state of Subscription
type SubscriptionStatus struct {
	Tenant  string           `json:"tenant,omitempty"`
	Outputs []NamespacedName `json:"outputs,omitempty"`
	State   state.State      `json:"state,omitempty"`
}

func (s *Subscription) GetTenant() string {
	return s.Status.Tenant
}

func (s *Subscription) SetTenant(tenant string) {
	s.Status.Tenant = tenant
}

func (s *Subscription) GetState() state.State {
	return s.Status.State
}

func (s *Subscription) SetState(state state.State) {
	s.Status.State = state
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.status.tenant`
// +kubebuilder:printcolumn:name="Outputs",type=string,JSONPath=`.status.outputs`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:resource:categories=telemetry-all

// Subscription is the Schema for the subscriptions API
type Subscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubscriptionSpec   `json:"spec,omitempty"`
	Status SubscriptionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubscriptionList contains a list of Subscription
type SubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subscription `json:"items"`
}

func (l *SubscriptionList) GetItems() []resources.ResourceOwnedByTenant {
	items := make([]resources.ResourceOwnedByTenant, len(l.Items))
	for i := range l.Items {
		items[i] = &l.Items[i]
	}
	return items
}

func init() {
	SchemeBuilder.Register(&Subscription{}, &SubscriptionList{})
}

func (s *Subscription) NamespacedName() NamespacedName {
	return NamespacedName{Namespace: s.Namespace, Name: s.Name}
}
