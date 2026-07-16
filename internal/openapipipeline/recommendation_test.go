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

// TestGenerateRecommendationCarriesOnlyVisibleCommand verifies drafts exclude
// source-only API coordinates and notice text.
func TestGenerateRecommendationCarriesOnlyVisibleCommand(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations = catalog.Operations[:1]
	catalog.Operations[0].DocsURL = "https://example.test/current-operation"
	catalog.Operations[0].Recommendation = &APIRecommendation{
		Notice: "推荐使用新版监控查询接口",
		TargetAPI: APIReference{
			Method:  "POST",
			Path:    "/v4.2/monitor/query-history-metric-data",
			DocsURL: "https://eop.ctyun.cn/ebp/ctapiDocument/search",
		},
		TargetCommand: &plugin.CommandTarget{
			Plugin: "monitor",
			Path:   []string{"monitor", "metric", "history"},
		},
	}

	commands := buildCommands(catalog)
	got := commands.Commands[0].Recommendation
	if got == nil || got.TargetCommand.Plugin != "monitor" || strings.Join(got.TargetCommand.Path, " ") != "monitor metric history" {
		t.Fatalf("generated recommendation = %#v, want visible monitor command", got)
	}
	raw, err := json.Marshal(commands)
	if err != nil {
		t.Fatalf("marshal generated commands: %v", err)
	}
	for _, unwanted := range []string{"/v4.2/monitor/", "ctapiDocument", "推荐使用"} {
		if strings.Contains(string(raw), unwanted) {
			t.Fatalf("generated commands contain source-only recommendation data %q:\n%s", unwanted, raw)
		}
	}
	if deprecation := buildAPIs(catalog).Operations[catalog.Operations[0].ID].Deprecation; deprecation != nil {
		t.Fatalf("neutral recommendation generated deprecation = %#v, want nil", deprecation)
	}
}

// TestGenerateUnresolvedRecommendationOmitsPluginMetadata verifies unresolved
// upstream targets do not become user-facing command guidance.
func TestGenerateUnresolvedRecommendationOmitsPluginMetadata(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].Recommendation = &APIRecommendation{
		Notice: "推荐使用新版监控查询接口",
		TargetAPI: APIReference{
			Method:  "POST",
			Path:    "/v4.2/monitor/query-history-metric-data",
			DocsURL: "https://eop.ctyun.cn/ebp/ctapiDocument/search",
		},
	}

	commands := buildCommands(catalog)
	if got := commands.Commands[0].Recommendation; got != nil {
		t.Fatalf("generated recommendation = %#v, want nil for unresolved command", got)
	}
}

// TestOperationRecommendationIgnoresModifyUserWording verifies the Chinese
// verb-object boundary in 修改用户 does not masquerade as 改用 guidance.
func TestOperationRecommendationIgnoresModifyUserWording(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	operation.Title = "修改用户信息"
	operation.Description = map[string]string{"zh-CN": "修改用户信息。"}

	if operationHasUnclassifiedRecommendation(operation) {
		t.Fatal("修改用户信息 was classified as recommendation wording")
	}
}

// TestGenerateDeprecatedRecommendationUsesCommandReplacement verifies
// lifecycle guidance uses deprecation replacement metadata exclusively.
func TestGenerateDeprecatedRecommendationUsesCommandReplacement(t *testing.T) {
	catalog := loadCatalogFixture(t)
	operation := &catalog.Operations[0]
	operation.Description["zh-CN"] = "该接口已弃用，推荐使用新版监控查询接口"
	operation.Recommendation = &APIRecommendation{
		Notice: "推荐使用新版监控查询接口",
		TargetAPI: APIReference{
			Method:  "POST",
			Path:    "/v4.2/monitor/query-history-metric-data",
			DocsURL: "https://eop.ctyun.cn/ebp/ctapiDocument/search",
		},
		TargetCommand: &plugin.CommandTarget{
			Plugin: "monitor",
			Path:   []string{"monitor", "metric", "history"},
		},
	}

	commands := buildCommands(catalog)
	if got := commands.Commands[0].Recommendation; got != nil {
		t.Fatalf("deprecated command recommendation = %#v, want nil", got)
	}
	deprecation := buildAPIs(catalog).Operations[operation.ID].Deprecation
	if deprecation == nil || deprecation.Replacement == nil {
		t.Fatalf("generated deprecation = %#v, want command replacement", deprecation)
	}
	want := plugin.Replacement{Kind: "command", Label: "ctyun monitor metric history"}
	if got := *deprecation.Replacement; got != want {
		t.Fatalf("generated replacement = %#v, want %#v", got, want)
	}
}

