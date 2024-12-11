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

// TransformStatement represents a single statement in a Transform processor.
// ref: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor
type TransformStatement struct {
	// +kubebuilder:validation:Enum:=resource;scope;span;spanevent;metric;datapoint;log

	Context    string   `json:"context,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Statements []string `json:"statements,omitempty"`
}

// Transform represents the Transform processor, which modifies telemetry based on its configuration.
type Transform struct {
	// Name of the Transform processor
	Name string `json:"name,omitempty"`

	// When FlattenData is true, the processor provides each log record with
	// a distinct copy of its resource and scope. Then, after applying all transformations,
	// the log records are regrouped by resource and scope.
	FlattenData bool `json:"flattenData,omitempty"`

	// +kubebuilder:validation:Enum:=ignore;silent;propagate

	// ErrorMode specifies how errors are handled while processing a statement
	// vaid options are: ignore, silent, propagate; (default: propagate)
	ErrorMode string `json:"errorMode,omitempty"`

	TraceStatements  []TransformStatement `json:"traceStatements,omitempty"`
	MetricStatements []TransformStatement `json:"metricStatements,omitempty"`
	LogStatements    []TransformStatement `json:"logStatements,omitempty"`
}

// RouteConfig defines the routing configuration for a tenant
// it will be used to generate routing connectors.
type RouteConfig struct {
	// Contains the list of pipelines to use when a record does not meet any of specified conditions.
	DefaultPipelines []string `json:"defaultPipelines,omitempty"` // TODO: Provide users with a guide to determine generated pipeline names

	// +kubebuilder:validation:Enum:=ignore;silent;propagate

	// ErrorMode specifies how errors are handled while processing a statement
	// vaid options are: ignore, silent, propagate; (default: propagate)
	ErrorMode string `json:"errorMode,omitempty"`

	// Determines whether the connector matches multiple statements or not.
	// If enabled, the payload will be routed to the first pipeline in the table whose routing condition is met.
	// May only be false when used with resource context.
	MatchOnce bool `json:"matchOnce,omitempty"`
}

// Configuration for persistence, will be used to generate
// the filestorage extension.
type PersistenceConfig struct {
	// Determines whether file storage is enabled or not.
	EnableFileStorage bool `json:"enableFileStorage,omitempty"`

	// The directory where logs will be persisted.
	// If unset or an invalid path is given, then an OS specific
	// default value will be used.
	// The cluster administrator must ensure that the directory
	// is unique for each tenant.
	// If unset /var/lib/otelcol/file_storage/<tenant_name> will be used.
	Directory string `json:"directory,omitempty"`
}

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// Determines the namespaces from which subscriptions are collected by this tenant.
	SubscriptionNamespaceSelectors []metav1.LabelSelector `json:"subscriptionNamespaceSelectors,omitempty"`

	// Determines the namespaces from which logs are collected by this tenant.
	// Cannot be used together with SelectFromAllNamespaces.
	LogSourceNamespaceSelectors []metav1.LabelSelector `json:"logSourceNamespaceSelectors,omitempty"`

	// If true, logs are collected from all namespaces.
	// Cannot be used together with LogSourceNamespaceSelectors.
	SelectFromAllNamespaces bool `json:"selectFromAllNamespaces,omitempty"`
	Transform               `json:"transform,omitempty"`
	RouteConfig             `json:"routeConfig,omitempty"`
	PersistenceConfig       `json:"persistenceConfig,omitempty"`
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
