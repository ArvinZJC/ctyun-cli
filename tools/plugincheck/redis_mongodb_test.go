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

// TestRedisMongoDBRegionOnWire catches misplaced region headers and pagination bindings.
func TestRedisMongoDBRegionOnWire(t *testing.T) {
	for _, tc := range []struct{ name, path, page string }{
		{"redis", "/v3/instanceManageMgrServant/describeInstances", "pageIndex"},
		{"mongodb", "/DDS2/v2/openApi/getAllInstances", "pageNow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, override := range []bool{false, true} {
				calls := 0
				transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					want := "profile-region"
					if override {
						want = "explicit-region"
					}
					if r.Method != "GET" || r.URL.Path != tc.path || r.Header.Get("regionId") != want || r.URL.Query().Get(tc.page) != "2" || r.URL.Query().Has("regionId") {
						t.Fatalf("incorrect request: %s %s %#v", r.Method, r.URL, r.Header)
					}
					if r.Header.Get("Eop-Authorization") == "" {
						t.Fatal("unsigned request")
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"rows":[]}}`))}, nil
				})
				flag := "--page-index"
				if tc.name == "mongodb" {
					flag = "--page-now"
				}
				args := []string{"--output", "json", tc.name, "instance", "list", flag, "2"}
				if override {
					args = append(args, "--region", "explicit-region")
				}
				var out bytes.Buffer
				err := cli.Run(cli.Config{Args: args, Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile-region"}}}`), Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
					return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k]
				}})
				if err != nil || calls != 1 {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
			}
		})
	}
}

// TestRedisMongoDBMutatingGETSafety prevents task creation and billing changes from being retried or polled.
func TestRedisMongoDBMutatingGETSafety(t *testing.T) {
	for name, ids := range map[string][]string{
		"redis":   {"redis.key.big-analysis.create", "redis.key.hot-analysis.create"},
		"mongodb": {"mongodb.billing.convert-to-demand", "mongodb.billing.convert-to-package", "mongodb.instance.destroy"},
	} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range ids {
			found := false
			for _, command := range bundle.Commands.Commands {
				if command.ID != id {
					continue
				}
				found = true
				operation := bundle.APIs.Operations[command.Operation]
				if operation.Method != "GET" || operation.Retryable || command.Dangerous.Confirm != "yes" {
					t.Fatalf("unsafe mutating GET %s: %#v %#v", id, command, operation)
				}
			}
			if !found {
				t.Fatalf("missing command %s", id)
			}
		}
	}
}

// TestRedisMongoDBResponseContracts rejects HTTP-success business failures and decodes every captured fixture.
func TestRedisMongoDBResponseContracts(t *testing.T) {
	for _, name := range []string{"redis", "mongodb"} {
		dir := repoPath(t, "plugins/"+name)
		bundle, err := plugin.LoadBundle(dir, version.Version)
		if err != nil {
			t.Fatal(err)
		}
		for _, command := range bundle.Commands.Commands {
			op := bundle.APIs.Operations[command.Operation]
			for _, body := range []string{`{"statusCode":900,"code":"error","message":"failed"}`, `{"statusCode":200,"returnObj":{"data":{"isSucceed":false}},"isSucceed":false}`} {
				response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
				if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response}); err == nil {
					// Some order contracts use only statusCode 200; their documented success has no isSucceed field.
					if strings.Contains(body, `"statusCode":200`) && len(op.Response.Success) == 1 && op.Response.Success[0].Path == "statusCode" && op.Response.Success[0].Values[0] == "200" {
						continue
					}
					t.Errorf("%s accepted failure %s", command.ID, body)
				}
			}
			if command.FixtureResponse == "" {
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
			if _, err = client.DecodeHTTPResponse(fixture, client.RequestSpec{Method: op.Method, Response: op.Response}); err != nil {
				t.Fatalf("%s fixture: %v", command.ID, err)
			}
		}
	}
}

