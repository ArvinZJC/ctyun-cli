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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestRDSPostgreSQLBundle protects the captured inventory, real fixtures and unsafe GET operations.
func TestRDSPostgreSQLBundle(t *testing.T) {
	dir := repoPath(t, "plugins/rds-postgresql")
	bundle, err := plugin.LoadBundle(dir, version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Commands.Commands) != 156 || len(bundle.APIs.Operations) != 156 {
		t.Fatal("captured PostgreSQL inventory changed")
	}
	if bundle.Manifest.API.CtyunProductID != 65 || bundle.Manifest.API.SourceRevision != "67" || bundle.Manifest.API.EndpointURL != "https://pgsql-global.ctapi.ctyun.cn" {
		t.Fatal("PostgreSQL provenance changed")
	}
	for _, command := range bundle.Commands.Commands {
		op := bundle.APIs.Operations[command.Operation]
		if command.ID == "rds-postgresql.order.convert-to-demand" || command.ID == "rds-postgresql.order.convert-to-package" || command.ID == "rds-postgresql.order.recover" {
			if command.FixtureResponse != "" || op.Retryable || command.Dangerous.Confirm != "yes" {
				t.Fatal("new order mutations must require confirmation without retry or fabricated fixtures")
			}
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, command.FixtureResponse))
		if err != nil {
			t.Fatal(err)
		}
		fixture, err := client.DecodeFixture(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.DecodeHTTPResponse(fixture, client.RequestSpec{Method: op.Method, Response: op.Response}); err != nil {
			t.Fatalf("%s: %v", command.ID, err)
		}
		switch command.ID {
		case "rds-postgresql.order.renew", "rds-postgresql.order.refund", "rds-postgresql.order.unsubscribe", "rds-postgresql.node.start", "rds-postgresql.node.stop":
			if op.Method != "GET" || op.Retryable || command.Dangerous.Confirm == "" {
				t.Fatalf("unsafe GET mutation %s", command.ID)
			}
		}
	}
	for _, id := range []string{"rds-postgresql.instance.running", "rds-postgresql.order.running"} {
		spec := bundle.Waiters.Waiters[id]
		states := []struct {
			value any
			want  waiter.State
		}{{nil, waiter.Pending}, {json.Number("999999"), waiter.Pending}}
		if strings.Contains(id, ".instance.") {
			states = append(states, struct {
				value any
				want  waiter.State
			}{json.Number("0"), waiter.Success}, struct {
				value any
				want  waiter.State
			}{json.Number("1006"), waiter.Failure}, struct {
				value any
				want  waiter.State
			}{json.Number("2005"), waiter.Pending})
		} else {
			states = append(states, struct {
				value any
				want  waiter.State
			}{json.Number("2"), waiter.Success}, struct {
				value any
				want  waiter.State
			}{json.Number("1"), waiter.Failure})
		}
		for _, tc := range states {
			got, err := waiter.Evaluate(waiter.Spec{Path: "state", Success: spec.Success, FailureValues: spec.FailureValues}, map[string]any{"state": tc.value})
			if err != nil || got != tc.want {
				t.Fatalf("%s value %v: %s %v", id, tc.value, got, err)
			}
		}
	}
	for _, op := range []string{"rds-postgresql.instance.show", "rds-postgresql.order.status.show", "rds-postgresql.backup.cross-region.destination.list"} {
		response := &client.HTTPResponse{Status: 200, Body: io.NopCloser(strings.NewReader(`{"statusCode":500,"code":500,"success":false,"message":"failure"}`))}
		if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Response: bundle.APIs.Operations[op].Response}); err == nil {
			t.Fatalf("%s accepted an error envelope", op)
		}
	}
}

