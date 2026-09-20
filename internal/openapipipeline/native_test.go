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
