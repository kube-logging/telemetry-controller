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

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/pkg/resources/problem"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"
)

type MemoryLimiter struct {
	// From 	go.opentelemetry.io/collector/processor/memorylimiterprocessor

	// CheckInterval is the time between measurements of memory usage for the
	// purposes of avoiding going over the limits. Defaults to zero, so no
	// checks will be performed.
	CheckInterval time.Duration `json:"check_interval"`

	// MemoryLimitMiB is the maximum amount of memory, in MiB, targeted to be
	// allocated by the process.
	MemoryLimitMiB uint32 `json:"limit_mib"`

	// MemorySpikeLimitMiB is the maximum, in MiB, spike expected between the
	// measurements of memory usage.
	MemorySpikeLimitMiB uint32 `json:"spike_limit_mib"`

	// MemoryLimitPercentage is the maximum amount of memory, in %, targeted to be
	// allocated by the process. The fixed memory settings MemoryLimitMiB has a higher precedence.
	MemoryLimitPercentage uint32 `json:"limit_percentage"`

	// MemorySpikePercentage is the maximum, in percents against the total memory,
	// spike expected between the measurements of memory usage.
	MemorySpikePercentage uint32 `json:"spike_limit_percentage"`
}

// +kubebuilder:validation:XValidation:rule="!has(self.dryRunMode) || !self.dryRunMode || (has(self.debug) && self.debug)",message="dryRunMode can only be set to true when debug is explicitly set to true"

// CollectorSpec defines the desired state of Collector
type CollectorSpec struct {
	// +kubebuilder:validation:Required

	// TenantSelector is used to select tenants for which the collector should collect data.
	TenantSelector metav1.LabelSelector `json:"tenantSelector"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable, please recreate the resource"

	// Namespace where the Otel collector DaemonSet is deployed.
	ControlNamespace string `json:"controlNamespace"`

	// Enables debug logging for the collector.
	Debug *bool `json:"debug,omitempty"`

	// DryRunMode disables all exporters except for the debug exporter, as well as persistence options configured for the collector.
	// This can be useful for testing and debugging purposes.
	DryRunMode *bool `json:"dryRunMode,omitempty"`

	// Setting memory limits for the Collector using the memory limiter processor.
	MemoryLimiter *MemoryLimiter `json:"memoryLimiter,omitempty"`

	// OtelcommonFields is used to override the default DaemonSet's common fields.
	OtelCommonFields *otelv1beta1.OpenTelemetryCommonFields `json:"otelCommonFields,omitempty"`
}

func (c *CollectorSpec) SetDefaults() {
	if c.MemoryLimiter == nil {
		c.MemoryLimiter = &MemoryLimiter{
			CheckInterval:         1 * time.Second,
			MemoryLimitPercentage: 75,
			MemorySpikeLimitMiB:   25,
		}
	}
	if c.OtelCommonFields == nil {
		c.OtelCommonFields = &otelv1beta1.OpenTelemetryCommonFields{}
	}
}

func (c CollectorSpec) GetMemoryLimit() *resource.Quantity {
	if c.OtelCommonFields.Resources.Requests != nil && c.OtelCommonFields.Resources.Limits != nil {
		return c.OtelCommonFields.Resources.Limits.Memory()
	}

	return nil
}

// CollectorStatus defines the observed state of Collector
type CollectorStatus struct {
	Tenants       []string    `json:"tenants,omitempty"`
	State         state.State `json:"state,omitempty"`
	Problems      []string    `json:"problems,omitempty"`
	ProblemsCount int         `json:"problemsCount,omitempty"`
}

func (c *Collector) GetProblems() []string {
	return c.Status.Problems
}

func (c *Collector) SetProblems(problems []string) {
	c.Status.Problems = problems
	c.Status.ProblemsCount = len(problems)
}

func (c *Collector) AddProblem(probs ...string) {
	problem.Add(c, probs...)
}

func (c *Collector) ClearProblems() {
	c.SetProblems([]string{})
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,categories=telemetry-all
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Tenants",type=string,JSONPath=`.status.tenants`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// Collector is the Schema for the collectors API
type Collector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollectorSpec   `json:"spec,omitempty"`
	Status CollectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CollectorList contains a list of Collector
type CollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Collector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Collector{}, &CollectorList{})
}
