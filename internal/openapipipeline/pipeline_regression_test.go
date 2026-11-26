/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestWriteDiffRejectsUnreadableBaseline prevents malformed or unreadable accepted
// evidence from being silently reported as a first-time catalog.
func TestWriteDiffRejectsUnreadableBaseline(t *testing.T) {
	for _, kind := range []string{"malformed", "invalid catalog", "directory"} {
		t.Run(kind, func(t *testing.T) {
			workspace := Workspace{Root: t.TempDir()}
			if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t)); err != nil {
				t.Fatal(err)
			}
			baseline := workspace.ProductPath("ecs", "baseline.json")
			var err error
			switch kind {
			case "malformed":
				err = os.WriteFile(baseline, []byte(`{broken`), 0o644)
			case "invalid catalog":
				err = os.WriteFile(baseline, []byte(`{}`), 0o644)
			case "directory":
				err = os.Mkdir(baseline, 0o755)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := workspace.WriteDiff("ecs"); err == nil {
				t.Fatal("invalid baseline was reported as missing")
			}
			if _, err := os.Stat(workspace.ProductPath("ecs", "changes.md")); !os.IsNotExist(err) {
				t.Fatal("failed diff wrote a misleading report")
			}
		})
	}
}

// TestTypedBodyCompatibilityOnRegeneration prevents promoted release identity
// from retaining a core floor that cannot preserve JSON scalar request types.
func TestTypedBodyCompatibilityOnRegeneration(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := typedBodyCatalog(t)
	promoted := buildManifest(catalog)
	promoted.Requires.Ctyun = ">=0.3.1 <1.0.0"
	if err := writeJSON(filepath.Join(workspace.Root, "plugins", "ecs", "plugin.json"), promoted); err != nil {
		t.Fatal(err)
	}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
	manifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	if manifest.Requires.Ctyun != ">=0.4.0 <1.0.0" {
		t.Fatalf("typed body generated with obsolete core constraint %q", manifest.Requires.Ctyun)
	}
}

// TestReviewRejectsObsoleteTypedBodyCompatibility prevents manual draft edits
// from bypassing the minimum core needed for typed JSON body values.
func TestReviewRejectsObsoleteTypedBodyCompatibility(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", typedBodyCatalog(t))
	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("valid draft failed review: %#v, %v", report, err)
	}
	path := workspace.ProductPath("ecs", "draft", "plugin.json")
	manifest := readJSONFile[plugin.Manifest](t, path)
	manifest.Requires.Ctyun = ">=0.3.1 <1.0.0"
	if err := writeJSON(path, manifest); err != nil {
		t.Fatal(err)
	}
	report, err = workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready {
		t.Fatal("review accepted typed body with obsolete compatibility")
	}
	if err := workspace.PromoteDraft("ecs", filepath.Join(workspace.Root, "plugins", "ecs")); err == nil {
		t.Fatal("promoted incompatible draft")
	}
}

// typedBodyCatalog isolates typed JSON scalar requirements from newer transport
// and independent request-binding features.
func typedBodyCatalog(t *testing.T) Catalog {
	t.Helper()
	catalog := loadCatalogFixture(t)
	catalog.Operations = catalog.Operations[:1]
	operation := &catalog.Operations[0]
	operation.Parameters = []Parameter{{Name: "pageNo", Location: "body", Type: "integer", CLIName: "page_no", CLIFlag: "page-no", Description: "Page number", Descriptions: map[string]string{"en-US": "Page number", "en-GB": "Page number", "zh-CN": "页码"}}}
	operation.Examples = nil
	operation.RequestExample = nil
	return catalog
}
