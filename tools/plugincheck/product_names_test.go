/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
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
			source := readProductCatalog(t, filepath.Join(catalogRoot, product, "source.json"))
			baseline := readProductCatalog(t, filepath.Join(catalogRoot, product, "baseline.json"))
			if source.Product.DisplayNamePolicy != baseline.Product.DisplayNamePolicy {
				t.Fatalf("source display-name policy = %#v, baseline = %#v", source.Product.DisplayNamePolicy, baseline.Product.DisplayNamePolicy)
			}

			bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", product)), version.Version)
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
