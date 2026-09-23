/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package jsonvalue preserves exact JSON numbers across response and catalog consumers.
package jsonvalue

import (
	"bytes"
	"encoding/json"
	"io"
	"math/big"
	"reflect"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// Decode reads exactly one JSON value without rounding numeric identities.
func Decode(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, diagnostic.New("error.expected_single_json_value")
	}
	return value, nil
}

// NumberText canonicalizes integral decimal and exponent spellings without a
// float64 conversion. Fractional values retain their original JSON spelling.
func NumberText(number json.Number) string {
	value, ok := new(big.Rat).SetString(number.String())
	if ok && value.IsInt() {
		return value.Num().String()
	}
	return number.String()
}

// Equal compares decoded JSON trees by exact numeric value, retaining semantic
// equality across decimal and exponent spellings without rounding identities.
func Equal(left, right any) bool {
	switch value := left.(type) {
	case json.Number:
		other, ok := right.(json.Number)
		if !ok {
			return false
		}
		a, validA := new(big.Rat).SetString(value.String())
		b, validB := new(big.Rat).SetString(other.String())
		if !validA || !validB {
			return value == other
		}
		return a.Cmp(b) == 0
	case map[string]any:
		other, ok := right.(map[string]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for key, item := range value {
			candidate, exists := other[key]
			if !exists || !Equal(item, candidate) {
				return false
			}
		}
		return true
	case []any:
		other, ok := right.([]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for index, item := range value {
			if !Equal(item, other[index]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(left, right)
	}
}
