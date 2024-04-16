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
	"math"
	"reflect"
	"slices"
	"time"

	"emperror.dev/errors"
	"github.com/cisco-open/operator-tools/pkg/reconciler"
	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const tenantReferenceField = ".status.tenant"
const requeueDelayOnFailedTenant = 20 * time.Second

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type TenantFailedError struct {
	msg string
}

func (e *TenantFailedError) Error() string { return e.msg }

func (r *CollectorReconciler) buildConfigInputForCollector(ctx context.Context, collector *v1alpha1.Collector) (OtelColConfigInput, error) {
	logger := log.FromContext(ctx)
	tenantSubscriptionMap := make(map[string][]v1alpha1.NamespacedName)
	subscriptionOutputMap := make(map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName)

	tenants, err := r.getTenantsMatchingSelectors(ctx, collector.Spec.TenantSelector)
	subscriptions := make(map[v1alpha1.NamespacedName]v1alpha1.Subscription)
	outputs := []v1alpha1.OtelOutput{}

	if err != nil {
		logger.Error(errors.WithStack(err), "failed listing tenants")
		return OtelColConfigInput{}, err
	}

	for _, tenant := range tenants {

		if tenant.Status.State == v1alpha1.StateFailed {
			logger.Info("tenant %q is in failed state, retrying later", tenant.Name)
			return OtelColConfigInput{}, &TenantFailedError{msg: "tenant failed"}
		}

		subscriptionNames := tenant.Status.Subscriptions
		tenantSubscriptionMap[tenant.Name] = subscriptionNames

		for _, subsName := range subscriptionNames {
			queriedSubs := &v1alpha1.Subscription{}
			if err = r.Client.Get(ctx, types.NamespacedName(subsName), queriedSubs); err != nil {
				logger.Error(errors.WithStack(err), "failed getting subscriptions for tenant", "tenant", tenant.Name)
				return OtelColConfigInput{}, err
			}
			subscriptions[subsName] = *queriedSubs
		}
	}

	for _, subscription := range subscriptions {
		outputNames := subscription.Status.Outputs
		subscriptionOutputMap[subscription.NamespacedName()] = outputNames

		for _, outputName := range outputNames {
			queriedOutput := &v1alpha1.OtelOutput{}
			if err = r.Client.Get(ctx, types.NamespacedName(outputName), queriedOutput); err != nil {
				logger.Error(errors.WithStack(err), "failed getting outputs for subscription", "subscription", subscription.NamespacedName().String())
				return OtelColConfigInput{}, err
			}
			outputs = append(outputs, *queriedOutput)
		}
	}

	otelConfigInput := OtelColConfigInput{
		Tenants:               tenants,
		Subscriptions:         subscriptions,
		Outputs:               outputs,
		TenantSubscriptionMap: tenantSubscriptionMap,
		SubscriptionOutputMap: subscriptionOutputMap,
		Debug:                 collector.Spec.Debug,
		MemoryLimiter:         *collector.Spec.MemoryLimiter,
	}

	return otelConfigInput, nil
}

// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors;tenants;subscriptions;oteloutputs;,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/status;tenants/status;subscriptions/status;oteloutputs/status;,verbs=get;update;patch
// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes;namespaces;endpoints;nodes/proxy,verbs=get;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;serviceaccounts;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets;daemonsets;replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete

