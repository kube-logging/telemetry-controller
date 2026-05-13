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

// NodeNameEnvVar is the downward API environment variable that the k8sattributes
// processor reads to scope its informers to a single node.
const NodeNameEnvVar = "KUBE_NODE_NAME"

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

	// Metadata fields are scoped to what the existing collector ClusterRole can
	// resolve: pods, namespaces, nodes (core) and replicasets (apps).
	defaultMetadata := []string{
		"k8s.namespace.name",
		"k8s.node.name",
		"k8s.pod.name",
		"k8s.pod.uid",
		"k8s.pod.ip",
		"k8s.pod.start_time",
		"k8s.replicaset.name",
		"k8s.deployment.name",
		"k8s.cluster.uid",
		"k8s.container.name",
		"container.image.name",
		"container.image.tag",
	}

	type FieldExtract struct {
		TagName  *string `json:"tag_name,omitempty"`
		Key      *string `json:"key,omitempty"`
		KeyRegex *string `json:"key_regex,omitempty"`
		From     *string `json:"from,omitempty"`
	}

	defaultLabels := []FieldExtract{
		{
			From:     new("pod"),
			TagName:  new("all_labels"),
			KeyRegex: new(".*"),
		},
	}

	k8sProcessor := map[string]any{
		"auth_type":   "serviceAccount",
		"passthrough": false,
		// Filter pods to those scheduled on the same node as the collector pod.
		"filter": map[string]any{
			"node_from_env_var": NodeNameEnvVar,
		},
		"extract": map[string]any{
			"metadata": defaultMetadata,
			// Derive `k8s.deployment.name` from the owning ReplicaSet name
			// instead of watching Deployment resources. Cuts API server load
			// and informer memory.
			"deployment_name_from_replicaset": true,
			"labels":                          defaultLabels,
			"annotations": []FieldExtract{
				{
					TagName: new("exclude-tc"),
					Key:     new("telemetry.kube-logging.dev/exclude"),
					From:    new("pod"),
				},
				{
					TagName: new("exclude-fluentbit"),
					Key:     new("fluentbit.io/exclude"),
					From:    new("pod"),
				},
			},
		},
		"pod_association": defaultPodAssociation,
	}

	return k8sProcessor
}
