/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package cli

import (
	"encoding/base64"
	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/signing"
	"io"
	"testing"
)

// TestResolvePOSTPolicy keeps storage secrets separate and preserves supplied signatures.
func TestResolvePOSTPolicy(t *testing.T) {
	r := &apicontract.Request{PostPolicy: &apicontract.PostPolicy{Algorithm: "s3-v2", AccessKey: "$param.ak", Policy: "$param.policy", Signature: "$param.sig"}}
	policy := base64.StdEncoding.EncodeToString([]byte(`{"expiration":"2030-01-01T00:00:00Z","conditions":[]}`))
	original := map[string]string{"ak": "storage-ak", "policy": policy}
	got, err := resolvePOSTPolicy(r, original, func(name string) string {
		if name == "CTYUN_STORAGE_SK" {
			return "storage-secret"
		}
		return "eop-secret"
	})
	if err != nil || got["sig"] != signing.SignPOSTPolicyV2(policy, "storage-secret") || original["sig"] != "" {
		t.Fatal(got, err)
	}
	for _, values := range []map[string]string{nil, {"ak": "a"}, {"policy": policy}, {"sig": "s"}, {"ak": "a", "policy": "bad"}, {"ak": "a", "policy": base64.StdEncoding.EncodeToString([]byte(`null`))}} {
		_, err := resolvePOSTPolicy(r, values, func(string) string { return "" })
		if len(values) == 0 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatal("accepted incomplete signing inputs", values)
		}
	}
	values := map[string]string{"ak": "a", "policy": policy, "sig": "external"}
	if got, err := resolvePOSTPolicy(r, values, func(string) string { t.Fatal("read secret for presigned input"); return "" }); err != nil || got["sig"] != "external" {
		t.Fatal(got, err)
	}
	if _, err := resolvePOSTPolicy(r, original, func(string) string { return "" }); err == nil {
		t.Fatal("accepted absent storage secret")
	}
	if _, err := resolvePOSTPolicy(nil, original, nil); err != nil {
		t.Fatal(err)
	}
}

// TestPOSTPolicyRejectsInvalidInputsBeforeUpload checks shared CLI integration without reading files.
func TestPOSTPolicyRejectsInvalidInputsBeforeUpload(t *testing.T) {
	request := &apicontract.Request{Encoding: "multipart", PostPolicy: &apicontract.PostPolicy{Algorithm: "s3-v2", AccessKey: "$param.ak", Policy: "$param.policy", Signature: "$param.sig"}}
	operation := plugin.Operation{Method: "POST", Path: "/upload", Request: request}
	bundle := plugin.Bundle{APIs: plugin.APIs{Operations: map[string]plugin.Operation{"post": operation}}}
	command := plugin.Command{Operation: "post"}
	profile := coreconfig.Profile{EndpointURL: "https://example.test", AccessKey: "gateway-ak", SecretKey: "gateway-sk"}
	_, err := buildAPIRequest(bundle, command, nil, map[string]string{"ak": "storage-ak"}, profile, func(string) string { return "" }, nil, io.Discard, nil, "en-US")
	if err == nil {
		t.Fatal("incomplete storage inputs accepted")
	}
}
