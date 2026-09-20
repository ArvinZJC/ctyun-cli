/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"slices"
	"sort"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// WaiterApplies reports whether the command can safely poll this waiter. Legacy
// fixture-only commands remain supported; live polling requires retryable metadata.
func WaiterApplies(bundle Bundle, command Command, spec Waiter) bool {
	if command.Dangerous.Confirm != "" {
		return false
	}
	if command.Operation != "" && !bundle.APIs.Operations[command.Operation].Retryable {
		return false
	}
	return len(spec.Commands) == 0 || slices.Contains(spec.Commands, command.ID)
}

// CommandWaiters returns sorted waiter IDs applicable to a resolved command.
func CommandWaiters(bundle Bundle, command Command) []string {
	var ids []string
	for id, spec := range bundle.Waiters.Waiters {
		if WaiterApplies(bundle, command, spec) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// ValidateWaiterBindings rejects unusable terminal conditions and references to
// missing or unsafe commands before a bundle is executed or promoted.
func ValidateWaiterBindings(bundle Bundle) error {
	if err := validateWaiters(bundle.Waiters); err != nil {
		return err
	}
	for id, spec := range bundle.Waiters.Waiters {
		if spec.Selector != nil && (strings.TrimSpace(spec.Selector.Path) == "" || strings.TrimSpace(spec.Selector.Key) == "" || len(spec.Commands) == 0) {
			return diagnostic.New("error.waiter_invalid_selector", id)
		}
		if strings.TrimSpace(spec.Path) == "" || (spec.Success == "" && len(spec.SuccessValues) == 0) {
			return diagnostic.New("error.waiter_invalid_conditions", id)
		}
		success := append([]string{}, spec.SuccessValues...)
		if spec.Success != "" {
			success = append(success, spec.Success)
		}
		failure := append([]string{}, spec.FailureValues...)
		if spec.Failure != "" {
			failure = append(failure, spec.Failure)
		}
		for _, value := range success {
			if value == "" || slices.Contains(failure, value) {
				return diagnostic.New("error.waiter_invalid_conditions", id)
			}
		}
		for _, value := range failure {
			if value == "" {
				return diagnostic.New("error.waiter_invalid_conditions", id)
			}
		}
		for _, commandID := range spec.Commands {
			found := false
			for _, command := range bundle.Commands.Commands {
				if command.ID == commandID && WaiterApplies(bundle, command, spec) && (spec.Selector == nil || WaiterSelectorInput(command, spec) != "") {
					found = true
					break
				}
			}
			if !found {
				return diagnostic.New("error.waiter_invalid_binding", id, commandID)
			}
		}
	}
	return nil
}

// WaiterSelectorInput resolves a selector reference to its visible positional
// argument or option; unsupported or nonexistent references return no input.
func WaiterSelectorInput(command Command, spec Waiter) string {
	if spec.Selector == nil {
		return ""
	}
	if name, ok := strings.CutPrefix(spec.Selector.Value, "$arg."); ok {
		placeholder := "{" + name + "}"
		if slices.Contains(command.Path, placeholder) {
			return placeholder
		}
	}
	if name, ok := strings.CutPrefix(spec.Selector.Value, "$param."); ok {
		for _, parameter := range command.Parameters {
			if parameter.Name == name && (parameter.ValueType == "" || parameter.ValueType == ParameterValueString || parameter.ValueType == ParameterValueInteger) {
				return "--" + parameter.Flag
			}
		}
	}
	return ""
}
