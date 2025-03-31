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

import (
	"fmt"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
)

type TransformProcessorStatement struct {
	Context    string               `json:"context"`
	Conditions []string             `json:"conditions"`
	Statements []string             `json:"statements"`
	ErrorMde   components.ErrorMode `json:"error_mode,omitempty"`
}

type TransformProcessor struct {
	ErrorMode        components.ErrorMode          `json:"error_mode,omitempty"`
	FlattenData      bool                          `json:"flatten_data,omitempty"`
	TraceStatements  []TransformProcessorStatement `json:"trace_statements,omitempty"`
	MetricStatements []TransformProcessorStatement `json:"metric_statements,omitempty"`
	LogStatements    []TransformProcessorStatement `json:"log_statements,omitempty"`
}

func GenerateTransformProcessorForTenant(tenant v1alpha1.Tenant) TransformProcessor {
	return TransformProcessor{
		ErrorMode:        components.ErrorMode(tenant.Spec.Transform.ErrorMode),
		FlattenData:      tenant.Spec.Transform.FlattenData,
		TraceStatements:  convertAPIStatements(tenant.Spec.Transform.TraceStatements),
		MetricStatements: convertAPIStatements(tenant.Spec.Transform.MetricStatements),
		LogStatements:    convertAPIStatements(tenant.Spec.Transform.LogStatements),
	}
}

func GenerateTransformProcessorForTenantPipeline(tenantName string, pipeline *otelv1beta1.Pipeline, tenants []v1alpha1.Tenant) {
	for _, tenant := range tenants {
		if tenant.Name == tenantName && tenant.Spec.Transform.Name != "" {
			pipeline.Processors = append(pipeline.Processors, fmt.Sprintf("transform/%s", tenant.Spec.Transform.Name))
		}
	}
}

func convertAPIStatements(APIStatements []v1alpha1.TransformStatement) []TransformProcessorStatement {
	statements := make([]TransformProcessorStatement, len(APIStatements))
	for i, statement := range APIStatements {
		statements[i] = TransformProcessorStatement{
			Context:    statement.Context,
			Conditions: statement.Conditions,
			Statements: statement.Statements,
			ErrorMde:   components.ErrorMode(statement.ErrorMode),
		}
	}

	return statements
}