// TestGenerateTitleLifecycleRecommendationUsesCommandReplacement verifies
// lifecycle wording in the operation title takes precedence over standalone
// recommendation help.
func TestGenerateTitleLifecycleRecommendationUsesCommandReplacement(t *testing.T) {
	catalog := loadCatalogFixture(t)
	operation := &catalog.Operations[0]
	operation.Title = "已弃用接口"
	recommendation := validAPIRecommendation()
	recommendation.TargetCommand = &plugin.CommandTarget{
		Plugin: "monitor",
		Path:   []string{"monitor", "metric", "history"},
	}
	operation.Recommendation = recommendation

	if got := buildCommands(catalog).Commands[0].Recommendation; got != nil {
		t.Fatalf("deprecated command recommendation = %#v, want nil", got)
	}
	deprecation := buildAPIs(catalog).Operations[operation.ID].Deprecation
	if deprecation == nil || deprecation.Replacement == nil {
		t.Fatalf("generated deprecation = %#v, want command replacement", deprecation)
	}
	want := plugin.Replacement{Kind: "command", Label: "ctyun monitor metric history"}
	if got := *deprecation.Replacement; got != want {
		t.Fatalf("generated replacement = %#v, want %#v", got, want)
	}
}

// TestGenerateNoticeLifecycleRecommendationUsesCommandReplacement verifies
// exact upstream recommendation evidence participates in lifecycle precedence.
func TestGenerateNoticeLifecycleRecommendationUsesCommandReplacement(t *testing.T) {
	catalog := loadCatalogFixture(t)
	operation := &catalog.Operations[0]
	recommendation := validAPIRecommendation()
	recommendation.Notice = "This API is deprecated; use the replacement API."
	recommendation.TargetCommand = &plugin.CommandTarget{
		Plugin: "monitor",
		Path:   []string{"monitor", "metric", "history"},
	}
	operation.Recommendation = recommendation

	if got := buildCommands(catalog).Commands[0].Recommendation; got != nil {
		t.Fatalf("deprecated command recommendation = %#v, want nil", got)
	}
	deprecation := buildAPIs(catalog).Operations[operation.ID].Deprecation
	if deprecation == nil || deprecation.Notice != recommendation.Notice || deprecation.Replacement == nil {
		t.Fatalf("generated deprecation = %#v, want notice and command replacement", deprecation)
	}
	want := plugin.Replacement{Kind: "command", Label: "ctyun monitor metric history"}
	if got := *deprecation.Replacement; got != want {
		t.Fatalf("generated replacement = %#v, want %#v", got, want)
	}
}

// TestOperationRecommendationAllowsUnresolvedTargetAPI verifies source evidence
// does not require a reviewed visible command mapping.
func TestOperationRecommendationAllowsUnresolvedTargetAPI(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].Recommendation = validAPIRecommendation()

	if err := catalog.Validate(); err != nil {
		t.Fatalf("Validate() returned error for unresolved recommendation: %v", err)
	}
}

// TestOperationRecommendationRejectsInvalidEvidence covers malformed or unsafe
// source API and command targets.
func TestOperationRecommendationRejectsInvalidEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Operation, *APIRecommendation)
	}{
		{
			name: "missing notice",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.Notice = ""
			},
		},
		{
			name: "missing method",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.Method = ""
			},
		},
		{
			name: "unsupported method",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.Method = "OPTIONS"
			},
		},
		{
			name: "relative path",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.Path = "v4.2/monitor/query-history-metric-data"
			},
		},
		{
			name: "unsafe path",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.Path = "/v4.2/monitor/../query-history-metric-data"
			},
		},
		{
			name: "network path",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.Path = "//example.test/v4.2/monitor"
			},
		},
		{
			name: "terminal traversal segment",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.Path = "/v4.2/monitor/.."
			},
		},
		{
			name: "non HTTPS docs URL",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.DocsURL = "http://example.test/monitor"
			},
		},
		{
			name: "self reference",
			mutate: func(operation *Operation, recommendation *APIRecommendation) {
				recommendation.TargetAPI.Method = operation.Method
				recommendation.TargetAPI.Path = operation.Path
			},
		},
		{
			name: "invalid target command",
			mutate: func(_ *Operation, recommendation *APIRecommendation) {
				recommendation.TargetCommand = &plugin.CommandTarget{Plugin: "monitor", Path: []string{"monitor", "metric history"}}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := loadCatalogFixture(t).Operations[0]
			recommendation := validAPIRecommendation()
			test.mutate(&operation, recommendation)
			operation.Recommendation = recommendation

			if err := operation.Validate(); err == nil {
				t.Fatal("Validate() returned nil, want recommendation validation error")
			}
		})
	}
}

