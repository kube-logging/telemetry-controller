// Copyright © 2025 Kube logging authors
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

package manager

import (
	"context"
	"fmt"
	"math"
	"strings"

	"emperror.dev/errors"
	"github.com/cisco-open/operator-tools/pkg/reconciler"
	"github.com/hashicorp/go-multierror"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	otelcolconfgen "github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components/extension/storage"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

const (
	otelCollectorKind            = "OpenTelemetryCollector"
	axoflowOtelCollectorImageRef = "ghcr.io/axoflow/axoflow-otel-collector/axoflow-otel-collector:0.129.0-axoflow.3"
)

var (
	ErrTenantFailed = errors.New("tenant failed")
	ErrNoResources  = errors.New("there are no resources deployed that the collector(s) can use")
)

// CollectorManager manages collector resources
type CollectorManager struct {
	BaseManager
}

func (c *CollectorManager) BuildConfigInputForCollector(ctx context.Context, collector *v1alpha1.Collector) (otelcolconfgen.OtelColConfigInput, error) {
	tenantSubscriptionMap := make(map[string][]v1alpha1.NamespacedName)
	subscriptionOutputMap := make(map[v1alpha1.NamespacedName][]v1alpha1.NamespacedName)
	var bridgesReferencedByTenant []v1alpha1.Bridge

	tenants, err := c.getTenantsMatchingSelectors(ctx, collector.Spec.TenantSelector)
	subscriptions := make(map[v1alpha1.NamespacedName]v1alpha1.Subscription)
	outputs := []components.OutputWithSecretData{}
	if err != nil {
		c.Error(errors.WithStack(err), "failed listing tenants")
		return otelcolconfgen.OtelColConfigInput{}, err
	}

	for _, tenant := range tenants {
		if tenant.Status.State == state.StateFailed {
			c.Info(fmt.Sprintf("tenant %q is in failed state, retrying later", tenant.Name))
			return otelcolconfgen.OtelColConfigInput{}, ErrTenantFailed
		}

		subscriptionNames := tenant.Status.Subscriptions
		tenantSubscriptionMap[tenant.Name] = subscriptionNames
		for _, subsName := range subscriptionNames {
			queriedSubs := &v1alpha1.Subscription{}
			if err = c.Get(ctx, types.NamespacedName(subsName), queriedSubs); err != nil {
				c.Error(errors.WithStack(err), "failed getting subscriptions for tenant", "tenant", tenant.Name)
				return otelcolconfgen.OtelColConfigInput{}, err
			}
			subscriptions[subsName] = *queriedSubs
		}

		bridgeNames := tenant.Status.ConnectedBridges
		for _, bridgeName := range bridgeNames {
			queriedBridge := &v1alpha1.Bridge{}
			if err = c.Get(ctx, types.NamespacedName{Name: bridgeName}, queriedBridge); err != nil {
				c.Error(errors.WithStack(err), "failed getting bridges for tenant", "tenant", tenant.Name)
				return otelcolconfgen.OtelColConfigInput{}, err
			}
			bridgesReferencedByTenant = append(bridgesReferencedByTenant, *queriedBridge)
		}
	}

	for _, subscription := range subscriptions {
		outputNames := subscription.Status.Outputs
		subscriptionOutputMap[subscription.NamespacedName()] = outputNames
		for _, outputName := range outputNames {
			queriedOutput := &v1alpha1.Output{}
			if err = c.Get(ctx, types.NamespacedName(outputName), queriedOutput); err != nil {
				c.Error(errors.WithStack(err), "failed getting outputs for subscription", "subscription", subscription.NamespacedName().String())
				return otelcolconfgen.OtelColConfigInput{}, err
			}
			outputWithSecretData := components.OutputWithSecretData{
				Output: *queriedOutput,
			}

			if queriedOutput.Spec.Authentication != nil {
				outputSecret, err := components.QueryOutputSecretWithData(ctx, c.Client, queriedOutput)
				if err != nil {
					c.Error(errors.WithStack(err), "failed querying output secret for output", "output", queriedOutput.NamespacedName().String())
				} else {
					outputWithSecretData.Secret = *outputSecret
				}
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

func (c *CollectorManager) ValidateConfigInput(cfgInput otelcolconfgen.OtelColConfigInput) error {
	if cfgInput.IsEmpty() {
		return ErrNoResources
	}

	var result *multierror.Error
	if err := validateTenants(&cfgInput.Tenants); err != nil {
		result = multierror.Append(result, err)
	}
	if err := validateSubscriptionsAndBridges(&cfgInput.Tenants, &cfgInput.Subscriptions, &cfgInput.Bridges); err != nil {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func (c *CollectorManager) ReconcileRBAC(collector *v1alpha1.Collector, scheme *runtime.Scheme) (v1alpha1.NamespacedName, error) {
	errCR := c.reconcileClusterRole(collector, scheme)
	sa, errSA := c.reconcileServiceAccount(collector, scheme)
	errCRB := c.reconcileClusterRoleBinding(collector, scheme)
	if allErr := errors.Combine(errCR, errSA, errCRB); allErr != nil {
		return v1alpha1.NamespacedName{}, allErr
	}

	return sa, nil
}

func (c *CollectorManager) OtelCollector(collector *v1alpha1.Collector, otelConfig otelv1beta1.Config, additionalArgs map[string]string, tenants []v1alpha1.Tenant, saName string) (*otelv1beta1.OpenTelemetryCollector, reconciler.DesiredState) {
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

func (c *CollectorManager) getTenantsMatchingSelectors(ctx context.Context, labelSelector metav1.LabelSelector) ([]v1alpha1.Tenant, error) {
	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return nil, err
	}

	var tenantsForSelector v1alpha1.TenantList
	listOpts := &client.ListOptions{
		LabelSelector: selector,
	}

	if err := c.List(ctx, &tenantsForSelector, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return tenantsForSelector.Items, nil
}

func (c *CollectorManager) reconcileServiceAccount(collector *v1alpha1.Collector, scheme *runtime.Scheme) (v1alpha1.NamespacedName, error) {
	serviceAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-sa", collector.Name),
			Namespace: collector.Spec.ControlNamespace,
		},
	}
	if err := ctrl.SetControllerReference(collector, &serviceAccount, scheme); err != nil {
		return v1alpha1.NamespacedName{}, err
	}

	resourceReconciler := reconciler.NewReconcilerWith(c.Client, reconciler.WithLog(c.Logger))
	_, err := resourceReconciler.ReconcileResource(&serviceAccount, reconciler.StatePresent)
	if err != nil {
		return v1alpha1.NamespacedName{}, err
	}

	return v1alpha1.NamespacedName{Namespace: serviceAccount.Namespace, Name: serviceAccount.Name}, nil
}

func (c *CollectorManager) reconcileClusterRoleBinding(collector *v1alpha1.Collector, scheme *runtime.Scheme) error {
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
	if err := ctrl.SetControllerReference(collector, &clusterRoleBinding, scheme); err != nil {
		return err
	}

	resourceReconciler := reconciler.NewReconcilerWith(c.Client, reconciler.WithLog(c.Logger))
	_, err := resourceReconciler.ReconcileResource(&clusterRoleBinding, reconciler.StatePresent)

	return err
}

func (c *CollectorManager) reconcileClusterRole(collector *v1alpha1.Collector, scheme *runtime.Scheme) error {
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
	if err := ctrl.SetControllerReference(collector, &clusterRole, scheme); err != nil {
		return err
	}

	resourceReconciler := reconciler.NewReconcilerWith(c.Client, reconciler.WithLog(c.Logger))
	_, err := resourceReconciler.ReconcileResource(&clusterRole, reconciler.StatePresent)

	return err
}

func validateTenants(tenants *[]v1alpha1.Tenant) error {
	var result *multierror.Error

	if len(*tenants) == 0 {
		return errors.New("no tenants provided, at least one tenant must be provided")
	}

	for _, tenant := range *tenants {
		if tenant.Status.LogSourceNamespaces != nil && tenant.Spec.SelectFromAllNamespaces {
			result = multierror.Append(result, fmt.Errorf("tenant %s has both log source namespace selectors and select from all namespaces enabled", tenant.Name))
		}
	}

	return result.ErrorOrNil()
}

func validateSubscriptionsAndBridges(tenants *[]v1alpha1.Tenant, subscriptions *map[v1alpha1.NamespacedName]v1alpha1.Subscription, bridges *[]v1alpha1.Bridge) error {
	var result *multierror.Error

	hasSubs := len(*subscriptions) > 0
	hasBridges := len(*bridges) > 0
	if !hasSubs && !hasBridges {
		return errors.New("no subscriptions or bridges provided, at least one subscription or bridge must be provided")
	}

	if hasSubs {
		for _, subscription := range *subscriptions {
			if len(subscription.Spec.Outputs) == 0 {
				result = multierror.Append(result, fmt.Errorf("subscription %s has no outputs", subscription.Name))
			}
		}
	}

	if hasBridges {
		tenantMap := make(map[string]struct{})
		for _, tenant := range *tenants {
			tenantMap[tenant.Name] = struct{}{}
		}

		for _, bridge := range *bridges {
			if _, sourceFound := tenantMap[bridge.Spec.SourceTenant]; !sourceFound {
				result = multierror.Append(result, fmt.Errorf("bridge: %s has a source tenant: %s that does not exist", bridge.Name, bridge.Spec.SourceTenant))
			}
			if _, targetFound := tenantMap[bridge.Spec.TargetTenant]; !targetFound {
				result = multierror.Append(result, fmt.Errorf("bridge: %s has a target tenant: %s that does not exist", bridge.Name, bridge.Spec.TargetTenant))
			}
		}
	}

	return result.ErrorOrNil()
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
