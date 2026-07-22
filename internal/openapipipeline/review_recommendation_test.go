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

// Recommendation target fixtures define one tracked monitor API identity.
const (
	recommendationTargetMethod = "POST"
	recommendationTargetPath   = "/v4.2/monitor/query-history-metric-data"
)

// recommendationTargetCommand is the reviewed visible command for the target
// API fixture.
var recommendationTargetCommand = plugin.CommandTarget{
	Plugin: "monitor",
	Path:   []string{"monitor", "metric", "history"},
}

// TestReviewUnresolvedRecommendationIsReadyWithNote verifies an external API
// target remains promotable without visible command metadata.
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

// TestReviewUnclassifiedRecommendationTextIsBlocking verifies detected guidance
// must be classified explicitly.
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

// TestReviewExternalRecommendationRejectsCommandMapping verifies command
// metadata cannot claim an API outside all tracked catalogs.
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

// TestReviewTrackedRecommendationRejectsMultipleOwners verifies target API
// ownership must be unique across tracked catalogs.
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

// TestReviewResolvedRecommendationIsReady verifies exact promoted target
// metadata passes repository review.
func TestReviewResolvedRecommendationIsReady(t *testing.T) {
	workspace, _ := resolvedRecommendationWorkspace(t)
	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("ReviewDraft = %#v, %v", report, err)
	}
	if _, err := plugin.LoadBundle(workspace.ProductPath("ecs", "draft"), "0.4.0"); err != nil {
		t.Fatalf("load resolved recommendation draft: %v", err)
	}
}

// TestReviewTrackedRecommendationRequiresTargetCommand verifies tracked target
// APIs require a visible command mapping.
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

// TestReviewTrackedRecommendationRejectsWrongPlugin verifies target command
// ownership matches the tracked target catalog.
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

// TestReviewTrackedRecommendationRejectsStaleCommandPath verifies target paths
// resolve exactly in promoted plugin metadata.
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

// TestReviewTrackedRecommendationRequiresPromotedPlugin verifies ignored drafts
// cannot satisfy a repository command reference.
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

// TestReviewTrackedRecommendationRejectsCommandMappedToAnotherAPI verifies the
// promoted command must own the exact target HTTP identity.
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

// TestReviewTrackedRecommendationRejectsCommandWithoutAPIOperation verifies a
// visible command without HTTP metadata cannot satisfy the target.
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

// TestReviewTrackedRecommendationRejectsDeprecatedTarget verifies promoted
// command and operation lifecycle metadata both block guidance.
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

// TestReviewTrackedRecommendationRejectsLifecycleSourceDrift verifies current
// tracked lifecycle evidence blocks a target before its older promoted bundle
// is regenerated.
func TestReviewTrackedRecommendationRejectsLifecycleSourceDrift(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Operation)
	}{
		{
			name: "title",
			mutate: func(operation *Operation) {
				operation.Title = "已弃用接口"
			},
		},
		{
			name: "localized description",
			mutate: func(operation *Operation) {
				operation.Description["zh-CN"] = "该接口即将下线"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace, _ := resolvedRecommendationWorkspace(t)
			target := readCatalogFile(t, workspace.ProductPath("monitor", "source.json"))
			test.mutate(&target.Operations[0])
			if err := workspace.WriteCatalog(workspace.ProductPath("monitor", "source.json"), target); err != nil {
				t.Fatalf("write drifted target source: %v", err)
			}

			report := reviewRecommendationReport(t, workspace, "ecs")
			want := "operation v4.ecs.instance.list recommendation target POST /v4.2/monitor/query-history-metric-data is deprecated in current tracked source"
			if !slices.Contains(report.Findings, want) {
				t.Fatalf("findings missing %q: %#v", want, report.Findings)
			}
		})
	}
}

