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

package extension

import (
	"fmt"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
)

const defaultBearerAuthTokenField = "token"

type BearerTokenAuthExtensionConfig struct {
	BearerToken string `json:"token,omitempty"`
}

// BearerAuthEnvVarName returns the env var name for bearerauth token
// for the given output namespace and name.
func BearerAuthEnvVarName(namespacedName v1alpha1.NamespacedName) string {
	return fmt.Sprintf("TC_BEARERAUTH_%s_%s_TOKEN", sanitizeEnvVarSegment(namespacedName.Namespace), sanitizeEnvVarSegment(namespacedName.Name))
}

// BearerAuthSecretKey returns the secret data key for bearerauth token.
func BearerAuthSecretKey(namespacedName v1alpha1.NamespacedName) string {
	return fmt.Sprintf("bearerauth_%s_%s_token", sanitizeSecretKeySegment(namespacedName.Namespace), sanitizeSecretKeySegment(namespacedName.Name))
}

func GenerateBearerAuthExtensionsForOutput(output components.OutputWithSecretData) BearerTokenAuthExtensionConfig {
	return BearerTokenAuthExtensionConfig{
		BearerToken: fmt.Sprintf("${%s}", BearerAuthEnvVarName(output.Output.NamespacedName())),
	}
}

// BearerAuthSecretData returns the secret data map entries for bearer token credentials
// extracted from the output's secret data.
func BearerAuthSecretData(output components.OutputWithSecretData) map[string][]byte {
	var effectiveTokenField string
	if output.Output.Spec.Authentication.BearerAuth.TokenField != "" {
		effectiveTokenField = output.Output.Spec.Authentication.BearerAuth.TokenField
	} else {
		effectiveTokenField = defaultBearerAuthTokenField
	}

	data := make(map[string][]byte)
	if t, ok := output.Secret.Data[effectiveTokenField]; ok {
		data[BearerAuthSecretKey(output.Output.NamespacedName())] = t
	}

	return data
}
