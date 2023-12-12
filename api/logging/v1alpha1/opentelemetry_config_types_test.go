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

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO move this to its appropiate place
type OtelColConfigInput struct {
	// Input
	// TODO: use Tenant struct here
	Tenants []Tenant
}

func (cfgInput *OtelColConfigInput) generateExporters() map[string]interface{} {
	var result = make(map[string]interface{})
	common_prefix := "/foo"

	// Create file outputs based on tenant names
	for _, tenant := range cfgInput.Tenants {
		fileOutputName := fmt.Sprintf("file/%s", tenant.Name)
		fileOutputPath := fmt.Sprintf("%s/%s", common_prefix, tenant.Name)

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
	Name             string                      `yaml:"-"`
	DefaultPipelines []string                    `yaml:"default_pipelines"`
	Table            []RoutingConnectorTableItem `yaml:"table"`
}

func (rc *RoutingConnector) AddRoutingConnectorTableElem(attribute string, value string, pipelines []string) {
	newTableItem := RoutingConnectorTableItem{
		Statement: fmt.Sprintf(`route() where attributes[%q] == %q`, attribute, value),
		Pipelines: pipelines,
	}
	rc.Table = append(rc.Table, newTableItem)
}

func GenerateRoutingConnector(name string, defaultPipelines []string) RoutingConnector {
	result := RoutingConnector{}

	result.DefaultPipelines = defaultPipelines
	result.Name = name

	return result
}

func (cfgInput *OtelColConfigInput) ToIntermediateRepresentation(connectors map[string]interface{}) OtelColConfigIR {
	result := OtelColConfigIR{}

	// Get file outputs based tenant names
	result.Exporters = cfgInput.generateExporters()

	result.Connectors = map[string]interface{}{}
	result.Services.Pipelines.NamedPipelines = make(map[string]Pipeline)

	for connectorName, connector := range connectors {
		result.Connectors[connectorName] = connector
	}

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
		Tenants: []Tenant{
			Tenant{
				ObjectMeta: v1.ObjectMeta{
					Name: "tenantA",
				},
			},
			Tenant{
				ObjectMeta: v1.ObjectMeta{
					Name: "tenantB",
				},
			},
		},
	}
	dummyReceivers := []string{"file/in"}
	generatedIR := inputCfg.ToIntermediateRepresentation(map[string]interface{}{})
	generatedIR.Services.Pipelines.Metrics = generatePipeline(dummyReceivers, []string{}, []string{"file/tenantA", "file/tenantB"})

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
		Tenants: []Tenant{
			Tenant{
				ObjectMeta: v1.ObjectMeta{
					Name: "tenantA",
				},
			},
			Tenant{
				ObjectMeta: v1.ObjectMeta{
					Name: "tenantB",
				},
			},
		},
	}
	rcTenants := GenerateRoutingConnector("routing/tenants", []string{"logs/default"})
	rcTenants.AddRoutingConnectorTableElem("kubernetes.namespace.labels.tenant", "A", []string{"logs/tenant_A"})
	rcTenants.AddRoutingConnectorTableElem("kubernetes.namespace.labels.tenant", "B", []string{})

	rcTenantA := GenerateRoutingConnector("routing/tenant_A_subscriptions", []string{"logs/tenant_A_default"})
	rcTenantA.AddRoutingConnectorTableElem("kubernetes.labels.app", "app1", []string{"logs/tenant_A_subscription_A_S1"})
	rcTenantA.AddRoutingConnectorTableElem("kubernetes.labels.app", "app2", []string{"logs/tenant_A_subscription_A_S2"})

	rcTenantB := GenerateRoutingConnector("routing/tenant_B_subscriptions", []string{"logs/tenant_B_default"})

	connectors := make(map[string]interface{})
	connectors[rcTenants.Name] = rcTenants
	connectors[rcTenantA.Name] = rcTenantA
	connectors[rcTenantB.Name] = rcTenantB

	// IR
	generatedIR := inputCfg.ToIntermediateRepresentation(connectors)

	generatedIR.Receivers = map[string]interface{}{
		"file/in": map[string]interface{}{
			"path": "/dev/stdin",
		},
	}
	generatedIR.Connectors[rcTenants.Name] = rcTenants
	generatedIR.Services.Pipelines.NamedPipelines["logs/all"] = generatePipeline([]string{"file/in"}, []string{"kubernetes"}, []string{"routing/tenants"})
	generatedIR.Services.Pipelines.NamedPipelines["logs/tenant_A"] = generatePipeline([]string{"routing/tenants"}, []string{"add tenant label"}, []string{"routing/tenant_A_subscriptions"})
	generatedIR.Services.Pipelines.NamedPipelines["logs/tenant_A_subscription_A_S1"] = generatePipeline([]string{"routing/tenant_A_subscriptions"}, []string{"add subscription label"}, []string{"file/tenantA", "file/tenantB"})
	generatedIR.Services.Pipelines.NamedPipelines["logs/tenant_A_subscription_A_S2"] = generatePipeline([]string{"routing/tenant_A_subscriptions"}, []string{}, []string{"file/tenantA"})
	generatedIR.Services.Pipelines.NamedPipelines["logs/tenant_B"] = generatePipeline([]string{"routing/tenants"}, []string{}, []string{"routing/tenant_B_subscriptions"})

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

	// use dyff for YAML comparison
	if diff := cmp.Diff(expectedUniversalMap, actualUniversalMap); diff != "" {
		t.Logf("mismatch:\n---%s\n---\n", diff)
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
