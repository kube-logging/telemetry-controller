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

package resources

import (
	"github.com/kube-logging/telemetry-controller/internal/controller/telemetry/resources/state"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StatusTenantReferenceField       = ".status.tenant"
	BridgeSourceTenantReferenceField = ".spec.sourceTenant"
	BridgeTargetTenantReferenceField = ".spec.targetTenant"
	TenantNameField                  = ".metadata.name"
)

// ResourceOwnedByTenant is an interface that must be implemented by resources that can be owned by a tenant
type ResourceOwnedByTenant interface {
	client.Object
	GetTenant() string
	SetTenant(tenant string)
	GetState() state.State
	SetState(state state.State)
}

// ResourceList is an interface for Kubernetes list types
type ResourceList interface {
	client.ObjectList
	GetItems() []ResourceOwnedByTenant
}