// TestOperationRecommendationAllowsValidTargetCommand verifies a complete
// reviewed visible command mapping is valid catalog evidence.
func TestOperationRecommendationAllowsValidTargetCommand(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	recommendation := validAPIRecommendation()
	recommendation.TargetCommand = &plugin.CommandTarget{Plugin: "monitor", Path: []string{"monitor", "metric", "history"}}
	operation.Recommendation = recommendation

	if err := operation.Validate(); err != nil {
		t.Fatalf("Validate() returned error for valid target command: %v", err)
	}
}

// TestHasRecommendationTextRecognizesRecommendationPhrases covers supported
// Chinese and English recommendation markers.
func TestHasRecommendationTextRecognizesRecommendationPhrases(t *testing.T) {
	for _, text := range []string{
		"推荐使用新版接口",
		"建议使用新版接口",
		"请使用新版接口",
		"改用新版接口",
		"We recommend the newer API.",
		"Prefer the newer API.",
	} {
		if !hasRecommendationText([]string{text}) {
			t.Fatalf("hasRecommendationText(%q) = false, want true", text)
		}
	}
	if hasRecommendationText([]string{"This API returns monitoring data."}) {
		t.Fatal("hasRecommendationText() = true for neutral text")
	}
}

// TestOperationHasUnclassifiedRecommendationRequiresRecommendationOnlyText
// verifies metadata suppresses a classified recommendation candidate.
func TestOperationHasUnclassifiedRecommendationRequiresRecommendationOnlyText(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	operation.Description["zh-CN"] = "推荐使用新版监控查询接口"
	if !operationHasUnclassifiedRecommendation(operation) {
		t.Fatal("operationHasUnclassifiedRecommendation() = false for recommendation-only text")
	}

	operation.Recommendation = validAPIRecommendation()
	if operationHasUnclassifiedRecommendation(operation) {
		t.Fatal("operationHasUnclassifiedRecommendation() = true for classified recommendation")
	}
}

// TestOperationHasUnclassifiedRecommendationGivesDeprecationPrecedence verifies
// explicit lifecycle wording is not reported as unclassified guidance.
func TestOperationHasUnclassifiedRecommendationGivesDeprecationPrecedence(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	operation.Description["zh-CN"] = "该接口已弃用，推荐使用新版监控查询接口"

	if operationHasUnclassifiedRecommendation(operation) {
		t.Fatal("operationHasUnclassifiedRecommendation() = true for deprecation text")
	}
}

// TestCatalogFingerprintTracksRecommendationEvidence verifies recommendation
// source evidence participates in catalog provenance.
func TestCatalogFingerprintTracksRecommendationEvidence(t *testing.T) {
	catalog := loadCatalogFixture(t)
	before := catalogFingerprint(catalog)
	catalog.Operations[0].Recommendation = validAPIRecommendation()

	after := catalogFingerprint(catalog)
	if after == before {
		t.Fatal("catalog fingerprint did not change with recommendation evidence")
	}
	catalog.Operations[0].Recommendation.Applicability = validRecommendationApplicability()
	withApplicability := catalogFingerprint(catalog)
	if withApplicability == after {
		t.Fatal("catalog fingerprint did not change with recommendation applicability")
	}
	catalog.Operations[0].Recommendation.Applicability["en-US"] = "bare-metal images"
	if changed := catalogFingerprint(catalog); changed == withApplicability {
		t.Fatal("catalog fingerprint did not change with applicability content")
	}
}

// TestRecommendationApplicabilityRequiresPublicLocales verifies qualified
// source guidance has exact, non-empty coverage for every public locale.
func TestRecommendationApplicabilityRequiresPublicLocales(t *testing.T) {
	valid := loadCatalogFixture(t)
	valid.Operations[0].Recommendation = validAPIRecommendation()
	valid.Operations[0].Recommendation.Applicability = validRecommendationApplicability()
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate rejected complete applicability: %v", err)
	}
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		t.Run("missing "+language, func(t *testing.T) {
			catalog := loadCatalogFixture(t)
			catalog.Operations[0].Recommendation = validAPIRecommendation()
			catalog.Operations[0].Recommendation.Applicability = validRecommendationApplicability()
			delete(catalog.Operations[0].Recommendation.Applicability, language)
			if err := catalog.Validate(); err == nil {
				t.Fatal("Validate accepted missing applicability locale")
			}
		})
		t.Run("blank "+language, func(t *testing.T) {
			catalog := loadCatalogFixture(t)
			catalog.Operations[0].Recommendation = validAPIRecommendation()
			catalog.Operations[0].Recommendation.Applicability = validRecommendationApplicability()
			catalog.Operations[0].Recommendation.Applicability[language] = " \t "
			if err := catalog.Validate(); err == nil {
				t.Fatal("Validate accepted blank applicability locale")
			}
		})
	}
	extra := loadCatalogFixture(t)
	extra.Operations[0].Recommendation = validAPIRecommendation()
	extra.Operations[0].Recommendation.Applicability = validRecommendationApplicability()
	extra.Operations[0].Recommendation.Applicability["fr-FR"] = "images de machine physique"
	if err := extra.Validate(); err == nil {
		t.Fatal("Validate accepted unsupported applicability locale")
	}
}

