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

	"emperror.dev/errors"
	"golang.org/x/exp/slices"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/subscription-operator/api/telemetry/v1alpha1"

	"github.com/cisco-open/operator-tools/pkg/reconciler"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/finalizers,verbs=update

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
	tenants, err := r.getTenantsMatchingSelectors(ctx, collector.Spec.TenantSelector)
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

		logsourceNamespacesForTenant, err := r.getLogsourceNamespaceNamesForTenant(ctx, &tenant)
		if err != nil {
			return ctrl.Result{}, err
		}

		slices.Sort(logsourceNamespacesForTenant)
		tenant.Status.LogSourceNamespaces = logsourceNamespacesForTenant

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

	saName, err := r.reconcileRBAC(ctx, collector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("%+v", err)
	}

	otelCollector := otelv1alpha1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("otelcollector-%s", collector.Name),
			Namespace: collector.Namespace,
		},
		Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
			Config:         otelConfig,
			Mode:           otelv1alpha1.ModeDaemonSet,
			Image:          "otel/opentelemetry-collector-contrib:0.91.0",
			ServiceAccount: saName.Name,
			VolumeMounts: []apiv1.VolumeMount{
				{
					Name:      "varlog",
					ReadOnly:  true,
					MountPath: "/var/log",
				},
				{
					Name:      "varlibdockercontainers",
					ReadOnly:  true,
					MountPath: "/var/lib/docker/containers",
				},
			},
			Volumes: []apiv1.Volume{
				{
					Name: "varlog",
					VolumeSource: apiv1.VolumeSource{
						HostPath: &apiv1.HostPathVolumeSource{
							Path: "/var/log",
						},
					},
				},
				{
					Name: "varlibdockercontainers",
					VolumeSource: apiv1.VolumeSource{
						HostPath: &apiv1.HostPathVolumeSource{
							Path: "/var/lib/docker/containers",
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(collector, &otelCollector, r.Scheme)

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))

	_, err = resourceReconciler.ReconcileResource(&otelCollector, reconciler.StatePresent)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Collector{}).
		Complete(r)
}

func (r *CollectorReconciler) reconcileRBAC(ctx context.Context, collector *v1alpha1.Collector) (v1alpha1.NamespacedName, error) {

	errCR := r.reconcileClusterRole(ctx, collector)

	sa, errSA := r.reconcileServiceAccount(ctx, collector)

	errCRB := r.reconcileClusterRoleBinding(ctx, collector)

	if allErr := errors.Combine(errCR, errSA, errCRB); allErr != nil {
		return v1alpha1.NamespacedName{}, allErr
	}

	return sa, nil
}

func (r *CollectorReconciler) reconcileServiceAccount(ctx context.Context, collector *v1alpha1.Collector) (v1alpha1.NamespacedName, error) {
	logger := log.FromContext(ctx)

	serviceAccount := apiv1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sa", collector.Name),
			Namespace: collector.Namespace,
		},
	}

	ctrl.SetControllerReference(collector, &serviceAccount, r.Scheme)

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))

	_, err := resourceReconciler.ReconcileResource(&serviceAccount, reconciler.StatePresent)
	if err != nil {
		return v1alpha1.NamespacedName{}, err
	}

	return v1alpha1.NamespacedName{Namespace: serviceAccount.Namespace, Name: serviceAccount.Name}, nil
}
func (r *CollectorReconciler) reconcileClusterRoleBinding(ctx context.Context, collector *v1alpha1.Collector) error {
	logger := log.FromContext(ctx)

	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-crb", collector.Name),
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      fmt.Sprintf("%s-sa", collector.Name),
			Namespace: collector.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: fmt.Sprintf("%s-pod-association-reader", collector.Name),
		},
	}

	ctrl.SetControllerReference(collector, &clusterRoleBinding, r.Scheme)

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))

	_, err := resourceReconciler.ReconcileResource(&clusterRoleBinding, reconciler.StatePresent)
	return err
}

func (r *CollectorReconciler) reconcileClusterRole(ctx context.Context, collector *v1alpha1.Collector) error {
	logger := log.FromContext(ctx)

	clusterRole := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-pod-association-reader", collector.Name),
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{""},
				Resources: []string{"pods", "namespaces"},
			},
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets"},
			},
		},
	}

	ctrl.SetControllerReference(collector, &clusterRole, r.Scheme)

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))

	_, err := resourceReconciler.ReconcileResource(&clusterRole, reconciler.StatePresent)

	return err
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

func (r *CollectorReconciler) getTenantsMatchingSelectors(ctx context.Context, labelSelector metav1.LabelSelector) ([]v1alpha1.Tenant, error) {

	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return nil, err
	}

	var tenantsForSelector v1alpha1.TenantList
	listOpts := &client.ListOptions{
		LabelSelector: selector,
	}

	if err := r.Client.List(ctx, &tenantsForSelector, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return tenantsForSelector.Items, nil
}

func (r *CollectorReconciler) getAllOutputs(ctx context.Context) ([]v1alpha1.OtelOutput, error) {

	var outputList v1alpha1.OtelOutputList

	if err := r.List(ctx, &outputList); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return outputList.Items, nil
}

func (r *CollectorReconciler) getSubscriptionsForTenant(ctx context.Context, tentant *v1alpha1.Tenant) ([]v1alpha1.Subscription, error) {

	namespaces, err := r.getNamespacesForSelector(ctx, &tentant.Spec.SubscriptionNamespaceSelector)

	if err != nil {
		return nil, err
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

func (r *CollectorReconciler) getNamespacesForSelector(ctx context.Context, labelSelector *metav1.LabelSelector) ([]apiv1.Namespace, error) {
	var namespaces apiv1.NamespaceList

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)

	if err != nil {
		return nil, err
	}

	listOpts := &client.ListOptions{
		LabelSelector: selector,
	}

	if err := r.List(ctx, &namespaces, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return namespaces.Items, nil
}

func (r *CollectorReconciler) getLogsourceNamespaceNamesForTenant(ctx context.Context, tentant *v1alpha1.Tenant) ([]string, error) {

	namespaces, err := r.getNamespacesForSelector(ctx, &tentant.Spec.SubscriptionNamespaceSelector)
	if err != nil {
		return nil, err
	}

	var namespaceNames []string

	for _, namespace := range namespaces {
		namespaceNames = append(namespaceNames, namespace.Name)
	}

	return namespaceNames, nil

}
