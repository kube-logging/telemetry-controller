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

package logging

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/subscription-operator/api/logging/v1alpha1"
	loggingv1alpha1 "github.com/kube-logging/subscription-operator/api/logging/v1alpha1"
)

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=logging.kube-logging.dev,resources=collectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=logging.kube-logging.dev,resources=collectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=logging.kube-logging.dev,resources=collectors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Collector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *CollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	collector := &v1alpha1.Collector{}

	logger.Info("Reconciling collector")

	if err := r.Get(ctx, req.NamespacedName, collector); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	tenantSubscriptionMap := make(map[string][]loggingv1alpha1.Subscription)
	// TODO deduplicate tenant list
	tenants, err := r.getTenantsMatchingSelector(ctx, collector.Spec.TenantSelectors)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, tenant := range tenants {
		tenantSubscriptionMap[tenant.Name], err = r.getSubscriptionsForTenant(ctx, &tenant)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&loggingv1alpha1.Collector{}).
		Complete(r)
}

func (r *CollectorReconciler) getTenantsMatchingSelector(ctx context.Context, labelSelectorsList []metav1.LabelSelector) ([]loggingv1alpha1.Tenant, error) {
	var selectors []labels.Selector

	for _, labelSelector := range labelSelectorsList {
		selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, selector)
	}

	var tenants []loggingv1alpha1.Tenant

	for _, selector := range selectors {
		var tenantsForSelector loggingv1alpha1.TenantList
		listOpts := &client.ListOptions{
			LabelSelector: selector,
		}

		if err := r.Client.List(ctx, &tenantsForSelector, listOpts); client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		tenants = append(tenants, tenantsForSelector.Items...)
	}

	return tenants, nil
}

func (r *CollectorReconciler) getSubscriptionsForTenant(ctx context.Context, tentant *v1alpha1.Tenant) ([]loggingv1alpha1.Subscription, error) {
	var selectors []labels.Selector

	for _, labelSelector := range tentant.Spec.ResourceNamespaceSelectors {
		selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, selector)
	}

	var namespaces []apiv1.Namespace
	for _, selector := range selectors {
		var namespacesForSelector apiv1.NamespaceList
		listOpts := &client.ListOptions{
			LabelSelector: selector,
		}

		if err := r.List(ctx, &namespacesForSelector, listOpts); client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		namespaces = append(namespaces, namespacesForSelector.Items...)
	}

	var subscriptions []loggingv1alpha1.Subscription

	for _, ns := range namespaces {

		var subscriptionsForNS loggingv1alpha1.SubscriptionList
		listOpts := &client.ListOptions{
			Namespace: ns.Name,
		}

		if err := r.List(ctx, &subscriptionsForNS, listOpts); client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, subscriptionsForNS.Items...)

	}

	return subscriptions, nil
}
