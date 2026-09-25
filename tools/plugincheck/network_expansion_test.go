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

// TestNetworkProductNestedFailures rejects failed pricing and order submission
// even when the surrounding product envelope reports success.
func TestNetworkProductNestedFailures(t *testing.T) {
	for _, tc := range []struct{ name, id, field string }{
		{"native-firewall", "native-firewall.n100.order.create", "submitted"},
		{"native-firewall", "native-firewall.n100.order.upgrade", "submitted"},
		{"native-firewall", "native-firewall.n100.price.creation.show", "isSucceed"},
		{"sdwan", "sdwan.price.creation.show", "isSucceed"},
		{"sdwan", "sdwan.price.renewal.show", "isSucceed"},
		{"sdwan", "sdwan.price.upgrade.show", "isSucceed"},
	} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		op, ok := bundle.APIs.Operations[tc.id]
		if !ok {
			t.Fatalf("missing operation %s", tc.id)
		}
		for _, value := range []string{"true", "false", "null"} {
			response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"` + tc.field + `":` + value + `}}`))}
			_, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
			if (err == nil) != (value == "true") {
				t.Errorf("%s %s=%s: %v", tc.id, tc.field, value, err)
			}
		}
	}
}

// TestNetworkWaitersSelectExactIdentity prevents an unrelated completed row
// from making firewall readiness or SD-WAN task completion succeed.
func TestNetworkWaitersSelectExactIdentity(t *testing.T) {
	for _, tc := range []struct{ name, id, array, key, state, success, pending string }{
		{"native-firewall", "native-firewall.instance.ready", "list", "firewallId", "firewallState", "normal", "creating"},
		{"sdwan", "sdwan.task.completed", "result", "operationID", "status", "done", "undone"},
	} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		w, ok := bundle.Waiters.Waiters[tc.id]
		if !ok || w.Selector == nil {
			t.Fatalf("missing selector for %s", tc.id)
		}
		selector := *w.Selector
		selector.Value = "wanted"
		spec := waiter.Spec{Selector: &selector, Path: w.Path, Success: w.Success, FailureValues: w.FailureValues}
		payload := map[string]any{"returnObj": map[string]any{tc.array: []any{
			map[string]any{tc.key: "other", tc.state: tc.success},
			map[string]any{tc.key: "wanted", tc.state: tc.pending},
		}}}
		got, err := waiter.Evaluate(spec, payload)
		if err != nil || got != waiter.Pending {
			t.Fatalf("%s selected unrelated success: %s %v", tc.id, got, err)
		}
		selector.Value = "other"
		got, err = waiter.Evaluate(spec, payload)
		if err != nil || got != waiter.Success {
			t.Fatalf("%s failed exact match: %s %v", tc.id, got, err)
		}
	}
}
