/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package signing

import (
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/config"
)

func TestGenerateEOPAuthorizationMatchesSDKShape(t *testing.T) {
	creds := config.Credentials{AccessKey: "test-ak", SecretKey: "test-sk"}

	got := GenerateEOPAuthorization(EOPRequest{
		Query:     "a=1&b=two",
		Body:      []byte(`{"regionID":"cn-huadong1"}`),
		Date:      "20260613T010203Z",
		RequestID: "request-123",
	}, creds)

	want := "test-ak Headers=ctyun-eop-request-id;eop-date Signature=fCBi1tDyjHY9xVs27pEM/UhWkCf8ueyeiHa77Dy8D58="
	if got != want {
		t.Fatalf("GenerateEOPAuthorization() = %q, want %q", got, want)
	}
}

func TestGenerateEOPAuthorizationReturnsEmptyWithoutCredentials(t *testing.T) {
	got := GenerateEOPAuthorization(EOPRequest{
		Date:      "20260613T010203Z",
		RequestID: "request-123",
	}, config.Credentials{})

	if got != "" {
		t.Fatalf("GenerateEOPAuthorization() = %q, want empty string", got)
	}
}

func TestRedactSecretsMasksCredentialMaterial(t *testing.T) {
	redacted := RedactSecrets("ak=test-ak sk=test-sk Signature=abc RequestID=req-1", []string{"test-ak", "test-sk", "abc"})

	for _, secret := range []string{"test-ak", "test-sk", "abc"} {
		if strings.Contains(redacted, secret) {
			t.Fatalf("redacted output still contains secret %q: %s", secret, redacted)
		}
	}
}

func TestRedactSecretsIgnoresEmptySecrets(t *testing.T) {
	got := RedactSecrets("keep this text", []string{"", "text"})
	if got != "keep this [REDACTED]" {
		t.Fatalf("RedactSecrets() = %q", got)
	}
}
