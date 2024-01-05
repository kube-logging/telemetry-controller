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
	_ "embed"
)

//go:embed otel_col_conf_test_fixtures/complex.yaml
var otelColTargetYaml string

// func TestOtelColConfComplex(t *testing.T) {
// 	// Required inputs
// 	inputCfg := OtelColConfigInput{
// 		TenantSubscriptionMap: map[string][]string{
// 			"A": {"S1", "S2"},
// 			"B": {"S1", "S2"},
// 		},
// 	}

// 	// IR
// 	generatedIR := inputCfg.ToIntermediateRepresentation()

// 	generatedIR.Receivers = map[string]any{
// 		"file/in": map[string]any{
// 			"path": "/dev/stdin",
// 		},
// 	}

// 	// Final YAML
// 	generatedYAML, err := generatedIR.ToYAML()
// 	if err != nil {
// 		t.Fatalf("YAML formatting failed, err=%v", err)
// 	}
// 	t.Logf(`the generated YAML is:
// ---
// %v
// ---`, generatedYAML)

// 	actualYAMLBytes, err := generatedIR.ToYAMLRepresentation()
// 	if err != nil {
// 		t.Fatalf("error %v", err)
// 	}
// 	actualYAML, err := generatedIR.ToYAML()
// 	if err != nil {
// 		t.Fatalf("error %v", err)
// 	}
// 	var actualUniversalMap map[string]any
// 	if err := yaml.Unmarshal(actualYAMLBytes, &actualUniversalMap); err != nil {
// 		t.Fatalf("error: %v", err)
// 	}

// 	var expectedUniversalMap map[string]any
// 	if err := yaml.Unmarshal([]byte(otelColTargetYaml), &expectedUniversalMap); err != nil {
// 		t.Fatalf("error: %v", err)
// 	}

// 	// use dyff for YAML comparison
// 	if diff := cmp.Diff(expectedUniversalMap, actualUniversalMap); diff != "" {
// 		t.Logf("mismatch:\n---%s\n---\n", diff)
// 	}

// 	if !reflect.DeepEqual(actualUniversalMap, expectedUniversalMap) {
// 		t.Logf(`yaml mismatch:
// expected=
// ---
// %s
// ---
// actual=
// ---
// %s
// ---`, otelColTargetYaml, actualYAML)
// 		t.Fatalf(`yaml marshaling failed
// expected=
// ---
// %v
// ---,
// actual=
// ---
// %v
// ---`,
// 			expectedUniversalMap, actualUniversalMap)
// 	}
// }
