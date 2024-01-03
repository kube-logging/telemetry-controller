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
	"fmt"

	"golang.org/x/exp/slices"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/subscription-operator/api/logging/v1alpha1"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
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

	if err := r.Get(ctx, req.NamespacedName, collector); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	tenantSubscriptionMap := make(map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName)
	tenants, err := r.getTenantsMatchingSelectors(ctx, collector.Spec.TenantSelectors)
	if err != nil {
		return ctrl.Result{}, err
	}

	tenantNames := getTenantNamesFromTenants(tenants)
	slices.Sort(tenantNames)

	collector.Status.Tenants = tenantNames

	r.Status().Update(ctx, collector)
	logger.Info("Setting collector status")

	subscriptions := []v1alpha1.Subscription{}

	for _, tenant := range tenants {

		subscriptionsForTenant, err := r.getSubscriptionsForTenant(ctx, &tenant)

		if err != nil {
			return ctrl.Result{}, err
		}

		subscriptions = append(subscriptions, subscriptionsForTenant...)

		subscriptionNames := getSubscriptionNamesFromSubscription(subscriptionsForTenant)

		tenantSubscriptionMap[tenant.NamespacedName()] = subscriptionNames

		stringSubscriptionNames := []string{}

		for _, name := range subscriptionNames {
			stringSubscriptionNames = append(stringSubscriptionNames, name.String())
		}

		slices.Sort(stringSubscriptionNames)

		tenant.Status.Subscriptions = stringSubscriptionNames

		r.Status().Update(ctx, &tenant)
		logger.Info("Setting tenant status")

	}

	outputs, err := r.getAllOutputs(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	subscriptionOutputMap := map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName{}

	for _, subscription := range subscriptions {
		subscriptionOutputMap[subscription.NamespacedName()] = subscription.Spec.Outputs
	}

	otelConfigInput := OtelColConfigInput{
		Tenants:               tenants,
		Subscriptions:         subscriptions,
		Outputs:               outputs,
		TenantSubscriptionMap: tenantSubscriptionMap,
		SubscriptionOutputMap: subscriptionOutputMap,
	}

	otelConfig, err := otelConfigInput.ToIntermediateRepresentation().ToYAML()
	if err != nil {
		return ctrl.Result{}, err
	}

	otelCollector := otelv1alpha1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("otelcollector-%s", collector.Name),
			Namespace: collector.Namespace,
		},
		Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
			Config: otelConfig,
			Mode:   otelv1alpha1.ModeDaemonSet,
			Image:  "otel/opentelemetry-collector-contrib:0.91.0",
		},
	}

	ctrl.SetControllerReference(collector, &otelCollector, r.Scheme)

	foundOtelCollector := otelv1alpha1.OpenTelemetryCollector{}

	err = r.Client.Get(ctx, client.ObjectKey{Namespace: otelCollector.Namespace, Name: otelCollector.Name}, &foundOtelCollector)
	if apierrors.IsNotFound(err) {
		if err := r.Client.Create(ctx, &otelCollector); err != nil {
			logger.Error(err, "failed to create OpentelemetryCollector resource")
			return ctrl.Result{}, err
		}

		logger.Info("created OpentelemetryCollector resource for Collector")
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "failed to get OpentelemetryCollector for Collector")
		return ctrl.Result{}, err
	}

	if foundOtelCollector.Spec.Config != otelCollector.Spec.Config || foundOtelCollector.Spec.Mode != otelCollector.Spec.Mode || foundOtelCollector.Spec.Image != otelCollector.Spec.Image {
		otelCollector.SetResourceVersion(foundOtelCollector.GetResourceVersion())
		if err := r.Client.Update(ctx, &otelCollector); err != nil {
			logger.Error(err, "failed to update ConfigMap")
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Collector{}).
		Complete(r)
}

func getTenantNamesFromTenants(tenants []v1alpha1.Tenant) []string {
	var tenantNames []string
	for _, tenant := range tenants {
		tenantNames = append(tenantNames, tenant.Name)
	}

	return tenantNames
}

func getSubscriptionNamesFromSubscription(subscriptions []v1alpha1.Subscription) []v1alpha1.NamespacedName {
	var subscriptionNames []v1alpha1.NamespacedName
	for _, subscription := range subscriptions {
		subscriptionNames = append(subscriptionNames, subscription.NamespacedName())
	}

	return subscriptionNames
}

func (r *CollectorReconciler) getTenantsMatchingSelectors(ctx context.Context, labelSelectorsList []metav1.LabelSelector) ([]v1alpha1.Tenant, error) {
	var selectors []labels.Selector

	for _, labelSelector := range labelSelectorsList {
		selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, selector)
	}

	var tenants []v1alpha1.Tenant

	for _, selector := range selectors {
		var tenantsForSelector v1alpha1.TenantList
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

func (r *CollectorReconciler) getAllOutputs(ctx context.Context) ([]v1alpha1.OtelOutput, error) {

	var outputList v1alpha1.OtelOutputList

	if err := r.List(ctx, &outputList); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return outputList.Items, nil
}

func (r *CollectorReconciler) getSubscriptionsForTenant(ctx context.Context, tentant *v1alpha1.Tenant) ([]v1alpha1.Subscription, error) {
	var selectors []labels.Selector

	for _, labelSelector := range tentant.Spec.SubscriptionNamespaceSelector {
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

	var subscriptions []v1alpha1.Subscription

	for _, ns := range namespaces {

		var subscriptionsForNS v1alpha1.SubscriptionList
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
