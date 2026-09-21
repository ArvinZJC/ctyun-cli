/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import "testing"

// TestCommandExampleConditionalDefaultMatchesRuntime rejects examples that
// omit inputs required by the API's declared default selector value.
func TestCommandExampleConditionalDefaultMatchesRuntime(t *testing.T) {
	command := Command{ID: "demo.restore", Path: []string{"demo", "restore"}, Parameters: []Parameter{
		{Name: "recycle", Flag: "recycle", Default: "false", ValueType: ParameterValueBoolean},
		{Name: "backup", Flag: "backup"},
	}, ConditionalRequirements: []ConditionalRequirement{{When: ParameterCondition{Parameter: "recycle", Equals: "false"}, Required: []string{"backup"}}}}
	for _, anyOf := range []bool{false, true} {
		if anyOf {
			command.ConditionalRequirements[0].Required = nil
			command.ConditionalRequirements[0].AnyOf = []string{"backup"}
		}
		if err := ValidateCommandExample(command, "ctyun demo restore"); err == nil {
			t.Errorf("anyOf=%t example accepted missing default-branch input", anyOf)
		}
		for _, example := range []string{"ctyun demo restore --backup backup-1", "ctyun demo restore --recycle true"} {
			if err := ValidateCommandExample(command, example); err != nil {
				t.Errorf("example %q: %v", example, err)
			}
		}
	}
}

// TestParameterDefaultLookupPreservesExplicitEmpty distinguishes omission
// from a supplied empty selector and tolerates an unknown metadata name.
func TestParameterDefaultLookupPreservesExplicitEmpty(t *testing.T) {
	parameters := []Parameter{{Name: "mode", Default: "automatic"}}
	if got := ParameterValueOrDefault("mode", parameters, map[string]string{"mode": ""}); got != "" {
		t.Fatalf("explicit empty selector replaced with %q", got)
	}
	if got := ParameterValueOrDefault("missing", parameters, nil); got != "" {
		t.Fatalf("unknown selector resolved to %q", got)
	}
}
