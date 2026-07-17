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

package receiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKubernetesEventsReceiver(t *testing.T) {
	tests := []struct {
		name                    string
		namespaces              []string
		selectFromAllNamespaces bool
		want                    map[string]any
	}{
		{
			name:                    "namespaced tenant",
			namespaces:              []string{"ns-a", "ns-b"},
			selectFromAllNamespaces: false,
			want: map[string]any{
				"namespaces": []string{"ns-a", "ns-b"},
			},
		},
		{
			name:                    "select from all namespaces",
			namespaces:              []string{"ns-a"},
			selectFromAllNamespaces: true,
			want:                    map[string]any{},
		},
		{
			name:                    "empty namespaces",
			namespaces:              []string{},
			selectFromAllNamespaces: false,
			want:                    map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateKubernetesEventsReceiver(tt.namespaces, tt.selectFromAllNamespaces)
			assert.Equal(t, tt.want, got)
		})
	}
}
