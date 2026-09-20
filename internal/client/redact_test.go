/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"github.com/ArvinZJC/ctyun-cli/internal/config"
	"strings"
	"testing"
)

// TestRedactionCoversStorageRepresentations prevents credential leakage across body formats.
func TestRedactionCoversStorageRepresentations(t *testing.T) {
	for _, input := range []string{
		`Signature=hidden&Signature=another`,
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
