/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package waiter evaluates metadata-defined command waiters against CTyun JSON
// responses.
package waiter

import (
	"fmt"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
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
	Path    string
	Success string
	Failure string
}

// Evaluate reads spec.Path from payload and classifies it as success, failure,
// or pending.
func Evaluate(spec Spec, payload map[string]any) (State, error) {
	value, err := valueAtPath(payload, spec.Path)
	if err != nil {
		return Pending, err
	}
	text := fmt.Sprint(value)
	switch text {
	case spec.Success:
		return Success, nil
	case spec.Failure:
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
