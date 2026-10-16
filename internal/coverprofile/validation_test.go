/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package coverprofile

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// TestCheckFullCoverageRejectsEveryMiss prevents exclusions and rounded totals
// from concealing an unexecuted statement in any package.
func TestCheckFullCoverageRejectsEveryMiss(t *testing.T) {
	for _, file := range []string{"cmd/ctyun/main.go", "tools/coverage/main.go", "tools/openapi/main.go", "tools/release/main.go", "internal/testarchive/archive.go", "internal/distribution/fetch.go", "internal/release/install.go", "internal/plugin/install.go"} {
		t.Run(file, func(t *testing.T) {
			profile := "mode: set\nother.go:1.1,2.1 1000000 1\n" + file + ":1.1,2.1 1 0\n"
			if err := CheckFullCoverage(strings.NewReader(profile)); err == nil || !strings.Contains(err.Error(), file) {
				t.Fatalf("uncovered statement was not identified: %v", err)
			}
		})
	}
}

// TestCheckFullCoverageAcceptsExecutedStatements permits each Go profile mode
// and empty marker functions without ignoring executable statements.
func TestCheckFullCoverageAcceptsExecutedStatements(t *testing.T) {
	for _, mode := range []string{"set", "count", "atomic"} {
		profile := "mode: " + mode + "\nfile.go:1.1,2.1 3 2\nfile.go:4.1,4.2 0 0\n"
		if err := CheckFullCoverage(strings.NewReader(profile)); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCheckFullCoverageRejectsInvalidEvidence prevents an empty or corrupt
// profile from passing the gate without tested statements.
func TestCheckFullCoverageRejectsInvalidEvidence(t *testing.T) {
	for _, profile := range []string{"", "mode: unknown\n", "mode: set\n", "mode: set\nfile.go:1.1,2.1 0 0\n", "mode: set\nbroken\n", "mode: set\nfile.go:1.1,2.1 invalid 1\n", "mode: set\nfile.go:1.1,2.1 1 -1\n"} {
		if err := CheckFullCoverage(strings.NewReader(profile)); err == nil {
			t.Fatalf("invalid profile accepted: %q", profile)
		}
	}
	want := errors.New("profile read failed")
	if err := CheckFullCoverage(failingProfileReader{want}); !errors.Is(err, want) {
		t.Fatalf("read error = %v, want %v", err, want)
	}
	if err := CheckFullCoverage(io.MultiReader(strings.NewReader("mode: set\nfile.go:1.1,2.1 1 1\n"), failingProfileReader{want})); !errors.Is(err, want) {
		t.Fatalf("truncated profile error = %v, want %v", err, want)
	}
}

// failingProfileReader models a profile that cannot be fully read.
type failingProfileReader struct{ err error }

// Read returns the underlying storage failure.
func (r failingProfileReader) Read([]byte) (int, error) { return 0, r.err }
