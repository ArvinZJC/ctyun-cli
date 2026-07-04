/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package diagnostic carries runtime diagnostic message keys and arguments
// across package boundaries without hard-coding user-facing English templates
// in lower-level runtime paths.
package diagnostic

// Error is a runtime diagnostic with a localized-message key, optional
// arguments, and an optional wrapped cause.
type Error struct {
	Key   string
	Args  []any
	Cause error
}

// New returns a runtime diagnostic error for key and args.
func New(key string, args ...any) error {
	return Error{Key: key, Args: args}
}

// Wrap returns a runtime diagnostic error that wraps cause.
func Wrap(key string, cause error, args ...any) error {
	return Error{Key: key, Args: args, Cause: cause}
}

// Error returns a stable non-localized fallback for logs and tests that do not
// have language context.
func (e Error) Error() string {
	return e.Key
}

// Unwrap returns the wrapped cause.
func (e Error) Unwrap() error {
	return e.Cause
}

// MessageKey returns the localization key.
func (e Error) MessageKey() string {
	return e.Key
}

// MessageArgs returns localization arguments.
func (e Error) MessageArgs() []any {
	return e.Args
}
