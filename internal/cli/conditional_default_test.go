/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestConditionalRequirementsUseOmittedAPIDefault verifies that an omitted
// selector activates its default branch without populating request input.
func TestConditionalRequirementsUseOmittedAPIDefault(t *testing.T) {
	for _, anyOf := range []bool{false, true} {
		command := plugin.Command{ID: "demo.restore", Parameters: []plugin.Parameter{
			{Name: "recycle", Flag: "recycle", Target: "recycle", Default: "false", ValueType: plugin.ParameterValueBoolean},
			{Name: "backup", Flag: "backup", Target: "backup"},
		}}
		rule := plugin.ConditionalRequirement{When: plugin.ParameterCondition{Parameter: "recycle", Equals: "false"}}
		if anyOf {
			rule.When = plugin.ParameterCondition{Parameter: "recycle", In: []string{"false"}}
			rule.AnyOf = []string{"backup"}
		} else {
			rule.Required = []string{"backup"}
		}
		command.ConditionalRequirements = []plugin.ConditionalRequirement{rule}
		if _, err := parseCommandParameters(command, nil, "en-US"); err == nil || !strings.Contains(err.Error(), "--backup") {
			t.Errorf("anyOf=%t omitted false default accepted missing backup: %v", anyOf, err)
		}
		if _, err := parseCommandParameters(command, []string{"--recycle", "true"}, "en-US"); err != nil {
			t.Errorf("anyOf=%t explicit true did not override default: %v", anyOf, err)
		}
		values, err := parseCommandParameters(command, []string{"--backup", "backup-1"}, "en-US")
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := values["recycle"]; exists {
			t.Errorf("default was injected into parsed request inputs: %#v", values)
		}
	}
}
