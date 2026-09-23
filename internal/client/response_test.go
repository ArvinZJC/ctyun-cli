/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"encoding/json"
	"testing"
)

// TestDecodeResponsePreservesResourceIdentities protects exact waiter matching
// and JSON output from scientific notation and integer rounding.
func TestDecodeResponsePreservesResourceIdentities(t *testing.T) {
	payload, err := DecodeResponse([]byte(`{"small":1000000,"large":9007199254740993}`))
	if err != nil {
		t.Fatal(err)
	}
	if payload["small"] != json.Number("1000000") || payload["large"] != json.Number("9007199254740993") {
		t.Fatal(payload)
	}
	for _, data := range []string{`{`, `[]`, `{} {}`, `{} broken`} {
		if _, err := DecodeResponse([]byte(data)); err == nil {
			t.Errorf("accepted %s", data)
		}
	}
}

// TestStatusCodeScalarForms retains compatibility for direct in-memory callers
// as well as losslessly decoded API responses.
func TestStatusCodeScalarForms(t *testing.T) {
	for _, value := range []any{float64(800), "800", json.Number("800"), json.Number("800.0"), json.Number("8e2")} {
		if got := ctyunStatusCode(value); got != "800" {
			t.Fatalf("%v: %s", value, got)
		}
	}
}

// TestNullResponsePreservesEmptyPayload retains the response decoder's legacy
// null behavior instead of turning it into a malformed-object diagnostic.
func TestNullResponsePreservesEmptyPayload(t *testing.T) {
	payload, err := DecodeResponse([]byte(`null`))
	if err != nil || payload != nil {
		t.Fatalf("payload=%v error=%v", payload, err)
	}
}