// TestRedisMongoDBWaiterStates checks published terminal states without inferring readiness from an envelope.
func TestRedisMongoDBWaiterStates(t *testing.T) {
	for _, tc := range []struct{ name, id, success, failure, pending string }{
		{"redis", "redis.instance.running", "0", "4", "1"},
		{"redis", "redis.node.running", "0", "", "1"},
		{"redis", "redis.legacy.node.running", "0", "", "1"},
		{"redis", "redis.migration.complete", "2", "3", "1"},
		{"redis", "redis.task.complete", "2", "4", "1"},
		{"redis", "redis.flashback.complete", "2", "3", "0"},
		{"redis", "redis.backup.complete", "success", "fail", "processing"},
		{"redis", "redis.backup.restored", "success", "fail", "create"},
		{"mongodb", "mongodb.instance.running", "0", "8", "9"},
		{"redis", "redis.instance.list.running", "0", "4", "1"},
		{"redis", "redis.legacy.instance.list.running", "0", "10", "9"},
		{"redis", "redis.legacy.dedicated-cluster.instance.running", "0", "8", "3"},
		{"redis", "redis.legacy.instance.running", "0", "4", "15"},
		{"redis", "redis.parameter.history.complete", "SUCCESS", "FAILED", "PENDING"},
		{"mongodb", "mongodb.instance.list.running", "0", "10", "12"},
	} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		w, ok := bundle.Waiters.Waiters[tc.id]
		if !ok {
			t.Fatalf("missing waiter %s", tc.id)
		}
		states := []struct {
			value any
			want  waiter.State
		}{{tc.success, waiter.Success}, {tc.pending, waiter.Pending}, {nil, waiter.Pending}, {json.Number("999999"), waiter.Pending}}
		if tc.failure != "" {
			states = append(states, struct {
				value any
				want  waiter.State
			}{tc.failure, waiter.Failure})
		}
		for _, state := range states {
			got, err := waiter.Evaluate(waiter.Spec{Path: "state", Success: w.Success, Failure: w.Failure, FailureValues: w.FailureValues}, map[string]any{"state": state.value})
			if err != nil || got != state.want {
				t.Fatalf("%s state %v: %v %v", tc.id, state.value, got, err)
			}
		}
	}
}

// TestRedisBackupWaiterRequiresExactIdentity rejects ambiguous polling before any API call.
func TestRedisBackupWaiterRequiresExactIdentity(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("missing backup identity reached the network")
		return nil, nil
	})
	err := cli.Run(cli.Config{Args: []string{"redis", "backup", "list", "--prod-inst-id", "instance", "--region", "region", "--wait", "redis.backup.complete"}, Config: []byte(`{}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err == nil || calls != 0 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/redis"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	w := bundle.Waiters.Waiters["redis.backup.restored"]
	if w.Selector == nil || w.Selector.Value != "$param.restore_name" {
		t.Fatalf("incorrect backup binding: %#v", w)
	}
	selector := *w.Selector
	selector.Value = "target"
	payload := map[string]any{"returnObj": map[string]any{"rows": []any{map[string]any{"restoreName": "other", "recoveryStatus": "success"}, map[string]any{"restoreName": "target", "recoveryStatus": "create"}}}}
	state, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Selector: &selector, Success: w.Success, FailureValues: w.FailureValues}, payload)
	if err != nil || state != waiter.Pending {
		t.Fatalf("selected wrong backup or treated preparation as completion: %s %v", state, err)
	}
}

// TestRedisMongoDBWaiterHelpAndCompletion verifies the shipped bindings are discoverable on their commands.
func TestRedisMongoDBWaiterHelpAndCompletion(t *testing.T) {
	for _, tc := range []struct {
		path []string
		id   string
	}{
		{[]string{"redis", "instance", "show"}, "redis.instance.running"},
		{[]string{"redis", "backup", "list"}, "redis.backup.complete"},
		{[]string{"mongodb", "instance", "show"}, "mongodb.instance.running"},
	} {
		for _, complete := range []bool{false, true} {
			args := append([]string{}, tc.path...)
			if complete {
				args = append([]string{"__complete"}, args...)
				args = append(args, "--wait", "")
			} else {
				args = append(args, "--help")
			}
			var out bytes.Buffer
			err := cli.Run(cli.Config{Args: args, Config: []byte(`{}`), Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir()})
			if err != nil || !strings.Contains(out.String(), tc.id) {
				t.Fatalf("%v: %v %s", args, err, out.String())
			}
		}
	}
}

// TestRedisCreateConditionalInputs rejects incomplete prepaid and automatic-renewal requests locally.
func TestRedisCreateConditionalInputs(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/redis"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	var create plugin.Command
	for _, c := range bundle.Commands.Commands {
		if c.ID == "redis.instance.create" {
			create = c
		}
	}
	for _, tc := range []struct{ flag, value, missing string }{{"charge-type", "PrePaid", "period"}, {"auto-renew", "true", "auto-renew-period"}} {
		args := commandSmokeArgs(t, create)
		for i := 0; i < len(args); i++ {
			if args[i] == "--"+tc.flag || args[i] == "--"+tc.missing {
				args = append(args[:i], args[i+2:]...)
				i--
			}
		}
		args = append([]string{"--lang", "en-US", "--yes"}, args...)
		args = append(args, "--region", "region", "--"+tc.flag, tc.value)
		calls := 0
		transport := nativeWireTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
		})
		var errs bytes.Buffer
		err := cli.Run(cli.Config{Args: args, Config: []byte(`{}`), Stdout: io.Discard, Stderr: &errs, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
		text := errs.String()
		if err != nil {
			text += err.Error()
		}
		if err == nil || calls != 0 || !strings.Contains(text, "--"+tc.missing) {
			t.Fatalf("missing %s: calls=%d err=%v stderr=%s", tc.missing, calls, err, errs.String())
		}
	}
}

// TestRedisMongoDBCollectionWaitersSelectExactRows rejects missing identities and ignores unrelated successes.
func TestRedisMongoDBCollectionWaitersSelectExactRows(t *testing.T) {
	for _, tc := range []struct {
		product, id, identity, pending string
		path                           []string
	}{
		{"redis", "redis.instance.list.running", "prod_inst_id", "1", []string{"redis", "instance", "list"}},
		{"redis", "redis.legacy.instance.list.running", "prod_inst_id", "3", []string{"redis", "legacy", "instance", "list"}},
		{"redis", "redis.legacy.dedicated-cluster.instance.running", "prod_inst_id", "9", []string{"redis", "legacy", "dedicated-cluster", "instance", "list"}},
		{"redis", "redis.parameter.history.complete", "history_id", "PENDING", []string{"redis", "parameter", "history", "list", "--prod-inst-id", "instance"}},
		{"mongodb", "mongodb.instance.list.running", "prod_inst_id", "9", []string{"mongodb", "instance", "list"}},
	} {
		t.Run(tc.id, func(t *testing.T) {
			bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.product), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			w := bundle.Waiters.Waiters[tc.id]
			if w.Selector == nil || w.Selector.Value != "$param."+tc.identity {
				t.Fatalf("incorrect selector: %#v", w)
			}
			args := append(append([]string{}, tc.path...), "--region", "region", "--wait", tc.id)
			err = cli.Run(cli.Config{Args: args, Config: []byte(`{}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: nativeWireTransport(func(*http.Request) (*http.Response, error) {
				t.Fatal("missing identity reached network")
				return nil, nil
			}), Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
			if err == nil || err.Error() != "error.waiter_requires_input" {
				t.Fatalf("missing identity error: %v", err)
			}
			set := func(row map[string]any, path string, value any) {
				parts := strings.Split(path, ".")
				for _, key := range parts[:len(parts)-1] {
					if row[key] == nil {
						row[key] = map[string]any{}
					}
					row = row[key].(map[string]any)
				}
				row[parts[len(parts)-1]] = value
			}
			other, target := map[string]any{}, map[string]any{}
			set(other, w.Selector.Key, "other")
			set(other, w.Path, w.Success)
			set(target, w.Selector.Key, "target")
			set(target, w.Path, tc.pending)
			payload := map[string]any{}
			set(payload, w.Selector.Path, []any{other, target})
			selector := *w.Selector
			selector.Value = "target"
			state, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Selector: &selector, Success: w.Success, FailureValues: w.FailureValues}, payload)
			if err != nil || state != waiter.Pending {
				t.Fatalf("selected unrelated success: %v %v", state, err)
			}
		})
	}
}

