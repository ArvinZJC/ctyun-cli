/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestEHPCPublicRefreshSafetyAndTaskStates pins mutation safety and every
// documented management-task outcome without inferring resource readiness.
func TestEHPCPublicRefreshSafetyAndTaskStates(t *testing.T) {
	bundle, err := plugin.LoadBundle(filepath.Join(repoPath(t, "plugins"), "ehpc"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]bool{"ehpc.delegate.create": true, "ehpc.hpc.cluster.create": true, "ehpc.hpc.cluster.expand": true, "ehpc.hpc.cluster.shrink": true, "ehpc.hpc.queue.create": true, "ehpc.hpc.queue.delete": true}
	for _, c := range bundle.Commands.Commands {
		if mutations[c.ID] && (c.Dangerous.Confirm == "" || bundle.APIs.Operations[c.Operation].Retryable) {
			t.Errorf("unsafe mutation %s", c.ID)
		}
	}
	w, ok := bundle.Waiters.Waiters["ehpc.job.completed"]
	if !ok {
		t.Fatal("missing task waiter")
	}
	for _, tc := range []struct {
		value any
		want  waiter.State
	}{{"wait", waiter.Pending}, {"pending_submit", waiter.Pending}, {"submitted", waiter.Pending}, {"success", waiter.Success}, {"force_completed", waiter.Success}, {"failed", waiter.Failure}, {"cancel", waiter.Failure}, {"console_failed", waiter.Failure}, {nil, waiter.Pending}, {"unknown", waiter.Pending}} {
		got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, SuccessValues: w.SuccessValues, Failure: w.Failure, FailureValues: w.FailureValues}, map[string]any{"returnObj": map[string]any{"status": tc.value}})
		if err != nil || got != tc.want {
			t.Errorf("state %v = %s, %v; want %s", tc.value, got, err, tc.want)
		}
	}
}

// TestEHPCPublicRefreshCLI exercises captured responses, the new waiter help,
// completion, and billing validation before an offline request is executed.
func TestEHPCPublicRefreshCLI(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		contains string
		fails    bool
	}{
		{"job fixture", []string{"ehpc", "job", "show", "--job-uuid", "documented-task", "--offline", "--output", "json"}, "status", false},
		{"job help", []string{"ehpc", "job", "show", "--help"}, "ehpc.job.completed", false},
		{"job completion", []string{"__complete", "ehpc", "job", "show", "--wait", ""}, "ehpc.job.completed", false},
		{"new region fixture", []string{"ehpc", "region", "list", "--offline", "--output", "json"}, "isSupportOceanFS", false},
		{"billing requirement", []string{"ehpc", "hpc", "cluster", "expand", "--region", "region", "--az-name", "az", "--cluster-uuid", "cluster", "--cluster-type-role-uuid", "role", "--node-create-method", "AutoCreate", "--ecs-compute-node-orders", "[]", "--on-demand", "false", "--yes", "--offline"}, "--cycle-count", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errs bytes.Buffer
			args := append([]string{"--lang", "en-US"}, tc.args...)
			if tc.args[0] == "__complete" {
				args = tc.args
			}
			err := cli.Run(cli.Config{Args: args, Stdout: &out, Stderr: &errs, PluginRoot: t.TempDir()})
			text := out.String() + errs.String()
			if err != nil {
				text += err.Error()
			}
			if (err != nil) != tc.fails || !strings.Contains(text, tc.contains) {
				t.Fatalf("err=%v output=%s", err, text)
			}
		})
	}
}

