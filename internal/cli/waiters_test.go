/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestWaiterBindingRejectsBeforeOutput prevents applying a valid waiter to a
// different command even when that command happens to return the same path.
func TestWaiterBindingRejectsBeforeOutput(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ecs")
	writeWaitBundle(t, dir)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[{"id":"ecs.instance.show","path":["ecs","instance","show","{instance_id}"],"table":"ecs.instance.show","fixture_response":"fixtures/show.json"},{"id":"ecs.instance.other","path":["ecs","instance","other","{instance_id}"],"table":"ecs.instance.show","fixture_response":"fixtures/show.json"}]}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{"waiters":{"ecs.instance.running":{"commands":["ecs.instance.show"],"path":"returnObj.status","success":"running","failure":"error"}}}`)
	var out bytes.Buffer
	err := Run(Config{Args: []string{"ecs", "instance", "other", "id", "--offline", "--wait", "ecs.instance.running"}, Stdout: &out, PluginRoot: root})
	if err == nil || out.Len() != 0 {
		t.Fatalf("inapplicable waiter: error=%v output=%s", err, out.String())
	}
	got := completeArgs([]string{"ecs", "instance", "other", "id", "--wait", ""}, root)
	if len(got) != 0 {
		t.Fatalf("inapplicable completion: %v", got)
	}
	out.Reset()
	if err := Run(Config{Args: []string{"ecs", "instance", "show", "--help"}, Stdout: &out, PluginRoot: root}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "ecs.instance.running") {
		t.Fatalf("help does not expose waiter: %s", out.String())
	}
}

// TestWaiterRejectsUnsafeOperationBeforeHTTP protects against accidentally
// submitting or repeating a mutation, including legacy unbound waiters.
func TestWaiterRejectsUnsafeOperationBeforeHTTP(t *testing.T) {
	for _, tc := range []struct{ name, operation, confirm, wait string }{
		{"not retryable", "false", "", "ecs.instance.running"},
		{"dangerous", "true", "yes", "ecs.instance.running"},
		{"unknown waiter", "true", "", "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "ecs")
			writePollingWaitBundle(t, dir)
			mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[{"id":"ecs.instance.show","path":["ecs","instance","show","{instance_id}"],"operation":"v4.ecs.instance.show","table":"ecs.instance.show","dangerous":{"confirm":"`+tc.confirm+`"}}]}`)
			mustWrite(t, filepath.Join(dir, "apis.json"), `{"operations":{"v4.ecs.instance.show":{"method":"POST","path":"/mutate","retryable":`+tc.operation+`}}}`)
			transport := &waiterRejectTransport{}
			var out bytes.Buffer
			err := Run(Config{Args: []string{"ecs", "instance", "show", "id", "--wait", tc.wait, "--yes"}, Stdout: &out, HTTPTransport: transport, PluginRoot: root})
			if err == nil || transport.calls != 0 || out.Len() != 0 {
				t.Fatalf("error=%v requests=%d stdout=%s", err, transport.calls, out.String())
			}
		})
	}
	if err := renderWaiter(io.Discard, plugin.Bundle{}, "missing", nil, nil, "en-US", ""); err == nil {
		t.Fatal("unknown waiter accepted")
	}
}

// waiterRejectTransport counts attempted outbound requests without network access.
type waiterRejectTransport struct{ calls int }

// RoundTrip fails any request reaching a preflight-rejection test.
func (transport *waiterRejectTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls++
	return nil, fmt.Errorf("unexpected HTTP request")
}

// TestWaiterMultipleOutcomesPollsAndStops exercises extra success/failure values
// through the polling loop and verifies the request limit for unknown states.
func TestWaiterMultipleOutcomesPollsAndStops(t *testing.T) {
	for _, tc := range []struct {
		next, want string
		calls      int
	}{{"syncing", "success", 1}, {"expired", "failure", 1}, {"unknown", "timeout", 2}} {
		var out bytes.Buffer
		calls := 0
		bundle := plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"ready": {Path: "state", Success: "available", SuccessValues: []string{"syncing"}, Failure: "error", FailureValues: []string{"expired"}, MaxAttempts: 3}}}}
		err := renderWaiter(&out, bundle, "ready", map[string]any{"state": "creating"}, func() (map[string]any, error) { calls++; return map[string]any{"state": tc.next}, nil }, "en-US", "")
		if err != nil || calls != tc.calls || !strings.Contains(out.String(), ": "+tc.want) {
			t.Fatalf("%s: requests=%d result=%s error=%v", tc.next, calls, out.String(), err)
		}
	}
}

