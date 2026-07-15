/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// GenerateDraft writes candidate plugin metadata under
// openapi-catalogs/<product>/draft without modifying promoted plugins.
func (workspace Workspace) GenerateDraft(product string) error {
	catalog, err := workspace.ReadSource(product)
	if err != nil {
		return err
	}
	draftDir := workspace.ProductPath(product, "draft")
	if err := prepareDraftDir(draftDir); err != nil {
		return err
	}
	return writeDraft(draftDir, catalog)
}

// writeDraft writes generated plugin metadata, i18n catalogs, and fixtures.
func writeDraft(draftDir string, catalog Catalog) error {
	files := map[string]any{
		"plugin.json":   buildManifest(catalog),
		"apis.json":     buildAPIs(catalog),
		"commands.json": buildCommands(catalog),
		"tables.json":   buildTables(catalog),
		"waiters.json":  buildWaiters(catalog),
	}
	for name, value := range files {
		if err := writeJSON(filepath.Join(draftDir, name), value); err != nil {
			return err
		}
	}
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		if err := writeJSON(filepath.Join(draftDir, "i18n", language+".json"), buildI18N(catalog, language)); err != nil {
			return err
		}
	}
	return writeFixtures(draftDir, catalog)
}

// prepareDraftDir removes stale generated draft contents before regeneration.
func prepareDraftDir(draftDir string) error {
	info, err := os.Stat(draftDir)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("draft path %s is not a directory", draftDir)
		}
		return os.RemoveAll(draftDir)
	case os.IsNotExist(err):
		return nil
	default:
		return err
	}
}

// buildManifest converts catalog product metadata into plugin.json.
func buildManifest(catalog Catalog) plugin.Manifest {
	return plugin.Manifest{
		Name:    catalog.Product.PluginName,
		Version: "0.1.0-alpha.1",
		Channel: "alpha",
		Quality: "generated",
		Requires: plugin.Requirements{
			Ctyun: ">=0.3.1 <1.0.0",
		},
		API: plugin.APIInfo{
			Product:           catalog.Product.APIProduct,
			CtyunProductID:    catalog.Product.CtyunProductID,
			SourceRevision:    catalog.Product.SourceRevision,
			SourceFingerprint: catalogFingerprint(catalog),
			Scope:             catalog.Product.APIScope,
			EndpointURL:       catalog.Product.EndpointURL,
		},
	}
}

// buildAPIs converts catalog operations into apis.json.
func buildAPIs(catalog Catalog) plugin.APIs {
	operations := make(map[string]plugin.Operation, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		next := plugin.Operation{
			Method:           operation.Method,
			Path:             operation.Path,
			ContentType:      operation.ContentType,
			Retryable:        operation.Retryable,
			AcceptedStatuses: operation.Response.AcceptedStatuses,
			Deprecation:      deprecationFromOperation(operation),
			Query:            map[string]string{},
			Headers:          map[string]string{},
			Body:             map[string]string{},
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
			ID:                      commandID(operation),
			Path:                    commandPath(catalog, operation),
			Operation:               operation.ID,
			Table:                   tableID(catalog, operation),
			ConditionalRequirements: operation.ConditionalRequirements,
			FixtureResponse:         fixturePath(operation),
			DocsURL:                 operation.DocsURL,
			Dangerous:               plugin.Dangerous{},
			Recommendation:          generatedRecommendation(operation),
		}
		if operation.Dangerous {
			command.Dangerous = plugin.Dangerous{Confirm: "yes", Message: action}
		}
		for _, parameter := range operation.Parameters {
			cliName, flag, target, required := commandParameterMetadata(parameter)
			if cliName == "" {
				continue
			}
			command.Parameters = append(command.Parameters, plugin.Parameter{
				Name:          cliName,
				Flag:          flag,
				Target:        target,
				Required:      required,
				ValueType:     generatedParameterValueType(parameter),
				AllowedValues: parameter.Enum,
				Pattern:       parameter.Pattern,
				Description:   parameterEnglishDescription(parameter),
				Deprecation:   deprecationFromParameter(parameter),
			})
		}
		command.Examples = generatedCommandExamples(operation, command)
		commands = append(commands, command)
	}
	return plugin.Commands{Commands: commands}
}

// generatedRecommendation copies reviewed visible-command guidance without
// exposing source-only API recommendation evidence.
func generatedRecommendation(operation Operation) *plugin.Recommendation {
	if operation.Recommendation == nil || operation.Recommendation.TargetCommand == nil {
		return nil
	}
	if operationHasDeprecationText(operation) {
		return nil
	}
	return &plugin.Recommendation{TargetCommand: *operation.Recommendation.TargetCommand}
}

