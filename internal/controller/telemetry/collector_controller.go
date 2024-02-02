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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-logging/subscription-operator/api/telemetry/v1alpha1"

	"github.com/cisco-open/operator-tools/pkg/reconciler"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	PersistPath                   = "/opt/telemetry-controller/persist"
	ReceiversPersistPath          = PersistPath + "/receivers"
	ExportersPersistPath          = PersistPath + "/exporters"
	DefaultMountContainerImage    = "busybox"
	DefaultMountContainerImageTag = "latest"
)

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

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

	tenantNames := getTenantNamesFromTenants(tenants)
	slices.Sort(tenantNames)

	collector.Status.Tenants = tenantNames

	if err := r.Status().Update(ctx, collector); err != nil {
		return ctrl.Result{}, err
	}

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

		if err := r.Status().Update(ctx, &tenant); err != nil {
			return ctrl.Result{}, err
		}

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
		Fsync:                 collector.Spec.Fsync,
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

	persistVolumeMount := apiv1.VolumeMount{
		Name:      "persist",
		ReadOnly:  false,
		MountPath: PersistPath,
	}
	otelCollector.Spec.VolumeMounts = append(otelCollector.Spec.VolumeMounts, persistVolumeMount)

	mountInitContainer := apiv1.Container{
		Name:    "persist-mount-fix",
		Image:   fmt.Sprintf("%s:%s", DefaultMountContainerImage, DefaultMountContainerImageTag),
		Command: []string{"sh", "-c", "mkdir -p " + ReceiversPersistPath + "; mkdir -p " + ExportersPersistPath + "; " + "chmod -R 777 " + PersistPath},

		VolumeMounts: []apiv1.VolumeMount{persistVolumeMount},
	}
	otelCollector.Spec.InitContainers = append(otelCollector.Spec.InitContainers, mountInitContainer)

	persistVolumeType := apiv1.HostPathDirectoryOrCreate
	persistVolume := apiv1.Volume{
		Name: "persist",
		VolumeSource: apiv1.VolumeSource{
			HostPath: &apiv1.HostPathVolumeSource{
				Path: PersistPath,
				Type: &persistVolumeType,
			},
		},
	}
	otelCollector.Spec.Volumes = append(otelCollector.Spec.Volumes, persistVolume)

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

func getTenantNamesFromTenants(tenants []v1alpha1.Tenant) []string {
	tenantNames := make([]string, len(tenants))
	for i, tenant := range tenants {
		tenantNames[i] = tenant.Name
	}

	return tenantNames
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

func (r *CollectorReconciler) getAllOutputs(ctx context.Context) ([]v1alpha1.OtelOutput, error) {

	var outputList v1alpha1.OtelOutputList

	if err := r.List(ctx, &outputList); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return outputList.Items, nil
}

func (r *CollectorReconciler) getSubscriptionsForTenant(ctx context.Context, tentant *v1alpha1.Tenant) ([]v1alpha1.Subscription, error) {

	namespaces, err := r.getNamespacesForSelectorSlice(ctx, tentant.Spec.SubscriptionNamespaceSelectors)

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
