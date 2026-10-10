/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestDRDSOrderSubmissionRequiresBusinessSuccess protects GET order mutations
// from retries and rejects failed submissions inside HTTP-success envelopes.
func TestDRDSOrderSubmissionRequiresBusinessSuccess(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/drds"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"drds.instance.renew", "drds.instance.unsubscribe"} {
		var command plugin.Command
		for _, candidate := range bundle.Commands.Commands {
			if candidate.ID == id {
				command = candidate
			}
		}
		op, ok := bundle.APIs.Operations[command.Operation]
		if !ok || op.Method != "GET" || op.Retryable || command.Dangerous.Confirm != "yes" {
			t.Fatalf("unsafe order command %s", id)
		}
		for _, submitted := range []string{"true", "false", "null"} {
			response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":200,"returnObj":{"data":{"submitted":` + submitted + `}}}`))}
			_, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
			if (err == nil) != (submitted == "true") {
				t.Errorf("%s submitted=%s: %v", id, submitted, err)
			}
		}
	}
}

// TestDRDSWaitersDistinguishPreparationFromCompletion keeps cutover readiness
// separate from final success and rejects warning-only DDL outcomes.
func TestDRDSWaitersDistinguishPreparationFromCompletion(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/drds"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id    string
		value int
		want  waiter.State
	}{
		{"drds.redistribution.ready-to-switch", 8, waiter.Success},
		{"drds.redistribution.succeeded", 8, waiter.Pending},
		{"drds.redistribution.succeeded", 0, waiter.Success},
		{"drds.redistribution.succeeded", 2, waiter.Failure},
		{"drds.redistribution.succeeded", 9, waiter.Failure},
		{"drds.redistribution.succeeded", 6, waiter.Pending},
		{"drds.ddl-task.succeeded", 4, waiter.Failure},
		{"drds.ddl-task.succeeded", 1, waiter.Pending},
		{"drds.ddl-task.succeeded", 0, waiter.Success},
	} {
		w, ok := bundle.Waiters.Waiters[tc.id]
		if !ok {
			t.Fatalf("missing waiter %s", tc.id)
		}
		got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, FailureValues: w.FailureValues}, map[string]any{"returnObj": map[string]any{"result": tc.value}})
		if err != nil || got != tc.want {
			t.Errorf("%s result=%d: %s %v", tc.id, tc.value, got, err)
		}
	}
}
