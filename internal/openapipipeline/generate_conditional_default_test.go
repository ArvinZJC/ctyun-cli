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
