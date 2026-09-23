/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"slices"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// ParameterConditionMatches evaluates requirement selectors with documented
// defaults without mutating explicit inputs or constructing request fields.
func ParameterConditionMatches(condition ParameterCondition, parameters []Parameter, values map[string]string) bool {
	if condition.Always {
		return true
	}
	if len(condition.All) > 0 {
		for _, child := range condition.All {
			if !ParameterConditionMatches(child, parameters, values) {
				return false
			}
		}
		return true
	}
	value := ParameterValueOrDefault(condition.Parameter, parameters, values)
	if value == "" {
		return false
	}
	if condition.Equals != "" {
		return value == condition.Equals
	}
	return slices.Contains(condition.In, value)
}

// ValidateParameterCondition checks mutually exclusive condition forms and
// recursively verifies selectors against declared command parameters.
func ValidateParameterCondition(condition ParameterCondition, commandID string, parameters map[string]Parameter) error {
	composite := condition.Always || len(condition.All) > 0
	if composite {
		if (condition.Always && len(condition.All) > 0) || condition.Parameter != "" || condition.Equals != "" || len(condition.In) > 0 {
			return diagnostic.New("error.command_conditional_mixed_modes", commandID)
		}
		for _, child := range condition.All {
			if err := ValidateParameterCondition(child, commandID, parameters); err != nil {
				return err
			}
		}
		return nil
	}
	if condition.Parameter == "" {
		return diagnostic.New("error.command_conditional_missing_parameter", commandID)
	}
	if _, ok := parameters[condition.Parameter]; !ok {
		return diagnostic.New("error.command_conditional_unknown_parameter", commandID, condition.Parameter)
	}
	if condition.Equals == "" && len(condition.In) == 0 {
		return diagnostic.New("error.command_conditional_missing_match", commandID, condition.Parameter)
	}
	if condition.Equals != "" && len(condition.In) > 0 {
		return diagnostic.New("error.command_conditional_duplicate_match", commandID, condition.Parameter)
	}
	return nil
}
