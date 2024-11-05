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

import "github.com/kube-logging/telemetry-controller/internal/controller/telemetry/pipeline/components"

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

func GenerateBasicAuthExtensionsForOutput(output components.OutputWithSecretData) BasicAuthExtensionConfig {
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

	config := BasicAuthExtensionConfig{}
	if u, ok := output.Secret.Data[effectiveUsernameField]; ok {
		config.ClientAuth.Username = string(u)
	}
	if p, ok := output.Secret.Data[effectivePasswordField]; ok {
		config.ClientAuth.Password = string(p)
	}

	return config
}
