/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/config"
)

// TestRedactionCoversStorageRepresentations prevents credential leakage across body formats.
func TestRedactionCoversStorageRepresentations(t *testing.T) {
	for _, input := range []string{
		`Signature=hidden&Signature=another`,
		`{"x-amz-security-token":"hidden","policy":"another"}`,
		`<x-amz-security-token>hidden</x-amz-security-token><x-amz-signature>another</x-amz-signature>`,
		`<Base32StringSeed>hidden</Base32StringSeed><QRCodePNG>another</QRCodePNG>`,
		`{"SecretAccessKey":"hidden","AccessKeyId":"another","plain":"visible"}`,
		`<r><x:SecretAccessKey>hidden</x:SecretAccessKey><Signature>another</Signature></r>`,
		`https://user:hidden@example.com/a?AWSAccessKeyId=another&key=visible`,
		`secret_key=hidden&session_token=another`,
	} {
		got := RedactHTTPDetails(input, config.Credentials{}, "")
		if strings.Contains(got, "hidden") || strings.Contains(got, "another") {
			t.Fatalf("leaked %s", got)
		}
	}
}

// TestMultipartCredentialsAreRedactedFromFreeTextErrors retains known values across request dispatch.
func TestMultipartCredentialsAreRedactedFromFreeTextErrors(t *testing.T) {
	body, err := PrepareBody(BodyInput{Encoding: "multipart", Parts: []BodyPart{{Name: "x-amz-security-token", Value: "storage-temporary-token"}, {Name: "key", Value: "visible-object"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	var debug bytes.Buffer
	spec := RequestSpec{Method: "POST", BaseURL: "https://example.test", Path: "/upload", PreparedBody: body, Debug: &debug, Credentials: config.Credentials{AccessKey: "gateway-ak", SecretKey: "gateway-sk"}}
	_, err = Do(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if closeErr := request.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader(`{"message":"storage-temporary-token","x-amz-security-token":"storage-temporary-token","key":"visible-object"}`))}, nil
	}), spec)
	if err == nil || strings.Contains(err.Error(), "storage-temporary-token") || strings.Contains(debug.String(), "storage-temporary-token") || !strings.Contains(debug.String(), "visible-object") {
		t.Fatal(err, debug.String())
	}
}