// buildTables converts catalog response columns into tables.json.
func buildTables(catalog Catalog) plugin.Tables {
	tables := make(map[string]plugin.Table, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		columns := make([]plugin.TableColumn, 0, len(operation.Response.Columns))
		for _, column := range operation.Response.Columns {
			labelEN := englishColumnLabel(column)
			columns = append(columns, plugin.TableColumn{
				Key:         column.Key,
				Path:        column.Path,
				Deprecation: deprecationFromColumn(column),
				Labels: map[string]string{
					"en-US": labelEN,
					"en-GB": labelEN,
					"zh-CN": chineseColumnLabel(column, labelEN),
				},
			})
		}
		tables[tableID(catalog, operation)] = plugin.Table{
			RowPath:        operation.Response.RowPath,
			Layout:         operation.Response.Layout,
			DefaultColumns: operation.Response.DefaultColumns,
			Columns:        columns,
		}
	}
	return plugin.Tables{Tables: tables}
}

// buildWaiters derives conservative waiters from reviewed response evidence.
func buildWaiters(catalog Catalog) plugin.Waiters {
	waiters := map[string]plugin.Waiter{}
	for _, operation := range catalog.Operations {
		if commandID(operation) != catalog.Product.PluginName+".instance.show" {
			continue
		}
		if operation.Response.RowPath != "returnObj" || !hasResponseColumnPath(operation.Response, "instanceStatus") {
			continue
		}
		waiters[catalog.Product.PluginName+".instance.running"] = plugin.Waiter{
			Path:            "returnObj.instanceStatus",
			Success:         "running",
			Failure:         "error",
			MaxAttempts:     20,
			IntervalSeconds: 3,
		}
		waiters[catalog.Product.PluginName+".instance.stopped"] = plugin.Waiter{
			Path:            "returnObj.instanceStatus",
			Success:         "stopped",
			Failure:         "error",
			MaxAttempts:     20,
			IntervalSeconds: 3,
		}
	}
	return plugin.Waiters{Waiters: waiters}
}

// buildI18N converts catalog display names into plugin i18n entries.
func buildI18N(catalog Catalog, language string) map[string]string {
	entries := map[string]string{}
	if name := catalog.Product.DisplayName[language]; name != "" {
		entries["name"] = name
	}
	for _, operation := range catalog.Operations {
		id := commandID(operation)
		if description := operation.Description[language]; description != "" {
			entries["command."+id+".description"] = description
		}
		for _, parameter := range operation.Parameters {
			if parameter.Argument != "" {
				description := parameterLocalizedDescription(parameter, language)
				if description != "" {
					entries["argument."+id+"."+parameter.Argument+".description"] = description
				}
			}
			cliName, _, _, _ := commandParameterMetadata(parameter)
			if cliName == "" {
				continue
			}
			description := parameterLocalizedDescription(parameter, language)
			if description != "" {
				entries["parameter."+id+"."+cliName+".description"] = description
			}
		}
	}
	return entries
}

// commandParameterMetadata returns the command option metadata generated for a
// catalog parameter.
func commandParameterMetadata(parameter Parameter) (name, flag, target string, required bool) {
	if parameter.CLIName != "" {
		flag := parameter.CLIFlag
		if flag == "" {
			flag = parameter.CLIName
		}
		target := parameter.TableTarget
		if target == "" {
			target = parameter.Name
		}
		return parameter.CLIName, flag, target, parameter.Required
	}
	if parameter.Profile == "region" {
		return "region", "region", parameter.Name, false
	}
	return "", "", "", false
}

// hasResponseColumnPath reports whether response exposes a table column path.
func hasResponseColumnPath(response Response, path string) bool {
	for _, column := range response.Columns {
		if column.Path == path {
			return true
		}
	}
	return false
}

// parameterLocalizedDescription returns safe localized help text for a CLI
// parameter without leaking Chinese-only upstream prose into English catalogs.
func parameterLocalizedDescription(parameter Parameter, language string) string {
	if description := strings.TrimSpace(parameter.Descriptions[language]); description != "" {
		if language == "zh-CN" && generatedChineseParameterDescription(description) {
			return chineseNameForIdentifier(parameterIdentifier(parameter))
		}
		if language == "zh-CN" {
			return normalizeChineseTechnicalLabel(description)
		}
		return description
	}
	if language == "zh-CN" {
		if description := strings.TrimSpace(parameter.Description); description != "" {
			if generatedChineseParameterDescription(description) {
				return chineseNameForIdentifier(parameterIdentifier(parameter))
			}
			return normalizeChineseTechnicalLabel(description)
		}
		return chineseNameForIdentifier(parameterIdentifier(parameter))
	}
	return parameterEnglishDescription(parameter)
}

// generatedChineseParameterDescription reports upstream prose that should be
// replaced with a concise generated CLI label in Chinese help.
func generatedChineseParameterDescription(description string) bool {
	if strings.Contains(description, "您可以查看") ||
		(strings.Contains(description, "获取：") && (strings.Contains(description, " 查 ") || strings.Contains(description, " 创 "))) {
		return true
	}
	//goland:noinspection HttpUrlsUsage
	if strings.Contains(description, "http://") || strings.Contains(description, "https://") {
		return true
	}
	if containsNonTechnicalASCIIWord(description) {
		return true
	}
	if strings.ContainsAny(description, "，。；;：:") {
		return true
	}
	return len([]rune(description)) > 24
}