// TestEHPCTypedRequestBindings verifies arrays, booleans and region overrides
// through the signed HTTP boundary using a local in-memory transport.
func TestEHPCTypedRequestBindings(t *testing.T) {
	calls := 0
	err := cli.Run(cli.Config{Args: []string{"--yes", "--output", "json", "ehpc", "hpc", "cluster", "expand", "--region", "override", "--az-name", "az1", "--cluster-uuid", "cluster", "--cluster-type-role-uuid", "role", "--node-create-method", "AutoCreate", "--on-demand", "true", "--ecs-compute-node-orders", `[{"orderCount":2,"queueUUID":"queue"}]`}, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile"}}}`), Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" || r.URL.Path != "/v4/cthpc/hpc/expand-cluster" || body["regionID"] != "override" || body["onDemand"] != true || r.Header.Get("Eop-Authorization") == "" {
			t.Fatalf("invalid request %s %v", r.URL, body)
		}
		orders, ok := body["ecsComputeNodeOrders"].([]any)
		if !ok || len(orders) != 1 || orders[0].(map[string]any)["orderCount"] != float64(2) {
			t.Fatalf("orders lost JSON types: %v", body)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"jobUUID":"job"}}`))}, nil
	})})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestEHPCNodeListAndShrinkHelp keeps distinct upstream CSV and JSON array
// contracts from inheriting each other's descriptions or size constraints.
func TestEHPCNodeListAndShrinkHelp(t *testing.T) {
	for _, lang := range []string{"en-US", "en-GB", "zh-CN"} {
		for _, tc := range []struct {
			path         []string
			want, absent string
		}{
			{[]string{"hpc", "node", "list"}, map[string]string{"en-US": "Comma-separated node UUIDs", "en-GB": "Comma-separated node UUIDs", "zh-CN": "以英文逗号分隔的节点 UUID"}[lang], "JSON"},
			{[]string{"hpc", "cluster", "shrink"}, "[1, 10]", "100"},
		} {
			var out bytes.Buffer
			args := append([]string{"--lang", lang, "ehpc"}, tc.path...)
			args = append(args, "--help")
			if err := cli.Run(cli.Config{Args: args, Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir()}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("%s %v missing %q: %s", lang, tc.path, tc.want, out.String())
			}
			for line := range strings.SplitSeq(out.String(), "\n") {
				if strings.Contains(line, "--node-uuid-list") && strings.Contains(line, tc.absent) {
					t.Fatalf("wrong node input description: %s", line)
				}
			}
		}
	}
}

// TestEHPCQueueFixtureTable protects the captured direct-array response from
// being rendered as a blank wrapper row inferred from the conflicting schema.
func TestEHPCQueueFixtureTable(t *testing.T) {
	var out bytes.Buffer
	err := cli.Run(cli.Config{Args: []string{"--lang", "en-US", "--table", "plain", "ehpc", "hpc", "queue", "list", "--region", "region", "--az-name", "az", "--cluster-uuid", "cluster", "--offline"}, Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"pt-g6wwn5lcxjalhdjxrctnlilnowro", "a2", "creating"} {
		if !strings.Contains(out.String(), value) {
			t.Fatalf("missing queue value %q: %s", value, out.String())
		}
	}
}

// TestEHPCNodeSelectionWireTypes prevents CSV retrieval selectors from being
// serialized as mutation arrays, or JSON shrink arrays from becoming strings.
func TestEHPCNodeSelectionWireTypes(t *testing.T) {
	for _, tc := range []struct {
		path   []string
		value  string
		method string
	}{
		{[]string{"hpc", "node", "list"}, "node-a,node-b", "GET"},
		{[]string{"hpc", "cluster", "shrink"}, `["node-a","node-b"]`, "POST"},
	} {
		args := append([]string{"--yes", "--output", "json", "ehpc"}, tc.path...)
		args = append(args, "--region", "region", "--az-name", "az", "--cluster-uuid", "cluster", "--node-uuid-list", tc.value)
		calls := 0
		err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != tc.method {
				t.Fatalf("method = %s", r.Method)
			}
			if tc.method == "GET" {
				if r.URL.Query().Get("nodeUUIDList") != tc.value {
					t.Fatalf("wrong CSV selector: %s", r.URL)
				}
			} else {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				values, ok := body["nodeUUIDList"].([]any)
				if !ok || len(values) != 2 || values[0] != "node-a" || values[1] != "node-b" {
					t.Fatalf("wrong JSON selector: %v", body)
				}
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
		})})
		if err != nil || calls != 1 {
			t.Fatalf("%v calls=%d err=%v", tc.path, calls, err)
		}
	}
}
