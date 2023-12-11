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

	"gopkg.in/yaml.v3"
)

// TODO move this to its appropiate place
type OtelColConfigInput struct {
	// Input
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

type RoutingConnectorTableItem struct {
	Statement string   `yaml:"statement"`
	Pipelines []string `yaml:"pipelines,flow"`
}

type RoutingConnector struct {
	Name             string                      `yaml:"-,omitempty"`
	DefaultPipelines []string                    `yaml:"default_pipelines"`
	Table            []RoutingConnectorTableItem `yaml:"table"`
}

func (rc *RoutingConnector) AddRoutingConnectorTableElem(attribute string, value string, pipelines []string) {
	newTableItem := RoutingConnectorTableItem{
		Statement: fmt.Sprintf(`'route() where attributes[%q]' == %q`, attribute, value),
		Pipelines: pipelines,
	}
	rc.Table = append(rc.Table, newTableItem)
}

func GenerateRoutingConnector(name string, defaultPipelines []string) RoutingConnector {
	result := RoutingConnector{}

	result.DefaultPipelines = defaultPipelines

	return result
}

func (cfgInput *OtelColConfigInput) ToIntermediateRepresentation(receivers []string) OtelColConfigIR {
	result := OtelColConfigIR{}

	// Get file outputs based tenant names
	result.Exporters = cfgInput.generateExporters()

	// Fill the service field
	// Add pipelines
	exporters := []string{}
	for exporter := range result.Exporters {
		exporters = append(exporters, exporter)
	}
	result.Services.Pipelines.Metrics = generatePipeline(receivers, []string{}, exporters)
	result.Connectors = map[string]interface{}{}

	return result
}

type Pipeline struct {
	Receivers  []string `yaml:"receivers,omitempty,flow"`
	Processors []string `yaml:"processors,omitempty,flow"`
	Exporters  []string `yaml:"exporters,omitempty,flow"`
}

type Pipelines struct {
	Traces         Pipeline            `yaml:"traces,omitempty"`
	Metrics        Pipeline            `yaml:"metrics,omitempty"`
	Logs           Pipeline            `yaml:"logs,omitempty"`
	NamedPipelines map[string]Pipeline `yaml:",inline,omitempty"`
}

type Services struct {
	Extensions map[string]interface{} `yaml:"extensions,omitempty"`
	Pipelines  Pipelines              `yaml:"pipelines,omitempty"`
	Telemetry  map[string]interface{} `yaml:"telemetry,omitempty"`
}

type OtelColConfigIR struct {
	Receivers  map[string]interface{} `yaml:"receivers,omitempty"`
	Exporters  map[string]interface{} `yaml:"exporters,omitempty"`
	Services   Services               `yaml:"service,omitempty"`
	Connectors map[string]interface{} `yaml:"connectors,omitempty"`
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

func TestOtelColConfMinimal(t *testing.T) {
	inputCfg := OtelColConfigInput{
		Tenants: []string{"tenantA", "tenantB"},
	}
	dummyReceivers := []string{"file/in"}
	generatedIR := inputCfg.ToIntermediateRepresentation(dummyReceivers)

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
	var actualYAML map[string]interface{}
	if err := yaml.Unmarshal(actualYAMLBytes, &actualYAML); err != nil {
		t.Fatalf("error: %v", err)
	}

	var expectedYAML map[string]interface{}
	if err := yaml.Unmarshal([]byte(otelColMinimalYaml), &expectedYAML); err != nil {
		t.Fatalf("error: %v", err)
	}

	if !reflect.DeepEqual(actualYAML, expectedYAML) {
		t.Fatalf(`yaml marshaling failed
expected=
---
%v
---,
actual=
---
%v
---`,
			expectedYAML, actualYAML)
	}
}

//go:embed otel_col_conf_test_fixtures/complex.yaml
var otelColTargetYaml string

func TestOtelColConfComplex(t *testing.T) {
	// Required inputs
	inputCfg := OtelColConfigInput{
		Tenants: []string{"tenantA", "tenantB"},
	}
	rc := GenerateRoutingConnector("routing/tenants", []string{"logs/default"})
	rc.AddRoutingConnectorTableElem("kubernetes.namespace.labels.tenant", "A", []string{"logs/tenant_A"})
	rc.AddRoutingConnectorTableElem("kubernetes.namespace.labels.tenant", "B", []string{})

	// IR
	generatedIR := inputCfg.ToIntermediateRepresentation([]string{})

	generatedIR.Receivers = map[string]interface{}{
		"file/in": map[string]interface{}{
			"path": "/dev/stdin",
		},
	}
	generatedIR.Connectors[rc.Name] = rc

	// Final YAML
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
	if err := yaml.Unmarshal([]byte(otelColTargetYaml), &expectedUniversalMap); err != nil {
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
