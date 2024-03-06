// Copyright © 2024 Kube logging authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetry

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
	//+kubebuilder:scaffold:imports
)

var _ = Describe("Telemetry controller integration test", func() {
	const (
		timeout  = time.Second * 5
		interval = time.Millisecond * 250
	)

	Context("Deploying a telemetry pipeline", Ordered, func() {
		It("Namespace should exist beforehand ", func() {
			namespaces := []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tenant-1-workload",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tenant-1-ctrl",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tenant-2-all",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "collector",
					},
				},
			}

			for _, namespace := range namespaces {
				Expect(k8sClient.Create(ctx, &namespace)).Should(Succeed())
			}

		})

		It("Subscriptions should be created in the annotated namespaces", func() {
			ctx := context.Background()

			subscriptions := []v1alpha1.Subscription{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "subscription-example-1",
						Namespace: "tenant-1-ctrl",
					},
					Spec: v1alpha1.SubscriptionSpec{
						OTTL: "route()",
						Outputs: []v1alpha1.NamespacedName{
							{
								Name:      "otlp-test-output-1",
								Namespace: "collector",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "subscription-example-2",
						Namespace: "tenant-2-all",
					},
					Spec: v1alpha1.SubscriptionSpec{
						OTTL: "route()",
						Outputs: []v1alpha1.NamespacedName{
							{
								Name:      "otlp-test-output-2",
								Namespace: "collector",
							},
						},
					},
				},
			}

			for _, subscription := range subscriptions {
				Expect(k8sClient.Create(ctx, &subscription)).Should(Succeed())
			}
		})

		It("Tenants should be created", func() {
			tenants := []v1alpha1.Tenant{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tenant-1",
					},
					Spec: v1alpha1.TenantSpec{
						SubscriptionNamespaceSelectors: []metav1.LabelSelector{
							{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "tenant-1-ctrl",
								},
							},
						},
						LogSourceNamespaceSelectors: []metav1.LabelSelector{
							{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "tenant-1-workload",
								},
							},
						},
					},
					Status: v1alpha1.TenantStatus{
						Subscriptions:       []v1alpha1.NamespacedName{{Name: "asd", Namespace: "bsd"}},
						LogSourceNamespaces: []string{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tenant-2",
					},
					Spec: v1alpha1.TenantSpec{
						SubscriptionNamespaceSelectors: []metav1.LabelSelector{
							{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "tenant-2-all",
								},
							},
						},
						LogSourceNamespaceSelectors: []metav1.LabelSelector{
							{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "tenant-2-all",
								},
							},
						},
					},
					Status: v1alpha1.TenantStatus{
						Subscriptions:       []v1alpha1.NamespacedName{{Name: "asd", Namespace: "bsd"}},
						LogSourceNamespaces: []string{},
					},
				},
			}

			for _, tenant := range tenants {
				Expect(k8sClient.Create(ctx, &tenant)).Should(Succeed())
			}
		})

		It("Outputs should be created", func() {

			outputs := []v1alpha1.OtelOutput{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "otlp-test-output",
						Namespace: "collector",
					},
					Spec: v1alpha1.OtelOutputSpec{
						OTLP: v1alpha1.OTLPgrpc{
							ClientConfig: v1alpha1.ClientConfig{
								Endpoint: "receiver-collector.example-tenant-ns.svc.cluster.local:4317",
								TLSSetting: v1alpha1.TLSClientSetting{
									Insecure: true,
								},
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "otlp-test-output-2",
						Namespace: "collector",
					},
					Spec: v1alpha1.OtelOutputSpec{
						OTLP: v1alpha1.OTLPgrpc{
							ClientConfig: v1alpha1.ClientConfig{
								Endpoint: "receiver-collector.example-tenant-ns.svc.cluster.local:4317",
								TLSSetting: v1alpha1.TLSClientSetting{
									Insecure: true,
								},
							},
						},
					},
				},
			}

			for _, output := range outputs {
				Expect(k8sClient.Create(ctx, &output)).Should(Succeed())
			}
		})

		It("Collector should be created", func() {
			collector := v1alpha1.Collector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example-collector",
				},
				Spec: v1alpha1.CollectorSpec{
					TenantSelector: metav1.LabelSelector{
						MatchLabels:      map[string]string{},
						MatchExpressions: []metav1.LabelSelectorRequirement{},
					},
					ControlNamespace: "collector",
				},
			}
			Expect(k8sClient.Create(ctx, &collector)).Should(Succeed())
		})

	})
	When("The controller reconciles based on deployed resources", Ordered, func() {

		It("RBAC resources should be reconciled by controller", func() {

			createdServiceAccount := &v1.ServiceAccount{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "collector", Name: "example-collector-sa"}, createdServiceAccount)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdServiceAccount.OwnerReferences[0].Name).To(Equal("example-collector"))

			createdClusterRole := &rbacv1.ClusterRole{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "example-collector-pod-association-reader"}, createdClusterRole)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdClusterRole.OwnerReferences[0].Name).To(Equal("example-collector"))

			createdClusterRoleBinding := &rbacv1.ClusterRoleBinding{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "example-collector-crb"}, createdClusterRoleBinding)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdClusterRoleBinding.OwnerReferences[0].Name).To(Equal("example-collector"))

		})

		It("OpentelemetryCollector resource should be reconciled by controller", func() {

			createdOtelCollector := &otelv1alpha1.OpenTelemetryCollector{}

			expectedConfig, err := os.ReadFile(path.Join("envtest_testdata", "config.yaml"))
			Expect(err).NotTo(HaveOccurred())
			exptectedOtelCollector := &otelv1alpha1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "otelcollector-example-collector",
					Namespace: "collector",
				},
				Spec: otelv1alpha1.OpenTelemetryCollectorSpec{
					Config: string(expectedConfig),
				},
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "collector", Name: "otelcollector-example-collector"}, createdOtelCollector)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			expectedMap := map[string]any{}
			err = yaml.Unmarshal([]byte(exptectedOtelCollector.Spec.Config), expectedMap)
			Expect(err).NotTo(HaveOccurred())

			createdMap := map[string]any{}
			err = yaml.Unmarshal([]byte(createdOtelCollector.Spec.Config), createdMap)
			Expect(err).NotTo(HaveOccurred())

			Expect(createdMap).To(Equal(expectedMap))

		})
	})
})
