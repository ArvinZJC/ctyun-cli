/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestRDSMySQLRequestSafety preserves MySQL routing and GET mutation safety.
func TestRDSMySQLRequestSafety(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-mysql"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.API.CtyunProductID != 63 || bundle.Manifest.API.EndpointURL != "https://rds2-global.ctapi.ctyun.cn" {
		t.Fatal("MySQL product routing differs from the official endpoint")
	}
	foundDetail, foundRefund := false, false
	for _, command := range bundle.Commands.Commands {
		op := bundle.APIs.Operations[command.Operation]
		switch op.Path {
		case "/RDS2/v1/open-api/instance":
			foundDetail = true
			if op.Headers["regionId"] != "$profile.region" || op.Query["outerProdInstId"] == "" || !op.Retryable || op.Response == nil {
				t.Fatal("instance detail must preserve header region, identity query and explicit response contract")
			}
		case "/teledb-acceptor/v2/openapi/accept-order-info/refundOrder":
			foundRefund = true
			if op.Method != "GET" || op.Retryable || command.Dangerous.Confirm == "" {
				t.Fatal("GET unsubscription must require confirmation and must not retry")
			}
		}
	}
	if !foundDetail || !foundRefund {
		t.Fatal("missing MySQL detail or unsubscription contract")
	}
}

// TestRDSMySQLWireRegion verifies explicit and profile region bindings without cloud calls.
func TestRDSMySQLWireRegion(t *testing.T) {
	for _, override := range []bool{false, true} {
		calls := 0
		args := []string{"--output", "json", "rds-mysql", "instance", "show", "--outer-prod-inst-id", "instance-test"}
		if override {
			args = append(args, "--region", "explicit-region")
		}
		err := cli.Run(cli.Config{Args: args, Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile-region"}}}`), PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string {
			return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk", "CTYUN_REGION": "profile-region"}[k]
		}, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			want := "profile-region"
			if override {
				want = "explicit-region"
			}
			if r.Method != "GET" || r.URL.Path != "/RDS2/v1/open-api/instance" || r.Header.Get("regionId") != want || r.URL.Query().Get("outerProdInstId") != "instance-test" || r.URL.Query().Has("regionId") || r.Header.Get("Eop-Authorization") == "" {
				t.Fatalf("unexpected request %s %s %v", r.Method, r.URL, r.Header)
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":0,"returnObj":{}}`))}, nil
		})})
		if err != nil || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}

// TestRDSMySQLRetiredAPIKeepsFailureSemantics protects the published retired command without inventing success data.
func TestRDSMySQLRetiredAPIKeepsFailureSemantics(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-mysql"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Commands.Commands) != 224 || len(bundle.Waiters.Waiters) != 7 {
		t.Fatal("captured MySQL inventory changed")
	}
	id := "rds-mysql.legacy.backup.cross-region.region.list"
	found := false
	for _, cmd := range bundle.Commands.Commands {
		if cmd.ID == id {
			found = true
			if cmd.FixtureResponse != "" || bundle.APIs.Operations[id].Deprecation == nil {
				t.Fatal("retired API must retain a warning but no fabricated fixture")
			}
		}
	}
	if !found {
		t.Fatal("retired published API removed")
	}
	response := &client.HTTPResponse{Status: 200, Body: io.NopCloser(strings.NewReader(`{"statusCode":3001,"message":"There is currently no destination domain for cross-region backup","returnObj":null}`))}
	if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Response: bundle.APIs.Operations[id].Response}); err == nil {
		t.Fatal("published business error accepted as success")
	}
}

// TestRDSMySQLConditionalRequests rejects missing branch inputs without blocking valid alternative branches.
func TestRDSMySQLConditionalRequests(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		valid bool
	}{
		{"missing recovery source", []string{"recovery", "create", "--src-outer-prod-inst-id", "source-test", "--dst-outer-prod-inst-id", "destination-test"}, false},
		{"task recovery", []string{"recovery", "create", "--src-outer-prod-inst-id", "source-test", "--dst-outer-prod-inst-id", "destination-test", "--task-id", "1700129225-322"}, true},
		{"time recovery", []string{"recovery", "create", "--src-outer-prod-inst-id", "source-test", "--dst-outer-prod-inst-id", "destination-test", "--to-timepoint", "2023-12-26 14:00:00"}, true},
		{"both recovery sources", []string{"recovery", "create", "--src-outer-prod-inst-id", "source-test", "--dst-outer-prod-inst-id", "destination-test", "--task-id", "1700129225-322", "--to-timepoint", "2023-12-26 14:00:00"}, true},
		{"audit configuration", []string{"audit-log", "configuration", "update", "--outer-prod-inst-id", "instance-test", "--audit-log-retention-day", "1", "--retention-percentage", "10", "--audit-type", "0"}, false},
		{"audit retention only", []string{"audit-log", "configuration", "update", "--outer-prod-inst-id", "instance-test", "--audit-log-retention-day", "1", "--retention-percentage", "10", "--audit-type", "1"}, true},
		{"missing slow log fields", []string{"cloud-log", "configuration", "update", "--outer-prod-inst-id", "instance-test", "--log-type", "slow"}, false},
		{"slow log only", []string{"cloud-log", "configuration", "update", "--outer-prod-inst-id", "instance-test", "--log-type", "slow", "--slow-log-group", "g", "--slow-log-stream", "s", "--slow-log-group-name", "gn", "--slow-log-stream-name", "sn"}, true},
		{"error log only", []string{"cloud-log", "configuration", "update", "--outer-prod-inst-id", "instance-test", "--log-type", "error", "--error-log-group", "g", "--error-log-stream", "s", "--error-log-group-name", "gn", "--error-log-stream-name", "sn"}, true},
		{"subscription period", []string{"instance", "rebuild", "--bill-mode", "1", "--source-inst-id", "instance-test", "--count", "1"}, false},
		{"on demand", []string{"instance", "rebuild", "--bill-mode", "2", "--source-inst-id", "instance-test", "--count", "1"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := cli.Run(cli.Config{Args: append([]string{"--yes", "--output", "json", "rds-mysql"}, tc.args...), Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"region-test"}}}`), PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":500,"message":"controlled response"}`))}, nil
			})})
			if tc.valid && calls != 1 {
				t.Fatalf("valid branch rejected: %v", err)
			}
			if !tc.valid && (calls != 0 || err == nil) {
				t.Fatalf("missing branch input reached transport: calls=%d err=%v", calls, err)
			}
		})
	}
}
