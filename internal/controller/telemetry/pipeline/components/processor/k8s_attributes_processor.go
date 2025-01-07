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

package processor

import "github.com/kube-logging/telemetry-controller/internal/controller/telemetry/utils"

func GenerateDefaultKubernetesProcessor() map[string]any {
	type Source struct {
		Name string `json:"name,omitempty"`
		From string `json:"from,omitempty"`
	}

	defaultSources := []Source{
		{
			Name: "k8s.namespace.name",
			From: "resource_attribute",
		},
		{
			Name: "k8s.pod.name",
			From: "resource_attribute",
		},
	}

	defaultPodAssociation := []map[string]any{
		{"sources": defaultSources},
	}

	defaultMetadata := []string{
		"k8s.pod.name",
		"k8s.pod.uid",
		"k8s.deployment.name",
		"k8s.namespace.name",
		"k8s.node.name",
		"k8s.pod.start_time",
	}

	type FieldExtract struct {
		TagName  *string `json:"tag_name,omitempty"`
		Key      *string `json:"key,omitempty"`
		KeyRegex *string `json:"key_regex,omitempty"`
		From     *string `json:"from,omitempty"`
	}

	defaultLabels := []FieldExtract{
		{
			From:     utils.ToPtr("pod"),
			TagName:  utils.ToPtr("all_labels"),
			KeyRegex: utils.ToPtr(".*"),
		},
	}

	k8sProcessor := map[string]any{
		"auth_type":   "serviceAccount",
		"passthrough": false,
		"extract": map[string]any{
			"metadata": defaultMetadata,
			"labels":   defaultLabels,
			"annotations": []FieldExtract{
				{
					TagName: utils.ToPtr("exclude-tc"),
					Key:     utils.ToPtr("telemetry.kube-logging.dev/exclude"),
					From:    utils.ToPtr("pod"),
				},
				{
					TagName: utils.ToPtr("exclude-fluentbit"),
					Key:     utils.ToPtr("fluentbit.io/exclude"),
					From:    utils.ToPtr("pod"),
				},
			},
		},
		"pod_association": defaultPodAssociation,
	}

	return k8sProcessor
}
