/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import "fmt"

// compareCatalogMetadata makes product routing, scope, names and schema changes
// visible independently of operation inventories and upstream revision numbers.
func compareCatalogMetadata(report *DiffReport, old, next Catalog) {
	if old.SchemaVersion != next.SchemaVersion {
		report.Changes = append(report.Changes, "Catalog schema version changed.")
	}
	before, after := old.Product, next.Product
	before.SourceRevision, after.SourceRevision = "", ""
	if !sameExecutionJSON(before, after) {
		report.Changes = append(report.Changes, "Product identity, endpoint, display names, source URL, or API scope changed.")
	}
}

// compareOperationMetadata covers execution and generated presentation fields
// omitted by the more detailed method, path and parameter comparisons.
func compareOperationMetadata(report *DiffReport, old, next Operation) {
	if !sameExecutionJSON(old.Request, next.Request) || !sameExecutionJSON(old.Response.HTTP, next.Response.HTTP) || !sameExecutionJSON(old.Fixture, next.Fixture) || old.Download != next.Download {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` HTTP transport or fixture contract changed.", old.ID))
	}
	if old.Retryable != next.Retryable || old.Dangerous != next.Dangerous || old.ContentType != next.ContentType {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` retry, confirmation, or content-type contract changed.", old.ID))
	}
	if !sameExecutionJSON(old.ConditionalRequirements, next.ConditionalRequirements) {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` conditional requirements changed.", old.ID))
	}
	if old.Category != next.Category || !sameExecutionJSON(old.CommandPath, next.CommandPath) {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` command path changed.", old.ID))
	}
	if !sameExecutionJSON(old.Response.AcceptedStatuses, next.Response.AcceptedStatuses) || old.Response.SuccessCode != next.Response.SuccessCode || old.Response.ResultPath != next.Response.ResultPath {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` response status or result contract changed.", old.ID))
	}
	if !sameExecutionJSON(old.Response.XML, next.Response.XML) || old.Response.Layout != next.Response.Layout || !sameExecutionJSON(old.Response.DefaultColumns, next.Response.DefaultColumns) || !sameExecutionJSON(old.Response.Columns, next.Response.Columns) {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` table columns, default columns, or layout changed.", old.ID))
	}
	if old.APIID != next.APIID || old.Title != next.Title || old.DocsURL != next.DocsURL || !sameExecutionJSON(old.Description, next.Description) || !sameExecutionJSON(old.Examples, next.Examples) || !sameExecutionJSON(old.RequestExample, next.RequestExample) || !sameExecutionJSON(old.ExampleResponse, next.ExampleResponse) {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` documentation or captured examples changed.", old.ID))
	}
	if old.Recommendation != nil && next.Recommendation != nil && (!sameExecutionJSON(old.Recommendation.TargetCommand, next.Recommendation.TargetCommand) || !sameExecutionJSON(old.Recommendation.Applicability, next.Recommendation.Applicability)) {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` visible recommendation guidance changed.", old.ID))
	}
}

// compareParameterMetadata reports constraints, defaults, bindings and source
// evidence changes while leaving required/type changes to their detailed rows.
func compareParameterMetadata(report *DiffReport, operationID, key string, old, next Parameter) {
	old.Required, next.Required = false, false
	old.Type, next.Type = "", ""
	if !sameExecutionJSON(old, next) {
		report.Changes = append(report.Changes, fmt.Sprintf("Operation `%s` parameter `%s` constraints, defaults, bindings, or evidence changed.", operationID, key))
	}
}
