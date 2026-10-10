/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestAPIGatewayFormJSONBindings preserves the two published form contracts:
// already serialized JSON strings and object fields serialized exactly once.
func TestAPIGatewayFormJSONBindings(t *testing.T) {
	for _, tc := range []struct {
		name, path, field, value string
		args                     []string
	}{
		{"order string", "/v3/order/placeNewPurchaseOrder", "orderDetailJson", `{"orders":[],"note":"a&b+中文"}`, []string{"order", "create", "--order-detail-json", `{"orders":[],"note":"a&b+中文"}`}},
		{"rule object", "/v3/createSecurityGroupRule", "jsonStr", `{"iptype":"IPv4","regionId":"self-operated-test"}`, []string{"security-group", "rule", "create", "--json-str", `{"iptype":"IPv4","regionId":"self-operated-test"}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.Path != tc.path || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Fatalf("unexpected request: %s %s %s", r.Method, r.URL, r.Header.Get("Content-Type"))
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				values, err := url.ParseQuery(string(body))
				if err != nil {
					t.Fatal(err)
				}
				var decoded map[string]any
				if err := json.Unmarshal([]byte(values.Get(tc.field)), &decoded); err != nil {
					t.Fatalf("form field is not a JSON object: %q %v", values.Get(tc.field), err)
				}
				var expected map[string]any
				if err := json.Unmarshal([]byte(tc.value), &expected); err != nil {
					t.Fatal(err)
				}
				for key, value := range expected {
					if key == "orders" {
						continue
					}
					if decoded[key] != value {
						t.Fatalf("field %s changed: %#v", key, decoded)
					}
				}
				if len(values) != 1 {
					t.Fatalf("unexpected form fields: %v", values)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"submitted":true}}`))}, nil
			})
			args := append([]string{"--yes", "api-gateway"}, tc.args...)
			err := cli.Run(cli.Config{Args: args, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

// TestAPIGatewayProjectScopeOverridesUpstreamDefault checks that customer project
// queries cannot silently take the upstream joint-operation default branch.
func TestAPIGatewayProjectScopeOverridesUpstreamDefault(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/v3/ondemand/queryProjectIds" || r.Header.Get("isHw") != "false" {
			t.Fatalf("incorrect self-operated scope: %s %v", r.URL, r.Header)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":[]}`))}, nil
	})
	cfg := cli.Config{Args: []string{"api-gateway", "project", "list"}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }}
	if err := cli.Run(cfg); err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	cfg.Args = append(cfg.Args, "--is-hw", "true")
	if err := cli.Run(cfg); err == nil || calls != 1 {
		t.Fatalf("joint-operation scope accepted: calls=%d err=%v", calls, err)
	}
}

// TestAPIGatewayGETRestorationSafety keeps restoration behind confirmation even
// though the published APIs use GET rather than a mutation-specific HTTP verb.
func TestAPIGatewayGETRestorationSafety(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/api-gateway"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, c := range b.Commands.Commands {
		if c.ID != "api-gateway.snapshot.restore" && c.ID != "api-gateway.disk.backup.restore" {
			continue
		}
		count++
		op := b.APIs.Operations[c.Operation]
		if op.Method != "GET" || op.Retryable || c.Dangerous.Confirm != "yes" {
			t.Errorf("unsafe restoration contract: %s", c.ID)
		}
	}
	if count != 2 {
		t.Fatalf("restoration commands=%d", count)
	}
}
