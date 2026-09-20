/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package client

import (
	"github.com/ArvinZJC/ctyun-cli/internal/config"
	"strings"
	"testing"
	"time"
)

// TestNativeRequestRouting verifies exact object identity and isolated authorization.
func TestNativeRequestRouting(t *testing.T) {
	for _, style := range []string{"path", "virtual"} {
		spec := RequestSpec{Method: "GET", BaseURL: "https://storage.example:8443", Path: "/", Credentials: config.Credentials{AccessKey: "native-ak", SecretKey: "native-sk"}, Now: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), Native: &NativeRequest{Version: "v4", Region: "cn", Service: "s3", Addressing: style, Bucket: "test-bucket", Object: "/a//../中 %+?", Token: "temporary"}}
		req, err := BuildRequest(spec)
		if err != nil {
			t.Fatal(err)
		}
		want := "//a//../%E4%B8%AD%20%25%2B%3F"
		host := "test-bucket.storage.example:8443"
		if style == "path" {
			want = "/test-bucket" + want
			host = "storage.example:8443"
		}
		if req.URL.EscapedPath() != want || req.URL.Host != host {
			t.Fatalf("%s", req.URL)
		}
		if req.Header.Get("Eop-date") != "" || req.Header.Get("ctyun-eop-request-id") != "" || req.Header.Get("ctyun-eop-ak") != "" || !strings.HasPrefix(req.Header.Get("Authorization"), "AWS4-HMAC-SHA256") {
			t.Fatal(req.Header)
		}
	}
}

// TestNativeEndpointRejection rejects endpoint paths and ambiguous host inputs before signing.
func TestNativeEndpointRejection(t *testing.T) {
	for _, endpoint := range []string{"http://storage.example", "https://u:p@storage.example", "https://storage.example/path", "https://storage.example?x=y", "https://storage.example#fragment"} {
		_, err := BuildRequest(RequestSpec{BaseURL: endpoint, Path: "/", Native: &NativeRequest{Addressing: "path"}})
		if err == nil {
			t.Fatalf("accepted %s", endpoint)
		}
	}
}

// TestNativeRequestFailuresAndBodies checks body ownership, service roots and V2 resources.
func TestNativeRequestFailuresAndBodies(t *testing.T) {
	for _, native := range []NativeRequest{{Addressing: "wrong"}, {Addressing: "path", Bucket: "bad/bucket"}, {Addressing: "path", Object: "orphan"}, {Addressing: "virtual", Bucket: "test-bucket"}} {
		endpoint := "https://storage.example"
		if native.Addressing == "virtual" {
			endpoint = "https://127.0.0.1"
		}
		if _, _, err := nativeURL(endpoint, native); err == nil {
			t.Fatal("invalid route accepted", native)
		}
	}
	native := &NativeRequest{Version: "v2", Addressing: "path"}
	spec := RequestSpec{Method: "GET", BaseURL: "https://storage.example", Path: "/", Native: native, Credentials: config.Credentials{AccessKey: "ak", SecretKey: "sk"}}
	if req, err := BuildRequest(spec); err != nil || req.URL.Path != "/" {
		t.Fatal(req, err)
	}
	body, err := PrepareBody(BodyInput{Encoding: "json", Fields: map[string]any{"value": "payload"}})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	spec.PreparedBody = body
	spec.Method = "PUT"
	native.Version = "v4"
	native.Region = "cn"
	native.Service = "s3"
	req, err := BuildRequest(spec)
	if err != nil {
		t.Fatal(err)
	}
	req.Body.Close()
	if req.Header.Get("X-Amz-Content-Sha256") != body.SHA256 {
		t.Fatal(req.Header)
	}
	native.Version = "invalid"
	if _, err := BuildRequest(spec); err == nil {
		t.Fatal("invalid signing accepted")
	}
	native.Version = "post-policy"
	if _, err := BuildRequest(spec); err == nil {
		t.Fatal("non-policy body accepted")
	}
	form, err := PrepareBody(BodyInput{Encoding: "multipart", Parts: []BodyPart{{Name: "policy", Value: "policy-data"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer form.Close()
	spec.PreparedBody = form
	spec.Method = "POST"
	req, err = BuildRequest(spec)
	if err != nil {
		t.Fatal(err)
	}
	req.Body.Close()
	if req.Header.Get("Authorization") != "" {
		t.Fatal("policy POST acquired header authorization")
	}
	native.Token = "temporary-secret"
	if got := RedactHTTPDetails("token temporary-secret", spec.Credentials, "", spec.sensitiveValues()...); strings.Contains(got, "temporary-secret") {
		t.Fatal(got)
	}
}

// TestNativeV2AuthorizationRedaction removes complete signatures from echoed HTTP diagnostics.
func TestNativeV2AuthorizationRedaction(t *testing.T) {
	got := RedactHTTPDetails("Authorization: AWS native-ak:abc/def+123=", config.Credentials{}, "")
	if strings.Contains(got, "native-ak") || strings.Contains(got, "abc/def") {
		t.Fatal(got)
	}
}
