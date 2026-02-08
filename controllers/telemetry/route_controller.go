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

package telemetry

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"emperror.dev/errors"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/manager"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

// tenantReconcileStep represents a step in the reconciliation process for a Tenant resource.
// This solution is sufficient for the current use case, where we have a few steps to execute.
type tenantReconcileStep struct {
	name string
	fn   func() error
}

// RouteReconciler is responsible for reconciling Tenant resources
// It also watches for changes to Subscriptions, Outputs, and Namespaces
// to trigger the appropriate reconciliation logic when related resources change.
type RouteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors;tenants;subscriptions;outputs;bridges;,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/status;tenants/status;subscriptions/status;outputs/status;bridges/status;,verbs=get;update;patch
// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets;nodes;namespaces;endpoints;nodes/proxy,verbs=get;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;serviceaccounts;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets;daemonsets;replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete

func (r *RouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	baseManager := manager.NewBaseManager(r.Client, log.FromContext(ctx, "tenant", req.Name))

	tenant := &v1alpha1.Tenant{}
	baseManager.Info("getting tenant", "name", req.Name)

	if err := baseManager.Get(ctx, req.NamespacedName, tenant); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalTenantStatus := tenant.Status
	baseManager.Info("reconciling tenant", "name", tenant.Name)

	steps := []tenantReconcileStep{
		{
			name: "handle log source namespaces",
			fn: func() error {
				return handleLogSourceNamespaces(ctx, baseManager.GetTenantResourceManager(), tenant)
			},
		},
		{
			name: "handle owned resources",
			fn: func() error {
				return handleOwnedResources(ctx, baseManager.GetTenantResourceManager(), tenant)
			},
		},
		{
			name: "handle bridge resources",
			fn: func() error {
				return handleBridgeResources(ctx, baseManager.GetBridgeManager(), tenant)
			},
		},
	}
	for _, step := range steps {
		if err := step.fn(); err != nil {
			return r.handleTenantReconcileError(ctx, &baseManager, tenant, step.name, err)
		}
	}

	tenant.Status.State = state.StateReady
	tenant.ClearProblems()
	if !reflect.DeepEqual(originalTenantStatus, tenant.Status) {
		baseManager.Info("tenant status changed")
		if updateErr := r.Status().Update(ctx, tenant); updateErr != nil {
			return ctrl.Result{}, errors.Wrap(updateErr, "failed updating tenant status")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *RouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueueAllTenants := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, _ client.Object) []reconcile.Request {
		logger := log.FromContext(ctx)

		tenants := &v1alpha1.TenantList{}
		if err := r.List(ctx, tenants); err != nil {
			logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
			return nil
		}

		requests := make([]reconcile.Request, 0, len(tenants.Items))
		for _, tenant := range tenants.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: tenant.Name,
				},
			})
		}

		return requests
	})

	fieldIndexes := []struct {
		obj       client.Object
		field     string
		extractor func(client.Object) []string
	}{
		{
			obj:   &v1alpha1.Subscription{},
			field: model.StatusTenantReferenceField,
			extractor: func(rawObj client.Object) []string {
				subscription := rawObj.(*v1alpha1.Subscription)
				if subscription.Status.Tenant == "" {
					return nil
				}
				return []string{subscription.Status.Tenant}
			},
		},
		{
			obj:   &v1alpha1.Output{},
			field: model.StatusTenantReferenceField,
			extractor: func(rawObj client.Object) []string {
				output := rawObj.(*v1alpha1.Output)
				if output.Status.Tenant == "" {
					return nil
				}
				return []string{output.Status.Tenant}
			},
		},
		{
			obj:   &v1alpha1.Bridge{},
			field: model.BridgeSourceTenantReferenceField,
			extractor: func(rawObj client.Object) []string {
				bridge := rawObj.(*v1alpha1.Bridge)
				if bridge.Spec.SourceTenant == "" {
					return nil
				}
				return []string{bridge.Spec.SourceTenant}
			},
		},
		{
			obj:   &v1alpha1.Bridge{},
			field: model.BridgeTargetTenantReferenceField,
			extractor: func(rawObj client.Object) []string {
				bridge := rawObj.(*v1alpha1.Bridge)
				if bridge.Spec.TargetTenant == "" {
					return nil
				}
				return []string{bridge.Spec.TargetTenant}
			},
		},
		{
			obj:   &v1alpha1.Tenant{},
			field: model.TenantNameField,
			extractor: func(rawObj client.Object) []string {
				tenant := rawObj.(*v1alpha1.Tenant)
				return []string{tenant.Name}
			},
		},
	}
	indexer := mgr.GetFieldIndexer()
	for _, idx := range fieldIndexes {
		if err := indexer.IndexField(context.Background(), idx.obj, idx.field, idx.extractor); err != nil {
			return errors.Wrapf(err, "failed to index field %s for object %T", idx.field, idx.obj)
		}
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Tenant{})
	watchedResources := []client.Object{
		&v1alpha1.Subscription{},
		&v1alpha1.Output{},
		&v1alpha1.Bridge{},
		&apiv1.Namespace{},
	}
	for _, resource := range watchedResources {
		builder = builder.Watches(resource, enqueueAllTenants)
	}

	return builder.Complete(r)
}

