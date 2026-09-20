/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package signing

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/config"
)

// TestNativeV4Reference checks the published CTyun GET signature, using public example keys.
func TestNativeV4Reference(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example-bucket.oos-cn.ctyunapi.cn/test.txt", nil)
	req.Header.Set("Range", "bytes=0-9")
	now, _ := time.Parse("20060102T150405Z", "20190220T060724Z")
	err := SignNative(req, NativeOptions{Version: "v4", Region: "cn", Service: "s3", PayloadHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", Now: now}, config.Credentials{AccessKey: "2a948fd3f00ba0925806", SecretKey: "ef2017c2e5ffa0b1761717ecbca021da16501384"})
	want := "AWS4-HMAC-SHA256 Credential=2a948fd3f00ba0925806/20190220/cn/s3/aws4_request, SignedHeaders=host;range;x-amz-content-sha256;x-amz-date, Signature=dcefeb864c1ffad98f8f0307af32ceb584b38dc2a9c7a65459363cdb03fc6f12"
	if err != nil || req.Header.Get("Authorization") != want {
		t.Fatalf("signature: %s; %v", req.Header.Get("Authorization"), err)
	}
}

// TestNativeV2Reference checks the published CTyun GET signature and virtual-host resource.
func TestNativeV2Reference(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example-bucket.oos-cn.ctyunapi.cn/photos/puppy.jpg", nil)
	req.Header.Set("Content-Type", "application/octet-stream")
	now, _ := time.Parse(http.TimeFormat, "Tue, 11 Jun 2024 01:32:55 GMT")
	err := SignNative(req, NativeOptions{Version: "v2", Resource: "/example-bucket/photos/puppy.jpg", Now: now}, config.Credentials{AccessKey: "3a7451ae6b635b4f5ded", SecretKey: "c458417af3507ca686128f54efb3a00d5ad7ff09"})
	if err != nil || req.Header.Get("Authorization") != "AWS 3a7451ae6b635b4f5ded:icJnqU3Zfm1sEOBCBwJPKymwWds=" {
		t.Fatalf("signature: %s; %v", req.Header.Get("Authorization"), err)
	}
}

// TestNativeCanonicalQuery preserves duplicate values and sorts after percent encoding.
func TestNativeCanonicalQuery(t *testing.T) {
	got, err := NativeQuery("z=+&a=%2B&a=+&acl&%C3%A9=1")
	if err != nil || got != "%C3%A9=1&a=%20&a=%2B&acl=&z=%20" {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := NativeQuery("bad=%XX"); err == nil {
		t.Fatal("invalid escape accepted")
	}
	if got := NativePath("/a//.././中 %+?#"); got != "/a//.././%E4%B8%AD%20%25%2B%3F%23" {
		t.Fatal(got)
	}
}

// TestNativeTokenAndScope verifies session token signing and explicit scope.
func TestNativeTokenAndScope(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://storage.example/a//../x%252F?acl", nil)
	req.Header.Set("X-Amz-Meta-Test", "  a   b\t c  ")
	req.Header.Add("X-Amz-Meta-Test", "d")
	req.Header.Set("User-Agent", "ctyun")
	err := SignNative(req, NativeOptions{Version: "v4", Region: "cn-mg", Service: "sts", PayloadHash: strings.Repeat("0", 64), Token: "temporary", Now: time.Date(2026, 9, 20, 1, 2, 3, 0, time.UTC)}, config.Credentials{AccessKey: "ak", SecretKey: "sk"})
	if err != nil {
		t.Fatal(err)
	}
	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "/cn-mg/sts/") || !strings.Contains(auth, "x-amz-security-token") || strings.Contains(auth, "user-agent") || req.Header.Get("X-Amz-Security-Token") != "temporary" {
		t.Fatal(auth)
	}
	if req.URL.EscapedPath() != "/a//../x%252F" || req.URL.RawQuery != "acl=" {
		t.Fatal(req.URL)
	}
}

// TestNativeSigningRejectsIncompleteContracts prevents unsigned or ambiguous native requests.
func TestNativeSigningRejectsIncompleteContracts(t *testing.T) {
	valid := NativeOptions{Version: "v4", Region: "cn", Service: "s3", PayloadHash: strings.Repeat("0", 64), Now: time.Now()}
	creds := config.Credentials{AccessKey: "ak", SecretKey: "sk"}
	for _, change := range []func(*NativeOptions){func(o *NativeOptions) { o.Version = "bad" }, func(o *NativeOptions) { o.Now = time.Time{} }, func(o *NativeOptions) { o.Region = "" }, func(o *NativeOptions) { o.PayloadHash = "invalid" }, func(o *NativeOptions) { o.Version = "v2" }} {
		req, _ := http.NewRequest("GET", "https://storage.example", nil)
		opts := valid
		change(&opts)
		if err := SignNative(req, opts, creds); err == nil {
			t.Fatal("invalid contract accepted", opts)
		}
	}
	req, _ := http.NewRequest("GET", "https://storage.example", nil)
	if err := SignNative(req, valid, config.Credentials{}); err == nil {
		t.Fatal("missing credentials accepted")
	}
	req.URL.RawQuery = "bad=%ZZ"
	if err := SignNative(req, valid, creds); err == nil {
		t.Fatal("malformed query accepted")
	}
	req.URL.RawQuery = ""
	req.Host = ""
	if err := SignNative(req, valid, creds); err != nil {
		t.Fatal(err)
	}
}

// TestNativeV2CanonicalResource tests sorted subresources, unescaped overrides and AMZ headers.
func TestNativeV2CanonicalResource(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://storage.example/bucket/key?prefix=ignored&acl&uploadId=a%2Bb", nil)
	req.Header.Set("X-Amz-Date", "Mon, 21 Sep 2026 00:00:00 GMT")
	req.Header.Set("X-Amz-Meta-Example", " a  b ")
	options := NativeOptions{Version: "v2", Resource: "/bucket/key", QueryKeys: []string{"uploadId", "acl"}, Now: time.Now()}
	if err := SignNative(req, options, config.Credentials{AccessKey: "ak", SecretKey: "sk"}); err != nil {
		t.Fatal(err)
	}
	canonical := "GET\n\n\n\nx-amz-date:Mon, 21 Sep 2026 00:00:00 GMT\nx-amz-meta-example:a b\n/bucket/key?acl&uploadId=a+b"
	if want := "AWS ak:" + SignPOSTPolicyV2(canonical, "sk"); req.Header.Get("Authorization") != want {
		t.Fatal(req.Header)
	}
}

// TestNativeQueryPrefixNames sorts encoded names before their values or equals separators.
func TestNativeQueryPrefixNames(t *testing.T) {
	got, err := NativeQuery("a-b=2&a=1&a=0")
	if err != nil || got != "a=0&a=1&a-b=2" {
		t.Fatal(got, err)
	}
}

// TestNativeV2QueryPrefixNames keeps V2 resource parameters in name order as well.
func TestNativeV2QueryPrefixNames(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://storage.example/?a-b=2&a=1", nil)
	now := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	err := SignNative(req, NativeOptions{Version: "v2", Resource: "/", QueryKeys: []string{"a-b", "a"}, Now: now}, config.Credentials{AccessKey: "ak", SecretKey: "sk"})
	want := "AWS ak:" + SignPOSTPolicyV2("GET\n\n\nMon, 21 Sep 2026 00:00:00 GMT\n/?a=1&a-b=2", "sk")
	if err != nil || req.Header.Get("Authorization") != want {
		t.Fatal(req.Header, err)
	}
}

// TestNativeV2EscapedObject signs the escaped resource used by CTyun's V2 Java example.
func TestNativeV2EscapedObject(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://bucket.storage.example/a//../%E4%B8%AD%20%25%2B%3F", nil)
	now := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	err := SignNative(req, NativeOptions{Version: "v2", Resource: "/bucket/a//../中 %+?", Now: now}, config.Credentials{AccessKey: "ak", SecretKey: "sk"})
	canonical := "GET\n\n\nMon, 21 Sep 2026 00:00:00 GMT\n/bucket/a//../%E4%B8%AD%20%25%2B%3F"
	if want := "AWS ak:" + SignPOSTPolicyV2(canonical, "sk"); err != nil || req.Header.Get("Authorization") != want {
		t.Fatal(req.Header, err)
	}
}
