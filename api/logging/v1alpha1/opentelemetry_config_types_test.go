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

package v1alpha1

import (
	_ "embed"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

// TODO move this to its appropiate place
type OtelColConfigInput struct {
	// Input
	// TODO: use Tenant struct here
	//Tenants []Tenant

	// Subscriptions map, where the key is the Tenants'name
	TenantSubscriptionMap map[string][]string
}

func (cfgInput *OtelColConfigInput) generateExporters() map[string]interface{} {
	var result = make(map[string]interface{})
	common_prefix := "/foo"

	// Create file outputs based on tenant names
	for tenantName, subscriptions := range cfgInput.TenantSubscriptionMap {
		for _, subsubscription := range subscriptions {
			fileOutputName := fmt.Sprintf("file/tenant_%s_%s", tenantName, subsubscription)
			fileOutputPath := fmt.Sprintf("%s/tenant_%s/%s", common_prefix, tenantName, subsubscription)

			result[fileOutputName] = map[string]interface{}{
				"path": fileOutputPath,
			}
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
	DefaultPipelines []string                    `yaml:"default_pipelines,flow"`
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

func generateRootRoutingConnector(tenantNames []string) RoutingConnector {
	// Generate routing table's first hop that will sort it's input by tenant name
	defaultRc := GenerateRoutingConnector("routing/tenants", []string{"logs/default"})
	for _, tenantName := range tenantNames {
		defaultRc.AddRoutingConnectorTableElem("kubernetes.namespace.labels.tenant", tenantName, []string{fmt.Sprintf("logs/tenant_%s", tenantName)})
	}
	return defaultRc
}

func generateRoutingConnectorForTenantsSubscription(tenantName string, subscriptions []string) RoutingConnector {
	rcName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName)
	rcDefaultPipelines := []string{fmt.Sprintf("logs/tenant_%s_default", tenantName)}
	rc := GenerateRoutingConnector(rcName, rcDefaultPipelines)

	for _, subscriptionName := range subscriptions {
		pipeline := fmt.Sprintf("logs/tenant_%s_subscription_%s", tenantName, subscriptionName)
		rc.AddRoutingConnectorTableElem("kubernetes.labels.app", subscriptionName, []string{pipeline})
	}

	return rc
}

func (cfgInput *OtelColConfigInput) generateConnectors() map[string]interface{} {
	var connectors = make(map[string]interface{})

	tenantNames := []string{}
	for tenantName := range cfgInput.TenantSubscriptionMap {
		tenantNames = append(tenantNames, tenantName)

	}
	rootRoutingConnector := generateRootRoutingConnector(tenantNames)
	connectors[rootRoutingConnector.Name] = rootRoutingConnector

	for _, tenantName := range tenantNames {
		rc := generateRoutingConnectorForTenantsSubscription(tenantName, cfgInput.TenantSubscriptionMap[tenantName])
		connectors[rc.Name] = rc
	}

	return connectors

}

func generateRootPipeline() Pipeline {
	return generatePipeline([]string{"file/in"}, []string{"kubernetes"}, []string{"routing/tenants"})
}

func (cfgInput *OtelColConfigInput) generateNamedPipelines() map[string]Pipeline {
	var namedPipelines = make(map[string]Pipeline)

	namedPipelines["logs/all"] = generateRootPipeline()

	tenantNames := []string{}
	for tenantName := range cfgInput.TenantSubscriptionMap {
		tenantNames = append(tenantNames, tenantName)
	}

	for _, tenantName := range tenantNames {
		// Generate a pipeline for the tenant
		tenantPipelineName := fmt.Sprintf("logs/tenant_%s", tenantName)
		tenantRoutingName := fmt.Sprintf("routing/tenant_%s_subscriptions", tenantName)
		namedPipelines[tenantPipelineName] = generatePipeline([]string{"routing/tenants"}, []string{"add tenant label"}, []string{tenantRoutingName})

		// Generate pipelines for the subscriptions for the tenant
		for _, subscription := range cfgInput.TenantSubscriptionMap[tenantName] {
			tenantSubscriptionPipelineName := fmt.Sprintf("%s_subscription_%s", tenantPipelineName, subscription)
			tenantSubscriptionPipelineExporterName := fmt.Sprintf("file/tenant_%s_%s", tenantName, subscription)
			namedPipelines[tenantSubscriptionPipelineName] = generatePipeline([]string{tenantRoutingName}, []string{"add subscription label"}, []string{tenantSubscriptionPipelineExporterName})
		}
	}

	return namedPipelines

}

func (cfgInput *OtelColConfigInput) ToIntermediateRepresentation() OtelColConfigIR {
	result := OtelColConfigIR{}

	// Get file outputs based tenant names
	result.Exporters = cfgInput.generateExporters()

	result.Connectors = cfgInput.generateConnectors()
	result.Services.Pipelines.NamedPipelines = make(map[string]Pipeline)

	result.Services.Pipelines.NamedPipelines = cfgInput.generateNamedPipelines()

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
	Connectors map[string]interface{} `yaml:"connectors,omitempty"`
	Services   Services               `yaml:"service,omitempty"`
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
	t.Skip()
	inputCfg := OtelColConfigInput{
		TenantSubscriptionMap: map[string][]string{
			"tenantA": {},
			"tenantB": {},
		},
	}
	dummyReceivers := []string{"file/in"}
	generatedIR := inputCfg.ToIntermediateRepresentation()
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
		TenantSubscriptionMap: map[string][]string{
			"A": {"S1", "S2"},
			"B": {"S1", "S2"},
		},
	}

	// IR
	generatedIR := inputCfg.ToIntermediateRepresentation()

	generatedIR.Receivers = map[string]interface{}{
		"file/in": map[string]interface{}{
			"path": "/dev/stdin",
		},
	}

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
	actualYAML, err := generatedIR.ToYAML()
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
		t.Logf(`yaml mismatch:
expected=
---
%s
---
actual=
---
%s
---`, otelColTargetYaml, actualYAML)
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
