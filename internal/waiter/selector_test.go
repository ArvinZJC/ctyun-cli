/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package waiter

import (
	"encoding/json"
	"testing"
)

// TestSelectedResourceWaitsForExactIdentity prevents unrelated or duplicated
// collection rows from satisfying a requested resource's lifecycle wait.
func TestSelectedResourceWaitsForExactIdentity(t *testing.T) {
	var spec Spec
	if err := json.Unmarshal([]byte(`{"Path":"status","Success":"ready","Failure":"error","Selector":{"Path":"result.items","Key":"id","Value":"target"}}`), &spec); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, payload string
		want          State
		bad           bool
	}{
		{"not first", `{"result":{"items":[{"id":"other","status":"error"},{"id":"target","status":"ready"}]}}`, Success, false},
		{"wrong resource", `{"result":{"items":[{"id":"other","status":"ready"}]}}`, Pending, false},
		{"empty", `{"result":{"items":[]}}`, Pending, false},
		{"null state", `{"result":{"items":[{"id":"target","status":null}]}}`, Pending, false},
		{"pending", `{"result":{"items":[{"id":"target","status":"creating"}]}}`, Pending, false},
		{"failed", `{"result":{"items":[{"id":"target","status":"error"}]}}`, Failure, false},
		{"duplicate", `{"result":{"items":[{"id":"target","status":"ready"},{"id":"target","status":"creating"}]}}`, Pending, true},
		{"wrong collection", `{"result":{"items":{}}}`, Pending, true},
		{"missing collection", `{"result":{}}`, Pending, true},
		{"invalid identity", `{"result":{"items":[{"id":{},"status":"ready"}]}}`, Pending, true},
		{"missing identity", `{"result":{"items":[{"status":"ready"}]}}`, Pending, true},
		{"missing state", `{"result":{"items":[{"id":"target"}]}}`, Pending, true},
		{"object state", `{"result":{"items":[{"id":"target","status":{}}]}}`, Pending, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var payload map[string]any
			if err := json.Unmarshal([]byte(tc.payload), &payload); err != nil {
				t.Fatal(err)
			}
			state, err := Evaluate(spec, payload)
			if (err != nil) != tc.bad || state != tc.want {
				t.Fatalf("got %s, %v; want %s, error=%t", state, err, tc.want, tc.bad)
			}
		})
	}
}

// TestSelectorExampleValidation refuses examples that cannot establish the
// declared collection shape, identity and scalar state.
func TestSelectorExampleValidation(t *testing.T) {
	spec := Spec{Path: "status", Selector: &Selector{Path: "items", Key: "id", Value: "$param.id"}}
	for _, tc := range []struct {
		payload string
		bad     bool
	}{
		{`{"items":[{"id":"a","status":true},{"id":"b","status":null}]}`, false},
		{`{"items":[]}`, true}, {`{}`, true}, {`{"items":{}}`, true},
		{`{"items":[{"status":"ready"}]}`, true},
		{`{"items":[{"id":{},"status":"ready"}]}`, true},
		{`{"items":[{"id":"a","status":[]}]}`, true},
		{`{"items":[{"id":"a"}]}`, true},
		{`{"items":[{"id":"a","status":"ready"},{"id":"a","status":"ready"}]}`, true},
	} {
		var payload map[string]any
		if err := json.Unmarshal([]byte(tc.payload), &payload); err != nil {
			t.Fatal(err)
		}
		if err := ValidateExample(spec, payload); (err != nil) != tc.bad {
			t.Errorf("%s: %v", tc.payload, err)
		}
	}
	if err := ValidateExample(Spec{Path: "state"}, map[string]any{"state": 1}); err != nil {
		t.Fatal(err)
	}
}

// TestNumericSelectorIdentity retains plain decimal identities regardless of
// the decoder used by a caller and never rounds json.Number integers.
func TestNumericSelectorIdentity(t *testing.T) {
	for _, value := range []any{float64(1000000), json.Number("9007199254740993")} {
		want := "1000000"
		if number, ok := value.(json.Number); ok {
			want = number.String()
		}
		state, err := Evaluate(Spec{Selector: &Selector{Path: "items", Key: "id", Value: want}, Path: "state", Success: "ready"}, map[string]any{"items": []any{map[string]any{"id": value, "state": "ready"}}})
		if err != nil || state != Success {
			t.Fatalf("%v: %s %v", value, state, err)
		}
	}
}

// TestNumericResponseSpellings preserves numeric state and identity semantics
// when response decoding retains the original JSON number text.
func TestNumericResponseSpellings(t *testing.T) {
	for _, spelling := range []string{"1", "1.0", "1e0"} {
		payload := map[string]any{"items": []any{map[string]any{"id": json.Number("9007199254740993.0"), "state": json.Number(spelling)}}}
		state, err := Evaluate(Spec{Selector: &Selector{Path: "items", Key: "id", Value: "9007199254740993"}, Path: "state", Success: "1"}, payload)
		if err != nil || state != Success {
			t.Fatalf("%s: %s %v", spelling, state, err)
		}
	}
}
