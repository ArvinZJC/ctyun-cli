/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package waiter

import "testing"

func TestEvaluateWaiterSuccessFailureAndPending(t *testing.T) {
	spec := Spec{Path: "returnObj.status", Success: "running", Failure: "error"}

	state, err := Evaluate(spec, map[string]any{"returnObj": map[string]any{"status": "running"}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if state != Success {
		t.Fatalf("state = %s, want success", state)
	}

	state, err = Evaluate(spec, map[string]any{"returnObj": map[string]any{"status": "starting"}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if state != Pending {
		t.Fatalf("state = %s, want pending", state)
	}

	state, err = Evaluate(spec, map[string]any{"returnObj": map[string]any{"status": "error"}})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if state != Failure {
		t.Fatalf("state = %s, want failure", state)
	}
}

func TestEvaluateWaiterReportsMissingAndNonObjectPaths(t *testing.T) {
	spec := Spec{Path: "returnObj.status", Success: "running", Failure: "error"}

	state, err := Evaluate(spec, map[string]any{"returnObj": map[string]any{}})
	if err == nil {
		t.Fatal("Evaluate returned nil error for missing path")
	}
	if state != Pending {
		t.Fatalf("state = %s, want pending on missing path", state)
	}

	state, err = Evaluate(spec, map[string]any{"returnObj": "not-object"})
	if err == nil {
		t.Fatal("Evaluate returned nil error for non-object path")
	}
	if state != Pending {
		t.Fatalf("state = %s, want pending on non-object path", state)
	}
}
