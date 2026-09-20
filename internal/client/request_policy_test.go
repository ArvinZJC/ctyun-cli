/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// TestPreparedRequestGuards protects signing-owned headers and the unique body source.
func TestPreparedRequestGuards(t *testing.T) {
	body, err := PrepareBody(BodyInput{Encoding: "json"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	for _, headers := range []map[string]string{{"bad name": "x"}, {"X-Test": "bad\nvalue"}, {"Host": "evil"}, {"Content-Length": "1"}, {"Eop-Date": "wrong"}, {"Content-Type": "wrong"}} {
		if _, err := BuildRequest(RequestSpec{BaseURL: "https://example.test", PreparedBody: body, Headers: headers}); err == nil {
			t.Fatal(headers)
		}
	}
	if _, err := BuildRequest(RequestSpec{BaseURL: "https://example.test", PreparedBody: body, Body: []byte("other")}); err == nil {
		t.Fatal("two body sources accepted")
	}
	if _, err := BuildRequest(RequestSpec{BaseURL: ":bad", PreparedBody: body}); err == nil {
		t.Fatal("invalid URL accepted")
	}
	req, err := BuildRequest(RequestSpec{BaseURL: "https://example.test", PreparedBody: body})
	if err != nil {
		t.Fatal(err)
	}
	if req.Body != http.NoBody || req.ContentLength != 0 {
		t.Fatal("empty prepared body not marked empty")
	}
	body.Close()
	if _, err := BuildRequest(RequestSpec{BaseURL: "https://example.test", PreparedBody: body}); err == nil {
		t.Fatal("closed body accepted")
	}
}

// TestHTTPFailuresReleaseResponses rejects malformed transports and keeps failure cleanup explicit.
func TestHTTPFailuresReleaseResponses(t *testing.T) {
	for _, tc := range []struct {
		response *http.Response
		err      error
		debug    io.Writer
	}{
		{nil, nil, nil}, {&http.Response{StatusCode: 200}, nil, nil},
		{&http.Response{Body: &failingBody{reader: strings.NewReader("")}}, io.ErrUnexpectedEOF, nil},
		{&http.Response{StatusCode: 400, Body: &failingBody{readErr: io.ErrUnexpectedEOF}}, nil, nil},
		{&http.Response{StatusCode: 400, Body: &failingBody{reader: strings.NewReader("bad"), closeErr: io.ErrClosedPipe}}, nil, nil},
		{&http.Response{StatusCode: 400, Body: &failingBody{reader: strings.NewReader("bad")}}, nil, &failingDebugWriter{failOn: 3}},
	} {
		_, err := Do(roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Body != nil {
				req.Body.Close()
			}
			return tc.response, tc.err
		}), RequestSpec{BaseURL: "https://example.test", Debug: tc.debug})
		if err == nil {
			t.Fatal("failure accepted")
		}
		if tc.response != nil && tc.response.Body != nil {
			if b, ok := tc.response.Body.(*failingBody); ok && b.closes != 1 {
				t.Fatalf("closes %d", b.closes)
			}
		}
	}
	if _, err := Do(nil, RequestSpec{Response: &apicontract.Response{}}); err == nil {
		t.Fatal("invalid contract accepted")
	}
	var called bool
	_, err := Do(roundTripFunc(func(req *http.Request) (*http.Response, error) { called = true; return nil, errors.New("unexpected") }), RequestSpec{BaseURL: "https://example.test", Debug: &failingDebugWriter{failOn: 1}, Body: []byte("body")})
	if err == nil || called {
		t.Fatal("debug failure dispatched request")
	}
}
