/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package waiter evaluates metadata-defined command waiters against CTyun JSON
// responses.
package waiter

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/jsonvalue"
)

// State is the normalized result of evaluating a waiter condition.
type State string

const (
	// Success means the observed value matched the configured success value.
	Success State = "success"
	// Failure means the observed value matched the configured failure value.
	Failure State = "failure"
	// Pending means the observed value is neither successful nor failed yet.
	Pending State = "pending"
	// Timeout is reserved for callers that exhaust polling attempts.
	Timeout State = "timeout"
)

// Spec describes the response path and terminal values for one waiter.
type Spec struct {
	Selector      *Selector
	SuccessValues []string
	FailureValues []string
	Path          string
	Success       string
	Failure       string
}

// Evaluate reads spec.Path from payload and classifies it as success, failure,
// or pending.
func Evaluate(spec Spec, payload map[string]any) (State, error) {
	var root any = payload
	if spec.Selector != nil {
		rows, err := collectionRows(payload, spec.Selector.Path)
		if err != nil {
			return Pending, err
		}
		var selected any
		for _, row := range rows {
			identity, err := valueAtPath(row, spec.Selector.Key)
			if err != nil {
				return Pending, err
			}
			text, err := scalarText(identity, spec.Selector.Key)
			if err != nil {
				return Pending, err
			}
			if text != spec.Selector.Value {
				continue
			}
			if selected != nil {
				return Pending, diagnostic.New("error.waiter_ambiguous_resource", spec.Selector.Value)
			}
			selected = row
		}
		if selected == nil {
			return Pending, nil
		}
		root = selected
	}
	value, err := valueAtPath(root, spec.Path)
	if err != nil {
		return Pending, err
	}
	if value == nil {
		return Pending, nil
	}
	text, err := scalarText(value, spec.Path)
	if err != nil {
		return Pending, err
	}
	switch {
	case (spec.Success != "" && text == spec.Success) || slices.Contains(spec.SuccessValues, text):
		return Success, nil
	case (spec.Failure != "" && text == spec.Failure) || slices.Contains(spec.FailureValues, text):
		return Failure, nil
	default:
		return Pending, nil
	}
}

// valueAtPath walks a dot-separated object path through decoded JSON values.
func valueAtPath(value any, path string) (any, error) {
	current := value
	for part := range strings.SplitSeq(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, diagnostic.New("error.path_cannot_read", path, part)
		}
		current, ok = object[part]
		if !ok {
			return nil, diagnostic.New("error.path_missing", path, part)
		}
	}
	return current, nil
}

// Selector selects one exact resource identity from a JSON collection. Runtime
// Value is resolved from the catalog's command argument or option reference.
type Selector struct {
	Path  string `json:"path"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// collectionRows requires a JSON array rather than treating object or null
// responses as empty collections and hiding response-shape errors.
func collectionRows(payload map[string]any, path string) ([]any, error) {
	value, err := valueAtPath(payload, path)
	if err != nil {
		return nil, err
	}
	rows, ok := value.([]any)
	if !ok {
		return nil, diagnostic.New("error.waiter_expected_collection", path)
	}
	return rows, nil
}

// scalarText preserves JSON scalar spellings without accepting objects or
// arrays as lifecycle states or resource identities.
func scalarText(value any, path string) (string, error) {
	switch value.(type) {
	case float64:
		return strconv.FormatFloat(value.(float64), 'f', -1, 64), nil
	case json.Number:
		return jsonvalue.NumberText(value.(json.Number)), nil
	case string, bool, int, int64:
		return fmt.Sprint(value), nil
	default:
		return "", diagnostic.New("error.waiter_expected_scalar", path)
	}
}

// ValidateExample checks response paths and scalar types for every captured
// row. Empty examples cannot prove a collection identity or state mapping.
func ValidateExample(spec Spec, payload map[string]any) error {
	if spec.Selector == nil {
		_, err := Evaluate(spec, payload)
		return err
	}
	rows, err := collectionRows(payload, spec.Selector.Path)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return diagnostic.New("error.waiter_empty_example")
	}
	for _, row := range rows {
		identity, err := valueAtPath(row, spec.Selector.Key)
		if err != nil {
			return err
		}
		value, err := scalarText(identity, spec.Selector.Key)
		if err != nil {
			return err
		}
		selector := *spec.Selector
		selector.Value = value
		selected := spec
		selected.Selector = &selector
		if _, err := Evaluate(selected, payload); err != nil {
			return err
		}
	}
	return nil
}
