/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// GenerateDraft writes candidate plugin metadata under
// openapi/products/<product>/draft without modifying promoted plugins.
func (workspace Workspace) GenerateDraft(product string) error {
	catalog, err := workspace.ReadSource(product)
	if err != nil {
		return err
	}
	draftDir := workspace.ProductPath(product, "draft")
	files := map[string]any{
		"plugin.json":   buildManifest(catalog),
		"apis.json":     buildAPIs(catalog),
		"commands.json": buildCommands(catalog),
		"tables.json":   buildTables(catalog),
		"waiters.json":  plugin.Waiters{Waiters: map[string]plugin.Waiter{}},
	}
	for name, value := range files {
		if err := writeJSON(filepath.Join(draftDir, name), value); err != nil {
			return err
		}
	}
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		if err := writeJSON(filepath.Join(draftDir, "i18n", language+".json"), map[string]string{}); err != nil {
			return err
		}
	}
	return writeFixtures(draftDir, catalog)
}

// buildManifest converts catalog product metadata into plugin.json.
func buildManifest(catalog Catalog) plugin.Manifest {
	return plugin.Manifest{
		Name:    catalog.Product.PluginName,
		Version: "0.1.0-alpha.1",
		Channel: "alpha",
		Quality: "generated",
		Requires: plugin.Requirements{
			Ctyun: ">=0.1.0-alpha.1 <1.0.0",
		},
		API: plugin.APIInfo{
			Product:           catalog.Product.APIProduct,
			CtyunProductID:    catalog.Product.CtyunProductID,
			SourceRevision:    catalog.Product.SourceRevision,
			SourceFingerprint: catalogFingerprint(catalog),
			EndpointURL:       catalog.Product.EndpointURL,
		},
	}
}

// buildAPIs converts catalog operations into apis.json.
func buildAPIs(catalog Catalog) plugin.APIs {
	operations := make(map[string]plugin.Operation, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		next := plugin.Operation{
			Method:      operation.Method,
			Path:        operation.Path,
			ContentType: operation.ContentType,
			Retryable:   operation.Retryable,
			Query:       map[string]string{},
			Headers:     map[string]string{},
			Body:        map[string]string{},
		}
		for _, parameter := range operation.Parameters {
			binding := parameterBinding(parameter)
			if binding == "" {
				continue
			}
			switch parameter.Location {
			case "query":
				next.Query[parameter.Name] = binding
			case "header":
				next.Headers[parameter.Name] = binding
			case "body":
				next.Body[parameter.Name] = binding
			}
		}
		operations[operation.ID] = next
	}
	return plugin.APIs{Operations: operations}
}

// buildCommands converts catalog operations into commands.json.
func buildCommands(catalog Catalog) plugin.Commands {
	commands := make([]plugin.Command, 0, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		action := commandAction(operation)
		command := plugin.Command{
			ID:              commandID(operation),
			Path:            commandPath(catalog, operation),
			Operation:       operation.ID,
			Table:           tableID(catalog, operation),
			FixtureResponse: fixturePath(operation),
			DocsURL:         operation.DocsURL,
			Examples:        []string{"ctyun " + strings.Join(commandPath(catalog, operation), " ")},
			Dangerous:       plugin.Dangerous{},
		}
		if operation.Dangerous {
			command.Dangerous = plugin.Dangerous{Confirm: "yes", Message: action}
		}
		for _, parameter := range operation.Parameters {
			if parameter.CLIName == "" {
				continue
			}
			command.Parameters = append(command.Parameters, plugin.Parameter{
				Name:          parameter.CLIName,
				Flag:          parameter.CLIName,
				Target:        parameter.TableTarget,
				Required:      parameter.Required,
				AllowedValues: parameter.Enum,
				Pattern:       parameter.Pattern,
				Description:   parameter.Description,
			})
		}
		commands = append(commands, command)
	}
	return plugin.Commands{Commands: commands}
}

// buildTables converts catalog response columns into tables.json.
func buildTables(catalog Catalog) plugin.Tables {
	tables := make(map[string]plugin.Table, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		columns := make([]plugin.TableColumn, 0, len(operation.Response.Columns))
		for _, column := range operation.Response.Columns {
			columns = append(columns, plugin.TableColumn{
				Key:  column.Key,
				Path: column.Path,
				Labels: map[string]string{
					"en-US": column.LabelEN,
					"en-GB": column.LabelEN,
					"zh-CN": column.LabelZH,
				},
			})
		}
		tables[tableID(catalog, operation)] = plugin.Table{
			RowPath: operation.Response.RowPath,
			Columns: columns,
		}
	}
	return plugin.Tables{Tables: tables}
}

// commandPath derives the canonical plugin command path for an operation.
func commandPath(catalog Catalog, operation Operation) []string {
	path := []string{catalog.Product.PluginName}
	if operation.Category != "" {
		path = append(path, operation.Category)
	}
	path = append(path, commandAction(operation))
	if argument := firstArgument(operation); argument != "" && commandAction(operation) != "list" {
		path = append(path, "{"+argument+"}")
	}
	return path
}

// commandAction derives the leaf command action from operation evidence.
func commandAction(operation Operation) string {
	lowerID := strings.ToLower(operation.ID)
	lowerPath := strings.ToLower(operation.Path)
	if strings.Contains(lowerID, ".list") || strings.Contains(lowerPath, "list-") || strings.Contains(lowerPath, "-list") {
		return "list"
	}
	if strings.Contains(lowerID, ".show") || strings.Contains(lowerPath, "detail") || strings.Contains(lowerPath, "details") {
		return "show"
	}
	parts := strings.Split(operation.ID, ".")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}
	return "call"
}

// parameterBinding returns the metadata binding expression for a parameter.
func parameterBinding(parameter Parameter) string {
	switch {
	case parameter.Profile != "":
		return "$profile." + parameter.Profile
	case parameter.Argument != "":
		return "$arg." + parameter.Argument
	case parameter.CLIName != "":
		return "$param." + parameter.CLIName
	default:
		return ""
	}
}

// tableID returns the generated table identifier for an operation.
func tableID(catalog Catalog, operation Operation) string {
	parts := []string{catalog.Product.PluginName}
	if operation.Category != "" {
		parts = append(parts, operation.Category)
	}
	parts = append(parts, commandAction(operation))
	return strings.Join(parts, ".")
}

// commandID returns the generated command identifier for an operation.
func commandID(operation Operation) string {
	return strings.TrimPrefix(operation.ID, "v4.")
}

// firstArgument returns the first path argument bound by an operation.
func firstArgument(operation Operation) string {
	for _, parameter := range operation.Parameters {
		if parameter.Argument != "" {
			return parameter.Argument
		}
	}
	return ""
}

// fixturePath returns the generated fixture path for an operation.
func fixturePath(operation Operation) string {
	return "fixtures/" + strings.ReplaceAll(operation.ID, ".", "-") + ".json"
}

// writeFixtures writes one generated fixture file per catalog operation.
func writeFixtures(draftDir string, catalog Catalog) error {
	for _, operation := range catalog.Operations {
		raw := compactRawMessage(operation.ExampleResponse)
		if len(raw) == 0 {
			raw = json.RawMessage(`{}`)
		}
		if err := writeRawJSON(filepath.Join(draftDir, fixturePath(operation)), raw); err != nil {
			return err
		}
	}
	return nil
}

// writeRawJSON writes an already validated JSON document with indentation.
func writeRawJSON(path string, raw json.RawMessage) error {
	var buffer bytes.Buffer
	if err := json.Indent(&buffer, raw, "", "  "); err != nil {
		return err
	}
	data := buffer.Bytes()
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
