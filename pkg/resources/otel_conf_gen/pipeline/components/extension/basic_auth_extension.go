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
	"regexp"
	"strings"

	"github.com/kube-logging/telemetry-controller/api/telemetry/v1alpha1"
	"github.com/kube-logging/telemetry-controller/pkg/resources/otel_conf_gen/pipeline/components"
)

const (
	defaultBasicAuthUsernameField = "username"
	defaultBasicAuthPasswordField = "password"
)

type BasicClientAuthConfig struct {
	// Username holds the username to use for client authentication.
	Username string `json:"username"`
	// Password holds the password to use for client authentication.
	Password string `json:"password"`
}

type BasicAuthExtensionConfig struct {
	ClientAuth BasicClientAuthConfig `json:"client_auth,omitempty"`
}

// BasicAuthEnvVarNames returns the env var names for basicauth username and password
// for the given output namespace and name.
func BasicAuthEnvVarNames(namespacedName v1alpha1.NamespacedName) (userVar, passVar string) {
	base := fmt.Sprintf("TC_BASICAUTH_%s_%s", sanitizeEnvVarSegment(namespacedName.Namespace), sanitizeEnvVarSegment(namespacedName.Name))
	return base + "_USERNAME", base + "_PASSWORD"
}

// BasicAuthSecretKeys returns the secret data keys for basicauth username and password.
func BasicAuthSecretKeys(namespacedName v1alpha1.NamespacedName) (userKey, passKey string) {
	base := fmt.Sprintf("basicauth_%s_%s", sanitizeSecretKeySegment(namespacedName.Namespace), sanitizeSecretKeySegment(namespacedName.Name))
	return base + "_username", base + "_password"
}

func GenerateBasicAuthExtensionsForOutput(output components.OutputWithSecretData) BasicAuthExtensionConfig {
	userVar, passVar := BasicAuthEnvVarNames(output.Output.NamespacedName())
	return BasicAuthExtensionConfig{
		ClientAuth: BasicClientAuthConfig{
			Username: fmt.Sprintf("${%s}", userVar),
			Password: fmt.Sprintf("${%s}", passVar),
		},
	}
}

var (
	nonAlphanumUpperRe = regexp.MustCompile(`[^A-Z0-9]`)
	nonAlphanumLowerRe = regexp.MustCompile(`[^a-z0-9]`)
)

func sanitizeEnvVarSegment(s string) string {
	return nonAlphanumUpperRe.ReplaceAllString(strings.ToUpper(s), "_")
}

func sanitizeSecretKeySegment(s string) string {
	return nonAlphanumLowerRe.ReplaceAllString(strings.ToLower(s), "_")
}

// BasicAuthSecretData returns the secret data map entries for basicauth credentials
// extracted from the output's secret data.
func BasicAuthSecretData(output components.OutputWithSecretData) map[string][]byte {
	var effectiveUsernameField, effectivePasswordField string
	if output.Output.Spec.Authentication.BasicAuth.UsernameField != "" {
		effectiveUsernameField = output.Output.Spec.Authentication.BasicAuth.UsernameField
	} else {
		effectiveUsernameField = defaultBasicAuthUsernameField
	}
	if output.Output.Spec.Authentication.BasicAuth.PasswordField != "" {
		effectivePasswordField = output.Output.Spec.Authentication.BasicAuth.PasswordField
	} else {
		effectivePasswordField = defaultBasicAuthPasswordField
	}

	data := make(map[string][]byte)
	userKey, passKey := BasicAuthSecretKeys(output.Output.NamespacedName())
	if u, ok := output.Secret.Data[effectiveUsernameField]; ok {
		data[userKey] = u
	}
	if p, ok := output.Secret.Data[effectivePasswordField]; ok {
		data[passKey] = p
	}

	return data
}
