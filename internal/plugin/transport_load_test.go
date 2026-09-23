/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// TestLoadBundleTransportContracts rejects invalid cross-file HTTP bindings.
func TestLoadBundleTransportContracts(t *testing.T) {
	for _, kind := range []string{"core-floor", "missing-table", "xml-legacy", "xml-json", "file-role", "invalid-operation", "valid-xml"} {
		t.Run(kind, func(t *testing.T) {
			dir := writeBundle(t, "ecs", ">=0.5.0 <1.0.0")
			bundle, err := LoadBundle(dir, "0.5.0")
			if err != nil {
				t.Fatal(err)
			}
			command := &bundle.Commands.Commands[0]
			op := bundle.APIs.Operations[command.Operation]
			table := bundle.Tables.Tables[command.Table]
			table.RowPath = ""
			table.XML = &apicontract.XMLTable{Rows: []apicontract.XMLName{{Local: "root"}}, Columns: map[string]apicontract.XMLSelector{"name": {}}}
			table.Columns = table.Columns[1:2]
			op.Response = &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "xml"}}}
			switch kind {
			case "core-floor":
				bundle.Manifest.Requires.Ctyun = ">=0.4.0 <1.0.0"
			case "missing-table":
				command.Table = ""
			case "xml-legacy":
				op.Response = nil
			case "xml-json":
				op.Response.Variants[0].Format = "json"
			case "file-role":
				command.Parameters[0].Input = "file"
			case "invalid-operation":
				op.Response.Variants[0].Format = "bad"
			}
			bundle.APIs.Operations[command.Operation] = op
			bundle.Tables.Tables["ecs.instance.list"] = table
			for name, value := range map[string]any{"plugin.json": bundle.Manifest, "commands.json": bundle.Commands, "apis.json": bundle.APIs, "tables.json": bundle.Tables} {
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				mustWrite(t, filepath.Join(dir, name), string(data))
			}
			_, err = LoadBundle(dir, "0.5.0")
			if (err == nil) != (kind == "valid-xml") {
				t.Fatalf("%s: %v", kind, err)
			}
		})
	}
}