// TestReviewTrackedRecommendationRejectsNoticeLifecycleSourceDrift verifies
// preserved recommendation evidence blocks a lifecycle-marked target before
// its promoted plugin is regenerated.
func TestReviewTrackedRecommendationRejectsNoticeLifecycleSourceDrift(t *testing.T) {
	workspace, _ := resolvedRecommendationWorkspace(t)
	target := readCatalogFile(t, workspace.ProductPath("monitor", "source.json"))
	target.Operations[0].Recommendation = &APIRecommendation{
		Notice:    "This API is obsolete; use the replacement API.",
		TargetAPI: APIReference{Method: "POST", Path: "/v5/monitor/history"},
	}
	if err := workspace.WriteCatalog(workspace.ProductPath("monitor", "source.json"), target); err != nil {
		t.Fatalf("write drifted target source: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list recommendation target POST /v4.2/monitor/query-history-metric-data is deprecated in current tracked source"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

// TestReviewRecommendationCycleIsStableAndBlocking verifies repository cycles
// are canonical and block promotion regardless of catalog order.
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
	if got := analyzeRecommendationGraph([]Catalog{right, left}).Cycles; len(got) != 1 || strings.Join(got[0], " -> ") != "POST /a -> POST /b -> POST /a" {
		t.Fatalf("cycles = %#v", got)
	}
}

// TestReviewRecommendationCycleCanonicalizesIncomingTraversal verifies an
// incoming path does not alter the reported cycle boundary.
func TestReviewRecommendationCycleCanonicalizesIncomingTraversal(t *testing.T) {
	catalog := Catalog{Operations: []Operation{
		{Method: "POST", Path: "/a", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/c"}}},
		{Method: "POST", Path: "/c", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/b"}}},
		{Method: "POST", Path: "/b", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/c"}}},
	}}
	got := analyzeRecommendationGraph([]Catalog{catalog}).Cycles
	if len(got) != 1 || strings.Join(got[0], " -> ") != "POST /b -> POST /c -> POST /b" {
		t.Fatalf("cycles = %#v", got)
	}
}

// TestRecommendationGraphSortsMultipleCycles verifies independent cycles have
// a deterministic repository-wide order.
func TestRecommendationGraphSortsMultipleCycles(t *testing.T) {
	catalog := Catalog{Operations: []Operation{
		{Method: "POST", Path: "/z", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/y"}}},
		{Method: "POST", Path: "/y", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/z"}}},
		{Method: "POST", Path: "/b", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/a"}}},
		{Method: "POST", Path: "/a", Recommendation: &APIRecommendation{TargetAPI: APIReference{Method: "POST", Path: "/b"}}},
	}}
	got := analyzeRecommendationGraph([]Catalog{catalog}).Cycles
	if len(got) != 2 || strings.Join(got[0], " -> ") != "POST /a -> POST /b -> POST /a" || strings.Join(got[1], " -> ") != "POST /y -> POST /z -> POST /y" {
		t.Fatalf("cycles = %#v", got)
	}
}

// TestReviewRecommendationRejectsAmbiguousSourceIdentity verifies duplicate
// operations cannot branch one HTTP API identity to multiple targets.
func TestReviewRecommendationRejectsAmbiguousSourceIdentity(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := overlappingCycleCatalog(t)
	writeCatalogAndGenerateDraft(t, workspace, "cycles", catalog)

	report := reviewRecommendationReport(t, workspace, "cycles")
	var cycleFindings []string
	for _, finding := range report.Findings {
		if strings.HasPrefix(finding, "recommendation cycle detected:") {
			cycleFindings = append(cycleFindings, finding)
		}
	}
	want := "recommendation source POST /a has ambiguous targets: POST /b, POST /c"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
	if len(cycleFindings) != 0 {
		t.Fatalf("ambiguous source produced cycle findings: %#v", cycleFindings)
	}
	if got := analyzeRecommendationGraph([]Catalog{catalog}).Cycles; len(got) != 0 {
		t.Fatalf("ambiguous source produced cycles: %#v", got)
	}
}

// TestReviewGeneratedRecommendationDriftIsBlocking verifies draft help metadata
// must match the reviewed source command target.
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

// TestReviewGeneratedRecommendationApplicabilityDriftIsBlocking verifies the
// generated English fallback must match qualified source evidence.
func TestReviewGeneratedRecommendationApplicabilityDriftIsBlocking(t *testing.T) {
	workspace, source := resolvedRecommendationWorkspace(t)
	source.Operations[0].Recommendation.Applicability = validRecommendationApplicability()
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)
	commandsPath := workspace.ProductPath("ecs", "draft", "commands.json")
	commands := readJSONFile[plugin.Commands](t, commandsPath)
	commands.Commands[0].Recommendation.Applicability = "all images"
	if err := writeJSON(commandsPath, commands); err != nil {
		t.Fatalf("write drifted applicability: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list generated recommendation does not match source target command"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

// TestReviewGeneratedRecommendationLocaleDriftIsBlocking verifies draft i18n
// qualifiers must match each localized source value.
func TestReviewGeneratedRecommendationLocaleDriftIsBlocking(t *testing.T) {
	workspace, source := resolvedRecommendationWorkspace(t)
	source.Operations[0].Recommendation.Applicability = validRecommendationApplicability()
	writeCatalogAndGenerateDraft(t, workspace, "ecs", source)
	command := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json")).Commands[0]
	localePath := workspace.ProductPath("ecs", "draft", "i18n", "zh-CN.json")
	locale := readJSONFile[map[string]string](t, localePath)
	locale[plugin.RecommendationApplicabilityKey(command.ID)] = "全部镜像"
	if err := writeJSON(localePath, locale); err != nil {
		t.Fatalf("write drifted locale: %v", err)
	}

	report := reviewRecommendationReport(t, workspace, "ecs")
	want := "operation v4.ecs.instance.list generated recommendation applicability zh-CN does not match source"
	if !slices.Contains(report.Findings, want) {
		t.Fatalf("findings missing %q: %#v", want, report.Findings)
	}
}

// TestReviewUnresolvedRecommendationPromotionPreservesEvidenceOnly verifies
// baseline evidence advances without manufacturing plugin help metadata.
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

// TestReviewRecommendationMarkdownSeparatesFindingsAndNotes verifies blocking
// and informational review output remain distinct.
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

// TestReviewRecommendationReadsTrackedCatalogFailures covers missing, ignored,
// and malformed tracked catalog entries.
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

// TestReviewRecommendationOwnerOrderingUsesOperationID verifies duplicate owner
// diagnostics are deterministic within one plugin.
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

// TestReviewRecommendationRejectsMissingTargetPath verifies validation fails
// before graph review for incomplete API coordinates.
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

// recommendationSourceCatalog builds source evidence with an optional reviewed
// command target.
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

// recommendationTargetCatalog builds one catalog that owns the monitor API
// fixture.
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

// resolvedRecommendationWorkspace promotes the target plugin and generates the
// source draft in a temporary repository.
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

// cycleCatalog builds one recommendation edge for graph review tests.
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

// overlappingCycleCatalog builds duplicate source identities with distinct
// targets plus paths that would otherwise form overlapping cycles.
func overlappingCycleCatalog(t *testing.T) Catalog {
	t.Helper()
	catalog := loadCatalogFixture(t)
	catalog.Product.PluginName = "cycles"
	catalog.Product.APIProduct = "cycles"
	catalog.Product.APIScope = plugin.APIScope{IncludeURIPrefixes: []string{"/"}}
	template := catalog.Operations[0]
	operation := func(id, path, command, targetPath string) Operation {
		next := template
		next.ID = id
		next.Path = path
		next.CommandPath = []string{command}
		next.Examples = nil
		next.Recommendation = &APIRecommendation{
			Notice:    "Use the preferred API",
			TargetAPI: APIReference{Method: "POST", Path: targetPath},
		}
		return next
	}
	catalog.Operations = []Operation{
		operation("cycle.a.b", "/a", "a-to-b", "/b"),
		operation("cycle.a.c", "/a", "a-to-c", "/c"),
		operation("cycle.b.d", "/b", "b-to-d", "/d"),
		operation("cycle.c.d", "/c", "c-to-d", "/d"),
		operation("cycle.d.a", "/d", "d-to-a", "/a"),
	}
	return catalog
}

// reviewRecommendationReport requires a blocking review result for tests.
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
