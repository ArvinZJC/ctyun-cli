/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// storageReviewContext holds one promoted storage plugin and its accepted
// OpenAPI catalog indexed by operation ID.
type storageReviewContext struct {
	t          *testing.T
	product    string
	catalog    openapipipeline.Catalog
	bundle     plugin.Bundle
	operations map[string]openapipipeline.Operation
	commands   map[string]plugin.Command
}

// storageFixtureOptions controls product-specific fixture validation policy.
type storageFixtureOptions struct {
	skipRootTables      bool
	requireUniqueColumn bool
}

// loadStorageReviewContext loads one promoted storage bundle and its baseline
// catalog for semantic release tests.
func loadStorageReviewContext(t *testing.T, product string) storageReviewContext {
	t.Helper()
	content, err := os.ReadFile(repoPath(t, filepath.Join("openapi-catalogs", product, "baseline.json")))
	if err != nil {
		t.Fatal(err)
	}
	var catalog openapipipeline.Catalog
	if err := json.Unmarshal(content, &catalog); err != nil {
		t.Fatal(err)
	}
	bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", product)), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	context := storageReviewContext{
		t:          t,
		product:    product,
		catalog:    catalog,
		bundle:     bundle,
		operations: make(map[string]openapipipeline.Operation, len(catalog.Operations)),
		commands:   make(map[string]plugin.Command, len(bundle.Commands.Commands)),
	}
	for _, operation := range catalog.Operations {
		context.operations[operation.ID] = operation
	}
	for _, command := range bundle.Commands.Commands {
		context.commands[command.Operation] = command
	}
	return context
}

// sourceParameter returns one catalog parameter or fails the active test.
func (context storageReviewContext) sourceParameter(operationID, target string) openapipipeline.Parameter {
	context.t.Helper()
	for _, parameter := range context.operations[operationID].Parameters {
		if parameter.Name == target {
			return parameter
		}
	}
	context.t.Fatalf("%s source parameter %s is missing", operationID, target)
	return openapipipeline.Parameter{}
}

// commandParameter returns one generated command parameter or fails the
// active test.
func (context storageReviewContext) commandParameter(operationID, target string) plugin.Parameter {
	context.t.Helper()
	for _, parameter := range context.commands[operationID].Parameters {
		if parameter.Target == target {
			return parameter
		}
	}
	context.t.Fatalf("%s command parameter %s is missing", operationID, target)
	return plugin.Parameter{}
}

// storageParameterKey splits an operation-and-parameter key at its final dot.
func storageParameterKey(t *testing.T, key string) (string, string) {
	t.Helper()
	separator := strings.LastIndexByte(key, '.')
	if separator <= 0 || separator == len(key)-1 {
		t.Fatalf("invalid storage parameter key %q", key)
	}
	return key[:separator], key[separator+1:]
}

// assertStorageParameterEnums verifies source enums, generated allowed values,
// and the shared Boolean value convention.
func assertStorageParameterEnums(t *testing.T, context storageReviewContext, expected map[string][]string) {
	t.Helper()
	for key, want := range expected {
		operationID, target := storageParameterKey(t, key)
		if got := context.sourceParameter(operationID, target).Enum; !reflect.DeepEqual(got, want) {
			t.Errorf("%s source enum = %#v, want %#v", key, got, want)
		}
		if got := context.commandParameter(operationID, target).AllowedValues; !reflect.DeepEqual(got, want) {
			t.Errorf("%s allowed values = %#v, want %#v", key, got, want)
		}
	}
	booleanValues := []string{"true", "false"}
	for _, operation := range context.catalog.Operations {
		for _, source := range operation.Parameters {
			if source.Type != "Boolean" {
				continue
			}
			if !reflect.DeepEqual(source.Enum, booleanValues) {
				t.Errorf("%s.%s source Boolean enum = %#v, want %#v", operation.ID, source.Name, source.Enum, booleanValues)
			}
			if got := context.commandParameter(operation.ID, source.Name).AllowedValues; !reflect.DeepEqual(got, booleanValues) {
				t.Errorf("%s.%s Boolean allowed values = %#v, want %#v", operation.ID, source.Name, got, booleanValues)
			}
		}
	}
}

// assertStorageParameterNames verifies source and generated CLI names for a
// set of operation parameters.
func assertStorageParameterNames(t *testing.T, context storageReviewContext, expected map[string][2]string) {
	t.Helper()
	for key, want := range expected {
		operationID, target := storageParameterKey(t, key)
		source := context.sourceParameter(operationID, target)
		command := context.commandParameter(operationID, target)
		if source.CLIName != want[0] || source.CLIFlag != want[1] || command.Name != want[0] || command.Flag != want[1] {
			t.Errorf("%s names = source %s/%s command %s/%s, want %s/%s", key, source.CLIName, source.CLIFlag, command.Name, command.Flag, want[0], want[1])
		}
	}
}