func (r *CollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "collector", req.Name)

	collector := &v1alpha1.Collector{}
	logger.Info("Reconciling collector")

	if err := r.Get(ctx, req.NamespacedName, collector); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	collector.Spec.SetDefaults()
	originalCollectorStatus := collector.Status.DeepCopy()

	otelConfigInput, err := r.buildConfigInputForCollector(ctx, collector)
	if err != nil {
		if errors.Is(err, &TenantFailedError{}) {
			return ctrl.Result{RequeueAfter: requeueDelayOnFailedTenant}, err

		}
		return ctrl.Result{}, err
	}

	otelConfig, err := otelConfigInput.ToIntermediateRepresentation(ctx).ToYAML()
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
			UpgradeStrategy: "none",
			Config:          otelConfig,
			Mode:            otelv1alpha1.ModeDaemonSet,
			Image:           "ghcr.io/axoflow/axoflow-otel-collector/axoflow-otel-collector:0.97.0-dev3",
			ServiceAccount:  saName.Name,
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

	if memoryLimit := collector.Spec.GetMemoryLimit(); memoryLimit != nil {
		// Calculate 80% of the specified memory limit for GOMEMLIMIT
		goMemLimitPercent := 0.8
		goMemLimitValue := int64(math.Round(float64(memoryLimit.Value()) * goMemLimitPercent))
		goMemLimit := resource.NewQuantity(goMemLimitValue, resource.BinarySI)
		otelCollector.Spec.Env = append(otelCollector.Spec.Env, corev1.EnvVar{Name: "GOMEMLIMIT", Value: goMemLimit.String()})
	}

	if err := ctrl.SetControllerReference(collector, &otelCollector, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))

	_, err = resourceReconciler.ReconcileResource(&otelCollector, reconciler.StatePresent)
	if err != nil {
		return ctrl.Result{}, err
	}

	tenantNames := []string{}

	for _, tenant := range otelConfigInput.Tenants {
		tenantNames = append(tenantNames, tenant.Name)
	}

	collector.Status.Tenants = normalizeStringSlice(tenantNames)

	if !reflect.DeepEqual(originalCollectorStatus, collector.Status) {
		if err = r.Client.Status().Update(ctx, collector); err != nil {
			logger.Error(errors.WithStack(err), "failed updating collector status")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	addCollectorRequest := func(requests []reconcile.Request, collector string) []reconcile.Request {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: collector,
			},
		})
		return requests
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Collector{}).
		Watches(&v1alpha1.Tenant{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) (requests []reconcile.Request) {
			logger := log.FromContext(ctx)

			collectors := &v1alpha1.CollectorList{}
			err := r.List(ctx, collectors)
			if err != nil {
				logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
				return
			}

			for _, tenant := range collectors.Items {
				requests = addCollectorRequest(requests, tenant.Name)
			}

			return
		})).
		Watches(&v1alpha1.Subscription{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) (requests []reconcile.Request) {
			logger := log.FromContext(ctx)

			collectors := &v1alpha1.CollectorList{}
			err := r.List(ctx, collectors)
			if err != nil {
				logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
				return
			}

			for _, tenant := range collectors.Items {
				requests = addCollectorRequest(requests, tenant.Name)
			}

			return
		})).
		Watches(&v1alpha1.OtelOutput{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) (requests []reconcile.Request) {
			logger := log.FromContext(ctx)

			collectors := &v1alpha1.CollectorList{}
			err := r.List(ctx, collectors)
			if err != nil {
				logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
				return
			}

			for _, tenant := range collectors.Items {
				requests = addCollectorRequest(requests, tenant.Name)
			}

			return
		})).
		Watches(&apiv1.Namespace{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) (requests []reconcile.Request) {
			logger := log.FromContext(ctx)

			collectors := &v1alpha1.CollectorList{}
			err := r.List(ctx, collectors)
			if err != nil {
				logger.Error(errors.WithStack(err), "failed listing tenants for mapping requests, unable to send requests")
				return
			}

			for _, tenant := range collectors.Items {
				requests = addCollectorRequest(requests, tenant.Name)
			}

			return
		})).
		Owns(&otelv1alpha1.OpenTelemetryCollector{}).
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

func normalizeStringSlice(inputList []string) []string {
	allKeys := make(map[string]bool)
	uniqueList := []string{}
	for _, item := range inputList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			uniqueList = append(uniqueList, item)
		}
	}
	slices.Sort(uniqueList)

	return uniqueList
}
