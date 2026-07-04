/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapi

import (
	"fmt"
	"sort"
	"strings"
)

// DiffReport describes upstream changes between an accepted baseline and the
// latest source catalog.
type DiffReport struct {
	Product string
	Changes []string
}

// DiffCatalogs compares baseline and source catalogs.
func DiffCatalogs(baseline, source Catalog) DiffReport {
	report := DiffReport{Product: source.Product.PluginName}
	if baseline.Product.SourceRevision != source.Product.SourceRevision {
		report.Changes = append(report.Changes, fmt.Sprintf("Source revision changed from `%s` to `%s`.", baseline.Product.SourceRevision, source.Product.SourceRevision))
	}
	oldOps := operationsByID(baseline.Operations)
	newOps := operationsByID(source.Operations)
	for _, id := range sortedKeys(oldOps) {
		oldOperation := oldOps[id]
		newOperation, ok := newOps[id]
		if !ok {
			report.Changes = append(report.Changes, fmt.Sprintf("Removed operation `%s`.", id))
			continue
		}
		compareOperation(&report, oldOperation, newOperation)
	}
	for _, id := range sortedKeys(newOps) {
		if _, ok := oldOps[id]; !ok {
			report.Changes = append(report.Changes, fmt.Sprintf("Added operation `%s`.", id))
		}
	}
	return report
}

// WriteDiff compares workspace source and baseline then writes changes.md.
func (workspace Workspace) WriteDiff(product string) (DiffReport, error) {
	source, err := workspace.ReadSource(product)
	if err != nil {
		return DiffReport{}, err
	}
	baseline, err := workspace.ReadBaseline(product)
	if err != nil {
		report := DiffReport{Product: product, Changes: []string{"No baseline exists. The next reviewed promotion will establish the first baseline."}}
		return report, writeText(workspace.ProductPath(product, "changes.md"), report.Markdown())
	}
	report := DiffCatalogs(baseline, source)
	return report, writeText(workspace.ProductPath(product, "changes.md"), report.Markdown())
}

// Markdown renders a stable drift report.
func (report DiffReport) Markdown() string {
	var builder strings.Builder
	builder.WriteString("# OpenAPI Drift Report: " + report.Product + "\n\n")
	if len(report.Changes) == 0 {
		builder.WriteString("No drift detected.\n")
		return builder.String()
	}
	for _, change := range report.Changes {
		builder.WriteString("- " + change + "\n")
	}
	return builder.String()
}

// compareOperation records operation-level drift in report.
func compareOperation(report *DiffReport, oldOperation, newOperation Operation) {
	if oldOperation.Method != newOperation.Method {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` method changed from `%s` to `%s`.", oldOperation.ID, oldOperation.Method, newOperation.Method))
	}
	if oldOperation.Path != newOperation.Path {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` path changed from `%s` to `%s`.", oldOperation.ID, oldOperation.Path, newOperation.Path))
	}
	oldParams := parametersByKey(oldOperation.Parameters)
	newParams := parametersByKey(newOperation.Parameters)
	for _, key := range sortedKeys(oldParams) {
		if _, ok := newParams[key]; !ok {
			report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` removed parameter `%s`.", oldOperation.ID, key))
		}
	}
	for _, key := range sortedKeys(newParams) {
		oldParameter, ok := oldParams[key]
		if !ok {
			report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` added parameter `%s`.", oldOperation.ID, key))
			continue
		}
		newParameter := newParams[key]
		if oldParameter.Required != newParameter.Required {
			report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` parameter `%s` required changed from `%t` to `%t`.", oldOperation.ID, key, oldParameter.Required, newParameter.Required))
		}
		if oldParameter.Type != newParameter.Type {
			report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` parameter `%s` type changed from `%s` to `%s`.", oldOperation.ID, key, oldParameter.Type, newParameter.Type))
		}
	}
	if oldOperation.Response.RowPath != newOperation.Response.RowPath {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` row path changed from `%s` to `%s`.", oldOperation.ID, oldOperation.Response.RowPath, newOperation.Response.RowPath))
	}
	if oldOperation.Response.JobIDPath != newOperation.Response.JobIDPath {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` job ID path changed from `%s` to `%s`.", oldOperation.ID, oldOperation.Response.JobIDPath, newOperation.Response.JobIDPath))
	}
}

// operationsByID indexes operations by catalog operation ID.
func operationsByID(operations []Operation) map[string]Operation {
	result := make(map[string]Operation, len(operations))
	for _, operation := range operations {
		result[operation.ID] = operation
	}
	return result
}

// parametersByKey indexes parameters by request location and upstream name.
func parametersByKey(parameters []Parameter) map[string]Parameter {
	result := make(map[string]Parameter, len(parameters))
	for _, parameter := range parameters {
		result[parameter.Location+"."+parameter.Name] = parameter
	}
	return result
}

// sortedKeys returns map keys in stable lexical order.
func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
