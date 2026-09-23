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
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestVPCInventoryAndFixtures checks the full scoped inventory and captured responses.
func TestVPCInventoryAndFixtures(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/vpc"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Commands.Commands) != 193 || len(b.APIs.Operations) != 193 || b.Manifest.API.CtyunProductID != 18 || b.Manifest.API.SourceRevision != "88" || b.Manifest.API.EndpointURL != "https://ctvpc-global.ctapi.ctyun.cn" {
		t.Fatal("VPC inventory or provenance changed")
	}
	for _, c := range b.Commands.Commands {
		op := b.APIs.Operations[c.Operation]
		if op.Retryable == (c.Dangerous.Confirm != "") {
			t.Fatalf("unsafe retry and confirmation pairing: %s", c.ID)
		}
		assertCommandFixtureResponse(t, b, c)
		for _, p := range c.Parameters {
			if p.Name == "page_no" && p.Deprecation != nil {
				t.Fatalf("preferred pagination option deprecated: %s", c.ID)
			}
		}
	}
}

// TestVPCRequests preserves signed region routing and structured batch request values.
func TestVPCRequests(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		path string
	}{
		{"query", []string{"show", "--vpc-id", "vpc-test", "--region", "override"}, "/v4/vpc/query"},
		{"batch", []string{"security-group", "interface", "batch-bind", "--client-token", "test", "--security-group-id", "sg-test", "--port-ids", `["port-a","port-b"]`}, "/v4/vpc/batch-attach-security-group-ports"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := cli.Run(cli.Config{Args: append([]string{"--yes", "--output", "json", "vpc"}, tc.args...), Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile"}}}`), PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "ctvpc-global.ctapi.ctyun.cn" || r.URL.Path != tc.path || r.Header.Get("Eop-Authorization") == "" {
					t.Fatalf("unexpected routing: %s", r.URL)
				}
				if tc.name == "query" {
					if r.Method != "GET" || r.URL.Query().Get("regionID") != "override" || r.URL.Query().Get("vpcID") != "vpc-test" {
						t.Fatal("query bindings changed")
					}
				} else {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					ids, ok := body["portIDs"].([]any)
					if !ok || len(ids) != 2 || ids[1] != "port-b" || body["regionID"] != "profile" || body["securityGroupID"] != "sg-test" || r.Method != "POST" {
						t.Fatalf("batch body: %#v", body)
					}
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
			})})
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

// TestVPCHelpAndCompletion preserves canonical plural-ID inputs and profile-region overrides.
func TestVPCHelpAndCompletion(t *testing.T) {
	for _, args := range [][]string{{"--lang", "en-US", "vpc", "security-group", "interface", "batch-bind", "--help"}, {"__complete", "vpc", "security-group", "interface", "batch-bind", "--"}} {
		var out bytes.Buffer
		if err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: &out, Stderr: io.Discard}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "--port-ids") || !strings.Contains(out.String(), "--region") || strings.Contains(out.String(), "--port-i-ds") {
			t.Fatalf("bad options: %s", out.String())
		}
	}
}

// TestVPCConditionalInputs stops incomplete virtual-IP and address-allocation requests locally.
func TestVPCConditionalInputs(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		valid bool
	}{
		{"missing IPv6 source", []string{"interface", "ipv6", "assign", "--client-token", "test", "--network-interface-id", "port-test"}, false},
		{"IPv6 count", []string{"interface", "ipv6", "assign", "--client-token", "test", "--network-interface-id", "port-test", "--ipv6-addresses-count", "1"}, true},
		{"IPv6 addresses", []string{"interface", "ipv6", "assign", "--client-token", "test", "--network-interface-id", "port-test", "--ipv6-addresses", `["2001:db8::1"]`}, true},
		{"missing virtual machine", []string{"virtual-ip", "bind", "--client-token", "test", "--resource-type", "VM", "--ha-vip-id", "vip-test"}, false},
		{"elastic IP", []string{"virtual-ip", "bind", "--client-token", "test", "--resource-type", "NETWORK", "--ha-vip-id", "vip-test", "--floating-id", "eip-test"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := cli.Run(cli.Config{Args: append([]string{"--yes", "--output", "json", "vpc"}, tc.args...), Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"region-test"}}}`), PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
			})})
			if tc.valid && (calls != 1 || err != nil) || !tc.valid && (calls != 0 || err == nil) {
				t.Fatalf("valid=%v calls=%d err=%v", tc.valid, calls, err)
			}
		})
	}
}

// TestVPCWaiterFlows follows actual bundle bindings and enforces collection identities.
func TestVPCWaiterFlows(t *testing.T) {
	for _, tc := range []struct {
		args []string
		id   string
	}{
		{[]string{"interface", "status", "check", "--port-id", "port-test"}, "vpc.interface.ready"},
		{[]string{"peering", "show", "--instance-id", "vpr-test"}, "vpc.peering.accepted"},
		{[]string{"prefix-list", "list", "--prefix-list-id", "prefixlist-r5i4zghgvq"}, "vpc.prefix-list.available"},
	} {
		var out, stderr bytes.Buffer
		args := append([]string{"--lang", "en-US", "--output", "json", "vpc"}, tc.args...)
		args = append(args, "--offline", "--wait", tc.id)
		if err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: &out, Stderr: &stderr}); err != nil {
			t.Fatal(err)
		}
		if !json.Valid(out.Bytes()) || !strings.Contains(stderr.String(), tc.id+": success") {
			t.Fatalf("%s: stdout=%s stderr=%s", tc.id, &out, &stderr)
		}
	}
	called := false
	err := cli.Run(cli.Config{Args: []string{"vpc", "prefix-list", "list", "--region", "region-test", "--wait", "vpc.prefix-list.available"}, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(*http.Request) (*http.Response, error) { called = true; return nil, nil })})
	if err == nil || called {
		t.Fatalf("missing identity reached transport=%v err=%v", called, err)
	}
}
