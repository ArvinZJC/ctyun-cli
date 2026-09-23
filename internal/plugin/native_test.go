/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package plugin

import (
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// TestNativeOperationValidation checks explicit routing and bound resource inputs.
func TestNativeOperationValidation(t *testing.T) {
	op := Operation{Method: "GET", Path: "/", Native: &apicontract.Native{Service: "s3", Addressing: "path", Bucket: "$arg.bucket"}, Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "empty"}}}}
	cmd := Command{Path: []string{"native", "{bucket}"}}
	if err := validateTransport(op); err != nil {
		t.Fatal(err)
	}
	if err := validateTransportBindings(cmd, op); err != nil {
		t.Fatal(err)
	}
	op.Native.Bucket = "$arg.missing"
	if err := validateTransportBindings(cmd, op); err == nil {
		t.Fatal("missing resource source accepted")
	}
	op.Native.Service = "unknown"
	if err := validateTransport(op); err == nil {
		t.Fatal("unknown signer accepted")
	}
	op.Native.Service = "s3"
	op.Native.PolicyAuth = true
	if err := validateTransport(op); err == nil {
		t.Fatal("policy authentication without policy accepted")
	}
}

// TestNativeResourceSourcesMustBeScalar rejects file and composite resource inputs.
func TestNativeResourceSourcesMustBeScalar(t *testing.T) {
	op := Operation{Native: &apicontract.Native{Bucket: "$param.bucket"}}
	for _, parameter := range []Parameter{{Name: "bucket", Input: "file"}, {Name: "bucket", ValueType: ParameterValueInteger}} {
		if err := validateTransportBindings(Command{Parameters: []Parameter{parameter}}, op); err == nil {
			t.Fatal("non-string bucket accepted")
		}
	}
	if err := validateTransportBindings(Command{Parameters: []Parameter{{Name: "bucket"}}}, op); err != nil {
		t.Fatal(err)
	}
}

// TestNativeMetadataRequiresStringMap checks native header expansion declarations.
func TestNativeMetadataRequiresStringMap(t *testing.T) {
	op := Operation{Native: &apicontract.Native{Metadata: "$param.metadata"}}
	if err := validateTransportBindings(Command{}, op); err == nil {
		t.Fatal("missing metadata map accepted")
	}
	if err := validateTransportBindings(Command{Parameters: []Parameter{{Name: "metadata", ValueType: ParameterValueStringMap}}}, op); err != nil {
		t.Fatal(err)
	}
}
