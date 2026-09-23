/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
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

// TestRDSSQLServerCapturedSurface pins the SQL Server service boundary and actual fixtures.
func TestRDSSQLServerCapturedSurface(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-sqlserver"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Commands.Commands) != 114 || len(bundle.APIs.Operations) != 114 || len(bundle.Waiters.Waiters) != 6 {
		t.Fatalf("captured surface: commands=%d operations=%d waiters=%d", len(bundle.Commands.Commands), len(bundle.APIs.Operations), len(bundle.Waiters.Waiters))
	}
	if bundle.Manifest.API.CtyunProductID != 153 || bundle.Manifest.API.SourceRevision != "284" || bundle.Manifest.API.EndpointURL != "https://ctsqlserver-global.ctapi.ctyun.cn" {
		t.Fatalf("incorrect upstream identity: %#v", bundle.Manifest.API)
	}
	for _, command := range bundle.Commands.Commands {
		operation := bundle.APIs.Operations[command.Operation]
		if operation.Path == "/sqlserver/api/v1/xevent/log_download" {
			if !command.Download || command.FixtureResponse != "" || operation.Response == nil || operation.Response.Variants[0].Format != "binary" {
				t.Fatal("documented download must use binary transport without a fabricated fixture")
			}
			continue
		}
		if operation.Path == "/teledb-acceptor/v2/openapi/accept-order-info/refundOrder" && (operation.Retryable || command.Dangerous.Confirm != "yes") {
			t.Fatal("GET refund must require confirmation and must not retry")
		}
		data, err := os.ReadFile(repoPath(t, filepath.Join("plugins/rds-sqlserver", command.FixtureResponse)))
		if err != nil {
			t.Fatal(err)
		}
		if operation.Response != nil {
			response, err := client.DecodeFixture(data)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = client.DecodeHTTPResponse(response, client.RequestSpec{Method: operation.Method, Response: operation.Response}); err != nil {
				t.Fatalf("%s: %v", command.ID, err)
			}
		} else if _, err := client.DecodeResponse(data); err != nil {
			t.Fatalf("%s: %v", command.ID, err)
		}
	}
	status := bundle.Waiters.Waiters["rds-sqlserver.instance.running"]
	if status.Path != "returnObj" || status.Success != "1" {
		t.Fatalf("scalar instance status waiter: %#v", status)
	}
	applied := bundle.Waiters.Waiters["rds-sqlserver.parameter-template.applied"]
	if applied.Selector == nil || applied.Selector.Key != "taskId" || applied.Selector.Value != "$param.task_id" {
		t.Fatalf("task selector: %#v", applied.Selector)
	}
}

// TestRDSSQLServerHeaderRegionOnWire verifies header region resolution through the public command engine.
func TestRDSSQLServerHeaderRegionOnWire(t *testing.T) {
	for _, override := range []bool{false, true} {
		calls := 0
		transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			want := "profile-region"
			if override {
				want = "explicit-region"
			}
			if r.Method != "GET" || r.URL.Path != "/sqlserver/api/v1/instances" || r.Header.Get("regionId") != want || r.URL.Query().Get("pageNum") != "2" || r.URL.Query().Has("regionId") {
				t.Fatalf("request: %s %s %#v", r.Method, r.URL, r.Header)
			}
			if r.Header.Get("Eop-Authorization") == "" {
				t.Fatal("request is unsigned")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":[]}`))}, nil
		})
		args := []string{"--output", "json", "rds-sqlserver", "instance", "list", "--page-num", "2"}
		if override {
			args = append(args, "--region", "explicit-region")
		}
		var out bytes.Buffer
		err := cli.Run(cli.Config{Args: args, Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile-region"}}}`), Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
			return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk", "CTYUN_REGION": "profile-region"}[k]
		}})
		if err != nil || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}

// TestRDSSQLServerPriceBusinessFailureRejectsHTTP200 protects the nested pricing success contract.
func TestRDSSQLServerPriceBusinessFailureRejectsHTTP200(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-sqlserver"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"v1.rds-sqlserver.price.create", "v1.rds-sqlserver.price.change", "v1.rds-sqlserver.price.renew"} {
		operation := bundle.APIs.Operations[id]
		for _, body := range []string{`{"statusCode":200,"returnObj":{"data":{"isSucceed":false}}}`, `{"statusCode":500,"returnObj":{"data":{"isSucceed":true}}}`} {
			response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
			if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: operation.Method, Response: operation.Response}); err == nil {
				t.Fatalf("%s accepted business failure", id)
			}
		}
	}
}

