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

type Batch struct {
	// Name of the Batch processor
	Name string `json:"name,omitempty"`

	// From 	go.opentelemetry.io/collector/processor/batchprocessor

	// Timeout sets the time after which a batch will be sent regardless of size.
	// When this is set to zero, batched data will be sent immediately.
	Timeout time.Duration `json:"timeout,omitempty"`

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

// TransformStatement represents a single statement in a Transform processor
// ref: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor
type TransformStatement struct {
	// +kubebuilder:validation:Enum:=resource;scope;span;spanevent;metric;datapoint;log
	Context    string   `json:"context,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Statements []string `json:"statements,omitempty"`
}

// Transform represents the Transform processor, which modifies telemetry based on its configuration
type Transform struct {
	// Name of the Transform processor
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Enum:=ignore;silent;propagate

	// ErrorMode specifies how errors are handled while processing a statement
	// vaid options are: ignore, silent, propagate; (default: propagate)
	ErrorMode string `json:"errorMode,omitempty"`

	TraceStatements  []TransformStatement `json:"traceStatements,omitempty"`
	MetricStatements []TransformStatement `json:"metricStatements,omitempty"`
	LogStatements    []TransformStatement `json:"logStatements,omitempty"`
}

// Processors defines the tenant level processors
type Processors struct {
	Transform Transform `json:"transform,omitempty"`
	Batch     *Batch    `json:"batch,omitempty"`
}

// RouteConfig defines the routing configuration for a tenant
// it will be used to generate routing connectors
type RouteConfig struct {
	DefaultPipelines []string `json:"defaultPipelines,omitempty"` // TODO: Provide users with a guide to determine generated pipeline names

	// +kubebuilder:validation:Enum:=ignore;silent;propagate

	// ErrorMode specifies how errors are handled while processing a statement
	// vaid options are: ignore, silent, propagate; (default: propagate)
	ErrorMode string `json:"errorMode,omitempty"`
	MatchOnce bool   `json:"matchOnce,omitempty"`
}

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	SubscriptionNamespaceSelectors []metav1.LabelSelector `json:"subscriptionNamespaceSelectors,omitempty"`
	LogSourceNamespaceSelectors    []metav1.LabelSelector `json:"logSourceNamespaceSelectors,omitempty"`
	Processors                     Processors             `json:"processors,omitempty"`
	RouteConfig                    RouteConfig            `json:"routeConfig,omitempty"`
}

func (t *TenantSpec) SetDefaults(tenantName string) {
	if t.Processors.Batch == nil {
		t.Processors.Batch = &Batch{
			Name:          tenantName,
			Timeout:       200 * time.Millisecond,
			SendBatchSize: 8192,
		}
	}

	if len(t.Processors.Batch.MetadataKeys) > 0 && t.Processors.Batch.MetadataCardinalityLimit == 0 {
		t.Processors.Batch.MetadataCardinalityLimit = 1000
	}
}

// TenantStatus defines the observed state of Tenant
type TenantStatus struct {
	Subscriptions       []NamespacedName `json:"subscriptions,omitempty"`
	LogSourceNamespaces []string         `json:"logSourceNamespaces,omitempty"`
	ConnectedBridges    []string         `json:"connectedBridges,omitempty"`
	State               State            `json:"state,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,categories=telemetry-all
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Subscriptions",type=string,JSONPath=`.status.subscriptions`
//+kubebuilder:printcolumn:name="Logsource namespaces",type=string,JSONPath=`.status.logSourceNamespaces`
//+kubebuilder:printcolumn:name="Connected bridges",type=string,JSONPath=`.status.connectedBridges`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// Tenant is the Schema for the tenants API
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantList contains a list of Tenant
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
