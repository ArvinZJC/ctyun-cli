/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package coverprofile

import (
	"errors"
	"strings"
	"testing"
)

// TestFilterDropsDefaultExclusionsAndKeepsOtherBlocks checks the maintained exclusions without hiding unrelated or retired ranges.
func TestFilterDropsDefaultExclusionsAndKeepsOtherBlocks(t *testing.T) {
	input := strings.Join([]string{
		"mode: set",
		"github.com/ArvinZJC/ctyun-cli/cmd/ctyun/main.go:9.13,11.2 1 0",
		"github.com/ArvinZJC/ctyun-cli/tools/coverage/main.go:15.13,18.2 3 0",
		"github.com/ArvinZJC/ctyun-cli/tools/openapi/main.go:18.13,23.2 3 0",
		"github.com\\ArvinZJC\\ctyun-cli\\internal\\cli\\locale_windows.go:20.42,38.2 8 1",
		"github.com/ArvinZJC/ctyun-cli/internal/testarchive/archive.go:20.68,22.2 1 0",
		"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline/promote.go:40.46,42.4 1 0",
		"github.com/ArvinZJC/ctyun-cli/internal/plugin/install.go:145.52,147.3 1 0",
		"github.com/ArvinZJC/ctyun-cli/internal/plugin/install.go:222.38,224.5 1 0",
		"github.com/ArvinZJC/ctyun-cli/internal/cli/cli.go:1306.1,1311.2 2 1",
		"malformed profile line",
	}, "\n")

	var out strings.Builder
	if err := Filter(strings.NewReader(input), &out, DefaultExclusions()); err != nil {
		t.Fatalf("Filter returned error: %v", err)
	}

	got := out.String()
	for _, unwanted := range []string{"cmd/ctyun/main.go", "tools/coverage/main.go", "tools/openapi/main.go:18.13", "testarchive/archive.go", "install.go:145.52", "install.go:222.38"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("filtered output contains excluded block %q:\n%s", unwanted, got)
		}
	}
	for _, want := range []string{"mode: set", "promote.go:40.46", "locale_windows.go:20.42", "cli.go:1306.1,1311.2", "malformed profile line"} {
		if !strings.Contains(got, want) {
			t.Fatalf("filtered output missing %q:\n%s", want, got)
		}
	}
}

// TestTotalPercentFindsTotalLine extracts the reported aggregate gate value.
func TestTotalPercentFindsTotalLine(t *testing.T) {
	report := "pkg/file.go:1:\tRun\t100.0%\ntotal:\t\t\t(statements)\t100.0%\n"
	if got := TotalPercent(report); got != "100.0%" {
		t.Fatalf("TotalPercent = %q, want 100.0%%", got)
	}
	if got := TotalPercent("no total here"); got != "" {
		t.Fatalf("TotalPercent without total = %q, want empty", got)
	}
}

// TestFilterReturnsReaderErrors preserves profile read errors.
func TestFilterReturnsReaderErrors(t *testing.T) {
	var out strings.Builder
	if err := Filter(errReader{}, &out, nil); !errors.Is(err, errRead) {
		t.Fatalf("Filter error = %v, want %v", err, errRead)
	}
}

// TestFilterReturnsWriterErrors preserves filtered output write errors.
func TestFilterReturnsWriterErrors(t *testing.T) {
	err := Filter(strings.NewReader("mode: set\n"), errWriter{}, nil)
	if !errors.Is(err, errWrite) {
		t.Fatalf("Filter error = %v, want %v", err, errWrite)
	}
}

// TestFilterKeepsMalformedBlocks keeps unrecognized records visible for diagnosis.
func TestFilterKeepsMalformedBlocks(t *testing.T) {
	input := strings.Join([]string{
		"file.go:1.1 1 0",
		"file.go:1.1,2 1 0",
		"file.go:x.1,2.1 1 0",
		"file.go:1.1,y.1 1 0",
	}, "\n")
	var out strings.Builder
	if err := Filter(strings.NewReader(input), &out, DefaultExclusions()); err != nil {
		t.Fatalf("Filter returned error: %v", err)
	}
	for want := range strings.SplitSeq(input, "\n") {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("filtered output missing malformed block %q:\n%s", want, out.String())
		}
	}
}

// errRead is the synthetic failing reader result.
var errRead = errors.New("read failed")

// errWrite is the synthetic failing writer result.
var errWrite = errors.New("write failed")

// errReader supplies a deterministic profile read failure.
type errReader struct{}

// Read returns the synthetic read error.
func (errReader) Read([]byte) (int, error) {
	return 0, errRead
}

// errWriter supplies a deterministic output write failure.
type errWriter struct{}

// Write returns the synthetic write error.
func (errWriter) Write([]byte) (int, error) {
	return 0, errWrite
}