// TestCollectionWaiterRequiresIdentityBeforeRequest checks runtime selection,
// optional filter requirements and help against a synthetic collection bundle.
func TestCollectionWaiterRequiresIdentityBeforeRequest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ecs")
	writeWaitBundle(t, dir)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[{"id":"ecs.instance.show","path":["ecs","instance","show","{instance_id}"],"table":"ecs.instance.show","fixture_response":"fixtures/show.json","parameters":[{"name":"resource_id","flag":"resource-id","target":"id"}]}]}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{"waiters":{"ready":{"commands":["ecs.instance.show"],"selector":{"path":"items","key":"id","value":"$param.resource_id"},"path":"state","success":"ready","failure":"error","max_attempts":1}}}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "show.json"), `{"items":[{"id":"wrong","state":"error"},{"id":"target","state":"ready"}]}`)
	var out, errout bytes.Buffer
	err := Run(Config{Args: []string{"ecs", "instance", "show", "unused", "--wait", "ready", "--output", "json"}, Stdout: &out, Stderr: &errout, PluginRoot: root})
	if err == nil || out.Len() != 0 {
		t.Fatalf("omitted identity: error=%v stdout=%s", err, out.String())
	}
	out.Reset()
	errout.Reset()
	err = Run(Config{Args: []string{"ecs", "instance", "show", "unused", "--resource-id", "target", "--wait", "ready", "--offline", "--output", "json"}, Stdout: &out, Stderr: &errout, PluginRoot: root})
	if err != nil || !strings.Contains(errout.String(), "ready: success") {
		t.Fatalf("selected wait: %v %s", err, errout.String())
	}
	out.Reset()
	if err := Run(Config{Args: []string{"ecs", "instance", "show", "--help"}, Stdout: &out, PluginRoot: root}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Requires --resource-id") {
		t.Fatal(out.String())
	}
	command := plugin.Command{Path: []string{"test", "{id}"}}
	spec := plugin.Waiter{Selector: &waiter.Selector{Value: "$arg.id"}}
	if value, err := resolveWaiterSelection(command, spec, map[string]string{"id": "target"}, nil); err != nil || value != "target" {
		t.Fatalf("argument selection: %q %v", value, err)
	}
	command.Parameters = []plugin.Parameter{{Name: "numeric_id", Flag: "numeric-id", ValueType: plugin.ParameterValueInteger}}
	spec.Selector.Value = "$param.numeric_id"
	if value, err := resolveWaiterSelection(command, spec, nil, map[string]string{"numeric_id": " 9007199254740993 "}); err != nil || value != "9007199254740993" {
		t.Fatalf("numeric selection: %q %v", value, err)
	}
	spec.Selector.Value = "invalid"
	if _, err := resolveWaiterSelection(command, spec, nil, nil); err == nil {
		t.Fatal("invalid selector source accepted")
	}
}

// TestCollectionWaiterWrongCommandPrecedesIdentity rejects wrong bindings with
// a useful command diagnostic instead of requiring an absent selector input.
func TestCollectionWaiterWrongCommandPrecedesIdentity(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ecs")
	writeWaitBundle(t, dir)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[{"id":"selected","path":["ecs","selected","{id}"],"table":"ecs.instance.show","fixture_response":"fixtures/show.json"},{"id":"other","path":["ecs","other"],"table":"ecs.instance.show","fixture_response":"fixtures/show.json"}]}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{"waiters":{"ready":{"commands":["selected"],"selector":{"path":"items","key":"id","value":"$arg.id"},"path":"state","success":"ready"}}}`)
	transport := &waiterRejectTransport{}
	var out bytes.Buffer
	err := Run(Config{Args: []string{"ecs", "other", "--wait", "ready"}, Stdout: &out, HTTPTransport: transport, PluginRoot: root})
	if err == nil || !strings.Contains(err.Error(), "error.waiter_not_applicable") || transport.calls != 0 || out.Len() != 0 {
		t.Fatalf("error=%v requests=%d stdout=%s", err, transport.calls, out.String())
	}
}
