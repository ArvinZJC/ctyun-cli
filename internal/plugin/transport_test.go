/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"testing"
)

// TestTransportRejectsImplicitFileRoles prevents ordinary values from opening local files.
func TestTransportRejectsImplicitFileRoles(t *testing.T) {
	for _, tc := range []struct {
		request apicontract.Request
		input   string
		valid   bool
	}{
		{apicontract.Request{Encoding: "file", Document: "$param.data"}, "", false},
		{apicontract.Request{Encoding: "file", Document: "$param.data"}, "file", true},
		{apicontract.Request{Encoding: "multipart", Parts: []apicontract.Part{{Name: "file", Source: "$param.data", File: true}}}, "", false},
		{apicontract.Request{Encoding: "multipart", Parts: []apicontract.Part{{Name: "field", Source: "$param.data"}}}, "file", false},
	} {
		command := Command{Parameters: []Parameter{{Name: "data", Flag: "data", Input: tc.input}}}
		err := validateTransportBindings(command, Operation{Request: &tc.request})
		if (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc, err)
		}
	}
}

// TestTransportBindingSchema rejects contradictory input roles and missing sources.
func TestTransportBindingSchema(t *testing.T) {
	for _, tc := range []struct {
		command   Command
		operation Operation
		valid     bool
	}{
		{Command{Parameters: []Parameter{{Input: "other"}}}, Operation{}, false},
		{Command{Download: true, Parameters: []Parameter{{Flag: "output-file"}}}, Operation{}, false},
		{Command{}, Operation{Request: &apicontract.Request{Encoding: "xml", Document: "$param.missing"}}, false},
		{Command{}, Operation{Request: &apicontract.Request{Encoding: "xml", Document: "literal"}}, false},
		{Command{}, Operation{Request: &apicontract.Request{Encoding: "xml", Document: "$arg.missing"}}, false},
		{Command{Path: []string{"example", "{doc}"}}, Operation{Request: &apicontract.Request{Encoding: "xml", Document: "$arg.doc"}}, true},
		{Command{}, Operation{Request: &apicontract.Request{Encoding: "xml", Document: "$profile.region"}}, true},
		{Command{}, Operation{Request: &apicontract.Request{Encoding: "json"}}, true},
		{Command{}, Operation{Request: &apicontract.Request{Encoding: "multipart", Parts: []apicontract.Part{{Source: "$profile.region"}}}}, true},
		{Command{Parameters: []Parameter{{Name: "data", Input: "file"}}}, Operation{}, false},
		{Command{Parameters: []Parameter{{Name: "data", Input: "file", ValueType: ParameterValueInteger}}}, Operation{Request: &apicontract.Request{Encoding: "file", Document: "$param.data"}}, false},
		{Command{Parameters: []Parameter{{Name: "data", Input: "file"}}}, Operation{Request: &apicontract.Request{Encoding: "file", Document: "$param.data"}, Headers: map[string]string{"x": "$param.data"}}, false},
		{Command{Download: true}, Operation{}, false},
		{Command{Download: true}, Operation{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 204, Format: "empty"}}}}, false},
		{Command{Download: true}, Operation{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "binary"}}}}, true},
	} {
		if err := validateTransportBindings(tc.command, tc.operation); (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	for _, op := range []Operation{{Method: "BAD"}, {Method: "GET", Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}}, AcceptedStatuses: []AcceptedStatusRule{{Code: "900"}}}, {Method: "PUT", Request: &apicontract.Request{Encoding: "xml", Document: "x"}, Body: map[string]string{"x": "x"}}} {
		if err := validateTransport(op); err == nil {
			t.Fatal(op)
		}
	}
	if err := validateTransport(Operation{Method: "GET"}); err != nil {
		t.Fatal(err)
	}
}

// TestTransportCoreFloorAndOutputVocabulary keeps schema compatibility and visible formats aligned.
func TestTransportCoreFloorAndOutputVocabulary(t *testing.T) {
	op := Operation{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "binary"}}}}
	bundle := Bundle{APIs: APIs{Operations: map[string]Operation{"get": op}}}
	for _, tc := range []struct {
		requirement, version string
		valid                bool
	}{{">=0.4.0", "0.5.0", false}, {">=0.5.0 <1.0.0", "0.4.0", false}, {">=0.5.0 <1.0.0", "0.5.0", true}, {">=invalid", "0.5.0", false}, {">=0.5.0", "0.5.0-dev", true}} {
		bundle.Manifest.Requires.Ctyun = tc.requirement
		if err := validateTransportCore(bundle, tc.version); (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	if !BinaryOnly(op) || len(OutputFormats(op)) != 1 {
		t.Fatal("binary formats")
	}
	op.Response.Variants = append(op.Response.Variants, apicontract.Variant{Status: 201, Format: "json"})
	if BinaryOnly(op) {
		t.Fatal("mixed binary only")
	}
	op.Response.Variants = op.Response.Variants[1:]
	if len(OutputFormats(op)) != 3 {
		t.Fatal("structured formats")
	}
	if BinaryOnly(Operation{}) || len(OutputFormats(Operation{})) != 2 {
		t.Fatal("legacy formats")
	}
}

// TestXMLTableAndWaiterCompatibility validates projection keys and safe polling representations.
func TestXMLTableAndWaiterCompatibility(t *testing.T) {
	valid := apicontract.XMLTable{Rows: []apicontract.XMLName{{Local: "root"}}, Columns: map[string]apicontract.XMLSelector{"name": {}}}
	columns := []TableColumn{{Key: "name", Labels: map[string]string{"en-US": "Name", "en-GB": "Name", "zh-CN": "名称"}}}
	for _, tc := range []struct {
		table Table
		valid bool
	}{
		{Table{XML: &apicontract.XMLTable{}}, false},
		{Table{RowPath: "$", XML: &valid, Columns: columns}, false},
		{Table{XML: &valid}, false},
		{Table{XML: &valid, Columns: []TableColumn{{Key: "other"}}}, false},
		{Table{XML: &valid, Columns: columns}, true},
	} {
		if err := validateTables(Tables{Tables: map[string]Table{"view": tc.table}}); (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
	for _, op := range []Operation{
		{Request: &apicontract.Request{Encoding: "file"}},
		{Request: &apicontract.Request{Encoding: "multipart"}},
		{Response: &apicontract.Response{Variants: []apicontract.Variant{{Format: "xml"}}}},
	} {
		if WaiterApplies(Bundle{APIs: APIs{Operations: map[string]Operation{"read": op}}}, Command{Operation: "read"}, Waiter{}) {
			t.Fatal("unsafe waiter accepted")
		}
	}
	if !WaiterApplies(Bundle{APIs: APIs{Operations: map[string]Operation{"read": {Retryable: true, Response: &apicontract.Response{Variants: []apicontract.Variant{{Format: "json"}}}}}}}, Command{Operation: "read"}, Waiter{}) {
		t.Fatal("JSON waiter rejected")
	}
}

// TestFormJSONFieldBindingsRequireDeclaredBodyFields rejects misspelled field encoders.
func TestFormJSONFieldBindingsRequireDeclaredBodyFields(t *testing.T) {
	operation := Operation{Request: &apicontract.Request{Encoding: "form", JSONFields: []string{"statement"}}}
	if err := validateTransportBindings(Command{}, operation); err == nil {
		t.Fatal("unbound JSON field accepted")
	}
	operation.Body = map[string]string{"statement": "$param.statement"}
	if err := validateTransportBindings(Command{Parameters: []Parameter{{Name: "statement"}}}, operation); err != nil {
		t.Fatal(err)
	}
}
