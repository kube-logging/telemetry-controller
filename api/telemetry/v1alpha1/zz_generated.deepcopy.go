//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/cisco-open/operator-tools/pkg/typeoverride"
	"go.opentelemetry.io/collector/config/configopaque"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	timex "time"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackOffConfig) DeepCopyInto(out *BackOffConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackOffConfig.
func (in *BackOffConfig) DeepCopy() *BackOffConfig {
	if in == nil {
		return nil
	}
	out := new(BackOffConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Collector) DeepCopyInto(out *Collector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Collector.
func (in *Collector) DeepCopy() *Collector {
	if in == nil {
		return nil
	}
	out := new(Collector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Collector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CollectorList) DeepCopyInto(out *CollectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Collector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CollectorList.
func (in *CollectorList) DeepCopy() *CollectorList {
	if in == nil {
		return nil
	}
	out := new(CollectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CollectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CollectorSpec) DeepCopyInto(out *CollectorSpec) {
	*out = *in
	in.TenantSelector.DeepCopyInto(&out.TenantSelector)
	if in.MemoryLimiter != nil {
		in, out := &in.MemoryLimiter, &out.MemoryLimiter
		*out = new(MemoryLimiter)
		**out = **in
	}
	if in.DaemonSetOverrides != nil {
		in, out := &in.DaemonSetOverrides, &out.DaemonSetOverrides
		*out = new(typeoverride.DaemonSet)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CollectorSpec.
func (in *CollectorSpec) DeepCopy() *CollectorSpec {
	if in == nil {
		return nil
	}
	out := new(CollectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CollectorStatus) DeepCopyInto(out *CollectorStatus) {
	*out = *in
	if in.Tenants != nil {
		in, out := &in.Tenants, &out.Tenants
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CollectorStatus.
func (in *CollectorStatus) DeepCopy() *CollectorStatus {
	if in == nil {
		return nil
	}
	out := new(CollectorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Fluentforward) DeepCopyInto(out *Fluentforward) {
	*out = *in
	out.TCPClientSettings = in.TCPClientSettings
	if in.DefaultLabelsEnabled != nil {
		in, out := &in.DefaultLabelsEnabled, &out.DefaultLabelsEnabled
		*out = make(map[string]bool, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.QueueConfig = in.QueueConfig
	out.RetryConfig = in.RetryConfig
	if in.Kubernetes != nil {
		in, out := &in.Kubernetes, &out.Kubernetes
		*out = new(KubernetesMetadata)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Fluentforward.
func (in *Fluentforward) DeepCopy() *Fluentforward {
	if in == nil {
		return nil
	}
	out := new(Fluentforward)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GRPCClientConfig) DeepCopyInto(out *GRPCClientConfig) {
	*out = *in
	out.TLSSetting = in.TLSSetting
	if in.Keepalive != nil {
		in, out := &in.Keepalive, &out.Keepalive
		*out = new(KeepaliveClientConfig)
		**out = **in
	}
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GRPCClientConfig.
func (in *GRPCClientConfig) DeepCopy() *GRPCClientConfig {
	if in == nil {
		return nil
	}
	out := new(GRPCClientConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPClientConfig) DeepCopyInto(out *HTTPClientConfig) {
	*out = *in
	out.TLSSetting = in.TLSSetting
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make(map[string]configopaque.String, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.MaxIdleConns != nil {
		in, out := &in.MaxIdleConns, &out.MaxIdleConns
		*out = new(int)
		**out = **in
	}
	if in.MaxIdleConnsPerHost != nil {
		in, out := &in.MaxIdleConnsPerHost, &out.MaxIdleConnsPerHost
		*out = new(int)
		**out = **in
	}
	if in.MaxConnsPerHost != nil {
		in, out := &in.MaxConnsPerHost, &out.MaxConnsPerHost
		*out = new(int)
		**out = **in
	}
	if in.IdleConnTimeout != nil {
		in, out := &in.IdleConnTimeout, &out.IdleConnTimeout
		*out = new(timex.Duration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPClientConfig.
func (in *HTTPClientConfig) DeepCopy() *HTTPClientConfig {
	if in == nil {
		return nil
	}
	out := new(HTTPClientConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeepaliveClientConfig) DeepCopyInto(out *KeepaliveClientConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeepaliveClientConfig.
func (in *KeepaliveClientConfig) DeepCopy() *KeepaliveClientConfig {
	if in == nil {
		return nil
	}
	out := new(KeepaliveClientConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesMetadata) DeepCopyInto(out *KubernetesMetadata) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesMetadata.
func (in *KubernetesMetadata) DeepCopy() *KubernetesMetadata {
	if in == nil {
		return nil
	}
	out := new(KubernetesMetadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Loki) DeepCopyInto(out *Loki) {
	*out = *in
	out.QueueConfig = in.QueueConfig
	out.RetryConfig = in.RetryConfig
	in.HTTPClientConfig.DeepCopyInto(&out.HTTPClientConfig)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Loki.
func (in *Loki) DeepCopy() *Loki {
	if in == nil {
		return nil
	}
	out := new(Loki)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MemoryLimiter) DeepCopyInto(out *MemoryLimiter) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MemoryLimiter.
func (in *MemoryLimiter) DeepCopy() *MemoryLimiter {
	if in == nil {
		return nil
	}
	out := new(MemoryLimiter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespacedName) DeepCopyInto(out *NamespacedName) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespacedName.
func (in *NamespacedName) DeepCopy() *NamespacedName {
	if in == nil {
		return nil
	}
	out := new(NamespacedName)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OTLP) DeepCopyInto(out *OTLP) {
	*out = *in
	out.QueueConfig = in.QueueConfig
	out.RetryConfig = in.RetryConfig
	out.TimeoutSettings = in.TimeoutSettings
	in.GRPCClientConfig.DeepCopyInto(&out.GRPCClientConfig)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OTLP.
func (in *OTLP) DeepCopy() *OTLP {
	if in == nil {
		return nil
	}
	out := new(OTLP)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OtelColConfig) DeepCopyInto(out *OtelColConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OtelColConfig.
func (in *OtelColConfig) DeepCopy() *OtelColConfig {
	if in == nil {
		return nil
	}
	out := new(OtelColConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OtelOutput) DeepCopyInto(out *OtelOutput) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OtelOutput.
func (in *OtelOutput) DeepCopy() *OtelOutput {
	if in == nil {
		return nil
	}
	out := new(OtelOutput)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OtelOutput) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OtelOutputList) DeepCopyInto(out *OtelOutputList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]OtelOutput, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OtelOutputList.
func (in *OtelOutputList) DeepCopy() *OtelOutputList {
	if in == nil {
		return nil
	}
	out := new(OtelOutputList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *OtelOutputList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OtelOutputSpec) DeepCopyInto(out *OtelOutputSpec) {
	*out = *in
	if in.OTLP != nil {
		in, out := &in.OTLP, &out.OTLP
		*out = new(OTLP)
		(*in).DeepCopyInto(*out)
	}
	if in.Loki != nil {
		in, out := &in.Loki, &out.Loki
		*out = new(Loki)
		(*in).DeepCopyInto(*out)
	}
	if in.Fluentforward != nil {
		in, out := &in.Fluentforward, &out.Fluentforward
		*out = new(Fluentforward)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OtelOutputSpec.
func (in *OtelOutputSpec) DeepCopy() *OtelOutputSpec {
	if in == nil {
		return nil
	}
	out := new(OtelOutputSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OtelOutputStatus) DeepCopyInto(out *OtelOutputStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OtelOutputStatus.
func (in *OtelOutputStatus) DeepCopy() *OtelOutputStatus {
	if in == nil {
		return nil
	}
	out := new(OtelOutputStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QueueSettings) DeepCopyInto(out *QueueSettings) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueueSettings.
func (in *QueueSettings) DeepCopy() *QueueSettings {
	if in == nil {
		return nil
	}
	out := new(QueueSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Subscription) DeepCopyInto(out *Subscription) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Subscription.
func (in *Subscription) DeepCopy() *Subscription {
	if in == nil {
		return nil
	}
	out := new(Subscription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Subscription) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SubscriptionList) DeepCopyInto(out *SubscriptionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Subscription, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SubscriptionList.
func (in *SubscriptionList) DeepCopy() *SubscriptionList {
	if in == nil {
		return nil
	}
	out := new(SubscriptionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SubscriptionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SubscriptionSpec) DeepCopyInto(out *SubscriptionSpec) {
	*out = *in
	if in.Outputs != nil {
		in, out := &in.Outputs, &out.Outputs
		*out = make([]NamespacedName, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SubscriptionSpec.
func (in *SubscriptionSpec) DeepCopy() *SubscriptionSpec {
	if in == nil {
		return nil
	}
	out := new(SubscriptionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SubscriptionStatus) DeepCopyInto(out *SubscriptionStatus) {
	*out = *in
	if in.Outputs != nil {
		in, out := &in.Outputs, &out.Outputs
		*out = make([]NamespacedName, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SubscriptionStatus.
func (in *SubscriptionStatus) DeepCopy() *SubscriptionStatus {
	if in == nil {
		return nil
	}
	out := new(SubscriptionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TCPClientSettings) DeepCopyInto(out *TCPClientSettings) {
	*out = *in
	out.TLSSetting = in.TLSSetting
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TCPClientSettings.
func (in *TCPClientSettings) DeepCopy() *TCPClientSettings {
	if in == nil {
		return nil
	}
	out := new(TCPClientSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TLSClientSetting) DeepCopyInto(out *TLSClientSetting) {
	*out = *in
	out.TLSSetting = in.TLSSetting
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TLSClientSetting.
func (in *TLSClientSetting) DeepCopy() *TLSClientSetting {
	if in == nil {
		return nil
	}
	out := new(TLSClientSetting)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TLSSetting) DeepCopyInto(out *TLSSetting) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TLSSetting.
func (in *TLSSetting) DeepCopy() *TLSSetting {
	if in == nil {
		return nil
	}
	out := new(TLSSetting)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Tenant) DeepCopyInto(out *Tenant) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Tenant.
func (in *Tenant) DeepCopy() *Tenant {
	if in == nil {
		return nil
	}
	out := new(Tenant)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Tenant) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TenantList) DeepCopyInto(out *TenantList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Tenant, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TenantList.
func (in *TenantList) DeepCopy() *TenantList {
	if in == nil {
		return nil
	}
	out := new(TenantList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TenantList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TenantSpec) DeepCopyInto(out *TenantSpec) {
	*out = *in
	if in.SubscriptionNamespaceSelectors != nil {
		in, out := &in.SubscriptionNamespaceSelectors, &out.SubscriptionNamespaceSelectors
		*out = make([]v1.LabelSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.LogSourceNamespaceSelectors != nil {
		in, out := &in.LogSourceNamespaceSelectors, &out.LogSourceNamespaceSelectors
		*out = make([]v1.LabelSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TenantSpec.
func (in *TenantSpec) DeepCopy() *TenantSpec {
	if in == nil {
		return nil
	}
	out := new(TenantSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TenantStatus) DeepCopyInto(out *TenantStatus) {
	*out = *in
	if in.Subscriptions != nil {
		in, out := &in.Subscriptions, &out.Subscriptions
		*out = make([]NamespacedName, len(*in))
		copy(*out, *in)
	}
	if in.LogSourceNamespaces != nil {
		in, out := &in.LogSourceNamespaces, &out.LogSourceNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TenantStatus.
func (in *TenantStatus) DeepCopy() *TenantStatus {
	if in == nil {
		return nil
	}
	out := new(TenantStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TimeoutSettings) DeepCopyInto(out *TimeoutSettings) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TimeoutSettings.
func (in *TimeoutSettings) DeepCopy() *TimeoutSettings {
	if in == nil {
		return nil
	}
	out := new(TimeoutSettings)
	in.DeepCopyInto(out)
	return out
}
