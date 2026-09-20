/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/jsonvalue"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// renderWaiter evaluates optional waiter metadata and writes the final state.
func renderWaiter(stdout io.Writer, bundle plugin.Bundle, waiterID string, payload map[string]any, loadResponse func() (map[string]any, error), language string, selectorValue string) error {
	if waiterID == "" {
		return nil
	}
	spec, ok := bundle.Waiters.Waiters[waiterID]
	if !ok {
		return diagnostic.New("error.unknown_waiter", waiterID)
	}
	var selector *waiter.Selector
	if spec.Selector != nil {
		resolved := *spec.Selector
		resolved.Value = selectorValue
		selector = &resolved
	}
	attempts := spec.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	var state waiter.State
	for attempt := 1; attempt <= attempts; attempt++ {
		var err error
		state, err = waiter.Evaluate(waiter.Spec{
			Path:          spec.Path,
			Selector:      selector,
			SuccessValues: spec.SuccessValues,
			FailureValues: spec.FailureValues,
			Success:       spec.Success,
			Failure:       spec.Failure,
		}, payload)
		if err != nil {
			return err
		}
		if state != waiter.Pending {
			break
		}
		if attempt == attempts {
			state = waiter.Timeout
			break
		}
		if spec.IntervalSeconds > 0 {
			time.Sleep(time.Duration(spec.IntervalSeconds) * time.Second)
		}
		payload, err = loadResponse()
		if err != nil {
			return err
		}
	}
	return writeLine(stdout, waiterStatusMessage(language, waiterID, string(state)))
}

// resolveWaiterSelection requires an explicit resource identity before polling
// so an omitted optional list filter cannot become a broad readiness check.
func resolveWaiterSelection(command plugin.Command, spec plugin.Waiter, args, parameters map[string]string) (string, error) {
	value := ""
	if name, ok := strings.CutPrefix(spec.Selector.Value, "$arg."); ok {
		value = args[name]
	} else if name, ok := strings.CutPrefix(spec.Selector.Value, "$param."); ok {
		value = parameters[name]
		for _, parameter := range command.Parameters {
			if parameter.Name == name && parameter.ValueType == plugin.ParameterValueInteger {
				value = jsonvalue.NumberText(json.Number(strings.TrimSpace(value)))
			}
		}
	}
	if strings.TrimSpace(value) == "" {
		return "", diagnostic.New("error.waiter_requires_input", plugin.WaiterSelectorInput(command, spec))
	}
	return value, nil
}
