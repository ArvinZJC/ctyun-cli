/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"encoding/json"
	"testing"
)

// TestRecommendationMetadataDecodesVisibleCommandOnly verifies the plugin
// schema accepts only the narrow visible-command representation.
func TestRecommendationMetadataDecodesVisibleCommandOnly(t *testing.T) {
	var commands Commands
	err := json.Unmarshal([]byte(`{"commands":[{"id":"ecs.metric.history","path":["ecs","metric","history"],"recommendation":{"target_command":{"plugin":"monitor","path":["monitor","metric","history"]}}}]}`), &commands)
	if err != nil {
		t.Fatalf("decode commands: %v", err)
	}
	got := commands.Commands[0].Recommendation
	if !got.Active() || got.TargetCommand.Plugin != "monitor" || !equalStrings(got.TargetCommand.Path, []string{"monitor", "metric", "history"}) {
		t.Fatalf("recommendation = %#v", got)
	}
}

// TestQualifiedRecommendationMetadata verifies applicability and its stable
// localization key survive the narrow generated plugin schema.
func TestQualifiedRecommendationMetadata(t *testing.T) {
	var commands Commands
	err := json.Unmarshal([]byte(`{"commands":[{"id":"ims.image.list","path":["ims","image","list"],"recommendation":{"target_command":{"plugin":"dps","path":["dps","image","list"]},"applicability":"physical-machine images"}}]}`), &commands)
	if err != nil {
		t.Fatal(err)
	}
	got := commands.Commands[0].Recommendation
	if got.Applicability != "physical-machine images" {
		t.Fatalf("applicability = %q", got.Applicability)
	}
	if key := RecommendationApplicabilityKey("ims.image.list"); key != "recommendation.ims.image.list.applicability" {
		t.Fatalf("key = %q", key)
	}
}

// TestRecommendationApplicabilityValidation covers qualified guidance and
// rejects whitespace-only qualifiers or recommendations without a target.
func TestRecommendationApplicabilityValidation(t *testing.T) {
	validTarget := CommandTarget{Plugin: "dps", Path: []string{"dps", "image", "list"}}
	if err := validateRecommendation(&Recommendation{TargetCommand: validTarget, Applicability: "physical-machine images"}); err != nil {
		t.Fatalf("valid applicability: %v", err)
	}
	assertDiagnosticKey(t, validateRecommendation(&Recommendation{TargetCommand: validTarget, Applicability: " \t "}), "error.recommendation_applicability")
	if err := validateRecommendation(&Recommendation{Applicability: "physical-machine images"}); err == nil {
		t.Fatal("inactive target accepted qualified applicability")
	}
}

// TestCommandTargetValidation covers valid and unsafe local command targets.
func TestCommandTargetValidation(t *testing.T) {
	cases := []struct {
		name   string
		target CommandTarget
		want   string
	}{
		{name: "valid", target: CommandTarget{Plugin: "monitor", Path: []string{"monitor", "metric", "history"}}},
		{name: "plugin", target: CommandTarget{Plugin: "../monitor", Path: []string{"monitor", "metric", "history"}}, want: "error.recommendation_target_plugin"},
		{name: "empty path", target: CommandTarget{Plugin: "monitor"}, want: "error.recommendation_target_path"},
		{name: "segment", target: CommandTarget{Plugin: "monitor", Path: []string{"monitor", "/history"}}, want: "error.recommendation_target_path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.target.Validate()
			if tc.want == "" && err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}
			if tc.want != "" {
				assertDiagnosticKey(t, err, tc.want)
			}
		})
	}
}

// TestFindCommandTargetAndDeprecation verifies exact bundle resolution and
// command- or operation-level lifecycle checks.
func TestFindCommandTargetAndDeprecation(t *testing.T) {
	command := Command{ID: "monitor.metric.history", Path: []string{"monitor", "metric", "history"}, Operation: "monitor.history"}
	bundle := Bundle{Manifest: Manifest{Name: "monitor"}, Commands: Commands{Commands: []Command{command}}, APIs: APIs{Operations: map[string]Operation{"monitor.history": {Method: "POST", Path: "/v4.2/monitor/query-history-metric-data"}}}}
	gotBundle, gotCommand, ok := FindCommandTarget([]Bundle{bundle}, CommandTarget{Plugin: "monitor", Path: command.Path})
	if !ok || gotBundle.Manifest.Name != "monitor" || gotCommand.ID != command.ID {
		t.Fatalf("resolution = %#v %#v %t", gotBundle.Manifest, gotCommand, ok)
	}
	if _, _, ok := FindCommandTarget([]Bundle{bundle}, CommandTarget{Plugin: "ecs", Path: command.Path}); ok {
		t.Fatal("resolved wrong plugin")
	}
	deprecated := &Deprecation{Status: "deprecated"}
	if !CommandIsDeprecated(Bundle{}, Command{Deprecation: deprecated}) {
		t.Fatal("deprecated command accepted")
	}
	if !CommandIsDeprecated(Bundle{APIs: APIs{Operations: map[string]Operation{"op": {Deprecation: deprecated}}}}, Command{Operation: "op"}) {
		t.Fatal("deprecated operation accepted")
	}
}
