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

	"emperror.dev/errors"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/sdk/model"
)

// BridgeManager manages bridge resources
type BridgeManager struct {
	BaseManager
}

func (b *BridgeManager) GetBridgesForTenant(ctx context.Context, tenantName string) (bridgesOwned []v1alpha1.Bridge, err error) {
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(model.BridgeSourceTenantReferenceField, tenantName),
	}
	sourceBridge, err := b.getBridges(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	listOpts = &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(model.BridgeTargetTenantReferenceField, tenantName),
	}
	targetBridge, err := b.getBridges(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	bridges := append(sourceBridge, targetBridge...)
	for _, bridge := range bridges {
		if bridge.Spec.SourceTenant == tenantName || bridge.Spec.TargetTenant == tenantName {
			bridgesOwned = append(bridgesOwned, bridge)
		}
	}

	return
}

func (b *BridgeManager) getBridges(ctx context.Context, listOpts *client.ListOptions) ([]v1alpha1.Bridge, error) {
	var bridges v1alpha1.BridgeList
	if err := b.List(ctx, &bridges, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return bridges.Items, nil
}

func (b *BridgeManager) getTenants(ctx context.Context, listOpts *client.ListOptions) ([]v1alpha1.Tenant, error) {
	var tenants v1alpha1.TenantList
	if err := b.List(ctx, &tenants, listOpts); client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	return tenants.Items, nil
}

func (b *BridgeManager) ValidateBridgeConnection(ctx context.Context, tenantName string, bridge *v1alpha1.Bridge) error {
	for _, tenantReference := range []string{bridge.Spec.SourceTenant, bridge.Spec.TargetTenant} {
		if tenantReference != tenantName {
			listOpts := &client.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(model.TenantNameField, tenantReference),
			}
			tenant, err := b.getTenants(ctx, listOpts)
			if err != nil {
				return err
			}
			if len(tenant) == 0 {
				return errors.Errorf("bridge %s has a dangling tenant reference %s", bridge.Name, tenantReference)
			}
		}
	}

	return nil
}

func GetBridgeNamesFromBridges(bridges []v1alpha1.Bridge) []string {
	bridgeNames := make([]string, len(bridges))
	for i, bridge := range bridges {
		bridgeNames[i] = bridge.Name
	}

	return bridgeNames
}
