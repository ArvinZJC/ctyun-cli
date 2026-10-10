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

// TestLTSIndexDeletionIsAMutation protects the published GET deletion from
// automatic retries and execution without confirmation.
func TestLTSIndexDeletionIsAMutation(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/lts"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range b.Commands.Commands {
		if c.ID != "lts.index.delete" {
			continue
		}
		op := b.APIs.Operations[c.Operation]
		if op.Method != "GET" || op.Retryable || c.Dangerous.Confirm != "yes" {
			t.Fatal("GET index deletion lost mutation safety")
		}
		return
	}
	t.Fatal("index deletion missing")
}

// TestLTSInstanceStateIsNotAnEnvelopeStatus separates successful status
// retrieval from readiness of the regional log service instance.
func TestLTSInstanceStateIsNotAnEnvelopeStatus(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/lts"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	op := b.APIs.Operations["lts.instance.status.show"]
	w := b.Waiters.Waiters["lts.instance.ready"]
	for _, tc := range []struct {
		body  string
		state waiter.State
	}{
		{`{"statusCode":0,"returnObj":{"statusCode":-1}}`, waiter.Pending},
		{`{"statusCode":0,"returnObj":{"statusCode":0}}`, waiter.Failure},
		{`{"statusCode":0,"returnObj":{"statusCode":1}}`, waiter.Success},
		{`{"statusCode":0,"returnObj":{"statusCode":2}}`, waiter.Pending},
		{`{"statusCode":0,"returnObj":{"statusCode":3}}`, waiter.Failure},
	} {
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body))}
		payload, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
		if err != nil {
			t.Fatalf("status query rejected: %v", err)
		}
		got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, FailureValues: w.FailureValues}, payload.Payload)
		if err != nil || got != tc.state {
			t.Fatalf("%s: state=%s err=%v", tc.body, got, err)
		}
	}
}
