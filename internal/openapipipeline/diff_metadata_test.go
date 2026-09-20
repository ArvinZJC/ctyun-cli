/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"strings"
	"testing"
)

// TestDiffMaterialMetadataChanges prevents changed runtime or table behavior
// from being reported as an unchanged catalog.
func TestDiffMaterialMetadataChanges(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Catalog)
	}{
		{"defaults", func(c *Catalog) { c.Operations[0].Parameters[1].Default = "demo" }},
		{"enum", func(c *Catalog) { c.Operations[0].Parameters[1].Enum = []string{"one"} }},
		{"pattern", func(c *Catalog) { c.Operations[0].Parameters[1].Pattern = "^one$" }},
		{"binding", func(c *Catalog) { c.Operations[0].Parameters[1].CLIFlag = "other" }},
		{"retry", func(c *Catalog) { c.Operations[0].Retryable = false }},
		{"dangerous", func(c *Catalog) { c.Operations[0].Dangerous = true }},
		{"table defaults", func(c *Catalog) { c.Operations[0].Response.DefaultColumns = []string{"name"} }},
		{"endpoint", func(c *Catalog) { c.Product.EndpointURL = "https://other.example.com" }},
		{"schema", func(c *Catalog) { c.SchemaVersion++ }},
		{"conditional requirements", func(c *Catalog) { c.Operations[0].ConditionalRequirements = []plugin.ConditionalRequirement{{}} }},
		{"command path", func(c *Catalog) { c.Operations[0].CommandPath = []string{"another", "path"} }},
		{"response result", func(c *Catalog) { c.Operations[0].Response.ResultPath = "other" }},
		{"documentation", func(c *Catalog) { c.Operations[0].Title = "Changed upstream title" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseline, source := loadCatalogFixture(t), loadCatalogFixture(t)
			tc.mutate(&source)
			if len(DiffCatalogs(baseline, source).Changes) == 0 {
				t.Fatal("material change hidden")
			}
		})
	}
}

// TestEVSSnapshotDefaultColumnDrift checks the previously invisible response
// presentation change without depending on or modifying the live source drift.
func TestEVSSnapshotDefaultColumnDrift(t *testing.T) {
	baseline, source := loadCatalogFixture(t), loadCatalogFixture(t)
	baseline.Product.PluginName, source.Product.PluginName = "evs", "evs"
	baseline.Operations[0].ID, source.Operations[0].ID = "v4.evs.snapshot.list", "v4.evs.snapshot.list"
	baseline.Operations[0].Response.DefaultColumns = []string{"snapshot_id"}
	source.Operations[0].Response.DefaultColumns = []string{"snapshot_id", "status"}
	if got := DiffCatalogs(baseline, source).Markdown(); !strings.Contains(got, "v4.evs.snapshot.list") || !strings.Contains(got, "default columns") {
		t.Fatal(got)
	}
}

// TestDiffVisibleRecommendationGuidance reports changed CLI help mappings
// independently of intentionally ignored upstream prose-only revisions.
func TestDiffVisibleRecommendationGuidance(t *testing.T) {
	for _, change := range []func(*APIRecommendation){
		func(r *APIRecommendation) { r.Applicability = map[string]string{"en-US": "private images"} },
		func(r *APIRecommendation) {
			r.TargetCommand = &plugin.CommandTarget{Plugin: "other", Path: []string{"other", "show"}}
		},
	} {
		baseline, source := loadCatalogFixture(t), loadCatalogFixture(t)
		baseline.Operations[0].Recommendation = &APIRecommendation{Notice: "original", TargetAPI: APIReference{Method: "GET", Path: "/target", DocsURL: "https://example.test/old"}}
		copy := *baseline.Operations[0].Recommendation
		source.Operations[0].Recommendation = &copy
		change(source.Operations[0].Recommendation)
		report := DiffCatalogs(baseline, source).Markdown()
		if !strings.Contains(report, "visible recommendation guidance changed") {
			t.Fatal(report)
		}
		if strings.Contains(report, "https://example.test") || strings.Contains(report, "different upstream guidance") {
			t.Fatalf("raw evidence in summary: %s", report)
		}
	}
}
