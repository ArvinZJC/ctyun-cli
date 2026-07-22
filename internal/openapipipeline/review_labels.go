/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"fmt"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// reviewTableLabels verifies that generated table columns preserve source
// evidence and expose release-ready labels in every supported language.
func reviewTableLabels(report *ReviewReport, source Catalog, commands plugin.Commands, tables plugin.Tables) {
	commandsByOperation := make(map[string]plugin.Command, len(commands.Commands))
	for _, command := range commands.Commands {
		commandsByOperation[command.Operation] = command
	}
	for _, operation := range source.Operations {
		command, ok := commandsByOperation[operation.ID]
		if !ok || command.Table == "" {
			continue
		}
		table, ok := tables.Tables[command.Table]
		if !ok {
			addReviewFinding(report, fmt.Sprintf("operation %s table %s is missing", operation.ID, command.Table))
			continue
		}
		for _, sourceColumn := range operation.Response.Columns {
			matches := matchingTableColumns(table.Columns, sourceColumn)
			if len(matches) == 0 {
				addReviewFinding(report, fmt.Sprintf("operation %s response column %s has no generated table column", operation.ID, sourceColumn.Path))
				continue
			}
			if len(matches) > 1 {
				addReviewFinding(report, fmt.Sprintf("operation %s response column %s has duplicate generated table columns", operation.ID, sourceColumn.Path))
				continue
			}
			reviewSourceColumnLabels(report, operation.ID, sourceColumn)
			reviewGeneratedColumnLabels(report, operation.ID, sourceColumn, matches[0])
		}
	}
}

// matchingTableColumns returns generated columns with the same stable key and
// response path as one normalized source column.
func matchingTableColumns(columns []plugin.TableColumn, source Column) []plugin.TableColumn {
	var matches []plugin.TableColumn
	for _, column := range columns {
		if column.Key == source.Key && column.Path == source.Path {
			matches = append(matches, column)
		}
	}
	return matches
}

// reviewSourceColumnLabels blocks invalid normalized source evidence before a
// generator transformation can hide its defect.
func reviewSourceColumnLabels(report *ReviewReport, operationID string, column Column) {
	for _, label := range []struct {
		language string
		value    string
	}{
		{language: "en-US", value: column.LabelEN},
		{language: "zh-CN", value: column.LabelZH},
	} {
		if label.value == "" {
			continue
		}
		if finding := DisplayLabelQualityFinding(label.language, label.value); finding != "" {
			addReviewFinding(report, fmt.Sprintf("operation %s response column %s source label %s %q %s", operationID, column.Path, label.language, label.value, finding))
		}
	}
}

// reviewGeneratedColumnLabels verifies draft quality and source agreement for
// every supported table-label language.
func reviewGeneratedColumnLabels(report *ReviewReport, operationID string, source Column, generated plugin.TableColumn) {
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		label := generated.Labels[language]
		if finding := DisplayLabelQualityFinding(language, label); finding != "" {
			addReviewFinding(report, fmt.Sprintf("operation %s response column %s label %s %q %s", operationID, source.Path, language, label, finding))
		}
		sourceLabel := source.LabelEN
		if language == "zh-CN" {
			sourceLabel = source.LabelZH
		}
		if finding := DisplayLabelAgreementFinding(language, sourceLabel, label); finding != "" {
			addReviewFinding(report, fmt.Sprintf("operation %s response column %s label %s %q %s", operationID, source.Path, language, label, finding))
		}
	}
}
