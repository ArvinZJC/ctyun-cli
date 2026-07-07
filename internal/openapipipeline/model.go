/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package openapipipeline supports the repository-local CTyun OpenAPI catalog
// pipeline used to maintain plugin metadata.
package openapipipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// Catalog is the normalized upstream documentation evidence for one product.
type Catalog struct {
	SchemaVersion int         `json:"schema_version"`
	Product       Product     `json:"product"`
	Operations    []Operation `json:"operations"`
}

// Product describes one candidate plugin and its upstream CTyun product.
type Product struct {
	PluginName string `json:"plugin_name"`
	APIProduct string `json:"api_product"`
	// CtyunProductID records the CTyun OpenAPI docs sid for this product.
	CtyunProductID int `json:"ctyun_product_id"`
	// SourceRevision records the CTyun OpenAPI docs vid when upstream exposes it.
	SourceRevision string            `json:"source_revision"`
	DisplayName    map[string]string `json:"display_name"`
	EndpointURL    string            `json:"endpoint_url"`
	SourceURL      string            `json:"source_url"`
	APIScope       plugin.APIScope   `json:"api_scope,omitempty"`
}

// Operation describes one normalized upstream API operation.
type Operation struct {
	ID          string            `json:"id"`
	APIID       string            `json:"api_id"`
	Title       string            `json:"title"`
	Description map[string]string `json:"description"`
	Category    string            `json:"category"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	ContentType string            `json:"content_type"`
	DocsURL     string            `json:"docs_url"`
	Retryable   bool              `json:"retryable"`
	Examples    []string          `json:"examples"`
	Parameters  []Parameter       `json:"parameters"`
	// ConditionalRequirements preserves reviewed CLI-only parameter rules that
	// upstream prose describes but raw OpenAPI required flags cannot express.
	ConditionalRequirements []plugin.ConditionalRequirement `json:"conditional_requirements,omitempty"`
	Response                Response                        `json:"response"`
	Dangerous               bool                            `json:"dangerous"`
	ExampleResponse         json.RawMessage                 `json:"example_response"`
}

// Parameter captures a raw OpenAPI parameter and optional CLI binding hints.
type Parameter struct {
	Name         string            `json:"name"`
	Location     string            `json:"location"`
	Required     bool              `json:"required"`
	Type         string            `json:"type"`
	Enum         []string          `json:"enum"`
	Default      string            `json:"default"`
	Pattern      string            `json:"pattern"`
	Description  string            `json:"description"`
	Descriptions map[string]string `json:"descriptions"`
	Profile      string            `json:"profile"`
	Argument     string            `json:"argument"`
	CLIName      string            `json:"cli_name"`
	CLIFlag      string            `json:"cli_flag"`
	TableTarget  string            `json:"table_target"`
}

// Response captures response paths and table-generation hints.
type Response struct {
	SuccessCode      string                      `json:"success_code"`
	AcceptedStatuses []plugin.AcceptedStatusRule `json:"accepted_statuses,omitempty"`
	ResultPath       string                      `json:"result_path"`
	RowPath          string                      `json:"row_path"`
	Layout           string                      `json:"layout,omitempty"`
	DefaultColumns   []string                    `json:"default_columns,omitempty"`
	JobIDPath        string                      `json:"job_id_path"`
	Columns          []Column                    `json:"columns"`
}

// Column is a candidate table column derived from upstream response evidence.
type Column struct {
	Key          string            `json:"key"`
	Path         string            `json:"path"`
	LabelEN      string            `json:"label_en"`
	LabelZH      string            `json:"label_zh"`
	Description  string            `json:"description,omitempty"`
	Descriptions map[string]string `json:"descriptions,omitempty"`
}

// Validate checks that the catalog has enough trusted shape for the dev
// pipeline to diff, generate, and review it.
func (catalog Catalog) Validate() error {
	if catalog.SchemaVersion == 0 {
		return fmt.Errorf("schema_version is required")
	}
	if catalog.Product.PluginName == "" {
		return fmt.Errorf("product.plugin_name is required")
	}
	if !plugin.ValidName(catalog.Product.PluginName) {
		return fmt.Errorf("product.plugin_name %s is invalid", catalog.Product.PluginName)
	}
	if catalog.Product.APIProduct == "" {
		return fmt.Errorf("product.api_product is required")
	}
	if catalog.Product.CtyunProductID <= 0 {
		return fmt.Errorf("product.ctyun_product_id is required")
	}
	if err := validateAPIScope(catalog.Product.APIScope); err != nil {
		return err
	}
	seen := make(map[string]bool, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		if err := operation.Validate(); err != nil {
			return err
		}
		if !operationInAPIScope(operation.Path, catalog.Product.APIScope) {
			return fmt.Errorf("operation %s path %s is outside product.api_scope", operation.ID, operation.Path)
		}
		if seen[operation.ID] {
			return fmt.Errorf("operation %s is duplicated", operation.ID)
		}
		seen[operation.ID] = true
	}
	return nil
}

// validateAPIScope checks the optional upstream API selection boundary.
func validateAPIScope(scope plugin.APIScope) error {
	if len(scope.IncludeURIPrefixes) == 0 && len(scope.ExcludeURIPrefixes) == 0 {
		return nil
	}
	for _, prefix := range scope.IncludeURIPrefixes {
		if err := validateURIPrefix("product.api_scope.include_uri_prefixes", prefix); err != nil {
			return err
		}
	}
	for _, prefix := range scope.ExcludeURIPrefixes {
		if err := validateURIPrefix("product.api_scope.exclude_uri_prefixes", prefix); err != nil {
			return err
		}
	}
	return nil
}

// validateURIPrefix checks that prefix can safely match normalized operation
// paths.
func validateURIPrefix(field, prefix string) error {
	if prefix == "" {
		return fmt.Errorf("%s contains an empty prefix", field)
	}
	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("%s prefix %s must start with /", field, prefix)
	}
	if strings.ContainsAny(prefix, " \t\r\n?#") || strings.Contains(prefix, "/../") || strings.Contains(prefix, "/./") {
		return fmt.Errorf("%s prefix %s is invalid", field, prefix)
	}
	return nil
}

// operationInAPIScope reports whether path belongs to scope. Empty scope means
// the catalog has no machine-readable operation boundary yet.
func operationInAPIScope(path string, scope plugin.APIScope) bool {
	if len(scope.IncludeURIPrefixes) > 0 {
		included := false
		for _, prefix := range scope.IncludeURIPrefixes {
			if strings.HasPrefix(path, prefix) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	for _, prefix := range scope.ExcludeURIPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return true
}

// Validate checks the required identity and HTTP shape of one operation.
func (operation Operation) Validate() error {
	if operation.ID == "" {
		return fmt.Errorf("operation id is required")
	}
	if operation.Method == "" {
		return fmt.Errorf("operation %s method is required", operation.ID)
	}
	if !oneOf(operation.Method, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete) {
		return fmt.Errorf("operation %s method %s is unsupported", operation.ID, operation.Method)
	}
	if operation.Path == "" {
		return fmt.Errorf("operation %s path is required", operation.ID)
	}
	if !strings.HasPrefix(operation.Path, "/") {
		return fmt.Errorf("operation %s path must start with /", operation.ID)
	}
	if strings.ContainsAny(operation.Path, " \t\r\n?#") || strings.Contains(operation.Path, "/../") || strings.Contains(operation.Path, "/./") {
		return fmt.Errorf("operation %s path %s is invalid", operation.ID, operation.Path)
	}
	for _, example := range operation.Examples {
		if flag := devOnlyFixtureExampleFlag(example); flag != "" {
			return fmt.Errorf("operation %s example uses dev-only fixture flag %s", operation.ID, flag)
		}
	}
	for _, parameter := range operation.Parameters {
		if parameter.Name == "" {
			return fmt.Errorf("operation %s parameter name is required", operation.ID)
		}
		if !oneOf(parameter.Location, "path", "query", "body", "header") {
			return fmt.Errorf("operation %s parameter %s location %s is unsupported", operation.ID, parameter.Name, parameter.Location)
		}
	}
	for _, rule := range operation.Response.AcceptedStatuses {
		if !validStatusCode(rule.Code) || rule.Code != "900" {
			return fmt.Errorf("operation %s accepted status code %s is invalid", operation.ID, rule.Code)
		}
		if rule.RequiredPath == "" || !validResponsePath(rule.RequiredPath) {
			return fmt.Errorf("operation %s accepted status path %s is invalid", operation.ID, rule.RequiredPath)
		}
	}
	if err := operation.validateConditionalRequirements(); err != nil {
		return err
	}
	return nil
}

// validateConditionalRequirements checks catalog-level CLI requirement rules.
func (operation Operation) validateConditionalRequirements() error {
	seen := make(map[string]bool, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		if parameter.CLIName != "" {
			seen[parameter.CLIName] = true
		}
	}
	for _, requirement := range operation.ConditionalRequirements {
		if !seen[requirement.When.Parameter] {
			return fmt.Errorf("operation %s conditional parameter %s is unknown", operation.ID, requirement.When.Parameter)
		}
		if requirement.When.Equals == "" && len(requirement.When.In) == 0 {
			return fmt.Errorf("operation %s conditional parameter %s has no match value", operation.ID, requirement.When.Parameter)
		}
		if len(requirement.Required) == 0 && len(requirement.AnyOf) == 0 {
			return fmt.Errorf("operation %s conditional parameter %s has no requirements", operation.ID, requirement.When.Parameter)
		}
		for _, name := range append(requirement.Required, requirement.AnyOf...) {
			if !seen[name] {
				return fmt.Errorf("operation %s conditional requirement %s is unknown", operation.ID, name)
			}
		}
	}
	return nil
}

// validStatusCode reports whether status is a numeric CTyun application code.
func validStatusCode(status string) bool {
	if status == "" {
		return false
	}
	for _, char := range status {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// validResponsePath accepts dotted JSON object paths used as response guards.
func validResponsePath(path string) bool {
	if path == "" || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return false
	}
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			return false
		}
		for index, char := range part {
			if index == 0 {
				if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && char != '_' {
					return false
				}
				continue
			}
			if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
				return false
			}
		}
	}
	return true
}

// devOnlyFixtureExampleFlag returns the fixture-mode flag found in a public
// command example.
func devOnlyFixtureExampleFlag(example string) string {
	for _, field := range strings.Fields(example) {
		if oneOf(field, "--offline", "--fixture", "-O") {
			return field
		}
	}
	return ""
}

// oneOf reports whether value equals one of the allowed strings.
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

// decodeJSON decodes JSON while rejecting fields not declared by the target.
func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
