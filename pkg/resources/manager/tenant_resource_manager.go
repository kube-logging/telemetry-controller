// Copyright © 2025 Kube logging authors
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

package manager

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"emperror.dev/errors"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"
)

// TenantResourceManager is a manager for resources owned by tenants: Subscriptions and Outputs
type TenantResourceManager struct {
	BaseManager
}

// GetLogsourceNamespaceNamesForTenant returns the namespaces that match the log source namespace selectors of the tenant
func (t *TenantResourceManager) GetLogsourceNamespaceNamesForTenant(ctx context.Context, tenant *v1alpha1.Tenant) ([]string, error) {
	namespaces, err := t.getNamespacesForSelectorSlice(ctx, tenant.Spec.LogSourceNamespaceSelectors)
	if err != nil {
		return nil, err
	}

	namespaceNames := make([]string, len(namespaces))
	for i, namespace := range namespaces {
		namespaceNames[i] = namespace.Name
	}

	return namespaceNames, nil
}

// GetResourceOwnedByTenant returns a list of resources owned by the tenant and a list of resources that need to be updated
func (t *TenantResourceManager) GetResourceOwnedByTenant(ctx context.Context, resource model.ResourceOwnedByTenant, tenant *v1alpha1.Tenant) (ownedList []model.ResourceOwnedByTenant, updateList []model.ResourceOwnedByTenant, err error) {
	if resource == nil {
		return nil, nil, fmt.Errorf("resource cannot be nil")
	}

	namespaces, err := t.getNamespacesForSelectorSlice(ctx, tenant.Spec.SubscriptionNamespaceSelectors)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get namespaces: %w", err)
	}

	// Create the appropriate list type based on the resource type
	var resourceList model.ResourceList
	switch resource.(type) {
	case *v1alpha1.Subscription:
		resourceList = &v1alpha1.SubscriptionList{}
	case *v1alpha1.Output:
		resourceList = &v1alpha1.OutputList{}
	default:
		return nil, nil, fmt.Errorf("unsupported resource type: %T", resource)
	}

	// Collect resources from all namespaces
	var allResources []model.ResourceOwnedByTenant
	for _, ns := range namespaces {
		listOpts := &client.ListOptions{
			Namespace: ns.Name,
		}

		if err := t.List(ctx, resourceList, listOpts); err != nil {
			if !apierrors.IsNotFound(err) {
				return nil, nil, fmt.Errorf("failed to list resources in namespace %s: %w", ns.Name, err)
			}
			continue
		}

		allResources = append(allResources, resourceList.GetItems()...)
	}

	// Categorize resources
	for _, res := range allResources {
		currentTenant := res.GetTenant()
		if currentTenant != "" && currentTenant != tenant.Name {
			t.Error(
				fmt.Errorf("resource is owned by another tenant"),
				"skipping reconciliation",
				"current_tenant", currentTenant,
				"desired_tenant", tenant.Name,
				"action_required", "remove resource from previous tenant before adopting to new tenant",
			)
			continue
		}

		if currentTenant == "" {
			updateList = append(updateList, res)
		} else {
			ownedList = append(ownedList, res)
		}
	}

	return ownedList, updateList, nil
}

// updateResourcesForTenant fails internally and logs failures individually
// this is by design in order to avoid blocking the whole reconciliation in case we cannot update a single resource
func (t *TenantResourceManager) UpdateResourcesForTenant(ctx context.Context, tenantName string, resources []model.ResourceOwnedByTenant) (updatedResources []model.ResourceOwnedByTenant) {
	for _, res := range resources {
		res.SetTenant(tenantName)
		t.Info(fmt.Sprintf("updating resource (%s/%s) -> tenant (%s) reference", res.GetNamespace(), res.GetName(), tenantName))

		if updateErr := t.Status().Update(ctx, res); updateErr != nil {
			res.SetState(state.StateFailed)
			t.Error(updateErr, fmt.Sprintf("failed to update resource (%s/%s) -> tenant (%s) reference", res.GetNamespace(), res.GetName(), tenantName))
		} else {
			updatedResources = append(updatedResources, res)
		}

		res.SetState(state.StateReady)
	}

	return
}

// GetResourcesReferencingTenantButNotSelected returns a list of resources that are
// referencing the tenant in their status but are not selected by the tenant directly
func (t *TenantResourceManager) GetResourcesReferencingTenantButNotSelected(ctx context.Context, tenant *v1alpha1.Tenant, resource model.ResourceOwnedByTenant, selectedResources []model.ResourceOwnedByTenant) ([]model.ResourceOwnedByTenant, error) {
	// Create the appropriate list type based on the resource type
	var resourceList model.ResourceList
	switch resource.(type) {
	case *v1alpha1.Subscription:
		resourceList = &v1alpha1.SubscriptionList{}
	case *v1alpha1.Output:
		resourceList = &v1alpha1.OutputList{}
	default:
		return nil, fmt.Errorf("unsupported resource type: %T", resource)
	}

	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(model.StatusTenantReferenceField, tenant.Name),
	}
	if err := t.List(ctx, resourceList, listOpts); client.IgnoreNotFound(err) != nil {
		t.Error(err, "failed to list resources that need to be detached from tenant")
		return nil, err
	}

	var resourcesToDisown []model.ResourceOwnedByTenant
	for _, resourceReferencing := range resourceList.GetItems() {
		idx := slices.IndexFunc(selectedResources, func(selected model.ResourceOwnedByTenant) bool {
			return reflect.DeepEqual(resourceReferencing.GetName(), selected.GetName()) && reflect.DeepEqual(resourceReferencing.GetNamespace(), selected.GetNamespace())
		})
		if idx == -1 {
			resourcesToDisown = append(resourcesToDisown, resourceReferencing)
		}
	}

	return resourcesToDisown, nil
}

