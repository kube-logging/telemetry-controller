/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	_ "embed"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-yaml/yaml"
)

// TODO move this to its appropiate place
type OtelColConfigInput struct {
	Tenants []string
}

func (cfgInput *OtelColConfigInput) generateExporters() map[string]interface{} {
	var result = make(map[string]interface{})
	common_prefix := "/foo"

	// Create file outputs based on tenant names
	for _, tenantName := range cfgInput.Tenants {
		fileOutputName := fmt.Sprintf("file/%s", tenantName)
		fileOutputPath := fmt.Sprintf("%s/%s", common_prefix, tenantName)

		result[fileOutputName] = map[string]interface{}{
			"path": fileOutputPath,
		}
	}

	return result
}

func generatePipeline(receivers, processors, exporters []string) Pipeline {
	var result = Pipeline{}

	result.Receivers = receivers
	result.Processors = processors
	result.Exporters = exporters

	return result
}

func (cfgInput *OtelColConfigInput) ToIntermediateRepresentation() OtelColConfigIR {
	result := OtelColConfigIR{}

	// Get file outputs based tenant names
	result.Exporters = cfgInput.generateExporters()

	// Fill the service field
	// Add pipelines
	dummyReceivers := []string{"file/in"}
	exporters := []string{}
	for exporter := range result.Exporters {
		exporters = append(exporters, exporter)
	}
	result.Services.Pipelines.Metrics = generatePipeline(dummyReceivers, []string{}, exporters)

	return result
}

type Pipeline struct {
	Receivers  []string `yaml:"receivers,omitempty,flow"`
	Processors []string `yaml:"processors,omitempty,flow"`
	Exporters  []string `yaml:"exporters,omitempty,flow"`
}

type Pipelines struct {
	Traces  Pipeline `yaml:"traces,omitempty"`
	Metrics Pipeline `yaml:"metrics,omitempty"`
	Logs    Pipeline `yaml:"logs,omitempty"`
}

type Services struct {
	Extensions map[string]interface{} `yaml:"extensions,omitempty"`
	Pipelines  Pipelines              `yaml:"pipelines,omitempty"`
	Telemetry  map[string]interface{} `yaml:"telemetry,omitempty"`
}

type OtelColConfigIR struct {
	Receivers map[string]interface{} `yaml:"receivers,omitempty"`
	Exporters map[string]interface{} `yaml:"exporters,omitempty"`
	Services  Services               `yaml:"service,omitempty"`
}

func (cfg *OtelColConfigIR) ToYAML() (string, error) {
	bytes, err := cfg.ToYAMLRepresentation()
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (cfg *OtelColConfigIR) ToYAMLRepresentation() ([]byte, error) {
	if cfg != nil {
		bytes, err := yaml.Marshal(cfg)
		if err != nil {
			return []byte{}, err
		}
		return bytes, nil
	}
	return []byte{}, nil
}

//go:embed otel_col_conf_test_fixtures/minimal.yaml
var otelColMinimalYaml string

func TestTemplateMinimal(t *testing.T) {
	inputCfg := OtelColConfigInput{
		Tenants: []string{"tenantA", "tenantB"},
	}
	generatedIR := inputCfg.ToIntermediateRepresentation()

	generatedIR.Receivers = map[string]interface{}{
		"file/in": map[string]interface{}{
			"path": "/dev/stdin",
		},
	}

	generatedYAML, err := generatedIR.ToYAML()
	if err != nil {
		t.Fatalf("YAML formatting failed, err=%v", err)
	}
	t.Logf(`the generated YAML is:
---
%v
---`, generatedYAML)

	actualYAMLBytes, err := generatedIR.ToYAMLRepresentation()
	if err != nil {
		t.Fatalf("error %v", err)
	}
	var actualUniversalMap map[string]interface{}
	if err := yaml.Unmarshal(actualYAMLBytes, &actualUniversalMap); err != nil {
		t.Fatalf("error: %v", err)
	}

	var expectedUniversalMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(otelColMinimalYaml), &expectedUniversalMap); err != nil {
		t.Fatalf("error: %v", err)
	}

	if !reflect.DeepEqual(actualUniversalMap, expectedUniversalMap) {
		t.Fatalf(`yaml marshaling failed
expected=
---
%v
---,
actual=
---
%v
---`,
			expectedUniversalMap, actualUniversalMap)
	}
}
