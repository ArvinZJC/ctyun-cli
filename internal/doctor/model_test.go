/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package doctor

import "testing"

func TestCountAggregatesEveryStatus(t *testing.T) {
	got := Count([]Status{StatusPassed, StatusWarning, StatusFailed, StatusSkipped})
	want := Counts{Passed: 1, Warning: 1, Failed: 1, Skipped: 1}
	if got != want || !got.HasFailures() {
		t.Fatalf("Count() = %#v, HasFailures() = %t", got, got.HasFailures())
	}
}

func TestStatusStringReturnsStableTokens(t *testing.T) {
	tests := map[Status]string{
		StatusPassed:  "passed",
		StatusWarning: "warning",
		StatusFailed:  "failed",
		StatusSkipped: "skipped",
	}
	for status, want := range tests {
		if got := status.String(); got != want {
			t.Fatalf("Status(%d).String() = %q, want %q", status, got, want)
		}
	}
	if got := Status(255).String(); got != "unknown" {
		t.Fatalf("Status(255).String() = %q, want unknown", got)
	}
}
