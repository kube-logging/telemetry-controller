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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
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
	logger := log.FromContext(ctx)

	tenant := &v1alpha1.Tenant{}

	logger.Info(fmt.Sprintf("getting tenant: %q", req.NamespacedName.Name))

	if err := r.Get(ctx, req.NamespacedName, tenant); client.IgnoreNotFound(err) != nil {
		logger.Error(errors.New("failed getting tenant, possible API server error"), "failed getting tenant, possible API server error",
			"tenant", req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	logger.Info(fmt.Sprintf("reconciling tenant: %q", tenant.Name))

	originalTenantStatus := tenant.Status

	subscriptionsForTenant, updateList, err := r.getSubscriptionsForTenant(ctx, tenant)
	if err != nil {

		tenant.Status.State = v1alpha1.StateFailed
		logger.Error(errors.WithStack(err), "failed to get subscriptions for tenant", "tenant", tenant.Name)
		if updateErr := r.Status().Update(ctx, tenant); updateErr != nil {
			logger.Error(errors.WithStack(updateErr), "failed update tenant status", "tenant", tenant.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// add all newly updated subscriptions here
	subscriptionsForTenant = append(subscriptionsForTenant, r.updateSubscriptionsForTenant(ctx, tenant.Name, updateList)...)

	subscriptionsToDisown := r.getSubscriptionsReferencingTenantButNotSelected(ctx, tenant, subscriptionsForTenant)

	r.disownSubscriptions(ctx, subscriptionsToDisown)

	subscriptionNames := getSubscriptionNamesFromSubscription(subscriptionsForTenant)

	cmp := func(a, b v1alpha1.NamespacedName) int {
		return strings.Compare(a.String(), b.String())
	}

	slices.SortFunc(subscriptionNames, cmp)
	tenant.Status.Subscriptions = subscriptionNames

	for _, subscription := range subscriptionsForTenant {
		originalSubscriptionStatus := subscription.Status.DeepCopy()
		validOutputs := []v1alpha1.NamespacedName{}
		for _, outputRef := range subscription.Spec.Outputs {
			checkedOutput := &v1alpha1.Output{}
			if err := r.Client.Get(ctx, types.NamespacedName(outputRef), checkedOutput); err != nil {
				logger.Error(err, "referred output invalid", "output", outputRef.String())
			} else {
				validOutputs = append(validOutputs, outputRef)
			}

		}
		subscription.Status.Outputs = validOutputs

		if !reflect.DeepEqual(originalSubscriptionStatus, subscription.Status) {
			if updateErr := r.Status().Update(ctx, &subscription); updateErr != nil {
				logger.Error(errors.WithStack(updateErr), "failed update subscription status", "subscription", subscription.NamespacedName().String())
				return ctrl.Result{}, err
			}
		}
	}

	logsourceNamespacesForTenant, err := r.getLogsourceNamespaceNamesForTenant(ctx, tenant)
	if err != nil {
		tenant.Status.State = v1alpha1.StateFailed
		logger.Error(errors.WithStack(err), "failed to get logsource namespaces for tenant", "tenant", tenant.Name)
		if updateErr := r.Status().Update(ctx, tenant); updateErr != nil {
			logger.Error(errors.WithStack(updateErr), "failed update tenant status", "tenant", tenant.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	slices.Sort(logsourceNamespacesForTenant)
	tenant.Status.LogSourceNamespaces = logsourceNamespacesForTenant

	tenant.Status.State = v1alpha1.StateReady

	if !reflect.DeepEqual(originalTenantStatus, tenant.Status) {
		logger.Info("tenant status changed")
		if err := r.Status().Update(ctx, tenant); err != nil {
			logger.Error(errors.New("failed update tenant status"), "failed update tenant status", "tenant", tenant.Name)
			return ctrl.Result{}, err
		}
	}

	logger.Info("tenant reconciliation complete", "tenant", tenant.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.Subscription{}, tenantReferenceField, func(rawObj client.Object) []string {
		subscription := rawObj.(*v1alpha1.Subscription)
		if subscription.Status.Tenant == "" {
			return nil
		}
		return []string{subscription.Status.Tenant}
	}); err != nil {
		return err
	}
	addTenantRequest := func(requests []reconcile.Request, tenant string) []reconcile.Request {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: tenant,
			},
		})
		return requests
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Tenant{}).
		Watches(&v1alpha1.Subscription{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) (requests []reconcile.Request) {
			logger := log.FromContext(ctx)

			tenants := &v1alpha1.TenantList{}
			err := r.List(ctx, tenants)
			if err != nil {
				logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
				return
			}

			for _, tenant := range tenants.Items {
				requests = addTenantRequest(requests, tenant.Name)
			}

			return
		})).
		Watches(&v1alpha1.Output{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) (requests []reconcile.Request) {
			logger := log.FromContext(ctx)

			tenants := &v1alpha1.TenantList{}
			err := r.List(ctx, tenants)
			if err != nil {
				logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
				return
			}

			for _, tenant := range tenants.Items {
				requests = addTenantRequest(requests, tenant.Name)
			}

			return
		})).
		Watches(&apiv1.Namespace{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) (requests []reconcile.Request) {
			logger := log.FromContext(ctx)

			tenants := &v1alpha1.TenantList{}
			err := r.List(ctx, tenants)
			if err != nil {
				logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
				return
			}

			for _, tenant := range tenants.Items {
				requests = addTenantRequest(requests, tenant.Name)
			}

			return
		})).
		Complete(r)
}

func (r *RouteReconciler) getSubscriptionsForTenant(ctx context.Context, tenant *v1alpha1.Tenant) (ownedList []v1alpha1.Subscription, updateList []v1alpha1.Subscription, err error) {
	logger := log.FromContext(ctx)

	namespaces, err := r.getNamespacesForSelectorSlice(ctx, tenant.Spec.SubscriptionNamespaceSelectors)

	if err != nil {
		return nil, nil, err
	}

	var selectedSubscriptions []v1alpha1.Subscription

	for _, ns := range namespaces {
		var subscriptionsForNS v1alpha1.SubscriptionList
		listOpts := &client.ListOptions{
			Namespace: ns.Name,
		}

		if err := r.List(ctx, &subscriptionsForNS, listOpts); client.IgnoreNotFound(err) != nil {
			return nil, nil, err
		}

		selectedSubscriptions = append(selectedSubscriptions, subscriptionsForNS.Items...)
	}

	for _, subscription := range selectedSubscriptions {
		if subscription.Status.Tenant != "" && subscription.Status.Tenant != tenant.Name {
			logger.Error(errors.Errorf("subscription (%s) is owned by another tenant (%s), skipping reconciliation for this tenant (%s)", subscription.Name, subscription.Status.Tenant, tenant.Name),
				"make sure to remove subscription from the previous tenant before adopting to new tenant")
			continue
		}

		if subscription.Status.Tenant == "" {
			updateList = append(updateList, subscription)
		} else {
			ownedList = append(ownedList, subscription)
		}
	}

	return
}
func (r *RouteReconciler) getNamespacesForSelectorSlice(ctx context.Context, labelSelectors []metav1.LabelSelector) ([]apiv1.Namespace, error) {

	var namespaces []apiv1.Namespace

	for _, ls := range labelSelectors {
		var namespacesForSelector apiv1.NamespaceList

		selector, err := metav1.LabelSelectorAsSelector(&ls)

		if err != nil {
			return nil, err
		}

		listOpts := &client.ListOptions{
			LabelSelector: selector,
		}

		if err := r.List(ctx, &namespacesForSelector, listOpts); client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		namespaces = append(namespaces, namespacesForSelector.Items...)
	}

	namespaces = normalizeNamespaceSlice(namespaces)

	return namespaces, nil
}

// disownSubscriptions fails internally by logging errors individually
// this is by design so that we don't fail the whole reconciliation when a single subscription update fails
func (r *RouteReconciler) disownSubscriptions(ctx context.Context, subscriptionsToDisown []v1alpha1.Subscription) {
	logger := log.FromContext(ctx)
	for _, subscription := range subscriptionsToDisown {
		subscription.Status.Tenant = ""
		err := r.Client.Status().Update(ctx, &subscription)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to detach subscription %s/%s from collector", subscription.Namespace, subscription.Name))
		} else {
			logger.Info("disowning subscription", "subscription", fmt.Sprintf("%s/%s", subscription.Namespace, subscription.Name))
		}
	}
}

