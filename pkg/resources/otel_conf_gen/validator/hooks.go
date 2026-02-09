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
	"reflect"
	"strings"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pipeline"
	"go.uber.org/zap/zapcore"
)

func createDecoderConfig(result interface{}, hooks ...mapstructure.DecodeHookFunc) *mapstructure.DecoderConfig {
	return &mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(hooks...),
		Result:           result,
		WeaklyTypedInput: true,
	}
}

// decodeID converts string to component.ID or pipeline.ID
func decodeID(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
	// occasionally components don't follow the type/name format
	// in such cases, we need to handle them separately
	exceptionComponents := map[string]bool{
		"debug":             true,
		"deltatocumulative": true,
		"memory_limiter":    true,
		"k8sattributes":     true,
	}

	if from.Kind() == reflect.String {
		parts := strings.SplitN(data.(string), "/", 2)
		switch to {
		case reflect.TypeOf(component.ID{}):
			if len(parts) != 2 {
				if exceptionComponents[parts[0]] {
					return component.MustNewID(parts[0]), nil
				}

				return nil, errors.Errorf("invalid component ID format: %s", data.(string))
			}
			return component.NewIDWithName(component.MustNewType(parts[0]), parts[1]), nil
		case reflect.TypeOf(pipeline.ID{}):
			if len(parts) != 2 {
				return nil, errors.Errorf("invalid pipeline ID format: %s", data.(string))
			}
			signal := pipeline.Signal{}
			if err := signal.UnmarshalText([]byte(parts[0])); err != nil {
				return nil, errors.Errorf("invalid pipeline signal: %s", parts[0])
			}
			return pipeline.NewIDWithName(signal, parts[1]), nil
		}
	}

	return data, nil
}

// decodeLevel converts specific string values to corresponding int-based levels.
func decodeLevel(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
	if from.Kind() == reflect.String {
		switch to {
		case reflect.TypeOf(configtelemetry.Level(0)):
			var level configtelemetry.Level
			if err := level.UnmarshalText([]byte(data.(string))); err != nil {
				return nil, errors.Errorf("invalid telemetry level: %s", data.(string))
			}
			return level, nil
		case reflect.TypeOf(zapcore.Level(0)):
			level, err := zapcore.ParseLevel(data.(string))
			if err != nil {
				return nil, errors.Errorf("invalid zapcore level: %s", data.(string))
			}
			return level, nil
		}
	}

	return data, nil
}
