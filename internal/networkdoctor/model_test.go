/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import "testing"

func TestCountAggregatesEveryStatus(t *testing.T) {
	got := Count([]Result{
		{Status: StatusPassed},
		{Status: StatusWarning},
		{Status: StatusFailed},
		{Status: StatusSkipped},
	})
	want := Counts{Passed: 1, Warning: 1, Failed: 1, Skipped: 1}
	if got != want {
		t.Fatalf("Count() = %#v, want %#v", got, want)
	}
}
