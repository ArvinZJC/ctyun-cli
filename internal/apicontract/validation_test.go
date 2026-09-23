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

// TestPOSTPolicyContractsRejectAmbiguousSources checks protocol fields before signing.
func TestPOSTPolicyContractsRejectAmbiguousSources(t *testing.T) {
	good := Request{Encoding: "multipart", Parts: []Part{{Name: "AWSAccessKeyId", Source: "$param.ak"}, {Name: "policy", Source: "$param.policy"}, {Name: "Signature", Source: "$param.sig"}, {Name: "file", Source: "$param.file", File: true}}, PostPolicy: &PostPolicy{Algorithm: "s3-v2", AccessKey: "$param.ak", Policy: "$param.policy", Signature: "$param.sig"}}
	if err := Validate("POST", "", &good, nil); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Request){func(r *Request) { r.PostPolicy.Algorithm = "unknown" }, func(r *Request) { r.PostPolicy.Policy = "$arg.policy" }, func(r *Request) { r.PostPolicy.Signature = r.PostPolicy.Policy }, func(r *Request) { r.Parts = r.Parts[:2] }, func(r *Request) { r.Parts[0].File = true }, func(r *Request) { r.Parts[0].Expand = true }} {
		bad := good
		policy := *good.PostPolicy
		bad.PostPolicy = &policy
		bad.Parts = append([]Part(nil), good.Parts...)
		change(&bad)
		if Validate("POST", "", &bad, nil) == nil {
			t.Fatal("accepted malformed policy", bad)
		}
	}
	if Validate("PUT", "", &good, nil) == nil {
		t.Fatal("signed policy accepted for PUT")
	}
	good.Parts[0].Expand = true
	good.Parts[0].File = true
	good.PostPolicy = nil
	if Validate("POST", "", &good, nil) == nil {
		t.Fatal("file map accepted")
	}
}

// TestExplicitPOSTRedirect requires an empty response and Location projection for success redirects.
func TestExplicitPOSTRedirect(t *testing.T) {
	response := &Response{Variants: []Variant{{Status: 303, Format: "empty", Headers: map[string]string{"location": "Location"}}}}
	if err := Validate("POST", "", nil, response); err != nil {
		t.Fatal(err)
	}
	if Validate("GET", "", nil, response) == nil {
		t.Fatal("GET redirect accepted as success")
	}
	response.Variants[0].Headers = nil
	if Validate("POST", "", nil, response) == nil {
		t.Fatal("redirect without Location declaration")
	}
}

// TestFormMemberContracts rejects conflicting encoders and duplicate list declarations.
func TestFormMemberContracts(t *testing.T) {
	for _, r := range []Request{{Encoding: "json", MemberFields: []string{"Tags"}}, {Encoding: "form", MemberFields: []string{""}}, {Encoding: "form", MemberFields: []string{"Tags", "Tags"}}, {Encoding: "form", JSONFields: []string{"Tags"}, MemberFields: []string{"Tags"}}} {
		if err := Validate("POST", "", &r, nil); err == nil {
			t.Fatal("invalid list contract accepted", r)
		}
	}
	if err := Validate("POST", "", &Request{Encoding: "form", MemberFields: []string{"Tags"}}, nil); err != nil {
		t.Fatal(err)
	}
}
