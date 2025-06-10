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
	"sort"

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
// +kubebuilder:rbac:groups="",resources=nodes;namespaces;endpoints;nodes/proxy,verbs=get;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;serviceaccounts;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets;daemonsets;replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete

func (r *RouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	baseManager := manager.NewBaseManager(r.Client, log.FromContext(ctx))

	tenant := &v1alpha1.Tenant{}
	baseManager.Info(fmt.Sprintf("getting tenant: %q", req.Name))

	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalTenantStatus := tenant.Status
	baseManager.Info(fmt.Sprintf("reconciling tenant: %q", tenant.Name))

	if err := handleOwnedResources(ctx, baseManager.GetTenantResourceManager(), tenant); err != nil {
		tenant.Status.State = state.StateFailed
		baseManager.Error(errors.WithStack(err), "failed to handle resources owned by tenant", "tenant", tenant.Name)
		if updateErr := r.updateStatus(ctx, tenant); updateErr != nil {
			baseManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
			return ctrl.Result{}, errors.Append(err, updateErr)
		}
	}

	if err := handleBridgeResources(ctx, baseManager.GetBridgeManager(), tenant); err != nil {
		tenant.Status.State = state.StateFailed
		baseManager.Error(errors.WithStack(err), "failed to handle bridge resources", "tenant", tenant.Name)
		if updateErr := r.updateStatus(ctx, tenant); updateErr != nil {
			baseManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
			return ctrl.Result{}, errors.Append(err, updateErr)
		}
	}

	tenant.Status.State = state.StateReady
	if !reflect.DeepEqual(originalTenantStatus, tenant.Status) {
		baseManager.Info("tenant status changed")
		if updateErr := r.updateStatus(ctx, tenant); updateErr != nil {
			baseManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
			return ctrl.Result{}, updateErr
		}

		return ctrl.Result{}, nil
	}

	baseManager.Info("tenant reconciliation complete", "tenant", tenant.Name)
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
			return err
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

func (r *RouteReconciler) updateStatus(ctx context.Context, obj client.Object) error {
	return r.Status().Update(ctx, obj)
}

func handleOwnedResources(ctx context.Context, tenantResManager *manager.TenantResourceManager, tenant *v1alpha1.Tenant) error {
	logsourceNamespacesForTenant, err := tenantResManager.GetLogsourceNamespaceNamesForTenant(ctx, tenant)
	if err != nil {
		tenant.Status.State = state.StateFailed
		tenantResManager.Error(errors.WithStack(err), "failed to get logsource namespaces for tenant", "tenant", tenant.Name)
		if updateErr := tenantResManager.Status().Update(ctx, tenant); updateErr != nil {
			tenantResManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
			return errors.Append(err, updateErr)
		}

		return err
	}
	slices.Sort(logsourceNamespacesForTenant)
	tenant.Status.LogSourceNamespaces = logsourceNamespacesForTenant

	subscriptionsForTenant, subscriptionUpdateList, err := tenantResManager.GetResourceOwnedByTenant(ctx, &v1alpha1.Subscription{}, tenant)
	if err != nil {
		tenantResManager.Error(errors.WithStack(err), "failed to get subscriptions for tenant", "tenant", tenant.Name)

		tenant.Status.State = state.StateFailed
		if updateErr := tenantResManager.Status().Update(ctx, tenant); updateErr != nil {
			tenantResManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
			return errors.Append(err, updateErr)
		}

		return err
	}

	// add all newly updated subscriptions here
	subscriptionsForTenant = append(subscriptionsForTenant, tenantResManager.UpdateResourcesForTenant(ctx, tenant.Name, subscriptionUpdateList)...)
	subscriptionsToDisown, err := tenantResManager.GetResourcesReferencingTenantButNotSelected(ctx, tenant, &v1alpha1.Subscription{}, subscriptionsForTenant)
	if err != nil {
		tenantResManager.Error(errors.WithStack(err), "failed to get subscriptions to disown", "tenant", tenant.Name)
	}
	tenantResManager.DisownResources(ctx, subscriptionsToDisown)

	subscriptionNames := manager.GetResourceNamesFromResource(subscriptionsForTenant)
	components.SortNamespacedNames(subscriptionNames)
	tenant.Status.Subscriptions = subscriptionNames

	// Check outputs for tenant
	outputsForTenant, outputUpdateList, err := tenantResManager.GetResourceOwnedByTenant(ctx, &v1alpha1.Output{}, tenant)
	if err != nil {
		tenantResManager.Error(errors.WithStack(err), "failed to get outputs for tenant", "tenant", tenant.Name)

		tenant.Status.State = state.StateFailed
		if updateErr := tenantResManager.Status().Update(ctx, tenant); updateErr != nil {
			tenantResManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
			return errors.Append(err, updateErr)
		}

		return err
	}

	// add all newly updated outputs here
	outputsForTenant = append(outputsForTenant, tenantResManager.UpdateResourcesForTenant(ctx, tenant.Name, outputUpdateList)...)
	outputsToDisown, err := tenantResManager.GetResourcesReferencingTenantButNotSelected(ctx, tenant, &v1alpha1.Output{}, outputsForTenant)
	if err != nil {
		tenantResManager.Error(errors.WithStack(err), "failed to get outputs to disown", "tenant", tenant.Name)
	}
	tenantResManager.DisownResources(ctx, outputsToDisown)

	// Check outputs for subscriptions
	realSubscriptionsForTenant, err := utils.GetConcreteTypeFromList[*v1alpha1.Subscription](utils.ToObject(subscriptionsForTenant))
	if err != nil {
		tenantResManager.Error(errors.WithStack(err), "failed to get concrete type from list", "tenant", tenant.Name)
	}

	for _, subscription := range realSubscriptionsForTenant {
		originalSubscriptionStatus := subscription.Status.DeepCopy()
		subscription.Status.Outputs = tenantResManager.ValidateSubscriptionOutputs(ctx, subscription)
		if !reflect.DeepEqual(originalSubscriptionStatus, subscription.Status) {
			if updateErr := tenantResManager.Status().Update(ctx, subscription); updateErr != nil {
				tenantResManager.Error(errors.WithStack(updateErr), "failed updating subscription status", "subscription", subscription.NamespacedName().String())
				return updateErr
			}
		}
	}

	return nil
}

func handleBridgeResources(ctx context.Context, bridgeManager *manager.BridgeManager, tenant *v1alpha1.Tenant) error {
	bridgesForTenant, err := bridgeManager.GetBridgesForTenant(ctx, tenant.Name)
	if err != nil {
		tenant.Status.State = state.StateFailed
		bridgeManager.Error(errors.WithStack(err), "failed to get bridges for tenant", "tenant", tenant.Name)
		if updateErr := bridgeManager.Status().Update(ctx, tenant); updateErr != nil {
			bridgeManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
			return errors.Append(err, updateErr)
		}

		return err
	}

	bridgesForTenantNames := manager.GetBridgeNamesFromBridges(bridgesForTenant)
	sort.Strings(bridgesForTenantNames)
	tenant.Status.ConnectedBridges = bridgesForTenantNames

	for _, bridge := range bridgesForTenant {
		if err := bridgeManager.CheckBridgeConnection(ctx, tenant.Name, &bridge); err != nil {
			tenant.Status.State = state.StateFailed
			bridgeManager.Error(errors.WithStack(err), "failed to check bridge connection", "bridge", bridge.Name)
			if updateErr := bridgeManager.Status().Update(ctx, tenant); updateErr != nil {
				bridgeManager.Error(errors.WithStack(updateErr), "failed updating tenant status", "tenant", tenant.Name)
				return errors.Append(err, updateErr)
			}

			return err
		}
	}

	return nil
}
