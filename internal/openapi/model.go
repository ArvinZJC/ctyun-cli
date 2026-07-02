/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package openapi supports the repository-local CTyun OpenAPI harvest and
// review pipeline used to maintain plugin metadata.
package openapi

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
}

// Operation describes one normalized upstream API operation.
type Operation struct {
	ID              string          `json:"id"`
	APIID           string          `json:"api_id"`
	Title           string          `json:"title"`
	Category        string          `json:"category"`
	Method          string          `json:"method"`
	Path            string          `json:"path"`
	ContentType     string          `json:"content_type"`
	DocsURL         string          `json:"docs_url"`
	Retryable       bool            `json:"retryable"`
	Parameters      []Parameter     `json:"parameters"`
	Response        Response        `json:"response"`
	Dangerous       bool            `json:"dangerous"`
	ExampleResponse json.RawMessage `json:"example_response"`
}

// Parameter captures a raw OpenAPI parameter and optional CLI binding hints.
type Parameter struct {
	Name        string   `json:"name"`
	Location    string   `json:"location"`
	Required    bool     `json:"required"`
	Type        string   `json:"type"`
	Enum        []string `json:"enum"`
	Default     string   `json:"default"`
	Pattern     string   `json:"pattern"`
	Description string   `json:"description"`
	Profile     string   `json:"profile"`
	Argument    string   `json:"argument"`
	CLIName     string   `json:"cli_name"`
	TableTarget string   `json:"table_target"`
}

// Response captures response paths and table-generation hints.
type Response struct {
	SuccessCode string   `json:"success_code"`
	ResultPath  string   `json:"result_path"`
	RowPath     string   `json:"row_path"`
	JobIDPath   string   `json:"job_id_path"`
	Columns     []Column `json:"columns"`
}

// Column is a candidate table column derived from upstream response evidence.
type Column struct {
	Key     string `json:"key"`
	Path    string `json:"path"`
	LabelEN string `json:"label_en"`
	LabelZH string `json:"label_zh"`
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
	seen := make(map[string]bool, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		if err := operation.Validate(); err != nil {
			return err
		}
		if seen[operation.ID] {
			return fmt.Errorf("operation %s is duplicated", operation.ID)
		}
		seen[operation.ID] = true
	}
	return nil
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
	for _, parameter := range operation.Parameters {
		if parameter.Name == "" {
			return fmt.Errorf("operation %s parameter name is required", operation.ID)
		}
		if !oneOf(parameter.Location, "path", "query", "body", "header") {
			return fmt.Errorf("operation %s parameter %s location %s is unsupported", operation.ID, parameter.Name, parameter.Location)
		}
	}
	return nil
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