// TestDatabaseSupplementalFixtures preserves complete tag groups and documented MongoDB response envelopes.
func TestDatabaseSupplementalFixtures(t *testing.T) {
	for _, tc := range []struct{ product, command string }{
		{"redis", "redis.tag.list"},
		{"mongodb", "mongodb.v1.alarm.host.show"},
		{"mongodb", "mongodb.parameter.reset"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.product), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			var file string
			for _, command := range bundle.Commands.Commands {
				if command.ID == tc.command {
					file = command.FixtureResponse
				}
			}
			if file == "" {
				t.Fatal("missing documented response fixture")
			}
			data, err := os.ReadFile(repoPath(t, "plugins/"+tc.product+"/"+file))
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.DecodeFixture(data)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			var body map[string]any
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["statusCode"] != float64(800) {
				t.Fatal("missing success envelope")
			}
			switch tc.command {
			case "redis.tag.list":
				rows := body["returnObj"].(map[string]any)["list"].([]any)
				if len(rows) != 5 {
					t.Fatalf("lost tag groups: %d", len(rows))
				}
				for _, row := range rows {
					group := row.(map[string]any)
					for _, value := range group["data"].([]any) {
						if value.(map[string]any)["key"] != group["key"] {
							t.Fatal("tag nested under wrong key")
						}
					}
				}
			case "mongodb.v1.alarm.host.show":
				rows := body["returnObj"].([]any)
				if len(rows) != 1 || rows[0].(map[string]any)["thresholdCriticalOsCpu"] != float64(90) {
					t.Fatal("incorrect host alarm sample")
				}
			case "mongodb.parameter.reset":
				page := body["returnObj"].(map[string]any)
				if len(page["list"].([]any)) != 10 || page["hasNextPage"] != true {
					t.Fatal("lost parameter page data")
				}
			}
		})
	}
}
