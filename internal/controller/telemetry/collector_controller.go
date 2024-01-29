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
	"strings"

	"emperror.dev/errors"
	"golang.org/x/exp/slices"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kube-logging/subscription-operator/api/telemetry/v1alpha1"

	"github.com/cisco-open/operator-tools/pkg/reconciler"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const collectorReferenceField = ".status.collector"
const tenantReferenceField = ".status.tenant"

//+kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors;tenants;subscriptions;oteloutputs;,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/status;tenants/status;subscriptions/status;oteloutputs/status;,verbs=get;update;patch
//+kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes;namespaces;endpoints;nodes/proxy,verbs=get;list;watch
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;serviceaccounts;pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets;daemonsets;replicasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete

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

	tenantsToDisown, err := r.getTenantsReferencingCollectorButNotSelected(ctx, collector, tenants)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.disownTenants(ctx, tenantsToDisown)
	if err != nil {
		return ctrl.Result{}, err
	}

	tenantNames := []string{}

	if err := r.Status().Update(ctx, collector); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Setting collector status")


	subscriptions := []v1alpha1.Subscription{}

	for _, tenant := range tenants {
		// check if tenant is owned by us, or make it ours only if orphan
		// this update will connect the tenant and collector exclusively

		if tenant.Status.Collector != "" && tenant.Status.Collector != collector.Name {
			logger.Error(errors.Errorf("tenant (%s) is owned by another collector (%s), skipping reconciliation for this collector (%s)", tenant.Name, tenant.Status.Collector, collector.Name),
				"make sure to remove tenant from the previous collector before adopting to new collector")
			continue
		}

		tenantNames = append(tenantNames, tenant.Name)

		tenant.Status.Collector = collector.Name

		subscriptionsForTenant, err := r.getSubscriptionsForTenant(ctx, &tenant)

		if err != nil {
			return ctrl.Result{}, err
		}

		subscriptionsToDisown, err := r.getSubscriptionsReferencingTenantButNotSelected(ctx, &tenant, subscriptionsForTenant)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.disownSubscriptions(ctx, subscriptionsToDisown)
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

		tenant.Status.Collector = collector.Name

		slices.Sort(logsourceNamespacesForTenant)
		tenant.Status.LogSourceNamespaces = logsourceNamespacesForTenant


		if err := r.Status().Update(ctx, &tenant); err != nil {
			return ctrl.Result{}, err
		}

		logger.Info("Setting tenant status")

	}

	slices.Sort(tenantNames)

	collector.Status.Tenants = tenantNames

	r.Status().Update(ctx, collector)
	logger.Info("Setting collector status")

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
			Namespace: collector.Spec.ControlNamespace,
		},
		Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
			Config:         otelConfig,
			Mode:           otelv1alpha1.ModeDaemonSet,
			Image:          "otel/opentelemetry-collector-contrib:0.92.0",
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

	if err := ctrl.SetControllerReference(collector, &otelCollector, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))

	_, err = resourceReconciler.ReconcileResource(&otelCollector, reconciler.StatePresent)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.Tenant{}, collectorReferenceField, func(rawObj client.Object) []string {
		tenant := rawObj.(*v1alpha1.Tenant)
		if tenant.Status.Collector == "" {
			return nil
		}
		return []string{tenant.Status.Collector}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.Subscription{}, tenantReferenceField, func(rawObj client.Object) []string {
		subscription := rawObj.(*v1alpha1.Subscription)
		if subscription.Status.Tenant == "" {
			return nil
		}
		return []string{subscription.Status.Tenant}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Collector{}).
		Watches(&v1alpha1.Tenant{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
			requests := []reconcile.Request{}
			tenant, _ := object.(*v1alpha1.Tenant)

			collectors := v1alpha1.CollectorList{}
			r.List(ctx, &collectors)

			for _, collector := range collectors.Items {
				tenantsForCollector, err := r.getTenantsMatchingSelectors(ctx, collector.Spec.TenantSelector)
				if err != nil {
					return nil
				}

				for _, t := range tenantsForCollector {
					if t.Name == tenant.Name {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name: collector.Name,
							},
						})
					}
				}

				tenantsToDisown, err := r.getTenantsReferencingCollectorButNotSelected(ctx, &collector, tenantsForCollector)
				if err != nil {
					return nil
				}

				for _, t := range tenantsToDisown {
					if t.Name == tenant.Name {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name: collector.Name,
							},
						})
					}
				}
			}

			return requests
		})).
		Watches(&v1alpha1.Subscription{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
			requests := []reconcile.Request{}
			subscription, _ := object.(*v1alpha1.Subscription)

			tenants := v1alpha1.TenantList{}
			r.List(ctx, &tenants)

			for _, tenant := range tenants.Items {
				subscriptionsForTenant, err := r.getSubscriptionsForTenant(ctx, &tenant)
				if err != nil {
					return nil
				}

				for _, s := range subscriptionsForTenant {
					if s.Name == subscription.Name {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name: tenant.Status.Collector,
							},
						})
					}
				}

				subscriptionstoDisown, err := r.getSubscriptionsReferencingTenantButNotSelected(ctx, &tenant, subscriptionsForTenant)
				if err != nil {
					return nil
				}

				for _, s := range subscriptionstoDisown {
					if s.Name == subscription.Name {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name: tenant.Status.Collector,
							},
						})
					}
				}
			}

			return requests
		})).
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
			Namespace: collector.Spec.ControlNamespace,
		},
	}

	if err := ctrl.SetControllerReference(collector, &serviceAccount, r.Scheme); err != nil {
		return v1alpha1.NamespacedName{}, err
	}

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
			Namespace: collector.Spec.ControlNamespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: fmt.Sprintf("%s-pod-association-reader", collector.Name),
		},
	}

	if err := ctrl.SetControllerReference(collector, &clusterRoleBinding, r.Scheme); err != nil {
		return err
	}

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

	if err := ctrl.SetControllerReference(collector, &clusterRole, r.Scheme); err != nil {
		return err
	}

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))

	_, err := resourceReconciler.ReconcileResource(&clusterRole, reconciler.StatePresent)

	return err
}


