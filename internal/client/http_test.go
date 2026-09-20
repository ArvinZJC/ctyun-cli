/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestHTTPEmptyAndNestedSuccess verifies explicit representations and nested errors.
func TestHTTPEmptyAndNestedSuccess(t *testing.T) {
	for _, tc := range []struct {
		status       int
		format, body string
		checks       []apicontract.Check
		success      bool
	}{
		{204, "empty", "", nil, true}, {200, "empty", "", nil, true}, {200, "json", "", nil, false},
		{200, "json", `{"statusCode":0,"returnObj":{"code":1}}`, []apicontract.Check{{Path: "returnObj.code", Values: []string{"0"}}}, false},
		{200, "json", `{"statusCode":0}`, []apicontract.Check{{Path: "statusCode", Values: []string{"0"}}}, true},
		{201, "xml", `<r><v>001</v><v>2</v></r>`, nil, true},
	} {
		spec := RequestSpec{Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: tc.status, Format: tc.format}}, Success: tc.checks}}
		response := &HTTPResponse{Status: tc.status, Headers: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body))}
		_, err := DecodeHTTPResponse(response, spec)
		if (err == nil) != tc.success {
			t.Errorf("%+v: %v", tc, err)
		}
	}
}
