/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package jsonvalue

import (
	"encoding/json"
	"testing"
)

// TestExactNumbers exercises shared decoding and canonicalization at precision
// boundaries and for equivalent integral decimal and exponent representations.
func TestExactNumbers(t *testing.T) {
	for raw, want := range map[string]string{"9007199254740993": "9007199254740993", "9007199254740993.0": "9007199254740993", "8e2": "800", "800.0": "800", "-0": "0", "1.25": "1.25"} {
		decoded, err := Decode([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if got := NumberText(decoded.(json.Number)); got != want {
			t.Fatalf("%s: %s", raw, got)
		}
	}
	for _, raw := range []string{"{", "{} {}", "{} bad"} {
		if _, err := Decode([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

// TestEqualUsesExactNumericValues distinguishes precise resource identities
// while preserving JSON numeric equivalence recursively through containers.
func TestEqualUsesExactNumericValues(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		want        bool
	}{
		{`{"a":[1,1.25,9007199254740993]}`, `{"a":[1e0,1.250,9007199254740993.0]}`, true},
		{`9007199254740992`, `9007199254740993`, false},
		{`{"a":null}`, `{"b":null}`, false},
		{`[1]`, `[1,2]`, false},
		{`1`, `"1"`, false},
		{`null`, `null`, true},
		{`[1]`, `{"a":1}`, false},
		{`{"a":1}`, `[1]`, false},
		{`[1]`, `[2]`, false},
	} {
		left, err := Decode([]byte(tc.left))
		if err != nil {
			t.Fatal(err)
		}
		right, err := Decode([]byte(tc.right))
		if err != nil {
			t.Fatal(err)
		}
		if got := Equal(left, right); got != tc.want {
			t.Fatalf("%s vs %s: %t", tc.left, tc.right, got)
		}
	}
}

// TestInvalidNumberFallback retains literal comparison for callers constructing
// json.Number directly; decoded responses cannot contain these invalid tokens.
func TestInvalidNumberFallback(t *testing.T) {
	if !Equal(json.Number("invalid"), json.Number("invalid")) || Equal(json.Number("invalid"), json.Number("other")) {
		t.Fatal("invalid number fallback changed")
	}
	if NumberText(json.Number("invalid")) != "invalid" {
		t.Fatal("invalid number spelling changed")
	}
}
