/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestDescriptionQualityFindingRejectsObjectiveGeneratedDefects verifies the
// review gate without treating normal ECS terminology as a defect.
func TestDescriptionQualityFindingRejectsObjectiveGeneratedDefects(t *testing.T) {
	catalog := loadCatalogFixture(t)
	operation := Operation{ID: "v4.ecs.instance.vm-cpu-history-metric-data", Category: "instance"}
	command := plugin.Command{Path: []string{"ecs", "instance", "vm-cpu-history-metric-data"}}
	tests := []struct {
		description string
		wantFinding bool
	}{
		{description: "Vm CPU History Metric Data ECS instance", wantFinding: true},
		{description: "List Cpu metrics for ECS instances", wantFinding: true},
		{description: "Call /v4/ecs/vm-cpu-history-metric-data", wantFinding: true},
		{description: "List CPU metrics for ECS instances over a specified time range"},
		{description: "Update information for an ECS instance"},
		{description: "ECS instance vm CPU history metric data", wantFinding: true},
	}
	for _, tc := range tests {
		if got := descriptionQualityFinding(catalog, operation, command, tc.description); (got != "") != tc.wantFinding {
			t.Fatalf("descriptionQualityFinding(%q) = %q", tc.description, got)
		}
	}
	simpleOperation := Operation{ID: "v4.ecs.instance.create", Category: "instance"}
	simpleCommand := plugin.Command{Path: []string{"ecs", "instance", "create"}}
	if got := descriptionQualityFinding(catalog, simpleOperation, simpleCommand, "Create ECS instance"); got == "" {
		t.Fatal("descriptionQualityFinding accepted a simple mechanical fallback")
	}
}

// TestOperationMissingExampleEvidence distinguishes an explicit upstream
// absence from an accidental bare placeholder.
func TestOperationMissingExampleEvidence(t *testing.T) {
	operation := Operation{Parameters: []Parameter{{Name: "count", CLIName: "count", Required: true, Type: "Integer"}}}
	command := plugin.Command{Parameters: []plugin.Parameter{{Name: "count", Flag: "count", Required: true, ValueType: plugin.ParameterValueInteger}}}
	if got := operationMissingExampleEvidence(operation, command); !slices.Equal(got, []string{"count"}) {
		t.Fatalf("missing evidence = %#v", got)
	}
	operation.Parameters[0].ExampleUnavailable = true
	if got := operationMissingExampleEvidence(operation, command); len(got) != 0 {
		t.Fatalf("explicit unavailable evidence reported missing: %#v", got)
	}
	operation.Parameters[0].ExampleUnavailable = false
	operation.RequestExample = map[string]json.RawMessage{"count": json.RawMessage("2")}
	if got := operationMissingExampleEvidence(operation, command); len(got) != 0 {
		t.Fatalf("request example evidence reported missing: %#v", got)
	}
}

// TestReviewDraftRejectsInvalidPublishedExample verifies that draft review
// applies the shared command contract before promotion.
func TestReviewDraftRejectsInvalidPublishedExample(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	catalog := loadCatalogFixture(t)
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatal(err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatal(err)
	}
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	commands.Commands[0].Examples[0] = "ctyun ecs instance list --unknown value"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "commands.json"), commands); err != nil {
		t.Fatal(err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready {
		t.Fatalf("review accepted invalid example: %#v", report)
	}
	found := false
	for _, finding := range report.Findings {
		if slices.Contains([]string{finding}, "command ecs.instance.list example is invalid: unknown example option --unknown") {
			found = true
		}
	}
	if !found {
		t.Fatalf("review findings = %#v", report.Findings)
	}
	if _, err := os.Stat(workspace.ProductPath("ecs", "review.md")); err != nil {
		t.Fatalf("review report not written: %v", err)
	}
}

// TestReviewDraftReportsDescriptionAndEvidenceGaps verifies that review emits
// actionable findings for generated help text and required input evidence.
func TestReviewDraftReportsDescriptionAndEvidenceGaps(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].Description["en-US"] = "ECS instance list"
	catalog.Operations[0].Description["en-GB"] = "ECS instance list"
	catalog.Operations[0].Parameters[1].Required = true
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatal(err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatal(err)
	}
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	commands.Commands[0].Examples = nil
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "commands.json"), commands); err != nil {
		t.Fatal(err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"operation v4.ecs.instance.list description en-US is a title-cased command path",
		"operation v4.ecs.instance.list description en-GB is a title-cased command path",
		"command ecs.instance.list has no example",
		"operation v4.ecs.instance.list required input name lacks example evidence",
	}
	for _, finding := range want {
		if !slices.Contains(report.Findings, finding) {
			t.Errorf("review findings %#v do not include %q", report.Findings, finding)
		}
	}
}
