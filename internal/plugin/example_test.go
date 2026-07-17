/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"slices"
	"testing"
)

// TestParseCommandExample verifies the inert POSIX quoting subset used by
// published command examples.
func TestParseCommandExample(t *testing.T) {
	tests := []struct {
		name      string
		example   string
		want      []string
		wantError bool
	}{
		{name: "plain and json", example: `ctyun demo create --items '["a","b"]'`, want: []string{"ctyun", "demo", "create", "--items", `["a","b"]`}},
		{name: "double quoted", example: `ctyun demo create --name "two words"`, want: []string{"ctyun", "demo", "create", "--name", "two words"}},
		{name: "double quoted escape", example: "ctyun demo create --name \"two\\ words\"", want: []string{"ctyun", "demo", "create", "--name", "two words"}},
		{name: "escaped whitespace", example: `ctyun demo create --name two\ words`, want: []string{"ctyun", "demo", "create", "--name", "two words"}},
		{name: "embedded apostrophe", example: `ctyun demo create --name 'it'\''s'`, want: []string{"ctyun", "demo", "create", "--name", "it's"}},
		{name: "empty quoted value", example: `ctyun demo create --name ''`, want: []string{"ctyun", "demo", "create", "--name", ""}},
		{name: "unterminated single quote", example: `ctyun demo 'create`, wantError: true},
		{name: "unterminated double quote", example: `ctyun demo "create`, wantError: true},
		{name: "trailing escape", example: "ctyun demo \\", wantError: true},
		{name: "double quoted trailing escape", example: "ctyun demo \"value\\", wantError: true},
		{name: "operator", example: `ctyun demo; echo unsafe`, wantError: true},
		{name: "substitution", example: `ctyun demo $(whoami)`, wantError: true},
		{name: "backtick", example: "ctyun demo `whoami`", wantError: true},
		{name: "double quoted backtick", example: "ctyun demo \"`whoami`\"", wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCommandExample(tc.example)
			if tc.wantError {
				if err == nil {
					t.Fatalf("ParseCommandExample(%q) returned nil error", tc.example)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("ParseCommandExample(%q) = %#v, want %#v", tc.example, got, tc.want)
			}
		})
	}
}

// TestValidateCommandExample verifies canonical paths, declared options,
// required inputs, conditional branches, and typed values.
func TestValidateCommandExample(t *testing.T) {
	command := Command{
		ID:   "demo.policy.update",
		Path: []string{"demo", "policy", "update", "{policy_id}"},
		Parameters: []Parameter{
			{Name: "count", Flag: "count", Required: true, ValueType: ParameterValueInteger},
			{Name: "names", Flag: "names", ValueType: ParameterValueStringArray},
			{Name: "mode", Flag: "mode", Required: true, AllowedValues: []string{"text", "jwt"}},
			{Name: "alg", Flag: "alg"},
			{Name: "label", Flag: "label", Pattern: `^[a-z]+$`},
			{Name: "selector", Flag: "selector"},
			{Name: "id", Flag: "id"},
			{Name: "kind", Flag: "kind"},
		},
		ConditionalRequirements: []ConditionalRequirement{
			{
				When:     ParameterCondition{Parameter: "mode", Equals: "jwt"},
				Required: []string{"alg"},
			},
			{
				When:  ParameterCondition{Parameter: "kind", In: []string{"selected", "explicit"}},
				AnyOf: []string{"selector", "id"},
			},
		},
	}
	tests := []struct {
		name      string
		example   string
		wantError bool
	}{
		{name: "valid text", example: `ctyun demo policy update policy-1 --count 2 --mode text --names '["one"]' --label demo`},
		{name: "valid jwt", example: `ctyun demo policy update policy-1 --count=2 --mode jwt --alg RS256`},
		{name: "valid any-of branch", example: `ctyun demo policy update policy-1 --count 2 --mode text --kind selected --selector demo`},
		{name: "explicit unavailable placeholder", example: `ctyun demo policy update {policy_id} --count {count} --mode text`},
		{name: "wrong executable", example: `other demo policy update policy-1 --count 2 --mode text`, wantError: true},
		{name: "wrong path", example: `ctyun demo policy create policy-1 --count 2 --mode text`, wantError: true},
		{name: "missing argument", example: `ctyun demo policy update --count 2 --mode text`, wantError: true},
		{name: "extra argument", example: `ctyun demo policy update policy-1 extra --count 2 --mode text`, wantError: true},
		{name: "unknown option", example: `ctyun demo policy update policy-1 --count 2 --mode text --other value`, wantError: true},
		{name: "missing option value", example: `ctyun demo policy update policy-1 --count 2 --mode`, wantError: true},
		{name: "missing required option", example: `ctyun demo policy update policy-1 --mode text`, wantError: true},
		{name: "invalid integer", example: `ctyun demo policy update policy-1 --count two --mode text`, wantError: true},
		{name: "invalid json array", example: `ctyun demo policy update policy-1 --count 2 --mode text --names '[1]'`, wantError: true},
		{name: "invalid allowed value", example: `ctyun demo policy update policy-1 --count 2 --mode other`, wantError: true},
		{name: "invalid pattern", example: `ctyun demo policy update policy-1 --count 2 --mode text --label BAD`, wantError: true},
		{name: "missing conditional option", example: `ctyun demo policy update policy-1 --count 2 --mode jwt`, wantError: true},
		{name: "duplicate option", example: `ctyun demo policy update policy-1 --count 2 --count 3 --mode text`, wantError: true},
		{name: "missing any-of option", example: `ctyun demo policy update policy-1 --count 2 --mode text --kind explicit`, wantError: true},
		{name: "development option", example: `ctyun demo policy update policy-1 --count 2 --mode text --offline`, wantError: true},
		{name: "empty", example: ``, wantError: true},
		{name: "unsafe syntax", example: `ctyun demo; echo unsafe`, wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCommandExample(command, tc.example)
			if tc.wantError && err == nil {
				t.Fatalf("ValidateCommandExample(%q) returned nil error", tc.example)
			}
			if !tc.wantError && err != nil {
				t.Fatalf("ValidateCommandExample(%q): %v", tc.example, err)
			}
		})
	}
}
