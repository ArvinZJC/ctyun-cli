/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package apicontract

import "testing"

// TestNativeContractsRejectAmbiguousRouting checks explicit storage metadata boundaries.
func TestNativeContractsRejectAmbiguousRouting(t *testing.T) {
	response := &Response{Variants: []Variant{{Status: 200, Format: "empty"}}}
	if err := ValidateNative(nil, "", nil); err != nil {
		t.Fatal(err)
	}
	valid := Native{Versions: []string{"v4"}, Service: "s3", Addressing: "path", Bucket: "$arg.bucket", Object: "$param.object", Subresources: []string{"acl"}, V2QueryKeys: []string{"acl"}}
	if err := ValidateNative(&valid, "/", response); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Native){func(n *Native) { n.Service = "bad" }, func(n *Native) { n.Versions = []string{"v3"} }, func(n *Native) { n.Versions = []string{"v4", "v4"} }, func(n *Native) { n.Addressing = "bad" }, func(n *Native) { n.Bucket = "" }, func(n *Native) { n.Bucket = "literal" }, func(n *Native) { n.Subresources = []string{"a=b"} }, func(n *Native) { n.V2QueryKeys = []string{"acl", "acl"} }} {
		n := valid
		change(&n)
		if err := ValidateNative(&n, "/", response); err == nil {
			t.Fatal("invalid native contract accepted", n)
		}
	}
	if err := ValidateNative(&valid, "/ignored", response); err == nil {
		t.Fatal("ignored path accepted")
	}
}

// TestNativeHeaderSourcesUseParameters rejects argument or empty MIME/map bindings.
func TestNativeHeaderSourcesUseParameters(t *testing.T) {
	n := Native{Service: "s3", Addressing: "path", ContentType: "$arg.type"}
	if err := ValidateNative(&n, "/", &Response{}); err == nil {
		t.Fatal("argument header source accepted")
	}
}
