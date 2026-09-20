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

// failingBody records closure while simulating transfer and close failures.
type failingBody struct {
	reader            io.Reader
	readErr, closeErr error
	closes            int
}

// Read yields the configured transfer failure or delegates to the source.
func (b *failingBody) Read(p []byte) (int, error) {
	if b.readErr != nil {
		return 0, b.readErr
	}
	return b.reader.Read(p)
}

// Close records ownership releases and returns the configured failure.
func (b *failingBody) Close() error { b.closes++; return b.closeErr }

// TestHTTPFixtureValidation rejects incomplete or unsafe envelopes and preserves exact bytes.
func TestHTTPFixtureValidation(t *testing.T) {
	for _, data := range []string{`{`, `{"unknown":1}`, `{"schema_version":1,"status":200} {}`, `{"schema_version":2,"status":200}`, `{"schema_version":1,"status":99}`, `{"schema_version":1,"status":600}`, `{"schema_version":1,"status":200,"headers":{"bad name":["x"]}}`, `{"schema_version":1,"status":200,"headers":{"X-Test":["bad\nvalue"]}}`, `{"schema_version":1,"status":200,"body_base64":"@@"}`} {
		if _, err := DecodeFixture([]byte(data)); err == nil {
			t.Errorf("accepted %s", data)
		}
	}
	response, err := DecodeFixture([]byte(`{"schema_version":1,"status":200,"headers":{"x-test":["one","two"]},"body_base64":"AP8K"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil || !bytes.Equal(data, []byte{0, 255, 10}) || len(response.Headers.Values("X-Test")) != 2 {
		t.Fatalf("%v %v %#v", data, err, response.Headers)
	}
}

// TestExplicitResponsePolicies covers errors, exact statuses, XML roots and header projections.
func TestExplicitResponsePolicies(t *testing.T) {
	for _, tc := range []struct {
		format, body string
		status       int
		success      bool
	}{
		{"text", "hello", 200, true}, {"empty", "", 200, true}, {"empty", "unexpected", 200, false}, {"binary", "data", 200, false}, {"unknown", "data", 200, false}, {"json", "{}", 201, false}, {"xml", "<wrong/>", 200, false}, {"xml", "<root/>", 200, true}, {"xml", "<broken", 200, false},
	} {
		t.Run(tc.format+tc.body, func(t *testing.T) {
			variant := apicontract.Variant{Status: 200, Format: tc.format}
			if tc.format == "xml" {
				variant.XMLRoot = &apicontract.XMLName{Local: "root"}
			}
			if tc.format == "empty" {
				variant.Headers = map[string]string{"single": "X-One", "many": "X-Many", "missing": "X-Missing"}
			}
			body := &failingBody{reader: strings.NewReader(tc.body)}
			result, err := DecodeHTTPResponse(&HTTPResponse{Status: tc.status, Body: body, Headers: http.Header{"X-One": {"1"}, "X-Many": {"a", "b"}}}, RequestSpec{Response: &apicontract.Response{Variants: []apicontract.Variant{variant}}})
			if (err == nil) != tc.success || body.closes != 1 {
				t.Fatalf("result %#v error %v closes %d", result, err, body.closes)
			}
			if tc.format == "empty" && tc.success {
				headers := result.Payload["headers"].(map[string]any)
				if headers["single"] != "1" || len(headers["many"].([]string)) != 2 {
					t.Fatal(headers)
				}
			}
		})
	}
	for _, response := range []*HTTPResponse{{Status: 400, Body: io.NopCloser(strings.NewReader(""))}, {Status: 200, Body: &failingBody{readErr: io.ErrUnexpectedEOF}}, {Status: 200, Body: &failingBody{reader: strings.NewReader(`{"statusCode":800}`), closeErr: io.ErrClosedPipe}}} {
		if _, err := DecodeHTTPResponse(response, RequestSpec{}); err == nil {
			t.Fatal("expected decode failure")
		}
	}
	if _, err := DecodeHTTPResponse(&HTTPResponse{Status: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", MaxStructuredBody+1)))}, RequestSpec{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "text"}}}}); err == nil {
		t.Fatal("oversized body accepted")
	}
}

// TestSuccessChecksPreserveScalarTypesAndPrecision rejects absent or composite result codes.
func TestSuccessChecksPreserveScalarTypesAndPrecision(t *testing.T) {
	for _, tc := range []struct {
		body, path string
		values     []string
		valid      bool
	}{
		{`{"code":0}`, "code", []string{"0.0"}, true}, {`{"code":9007199254740993}`, "code", []string{"9007199254740992"}, false}, {`{"code":true}`, "code", []string{"true"}, true}, {`{"code":"ok"}`, "code", []string{"ok"}, true}, {`{}`, "code", []string{"0"}, false}, {`{"code":null}`, "code", []string{"0"}, false}, {`{"code":[]}`, "code", []string{"0"}, false}, {`{"code":0}`, "code.inner", []string{"0"}, false},
	} {
		var payload map[string]any
		decoder := json.NewDecoder(strings.NewReader(tc.body))
		decoder.UseNumber()
		if err := decoder.Decode(&payload); err != nil {
			t.Fatal(err)
		}
		err := validateSuccess(payload, []apicontract.Check{{Path: tc.path, Values: tc.values}})
		if (err == nil) != tc.valid {
			t.Errorf("%s: %v", tc.body, err)
		}
	}
}

// TestBinaryCopyClosesOnEveryOutcome verifies exact output and propagation of transfer failures.
func TestBinaryCopyClosesOnEveryOutcome(t *testing.T) {
	for _, tc := range []struct {
		status            int
		format            string
		readErr, closeErr error
		valid             bool
	}{
		{200, "binary", nil, nil, true}, {201, "binary", nil, nil, false}, {200, "text", nil, nil, false}, {200, "binary", io.ErrUnexpectedEOF, nil, false}, {200, "binary", nil, io.ErrClosedPipe, false},
	} {
		body := &failingBody{reader: strings.NewReader("\x00\xffdata"), readErr: tc.readErr, closeErr: tc.closeErr}
		var dest bytes.Buffer
		_, err := CopyHTTPResponse(&dest, &HTTPResponse{Status: tc.status, Body: body}, RequestSpec{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: tc.format}}}})
		if (err == nil) != tc.valid || body.closes != 1 {
			t.Fatalf("%+v: %v, closes %d", tc, err, body.closes)
		}
		if tc.valid && dest.String() != "\x00\xffdata" {
			t.Fatal(dest.Bytes())
		}
	}
	body := &failingBody{reader: strings.NewReader("data")}
	spec := RequestSpec{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "binary"}}}, Debug: &failingDebugWriter{failOn: 1}}
	if _, err := CopyHTTPResponse(io.Discard, &HTTPResponse{Status: 200, Body: body}, spec); err == nil {
		t.Fatal(err)
	}
}

// TestPOSTRedirectNeedsLocation prevents success claims for incomplete redirect responses.
func TestPOSTRedirectNeedsLocation(t *testing.T) {
	response := &HTTPResponse{Status: 303, Headers: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}
	_, err := DecodeHTTPResponse(response, RequestSpec{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 303, Format: "empty"}}}})
	if err == nil {
		t.Fatal("missing redirect target accepted")
	}
}
