/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestTransportFixtureUnavailableUsesSharedDiagnostic prevents an absent
// captured fixture from being treated as the plugin directory to read.
func TestTransportFixtureUnavailableUsesSharedDiagnostic(t *testing.T) {
	command := plugin.Command{ID: "demo.retired", Operation: "demo.retired"}
	bundle := plugin.Bundle{Dir: t.TempDir(), APIs: plugin.APIs{Operations: map[string]plugin.Operation{
		command.Operation: {Method: "GET", Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}}},
	}}}
	err := runTransportCommand(io.Discard, io.Discard, bundle, command, nil, nil, globalOptions{Fixture: true}, coreconfig.Profile{}, nil, nil, nil, "")
	requireDiagnosticKey(t, err, "error.command_missing_fixture_response")
}

// TestTransferOptionsAreCommandOwned tests command capability based parsing.
func TestTransferOptionsAreCommandOwned(t *testing.T) {
	command := plugin.Command{Download: true}
	values, err := parseCommandParameters(command, []string{"--output-file", "download.bin", "--overwrite"}, "en-US")
	if err != nil || values["__ctyun_output_file"] != "download.bin" || values["__ctyun_overwrite"] != "true" {
		t.Fatalf("%v %v", values, err)
	}
	if _, err := parseCommandParameters(plugin.Command{}, []string{"--output-file", "download.bin"}, "en-US"); err == nil {
		t.Fatal("unrelated command accepted transfer flags")
	}
}

// TestExpandedMultipartPartsPreserveOrderAndRejectInvalidMaps verifies user metadata expansion.
func TestExpandedMultipartPartsPreserveOrderAndRejectInvalidMaps(t *testing.T) {
	parts, err := expandMultipartPart("x-amz-meta-", `{"z":"last","a":"中文"}`)
	if err != nil || len(parts) != 2 || parts[0].Name != "x-amz-meta-a" || parts[0].Value != "中文" {
		t.Fatal(parts, err)
	}
	for _, value := range []string{`null`, `[]`, `{"x":1}`, `{"":"empty"}`, `{"x\n":"bad"}`, `{"A":"one","a":"two"}`} {
		if _, err := expandMultipartPart("x-amz-meta-", value); err == nil {
			t.Fatal("invalid fields accepted", value)
		}
	}
}

// TestMixedXMLAndEmptyTables renders status information when no XML body is returned.
func TestMixedXMLAndEmptyTables(t *testing.T) {
	table := plugin.Table{XML: &apicontract.XMLTable{Rows: []apicontract.XMLName{{Local: "PostResponse"}}, Columns: map[string]apicontract.XMLSelector{"status": {Path: []apicontract.XMLName{{Local: "Status"}}}}}, Columns: []plugin.TableColumn{{Key: "status", Path: "status"}}}
	rows, err := rowsFromPayload(map[string]any{"status": json.Number("204")}, table)
	if err != nil || len(rows) != 1 || rows[0]["status"] != "204" {
		t.Fatal(rows, err)
	}
}

// TestPrepareExpandedMultipartUsesSharedEncoder checks ordered map expansion and error propagation.
func TestPrepareExpandedMultipartUsesSharedEncoder(t *testing.T) {
	operation := plugin.Operation{Request: &apicontract.Request{Encoding: "multipart", Parts: []apicontract.Part{{Name: "x-meta-", Source: "$param.metadata", Expand: true}}}}
	for _, value := range []string{`{"location":"中文"}`, `{"location":null}`} {
		body, err := prepareCommandBody(operation, plugin.Command{}, nil, map[string]string{"metadata": value}, coreconfig.Profile{}, nil)
		if strings.Contains(value, "null") {
			if err == nil {
				if closeErr := body.Close(); closeErr != nil {
					t.Error(closeErr)
				}
				t.Fatal("invalid map accepted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		reader, err := body.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if closeErr := reader.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		if closeErr := body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		if err != nil || !strings.Contains(string(data), `name=x-meta-location`) || !strings.Contains(string(data), "中文") {
			t.Fatal(string(data), err)
		}
	}
}

// TestXMLTablesUseActualResponseStatus rejects wrong roots and projects transport status separately.
func TestXMLTablesUseActualResponseStatus(t *testing.T) {
	table := plugin.Table{XML: &apicontract.XMLTable{Rows: []apicontract.XMLName{{Local: "root"}}, Columns: map[string]apicontract.XMLSelector{"kind": {Name: true}}}, Columns: []plugin.TableColumn{{Key: "kind"}, {Key: "status", Path: "status"}}}
	payload := map[string]any{"name": map[string]any{"local": "root"}, "status": json.Number("201")}
	rows, err := rowsFromPayload(payload, table)
	if err != nil || len(rows) != 1 || rows[0]["status"] != "201" || rows[0]["kind"] != "root" {
		t.Fatal(rows, err)
	}
	payload["name"] = map[string]any{"local": "wrong"}
	if _, err := rowsFromPayload(payload, table); err == nil {
		t.Fatal("wrong XML root accepted")
	}
}