// TestDiffCatalogsReportsRecommendationChanges verifies an added target API is
// visible in catalog drift output.
func TestDiffCatalogsReportsRecommendationChanges(t *testing.T) {
	baseline := loadCatalogFixture(t)
	source := loadCatalogFixture(t)
	source.Operations[0].Recommendation = &APIRecommendation{Notice: "推荐使用新版监控查询接口", TargetAPI: APIReference{Method: "POST", Path: "/v4.2/monitor/query-history-metric-data"}}
	got := DiffCatalogs(baseline, source).Markdown()
	want := "Operation `v4.ecs.instance.list` added recommendation target `POST /v4.2/monitor/query-history-metric-data`."
	if !strings.Contains(got, want) {
		t.Fatalf("diff missing %q:\n%s", want, got)
	}
}

// TestDiffCatalogsReportsRemovedAndChangedRecommendationTargets verifies target
// removal and identity changes are visible in drift output.
func TestDiffCatalogsReportsRemovedAndChangedRecommendationTargets(t *testing.T) {
	baseline := loadCatalogFixture(t)
	baseline.Operations[0].Recommendation = validAPIRecommendation()

	removed := loadCatalogFixture(t)
	removedDiff := DiffCatalogs(baseline, removed).Markdown()
	removedWant := "Operation `v4.ecs.instance.list` removed recommendation target `POST /v4.2/monitor/query-history-metric-data`."
	if !strings.Contains(removedDiff, removedWant) {
		t.Fatalf("removed diff missing %q:\n%s", removedWant, removedDiff)
	}

	changed := loadCatalogFixture(t)
	changed.Operations[0].Recommendation = &APIRecommendation{
		Notice:    "推荐使用新版监控查询接口",
		TargetAPI: APIReference{Method: "GET", Path: "/v5/monitor/history"},
	}
	changedDiff := DiffCatalogs(baseline, changed).Markdown()
	changedWant := "Operation `v4.ecs.instance.list` recommendation target changed from `POST /v4.2/monitor/query-history-metric-data` to `GET /v5/monitor/history`."
	if !strings.Contains(changedDiff, changedWant) {
		t.Fatalf("changed diff missing %q:\n%s", changedWant, changedDiff)
	}
}

// TestDiffCatalogsIgnoresRecommendationNoticeAndDocsURLChanges verifies drift
// summaries compare API identity rather than source prose.
func TestDiffCatalogsIgnoresRecommendationNoticeAndDocsURLChanges(t *testing.T) {
	baseline := loadCatalogFixture(t)
	baseline.Operations[0].Recommendation = validAPIRecommendation()
	source := loadCatalogFixture(t)
	source.Operations[0].Recommendation = validAPIRecommendation()
	source.Operations[0].Recommendation.Notice = "Use the newer monitoring query API"
	source.Operations[0].Recommendation.TargetAPI.DocsURL = "https://example.test/newer-documentation"

	if got := DiffCatalogs(baseline, source).Markdown(); !strings.Contains(got, "No drift detected.") {
		t.Fatalf("notice or docs URL produced recommendation drift:\n%s", got)
	}
}

// validAPIRecommendation returns complete source evidence for tests that vary
// one recommendation property at a time.
func validAPIRecommendation() *APIRecommendation {
	return &APIRecommendation{
		Notice: "推荐使用新版监控查询接口",
		TargetAPI: APIReference{
			Method:  "POST",
			Path:    "/v4.2/monitor/query-history-metric-data",
			DocsURL: "https://example.test/monitor/query-history-metric-data",
		},
	}
}

// validRecommendationApplicability returns complete localized source
// applicability for tests that vary one locale at a time.
func validRecommendationApplicability() map[string]string {
	return map[string]string{
		"en-US": "physical-machine images",
		"en-GB": "physical-machine images",
		"zh-CN": "物理机镜像",
	}
}
