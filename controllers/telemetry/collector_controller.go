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
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/cisco-open/operator-tools/pkg/reconciler"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
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

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	otelcolconfgen "github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components/extension/storage"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/validator"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

const (
	otelCollectorKind            = "OpenTelemetryCollector"
	requeueDelayOnFailedTenant   = 20 * time.Second
	axoflowOtelCollectorImageRef = "ghcr.io/axoflow/axoflow-otel-collector/axoflow-otel-collector:0.120.0-axoflow.6"
)

var ErrTenantFailed = errors.New("tenant failed")

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type BasicAuthClientAuthConfig struct {
	Username string
	Password string
}

func (r *CollectorReconciler) buildConfigInputForCollector(ctx context.Context, collector *v1alpha1.Collector) (otelcolconfgen.OtelColConfigInput, error) {
	logger := log.FromContext(ctx)
	tenantSubscriptionMap := make(map[string][]v1alpha1.NamespacedName)
	var bridgesReferencedByTenant []v1alpha1.Bridge
	subscriptionOutputMap := make(map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName)

	tenants, err := r.getTenantsMatchingSelectors(ctx, collector.Spec.TenantSelector)
	subscriptions := make(map[v1alpha1.NamespacedName]v1alpha1.Subscription)
	outputs := []components.OutputWithSecretData{}

	if err != nil {
		logger.Error(errors.WithStack(err), "failed listing tenants")
		return otelcolconfgen.OtelColConfigInput{}, err
	}

	for _, tenant := range tenants {
		if tenant.Status.State == state.StateFailed {
			logger.Info(fmt.Sprintf("tenant %q is in failed state, retrying later", tenant.Name))
			return otelcolconfgen.OtelColConfigInput{}, ErrTenantFailed
		}

		subscriptionNames := tenant.Status.Subscriptions
		tenantSubscriptionMap[tenant.Name] = subscriptionNames
		for _, subsName := range subscriptionNames {
			queriedSubs := &v1alpha1.Subscription{}
			if err = r.Get(ctx, types.NamespacedName(subsName), queriedSubs); err != nil {
				logger.Error(errors.WithStack(err), "failed getting subscriptions for tenant", "tenant", tenant.Name)
				return otelcolconfgen.OtelColConfigInput{}, err
			}
			subscriptions[subsName] = *queriedSubs
		}

		bridgeNames := tenant.Status.ConnectedBridges
		for _, bridgeName := range bridgeNames {
			queriedBridge := &v1alpha1.Bridge{}
			if err = r.Get(ctx, types.NamespacedName{Name: bridgeName}, queriedBridge); err != nil {
				logger.Error(errors.WithStack(err), "failed getting bridges for tenant", "tenant", tenant.Name)
				return otelcolconfgen.OtelColConfigInput{}, err
			}

			bridgesReferencedByTenant = append(bridgesReferencedByTenant, *queriedBridge)
		}
	}

	for _, subscription := range subscriptions {
		outputNames := subscription.Status.Outputs
		subscriptionOutputMap[subscription.NamespacedName()] = outputNames
		for _, outputName := range outputNames {
			outputWithSecretData := components.OutputWithSecretData{}

			queriedOutput := &v1alpha1.Output{}
			if err = r.Get(ctx, types.NamespacedName(outputName), queriedOutput); err != nil {
				logger.Error(errors.WithStack(err), "failed getting outputs for subscription", "subscription", subscription.NamespacedName().String())
				return otelcolconfgen.OtelColConfigInput{}, err
			}
			outputWithSecretData.Output = *queriedOutput

			if err := r.populateSecretForOutput(ctx, queriedOutput, &outputWithSecretData); err != nil {
				return otelcolconfgen.OtelColConfigInput{}, err
			}
			outputs = append(outputs, outputWithSecretData)
		}
	}

	return otelcolconfgen.OtelColConfigInput{
		ResourceRelations: components.ResourceRelations{
			Tenants:               tenants,
			Subscriptions:         subscriptions,
			Bridges:               bridgesReferencedByTenant,
			OutputsWithSecretData: outputs,
			TenantSubscriptionMap: tenantSubscriptionMap,
			SubscriptionOutputMap: subscriptionOutputMap,
		},
		Debug:         collector.Spec.Debug,
		MemoryLimiter: *collector.Spec.MemoryLimiter,
	}, nil
}

