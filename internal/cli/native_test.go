/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package cli

import (
	"io"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestNativeEnvironmentIsolation exercises explicit storage requirements and EOP independence.
func TestNativeEnvironmentIsolation(t *testing.T) {
	env := map[string]string{"CTYUN_AK": "eop-ak", "CTYUN_SK": "eop-sk", "CTYUN_STORAGE_AK": "native-ak", "CTYUN_STORAGE_SK": "native-sk", "CTYUN_STORAGE_ENDPOINT": "https://storage.example", "CTYUN_STORAGE_REGION": "cn"}
	get := func(k string) string { return env[k] }
	contract := &apicontract.Native{Service: "s3", Addressing: "path", Bucket: "$arg.bucket", Object: "$param.object", Subresources: []string{"acl"}}
	args := map[string]string{"bucket": "test-bucket"}
	params := map[string]string{"object": "a//../中%"}
	op := plugin.Operation{Native: contract, Method: "GET", Path: "/", Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "empty"}}}, Query: map[string]string{"versionId": "$param.version"}}
	bundle := plugin.Bundle{APIs: plugin.APIs{Operations: map[string]plugin.Operation{"native": op}}}
	cmd := plugin.Command{Operation: "native"}
	for _, query := range []string{"", "version"} {
		params["version"] = query
		spec, err := buildAPIRequest(bundle, cmd, args, params, config.Profile{EndpointURL: "https://eop.example", AccessKey: "eop-ak", SecretKey: "eop-sk"}, get, nil, io.Discard, nil, "en-US")
		if err != nil {
			t.Fatal(err)
		}
		req, err := client.BuildRequest(spec)
		if err != nil {
			t.Fatal(err)
		}
		if req.URL.Host != "storage.example" || !strings.Contains(req.Header.Get("Authorization"), "native-ak/") || !strings.Contains(req.URL.RawQuery, "acl=") {
			t.Fatal(req)
		}
	}
	for _, key := range []string{"CTYUN_STORAGE_AK", "CTYUN_STORAGE_SK", "CTYUN_STORAGE_ENDPOINT", "CTYUN_STORAGE_REGION"} {
		before := env[key]
		delete(env, key)
		if _, _, _, err := nativeRequest(contract, args, params, get); err == nil {
			t.Fatal("missing storage setting accepted", key)
		}
		env[key] = before
	}
	env["CTYUN_STORAGE_SIGNATURE_VERSION"] = "post-policy"
	if _, _, _, err := nativeRequest(contract, args, params, get); err == nil {
		t.Fatal("environment bypassed signing")
	}
	env["CTYUN_STORAGE_SIGNATURE_VERSION"] = "v2"
	contract.Versions = []string{"v4"}
	if _, _, _, err := nativeRequest(contract, args, params, get); err == nil {
		t.Fatal("V2 accepted for V4-only service")
	}
	contract.Versions = nil
	if _, _, _, err := nativeRequest(contract, nil, params, get); err == nil {
		t.Fatal("missing bucket accepted")
	}
	contract.PolicyAuth = true
	delete(env, "CTYUN_STORAGE_AK")
	delete(env, "CTYUN_STORAGE_SK")
	if native, _, _, err := nativeRequest(contract, args, params, get); err != nil || native.Version != "post-policy" {
		t.Fatal(native, err)
	}
}

// TestNativeObjectHeaders validates MIME overrides and metadata maps before transmission.
func TestNativeObjectHeaders(t *testing.T) {
	contract := &apicontract.Native{ContentType: "$param.type", Metadata: "$param.metadata"}
	headers := map[string]string{}
	got, err := nativeRequestHeaders(contract, map[string]string{"type": "text/plain; charset=utf-8", "metadata": `{"colour":"blue"}`}, headers)
	if err != nil || got != "text/plain; charset=utf-8" || headers["x-amz-meta-colour"] != "blue" {
		t.Fatal(got, headers, err)
	}
	for _, p := range []map[string]string{{"type": "bad/type\n"}, {"metadata": `{"a":1}`}} {
		if _, err := nativeRequestHeaders(contract, p, headers); err == nil {
			t.Fatal("invalid native header input accepted", p)
		}
	}
	op := plugin.Operation{Native: contract, Method: "GET", Path: "/", ContentType: "application/octet-stream"}
	bundle := plugin.Bundle{APIs: plugin.APIs{Operations: map[string]plugin.Operation{"native": op}}}
	get := func(k string) string {
		return map[string]string{"CTYUN_STORAGE_AK": "a", "CTYUN_STORAGE_SK": "s", "CTYUN_STORAGE_REGION": "cn", "CTYUN_STORAGE_ENDPOINT": "https://storage.example"}[k]
	}
	for _, value := range []string{"text/plain", "bad/type\n"} {
		spec, err := buildAPIRequest(bundle, plugin.Command{Operation: "native"}, nil, map[string]string{"type": value}, config.Profile{}, get, nil, io.Discard, nil, "en-US")
		if strings.Contains(value, "\n") {
			if err == nil {
				t.Fatal("bad type accepted")
			}
		} else if err != nil || spec.ContentType != value {
			t.Fatal(spec, err)
		}
	}
}