// handleTenantReconcileError handles errors that occur during reconciliation steps
func (r *RouteReconciler) handleTenantReconcileError(ctx context.Context, baseManager *manager.BaseManager, tenant *v1alpha1.Tenant, stepName string, err error) (ctrl.Result, error) {
	wrappedErr := errors.Wrapf(err, "failed to %s for tenant %s", stepName, tenant.Name)

	tenant.AddProblem(wrappedErr.Error())
	tenant.Status.State = state.StateFailed

	baseManager.Error(wrappedErr, "tenant reconciliation step failed", "step", stepName)
	if updateErr := r.Status().Update(ctx, tenant); updateErr != nil {
		return ctrl.Result{}, errors.Combine(wrappedErr, errors.Wrap(updateErr, "failed updating tenant status"))
	}

	return ctrl.Result{}, wrappedErr
}

func handleLogSourceNamespaces(ctx context.Context, tenantResManager *manager.TenantResourceManager, tenant *v1alpha1.Tenant) error {
	logsourceNamespacesForTenant, err := tenantResManager.GetLogsourceNamespaceNamesForTenant(ctx, tenant)
	if err != nil {
		return errors.Wrapf(err, "failed to get logsource namespaces for tenant %s", tenant.Name)
	}
	slices.Sort(logsourceNamespacesForTenant)
	tenant.Status.LogSourceNamespaces = logsourceNamespacesForTenant

	return nil
}

func handleOwnedResources(ctx context.Context, tenantResManager *manager.TenantResourceManager, tenant *v1alpha1.Tenant) error {
	// Caching all outputs for the tenant to validate subscriptions against
	var allOutputsForTenant []model.ResourceOwnedByTenant

	// Process outputs first to ensure they have their tenant set before validating subscriptions
	tenantOwnedResources := []model.ResourceOwnedByTenant{
		&v1alpha1.Output{},
		&v1alpha1.Subscription{},
	}
	for _, resource := range tenantOwnedResources {
		resourcesForTenant, resourceUpdateList, err := tenantResManager.GetResourceOwnedByTenant(ctx, resource, tenant)
		if err != nil {
			return errors.Wrapf(err, "failed to get %T for tenant %s", resource, tenant.Name)
		}

		// Add all newly updated resources here
		resourcesForTenant = append(resourcesForTenant, tenantResManager.UpdateResourcesForTenant(ctx, tenant.Name, resourceUpdateList)...)

		resourcesToDisown, err := tenantResManager.GetResourcesReferencingTenantButNotSelected(ctx, tenant, resource, resourcesForTenant)
		if err != nil {
			return errors.Wrapf(err, "failed to get resources: %s to disown for tenant: %s", resource.GetObjectKind().GroupVersionKind().Kind, tenant.Name)
		}
		tenantResManager.DisownResources(ctx, resourcesToDisown)

		if _, ok := resource.(*v1alpha1.Output); ok {
			allOutputsForTenant = resourcesForTenant
		}

		if _, ok := resource.(*v1alpha1.Subscription); ok {
			subscriptionNames := manager.GetResourceNamesFromResource(resourcesForTenant)
			components.SortNamespacedNames(subscriptionNames)
			tenant.Status.Subscriptions = subscriptionNames

			if err := validateSubscriptionOutputs(ctx, tenantResManager, tenant, resourcesForTenant, allOutputsForTenant); err != nil {
				return errors.Wrapf(err, "failed to validate subscription outputs for tenant %s", tenant.Name)
			}
		}

		for _, res := range resourcesForTenant {
			if res.GetState() == state.StateFailed {
				return errors.Errorf("resource %s is in a failed state", res.GetName())
			}
		}
	}

	return nil
}

