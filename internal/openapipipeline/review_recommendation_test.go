/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

const (
	recommendationTargetMethod = "POST"
	recommendationTargetPath   = "/v4.2/monitor/query-history-metric-data"
)

var recommendationTargetCommand = plugin.CommandTarget{
	Plugin: "monitor",
	Path:   []string{"monitor", "metric", "history"},
}

func TestReviewUnresolvedRecommendationIsReadyWithNote(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	source := recommendationSourceCatalog(t, false)
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)

	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("ReviewDraft = %#v, %v", report, err)
	}
	if len(report.Notes) != 1 || !strings.Contains(report.Notes[0], recommendationTargetMethod+" "+recommendationTargetPath+" is outside tracked catalogs") {
		t.Fatalf("notes = %#v", report.Notes)
	}
}

func TestReviewUnclassifiedRecommendationTextIsBlocking(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	source := loadCatalogFixture(t)
	source.Operations[0].Description["zh-CN"] = "推荐使用新版监控查询接口"
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list has recommendation wording without recommendation metadata"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewExternalRecommendationRejectsCommandMapping(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	source := recommendationSourceCatalog(t, true)
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list recommendation target POST /v4.2/monitor/query-history-metric-data is outside tracked catalogs but declares target command ctyun monitor metric history"
	if !slices.Contains(report.Findings, want) || len(report.Notes) != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestReviewTrackedRecommendationRejectsMultipleOwners(t *testing.T) {
	workspace, _ := resolvedRecommendationWorkspace(t)
	duplicate := recommendationTargetCatalog(t, "observability")
	if err := workspace.WriteCatalog(workspace.ProductPath("observability", "source.json"), duplicate); err != nil {
		t.Fatalf("write duplicate owner: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list recommendation target POST /v4.2/monitor/query-history-metric-data is owned by multiple tracked catalogs: monitor, observability"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewResolvedRecommendationIsReady(t *testing.T) {
	workspace, _ := resolvedRecommendationWorkspace(t)
	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("ReviewDraft = %#v, %v", report, err)
	}
	if _, err := plugin.LoadBundle(workspace.ProductPath("ecs", "draft"), "0.3.1"); err != nil {
		t.Fatalf("load resolved recommendation draft: %v", err)
	}
}

func TestReviewTrackedRecommendationRequiresTargetCommand(t *testing.T) {
	workspace, source := resolvedRecommendationWorkspace(t)
	source.Operations[0].Recommendation.TargetCommand = nil
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list recommendation target POST /v4.2/monitor/query-history-metric-data is tracked by plugin monitor but has no target_command"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewTrackedRecommendationRejectsWrongPlugin(t *testing.T) {
	workspace, source := resolvedRecommendationWorkspace(t)
	source.Operations[0].Recommendation.TargetCommand.Plugin = "observability"
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list target command plugin observability does not match tracked owner monitor for POST /v4.2/monitor/query-history-metric-data"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewTrackedRecommendationRejectsStaleCommandPath(t *testing.T) {
	workspace, source := resolvedRecommendationWorkspace(t)
	source.Operations[0].Recommendation.TargetCommand.Path = []string{"monitor", "metric", "stale"}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list target command ctyun monitor metric stale does not resolve in promoted plugin monitor"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewTrackedRecommendationRequiresPromotedPlugin(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	target := recommendationTargetCatalog(t, "monitor")
	if err := workspace.WriteCatalog(workspace.ProductPath("monitor", "source.json"), target); err != nil {
		t.Fatalf("write tracked target: %v", err)
	}
	source := recommendationSourceCatalog(t, true)
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list target command ctyun monitor metric history has no valid promoted plugin monitor"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewTrackedRecommendationRejectsCommandMappedToAnotherAPI(t *testing.T) {
	workspace, _ := resolvedRecommendationWorkspace(t)
	apisPath := filepath.Join(workspace.Root, "plugins", "monitor", "apis.json")
	apis := readJSONFile[plugin.APIs](t, apisPath)
	operation := apis.Operations["v4.2.monitor.metric.history"]
	operation.Path = "/v4.2/monitor/query-current-metric-data"
	apis.Operations["v4.2.monitor.metric.history"] = operation
	if err := writeJSON(apisPath, apis); err != nil {
		t.Fatalf("write mismatched target APIs: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list target command ctyun monitor metric history maps to POST /v4.2/monitor/query-current-metric-data, want POST /v4.2/monitor/query-history-metric-data"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewTrackedRecommendationRejectsCommandWithoutAPIOperation(t *testing.T) {
	workspace, _ := resolvedRecommendationWorkspace(t)
	commandsPath := filepath.Join(workspace.Root, "plugins", "monitor", "commands.json")
	commands := readJSONFile[plugin.Commands](t, commandsPath)
	commands.Commands[0].Operation = ""
	if err := writeJSON(commandsPath, commands); err != nil {
		t.Fatalf("write target command without operation: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list target command ctyun monitor metric history maps to no API operation, want POST /v4.2/monitor/query-history-metric-data"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewTrackedRecommendationRejectsDeprecatedTarget(t *testing.T) {
	for _, target := range []string{"command", "operation"} {
		t.Run(target, func(t *testing.T) {
			workspace, _ := resolvedRecommendationWorkspace(t)
			deprecation := &plugin.Deprecation{Status: "deprecated", Notice: "Deprecated upstream"}
			if target == "command" {
				commandsPath := filepath.Join(workspace.Root, "plugins", "monitor", "commands.json")
				commands := readJSONFile[plugin.Commands](t, commandsPath)
				commands.Commands[0].Deprecation = deprecation
				if err := writeJSON(commandsPath, commands); err != nil {
					t.Fatalf("write deprecated command: %v", err)
				}
			} else {
				apisPath := filepath.Join(workspace.Root, "plugins", "monitor", "apis.json")
				apis := readJSONFile[plugin.APIs](t, apisPath)
				operation := apis.Operations["v4.2.monitor.metric.history"]
				operation.Deprecation = deprecation
				apis.Operations["v4.2.monitor.metric.history"] = operation
				if err := writeJSON(apisPath, apis); err != nil {
					t.Fatalf("write deprecated operation: %v", err)
				}
			}

			report := reviewRecommendationReport(t, workspace, "ecs")
			want := "operation v4.ecs.instance.list target command ctyun monitor metric history is deprecated"
			if !slices.Contains(report.Findings, want) {
				t.Fatalf("findings missing %q: %#v", want, report.Findings)
			}
		})
	}
}

func TestReviewRecommendationCycleIsStableAndBlocking(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	left := cycleCatalog(t, "a", "/a", "/b")
	right := cycleCatalog(t, "b", "/b", "/a")
	if err := workspace.WriteCatalog(workspace.ProductPath("a", "source.json"), left); err != nil {
		t.Fatalf("write left cycle catalog: %v", err)
	}
	if err := workspace.WriteCatalog(workspace.ProductPath("b", "source.json"), right); err != nil {
		t.Fatalf("write right cycle catalog: %v", err)
	}
	if err := workspace.GenerateDraft("a"); err != nil {
		t.Fatalf("generate left cycle draft: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "a")
	want := "recommendation cycle detected: POST /a -> POST /b -> POST /a"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
	if got := recommendationGraphCycles([]Catalog{right, left}); len(got) != 1 || strings.Join(got[0], " -> ") != "POST /a -> POST /b -> POST /a" {
		t.Fatalf("cycles = %#v", got)
	}
}

func TestReviewRecommendationCycleCanonicalizesIncomingTraversal(t *testing.T) {
	catalog := Catalog{Operations: []Operation{
		{Method: "POST", Path: "/a", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/c"}}},
		{Method: "POST", Path: "/c", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/b"}}},
		{Method: "POST", Path: "/b", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/c"}}},
	}}
	got := recommendationGraphCycles([]Catalog{catalog})
	if len(got) != 1 || strings.Join(got[0], " -> ") != "POST /b -> POST /c -> POST /b" {
		t.Fatalf("cycles = %#v", got)
	}
}

func TestReviewGeneratedRecommendationDriftIsBlocking(t *testing.T) {
	workspace, _ := resolvedRecommendationWorkspace(t)
	commandsPath := workspace.ProductPath("ecs", "draft", "commands.json")
	commands := readJSONFile[plugin.Commands](t, commandsPath)
	commands.Commands[0].Recommendation.TargetCommand.Path = []string{"monitor", "metric", "stale"}
	if err := writeJSON(commandsPath, commands); err != nil {
		t.Fatalf("write drifted recommendation: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list generated recommendation does not match source target command"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

func TestReviewUnresolvedRecommendationPromotionPreservesEvidenceOnly(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	source := recommendationSourceCatalog(t, false)
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)
	targetDir := filepath.Join(workspace.Root, "plugins", "ecs")
	if err := workspace.PromoteDraft("ecs", targetDir); err != nil {
		t.Fatalf("PromoteDraft returned error: %v", err)
	}

	baseline := readCatalogFile(t, workspace.ProductPath("ecs", "baseline.json"))
	gotEvidence := baseline.Operations[0].Recommendation
	wantEvidence := source.Operations[0].Recommendation
	if gotEvidence == nil || gotEvidence.Notice != wantEvidence.Notice || gotEvidence.TargetAPI != wantEvidence.TargetAPI || gotEvidence.TargetCommand != nil {
		t.Fatalf("baseline recommendation = %#v, want %#v", gotEvidence, wantEvidence)
	}
	commands := readJSONFile[plugin.Commands](t, filepath.Join(targetDir, "commands.json"))
	if commands.Commands[0].Recommendation != nil {
		t.Fatalf("promoted command recommendation = %#v, want nil", commands.Commands[0].Recommendation)
	}
}

func TestReviewRecommendationMarkdownSeparatesFindingsAndNotes(t *testing.T) {
	report := ReviewReport{
		Product:  "ecs",
		Quality:  "generated",
		Findings: []string{"blocking mismatch"},
		Notes:    []string{"external target remains unresolved"},
	}
	markdown := report.Markdown()
	findings := strings.Index(markdown, "## Findings\n\n- blocking mismatch")
	notes := strings.Index(markdown, "## Notes\n\n- external target remains unresolved")
	if findings < 0 || notes < 0 || findings >= notes {
		t.Fatalf("Markdown() did not separate findings before notes:\n%s", markdown)
	}
}

func TestReviewRecommendationReadsTrackedCatalogFailures(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	if _, err := workspace.readTrackedCatalogs(); err == nil {
		t.Fatal("readTrackedCatalogs returned nil error without catalog root")
	}
	source := loadCatalogFixture(t)
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)
	root := filepath.Join(workspace.Root, "openapi-catalogs")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("not a catalog"), 0o644); err != nil {
		t.Fatalf("write non-catalog entry: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatalf("create empty catalog directory: %v", err)
	}
	invalidPath := workspace.ProductPath("invalid", "source.json")
	if err := os.MkdirAll(filepath.Dir(invalidPath), 0o755); err != nil {
		t.Fatalf("create invalid catalog directory: %v", err)
	}
	if err := os.WriteFile(invalidPath, []byte(`{`), 0o644); err != nil {
		t.Fatalf("write invalid tracked catalog: %v", err)
	}
	if _, err := workspace.ReviewDraft("ecs"); err == nil {
		t.Fatal("ReviewDraft returned nil error for invalid tracked catalog")
	}
}

func TestReviewRecommendationOwnerOrderingUsesOperationID(t *testing.T) {
	target := APIReference{Method: "POST", Path: recommendationTargetPath}
	catalog := recommendationTargetCatalog(t, "monitor")
	second := catalog.Operations[0]
	second.ID = "a.operation"
	catalog.Operations[0].ID = "z.operation"
	catalog.Operations = append(catalog.Operations, second)
	owners := targetAPIOwners([]Catalog{catalog}, target)
	if len(owners) != 2 || owners[0].Operation.ID != "a.operation" || owners[1].Operation.ID != "z.operation" {
		t.Fatalf("owners = %#v", owners)
	}
}

func TestReviewRecommendationRejectsMissingTargetPath(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	operation.Recommendation = &APIRecommendation{
		Notice:    "Use the preferred API",
		TargetAPI: APIReference{Method: "POST"},
	}
	if err := operation.Validate(); err == nil || !strings.Contains(err.Error(), "target path is required") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func recommendationSourceCatalog(t *testing.T, withCommand bool) Catalog {
	t.Helper()
	source := loadCatalogFixture(t)
	recommendation := &APIRecommendation{
		Notice: "推荐使用新版监控查询接口",
		TargetAPI: APIReference{
			Method:  recommendationTargetMethod,
			Path:    recommendationTargetPath,
			DocsURL: "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=49&api=9001",
		},
	}
	if withCommand {
		target := recommendationTargetCommand
		target.Path = slices.Clone(target.Path)
		recommendation.TargetCommand = &target
	}
	source.Operations[0].Recommendation = recommendation
	return source
}

func recommendationTargetCatalog(t *testing.T, product string) Catalog {
	t.Helper()
	target := loadCatalogFixture(t)
	target.Product.PluginName = product
	target.Product.APIProduct = product
	target.Product.CtyunProductID = 49
	target.Product.APIScope = plugin.APIScope{IncludeURIPrefixes: []string{"/v4.2/monitor/"}}
	operation := target.Operations[0]
	operation.ID = "v4.2.monitor.metric.history"
	operation.APIID = "9001"
	operation.Method = recommendationTargetMethod
	operation.Path = recommendationTargetPath
	operation.CommandPath = []string{"metric", "history"}
	operation.Examples = nil
	operation.Recommendation = nil
	target.Operations = []Operation{operation}
	return target
}

func resolvedRecommendationWorkspace(t *testing.T) (Workspace, Catalog) {
	t.Helper()
	workspace := Workspace{Root: t.TempDir()}
	target := recommendationTargetCatalog(t, "monitor")
	writeCatalogAndGenerateDraft(t, workspace, "monitor", target)
	if err := workspace.PromoteDraft("monitor", filepath.Join(workspace.Root, "plugins", "monitor")); err != nil {
		t.Fatalf("promote monitor target: %v", err)
	}
	source := recommendationSourceCatalog(t, true)
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)
	return workspace, source
}

func cycleCatalog(t *testing.T, product, path, targetPath string) Catalog {
	t.Helper()
	catalog := loadCatalogFixture(t)
	catalog.Product.PluginName = product
	catalog.Product.APIProduct = product
	catalog.Product.APIScope = plugin.APIScope{IncludeURIPrefixes: []string{"/" + product}}
	operation := catalog.Operations[0]
	operation.ID = product + ".operation"
	operation.Path = path
	operation.CommandPath = []string{"show"}
	operation.Examples = nil
	operation.Recommendation = &APIRecommendation{
		Notice:    "Use the preferred API",
		TargetAPI: APIReference{Method: "POST", Path: targetPath},
	}
	catalog.Operations = []Operation{operation}
	return catalog
}

func reviewRecommendationReport(t *testing.T, workspace Workspace, product string) ReviewReport {
	t.Helper()
	report, err := workspace.ReviewDraft(product)
	if err != nil {
		t.Fatalf("ReviewDraft returned error: %v", err)
	}
	if report.Ready {
		t.Fatalf("ReviewDraft unexpectedly ready: %#v", report)
	}
	return report
}
