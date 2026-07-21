/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"os"
	"testing"
)

// TestNormalizeSourceLabelsAppliesOnlyTrustedRepairs verifies that the
// maintenance step fixes canonical casing and shared phrases while preserving
// unresolved source evidence for review.
func TestNormalizeSourceLabelsAppliesOnlyTrustedRepairs(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := loadCatalogFixture(t)
	for operationIndex := range catalog.Operations {
		for columnIndex := range catalog.Operations[operationIndex].Response.Columns {
			column := &catalog.Operations[operationIndex].Response.Columns[columnIndex]
			column.LabelEN = NormalizeDisplayLabel("en-US", column.LabelEN)
			column.LabelZH = NormalizeDisplayLabel("zh-CN", column.LabelZH)
		}
	}
	columns := catalog.Operations[0].Response.Columns
	columns[0].LabelEN = "Cmk UUID"
	columns[0].LabelZH = "实例ID"
	columns[1].Key = "created_time"
	columns[1].Path = "createdTime"
	columns[1].LabelZH = "Created Time"
	columns[2].Key = "custom_field"
	columns[2].Path = "customField"
	columns[2].LabelZH = "customField"
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatalf("write source: %v", err)
	}

	changed, err := workspace.NormalizeSourceLabels("ecs")
	if err != nil {
		t.Fatalf("NormalizeSourceLabels returned error: %v", err)
	}
	if changed != 3 {
		t.Fatalf("changed labels = %d, want 3", changed)
	}
	normalized, err := workspace.ReadSource("ecs")
	if err != nil {
		t.Fatalf("read normalized source: %v", err)
	}
	got := normalized.Operations[0].Response.Columns
	if got[0].LabelEN != "CMK UUID" || got[0].LabelZH != "实例 ID" {
		t.Fatalf("canonical labels = %q/%q", got[0].LabelEN, got[0].LabelZH)
	}
	if got[1].LabelZH != "创建时间" {
		t.Fatalf("shared phrase label = %q, want 创建时间", got[1].LabelZH)
	}
	if got[2].LabelZH != "customField" {
		t.Fatalf("unresolved source label = %q, want customField", got[2].LabelZH)
	}
	changed, err = workspace.NormalizeSourceLabels("ecs")
	if err != nil || changed != 0 {
		t.Fatalf("second NormalizeSourceLabels = %d, %v", changed, err)
	}
}

// TestNormalizeSourceLabelsReportsReadAndWriteFailures verifies maintenance
// failures are surfaced without claiming that labels were normalized.
func TestNormalizeSourceLabelsReportsReadAndWriteFailures(t *testing.T) {
	if _, err := (Workspace{Root: t.TempDir()}).NormalizeSourceLabels("missing"); err == nil {
		t.Fatal("NormalizeSourceLabels returned nil for a missing source")
	}

	workspace := Workspace{Root: t.TempDir()}
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].Response.Columns[0].LabelEN = "Cmk UUID"
	path := workspace.ProductPath("ecs", "source.json")
	if err := workspace.WriteCatalog(path, catalog); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatalf("make source read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if _, err := workspace.NormalizeSourceLabels("ecs"); err == nil {
		t.Fatal("NormalizeSourceLabels returned nil for an unwritable source")
	}
}
