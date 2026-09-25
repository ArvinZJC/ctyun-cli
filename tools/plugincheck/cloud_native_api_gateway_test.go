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
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestCloudNativeGatewayCertificateMultipart checks the documented form part
// names and keeps the regional header separate from uploaded certificate data.
func TestCloudNativeGatewayCertificateMultipart(t *testing.T) {
	file := filepath.Join(t.TempDir(), "certificate.pem")
	if err := os.WriteFile(file, []byte("test-certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/agw/v1/host/create-cert" || r.Header.Get("regionId") != "test-region" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				t.Errorf("remove multipart temporary files: %v", err)
			}
		}()
		f, _, err := r.FormFile("certFile")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				t.Errorf("close certificate form file: %v", err)
			}
		}()
		data, err := io.ReadAll(f)
		if err != nil || string(data) != "test-certificate" {
			t.Fatalf("certificate changed: %s %v", data, err)
		}
		if r.FormValue("certName") != "test-cert" || r.FormValue("regionId") != "" {
			t.Fatalf("form bindings changed: %v", r.MultipartForm.Value)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"error":"AGW_1000","returnObj":{}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "cloud-native-api-gateway", "certificate", "create", "--region", "test-region", "--cert-file", file, "--cert-name", "test-cert"}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
		return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k]
	}})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestCloudNativeGatewayDebugSafety preserves confirmation for backend calls
// and distinguishes resource undeployment from upstream API retirement.
func TestCloudNativeGatewayDebugSafety(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/cloud-native-api-gateway"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, c := range b.Commands.Commands {
		if strings.HasSuffix(c.ID, ".debug") {
			found++
			if b.APIs.Operations[c.Operation].Retryable || c.Dangerous.Confirm != "yes" {
				t.Errorf("unsafe debug command %s", c.ID)
			}
		}
		if strings.HasSuffix(c.ID, ".undeploy") && c.Deprecation != nil {
			t.Errorf("resource action marked deprecated: %s", c.ID)
		}
	}
	if found != 2 {
		t.Fatalf("debug commands=%d", found)
	}
}
