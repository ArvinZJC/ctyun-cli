/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package openapipipeline

import (
	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"testing"
)

// TestNativeScopeAndDrift keeps native selection separate from EOP URI prefixes.
func TestNativeScopeAndDrift(t *testing.T) {
	scope := plugin.APIScope{IncludeURIPrefixes: []string{"/v1/"}, NativeServices: []string{"s3"}}
	op := Operation{ID: "test.native", Path: "/", Native: &apicontract.Native{Service: "s3", Addressing: "path"}}
	if !catalogOperationInScope(op, scope) || catalogOperationInScope(Operation{Path: "/"}, scope) {
		t.Fatal("scope boundary changed")
	}
	if err := validateAPIScope(scope); err != nil {
		t.Fatal(err)
	}
	for _, services := range [][]string{{"unknown"}, {"s3", "s3"}} {
		scope.NativeServices = services
		if err := validateAPIScope(scope); err == nil {
			t.Fatal("invalid scope accepted")
		}
	}
	if err := validateOperationTransport(op); err == nil {
		t.Fatal("native response contract omitted")
	}
	report := DiffReport{}
	next := op
	next.Native = &apicontract.Native{Service: "s3", Addressing: "virtual"}
	compareOperationMetadata(&report, op, next)
	if len(report.Changes) != 1 {
		t.Fatal(report)
	}
	catalog := Catalog{Operations: []Operation{op}}
	if !catalogUsesTransport(catalog) || buildAPIs(catalog).Operations[op.ID].Native != op.Native {
		t.Fatal("native contract lost")
	}
}

// TestConstantParameterBindings emits fixed protocol fields without turning help defaults into inputs.
func TestConstantParameterBindings(t *testing.T) {
	parameter := Parameter{Name: "Action", Location: "query", Type: "String", Constant: "GetCapacity"}
	catalog := Catalog{Operations: []Operation{{ID: "read", Parameters: []Parameter{parameter}}}}
	if got := buildAPIs(catalog).Operations["read"].Query["Action"]; got != "GetCapacity" {
		t.Fatal(got)
	}
	if len(buildCommands(catalog).Commands[0].Parameters) != 0 {
		t.Fatal("constant exposed as option")
	}
	op := Operation{ID: "read", Method: "GET", Path: "/", Description: map[string]string{"en-US": "Read", "en-GB": "Read", "zh-CN": "读取"}, Parameters: []Parameter{parameter}}
	if err := op.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Parameter{{Name: "Action", Location: "query", Type: "String", Constant: "GetCapacity", CLIName: "action"}, {Name: "Action", Location: "query", Type: "String", Constant: "$param.action"}} {
		op.Parameters = []Parameter{bad}
		if err := op.Validate(); err == nil {
			t.Fatal("ambiguous binding accepted")
		}
	}
}
