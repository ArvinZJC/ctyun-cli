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
		table, ok := bundle.Tables.Tables[command.Table]
		if !ok {
			t.Errorf("%s operation %s has no table %s", product, operation.ID, command.Table)
			continue
		}
		for _, sourceColumn := range operation.Response.Columns {
			matches := promotedTableColumnMatches(table.Columns, sourceColumn)
			if len(matches) != 1 {
				t.Errorf("%s operation %s response column %s/%s has %d promoted matches, want 1", product, operation.ID, sourceColumn.Key, sourceColumn.Path, len(matches))
				continue
			}
			assertPromotedColumnLabels(t, product, operation.ID, sourceColumn, matches[0])
		}
	}
}

// promotedTableColumnMatches returns promoted columns with the same stable key
// and response path as one baseline column.
func promotedTableColumnMatches(columns []plugin.TableColumn, source openapipipeline.Column) []plugin.TableColumn {
	var matches []plugin.TableColumn
	for _, column := range columns {
		if column.Key == source.Key && column.Path == source.Path {
			matches = append(matches, column)
		}
	}
	return matches
}

// assertPromotedColumnLabels checks source and generated quality plus exact
// agreement after documented canonical normalization.
func assertPromotedColumnLabels(t *testing.T, product string, operationID string, source openapipipeline.Column, generated plugin.TableColumn) {
	t.Helper()
	for _, sourceLabel := range []struct {
		language string
		value    string
	}{
		{language: "en-US", value: source.LabelEN},
		{language: "zh-CN", value: source.LabelZH},
	} {
		if sourceLabel.value == "" {
			continue
		}
		if finding := openapipipeline.DisplayLabelQualityFinding(sourceLabel.language, sourceLabel.value); finding != "" {
			t.Errorf("%s operation %s response column %s source %s label %q %s", product, operationID, source.Path, sourceLabel.language, sourceLabel.value, finding)
		}
	}
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		label := generated.Labels[language]
		if finding := openapipipeline.DisplayLabelQualityFinding(language, label); finding != "" {
			t.Errorf("%s operation %s response column %s generated %s label %q %s", product, operationID, source.Path, language, label, finding)
		}
		sourceLabel := source.LabelEN
		if language == "zh-CN" {
			sourceLabel = source.LabelZH
		}
		if finding := openapipipeline.DisplayLabelAgreementFinding(language, sourceLabel, label); finding != "" {
			t.Errorf("%s operation %s response column %s source %s label %q generated label %q %s", product, operationID, source.Path, language, sourceLabel, label, finding)
		}
	}
}
