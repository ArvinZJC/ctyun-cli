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
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestMSEAuthorizationUpdatePreservesJSONString verifies the operation-specific
// encoded rules string and keeps governance scope headers out of the body.
func TestMSEAuthorizationUpdatePreservesJSONString(t *testing.T) {
	const rules = `[{"all":true,"appIds":"test-app","black":false}]`
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/msgc/v1/auth/update" || r.Header.Get("regionId") != "region-test" || r.Header.Get("msnamespace") != "namespace-test" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		var got string
		if err := json.Unmarshal(body["authRules"], &got); err != nil || got != rules {
			t.Fatalf("rules changed: %s %v", body["authRules"], err)
		}
		for _, key := range []string{"regionId", "msnamespace"} {
			if _, ok := body[key]; ok {
				t.Fatalf("header %s leaked into body", key)
			}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":2000,"returnObj":{}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "mse", "governance", "authorization-rule", "update", "--region", "region-test", "--msnamespace", "namespace-test", "--rule-name", "test", "--service-type", "SpringCloud", "--app-type", "1", "--enable", "1", "--app-id", "test-app", "--app-name", "test", "--auth-rules", rules}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestMSEGETLifecycleSafety checks that registry GET mutations retain confirmation
// and do not inherit misleading API-retirement metadata from resource actions.
func TestMSEGETLifecycleSafety(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/mse"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, c := range b.Commands.Commands {
		if c.ID != "mse.eureka.service.deploy" && c.ID != "mse.eureka.service.undeploy" {
			continue
		}
		found++
		op := b.APIs.Operations[c.Operation]
		if op.Method != "GET" || op.Retryable || c.Dangerous.Confirm != "yes" || op.Deprecation != nil || c.Deprecation != nil {
			t.Errorf("incorrect lifecycle contract for %s", c.ID)
		}
	}
	if found != 2 {
		t.Fatalf("lifecycle commands=%d", found)
	}
}