// TestRDSPostgreSQLRequestBindings verifies EOP region headers and numeric JSON body values.
func TestRDSPostgreSQLRequestBindings(t *testing.T) {
	for _, tc := range []struct {
		args         []string
		path, method string
	}{
		{[]string{"instance", "show", "--prod-inst-id", "instance-test", "--obtain-status", "true"}, "/PG/v1/product/get-paas-product", "GET"},
		{[]string{"node", "switch", "--prod-inst-id", "instance-test", "--node-id", "25"}, "/PG/v1/node/switch", "POST"},
	} {
		called := false
		var stdout, stderr bytes.Buffer
		args := append([]string{"--yes", "--lang", "en-US", "--output", "json", "rds-postgresql"}, tc.args...)
		args = append(args, "--region", "region-test", "--header-project-id", "project-test")
		err := cli.Run(cli.Config{Args: args, Stdout: &stdout, Stderr: &stderr, PluginRoot: t.TempDir(), Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "test-ak"
			case "CTYUN_SK":
				return "test-sk"
			}
			return ""
		}, HTTPTransport: nativeWireTransport(func(req *http.Request) (*http.Response, error) {
			called = true
			if req.Method != tc.method || req.URL.Path != tc.path || req.Header.Get("regionId") != "region-test" || req.Header.Get("Project-Id") != "project-test" {
				t.Fatalf("unexpected request: %s %s headers=%v", req.Method, req.URL, req.Header)
			}
			if req.Method == "GET" {
				if req.URL.Query().Get("prodInstId") != "instance-test" || req.URL.Query().Get("obtainStatus") != "true" {
					t.Fatal(req.URL.RawQuery)
				}
			} else {
				var body map[string]any
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["prodInstId"] != "instance-test" || body["nodeId"] != float64(25) {
					t.Fatalf("unexpected body %#v", body)
				}
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"message":"SUCCESS","returnObj":{}}`))}, nil
		})})
		if err != nil || !called || !json.Valid(stdout.Bytes()) {
			t.Fatalf("called=%t err=%v stderr=%s stdout=%s", called, err, stderr.String(), stdout.String())
		}
	}
}

// TestRDSPostgreSQLNestedSuccess rejects application failures inside successful platform envelopes.
func TestRDSPostgreSQLNestedSuccess(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-postgresql"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ command, field string }{
		{"order.price.show", "isSucceed"}, {"order.resize-price.show", "isSucceed"}, {"order.renewal-price.show", "isSucceed"},
		{"order.renew", "submitted"}, {"order.refund", "submitted"}, {"order.unsubscribe", "submitted"},
	} {
		for _, value := range []string{"false", "null"} {
			body := `{"statusCode":200,"returnObj":{"data":{"` + tc.field + `":` + value + `}}}`
			response := &client.HTTPResponse{Status: 200, Body: io.NopCloser(strings.NewReader(body))}
			op := bundle.APIs.Operations["rds-postgresql."+tc.command]
			if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Response: op.Response}); err == nil {
				t.Errorf("%s accepted %s=%s", tc.command, tc.field, value)
			}
		}
	}
}

// TestRDSPostgreSQLConditionalInputs rejects incomplete documented restore and database requests before transport.
func TestRDSPostgreSQLConditionalInputs(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/rds-postgresql"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		command string
		values  map[string]string
		extra   []string
		missing string
	}{
		{"database.create", map[string]string{"db-encoding": "LATIN1"}, nil, "db-collate"},
		{"database.create", map[string]string{"db-encoding": "LATIN1"}, []string{"--db-collate", "C"}, "db-ctype"},
		{"order.create", nil, []string{"--cross-instance-backup", "true"}, "source-inst-id"},
		{"order.create", nil, []string{"--cross-instance-backup", "true", "--source-inst-id", "source"}, "backup-id"},
		{"order.create", nil, []string{"--is-cross-region-recovery", "true", "--source-inst-id", "source"}, "backup-id"},
		{"audit.policy.update", map[string]string{"sql-collector-status": "invalid"}, nil, "sql-collector-status"},
		{"audit.policy.update", map[string]string{"sql-collector-status": "enable"}, []string{"--log-interval", "7"}, "log-interval"},
		{"monitor.log.purge", nil, []string{"--purge", "3"}, "purge"},
		{"monitor.log.purge", nil, nil, "log-time"},
	} {
		t.Run(tc.command+"/"+tc.missing, func(t *testing.T) {
			var command plugin.Command
			for _, c := range bundle.Commands.Commands {
				if c.ID == "rds-postgresql."+tc.command {
					command = c
				}
			}
			args := append([]string{}, command.Path...)
			published := commandSmokeArgs(t, command)
			for i := len(command.Path); i+1 < len(published); i += 2 {
				for _, parameter := range command.Parameters {
					if parameter.Required && published[i] == "--"+parameter.Flag {
						args = append(args, published[i:i+2]...)
					}
				}
			}
			for flag, value := range tc.values {
				for i := 0; i < len(args)-1; i++ {
					if args[i] == "--"+flag {
						args[i+1] = value
					}
				}
			}
			args = append(args, tc.extra...)
			args = append(args, "--region", "test", "--yes", "--lang", "en-US")
			called := false
			var stdout, stderr bytes.Buffer
			err := cli.Run(cli.Config{Args: args, Stdout: &stdout, Stderr: &stderr, PluginRoot: t.TempDir(), Env: func(key string) string {
				if key == "CTYUN_AK" || key == "CTYUN_SK" {
					return "test"
				}
				return ""
			}, HTTPTransport: nativeWireTransport(func(*http.Request) (*http.Response, error) {
				called = true
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"statusCode":800}`))}, nil
			})})
			if err == nil || called {
				t.Fatalf("incomplete %s reached transport=%t err=%v", tc.command, called, err)
			}
			var text bytes.Buffer
			text.WriteString(stderr.String())
			text.WriteString(err.Error())
			if !strings.Contains(text.String(), tc.missing) {
				t.Fatalf("missing diagnostic for %s: %s", tc.missing, text.String())
			}
		})
	}
}
