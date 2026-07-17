// Copyright © 2026 Kube logging authors
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

package receiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

func TestGenerateKubernetesEventsReceiver(t *testing.T) {
	tests := []struct {
		name         string
		namespaces   []string
		dryRunMode   bool
		tenant       v1alpha1.Tenant
		eventsToLogs v1alpha1.EventsToLogs
		want         map[string]any
	}{
		{
			name:       "namespaced tenant",
			namespaces: []string{"ns-a", "ns-b"},
			tenant: v1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
			},
			want: map[string]any{
				"namespaces": []string{"ns-a", "ns-b"},
			},
		},
		{
			name:       "select from all namespaces",
			namespaces: []string{"ns-a"},
			tenant: v1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
				Spec:       v1alpha1.TenantSpec{SelectFromAllNamespaces: true},
			},
			want: map[string]any{},
		},
		{
			name:       "empty namespaces",
			namespaces: []string{},
			tenant: v1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
			},
			want: map[string]any{},
		},
		{
			name:       "receiver tuning fields are set",
			namespaces: []string{"ns-a"},
			tenant: v1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
			},
			eventsToLogs: v1alpha1.EventsToLogs{
				KubeAPIQPS:    new(int32(10)),
				KubeAPIBurst:  new(int32(20)),
				DedupInterval: new("5m"),
			},
			want: map[string]any{
				"namespaces":     []string{"ns-a"},
				"kube_api_qps":   int32(10),
				"kube_api_burst": int32(20),
				"dedup_interval": "5m",
			},
		},
		{
			name:       "storage is wired when file storage is enabled",
			namespaces: []string{"ns-a"},
			tenant: v1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
				Spec: v1alpha1.TenantSpec{
					PersistenceConfig: v1alpha1.PersistenceConfig{EnableFileStorage: true},
				},
			},
			want: map[string]any{
				"namespaces": []string{"ns-a"},
				"storage":    "file_storage/tenant-a",
			},
		},
		{
			name:       "storage is omitted in dry-run mode",
			namespaces: []string{"ns-a"},
			dryRunMode: true,
			tenant: v1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"},
				Spec: v1alpha1.TenantSpec{
					PersistenceConfig: v1alpha1.PersistenceConfig{EnableFileStorage: true},
				},
			},
			want: map[string]any{
				"namespaces": []string{"ns-a"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateKubernetesEventsReceiver(tt.namespaces, tt.dryRunMode, tt.tenant, tt.eventsToLogs)
			assert.Equal(t, tt.want, got)
		})
	}
}
