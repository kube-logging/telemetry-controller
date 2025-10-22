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
	"time"

	"emperror.dev/errors"
	"github.com/cisco-open/operator-tools/pkg/reconciler"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/manager"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/validator"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model/state"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/utils"
)

const requeueDelayOnFailedTenant = 20 * time.Second

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
	collectorManager := &manager.CollectorManager{
		BaseManager: manager.NewBaseManager(r.Client, log.FromContext(ctx, "collector", req.Name)),
	}

	collector := &v1alpha1.Collector{}
	collectorManager.Info(fmt.Sprintf("getting collector: %q", req.Name))

	if err := collectorManager.Get(ctx, req.NamespacedName, collector); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	collector.Spec.SetDefaults()

	originalCollectorStatus := collector.Status
	collectorManager.Info(fmt.Sprintf("reconciling collector: %q", collector.Name))

	err := handleCollectorCreation(ctx, collectorManager, collector, r.Scheme)
	if err != nil {
		return r.handleCollectorReconcileError(ctx, &collectorManager.BaseManager, collector, err)
	}

	collector.Status.State = state.StateReady
	collector.ClearProblems()
	if !reflect.DeepEqual(originalCollectorStatus, collector.Status) {
		collectorManager.Info("collector status changed")
		if updateErr := r.Status().Update(ctx, collector); updateErr != nil {
			collectorManager.Error(errors.WithStack(updateErr), "failed updating collector status")
			return ctrl.Result{}, updateErr
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

// handleTenantReconcileError handles errors that occur during reconciliation steps
func (r *CollectorReconciler) handleCollectorReconcileError(ctx context.Context, baseManager *manager.BaseManager, collector *v1alpha1.Collector, err error) (ctrl.Result, error) {
	switch {
	case errors.Is(err, manager.ErrTenantFailed): // This error indicates that the tenant is in a failed state, and we should requeue after a delay.
		return ctrl.Result{RequeueAfter: requeueDelayOnFailedTenant}, nil

	case errors.Is(err, manager.ErrNoResources): // This error indicates that there are no resources to reconcile, which is not a failure state.
		return ctrl.Result{}, nil
	}

	collector.AddProblem(err.Error())
	collector.Status.State = state.StateFailed

	baseManager.Error(errors.WithStack(err), "failed reconciling collector", "collector", collector.Name)
	if updateErr := r.Status().Update(ctx, collector); updateErr != nil {
		baseManager.Error(errors.WithStack(updateErr), "failed updating collector status", "collector", collector.Name)
		return ctrl.Result{}, errors.Append(err, updateErr)
	}

	return ctrl.Result{}, err
}

func handleCollectorCreation(ctx context.Context, collectorManager *manager.CollectorManager, collector *v1alpha1.Collector, scheme *runtime.Scheme) error {
	collectorConfigInput, err := collectorManager.BuildConfigInputForCollector(ctx, collector)
	if err != nil {
		return fmt.Errorf("failed to build config input for collector %s: %w", collector.Name, err)
	}

	if err := collectorManager.ValidateConfigInput(collectorConfigInput); err != nil {
		if errors.Is(err, manager.ErrNoResources) {
			collectorManager.Info("no resources to reconcile for collector, skipping creation")
		}
		collectorManager.Error(errors.WithStack(err), "invalid otel config input")

		return err
	}

	otelConfig, additionalArgs := collectorConfigInput.AssembleConfig(ctx)
	if err := validator.ValidateAssembledConfig(otelConfig); err != nil {
		collectorManager.Error(errors.WithStack(err), "invalid otel config")

		return err
	}

	saName, err := collectorManager.ReconcileRBAC(collector, scheme)
	if err != nil {
		return fmt.Errorf("%+v", err)
	}

	otelCollector, collectorState := collectorManager.OtelCollector(collector, otelConfig, additionalArgs, collectorConfigInput.Tenants, collectorConfigInput.OutputsWithSecretData, saName.Name)
	if err := ctrl.SetControllerReference(collector, otelCollector, scheme); err != nil {
		return err
	}

	resourceReconciler := reconciler.NewReconcilerWith(collectorManager.Client, reconciler.WithLog(collectorManager.Logger))
	_, err = resourceReconciler.ReconcileResource(otelCollector, collectorState)
	if err != nil {
		collectorManager.Error(errors.WithStack(err), "failed reconciling collector")

		return err
	}

	tenantNames := []string{}
	for _, tenant := range collectorConfigInput.Tenants {
		tenantNames = append(tenantNames, tenant.Name)
	}
	collector.Status.Tenants = utils.NormalizeStringSlice(tenantNames)

	return nil
}
