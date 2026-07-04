/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Workspace resolves repo-local OpenAPI harvest and review paths.
type Workspace struct {
	Root string
}

// ProductPath returns a path under openapi/products/<product>.
func (workspace Workspace) ProductPath(product string, parts ...string) string {
	base := filepath.Join(workspace.Root, "openapi", "products", product)
	all := append([]string{base}, parts...)
	return filepath.Join(all...)
}

// HarvestFromFile validates a normalized catalog file and stores it as
// openapi/products/<product>/source.json.
func (workspace Workspace) HarvestFromFile(product, inputPath string) error {
	catalog, err := readCatalog(inputPath)
	if err != nil {
		return err
	}
	if catalog.Product.PluginName != product {
		return fmt.Errorf("input product %s does not match requested product %s", catalog.Product.PluginName, product)
	}
	return workspace.WriteCatalog(workspace.ProductPath(product, "source.json"), catalog)
}

// ReadSource reads openapi/products/<product>/source.json.
func (workspace Workspace) ReadSource(product string) (Catalog, error) {
	return readCatalog(workspace.ProductPath(product, "source.json"))
}

// ReadBaseline reads openapi/products/<product>/baseline.json.
func (workspace Workspace) ReadBaseline(product string) (Catalog, error) {
	return readCatalog(workspace.ProductPath(product, "baseline.json"))
}

// WriteCatalog writes catalog as stable, indented JSON.
func (workspace Workspace) WriteCatalog(path string, catalog Catalog) error {
	if err := catalog.Validate(); err != nil {
		return err
	}
	return writeJSON(path, catalog)
}

// readCatalog reads and validates a normalized catalog file.
func readCatalog(path string) (Catalog, error) {
	var catalog Catalog
	data, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	if err := decodeJSON(data, &catalog); err != nil {
		return Catalog{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := catalog.Validate(); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

// writeJSON writes value as stable, indented JSON.
func writeJSON(path string, value any) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0o644)
}

// writeText writes a text file, creating parent directories as needed.
func writeText(path, value string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), 0o644)
}

// compactRawMessage removes insignificant JSON whitespace from raw fixtures.
func compactRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return raw
	}
	return buffer.Bytes()
}
