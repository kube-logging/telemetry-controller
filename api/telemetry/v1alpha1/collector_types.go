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

	"github.com/cisco-open/operator-tools/pkg/typeoverride"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// CollectorSpec defines the desired state of Collector
type CollectorSpec struct {
	TenantSelector metav1.LabelSelector `json:"tenantSelector,omitempty"`
	// Namespace where OTel collector DaemonSet is deployed
	ControlNamespace string `json:"controlNamespace"`
	// Enables debug logging for the collector
	Debug bool `json:"debug,omitempty"`
	// Setting memory limits for the Collector
	MemoryLimiter      *MemoryLimiter          `json:"memoryLimiter,omitempty"`
	DaemonSetOverrides *typeoverride.DaemonSet `json:"daemonSet,omitempty"`
}

func (c *CollectorSpec) SetDefaults() {
	if c.MemoryLimiter == nil {
		c.MemoryLimiter = &MemoryLimiter{
			CheckInterval:         1 * time.Second,
			MemoryLimitPercentage: 75,
			MemorySpikeLimitMiB:   25,
		}
	}
}

func (c CollectorSpec) GetMemoryLimit() *resource.Quantity {
	if c.DaemonSetOverrides != nil && len(c.DaemonSetOverrides.Spec.Template.Spec.Containers) > 0 {
		if memoryLimit := c.DaemonSetOverrides.Spec.Template.Spec.Containers[0].Resources.Limits.Memory(); !memoryLimit.IsZero() {
			return memoryLimit
		}
	}
	return nil
}

// CollectorStatus defines the observed state of Collector
type CollectorStatus struct {
	State   State    `json:"state,omitempty"`
	Tenants []string `json:"tenants,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,categories=telemetry-all
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Tenants",type=string,JSONPath=`.status.tenants`

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