// assertStorageLocalizedOptionHelp verifies that reviewed source help is
// localized and copied exactly into generated plugin catalogs.
func assertStorageLocalizedOptionHelp(t *testing.T, context storageReviewContext) {
	t.Helper()
	for _, operation := range context.catalog.Operations {
		command := context.commands[operation.ID]
		for _, source := range operation.Parameters {
			parameter := context.commandParameter(operation.ID, source.Name)
			key := "parameter." + command.ID + "." + parameter.Name + ".description"
			for _, language := range []string{"en-GB", "en-US", "zh-CN"} {
				help := strings.TrimSpace(source.HelpDescriptions[language])
				if help == "" {
					t.Errorf("%s source help %s is blank", key, language)
					continue
				}
				if strings.Contains(help, "<") {
					t.Errorf("%s source help %s contains HTML", key, language)
				}
				if got := context.bundle.I18N[language][key]; got != help {
					t.Errorf("%s %s = %q, want %q", key, language, got, help)
				}
			}
			if source.HelpDescriptions["en-US"] == source.Descriptions["en-US"] || source.HelpDescriptions["en-GB"] == source.Descriptions["en-GB"] {
				t.Errorf("%s retains mechanical English help", key)
			}
			if !strings.ContainsFunc(source.HelpDescriptions["zh-CN"], func(char rune) bool { return unicode.Is(unicode.Han, char) }) {
				t.Errorf("%s zh-CN help is not localized: %q", key, source.HelpDescriptions["zh-CN"])
			}
		}
	}
}

// assertStorageTechnicalAcronymLabels rejects known incorrect acronym casing in source and table labels.
func assertStorageTechnicalAcronymLabels(t *testing.T, context storageReviewContext, wrongCasing []string) {
	t.Helper()
	for _, operation := range context.catalog.Operations {
		table := context.bundle.Tables.Tables[context.commands[operation.ID].Table]
		for _, column := range operation.Response.Columns {
			for _, wrong := range wrongCasing {
				if strings.Contains(column.LabelEN, wrong) {
					t.Errorf("%s source label %s has incorrect acronym casing %q", operation.ID, column.Path, column.LabelEN)
				}
			}
		}
		for _, column := range table.Columns {
			for language, label := range column.Labels {
				for _, wrong := range wrongCasing {
					if strings.Contains(label, wrong) {
						t.Errorf("%s %s label %s has incorrect acronym casing %q", operation.ID, language, column.Path, label)
					}
				}
			}
		}
	}
}

// assertStorageFixtureDefaults verifies that every visible default table
// column exists in an official fixture row.
func assertStorageFixtureDefaults(t *testing.T, context storageReviewContext, options storageFixtureOptions) {
	t.Helper()
	for _, command := range context.bundle.Commands.Commands {
		table := context.bundle.Tables.Tables[command.Table]
		if options.skipRootTables && table.RowPath == "$" {
			continue
		}
		content, err := os.ReadFile(repoPath(t, filepath.Join("plugins", context.product, command.FixtureResponse)))
		if err != nil {
			t.Errorf("%s fixture: %v", command.ID, err)
			continue
		}
		var value any
		if err := json.Unmarshal(content, &value); err != nil {
			t.Errorf("%s fixture JSON: %v", command.ID, err)
			continue
		}
		validPath := true
		if table.RowPath != "$" {
			for part := range strings.SplitSeq(strings.TrimPrefix(table.RowPath, "$."), ".") {
				object, ok := value.(map[string]any)
				if !ok {
					t.Errorf("%s row path %s reaches %T before %s", command.ID, table.RowPath, value, part)
					validPath = false
					break
				}
				value, ok = object[part]
				if !ok {
					t.Errorf("%s row path %s is absent at %s", command.ID, table.RowPath, part)
					validPath = false
					break
				}
			}
		}
		if !validPath {
			continue
		}
		if rows, ok := value.([]any); ok {
			if len(rows) == 0 {
				t.Errorf("%s row path %s has no official fixture row", command.ID, table.RowPath)
				continue
			}
			value = rows[0]
		}
		row, ok := value.(map[string]any)
		if !ok {
			t.Errorf("%s row path %s resolves to %T", command.ID, table.RowPath, value)
			continue
		}
		paths := make(map[string]string, len(table.Columns))
		for _, column := range table.Columns {
			if options.requireUniqueColumn {
				if _, duplicate := paths[column.Key]; duplicate {
					t.Errorf("%s has duplicate table key %s", command.ID, column.Key)
				}
			}
			paths[column.Key] = column.Path
		}
		for _, key := range table.DefaultColumns {
			path := paths[key]
			if _, exists := row[path]; !exists {
				t.Errorf("%s official fixture row lacks default column %s path %s", command.ID, key, path)
			}
		}
	}
}

// assertStorageNoCommandLifecycle verifies that commands and APIs do not gain
// unsupported lifecycle metadata.
func assertStorageNoCommandLifecycle(t *testing.T, context storageReviewContext) {
	t.Helper()
	for _, command := range context.bundle.Commands.Commands {
		if command.Deprecation != nil || command.Recommendation != nil {
			t.Errorf("%s has unsupported lifecycle metadata", command.ID)
		}
		if operation := context.bundle.APIs.Operations[command.Operation]; operation.Deprecation != nil {
			t.Errorf("%s API has unsupported deprecation metadata", command.ID)
		}
	}
}