func (r *CollectorReconciler) populateSecretForOutput(ctx context.Context, queriedOutput *v1alpha1.Output, outputWithSecret *components.OutputWithSecretData) error {
	logger := log.FromContext(ctx)

	if queriedOutput.Spec.Authentication != nil {
		if queriedOutput.Spec.Authentication.BasicAuth != nil && queriedOutput.Spec.Authentication.BasicAuth.SecretRef != nil {
			queriedSecret := &corev1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: queriedOutput.Spec.Authentication.BasicAuth.SecretRef.Namespace, Name: queriedOutput.Spec.Authentication.BasicAuth.SecretRef.Name}, queriedSecret); err != nil {
				logger.Error(errors.WithStack(err), "failed getting secrets for output", "output", queriedOutput.NamespacedName().String())
				return err
			}
			outputWithSecret.Secret = *queriedSecret
		}
		if queriedOutput.Spec.Authentication.BearerAuth != nil && queriedOutput.Spec.Authentication.BearerAuth.SecretRef != nil {
			queriedSecret := &corev1.Secret{}
			if err := r.Get(ctx, types.NamespacedName{Namespace: queriedOutput.Spec.Authentication.BearerAuth.SecretRef.Namespace, Name: queriedOutput.Spec.Authentication.BearerAuth.SecretRef.Name}, queriedSecret); err != nil {
				logger.Error(errors.WithStack(err), "failed getting secrets for output", "output", queriedOutput.NamespacedName().String())
				return err
			}
			outputWithSecret.Secret = *queriedSecret
		}
	}

	return nil
}

// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors;tenants;subscriptions;outputs;bridges;,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/status;tenants/status;subscriptions/status;outputs/status;bridges/status;,verbs=get;update;patch
// +kubebuilder:rbac:groups=telemetry.kube-logging.dev,resources=collectors/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets;nodes;namespaces;endpoints;nodes/proxy,verbs=get;list;watch
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;persistentvolumeclaims;serviceaccounts;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets;daemonsets;replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete

