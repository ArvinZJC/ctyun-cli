/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"strconv"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
)

// selectReportRows applies visible table controls and projects the matching
// structured records using each row's internal source index.
func selectReportRows[T any](rows []map[string]string, records []T, columns []output.Column, opts globalOptions) ([]map[string]string, []T, error) {
	filter, err := output.ResolveFilterExpression(columns, opts.Filter)
	if err != nil {
		return nil, nil, err
	}
	sortExpression, err := output.ResolveSortExpression(columns, opts.Sort)
	if err != nil {
		return nil, nil, err
	}
	rows, _ = output.FilterRows(rows, filter)
	rows, _ = output.SortRows(rows, sortExpression)
	selected := make([]T, 0, len(rows))
	for _, row := range rows {
		index, _ := strconv.Atoi(row["_index"])
		selected = append(selected, records[index])
	}
	return rows, selected, nil
}

// renderReportOutput writes structured JSON or a shared table with an optional
// trailing summary line.
func renderReportOutput(stdout io.Writer, rows []map[string]string, columns []output.Column, jsonReport any, tableFooter string, opts globalOptions) error {
	switch opts.Output {
	case "json":
		rendered, err := renderOutputJSON(jsonReport)
		if err != nil {
			return err
		}
		return writeString(stdout, rendered)
	case "table":
		rendered, err := renderTableOutput(stdout, rows, columns, output.TableOptions{Columns: opts.Columns, NoHeader: opts.NoHeader, Style: opts.Table})
		if err != nil {
			return err
		}
		if err := writeString(stdout, rendered); err != nil {
			return err
		}
		if tableFooter == "" {
			return nil
		}
		return writeLine(stdout, tableFooter)
	default:
		return diagnostic.New("error.unsupported_output", opts.Output)
	}
}
