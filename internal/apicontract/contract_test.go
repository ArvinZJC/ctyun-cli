/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package apicontract

import "testing"

// TestContractValidation checks HTTP and representation invariants before execution.
func TestContractValidation(t *testing.T) {
	for _, tc := range []struct {
		method, encoding, format string
		status                   int
		valid                    bool
	}{
		{"HEAD", "", "json", 200, false}, {"HEAD", "", "empty", 200, true},
		{"DELETE", "", "empty", 204, true}, {"GET", "", "json", 204, false},
		{"GET", "", "empty", 304, true}, {"GET", "", "json", 412, false},
		{"POST", "form", "json", 200, true}, {"OPTIONS", "", "empty", 200, true},
		{"POST", "unknown", "json", 200, false},
	} {
		var request *Request
		if tc.encoding != "" {
			request = &Request{Encoding: tc.encoding}
		}
		err := Validate(tc.method, "", request, &Response{Variants: []Variant{{Status: tc.status, Format: tc.format}}})
		if (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc, err)
		}
	}
}
