/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestRDSNewOrderCommands verifies the published state-changing GET contract without live cloud calls.
func TestRDSNewOrderCommands(t *testing.T) {
	for _, tc := range []struct{ plugin, action, endpoint, headerFlag string }{
		{"rds-postgresql", "convert-to-demand", "orderToDemand", "header-project-id"},
		{"rds-postgresql", "convert-to-package", "orderToPackage", "header-project-id"},
		{"rds-postgresql", "recover", "recoverOrder", ""},
		{"rds-sqlserver", "convert-to-demand", "orderToDemand", "project-id"},
		{"rds-sqlserver", "convert-to-package", "orderToPackage", "project-id"},
	} {
		t.Run(tc.plugin+"/"+tc.action, func(t *testing.T) {
			path := []string{tc.plugin, "order", tc.action}
			base := append(append([]string{}, path...), "--inst-id", "test-instance")
			if tc.headerFlag != "" {
				base = append(base, "--"+tc.headerFlag, "test-project")
			}
			if tc.action != "convert-to-demand" {
				base = append(base, "--month", "24")
			}
			for _, mode := range []string{"wire", "business-failure", "http-failure", "confirmation", "offline", "bad-month", "missing-instance", "missing-project", "missing-month", "help", "completion"} {
				t.Run(mode, func(t *testing.T) {
					if mode == "bad-month" && tc.action == "convert-to-demand" {
						return
					}
					if mode == "missing-project" && tc.headerFlag == "" || mode == "missing-month" && tc.action != "convert-to-package" {
						return
					}
					args := append([]string{}, base...)
					missingFlag := map[string]string{"missing-instance": "--inst-id", "missing-project": "--" + tc.headerFlag, "missing-month": "--month"}[mode]
					if missingFlag != "" {
						for i, arg := range args {
							if arg == missingFlag {
								args = append(args[:i], args[i+2:]...)
								break
							}
						}
					}
					if mode != "confirmation" {
						args = append(args, "--yes")
					}
					if mode == "offline" {
						args = append(args, "--offline")
					}
					if mode == "bad-month" {
						for i := range args {
							if args[i] == "24" {
								args[i] = "13"
							}
						}
					}
					if mode == "help" {
						args = append(append([]string{"--lang", "en-US"}, path...), "--help")
					}
					if mode == "completion" {
						args = append(append([]string{"__complete"}, path...), "--")
					}
					calls := 0
					transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.Method != "GET" || r.URL.Path != "/teledb-acceptor/v2/openapi/accept-order-info/"+tc.endpoint || r.URL.Query().Get("instId") != "test-instance" || r.URL.Query().Has("project-id") {
							t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
						}
						if tc.headerFlag != "" && r.Header.Get("project-id") != "test-project" {
							t.Fatal("missing project header")
						}
						if tc.action != "convert-to-demand" && r.URL.Query().Get("month") != "24" {
							t.Fatal("missing month query")
						}
						status := 200
						body := `{"statusCode":200,"returnObj":{"newOrderId":"test-order"}}`
						if mode == "business-failure" {
							body = `{"statusCode":500,"message":"test rejection"}`
						}
						if mode == "http-failure" {
							status = 503
							body = `{"statusCode":503,"message":"unavailable"}`
						}
						return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
					})
					var out bytes.Buffer
					err := cli.Run(cli.Config{Args: args, Stdin: strings.NewReader("no\n"), Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
					if mode == "wire" || mode == "help" || mode == "completion" {
						if err != nil {
							t.Fatal(err)
						}
					} else if err == nil {
						t.Fatal("expected rejection")
					}
					wantCalls := 0
					if mode == "wire" || mode == "business-failure" || mode == "http-failure" {
						wantCalls = 1
					}
					if calls != wantCalls {
						t.Fatalf("calls=%d want=%d", calls, wantCalls)
					}
					if mode == "wire" && !strings.Contains(out.String(), "test-order") {
						t.Fatal("missing new order result")
					}
					if (mode == "help" || mode == "completion") && !strings.Contains(out.String(), "--inst-id") {
						t.Fatal("missing instance option")
					}
					if mode == "offline" && !strings.Contains(err.Error(), "fixture") {
						t.Fatalf("unexpected offline diagnostic: %v", err)
					}
				})
			}
		})
	}
}

// TestRDSPostgreSQLRecoveryDefault preserves the documented optional month default and prepaid applicability.
func TestRDSPostgreSQLRecoveryDefault(t *testing.T) {
	for _, tc := range []struct{ locale, applicability, defaultLabel string }{
		{"en-US", "recovered prepaid instance", "default: 1"},
		{"en-GB", "recovered prepaid instance", "default: 1"},
		{"zh-CN", "恢复包年包月实例", "默认：1"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			var out bytes.Buffer
			err := cli.Run(cli.Config{Args: []string{"--lang", tc.locale, "rds-postgresql", "order", "recover", "--help"}, Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{tc.applicability, tc.defaultLabel, "[--month <"} {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("help missing %q: %s", want, out.String())
				}
			}
		})
	}
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/teledb-acceptor/v2/openapi/accept-order-info/recoverOrder" || r.URL.Query().Get("instId") != "test-instance" || r.URL.Query().Has("month") {
			t.Fatalf("omitted month must preserve the upstream default: %s %s", r.Method, r.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":200,"returnObj":{"newOrderId":"test-order"}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"rds-postgresql", "order", "recover", "--inst-id", "test-instance", "--yes"}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
		return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k]
	}})
	if err != nil || calls != 1 {
		t.Fatalf("omitted month: calls=%d err=%v", calls, err)
	}
}
