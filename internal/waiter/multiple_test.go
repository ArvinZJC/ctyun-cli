/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package waiter

import (
	"encoding/json"
	"testing"
)

// TestEvaluateMultipleTerminalValues checks numeric and string terminal states
// while leaving intermediate values pending.
func TestEvaluateMultipleTerminalValues(t *testing.T) {
	var spec Spec
	if err := json.Unmarshal([]byte(`{"Path":"status","Success":"available","Failure":"error","SuccessValues":["syncing"],"FailureValues":["expired","-1"]}`), &spec); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value any
		want  State
	}{{"available", Success}, {"syncing", Success}, {"error", Failure}, {"expired", Failure}, {-1, Failure}, {"creating", Pending}} {
		got, err := Evaluate(spec, map[string]any{"status": tc.value})
		if err != nil || got != tc.want {
			t.Errorf("%v: got %s, %v; want %s", tc.value, got, err, tc.want)
		}
	}
}
