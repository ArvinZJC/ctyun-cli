/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"fmt"
	"reflect"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// reviewTableLabels verifies that tables select the source response rows and
// columns and expose release-ready labels in every supported language.
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
		CheckOperationTable(report, operation, table)
	}
}

// CheckOperationTable compares a draft or promoted table with the source operation's
// response selectors, deprecation metadata and localized column labels. Column
// order, default selection and layout may be curated independently.
func CheckOperationTable(report *ReviewReport, operation Operation, table plugin.Table) {
	if table.RowPath != operation.Response.RowPath {
		addReviewFinding(report, fmt.Sprintf("operation %s JSON table row path %q does not match source path %q", operation.ID, table.RowPath, operation.Response.RowPath))
	}
	if !sameExecutionJSON(table.XML, operation.Response.XML) {
		addReviewFinding(report, fmt.Sprintf("operation %s XML table selectors do not match source evidence", operation.ID))
	}
	type selector struct{ key, path string }
	unmatched := make(map[selector][]plugin.TableColumn, len(table.Columns))
	for _, column := range table.Columns {
		id := selector{column.Key, column.Path}
		unmatched[id] = append(unmatched[id], column)
	}
	for _, sourceColumn := range operation.Response.Columns {
		id := selector{sourceColumn.Key, sourceColumn.Path}
		matches := unmatched[id]
		delete(unmatched, id)
		if len(matches) == 0 {
			addReviewFinding(report, fmt.Sprintf("operation %s response column %s has no generated table column", operation.ID, sourceColumn.Path))
			continue
		}
		if len(matches) > 1 {
			addReviewFinding(report, fmt.Sprintf("operation %s response column %s has duplicate generated table columns", operation.ID, sourceColumn.Path))
			continue
		}
		if !reflect.DeepEqual(matches[0].Deprecation, generatedColumnDeprecation(sourceColumn, operation.Response.Columns)) {
			addReviewFinding(report, fmt.Sprintf("operation %s response column %s deprecation does not match source evidence", operation.ID, sourceColumn.Path))
		}
		reviewSourceColumnLabels(report, operation.ID, sourceColumn)
		reviewGeneratedColumnLabels(report, operation.ID, sourceColumn, matches[0])
	}
	for _, column := range table.Columns {
		if _, extra := unmatched[selector{column.Key, column.Path}]; extra {
			addReviewFinding(report, fmt.Sprintf("operation %s table column %q with path %q has no source column", operation.ID, column.Key, column.Path))
		}
	}
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