func (r *CollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "collector", req.Name)

	collector := &v1alpha1.Collector{}
	logger.Info(fmt.Sprintf("getting collector: %q", req.Name))

	if err := r.Get(ctx, req.NamespacedName, collector); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	collector.Spec.SetDefaults()
	originalCollectorStatus := collector.Status
	logger.Info(fmt.Sprintf("reconciling collector: %q", collector.Name))

	otelConfigInput, err := r.buildConfigInputForCollector(ctx, collector)
	if err != nil {
		if errors.Is(err, ErrTenantFailed) {
			return ctrl.Result{RequeueAfter: requeueDelayOnFailedTenant}, err
		}
		return ctrl.Result{}, err
	}

	if err := otelConfigInput.ValidateConfig(); err != nil {
		if errors.Is(err, otelcolconfgen.ErrNoResources) {
			logger.Info(err.Error())
			return ctrl.Result{}, nil
		}
		logger.Error(errors.WithStack(err), "invalid otel config input")

		collector.Status.State = state.StateFailed
		if updateErr := r.updateStatus(ctx, collector); updateErr != nil {
			logger.Error(errors.WithStack(updateErr), "failed updating collector status")
			return ctrl.Result{}, errors.Append(err, updateErr)
		}

		return ctrl.Result{}, err
	}

	otelConfig, additionalArgs := otelConfigInput.AssembleConfig(ctx)
	if err := validator.ValidateAssembledConfig(otelConfig); err != nil {
		logger.Error(errors.WithStack(err), "invalid otel config")

		collector.Status.State = state.StateFailed
		if updateErr := r.updateStatus(ctx, collector); updateErr != nil {
			logger.Error(errors.WithStack(updateErr), "failed updating collector status")
			return ctrl.Result{}, errors.Append(err, updateErr)
		}

		return ctrl.Result{}, err
	}

	saName, err := r.reconcileRBAC(ctx, collector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("%+v", err)
	}

	otelCollector, collectorState := r.otelCollector(collector, otelConfig, additionalArgs, otelConfigInput.Tenants, saName.Name)

	if err := ctrl.SetControllerReference(collector, otelCollector, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(logger))
	_, err = resourceReconciler.ReconcileResource(otelCollector, collectorState)
	if err != nil {
		logger.Error(errors.WithStack(err), "failed reconciling collector")

		collector.Status.State = state.StateFailed
		if updateErr := r.updateStatus(ctx, collector); updateErr != nil {
			logger.Error(errors.WithStack(updateErr), "failed updating collector status")
			return ctrl.Result{}, errors.Append(err, updateErr)
		}

		return ctrl.Result{}, err
	}

	tenantNames := []string{}
	for _, tenant := range otelConfigInput.Tenants {
		tenantNames = append(tenantNames, tenant.Name)
	}
	collector.Status.Tenants = normalizeStringSlice(tenantNames)

	collector.Status.State = state.StateReady
	if !reflect.DeepEqual(originalCollectorStatus, collector.Status) {
		logger.Info("collector status changed")

		if updateErr := r.updateStatus(ctx, collector); updateErr != nil {
			logger.Error(errors.WithStack(updateErr), "failed updating collector status")
			return ctrl.Result{}, errors.Append(err, updateErr)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueueAllCollectors := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, _ client.Object) []reconcile.Request {
		logger := log.FromContext(ctx)

		collectors := &v1alpha1.CollectorList{}
		if err := r.List(ctx, collectors); err != nil {
			logger.Error(errors.WithStack(err), "failed listing collectors for mapping requests, unable to send requests")
			return nil
		}

		requests := make([]reconcile.Request, 0, len(collectors.Items))
		for _, collector := range collectors.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: collector.Name,
				},
			})
		}

		return requests
	})

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Collector{})
	watchedResources := []client.Object{
		&v1alpha1.Tenant{},
		&v1alpha1.Subscription{},
		&v1alpha1.Bridge{},
		&v1alpha1.Output{},
		&corev1.Namespace{},
		&corev1.Secret{},
	}
	for _, resource := range watchedResources {
		builder = builder.Watches(resource, enqueueAllCollectors)
	}

	return builder.
		Owns(&otelv1beta1.OpenTelemetryCollector{}).
		Complete(r)
}

