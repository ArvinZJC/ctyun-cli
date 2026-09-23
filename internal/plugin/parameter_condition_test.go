/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import "testing"

// TestParameterConditionValidation checks that malformed composites cannot hide
// unknown selectors or combine incompatible condition modes.
func TestParameterConditionValidation(t *testing.T) {
	parameters := map[string]Parameter{"mode": {Name: "mode"}}
	for _, tc := range []struct {
		name      string
		condition ParameterCondition
		valid     bool
	}{
		{"always", ParameterCondition{Always: true}, true},
		{"nested", ParameterCondition{All: []ParameterCondition{{Always: true}, {Parameter: "mode", In: []string{"a", "b"}}}}, true},
		{"all always", ParameterCondition{Always: true, All: []ParameterCondition{{Always: true}}}, false},
		{"always selector", ParameterCondition{Always: true, Parameter: "mode"}, false},
		{"always equals", ParameterCondition{Always: true, Equals: "a"}, false},
		{"always in", ParameterCondition{Always: true, In: []string{"a"}}, false},
		{"all selector", ParameterCondition{Parameter: "mode", All: []ParameterCondition{{Always: true}}}, false},
		{"empty", ParameterCondition{}, false},
		{"unknown child", ParameterCondition{All: []ParameterCondition{{Parameter: "missing", Equals: "a"}}}, false},
		{"missing match", ParameterCondition{Parameter: "mode"}, false},
		{"duplicate match", ParameterCondition{Parameter: "mode", Equals: "a", In: []string{"b"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateParameterCondition(tc.condition, "synthetic", parameters); (err == nil) != tc.valid {
				t.Fatalf("validation=%v valid=%v", err, tc.valid)
			}
		})
	}
}

// TestParameterConditionMatching verifies default-aware recursive evaluation
// without mutating explicitly supplied values.
func TestParameterConditionMatching(t *testing.T) {
	parameters := []Parameter{{Name: "mode", Default: "a"}, {Name: "enabled", Default: "false"}}
	condition := ParameterCondition{All: []ParameterCondition{{Always: true}, {Parameter: "mode", In: []string{"a", "b"}}, {Parameter: "enabled", Equals: "false"}}}
	for _, tc := range []struct {
		values  map[string]string
		matches bool
	}{
		{nil, true}, {map[string]string{"mode": "b"}, true}, {map[string]string{"mode": "c"}, false}, {map[string]string{"mode": ""}, false}, {map[string]string{"enabled": "true"}, false},
	} {
		before := len(tc.values)
		if got := ParameterConditionMatches(condition, parameters, tc.values); got != tc.matches {
			t.Fatalf("match=%v values=%v", got, tc.values)
		}
		if len(tc.values) != before {
			t.Fatal("mutated inputs")
		}
	}
}
