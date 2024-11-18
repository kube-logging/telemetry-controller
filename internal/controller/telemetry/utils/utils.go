// Copyright © 2024 Kube logging authors
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

package utils

import (
	"slices"
	"strings"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
)

func SortNamespacedNames(names []v1alpha1.NamespacedName) {
	slices.SortFunc(names, func(a, b v1alpha1.NamespacedName) int {
		return strings.Compare(a.String(), b.String())
	})
}

func ToPtr[T any](v T) *T {
	return &v
}

func ToValue[T any](v *T) T {
	return *v
}