func (r *CollectorReconciler) otelCollector(collector *v1alpha1.Collector, otelConfig otelv1beta1.Config, additionalArgs map[string]string, tenants []v1alpha1.Tenant, saName string) (*otelv1beta1.OpenTelemetryCollector, reconciler.DesiredState) {
	otelCollector := otelv1beta1.OpenTelemetryCollector{
		TypeMeta: metav1.TypeMeta{
			APIVersion: otelv1beta1.GroupVersion.String(),
			Kind:       otelCollectorKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("otelcollector-%s", collector.Name),
			Namespace: collector.Spec.ControlNamespace,
		},
		Spec: otelv1beta1.OpenTelemetryCollectorSpec{
			UpgradeStrategy:           "none",
			Config:                    otelConfig,
			Mode:                      otelv1beta1.ModeDaemonSet,
			OpenTelemetryCommonFields: *collector.Spec.OtelCommonFields,
		},
	}
	appendAdditionalVolumesForTenantsFileStorage(&otelCollector.Spec.OpenTelemetryCommonFields, tenants)
	setOtelCommonFieldsDefaults(&otelCollector.Spec.OpenTelemetryCommonFields, additionalArgs, saName)

	if memoryLimit := collector.Spec.GetMemoryLimit(); memoryLimit != nil {
		// Calculate 80% of the specified memory limit for GOMEMLIMIT
		goMemLimitPercent := 0.8
		goMemLimitValue := int64(math.Round(float64(memoryLimit.Value()) * goMemLimitPercent))
		goMemLimit := resource.NewQuantity(goMemLimitValue, resource.BinarySI)
		otelCollector.Spec.Env = append(otelCollector.Spec.Env, corev1.EnvVar{Name: "GOMEMLIMIT", Value: goMemLimit.String()})
	}

	beforeUpdateHook := reconciler.DesiredStateHook(func(current runtime.Object) error {
		if currentCollector, ok := current.(*otelv1beta1.OpenTelemetryCollector); ok {
			otelCollector.Finalizers = currentCollector.Finalizers
		} else {
			return errors.Errorf("unexpected type %T", current)
		}
		return nil
	})

	return &otelCollector, beforeUpdateHook
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

	serviceAccount := corev1.ServiceAccount{
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
				Resources: []string{"pods", "namespaces", "nodes"},
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

	if err := r.List(ctx, &tenantsForSelector, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return tenantsForSelector.Items, nil
}

func (r *CollectorReconciler) updateStatus(ctx context.Context, obj client.Object) error {
	return r.Status().Update(ctx, obj)
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

func appendAdditionalVolumesForTenantsFileStorage(otelCommonFields *otelv1beta1.OpenTelemetryCommonFields, tenants []v1alpha1.Tenant) {
	var volumeMountsForInit []corev1.VolumeMount
	var chmodCommands []string

	for _, tenant := range tenants {
		if tenant.Spec.PersistenceConfig.EnableFileStorage {
			bufferVolumeName := fmt.Sprintf("buffervolume-%s", tenant.Name)
			mountPath := storage.DetermineFileStorageDirectory(tenant.Spec.PersistenceConfig.Directory, tenant.Name)
			volumeMount := corev1.VolumeMount{
				Name:      bufferVolumeName,
				MountPath: mountPath,
			}
			volumeMountsForInit = append(volumeMountsForInit, volumeMount)
			chmodCommands = append(chmodCommands, fmt.Sprintf("chmod -R 777 %s", mountPath))

			otelCommonFields.VolumeMounts = append(otelCommonFields.VolumeMounts, volumeMount)
			otelCommonFields.Volumes = append(otelCommonFields.Volumes, corev1.Volume{
				Name: bufferVolumeName,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: mountPath,
						Type: utils.ToPtr(corev1.HostPathDirectoryOrCreate),
					},
				},
			})
		}
	}

	// Add a single initContainer to handle all chmod operations
	if len(chmodCommands) > 0 && len(volumeMountsForInit) > 0 {
		initContainer := corev1.Container{
			Name:         "init-chmod",
			Image:        "busybox",
			Command:      []string{"sh", "-c"},
			Args:         []string{strings.Join(chmodCommands, " && ")},
			VolumeMounts: volumeMountsForInit,
		}
		otelCommonFields.InitContainers = append(otelCommonFields.InitContainers, initContainer)
	}
}

func setOtelCommonFieldsDefaults(otelCommonFields *otelv1beta1.OpenTelemetryCommonFields, additionalArgs map[string]string, saName string) {
	if otelCommonFields == nil {
		otelCommonFields = &otelv1beta1.OpenTelemetryCommonFields{}
	}

	otelCommonFields.Image = axoflowOtelCollectorImageRef
	otelCommonFields.ServiceAccount = saName

	if otelCommonFields.Args == nil {
		otelCommonFields.Args = make(map[string]string)
	}
	for key, value := range additionalArgs {
		otelCommonFields.Args[key] = value
	}

	volumeMounts := []corev1.VolumeMount{
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
	}
	otelCommonFields.VolumeMounts = append(otelCommonFields.VolumeMounts, volumeMounts...)

	volumes := []corev1.Volume{
		{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: "varlibdockercontainers",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/docker/containers",
				},
			},
		},
	}
	otelCommonFields.Volumes = append(otelCommonFields.Volumes, volumes...)
}
