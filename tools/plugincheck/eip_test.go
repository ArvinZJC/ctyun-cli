/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestEIPInventorySafety preserves the linked EIP inventory and mutation boundaries.
func TestEIPInventorySafety(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/eip"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Commands.Commands) != 57 || len(b.Waiters.Waiters) != 5 || b.Manifest.API.CtyunProductID != 13 {
		t.Fatal("EIP inventory or provenance changed")
	}
	for _, c := range b.Commands.Commands {
		o := b.APIs.Operations[c.Operation]
		if o.Retryable == (c.Dangerous.Confirm != "") {
			t.Fatalf("unsafe retry/confirmation contract: %s", c.ID)
		}
		for _, p := range c.Parameters {
			if p.Name == "page_no" && p.Deprecation != nil {
				t.Fatalf("recommended page option deprecated: %s", c.ID)
			}
		}
	}
}

// TestEIPWireAndConditionalInputs checks signed retrievals and subscription branches without cloud calls.
func TestEIPWireAndConditionalInputs(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		calls int
	}{
		{"detail", []string{"show", "--eip-id", "eip-test", "--region", "explicit"}, 1},
		{"missing duration", []string{"create", "--client-token", "test", "--name", "test", "--cycle-type", "month"}, 0},
		{"on demand", []string{"create", "--client-token", "test", "--name", "test", "--cycle-type", "on_demand"}, 1},
		{"missing billing method", []string{"price", "create", "--cycle-type", "on_demand", "--bandwidth", "1"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := cli.Run(cli.Config{Args: append([]string{"--yes", "--output", "json", "eip"}, tc.args...), PluginRoot: t.TempDir(), Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile"}}}`), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.name == "detail" && (r.URL.Path != "/v4/eip/show" || r.Method != "GET" || r.URL.Query().Get("regionID") != "explicit" || r.URL.Query().Get("eipID") != "eip-test" || r.Header.Get("Eop-Authorization") == "") {
					t.Fatalf("wrong EIP request: %s", r.URL)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"status":"ACTIVE"}}`))}, nil
			})})
			if calls != tc.calls || (tc.calls == 0 && err == nil) || (tc.calls == 1 && err != nil) {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

// TestEIPHelpAndCompletion keeps first-class EIP options and waiter names discoverable.
func TestEIPHelpAndCompletion(t *testing.T) {
	for _, args := range [][]string{{"--lang", "en-US", "eip", "show", "--help"}, {"__complete", "eip", "show", "--"}} {
		var out bytes.Buffer
		if err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: &out, Stderr: io.Discard}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "--region") || !strings.Contains(out.String(), "--wait") {
			t.Fatalf("missing EIP controls: %s", out.String())
		}
	}
}

// TestEIPWaiterFixtures preserves upstream lifecycle spelling and offline evaluation.
func TestEIPWaiterFixtures(t *testing.T) {
	for _, tc := range []struct {
		args      []string
		id, state string
	}{
		{[]string{"show", "--eip-id", "eip-c2f7vle3vr"}, "unbound", "success"},

		{[]string{"bandwidth", "show", "--bandwidth-id", "bandwidth-sjwu65pf9t"}, "bandwidth-active", "success"},
	} {
		var out, stderr bytes.Buffer
		args := append([]string{"eip"}, tc.args...)
		args = append(args, "--offline", "--wait", tc.id, "--output", "json", "--lang", "en-US")
		err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: &out, Stderr: &stderr})
		if err != nil || !strings.Contains(stderr.String(), tc.id+": "+tc.state) {
			t.Fatalf("waiter %s: err=%v stderr=%s", tc.id, err, stderr.String())
		}
	}
}

// TestEIPWaiterStateSemantics distinguishes transitional, terminal and missing states.
func TestEIPWaiterStateSemantics(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/eip"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id    string
		value any
		want  waiter.State
	}{
		{"active", "DOWN", waiter.Pending}, {"active", "ACTIVE", waiter.Success}, {"active", "ERROR", waiter.Failure}, {"active", "EXPIRED", waiter.Failure}, {"unbound", "DOWN", waiter.Success}, {"deleted", "DELETED", waiter.Success}, {"ipv6-bandwidth-active", "CREATEING", waiter.Pending}, {"ipv6-bandwidth-active", "CREATING", waiter.Pending}, {"ipv6-bandwidth-active", nil, waiter.Pending}, {"ipv6-bandwidth-active", "FREEZING", waiter.Failure},
	} {
		w := b.Waiters.Waiters[tc.id]
		state, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, Failure: w.Failure, SuccessValues: w.SuccessValues, FailureValues: w.FailureValues}, map[string]any{"returnObj": map[string]any{"status": tc.value}})
		if err != nil || state != tc.want {
			t.Fatalf("%s/%v got %s,%v want %s", tc.id, tc.value, state, err, tc.want)
		}
	}
}
