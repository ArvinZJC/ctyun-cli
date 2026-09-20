/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package apicontract

import "testing"

// TestInvalidContractFields checks malformed declarations before they can affect IO.
func TestInvalidContractFields(t *testing.T) {
	for _, request := range []Request{
		{Encoding: "json", Document: "x"}, {Encoding: "xml"}, {Encoding: "file"},
		{Encoding: "multipart"}, {Encoding: "form", Parts: []Part{{Name: "a", Source: "x"}}},
		{Encoding: "multipart", Parts: []Part{{Name: "a", Source: "x"}, {Name: "a", Source: "y"}}},
		{Encoding: "multipart", Parts: []Part{{Name: "a\n", Source: "x"}}},
		{Encoding: "multipart", Parts: []Part{{Name: "a"}}},
		{Encoding: "multipart", Parts: []Part{{Name: "a", Source: "x", ContentType: "bad"}}},
	} {
		if err := Validate("POST", "", &request, nil); err == nil {
			t.Errorf("accepted %+v", request)
		}
	}
	for _, response := range []Response{
		{}, {Variants: []Variant{{Status: 200, Format: "unknown"}}},
		{Variants: []Variant{{Status: 200, Format: "json"}, {Status: 200, Format: "json"}}},
		{Variants: []Variant{{Status: 200, Format: "text"}}, Success: []Check{{Path: "x", Values: []string{"0"}}}},
		{Variants: []Variant{{Status: 200, Format: "json", Headers: map[string]string{"": "ETag"}}}},
		{Variants: []Variant{{Status: 200, Format: "json", Headers: map[string]string{"x": "bad header"}}}},
		{Variants: []Variant{{Status: 200, Format: "json"}}, Success: []Check{{}}},
		{Variants: []Variant{{Status: 200, Format: "json"}}, Success: []Check{{Path: "a..b", Values: []string{"0"}}}},
		{Variants: []Variant{{Status: 200, Format: "json"}}, Success: []Check{{Path: "a", Values: []string{""}}}},
	} {
		if err := Validate("GET", "", nil, &response); err == nil {
			t.Errorf("accepted %+v", response)
		}
	}
	for _, pair := range [][2]string{{"json", "text/plain"}, {"xml", "bad"}, {"multipart", "multipart/form-data; boundary=fixed"}} {
		req := &Request{Encoding: pair[0]}
		if req.Encoding == "xml" {
			req.Document = "x"
		}
		if req.Encoding == "multipart" {
			req.Parts = []Part{{Name: "a", Source: "x"}}
		}
		if err := Validate("POST", pair[1], req, nil); err == nil {
			t.Errorf("accepted MIME %v", pair)
		}
	}
	if err := Validate("TRACE", "", nil, nil); err == nil {
		t.Fatal("TRACE accepted")
	}
	for _, name := range []string{"", "a b", "a:b", "中文", "a\n"} {
		if HeaderName(name) {
			t.Errorf("header %q accepted", name)
		}
	}
	if !HeaderName("X-Test_123") {
		t.Fatal("valid header rejected")
	}
	for _, req := range []Request{{Encoding: "xml", Document: "x"}, {Encoding: "json"}, {Encoding: "form"}, {Encoding: "file", Document: "x"}, {Encoding: "multipart", Parts: []Part{{Name: "a", Source: "x", ContentType: "text/plain"}}}} {
		if err := Validate("POST", "", &req, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := Validate("POST", "text/xml; charset=utf-8", &Request{Encoding: "xml", Document: "x"}, nil); err != nil {
		t.Fatal(err)
	}
}

// TestXMLTableValidation checks explicit namespace selector syntax.
func TestXMLTableValidation(t *testing.T) {
	for _, table := range []XMLTable{
		{}, {Rows: []XMLName{{Local: "a:b"}}, Columns: map[string]XMLSelector{"x": {}}},
		{Rows: []XMLName{{Local: "r"}}, Columns: map[string]XMLSelector{"": {}}},
		{Rows: []XMLName{{Local: "r"}}, Columns: map[string]XMLSelector{"x": {Path: []XMLName{{}}}}},
		{Rows: []XMLName{{Local: "r"}}, Columns: map[string]XMLSelector{"x": {Attribute: &XMLName{}}}},
	} {
		if err := table.Validate(); err == nil {
			t.Errorf("accepted %#v", table)
		}
	}
}

// TestExplicitResponseSchema rejects projections the decoder cannot honour.
func TestExplicitResponseSchema(t *testing.T) {
	for _, variant := range []Variant{{Status: 200, Format: "json", Headers: map[string]string{"id": "X-ID"}}, {Status: 200, Format: "xml", XMLRoot: &XMLName{}}, {Status: 200, Format: "text", XMLRoot: &XMLName{Local: "root"}}} {
		if err := Validate("GET", "", nil, &Response{Variants: []Variant{variant}}); err == nil {
			t.Fatal(variant)
		}
	}
	for _, method := range []string{"HEAD", "OPTIONS"} {
		if err := Validate(method, "", nil, nil); err == nil {
			t.Fatal(method)
		}
	}
	for _, table := range []XMLTable{
		{RowPaths: [][]XMLName{{}}, Columns: map[string]XMLSelector{"x": {}}},
		{RowPaths: [][]XMLName{{{Local: "root"}}, {{Local: "root"}}}, Columns: map[string]XMLSelector{"x": {}}},
		{Rows: []XMLName{{Local: "root"}}, Columns: map[string]XMLSelector{"x": {Name: true, Attribute: &XMLName{Local: "id"}}}},
	} {
		if err := table.Validate(); err == nil {
			t.Fatal(table)
		}
	}
	if err := (XMLTable{Rows: []XMLName{{Local: "root"}}, Columns: map[string]XMLSelector{"x": {Name: true}}}).Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestFormJSONFieldDeclarationsRejectAmbiguity confines field encoding to explicit forms.
func TestFormJSONFieldDeclarationsRejectAmbiguity(t *testing.T) {
	for _, request := range []Request{{Encoding: "json", JSONFields: []string{"x"}}, {Encoding: "form", JSONFields: []string{""}}, {Encoding: "form", JSONFields: []string{"x", "x"}}} {
		if err := Validate("POST", "", &request, nil); err == nil {
			t.Fatal(request)
		}
	}
	if err := Validate("POST", "", &Request{Encoding: "form", JSONFields: []string{"statement"}}, nil); err != nil {
		t.Fatal(err)
	}
}