// TestRDSSQLServerRecoveryRequirementsRejectBeforeRequests checks documented recovery prerequisites.
func TestRDSSQLServerRecoveryRequirementsRejectBeforeRequests(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-sqlserver"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		command, policy, missing string
		recycle                  string
	}{
		{"v1.rds-sqlserver.recovery.create", "backupset", "src-backup-id", ""},
		{"v1.rds-sqlserver.recovery.create", "time", "restore-time", ""},
		{"v1.rds-sqlserver.recovery.same-region.create", "", "restore-policy", ""},
		{"v1.rds-sqlserver.recovery.same-region.create", "", "restore-policy", "false"},
		{"v1.rds-sqlserver.recovery.same-region.create", "backupset", "db-list", "false"},
		{"v1.rds-sqlserver.recovery.same-region.create", "backupset", "src-backup-id", "false"},
		{"v1.rds-sqlserver.recovery.same-region.create", "time", "restore-time", "false"},
		{"v1.rds-sqlserver.recovery.same-region.create", "time", "restore-time", ""},
	}
	for _, tc := range cases {
		t.Run(tc.command+"/"+tc.policy+"/"+tc.missing+"/"+tc.recycle, func(t *testing.T) {
			var command plugin.Command
			for _, candidate := range bundle.Commands.Commands {
				if candidate.ID == tc.command {
					command = candidate
				}
			}
			args := commandSmokeArgs(t, command)
			for _, flag := range []string{"restore-policy", "src-backup-id", "restore-time", "db-list", "recycle-bin-restore"} {
				for i := 0; i < len(args); i++ {
					if args[i] == "--"+flag {
						args = append(args[:i], args[i+2:]...)
						i--
					}
				}
			}
			if tc.policy != "" {
				args = append(args, "--restore-policy", tc.policy)
			}
			if tc.missing != "db-list" {
				args = append(args, "--db-list", `[{"srcDbName":"test","dstDbName":"test"}]`)
			}
			if tc.command == "v1.rds-sqlserver.recovery.same-region.create" {
				args = append(args, "--subnet-id", "test-subnet", "--security-group-id", "test-security-group")
			}
			// Supply the opposite recovery input to ensure policy-specific validation,
			// rather than the former loose any-of rule, rejects the request.
			if tc.command == "v1.rds-sqlserver.recovery.same-region.create" {
				if tc.missing == "src-backup-id" {
					args = append(args, "--restore-time", "2026-09-21 01:00:00")
				}
				if tc.missing == "restore-time" {
					args = append(args, "--src-backup-id", "5919")
				}
			}
			if tc.recycle != "" {
				args = append(args, "--recycle-bin-restore", tc.recycle)
			}
			calls := 0
			transport := nativeWireTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			args = append([]string{"--yes", "--lang", "en-US"}, args...)
			args = append(args, "--region", "test-region")
			err := cli.Run(cli.Config{Args: args, Config: []byte(`{}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
			if err == nil || calls != 0 || !strings.Contains(err.Error(), "--"+tc.missing) {
				t.Fatalf("calls=%d err=%v; expected missing --%s", calls, err, tc.missing)
			}
		})
	}
}

// TestRDSSQLServerRecycleRestoreAllowsInheritedNetwork preserves the documented recycle-bin exception.
func TestRDSSQLServerRecycleRestoreAllowsInheritedNetwork(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-sqlserver"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	var command plugin.Command
	for _, candidate := range bundle.Commands.Commands {
		if candidate.ID == "v1.rds-sqlserver.recovery.same-region.create" {
			command = candidate
		}
	}
	args := commandSmokeArgs(t, command)
	for _, flag := range []string{"recycle-bin-restore", "restore-policy", "src-backup-id", "restore-time", "db-list", "subnet-id", "security-group-id"} {
		for i := 0; i < len(args); i++ {
			if args[i] == "--"+flag {
				args = append(args[:i], args[i+2:]...)
				i--
			}
		}
	}
	args = append([]string{"--yes", "--output", "json"}, args...)
	args = append(args, "--region", "test-region", "--recycle-bin-restore", "true", "--restore-policy", "time")
	calls := 0
	transport := nativeWireTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":200,"returnObj":{"data":{"newOrderId":"test-order"}}}`))}, nil
	})
	err = cli.Run(cli.Config{Args: args, Config: []byte(`{}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestRDSSQLServerReportDownload checks the documented stream contract without claiming a captured binary fixture.
func TestRDSSQLServerReportDownload(t *testing.T) {
	payload := []byte{0, 255, 'X', 'E', 'L', '\n'}
	for _, tc := range []struct {
		name, contentType string
		body              []byte
		wantError         bool
	}{
		{"binary", "application/octet-stream", payload, false},
		{"binary with parameters", "application/octet-stream; charset=binary", payload, false},
		{"business error", "application/json", []byte(`{"statusCode":500,"message":"failed"}`), true},
		{"missing media type", "", payload, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.xel")
			calls := 0
			transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/sqlserver/api/v1/xevent/log_download" || r.Header.Get("regionId") != "test-region" || r.Header.Get("Eop-Authorization") == "" || r.URL.Query().Get("eventType") != "block" || r.URL.Query().Get("filename") != "report.xel" || r.URL.Query().Get("prodInstId") != "test-instance" {
					t.Fatalf("incorrect download request: %s %s", r.Method, r.URL)
				}
				header := http.Header{}
				if tc.contentType != "" {
					header.Set("Content-Type", tc.contentType)
				}
				return &http.Response{StatusCode: 200, Header: header, Body: io.NopCloser(bytes.NewReader(tc.body))}, nil
			})
			err := cli.Run(cli.Config{Args: []string{"rds-sqlserver", "log", "event", "report", "download", "--prod-inst-id", "test-instance", "--event-type", "block", "--filename", "report.xel", "--region", "test-region", "--output-file", path}, Config: []byte(`{}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
			if (err != nil) != tc.wantError || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			data, readErr := os.ReadFile(path)
			if tc.wantError {
				if readErr == nil {
					t.Fatal("invalid response created a download file")
				}
			} else if readErr != nil || !bytes.Equal(data, payload) {
				t.Fatalf("download changed bytes: %v", readErr)
			}
		})
	}
}