// commandPath derives the canonical plugin command path for an operation.
func commandPath(catalog Catalog, operation Operation) []string {
	path := []string{catalog.Product.PluginName}
	if len(operation.CommandPath) > 0 {
		return append(path, operation.CommandPath...)
	}
	if operation.Category != "" {
		path = append(path, operation.Category)
	}
	path = append(path, commandAction(operation))
	for _, argument := range arguments(operation) {
		path = append(path, "{"+argument+"}")
	}
	return path
}

// commandAction derives the leaf command action from operation evidence.
func commandAction(operation Operation) string {
	parts := strings.Split(operation.ID, ".")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}
	lowerPath := strings.ToLower(operation.Path)
	if strings.Contains(lowerPath, "list-") || strings.Contains(lowerPath, "-list") {
		return "list"
	}
	if strings.Contains(lowerPath, "detail") || strings.Contains(lowerPath, "details") {
		return "show"
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
	if len(operation.CommandPath) > 0 {
		for _, segment := range operation.CommandPath {
			if !isCommandPathArgument(segment) {
				parts = append(parts, segment)
			}
		}
		return strings.Join(parts, ".")
	}
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

// arguments returns path arguments bound by an operation in source order.
func arguments(operation Operation) []string {
	var values []string
	for _, parameter := range operation.Parameters {
		if parameter.Argument != "" {
			values = append(values, parameter.Argument)
		}
	}
	return values
}

// concreteExamples replaces argument placeholders with values captured from
// upstream example responses when the evidence is available.
func concreteExamples(operation Operation) []string {
	examples := append([]string(nil), operation.Examples...)
	values := exampleArgumentValues(operation)
	if len(values) == 0 {
		return examples
	}
	for index, example := range examples {
		for argument, value := range values {
			example = strings.ReplaceAll(example, "{"+argument+"}", value)
		}
		examples[index] = example
	}
	return examples
}

// exampleArgumentValues maps command arguments to concrete upstream example
// values by matching the bound API parameter name inside example_response.
func exampleArgumentValues(operation Operation) map[string]string {
	if len(operation.ExampleResponse) == 0 {
		return nil
	}
	var payload any
	if err := json.Unmarshal(operation.ExampleResponse, &payload); err != nil {
		return nil
	}
	values := make(map[string]string)
	for _, parameter := range operation.Parameters {
		if parameter.Argument == "" {
			continue
		}
		if value, ok := exampleValueForKey(payload, parameter.Name); ok {
			values[parameter.Argument] = value
		}
	}
	return values
}

// exampleValueForKey returns the first scalar value for key in a JSON tree.
func exampleValueForKey(value any, key string) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		if candidate, ok := typed[key]; ok {
			if scalar, ok := scalarExampleValue(candidate); ok {
				return scalar, true
			}
		}
		for _, candidate := range typed {
			if scalar, ok := exampleValueForKey(candidate, key); ok {
				return scalar, true
			}
		}
	case []any:
		for _, candidate := range typed {
			if scalar, ok := exampleValueForKey(candidate, key); ok {
				return scalar, true
			}
		}
	}
	return "", false
}

// scalarExampleValue formats scalar JSON example values for CLI examples.
func scalarExampleValue(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, typed != ""
	case float64, bool:
		return fmt.Sprint(typed), true
	default:
		return "", false
	}
}

// fixturePath returns the generated fixture path for an operation.
func fixturePath(operation Operation) string {
	return "fixtures/" + strings.ReplaceAll(commandID(operation), ".", "-") + ".json"
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
	data := collapseIndentedScalarArrays(buffer.String())
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// collapseIndentedScalarArrays keeps short fixture scalar lists on one line.
func collapseIndentedScalarArrays(value string) []byte {
	lines := strings.Split(value, "\n")
	collapsed := make([]string, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		if !strings.HasSuffix(trimmed, "[") {
			collapsed = append(collapsed, line)
			continue
		}
		items, endIndex, ok := scalarArrayItems(lines, index+1)
		if !ok {
			collapsed = append(collapsed, line)
			continue
		}
		endTrimmed := strings.TrimSpace(lines[endIndex])
		suffix := ""
		if strings.HasSuffix(endTrimmed, ",") {
			suffix = ","
		}
		prefix := strings.TrimSuffix(line, "[")
		collapsed = append(collapsed, prefix+"["+strings.Join(items, ", ")+"]"+suffix)
		index = endIndex
	}
	return []byte(strings.Join(collapsed, "\n"))
}

// scalarArrayItems returns compact item text for an indented scalar array.
func scalarArrayItems(lines []string, start int) ([]string, int, bool) {
	var items []string
	for index := start; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "]" || trimmed == "]," {
			return items, index, true
		}
		item := strings.TrimSuffix(trimmed, ",")
		if item == "" || strings.HasPrefix(item, "{") || strings.HasPrefix(item, "[") {
			return nil, 0, false
		}
		items = append(items, item)
	}
	return nil, 0, false
}
