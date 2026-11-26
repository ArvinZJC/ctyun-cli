/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestProductDisplayNamesStayAligned verifies official naming policy on both
// tracked snapshots and keeps promoted localization synchronized with README.
func TestProductDisplayNamesStayAligned(t *testing.T) {
	readmeZH := readRepoText(t, "README.md")
	readmeEN := readRepoText(t, "README-EN.md")
	catalogRoot := repoPath(t, "openapi-catalogs")
	pluginsRoot := repoPath(t, "plugins")
	entries, err := os.ReadDir(catalogRoot)
	if err != nil {
		t.Fatalf("read catalog root: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		product := entry.Name()
		t.Run(product, func(t *testing.T) {
			readProductCatalog(t, filepath.Join(catalogRoot, product, "source.json"))
			baselinePath := filepath.Join(catalogRoot, product, "baseline.json")
			pluginDir := filepath.Join(pluginsRoot, product)
			_, baselineErr := os.Stat(baselinePath)
			_, pluginErr := os.Stat(pluginDir)
			if os.IsNotExist(baselineErr) && os.IsNotExist(pluginErr) {
				return // Harvested source awaiting its first promotion.
			}
			baseline := readProductCatalog(t, baselinePath)
			bundle, err := plugin.LoadBundle(pluginDir, version.Version)
			if err != nil {
				t.Fatalf("load plugin: %v", err)
			}
			for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
				want := baseline.Product.DisplayName[language]
				if got := bundle.I18N[language]["name"]; got != want {
					t.Errorf("%s plugin name = %q, baseline = %q", language, got, want)
				}
			}
			assertREADMEProductName(t, readmeZH, baseline.Product.DisplayName["zh-CN"], product)
			assertREADMEProductName(t, readmeEN, baseline.Product.DisplayName["en-US"], product)
		})
	}
}

// TestProductDisplayNameReviewLifecycle runs the release gate in an isolated
// repository through first harvest, pending title changes and missing records.
func TestProductDisplayNameReviewLifecycle(t *testing.T) {
	root := t.TempDir()
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"README.md", "README-EN.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(readRepoText(t, name)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	catalogDir := filepath.Join(root, "openapi-catalogs", "region")
	pluginDir := filepath.Join(root, "plugins", "region")
	if err := os.MkdirAll(catalogDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Dir(pluginDir), 0o700); err != nil {
		t.Fatal(err)
	}
	baseline, err := os.ReadFile(repoPath(t, "openapi-catalogs/region/baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pending openapipipeline.Catalog
	if err := json.Unmarshal(baseline, &pending); err != nil {
		t.Fatal(err)
	}
	pending.Product.DisplayNamePolicy = openapipipeline.DisplayNamePolicy{Kind: "brand", Token: "Region"}
	pending.Product.DisplayName = map[string]string{"en-US": "Region", "en-GB": "Region", "zh-CN": "资源池 Region"}
	source, err := json.Marshal(pending)
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(catalogDir, "source.json")
	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	check := func(stage string, success bool) {
		t.Helper()
		cmd := exec.Command(testBinary, "-test.run=^TestProductDisplayNamesStayAligned$/^region$")
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if (err == nil) != success {
			t.Errorf("%s: %v\n%s", stage, err, output)
		}
	}
	check("source awaiting its first promotion", true)
	if err := os.CopyFS(pluginDir, os.DirFS(repoPath(t, "plugins/region"))); err != nil {
		t.Fatal(err)
	}
	check("published plugin missing baseline", false)
	if err := os.WriteFile(filepath.Join(catalogDir, "baseline.json"), baseline, 0o600); err != nil {
		t.Fatal(err)
	}
	check("changed source policy awaiting promotion", true)
	if err := os.Remove(sourcePath); err != nil {
		t.Fatal(err)
	}
	check("published plugin missing source", false)
}

// readProductCatalog reads and validates one repository catalog snapshot.
func readProductCatalog(t *testing.T, path string) openapipipeline.Catalog {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var catalog openapipipeline.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	if err := catalog.Validate(); err != nil {
		t.Fatalf("validate %s: %v", path, err)
	}
	return catalog
}

// readRepoText reads a repository text file for cross-artifact assertions.
func readRepoText(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(repoPath(t, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

// assertREADMEProductName checks one plugin-table row by its stable plugin ID.
func assertREADMEProductName(t *testing.T, readme, displayName, product string) {
	t.Helper()
	for line := range strings.SplitSeq(readme, "\n") {
		if !strings.Contains(line, "| `"+product+"`") {
			continue
		}
		columns := strings.Split(line, "|")
		if len(columns) < 3 {
			t.Fatalf("malformed README row for %s: %s", product, line)
		}
		if got := strings.TrimSpace(columns[1]); got != displayName {
			t.Fatalf("README name for %s = %q, want %q", product, got, displayName)
		}
		return
	}
	t.Fatalf("README row for plugin %s is missing", product)
}
