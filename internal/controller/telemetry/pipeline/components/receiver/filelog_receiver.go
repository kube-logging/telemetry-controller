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

package receiver

import "fmt"

func GenerateDefaultKubernetesReceiver(namespaces []string) map[string]any {
	// TODO: fix parser-crio
	operators := []map[string]any{
		{
			"type": "router",
			"id":   "get-format",
			"routes": []map[string]string{
				{
					"output": "parser-docker",
					"expr":   `body matches "^\\{"`,
				},
				{
					"output": "parser-containerd",
					"expr":   `body matches "^[^ Z]+Z"`,
				},
			},
		},
		{
			"type":   "regex_parser",
			"id":     "parser-containerd",
			"regex":  `^(?P<time>[^ ^Z]+Z) (?P<stream>stdout|stderr) (?P<logtag>[^ ]*) ?(?P<log>.*)$`,
			"output": "extract_metadata_from_filepath",
			"timestamp": map[string]string{
				"parse_from": "attributes.time",
				"layout":     "%Y-%m-%dT%H:%M:%S.%LZ",
			},
		},
		{
			"type":   "json_parser",
			"id":     "parser-docker",
			"output": "extract_metadata_from_filepath",
			"timestamp": map[string]string{
				"parse_from": "attributes.time",
				"layout":     "%Y-%m-%dT%H:%M:%S.%LZ",
			},
		},
		{
			"type":       "regex_parser",
			"id":         "extract_metadata_from_filepath",
			"regex":      `^.*\/(?P<namespace>[^_]+)_(?P<pod_name>[^_]+)_(?P<uid>[a-f0-9-]+)\/(?P<container_name>[^\/]+)\/(?P<restart_count>\d+)\.log$`,
			"parse_from": `attributes["log.file.path"]`,
			"cache": map[string]int{
				"size": 128,
			},
		},
		{
			"type": "move",
			"from": "attributes.log",
			"to":   "body",
		},
		{
			"type": "move",
			"from": "attributes.stream",
			"to":   `attributes["log.iostream"]`,
		},
		{
			"type": "move",
			"from": "attributes.container_name",
			"to":   `resource["k8s.container.name"]`,
		},
		{
			"type": "move",
			"from": "attributes.namespace",
			"to":   `resource["k8s.namespace.name"]`,
		},
		{
			"type": "move",
			"from": "attributes.pod_name",
			"to":   `resource["k8s.pod.name"]`,
		},
		{
			"type": "move",
			"from": "attributes.restart_count",
			"to":   `resource["k8s.container.restart_count"]`,
		},
		{
			"type": "move",
			"from": "attributes.uid",
			"to":   `resource["k8s.pod.uid"]`,
		},
	}

	includeList := make([]string, 0, len(namespaces))
	if len(namespaces) > 0 {
		for _, ns := range namespaces {
			include := fmt.Sprintf("/var/log/pods/%s_*/*/*.log", ns)
			includeList = append(includeList, include)
		}
	} else {
		includeList = append(includeList, "/var/log/pods/*/*/*.log")
	}

	k8sReceiver := map[string]any{
		"include":           includeList,
		"exclude":           []string{"/var/log/pods/*/otc-container/*.log"},
		"start_at":          "end",
		"include_file_path": true,
		"include_file_name": false,
		"operators":         operators,
		"retry_on_failure": map[string]any{
			"enabled":          true,
			"max_elapsed_time": 0,
		},
	}

	return k8sReceiver
}
