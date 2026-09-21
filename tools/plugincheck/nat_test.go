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
)

// TestNATCapturedSurface preserves each endpoint boundary, all captured fixtures, and mutation safety.
func TestNATCapturedSurface(t *testing.T) {
	for _, tc := range []struct {
		name, host     string
		count, waiters int
	}{
		{"nat", "ctvpc-global.ctapi.ctyun.cn", 22, 2}, {"private-nat", "ctnat-global.ctapi.ctyun.cn", 21, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.name), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Commands.Commands) != tc.count || len(b.Waiters.Waiters) != tc.waiters || b.Manifest.API.EndpointURL != "https://"+tc.host || b.Manifest.API.CtyunProductID != 102 {
				t.Fatalf("incorrect boundary: %#v", b.Manifest)
			}
			for _, c := range b.Commands.Commands {
				op := b.APIs.Operations[c.Operation]
				for _, parameter := range c.Parameters {
					if parameter.Name == "page_no" && parameter.Deprecation != nil {
						t.Fatalf("recommended page number deprecated: %s", c.ID)
					}
				}

				if (op.Retryable && c.Dangerous.Confirm != "") || (!op.Retryable && c.Dangerous.Confirm == "") {
					t.Fatalf("unsafe command %s", c.ID)
				}
				if strings.HasPrefix(op.Path, "/v4/privatenat/") != (tc.name == "private-nat") {
					t.Fatalf("crossed endpoint boundary: %s", op.Path)
				}
				data, err := os.ReadFile(repoPath(t, filepath.Join("plugins", tc.name, c.FixtureResponse)))
				if err != nil {
					t.Fatal(err)
				}
				response, err := client.DecodeFixture(data)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response}); err != nil {
					t.Fatalf("%s: %v", c.ID, err)
				}
			}
		})
	}
}

// TestNATWireRouting checks signing, region overrides, structured arrays and exact endpoint selection.
func TestNATWireRouting(t *testing.T) {
	for _, name := range []string{"nat", "private-nat"} {
		for _, override := range []bool{false, true} {
			calls := 0
			args := []string{"--yes", "--output", "json", name, "snat", "create", "--nat-gateway-id", "gateway-test", "--source-cidr", "10.0.0.0/24", "--snat-ips", `["192.0.2.1","192.0.2.2"]`}
			if name == "nat" {
				args = append(args, "--client-token", "token-test")
			}
			if override {
				args = append(args, "--region", "explicit-region")
			}
			err := cli.Run(natTestConfig(t, args, nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				wantHost := "ctvpc-global.ctapi.ctyun.cn"
				wantPath := "/v4/vpc/create-snat-entry"
				arrayName := "snatIps"
				if name == "private-nat" {
					wantHost = "ctnat-global.ctapi.ctyun.cn"
					wantPath = "/v4/privatenat/create-snat"
					arrayName = "snatIPs"
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				region := "profile-region"
				if override {
					region = "explicit-region"
				}
				values, ok := body[arrayName].([]any)
				if r.URL.Host != wantHost || r.URL.Path != wantPath || r.Method != "POST" || body["regionID"] != region || !ok || len(values) != 2 || values[0] != "192.0.2.1" || r.Header.Get("Eop-Authorization") == "" {
					t.Fatalf("request: %s %#v", r.URL, body)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
			})))
			if err != nil || calls != 1 {
				t.Fatalf("%s override=%v calls=%d err=%v", name, override, calls, err)
			}
		}
	}
}

// TestNATValidationAndWaitIdentity prevents incomplete branch inputs and unselected collection polling from reaching transport.
func TestNATValidationAndWaitIdentity(t *testing.T) {
	for _, args := range [][]string{
		{"nat", "dnat", "create", "--nat-gateway-id", "g", "--external-id", "e", "--external-port", "80", "--internal-port", "80", "--virtual-machine-type", "1", "--protocol", "tcp", "--client-token", "t"},
		{"nat", "dnat", "create", "--nat-gateway-id", "g", "--external-id", "e", "--external-port", "80", "--internal-port", "80", "--virtual-machine-type", "2", "--protocol", "tcp", "--client-token", "t"},

		{"nat", "snat", "create", "--nat-gateway-id", "g", "--snat-ips", `["192.0.2.1"]`, "--client-token", "t"},
		{"private-nat", "snat", "create", "--nat-gateway-id", "g", "--snat-ips", `["192.0.2.1"]`},
		{"private-nat", "dnat", "create", "--nat-gateway-id", "g", "--external-ip", "192.0.2.1", "--external-port", "80", "--internal-port", "80", "--protocol", "tcp"},
		{"private-nat", "gateway", "list", "--wait", "private-nat.gateway.running"},
		{"nat", "gateway", "list", "--wait", "nat.gateway.list-running"},
	} {
		calls := 0
		err := cli.Run(natTestConfig(t, append([]string{"--yes"}, args...), nativeWireTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })))
		if err == nil || calls != 0 {
			t.Fatalf("%v calls=%d err=%v", args, calls, err)
		}
	}
}

// TestNATHelpAndCompletion exposes localized options and the same region override at both completion boundaries.
func TestNATHelpAndCompletion(t *testing.T) {
	for _, name := range []string{"nat", "private-nat"} {
		for _, args := range [][]string{{"--lang", "en-US", name, "gateway", "list", "--help"}, {"__complete", name, "gateway", "list", ""}, {"__complete", name, "gateway", "list", "--reg"}} {
			var out bytes.Buffer
			cfg := natTestConfig(t, args, nil)
			cfg.Stdout = &out
			if err := cli.Run(cfg); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "--region") {
				t.Fatalf("missing region in %v: %s", args, out.String())
			}
		}
	}
}

// natTestConfig provides isolated CLI state and fake credentials for transport-only bundle checks.
func natTestConfig(t *testing.T, args []string, transport http.RoundTripper) cli.Config {
	t.Helper()
	return cli.Config{Args: args, Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile-region"}}}`), PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }}
}

// TestNATWaiterTerminalStates checks exact selection and every documented gateway outcome through CLI polling.
func TestNATWaiterTerminalStates(t *testing.T) {
	for _, state := range []string{"running", "freeze", "expired"} {
		calls := 0
		args := []string{"--lang", "en-US", "private-nat", "gateway", "list", "--nat-gateway-id", "selected", "--wait", "private-nat.gateway.running"}
		var out bytes.Buffer
		cfg := natTestConfig(t, args, nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			body := `{"statusCode":800,"returnObj":[{"natGatewayID":"other","state":"freeze"},{"natGatewayID":"selected","state":"` + state + `"}]}`
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		}))
		cfg.Stdout = &out
		err := cli.Run(cfg)
		want := "failure"
		if state == "running" {
			want = "success"
		}
		if calls != 1 || err != nil || !strings.Contains(out.String(), want) {
			t.Fatalf("state=%s calls=%d err=%v", state, calls, err)
		}
	}
}
