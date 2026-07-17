/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// ParameterValueType identifies the JSON value shape accepted by a command
// option when the value is bound into an API request.
type ParameterValueType string

const (
	// ParameterValueString preserves the command-line text as a string.
	ParameterValueString ParameterValueType = "string"
	// ParameterValueInteger accepts one JSON integer.
	ParameterValueInteger ParameterValueType = "integer"
	// ParameterValueNumber accepts one finite JSON number.
	ParameterValueNumber ParameterValueType = "number"
	// ParameterValueBoolean accepts the JSON literals true and false.
	ParameterValueBoolean ParameterValueType = "boolean"
	// ParameterValueStringArray accepts a JSON array containing only strings.
	ParameterValueStringArray ParameterValueType = "string_array"
	// ParameterValueIntegerArray accepts a JSON array containing only integers.
	ParameterValueIntegerArray ParameterValueType = "integer_array"
	// ParameterValueObjectArray accepts a JSON array containing only objects.
	ParameterValueObjectArray ParameterValueType = "object_array"
	// ParameterValueStringMap accepts a JSON object whose values are strings.
	ParameterValueStringMap ParameterValueType = "string_map"
	// ParameterValueJSON accepts any single JSON value.
	ParameterValueJSON ParameterValueType = "json"
)

// ParseParameterValue converts raw command-line text into the value shape
// declared by parameter without losing JSON number precision.
func ParseParameterValue(parameter Parameter, raw string) (any, error) {
	valueType := parameter.ValueType
	if valueType == "" || valueType == ParameterValueString {
		return raw, nil
	}
	value, err := decodeParameterJSON(raw)
	if err != nil {
		return nil, err
	}
	switch valueType {
	case ParameterValueInteger:
		number, ok := value.(json.Number)
		if !ok || strings.ContainsAny(number.String(), ".eE") {
			return nil, fmt.Errorf("expected JSON integer")
		}
		return number, nil
	case ParameterValueNumber:
		number, ok := value.(json.Number)
		if !ok {
			return nil, fmt.Errorf("expected JSON number")
		}
		parsed, err := strconv.ParseFloat(number.String(), 64)
		if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
			return nil, fmt.Errorf("expected finite JSON number")
		}
		return number, nil
	case ParameterValueBoolean:
		boolean, ok := value.(bool)
		if !ok || (raw != "true" && raw != "false") {
			return nil, fmt.Errorf("expected JSON boolean")
		}
		return boolean, nil
	case ParameterValueStringArray:
		values, ok := value.([]any)
		if !ok || !allStrings(values) {
			return nil, fmt.Errorf("expected JSON array of strings")
		}
		return values, nil
	case ParameterValueIntegerArray:
		values, ok := value.([]any)
		if !ok || !allIntegers(values) {
			return nil, fmt.Errorf("expected JSON array of integers")
		}
		return values, nil
	case ParameterValueObjectArray:
		values, ok := value.([]any)
		if !ok || !allObjects(values) {
			return nil, fmt.Errorf("expected JSON array of objects")
		}
		return values, nil
	case ParameterValueStringMap:
		values, ok := value.(map[string]any)
		if !ok || !allStringValues(values) {
			return nil, fmt.Errorf("expected JSON object with string values")
		}
		return values, nil
	case ParameterValueJSON:
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported parameter value type %q", valueType)
	}
}

// supportedParameterValueType reports whether valueType belongs to the stable
// plugin metadata vocabulary. An omitted value is the string default.
func supportedParameterValueType(valueType ParameterValueType) bool {
	switch valueType {
	case "", ParameterValueString, ParameterValueInteger, ParameterValueNumber,
		ParameterValueBoolean, ParameterValueStringArray, ParameterValueIntegerArray, ParameterValueObjectArray,
		ParameterValueStringMap, ParameterValueJSON:
		return true
	default:
		return false
	}
}

// decodeParameterJSON decodes exactly one JSON value while retaining numbers
// as json.Number values.
func decodeParameterJSON(raw string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("expected one JSON value")
		}
		return nil, err
	}
	return value, nil
}

// allStrings reports whether every array value is a string.
func allStrings(values []any) bool {
	for _, value := range values {
		if _, ok := value.(string); !ok {
			return false
		}
	}
	return true
}

// allIntegers reports whether every array value is a JSON integer.
func allIntegers(values []any) bool {
	for _, value := range values {
		number, ok := value.(json.Number)
		if !ok || strings.ContainsAny(number.String(), ".eE") {
			return false
		}
	}
	return true
}

// allObjects reports whether every array value is a JSON object.
func allObjects(values []any) bool {
	for _, value := range values {
		if _, ok := value.(map[string]any); !ok {
			return false
		}
	}
	return true
}

// allStringValues reports whether every JSON object value is a string.
func allStringValues(values map[string]any) bool {
	for _, value := range values {
		if _, ok := value.(string); !ok {
			return false
		}
	}
	return true
}
