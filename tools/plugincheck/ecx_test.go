/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestECXIPv6EnvelopeException preserves the v3 operation's explicitly documented
// legacy envelope instead of guessing its response from the URI generation.
func TestECXIPv6EnvelopeException(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/ecx"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	op := b.APIs.Operations["ecx.subnet.ipv6.update"]
	for i, body := range []string{`{"status":{"code":"Success"},"data":{"cidrV6":"240e::/96"}}`, `{"status":{"code":"Failure"},"data":{}}`, `{"statusCode":200,"data":{}}`} {
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
		_, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
		if (err == nil) != (i == 0) {
			t.Errorf("response %d: %v", i, err)
		}
	}
}

// TestECXSSHLoginDoesNotRequirePassword checks the documented login alternatives
// while ensuring password login still fails locally before issuing a request.
func TestECXSSHLoginDoesNotRequirePassword(t *testing.T) {
	for _, mode := range []string{"SSH", "PASSWD"} {
		calls := 0
		transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if _, ok := body["password"]; ok {
				t.Fatal("invented SSH password")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"status":{"code":"Success"},"data":{"id":"test-instance"}}`))}, nil
		})
		err := cli.Run(cli.Config{Args: []string{"--yes", "ecx", "v1", "instance", "create", "--node-code", "test-node", "--login-type", mode, "--network-infos", `[{"networkType":"DIRECT"}]`, "--image-id", "1", "--instance-type", "1", "--ssh-key-content", `["test-public-key"]`}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
		if mode == "SSH" && (err != nil || calls != 1) {
			t.Fatalf("SSH calls=%d err=%v", calls, err)
		}
		if mode == "PASSWD" && (err == nil || calls != 0) {
			t.Fatalf("PASSWD calls=%d err=%v", calls, err)
		}
	}
}