func validateSubscriptionOutputs(ctx context.Context, tenantResManager *manager.TenantResourceManager, tenant *v1alpha1.Tenant, subscriptionsForTenant []model.ResourceOwnedByTenant, outputsForTenant []model.ResourceOwnedByTenant) error {
	realSubscriptionsForTenant, err := utils.GetConcreteTypeFromList[*v1alpha1.Subscription](utils.ToObject(subscriptionsForTenant))
	if err != nil {
		return errors.Wrapf(err, "failed to get concrete type from list for subscriptions for tenant %s", tenant.Name)
	}

	realOutputsForTenant, err := utils.GetConcreteTypeFromList[*v1alpha1.Output](utils.ToObject(outputsForTenant))
	if err != nil {
		return errors.Wrapf(err, "failed to get concrete type from list for outputs for tenant %s", tenant.Name)
	}

	outputMap := make(map[v1alpha1.NamespacedName]*v1alpha1.Output)
	for _, output := range realOutputsForTenant {
		outputMap[output.NamespacedName()] = output
	}

	for _, subscription := range realSubscriptionsForTenant {
		originalSubscriptionStatus := subscription.Status.DeepCopy()
		validOutputs, invalidOutputs := tenantResManager.ValidateSubscriptionReferencedOutputsWithCache(ctx, subscription, outputMap)
		switch len(invalidOutputs) {
		case 0:
			subscription.Status.State = state.StateReady
			subscription.ClearProblems()
		default:
			subscription.Status.State = state.StateFailed
			subscription.AddProblem(fmt.Sprintf("invalid outputs referenced by subscription %s: %s", subscription.Name, strings.Join(invalidOutputs, ", ")))
		}
		components.SortNamespacedNames(validOutputs)
		subscription.Status.Outputs = validOutputs

		if !reflect.DeepEqual(originalSubscriptionStatus, subscription.Status) {
			if updateErr := tenantResManager.Status().Update(ctx, subscription); updateErr != nil {
				return errors.Wrapf(updateErr, "failed updating subscription status for %s", subscription.NamespacedName().String())
			}
		}
	}

	return nil
}

func handleBridgeResources(ctx context.Context, bridgeManager *manager.BridgeManager, tenant *v1alpha1.Tenant) error {
	bridgesForTenant, err := bridgeManager.GetBridgesForTenant(ctx, tenant.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to get bridges for tenant %s", tenant.Name)
	}

	bridgesForTenantNames := manager.GetBridgeNamesFromBridges(bridgesForTenant)
	slices.Sort(bridgesForTenantNames)
	tenant.Status.ConnectedBridges = bridgesForTenantNames

	for i := range bridgesForTenant {
		bridge := &bridgesForTenant[i]
		originalBridgeStatus := bridge.Status.DeepCopy()

		if err := bridgeManager.ValidateBridgeConnection(ctx, tenant.Name, bridge); err != nil {
			bridge.AddProblem(errors.Wrapf(err, "bridge %s validation failed", bridge.Name).Error())
			bridge.Status.State = state.StateFailed

			if updateErr := bridgeManager.Status().Update(ctx, bridge); updateErr != nil {
				return errors.Combine(err, errors.Wrap(updateErr, "failed updating bridge status"))
			}

			return errors.Wrapf(err, "failed to validate bridge %s for tenant %s", bridge.Name, tenant.Name)
		}

		bridge.Status.State = state.StateReady
		bridge.ClearProblems()

		if !reflect.DeepEqual(originalBridgeStatus, &bridge.Status) {
			if updateErr := bridgeManager.Status().Update(ctx, bridge); updateErr != nil {
				return errors.Wrapf(updateErr, "failed updating bridge %s status", bridge.Name)
			}
		}
	}

	return nil
}
