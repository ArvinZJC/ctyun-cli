/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestCMTokenRefreshAndPagination preserves the mutation semantics of the GET
// token refresh and keeps still-published pagination options with replacement guidance.
func TestCMTokenRefreshAndPagination(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/cm"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	foundRefresh, foundPage := false, false
	for _, c := range b.Commands.Commands {
		if c.ID == "cm.external-alarm.token.refresh" {
			op := b.APIs.Operations[c.Operation]
			if op.Method != "GET" || op.Retryable || c.Dangerous.Confirm != "yes" {
				t.Fatal("token refresh lost mutation safety")
			}
			foundRefresh = true
		}
		if c.ID == "cm.alarm.history.list" {
			for _, p := range c.Parameters {
				if p.Name != "page" {
					continue
				}
				if p.Deprecation == nil || p.Deprecation.Replacement == nil || p.Deprecation.Replacement.Kind != "option" || p.Deprecation.Replacement.Label != "--page-no" {
					t.Fatal("published page option lost replacement guidance")
				}
				foundPage = true
			}
		}
	}
	if !foundRefresh || !foundPage {
		t.Fatal("expected monitoring commands missing")
	}
}

// TestCMProbeWaiterUsesOverallTaskStatus separates task completion from the
// nested protocol result codes returned by each individual probe.
func TestCMProbeWaiterUsesOverallTaskStatus(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/cm"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	w, ok := b.Waiters.Waiters["cm.probe.completed"]
	if !ok {
		t.Fatal("probe waiter missing")
	}
	for _, tc := range []struct {
		status int
		want   waiter.State
	}{{1, waiter.Pending}, {2, waiter.Success}, {3, waiter.Failure}, {4, waiter.Failure}} {
		payload := map[string]any{"returnObj": map[string]any{"status": tc.status, "data": map[string]any{"pingData": []any{map[string]any{"status": 200}}}}}
		got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, FailureValues: w.FailureValues}, payload)
		if err != nil || got != tc.want {
			t.Errorf("task status %d: %s %v", tc.status, got, err)
		}
	}
}
