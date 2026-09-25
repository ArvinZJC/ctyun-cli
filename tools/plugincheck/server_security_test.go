/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
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

// TestServerSecurityArrayBodyPreservesWireShape checks that batch alarm IDs
// are sent as the captured top-level array, without an invented object wrapper.
func TestServerSecurityArrayBodyPreservesWireShape(t *testing.T) {
	const payload = `["first-alarm","second-alarm"]`
	file := filepath.Join(t.TempDir(), "ids.json")
	if err := os.WriteFile(file, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/v1/tamperProof/alarmLog/deleteByIdList" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != payload {
			t.Fatalf("array changed: %s %v", body, err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":"200","error":"CTCSSCN_000000","returnObj":""}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "server-security", "web-protection", "alarm", "batch", "delete", "--ids-file", file}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestServerSecurityBusinessFailuresAndGETMutations rejects business failures
// even with a 200 status field and retains confirmation on GET mutations.
func TestServerSecurityBusinessFailuresAndGETMutations(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/server-security"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for id, op := range b.APIs.Operations {
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":"200","error":"CTCSSCN_000001","returnObj":{}}`))}
		if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response}); err == nil {
			t.Errorf("%s accepted business failure", id)
		}
	}
	expected := map[string]bool{"server-security.asset.collect": false, "server-security.agreement.sign": false, "server-security.baseline.policy.delete": false, "server-security.web-protection.alarm.delete": false}
	for _, c := range b.Commands.Commands {
		if _, ok := expected[c.ID]; !ok {
			continue
		}
		op := b.APIs.Operations[c.Operation]
		if op.Method != "GET" || op.Retryable || c.Dangerous.Confirm != "yes" {
			t.Errorf("%s lost GET mutation safety", c.ID)
		}
		expected[c.ID] = true
	}
	for id, found := range expected {
		if !found {
			t.Errorf("missing %s", id)
		}
	}
}
