/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestAllOpenAPICatalogPluginLabelsMatchCatalog discovers every promoted
// catalog so future products cannot bypass the shared label release policy.
func TestAllOpenAPICatalogPluginLabelsMatchCatalog(t *testing.T) {
	catalogRoot := repoPath(t, "openapi-catalogs")
	entries, err := os.ReadDir(catalogRoot)
	if err != nil {
		t.Fatalf("read OpenAPI catalog root: %v", err)
	}
	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		product := entry.Name()
		if _, err := os.Stat(filepath.Join(catalogRoot, product, "baseline.json")); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("stat %s baseline: %v", product, err)
		}
		checked++
		t.Run(product, func(t *testing.T) {
			assertOpenAPIPluginLabelsMatchCatalog(t, product)
		})
	}
	if checked == 0 {
		t.Fatal("no promoted OpenAPI catalogs found")
	}
}

// assertOpenAPIPluginLabelsMatchCatalog verifies structural table-column
// mapping, source agreement, localization quality, and technical casing.
func assertOpenAPIPluginLabelsMatchCatalog(t *testing.T, product string) {
	t.Helper()
	baselinePath := repoPath(t, filepath.Join("openapi-catalogs", product, "baseline.json"))
	content, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read %s baseline: %v", product, err)
	}
	var baseline openapipipeline.Catalog
	if err := json.Unmarshal(content, &baseline); err != nil {
		t.Fatalf("parse %s baseline: %v", product, err)
	}
	bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", product)), version.Version)
	if err != nil {
		t.Fatalf("load %s plugin: %v", product, err)
	}
	commands := make(map[string]plugin.Command, len(bundle.Commands.Commands))
	for _, command := range bundle.Commands.Commands {
		commands[command.Operation] = command
	}
	for _, operation := range baseline.Operations {
		command, ok := commands[operation.ID]
		if !ok {
			t.Errorf("%s operation %s has no command", product, operation.ID)
			continue
		}
		// Bundle validation permits table-free binary commands; they have no
		// display labels to compare when the catalog also declares no columns.
		if command.Table == "" && len(operation.Response.Columns) == 0 {
			continue
		}
		table, ok := bundle.Tables.Tables[command.Table]
		if !ok {
			t.Errorf("%s operation %s has no table %s", product, operation.ID, command.Table)
			continue
		}
		var report openapipipeline.ReviewReport
		openapipipeline.CheckOperationTable(&report, operation, table)
		for _, finding := range report.Findings {
			t.Errorf("%s: %s", product, finding)
		}
	}
}
