/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package cli

import (
	"encoding/json"
	"fmt"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"slices"
	"strings"
)

// rowsFromPayload converts decoded JSON into stable-key table rows.
func rowsFromPayload(payload map[string]any, table plugin.Table) ([]map[string]string, error) {
	if table.XML != nil && (payload["status"] == nil || payload["name"] != nil) {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		var root client.XMLNode
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, err
		}
		rows, err := client.ProjectXML(&root, *table.XML)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			for _, column := range table.Columns {
				if column.Path == "status" {
					row[column.Key] = formatTableCell(payload["status"])
				}
			}
		}
		return rows, nil
	}
	rowPath := table.RowPath
	if table.XML != nil {
		rowPath = "$"
	}
	rawRows, err := valueAtPath(payload, rowPath)
	if err != nil {
		return nil, err
	}
	rowValues, ok := rawRows.([]any)
	if !ok {
		if rowMap, ok := rawRows.(map[string]any); ok {
			rowValues = []any{rowMap}
		} else {
			return nil, diagnostic.New("error.row_path_not_array", table.RowPath)
		}
	}

	rows := make([]map[string]string, 0, len(rowValues))
	for _, rawRow := range rowValues {
		rowMap, ok := rawRow.(map[string]any)
		if !ok {
			return nil, diagnostic.New("error.row_path_non_object", table.RowPath)
		}
		row := make(map[string]string, len(table.Columns))
		for _, column := range table.Columns {
			// Missing optional paths render as empty cells; malformed row paths
			// were already rejected above.
			value, err := valueAtPath(rowMap, column.Path)
			if err == nil {
				row[column.Key] = formatTableCell(value)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// formatTableCell converts decoded JSON values into readable table cells.
func formatTableCell(value any) string {
	if value == nil {
		return ""
	}
	return formatTableCellValue(value, false)
}

// formatTableCellValue formats nested JSON values with stable object ordering.
func formatTableCellValue(value any, nested bool) string {
	switch typed := value.(type) {
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, formatTableCellValue(item, true))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		parts := sortedMapCellParts(typed)
		if len(parts) == 0 {
			return "{}"
		}
		if nested {
			return "{" + strings.Join(parts, "; ") + "}"
		}
		return strings.Join(parts, "; ")
	}
	return fmt.Sprint(value)
}

// sortedMapCellParts returns stable key=value fragments for a JSON object cell.
func sortedMapCellParts(value map[string]any) []string {
	if len(value) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+formatTableCellValue(value[key], true))
	}
	return parts
}

// tableColumns localizes table column labels for rendering.
func tableColumns(table plugin.Table, language string) []output.Column {
	columns := make([]output.Column, 0, len(table.Columns))
	for _, column := range table.Columns {
		catalog := i18n.Catalog{column.Key: column.Labels}
		columns = append(columns, output.Column{
			Key:   column.Key,
			Label: catalog.Text(column.Key, language),
		})
	}
	return columns
}

// valueAtPath walks a dot-separated path through decoded JSON objects, and
// projects object paths through arrays so table columns can target leaf values.
func valueAtPath(value any, path string) (any, error) {
	if path == "$" {
		return value, nil
	}
	return valueAtPathParts(value, strings.Split(path, "."), path)
}

// valueAtPathParts recursively reads path parts, flattening projected arrays.
func valueAtPathParts(value any, parts []string, fullPath string) (any, error) {
	if len(parts) == 0 {
		return value, nil
	}
	switch typed := value.(type) {
	case map[string]any:
		next, ok := typed[parts[0]]
		if !ok {
			return nil, diagnostic.New("error.path_missing", fullPath, parts[0])
		}
		return valueAtPathParts(next, parts[1:], fullPath)
	case []any:
		projected := make([]any, 0, len(typed))
		for _, item := range typed {
			next, err := valueAtPathParts(item, parts, fullPath)
			if err != nil {
				return nil, err
			}
			projected = appendProjectedValue(projected, next)
		}
		return projected, nil
	default:
		return nil, diagnostic.New("error.path_cannot_read", fullPath, parts[0])
	}
}

// appendProjectedValue appends value, flattening arrays from nested projection.
func appendProjectedValue(values []any, value any) []any {
	if nested, ok := value.([]any); ok {
		return append(values, nested...)
	}
	return append(values, value)
}
