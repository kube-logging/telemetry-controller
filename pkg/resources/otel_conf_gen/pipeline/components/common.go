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
	"context"
	"fmt"
	"slices"
	"strings"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
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
	switch {
	case output.Spec.OTLPGRPC != nil:
		exporterName = fmt.Sprintf("otlp/%s_%s", output.Namespace, output.Name)
	case output.Spec.OTLPHTTP != nil:
		exporterName = fmt.Sprintf("otlphttp/%s_%s", output.Namespace, output.Name)
	case output.Spec.Fluentforward != nil:
		exporterName = fmt.Sprintf("fluentforwardexporter/%s_%s", output.Namespace, output.Name)
	case output.Spec.File != nil:
		exporterName = fmt.Sprintf("file/%s_%s", output.Namespace, output.Name)
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

	return nil, errors.Errorf("tenant for output %s not found", targetOutput)
}

func (r *ResourceRelations) GetTenantByName(tenantName string) (*v1alpha1.Tenant, error) {
	for _, tenant := range r.Tenants {
		if tenant.Name == tenantName {
			return &tenant, nil
		}
	}

	return nil, errors.Errorf("tenant %s not found", tenantName)
}

// QueryOutputSecret validates that the secret referenced by the output's
// authentication configuration exists and is retrievable.
func QueryOutputSecret(ctx context.Context, client client.Client, output *v1alpha1.Output) error {
	_, err := QueryOutputSecretWithData(ctx, client, output)
	return errors.Wrapf(err, "failed to query secret for output %s/%s", output.Namespace, output.Name)
}

// QueryOutputSecretWithData retrieves the secret referenced by the output's
// authentication configuration. Exactly one authentication method (BasicAuth
// or BearerAuth) with a valid SecretRef must be configured.
func QueryOutputSecretWithData(ctx context.Context, client client.Client, output *v1alpha1.Output) (*corev1.Secret, error) {
	auth := output.Spec.Authentication

	type authRef struct {
		secretRef *corev1.SecretReference
		name      string
	}

	var matches []authRef
	if auth.BasicAuth != nil && auth.BasicAuth.SecretRef != nil {
		matches = append(matches, authRef{auth.BasicAuth.SecretRef, "BasicAuth"})
	}
	if auth.BearerAuth != nil && auth.BearerAuth.SecretRef != nil {
		matches = append(matches, authRef{auth.BearerAuth.SecretRef, "BearerAuth"})
	}

	switch len(matches) {
	case 0:
		return nil, errors.New("no valid authentication method configured")
	case 1:
		// valid case, exactly one authentication method is configured
	default:
		return nil, errors.New("multiple authentication methods configured; only one is allowed")
	}

	ref := matches[0]
	namespacedName := types.NamespacedName{
		Namespace: ref.secretRef.Namespace,
		Name:      ref.secretRef.Name,
	}
	secret := &corev1.Secret{}
	if err := client.Get(ctx, namespacedName, secret); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve %s secret %s/%s",
			ref.name, namespacedName.Namespace, namespacedName.Name)
	}

	return secret, nil
}

func SortNamespacedNames(names []v1alpha1.NamespacedName) {
	slices.SortFunc(names, func(a, b v1alpha1.NamespacedName) int {
		return strings.Compare(a.String(), b.String())
	})
}
