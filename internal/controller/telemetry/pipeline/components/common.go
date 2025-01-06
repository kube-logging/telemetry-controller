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

package components

import (
	"fmt"
	"slices"
	"strings"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type ErrorMode string

const (
	ErrorModeIgnore    ErrorMode = "ignore"
	ErrorModeSilent    ErrorMode = "silent"
	ErrorModePropagate ErrorMode = "propagate"
)

type OutputWithSecretData struct {
	Output v1alpha1.Output
	Secret corev1.Secret
}

func GetExporterNameForOutput(output v1alpha1.Output) string {
	var exporterName string
	if output.Spec.OTLPGRPC != nil {
		exporterName = fmt.Sprintf("otlp/%s_%s", output.Namespace, output.Name)
	} else if output.Spec.OTLPHTTP != nil {
		exporterName = fmt.Sprintf("otlphttp/%s_%s", output.Namespace, output.Name)
	} else if output.Spec.Fluentforward != nil {
		exporterName = fmt.Sprintf("fluentforwardexporter/%s_%s", output.Namespace, output.Name)
	}

	return exporterName
}

type ResourceRelations struct {
	// These must only include resources that are selected by the collector, tenant labelselectors, and listed outputs in the subscriptions
	Tenants               []v1alpha1.Tenant
	Subscriptions         map[v1alpha1.NamespacedName]v1alpha1.Subscription
	Bridges               []v1alpha1.Bridge
	OutputsWithSecretData []OutputWithSecretData
	// Subscriptions map, where the key is the Tenants' name, value is a slice of subscriptions' namespaced name
	TenantSubscriptionMap map[string][]v1alpha1.NamespacedName
	SubscriptionOutputMap map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName
}

// FindOutputsForSubscription retrieves all outputs for a given subscription
func (r *ResourceRelations) FindOutputsForSubscription(subscription v1alpha1.NamespacedName) []v1alpha1.NamespacedName {
	outputs, exists := r.SubscriptionOutputMap[subscription]
	if !exists {
		return nil
	}

	return outputs
}

// IsOutputInSubscription checks if a specific output belongs to a subscription
func (r *ResourceRelations) IsOutputInSubscription(subscription, output v1alpha1.NamespacedName) bool {
	return slices.Contains(r.FindOutputsForSubscription(subscription), output)
}

// FindTenantForOutput determines which tenant owns a specific output
func (r *ResourceRelations) FindTenantForOutput(targetOutput v1alpha1.NamespacedName) (*v1alpha1.Tenant, error) {
	for tenantName, tenantSubscriptions := range r.TenantSubscriptionMap {
		// Check each of the tenant's subscriptions
		for _, subscription := range tenantSubscriptions {
			// Check if this output belongs to the current subscription
			if r.IsOutputInSubscription(subscription, targetOutput) {
				tenant, err := r.GetTenantByName(tenantName)
				if err != nil {
					return tenant, err
				}

				return tenant, nil
			}
		}
	}

	return nil, fmt.Errorf("tenant for output %s not found", targetOutput)
}

func (r *ResourceRelations) GetTenantByName(tenantName string) (*v1alpha1.Tenant, error) {
	for _, tenant := range r.Tenants {
		if tenant.Name == tenantName {
			return &tenant, nil
		}
	}

	return nil, fmt.Errorf("tenant %s not found", tenantName)
}

func SortNamespacedNames(names []v1alpha1.NamespacedName) {
	slices.SortFunc(names, func(a, b v1alpha1.NamespacedName) int {
		return strings.Compare(a.String(), b.String())
	})
}

func SortNamespacedNamesWithUID(names []v1alpha1.NamespacedNameWithUID) {
	slices.SortFunc(names, func(a, b v1alpha1.NamespacedNameWithUID) int {
		return strings.Compare(a.String(), b.String())
	})
}