func getSubscriptionNamesFromSubscription(subscriptions []v1alpha1.Subscription) []v1alpha1.NamespacedName {
	subscriptionNames := make([]v1alpha1.NamespacedName, len(subscriptions))
	for i, subscription := range subscriptions {
		subscriptionNames[i] = subscription.NamespacedName()
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

func (r *CollectorReconciler) getTenantsReferencingCollectorButNotSelected(ctx context.Context, collector *v1alpha1.Collector, selectedTenants []v1alpha1.Tenant) ([]v1alpha1.Tenant, error) {
	var tenantsReferencing v1alpha1.TenantList

	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(collectorReferenceField, collector.Name),
	}

	if err := r.Client.List(ctx, &tenantsReferencing, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	tenantsToDisown := []v1alpha1.Tenant{}

	for _, tenantReferencing := range tenantsReferencing.Items {
		selected := false

		for _, selectedTenant := range selectedTenants {
			if tenantReferencing.Name == selectedTenant.Name {
				selected = true
				break
			}
		}

		if !selected {
			tenantsToDisown = append(tenantsToDisown, tenantReferencing)
		}

	}

	return tenantsToDisown, nil

}

func (r *CollectorReconciler) disownTenants(ctx context.Context, tenantsToDisown []v1alpha1.Tenant) error {
	logger := log.FromContext(ctx)
	for _, tenant := range tenantsToDisown {
		tenant.Status.Collector = ""
		err := r.Client.Status().Update(ctx, &tenant)
		if err != nil {
			return err
		}
		logger.Info("Disowning tenant", "tenant", tenant.Name)
	}

	return nil
}

func (r *CollectorReconciler) getSubscriptionsReferencingTenantButNotSelected(ctx context.Context, tenant *v1alpha1.Tenant, selectedSubscriptions []v1alpha1.Subscription) ([]v1alpha1.Subscription, error) {
	var subscriptionsReferencing v1alpha1.SubscriptionList
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(tenantReferenceField, tenant.Name),
	}

	if err := r.Client.List(ctx, &subscriptionsReferencing, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	var subscriptionsToDisown []v1alpha1.Subscription

	for _, subscriptionReferencing := range subscriptionsReferencing.Items {
		selected := false

		for _, selectedSubscription := range selectedSubscriptions {
			if subscriptionReferencing.Name == selectedSubscription.Name {
				selected = true
				break
			}
		}

		if !selected {
			subscriptionsToDisown = append(subscriptionsToDisown, subscriptionReferencing)
		}

	}

	return subscriptionsToDisown, nil

}

func (r *CollectorReconciler) disownSubscriptions(ctx context.Context, subscriptionsToDisown []v1alpha1.Subscription) error {
	logger := log.FromContext(ctx)
	for _, subscription := range subscriptionsToDisown {
		subscription.Status.Tenant = ""
		err := r.Client.Status().Update(ctx, &subscription)
		if err != nil {
			return err
		}
		logger.Info("Disowning subscription", "subscription", fmt.Sprintf("%s/%s", subscription.Namespace, subscription.Name))
	}

	return nil
}

func (r *CollectorReconciler) getAllOutputs(ctx context.Context) ([]v1alpha1.OtelOutput, error) {

	var outputList v1alpha1.OtelOutputList

	if err := r.List(ctx, &outputList); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return outputList.Items, nil
}

func (r *CollectorReconciler) getSubscriptionsForTenant(ctx context.Context, tenant *v1alpha1.Tenant) ([]v1alpha1.Subscription, error) {
	logger := log.FromContext(ctx)

	namespaces, err := r.getNamespacesForSelectorSlice(ctx, tenant.Spec.SubscriptionNamespaceSelectors)

	if err != nil {
		return nil, err
	}

	var selectedSubscriptions []v1alpha1.Subscription

	for _, ns := range namespaces {

		var subscriptionsForNS v1alpha1.SubscriptionList
		listOpts := &client.ListOptions{
			Namespace: ns.Name,
		}

		if err := r.List(ctx, &subscriptionsForNS, listOpts); client.IgnoreNotFound(err) != nil {
			return nil, err
		}

		selectedSubscriptions = append(selectedSubscriptions, subscriptionsForNS.Items...)

	}

	var validSubscriptions []v1alpha1.Subscription

	for _, subscription := range selectedSubscriptions {
		if subscription.Status.Tenant != "" && subscription.Status.Tenant != tenant.Name {
			logger.Error(errors.Errorf("subscription (%s) is owned by another tenant (%s), skipping reconciliation for this tenant (%s)", subscription.Name, subscription.Status.Tenant, tenant.Name),
				"make sure to remove subscription from the previous tenant before adopting to new tenant")
			continue
		}

		subscription.Status.Tenant = tenant.Name
		validSubscriptions = append(validSubscriptions, subscription)

		r.Status().Update(ctx, &subscription)
		logger.Info("Setting subscription status")

	}

	return validSubscriptions, nil
}

func (r *CollectorReconciler) getNamespacesForSelectorSlice(ctx context.Context, labelSelectors []metav1.LabelSelector) ([]apiv1.Namespace, error) {

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

	normalizeNamespaceSlice(namespaces)

	return namespaces, nil
}

func normalizeNamespaceSlice(inputList []apiv1.Namespace) []apiv1.Namespace {
	allKeys := make(map[string]bool)
	uniqueList := []apiv1.Namespace{}
	for _, item := range inputList {
		if allKeys[item.Name] {
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

func (r *CollectorReconciler) getLogsourceNamespaceNamesForTenant(ctx context.Context, tentant *v1alpha1.Tenant) ([]string, error) {
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