// disownResources fails internally by logging errors individually
// this is by design so that we don't fail the whole reconciliation when a single resource cannot be disowned
func (t *TenantResourceManager) DisownResources(ctx context.Context, resourceToDisown []model.ResourceOwnedByTenant) {
	for _, res := range resourceToDisown {
		tenantName := res.GetTenant()
		res.SetTenant("")
		if updateErr := t.Status().Update(ctx, res); updateErr != nil {
			res.SetState(state.StateFailed)
			t.Error(updateErr, fmt.Sprintf("failed to detach subscription %s/%s from tenant: %s", res.GetNamespace(), res.GetName(), tenantName))
		} else {
			t.Info(fmt.Sprintf("disowning resource (%s/%s)", res.GetNamespace(), res.GetName()))
		}
	}
}

// ValidateSubscriptionOutputs validates the output references of a subscription
func (t *TenantResourceManager) ValidateSubscriptionOutputs(ctx context.Context, subscription *v1alpha1.Subscription) []v1alpha1.NamespacedName {
	validOutputs := []v1alpha1.NamespacedName{}
	invalidOutputs := []v1alpha1.NamespacedName{}

	for _, outputRef := range subscription.Spec.Outputs {
		checkedOutput := &v1alpha1.Output{}
		if err := t.Get(ctx, types.NamespacedName(outputRef), checkedOutput); err != nil {
			t.Error(err, "referred output invalid", "output", outputRef.String())

			invalidOutputs = append(invalidOutputs, outputRef)
			continue
		}

		// Ensure the output belongs to the same tenant
		if checkedOutput.Status.Tenant != subscription.Status.Tenant {
			t.Error(errors.New("output and subscription tenants mismatch"),
				"output and subscription tenants mismatch",
				"output", checkedOutput.NamespacedName().String(),
				"output's tenant", checkedOutput.Status.Tenant,
				"subscription", subscription.NamespacedName().String(),
				"subscription's tenant", subscription.Status.Tenant)

			invalidOutputs = append(invalidOutputs, outputRef)
			continue
		}

		// update the output state if validation was successful
		checkedOutput.SetState(state.StateReady)
		if updateErr := t.Status().Update(ctx, checkedOutput); updateErr != nil {
			checkedOutput.SetState(state.StateFailed)
			t.Error(updateErr, fmt.Sprintf("failed to update output (%s/%s) state", checkedOutput.GetNamespace(), checkedOutput.GetName()))
		}

		validOutputs = append(validOutputs, outputRef)
	}

	if len(invalidOutputs) > 0 {
		t.Error(errors.New("some outputs are invalid"), "some outputs are invalid", "invalidOutputs", invalidOutputs, "subscription", subscription.NamespacedName().String())
	}

	return validOutputs
}

// getNamespacesForSelectorSlice returns a list of namespaces that match the given label selectors
func (t *TenantResourceManager) getNamespacesForSelectorSlice(ctx context.Context, labelSelectors []metav1.LabelSelector) ([]apiv1.Namespace, error) {
	var namespaces []apiv1.Namespace
	for _, ls := range labelSelectors {
		selector, err := metav1.LabelSelectorAsSelector(&ls)
		if err != nil {
			return nil, err
		}

		var namespacesForSelector apiv1.NamespaceList
		listOpts := &client.ListOptions{
			LabelSelector: selector,
		}
		if err := t.List(ctx, &namespacesForSelector, listOpts); client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		namespaces = append(namespaces, namespacesForSelector.Items...)
	}

	namespaces = normalizeNamespaceSlice(namespaces)

	return namespaces, nil
}

func GetResourceNamesFromResource(resources []model.ResourceOwnedByTenant) []v1alpha1.NamespacedName {
	resourceNames := make([]v1alpha1.NamespacedName, len(resources))
	for i, resource := range resources {
		resourceNames[i] = v1alpha1.NamespacedName{
			Name:      resource.GetName(),
			Namespace: resource.GetNamespace(),
		}
	}

	return resourceNames
}

// normalizeNamespaceSlice removes duplicates from the input list and sorts it
func normalizeNamespaceSlice(inputList []apiv1.Namespace) []apiv1.Namespace {
	allKeys := make(map[string]bool)
	uniqueList := []apiv1.Namespace{}
	for _, item := range inputList {
		if _, value := allKeys[item.Name]; !value {
			allKeys[item.Name] = true
			uniqueList = append(uniqueList, item)
		}
	}

	cmp := func(a, b apiv1.Namespace) int {
		return strings.Compare(a.Name, b.Name)
	}
	slices.SortFunc(uniqueList, cmp)

	return uniqueList
}
