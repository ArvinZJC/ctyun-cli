/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// TestDeclaredResponseMediaType rejects error envelopes before any download bytes become visible.
func TestDeclaredResponseMediaType(t *testing.T) {
	var contract apicontract.Response
	if err := json.Unmarshal([]byte(`{"variants":[{"status":200,"format":"binary","media_type":"application/octet-stream"}]}`), &contract); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		headers []string
		valid   bool
	}{
		{"binary", []string{"application/octet-stream"}, true},
		{"parameters", []string{"Application/Octet-Stream; charset=binary"}, true},
		{"JSON business error", []string{"application/json"}, false},
		{"missing", nil, false},
		{"malformed", []string{"invalid"}, false},
		{"duplicate", []string{"application/octet-stream", "application/json"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &mediaTypeBody{Reader: strings.NewReader(`{"statusCode":500,"message":"failure"}`)}
			response := &HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": tc.headers}, Body: body}
			var out bytes.Buffer
			n, err := CopyHTTPResponse(&out, response, RequestSpec{Response: &contract})
			if tc.valid && (err != nil || n == 0) {
				t.Fatalf("valid media type: bytes=%d err=%v", n, err)
			}
			if !tc.valid && (err == nil || n != 0 || out.Len() != 0 || body.read) {
				t.Fatalf("invalid media type exposed body: bytes=%d err=%v read=%v", n, err, body.read)
			}
			if !body.closed {
				t.Fatal("response body not closed")
			}
		})
	}
}

// mediaTypeBody tracks response ownership and whether rejection precedes reading.
type mediaTypeBody struct {
	io.Reader
	read, closed bool
}

// Read records attempted consumption of response bytes.
func (b *mediaTypeBody) Read(p []byte) (int, error) { b.read = true; return b.Reader.Read(p) }

// Close records release of the response body.
func (b *mediaTypeBody) Close() error { b.closed = true; return nil }
