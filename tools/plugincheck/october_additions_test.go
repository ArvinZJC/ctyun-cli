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
	"reflect"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestOctoberAdditionRequests exercises the published methods and field bindings
// so missing commands and incorrectly encoded renewal or resource inputs fail.
func TestOctoberAdditionRequests(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, response string
		args                         []string
		body                         map[string]any
	}{
		{"delegation", "GET", "/v4/cthpc/check-delegate-exists", `{"statusCode":800,"returnObj":{"isExist":false}}`,
			[]string{"ehpc", "delegate", "exists"}, nil},
		{"renewal", "POST", "/v4/ebm/batch-renew", `{"statusCode":800,"returnObj":{}}`,
			[]string{"--yes", "dps", "instance", "renew-batch", "--az-name", "az2", "--instance-uuid-list", "first,second", "--cycle-type", "MONTH", "--cycle-count", "3", "--client-token", "unique-token", "--pay-voucher-price", "20.55"},
			map[string]any{"regionID": "profile-region", "azName": "az2", "instanceUUIDList": "first,second", "cycleType": "MONTH", "cycleCount": float64(3), "clientToken": "unique-token", "payVoucherPrice": 20.55}},
		{"order resources", "POST", "/api/bc/v2/listResourceId", `{"statusCode":"200","returnObj":{"status":{"code":"ok"},"orderResourceInfos":[{"chargingItem":"CPU","orderResourceId":"resource"}]}}`,
			[]string{"research-assistant", "order", "resource", "list", "--object-type", "ide", "--object-uuid", "resource-object"},
			map[string]any{"objectType": "ide", "objectUuid": "resource-object"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			args := append([]string{"--output", "json"}, tc.args...)
			err := cli.Run(cli.Config{
				Args: args, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(),
				Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile-region"}}}`),
				Env:    func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] },
				HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != tc.method || r.URL.Path != tc.path || r.Header.Get("Eop-Authorization") == "" {
						t.Fatalf("unexpected signed request: %s %s", r.Method, r.URL)
					}
					if r.URL.RawQuery != "" {
						t.Fatalf("unexpected query: %s", r.URL.RawQuery)
					}
					if tc.body != nil {
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(body, tc.body) {
							t.Fatalf("body = %#v, want %#v", body, tc.body)
						}
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.response))}, nil
				}),
			})
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}

// TestOctoberAdditionSafetyAndValidation prevents repeated billing mutations
// and rejects missing or unsupported inputs before reaching the HTTP transport.
func TestOctoberAdditionSafetyAndValidation(t *testing.T) {
	for _, tc := range []struct {
		product, operation   string
		retryable, dangerous bool
	}{
		{"ehpc", "v4.ehpc.delegate.exists", true, false},
		{"dps", "v4.dps.instance.renew-batch", false, true},
		{"research-assistant", "research-assistant.order.resource.list", true, false},
	} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.product), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		op, ok := bundle.APIs.Operations[tc.operation]
		if !ok {
			t.Errorf("missing operation %s", tc.operation)
			continue
		}
		if op.Retryable != tc.retryable {
			t.Errorf("%s retryable=%v", tc.operation, op.Retryable)
		}
		found := false
		for _, command := range bundle.Commands.Commands {
			if command.Operation == tc.operation {
				found = true
				if (command.Dangerous.Confirm != "") != tc.dangerous {
					t.Errorf("%s confirmation=%q", tc.operation, command.Dangerous.Confirm)
				}
			}
		}
		if !found {
			t.Errorf("missing command for %s", tc.operation)
		}
	}

	for _, args := range [][]string{
		{"dps", "instance", "renew-batch", "--region", "region", "--az-name", "az2", "--instance-uuid-list", "first,second", "--cycle-type", "MONTH", "--cycle-count", "3", "--client-token", "token"},
		{"--yes", "dps", "instance", "renew-batch", "--region", "region", "--az-name", "az2", "--instance-uuid-list", "first", "--cycle-type", "WEEK", "--cycle-count", "3", "--client-token", "token"},
		{"--yes", "dps", "instance", "renew-batch", "--region", "region", "--az-name", "az2", "--instance-uuid-list", "first", "--cycle-type", "MONTH", "--cycle-count", "3"},
		{"research-assistant", "order", "resource", "list", "--object-type", "unsupported", "--object-uuid", "object"},
		{"research-assistant", "order", "resource", "list", "--object-type", "ide"},
	} {
		calls := 0
		err := cli.Run(cli.Config{
			Args: args, Stdout: io.Discard, Stderr: io.Discard, Stdin: strings.NewReader("no\n"), PluginRoot: t.TempDir(),
			Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] },
			HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return nil, nil
			}),
		})
		if err == nil || calls != 0 {
			t.Fatalf("invalid/unconfirmed %v: calls=%d err=%v", args, calls, err)
		}
	}
}

// TestOctoberResearchOrderResponseGuard rejects bare success and failed business
// envelopes while accepting a documented empty list without invented rows.
func TestOctoberResearchOrderResponseGuard(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/research-assistant"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	op, ok := bundle.APIs.Operations["research-assistant.order.resource.list"]
	if !ok {
		t.Fatal("missing order resource operation")
	}
	for _, tc := range []struct {
		body    string
		success bool
	}{
		{`{"statusCode":"200","returnObj":{"status":{"code":"ok"},"orderResourceInfos":[]}}`, true},
		{`{"statusCode":"200"}`, false},
		{`{"statusCode":"200","returnObj":{"status":{"code":"failed"},"orderResourceInfos":[]}}`, false},
		{`{"statusCode":"400","returnObj":{"status":{"code":"ok"},"orderResourceInfos":[]}}`, false},
	} {
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body))}
		_, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
		if (err == nil) != tc.success {
			t.Fatalf("success=%v err=%v for %s", tc.success, err, tc.body)
		}
	}
}

// TestOctoberAdditionFixtureTables verifies the captured business fields render
// and object-type completion exposes the same documented options for prefixes.
func TestOctoberAdditionFixtureTables(t *testing.T) {
	for _, tc := range []struct {
		args   []string
		values []string
	}{
		{[]string{"ehpc", "delegate", "exists", "--offline"}, []string{"true"}},
		{[]string{"research-assistant", "order", "resource", "list", "--object-type", "ide", "--object-uuid", "07647aa3axxxx6a91320cf122ad9a83", "--offline"}, []string{"CPU", "MEM", "1dc1f8cxxxx3409f9910d997cd761688"}},
		{[]string{"__complete", "research-assistant", "order", "resource", "list", "--object-type", ""}, []string{"ide", "trainingJob", "storageResearch"}},
		{[]string{"__complete", "research-assistant", "order", "resource", "list", "--object-type", "training"}, []string{"trainingJob"}},
	} {
		var out bytes.Buffer
		args := append([]string{"--lang", "en-US", "--table", "plain"}, tc.args...)
		if tc.args[0] == "__complete" {
			args = tc.args
		}
		err := cli.Run(cli.Config{Args: args, Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir()})
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		for _, value := range tc.values {
			if !strings.Contains(out.String(), value) {
				t.Fatalf("%v missing %q: %s", args, value, out.String())
			}
		}
	}
}
