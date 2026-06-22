/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package diagnostic

import (
	"errors"
	"testing"
)

func TestErrorCarriesMessageKeyArgsAndCause(t *testing.T) {
	err := New("error.demo", "arg")
	diagnosticErr, ok := err.(Error)
	if !ok {
		t.Fatalf("New returned %T, want Error", err)
	}
	if diagnosticErr.Error() != "error.demo" {
		t.Fatalf("Error() = %q, want key", diagnosticErr.Error())
	}
	if diagnosticErr.MessageKey() != "error.demo" {
		t.Fatalf("MessageKey() = %q, want error.demo", diagnosticErr.MessageKey())
	}
	if got := diagnosticErr.MessageArgs(); len(got) != 1 || got[0] != "arg" {
		t.Fatalf("MessageArgs() = %#v, want arg", got)
	}
	if diagnosticErr.Unwrap() != nil {
		t.Fatal("New diagnostic unexpectedly wrapped a cause")
	}

	cause := errors.New("cause")
	wrapped := Wrap("error.wrap", cause, "subject")
	if !errors.Is(wrapped, cause) {
		t.Fatal("Wrap did not expose cause through errors.Is")
	}
	wrappedDiagnostic := wrapped.(Error)
	if wrappedDiagnostic.MessageKey() != "error.wrap" {
		t.Fatalf("wrapped MessageKey() = %q, want error.wrap", wrappedDiagnostic.MessageKey())
	}
	if got := wrappedDiagnostic.MessageArgs(); len(got) != 1 || got[0] != "subject" {
		t.Fatalf("wrapped MessageArgs() = %#v, want subject", got)
	}
}
