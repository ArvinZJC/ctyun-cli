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

// TestMSAPReleasePreservesBatchGroups verifies that release batches remain
// nested arrays and the regional header does not leak into the JSON payload.
func TestMSAPReleasePreservesBatchGroups(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/v2/plan/addPlan" || r.Header.Get("regionId") != "region-test" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["regionId"]; ok {
			t.Fatal("regional header leaked into body")
		}
		var batches [][]map[string]string
		if err := json.Unmarshal(body["list"], &batches); err != nil {
			t.Fatal(err)
		}
		if len(batches) != 2 || len(batches[0]) != 1 || len(batches[1]) != 1 || batches[0][0]["appDeployUuid"] != "first" || batches[1][0]["appDeployUuid"] != "second" {
			t.Fatalf("batch grouping changed: %v", batches)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":2000,"returnObj":{}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "msap", "release", "create", "--region", "region-test", "--name", "test", "--cluster-type", "ECS", "--env-uuid", "environment", "--deploy-strategy", `{}`, "--deploy-unit-list", `["unit"]`, "--list", `[[{"appDeployUuid":"first"}],[{"appDeployUuid":"second"}]]`}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestMSAPResourceRetirementIsNotAPIDeprecation keeps technology-stack
// retirement callable without a misleading obsolete-API warning.
func TestMSAPResourceRetirementIsNotAPIDeprecation(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/msap"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	op, ok := b.APIs.Operations["msap.technology-stack.retire"]
	if !ok || op.Deprecation != nil || op.Retryable {
		t.Fatalf("incorrect retirement metadata: %#v", op)
	}
}
