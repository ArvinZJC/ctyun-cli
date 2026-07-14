/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestParseParameterValue verifies every supported metadata value shape and
// keeps JSON numbers lossless until request marshaling.
func TestParseParameterValue(t *testing.T) {
	tests := []struct {
		name      string
		valueType ParameterValueType
		raw       string
		want      any
		wantError bool
	}{
		{name: "omitted string", raw: "demo", want: "demo"},
		{name: "explicit string", valueType: ParameterValueString, raw: "demo", want: "demo"},
		{name: "integer", valueType: ParameterValueInteger, raw: "1667486264", want: json.Number("1667486264")},
		{name: "large integer", valueType: ParameterValueInteger, raw: "9223372036854775808", want: json.Number("9223372036854775808")},
		{name: "integer rejects fraction", valueType: ParameterValueInteger, raw: "1.5", wantError: true},
		{name: "integer rejects leading zero", valueType: ParameterValueInteger, raw: "01", wantError: true},
		{name: "number", valueType: ParameterValueNumber, raw: "1.25e3", want: json.Number("1.25e3")},
		{name: "number rejects boolean", valueType: ParameterValueNumber, raw: "true", wantError: true},
		{name: "number rejects NaN", valueType: ParameterValueNumber, raw: "NaN", wantError: true},
		{name: "number rejects infinity", valueType: ParameterValueNumber, raw: "1e9999", wantError: true},
		{name: "boolean true", valueType: ParameterValueBoolean, raw: "true", want: true},
		{name: "boolean false", valueType: ParameterValueBoolean, raw: "false", want: false},
		{name: "boolean rejects shorthand", valueType: ParameterValueBoolean, raw: "1", wantError: true},
		{name: "boolean rejects uppercase", valueType: ParameterValueBoolean, raw: "TRUE", wantError: true},
		{name: "string array", valueType: ParameterValueStringArray, raw: `["a","b"]`, want: []any{"a", "b"}},
		{name: "string array rejects number", valueType: ParameterValueStringArray, raw: `["a",1]`, wantError: true},
		{name: "object array", valueType: ParameterValueObjectArray, raw: `[{"id":1}]`, want: []any{map[string]any{"id": json.Number("1")}}},
		{name: "object array rejects scalar", valueType: ParameterValueObjectArray, raw: `[1]`, wantError: true},
		{name: "string map", valueType: ParameterValueStringMap, raw: `{"a":"b"}`, want: map[string]any{"a": "b"}},
		{name: "string map rejects number", valueType: ParameterValueStringMap, raw: `{"a":1}`, wantError: true},
		{name: "arbitrary json", valueType: ParameterValueJSON, raw: `{"nested":[true,2]}`, want: map[string]any{"nested": []any{true, json.Number("2")}}},
		{name: "json rejects trailing value", valueType: ParameterValueJSON, raw: `{} {}`, wantError: true},
		{name: "json rejects invalid trailing token", valueType: ParameterValueJSON, raw: `{} x`, wantError: true},
		{name: "unsupported type", valueType: "binary", raw: `"value"`, wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseParameterValue(Parameter{ValueType: tc.valueType}, tc.raw)
			if tc.wantError {
				if err == nil {
					t.Fatalf("ParseParameterValue(%q, %q) returned nil error", tc.valueType, tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseParameterValue(%q, %q) = %#v, want %#v", tc.valueType, tc.raw, got, tc.want)
			}
		})
	}
}
