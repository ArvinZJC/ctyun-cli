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
)

// TestCCECompleteDocumentBodies preserves raw manifests and top-level repair
// arrays while binding each cluster, namespace and node-pool path segment safely.
func TestCCECompleteDocumentBodies(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType, path, flag string
		args                                []string
	}{
		{"manifest", "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test\n", "text/plain", "/v2/cce/clusters/cluster%2Fone/apis/apps/v1/namespaces/team%20one/deployments", "--manifest-file", []string{"cce", "kubernetes", "deployment", "create", "cluster/one", "team one"}},
		{"repair", `["node-one","node-two"]`, "application/json", "/v2/cce/clusters/cluster%2Fone/nodepool/pool%2Fone/repair", "--nodes-file", []string{"cce", "node-pool", "repair", "cluster/one", "pool/one"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "request.txt")
			if err := os.WriteFile(file, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.EscapedPath() != tc.path || r.Header.Get("Content-Type") != tc.contentType || r.Header.Get("regionId") != "region-test" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				got, err := io.ReadAll(r.Body)
				if err != nil || string(got) != tc.body {
					t.Fatalf("document wrapped or changed: %s %v", got, err)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
			})
			args := append([]string{"--yes"}, tc.args...)
			args = append(args, "--region", "region-test", tc.flag, file)
			err := cli.Run(cli.Config{Args: args, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}
