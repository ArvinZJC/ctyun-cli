/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestOperationRecommendationAllowsUnresolvedTargetAPI(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].Recommendation = validAPIRecommendation()

	if err := catalog.Validate(); err != nil {
		t.Fatalf("Validate() returned error for unresolved recommendation: %v", err)
	}
}

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

func TestOperationRecommendationAllowsValidTargetCommand(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	recommendation := validAPIRecommendation()
	recommendation.TargetCommand = &plugin.CommandTarget{Plugin: "monitor", Path: []string{"monitor", "metric", "history"}}
	operation.Recommendation = recommendation

	if err := operation.Validate(); err != nil {
		t.Fatalf("Validate() returned error for valid target command: %v", err)
	}
}

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

func TestOperationHasUnclassifiedRecommendationGivesDeprecationPrecedence(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	operation.Description["zh-CN"] = "该接口已弃用，推荐使用新版监控查询接口"

	if operationHasUnclassifiedRecommendation(operation) {
		t.Fatal("operationHasUnclassifiedRecommendation() = true for deprecation text")
	}
}

func TestCatalogFingerprintTracksRecommendationEvidence(t *testing.T) {
	catalog := loadCatalogFixture(t)
	before := catalogFingerprint(catalog)
	catalog.Operations[0].Recommendation = validAPIRecommendation()

	if after := catalogFingerprint(catalog); after == before {
		t.Fatal("catalog fingerprint did not change with recommendation evidence")
	}
}

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
