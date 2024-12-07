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

package validator

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

// ValidateAssembledConfig validates the assembled OpenTelemetry Collector configuration
// which first needs to be converted from the CRD format to the internally used one
func ValidateAssembledConfig(otelConfig otelv1beta1.Config) error {
	var cfg otelcol.Config
	var err error

	if cfg.Receivers, err = decodeAnyConfig(otelConfig.Receivers); err != nil {
		return errors.Wrap(err, "decoding receivers")
	}

	if cfg.Exporters, err = decodeAnyConfig(otelConfig.Exporters); err != nil {
		return errors.Wrap(err, "decoding exporters")
	}

	if otelConfig.Processors != nil {
		if cfg.Processors, err = decodeAnyConfig(*otelConfig.Processors); err != nil {
			return errors.Wrap(err, "decoding processors")
		}
	}

	if otelConfig.Connectors != nil {
		if cfg.Connectors, err = decodeAnyConfig(*otelConfig.Connectors); err != nil {
			return errors.Wrap(err, "decoding connectors")
		}
	}

	if otelConfig.Extensions != nil {
		if cfg.Extensions, err = decodeAnyConfig(*otelConfig.Extensions); err != nil {
			return errors.Wrap(err, "decoding extensions")
		}
	}

	if cfg.Service, err = decodeServiceConfig(otelConfig.Service); err != nil {
		return errors.Wrap(err, "decoding service configuration")
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid OpenTelemetry Collector configuration: %w", err)
	}

	return nil
}

func decodeAnyConfig(anyConfig otelv1beta1.AnyConfig) (map[component.ID]component.Config, error) {
	var result map[component.ID]component.Config
	decoder, err := mapstructure.NewDecoder(createDecoderConfig(&result, decodeID, mapstructure.StringToTimeDurationHookFunc(), mapstructure.StringToSliceHookFunc(",")))
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(anyConfig.Object); err != nil {
		return nil, fmt.Errorf("failed to decode AnyConfig: %w", err)
	}

	return result, nil
}

func decodeServiceConfig(serviceConfig otelv1beta1.Service) (service.Config, error) {
	var result service.Config
	var err error
	if serviceConfig.Telemetry != nil {
		if result.Telemetry, err = decodeTelemetryConfig(*serviceConfig.Telemetry); err != nil {
			return result, fmt.Errorf("failed to decode telemetry config: %w", err)
		}
	}

	if serviceConfig.Extensions != nil {
		result.Extensions = decodeExtensionsConfig(serviceConfig.Extensions)
	}

	if serviceConfig.Pipelines != nil {
		if result.Pipelines, err = decodePipelinesConfig(serviceConfig.Pipelines); err != nil {
			return result, fmt.Errorf("failed to decode pipelines config: %w", err)
		}
	}

	return result, nil
}

func decodeTelemetryConfig(anyConfig otelv1beta1.AnyConfig) (telemetry.Config, error) {
	var result telemetry.Config
	decoder, err := mapstructure.NewDecoder(createDecoderConfig(&result, decodeID, decodeLevel, mapstructure.StringToTimeDurationHookFunc(), mapstructure.StringToSliceHookFunc(",")))
	if err != nil {
		return result, fmt.Errorf("failed to create telemetry decoder: %w", err)
	}
	if err := decoder.Decode(anyConfig.Object); err != nil {
		return result, fmt.Errorf("failed to decode telemetry config: %w", err)
	}

	return result, nil
}

func decodeExtensionsConfig(extensionsConfig []string) extensions.Config {
	var result extensions.Config
	for _, extName := range extensionsConfig {
		parts := strings.Split(extName, "/")
		if len(parts) != 2 {
			result = append(result, component.MustNewID(extName))
		} else {
			result = append(result, component.MustNewIDWithName(parts[0], parts[1]))
		}
	}

	return result
}

func decodePipelinesConfig(pipelinesConfig map[string]*otelv1beta1.Pipeline) (pipelines.Config, error) {
	result := make(pipelines.Config)
	decoder, err := mapstructure.NewDecoder(createDecoderConfig(&result, decodeID, mapstructure.StringToSliceHookFunc(","), mapstructure.StringToTimeDurationHookFunc()))
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}
	if err := decoder.Decode(pipelinesConfig); err != nil {
		return nil, fmt.Errorf("failed to decode pipelines: %w", err)
	}

	return result, nil
}
