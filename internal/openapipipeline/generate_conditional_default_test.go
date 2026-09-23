/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestGeneratedExampleIncludesDefaultBranchRequirements verifies that API
// defaults activate conditional examples without adding an omitted selector.
func TestGeneratedExampleIncludesDefaultBranchRequirements(t *testing.T) {
	operation := Operation{Parameters: []Parameter{{Name: "backup", CLIName: "backup", Example: json.RawMessage(`"backup-1"`)}}}
	command := plugin.Command{Path: []string{"demo", "restore"}, Parameters: []plugin.Parameter{
		{Name: "recycle", Flag: "recycle", Default: "false", ValueType: plugin.ParameterValueBoolean},
		{Name: "backup", Flag: "backup"},
	}, ConditionalRequirements: []plugin.ConditionalRequirement{{When: plugin.ParameterCondition{Parameter: "recycle", Equals: "false"}, Required: []string{"backup"}}}}
	example := generatedCommandExample(operation, command)
	if !strings.Contains(example, "--backup backup-1") || strings.Contains(example, "--recycle") {
		t.Fatalf("default-branch example = %q", example)
	}
	if err := plugin.ValidateCommandExample(command, example); err != nil {
		t.Fatal(err)
	}
}

// TestCompositeRequirementExamples verifies that generated examples satisfy
// unconditional choices and nested conditions without injecting defaults.
func TestCompositeRequirementExamples(t *testing.T) {
	operation := Operation{ID: "synthetic", Parameters: []Parameter{{Name: "mode", CLIName: "mode"}, {Name: "backup", CLIName: "backup", Example: json.RawMessage(`"backup-1"`)}, {Name: "time", CLIName: "time"}}}
	command := plugin.Command{Path: []string{"demo", "restore"}, Parameters: []plugin.Parameter{{Name: "mode", Flag: "mode", Default: "a"}, {Name: "backup", Flag: "backup"}, {Name: "time", Flag: "time"}}}
	for _, condition := range []plugin.ParameterCondition{{Always: true}, {All: []plugin.ParameterCondition{{Always: true}, {Parameter: "mode", Equals: "a"}}}} {
		command.ConditionalRequirements = []plugin.ConditionalRequirement{{When: condition, AnyOf: []string{"backup", "time"}}}
		operation.ConditionalRequirements = command.ConditionalRequirements
		if err := operation.validateConditionalRequirements(); err != nil {
			t.Fatal(err)
		}
		example := generatedCommandExample(operation, command)
		if !strings.Contains(example, "--backup backup-1") || strings.Contains(example, "--mode") {
			t.Fatalf("example=%q", example)
		}
		if err := plugin.ValidateCommandExample(command, example); err != nil {
			t.Fatal(err)
		}
		if err := plugin.ValidateCommandExample(command, "ctyun demo restore"); err == nil {
			t.Fatal("missing any-of accepted")
		}
		command.ConditionalRequirements[0].Required = []string{"backup"}
		command.ConditionalRequirements[0].AnyOf = nil
		if err := plugin.ValidateCommandExample(command, "ctyun demo restore"); err == nil {
			t.Fatal("missing required accepted")
		}
	}
	operation.ConditionalRequirements = []plugin.ConditionalRequirement{{When: plugin.ParameterCondition{Always: true, Parameter: "mode"}, Required: []string{"backup"}}}
	if err := operation.validateConditionalRequirements(); err == nil {
		t.Fatal("mixed forms accepted")
	}
}
