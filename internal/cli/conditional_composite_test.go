/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestCompositeRequirements checks unconditional alternatives and conjunctions
// against omitted defaults, explicit overrides, and localized help.
func TestCompositeRequirements(t *testing.T) {
	for _, tc := range []struct {
		name, when string
		values     map[string]string
		missing    bool
	}{
		{"always missing", `{"always":true}`, nil, true},
		{"always supplied", `{"always":true}`, map[string]string{"backup": "b"}, false},
		{"all default", `{"all":[{"parameter":"recycle","equals":"false"},{"parameter":"policy","equals":"backupset"}]}`, map[string]string{"policy": "backupset"}, true},
		{"all override", `{"all":[{"parameter":"recycle","equals":"false"},{"parameter":"policy","equals":"backupset"}]}`, map[string]string{"policy": "backupset", "recycle": "true"}, false},
		{"all other", `{"all":[{"parameter":"recycle","equals":"false"},{"parameter":"policy","equals":"backupset"}]}`, map[string]string{"policy": "time"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var condition plugin.ParameterCondition
			if err := json.Unmarshal([]byte(tc.when), &condition); err != nil {
				t.Fatal(err)
			}
			command := plugin.Command{Parameters: []plugin.Parameter{{Name: "recycle", Flag: "recycle", Default: "false"}, {Name: "policy", Flag: "policy"}, {Name: "backup", Flag: "backup"}, {Name: "time", Flag: "time"}}, ConditionalRequirements: []plugin.ConditionalRequirement{{When: condition, AnyOf: []string{"backup", "time"}}}}
			for _, lang := range []string{"en-US", "zh-CN"} {
				err := validateConditionalParameterValues(command, tc.values, lang)
				if (err != nil) != tc.missing {
					t.Fatalf("validation = %v, missing = %v", err, tc.missing)
				}
				hint := parameterConditionalHint(command, command.Parameters[2], lang)
				if !strings.Contains(hint, "--backup") || !strings.Contains(hint, "--time") {
					t.Fatalf("hint = %q", hint)
				}
				if strings.Contains(tc.when, "all") && (!strings.Contains(hint, "--recycle") || !strings.Contains(hint, "--policy")) {
					t.Fatalf("conjunction hint = %q", hint)
				}
			}
		})
	}
}

// TestCompositeRequiredDiagnostics verifies readable condition text and missing
// required options for both unconditional and nested condition metadata.
func TestCompositeRequiredDiagnostics(t *testing.T) {
	command := plugin.Command{Parameters: []plugin.Parameter{{Name: "mode", Flag: "mode", Default: "a"}, {Name: "backup", Flag: "backup"}}}
	for _, condition := range []plugin.ParameterCondition{{Always: true}, {All: []plugin.ParameterCondition{{Always: true}, {Parameter: "mode", In: []string{"a", "b"}}}}} {
		command.ConditionalRequirements = []plugin.ConditionalRequirement{{When: condition, Required: []string{"backup"}}}
		for _, lang := range []string{"en-US", "en-GB", "zh-CN"} {
			if err := validateConditionalParameterValues(command, nil, lang); err == nil || !strings.Contains(err.Error(), "--backup") {
				t.Fatalf("error=%v", err)
			}
			if err := validateConditionalParameterValues(command, map[string]string{"backup": "b"}, lang); err != nil {
				t.Fatal(err)
			}
			if hint := parameterConditionalHint(command, command.Parameters[1], lang); hint == "" {
				t.Fatal("missing required hint")
			}
			if hint := parameterConditionalHint(command, command.Parameters[0], lang); hint != "" {
				t.Fatalf("unrelated hint=%q", hint)
			}
		}
	}
	if got := parameterConditionDescription(command, plugin.ParameterCondition{Parameter: "missing"}, "en-US", true); got != "" {
		t.Fatalf("unknown selector=%q", got)
	}
}