// updateSubscriptionsForTenant fails internally and logs failures individually
// this is by design in order to avoid blocking the whole reconciliation in case we cannot update a single subscription
func (r *RouteReconciler) updateSubscriptionsForTenant(ctx context.Context, tenantName string, subscriptions []v1alpha1.Subscription) (updatedSubscriptions []v1alpha1.Subscription) {
	logger := log.FromContext(ctx, "tenant", tenantName)
	for _, subscription := range subscriptions {
		subscription.Status.Tenant = tenantName

		logger.Info("updating subscription status for tenant ownership")
		err := r.Status().Update(ctx, &subscription)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to set subscription (%s/%s) -> tenant (%s) reference", subscription.Namespace, subscription.Name, tenantName))
		} else {
			updatedSubscriptions = append(updatedSubscriptions, subscription)
		}
	}
	return
}

func (r *RouteReconciler) getSubscriptionsReferencingTenantButNotSelected(ctx context.Context, tenant *v1alpha1.Tenant, selectedSubscriptions []v1alpha1.Subscription) []v1alpha1.Subscription {
	logger := log.FromContext(ctx)
	var subscriptionsReferencing v1alpha1.SubscriptionList
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(tenantReferenceField, tenant.Name),
	}

	if err := r.Client.List(ctx, &subscriptionsReferencing, listOpts); client.IgnoreNotFound(err) != nil {
		logger.Error(err, "failed to list subscriptions that need to be detached from tenant")
		return nil
	}

	var subscriptionsToDisown []v1alpha1.Subscription

	for _, subscriptionReferencing := range subscriptionsReferencing.Items {

		idx := slices.IndexFunc(selectedSubscriptions, func(selected v1alpha1.Subscription) bool {
			return reflect.DeepEqual(subscriptionReferencing.NamespacedName(), selected.NamespacedName())
		})

		if idx == -1 {
			subscriptionsToDisown = append(subscriptionsToDisown, subscriptionReferencing)
		}

	}

	return subscriptionsToDisown

}

func (r *RouteReconciler) getLogsourceNamespaceNamesForTenant(ctx context.Context, tentant *v1alpha1.Tenant) ([]string, error) {
	namespaces, err := r.getNamespacesForSelectorSlice(ctx, tentant.Spec.LogSourceNamespaceSelectors)
	if err != nil {
		return nil, err
	}

	namespaceNames := make([]string, len(namespaces))

	for i, namespace := range namespaces {
		namespaceNames[i] = namespace.Name
	}

	return namespaceNames, nil

}

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

func getSubscriptionNamesFromSubscription(subscriptions []v1alpha1.Subscription) []v1alpha1.NamespacedName {
	subscriptionNames := make([]v1alpha1.NamespacedName, len(subscriptions))
	for i, subscription := range subscriptions {
		subscriptionNames[i] = subscription.NamespacedName()
	}

	return subscriptionNames
}
