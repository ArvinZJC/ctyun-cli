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
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
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
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
	report := clearGeneratedExamplesAndReview(t, workspace, "ecs")
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

// TestReviewDraftAcceptsEmptyExamplesOnlyForBareCommands verifies the review
// boundary for intentionally omitted redundant examples.
func TestReviewDraftAcceptsEmptyExamplesOnlyForBareCommands(t *testing.T) {
	t.Run("bare command", func(t *testing.T) {
		root := t.TempDir()
		workspace := Workspace{Root: root}
		catalog := loadCatalogFixture(t)
		operation := &catalog.Operations[0]
		operation.Examples = nil
		operation.Parameters = []Parameter{{Name: "regionID", Location: "body", Required: true, Type: "string", Profile: "region"}}
		writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
		report := clearGeneratedExamplesAndReview(t, workspace, "ecs")
		if !report.Ready {
			t.Fatalf("review rejected eligible empty examples: %#v", report.Findings)
		}
	})

	for _, tc := range []struct {
		name      string
		operation Parameter
		path      []string
	}{
		{
			name:      "path argument",
			operation: Parameter{Name: "instanceID", Location: "query", Required: true, Type: "string", Argument: "instance_id", Example: json.RawMessage(`"instance-demo"`)},
			path:      []string{"instance", "show", "{instance_id}"},
		},
		{
			name:      "required option",
			operation: Parameter{Name: "name", Location: "query", Required: true, Type: "string", CLIName: "name", CLIFlag: "name", Example: json.RawMessage(`"captured-name"`)},
			path:      []string{"instance", "list"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			workspace := Workspace{Root: root}
			catalog := loadCatalogFixture(t)
			operation := &catalog.Operations[0]
			operation.Examples = nil
			operation.CommandPath = tc.path
			operation.Parameters = []Parameter{{Name: "regionID", Location: "query", Required: true, Type: "string", Profile: "region"}, tc.operation}
			writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
			report := clearGeneratedExamplesAndReview(t, workspace, "ecs")
			finding := "command ecs.instance.list has no example"
			if !slices.Contains(report.Findings, finding) {
				t.Fatalf("review findings %#v do not include %q", report.Findings, finding)
			}
		})
	}
}

// TestReviewDraftRejectsClearedCapturedExample verifies that review does not
// accept an empty draft example list when the source supplied a valid example
// for an otherwise input-free command.
func TestReviewDraftRejectsClearedCapturedExample(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	catalog := loadCatalogFixture(t)
	operation := &catalog.Operations[0]
	operation.Parameters = []Parameter{{Name: "regionID", Location: "body", Required: true, Type: "string", Profile: "region"}}
	operation.Examples = []string{"ctyun ecs instance list"}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	if len(commands.Commands[0].Examples) == 0 {
		t.Fatal("generation discarded valid captured source example")
	}
	report := clearGeneratedExamplesAndReview(t, workspace, "ecs")
	finding := "command ecs.instance.list has no example"
	if !slices.Contains(report.Findings, finding) {
		t.Fatalf("review findings %#v do not include %q", report.Findings, finding)
	}
}

// clearGeneratedExamplesAndReview removes the first generated command's
// examples and returns the resulting review report.
func clearGeneratedExamplesAndReview(t *testing.T, workspace Workspace, product string) ReviewReport {
	t.Helper()
	commandsPath := workspace.ProductPath(product, "draft", "commands.json")
	commands := readJSONFile[plugin.Commands](t, commandsPath)
	commands.Commands[0].Examples = nil
	if err := writeJSON(commandsPath, commands); err != nil {
		t.Fatalf("write commands without examples: %v", err)
	}
	report, err := workspace.ReviewDraft(product)
	if err != nil {
		t.Fatalf("ReviewDraft returned error: %v", err)
	}
	return report
}
