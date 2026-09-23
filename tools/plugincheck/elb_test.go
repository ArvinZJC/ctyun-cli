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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestELBBundle protects all published API families, captured fixtures and mutation safety.
func TestELBBundle(t *testing.T) {
	dir := repoPath(t, "plugins/elb")
	b, err := plugin.LoadBundle(dir, version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Commands.Commands) != 114 || len(b.APIs.Operations) != 114 || len(b.Waiters.Waiters) != 12 {
		t.Fatal("ELB inventory changed")
	}
	if b.Manifest.API.CtyunProductID != 24 || b.Manifest.API.SourceRevision != "82" || b.Manifest.API.EndpointURL != "https://ctelb-global.ctapi.ctyun.cn" {
		t.Fatal("ELB provenance changed")
	}
	for _, c := range b.Commands.Commands {
		op := b.APIs.Operations[c.Operation]
		if !op.Retryable && c.Dangerous.Confirm == "" {
			t.Fatalf("unguarded mutation %s", c.ID)
		}
		data, err := os.ReadFile(filepath.Join(dir, c.FixtureResponse))
		if err != nil {
			t.Fatal(err)
		}
		f, err := client.DecodeFixture(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = client.DecodeHTTPResponse(f, client.RequestSpec{Method: op.Method, Response: op.Response}); err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		failure := &client.HTTPResponse{Status: 200, Body: io.NopCloser(strings.NewReader(`{"statusCode":900,"errorCode":"Openapi.Parameter.Error"}`))}
		if _, err = client.DecodeHTTPResponse(failure, client.RequestSpec{Response: op.Response}); err == nil {
			t.Fatalf("%s accepted failure", c.ID)
		}
	}
	for _, id := range []string{"elb.instance.show", "elb.listener.show", "elb.target.show"} {
		found := false
		for _, c := range b.Commands.Commands {
			if c.ID == id {
				for _, p := range c.Parameters {
					if p.Name == "id" && p.Deprecation != nil {
						found = true
					}
				}
			}
		}
		if !found {
			t.Fatalf("%s lost deprecated ID", id)
		}
	}
}

// TestELBWire verifies query identity, profile region and JSON object preservation.
func TestELBWire(t *testing.T) {
	for _, tc := range []struct {
		args []string
		path string
	}{
		{[]string{"instance", "show", "--elb-id", "lb-test"}, "/v4/elb/show-loadbalancer"},
		{[]string{"rule", "create", "--client-token", "token", "--listener-id", "listener-test", "--priority", "1", "--conditions", `[{"type":"Host","hostConfig":{"values":["example.com"]}}]`, "--action", `{"type":"Forward","forwardConfig":{"targetGroups":[{"targetGroupID":"tg-test","weight":100}]}}`}, "/v4/elb/create-rule"},
	} {
		called := false
		var out, stderr bytes.Buffer
		args := append([]string{"--yes", "--lang", "en-US", "--output", "json", "elb"}, tc.args...)
		args = append(args, "--region", "region-test")
		err := cli.Run(cli.Config{Args: args, Stdout: &out, Stderr: &stderr, PluginRoot: t.TempDir(), Env: func(k string) string {
			if k == "CTYUN_AK" || k == "CTYUN_SK" {
				return "test"
			}
			return ""
		}, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			called = true
			if r.URL.Path != tc.path {
				t.Fatal(r.URL)
			}
			if r.Method == "GET" {
				if r.URL.Query().Get("regionID") != "region-test" || r.URL.Query().Get("elbID") != "lb-test" {
					t.Fatal(r.URL)
				}
			} else {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["regionID"] != "region-test" || body["priority"] != float64(1) {
					t.Fatal(body)
				}
				if _, ok := body["conditions"].([]any); !ok {
					t.Fatal(body)
				}
				if _, ok := body["action"].(map[string]any); !ok {
					t.Fatal(body)
				}
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
		})})
		if err != nil || !called {
			t.Fatalf("called=%v err=%v stderr=%s", called, err, &stderr)
		}
	}
}

// TestELBWaiterStates keeps recoverable down and unknown states pending.
func TestELBWaiterStates(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/elb"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for id, w := range b.Waiters.Waiters {
		for _, v := range []any{nil, "unknown", "DOWN", "INACTIVE", "offline"} {
			state, err := waiter.Evaluate(waiter.Spec{Path: "state", Success: w.Success}, map[string]any{"state": v})
			if err != nil || state != waiter.Pending {
				t.Fatalf("%s %v: %v %v", id, v, state, err)
			}
		}
		state, err := waiter.Evaluate(waiter.Spec{Path: "state", Success: w.Success}, map[string]any{"state": w.Success})
		if err != nil || state != waiter.Success {
			t.Fatalf("%s: %v %v", id, state, err)
		}
	}
}

// TestELBHelpAndCompletion exposes localized commands, identity options and waiters.
func TestELBHelpAndCompletion(t *testing.T) {
	for _, args := range [][]string{{"--lang", "en-US", "elb", "instance", "show", "--help"}, {"--lang", "zh-CN", "elb", "instance", "show", "--help"}, {"__complete", "elb", "instance", "show", "--"}} {
		var out bytes.Buffer
		if err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: &out, Stderr: io.Discard}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"--elb-id", "--region", "--wait"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("missing %s: %s", want, &out)
			}
		}
	}
}

// TestELBMissingWaitIdentity rejects collection polling before any request is sent.
func TestELBMissingWaitIdentity(t *testing.T) {
	called := false
	err := cli.Run(cli.Config{Args: []string{"elb", "gateway", "target", "list", "--region", "region-test", "--target-group-id", "tg-test", "--wait", "elb.gateway.target.list-healthy"}, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string {
		if k == "CTYUN_AK" || k == "CTYUN_SK" {
			return "test"
		}
		return ""
	}, HTTPTransport: nativeWireTransport(func(*http.Request) (*http.Response, error) { called = true; return nil, nil })})
	if err == nil || called {
		t.Fatalf("missing target identity: called=%v err=%v", called, err)
	}
}

// TestELBConditionalInputs rejects incomplete HTTPS and mutual authentication settings.
func TestELBConditionalInputs(t *testing.T) {
	for _, args := range [][]string{
		{"listener", "update", "--listener-id", "listener-test", "--ca-enabled", "true"},
		{"certificate", "create", "--client-token", "token", "--name", "server", "--type", "Server", "--certificate", "certificate"},
	} {
		called := false
		args = append(append([]string{"elb"}, args...), "--region", "region-test", "--yes")
		err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string {
			if k == "CTYUN_AK" || k == "CTYUN_SK" {
				return "test"
			}
			return ""
		}, HTTPTransport: nativeWireTransport(func(*http.Request) (*http.Response, error) { called = true; return nil, nil })})
		if err == nil || called {
			t.Fatalf("conditional input reached transport=%v err=%v", called, err)
		}
	}
	b, err := plugin.LoadBundle(repoPath(t, "plugins/elb"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range b.Commands.Commands {
		for _, p := range c.Parameters {
			if p.Name == "page_no" && p.Deprecation != nil {
				t.Fatalf("modern page option falsely deprecated: %s", c.ID)
			}
		}
	}
}
