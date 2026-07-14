/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"fmt"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// resolveRequestBody expands body bindings and converts command parameters to
// their declared JSON value shapes.
func resolveRequestBody(values map[string]string, profile coreconfig.Profile, commandArgs, parameterValues map[string]string, parameters []plugin.Parameter, language string) (map[string]any, error) {
	textValues := resolveMap(values, profile, commandArgs, parameterValues, parameters, len(values) > 0)
	resolved := make(map[string]any, len(textValues))
	for key, value := range textValues {
		resolved[key] = value
	}
	for _, parameter := range parameters {
		raw := parameterValues[parameter.Name]
		if raw == "" || parameter.Target == "" {
			continue
		}
		if _, exists := textValues[parameter.Target]; !exists {
			continue
		}
		typed, err := plugin.ParseParameterValue(parameter, raw)
		if err != nil {
			return nil, localizedInvalidOptionValueType(parameter, raw, language)
		}
		resolved[parameter.Target] = typed
	}
	return resolved, nil
}

// resolveQueryMap expands query bindings and canonicalizes composite command
// parameter values as compact JSON text.
func resolveQueryMap(values map[string]string, profile coreconfig.Profile, commandArgs, parameterValues map[string]string, parameters []plugin.Parameter, language string) (map[string]string, error) {
	resolved := resolveMap(values, profile, commandArgs, parameterValues, parameters, false)
	for _, parameter := range parameters {
		raw := parameterValues[parameter.Name]
		if raw == "" || parameter.Target == "" {
			continue
		}
		if _, exists := resolved[parameter.Target]; !exists {
			continue
		}
		typed, err := plugin.ParseParameterValue(parameter, raw)
		if err != nil {
			return nil, localizedInvalidOptionValueType(parameter, raw, language)
		}
		if !compositeParameterValueType(parameter.ValueType) {
			continue
		}
		compact, _ := json.Marshal(typed)
		resolved[parameter.Target] = string(compact)
	}
	return resolved, nil
}

// compositeParameterValueType reports whether an option uses JSON syntax in
// help and query serialization.
func compositeParameterValueType(valueType plugin.ParameterValueType) bool {
	switch valueType {
	case plugin.ParameterValueStringArray, plugin.ParameterValueObjectArray,
		plugin.ParameterValueStringMap, plugin.ParameterValueJSON:
		return true
	default:
		return false
	}
}

// localizedInvalidOptionValueType returns the runtime diagnostic for a value
// that does not match its metadata-declared JSON type.
func localizedInvalidOptionValueType(parameter plugin.Parameter, raw, language string) error {
	valueType := parameter.ValueType
	if valueType == "" {
		valueType = plugin.ParameterValueString
	}
	return fmt.Errorf(messageText("error.invalid_option_value_type", language), raw, "--"+parameter.Flag, valueType)
}
