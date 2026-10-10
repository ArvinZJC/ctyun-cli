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
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestExpandedProductBusinessErrors rejects HTTP 200 responses carrying business failures,
// including products whose success code is 0, 200 or 100000 rather than the EOP default.
func TestExpandedProductBusinessErrors(t *testing.T) {
	for _, name := range []string{"application-acceleration", "certificate", "cloud-audit", "cloud-search", "dec", "gtm", "smart-dns", "service-mesh", "agent-engine", "emr", "sms", "billing", "vpce", "traffic-package", "ec", "cda", "application-ha", "cae", "tokenhub", "eci", "cce-one", "rabbitmq", "rocketmq", "mqtt", "influxdb", "clickhouse", "htap", "ros", "research-assistant", "vod", "visual-service", "edge-ddos", "kms", "iam", "ai-capability", "dts", "ai-security", "dms", "edge-waf", "waf", "apm", "live-streaming", "cdn", "whole-site-acceleration", "edge-security-acceleration", "yunxiao", "model-training", "huiju", "kafka", "drds", "native-firewall", "sdwan", "cm", "msap", "lts", "server-security", "cloud-native-api-gateway", "mse", "crs", "ecx", "cce"} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		for id, op := range bundle.APIs.Operations {
			for _, body := range []string{`{"statusCode":500,"code":500,"success":false,"message":"failed"}`, `{"statusCode":"CTAPI_10002","code":"10002","error":"10002","returnObj":{}}`} {
				response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
				if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response}); err == nil {
					t.Errorf("%s accepted failure %s", id, body)
				}
			}
		}
	}
}

// TestCloudSearchRefundSafety protects the upstream GET mutation from retries or silent execution.
func TestCloudSearchRefundSafety(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/cloud-search"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range bundle.Commands.Commands {
		if command.ID != "cloud-search.instance.unsubscribe" {
			continue
		}
		op := bundle.APIs.Operations[command.Operation]
		if op.Method != "GET" || op.Retryable || command.Dangerous.Confirm != "yes" {
			t.Fatalf("unsafe refund: %#v %#v", command, op)
		}
		return
	}
	t.Fatal("refund command missing")
}

// TestCloudAuditHeaderBindings verifies that user identity stays in documented headers
// and that the explicit region overrides the profile without becoming a query field.
func TestCloudAuditHeaderBindings(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/v2/manager/user/enable" || r.Header.Get("regionid") != "explicit-region" || r.Header.Get("accountid") != "account" || r.Header.Get("userid") != "user" || r.URL.RawQuery != "" {
			t.Fatalf("incorrect audit request: %s %s %#v", r.Method, r.URL, r.Header)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":"0","returnObj":true}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"cloud-audit", "service", "status", "--region", "explicit-region", "--accountid", "account", "--userid", "user"}, Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"profile-region"}}}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestGTMDistinctCompletionStates ensures that a completed but disabled instance
// satisfies readiness without satisfying the separate enabled waiter.
func TestGTMDistinctCompletionStates(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/gtm"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id, value string
		want      waiter.State
	}{
		{"gtm.instance.ready", "2", waiter.Success}, {"gtm.instance.ready", "3", waiter.Success},
		{"gtm.instance.enabled", "2", waiter.Pending}, {"gtm.instance.enabled", "3", waiter.Success},
		{"gtm.instance.ready", "1", waiter.Pending}, {"gtm.instance.ready", "4", waiter.Failure},
		{"gtm.rule.ready", "2", waiter.Success}, {"gtm.rule.ready", "4", waiter.Failure}, {"gtm.rule.ready", "5", waiter.Pending},
	} {
		w, ok := bundle.Waiters.Waiters[tc.id]
		if !ok {
			t.Fatalf("missing waiter %s", tc.id)
		}
		got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, SuccessValues: w.SuccessValues, FailureValues: w.FailureValues}, map[string]any{"returnObj": map[string]any{"status": tc.value}})
		if err != nil || got != tc.want {
			t.Fatalf("%s state %s: %s %v", tc.id, tc.value, got, err)
		}
	}
}

// TestSMSLegacyReadActionIsFixed ensures that a legacy read uses the documented
// action-dispatch contract rather than exposing a caller-controlled mutation action.
func TestSMSLegacyReadActionIsFixed(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" || r.URL.Path != "/sms/api/v1" || body["action"] != "QuerySmsTemplate" || body["templateCode"] != "template" || len(body) != 2 {
			t.Fatalf("incorrect SMS request: %s %s %#v", r.Method, r.URL, body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"code":"OK","templateStatus":1}`))}, nil
	})
	args := []string{"sms", "v1", "template", "show", "--template-code", "template"}
	cfg := cli.Config{Args: args, Config: []byte(`{}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }}
	if err := cli.Run(cfg); err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	cfg.Args = append(args, "--action", "SendSms")
	if err := cli.Run(cfg); err == nil || calls != 1 {
		t.Fatalf("caller changed the read action: calls=%d err=%v", calls, err)
	}
}

// TestSMSApprovalWaiterStates verifies current and legacy approval response paths.
func TestSMSApprovalWaiterStates(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/sms"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for id, w := range bundle.Waiters.Waiters {
		for value, want := range map[string]waiter.State{"0": waiter.Pending, "1": waiter.Success, "2": waiter.Failure, "999": waiter.Pending} {
			field := strings.TrimPrefix(w.Path, "returnObj.")
			payload := map[string]any{field: value}
			if strings.HasPrefix(w.Path, "returnObj.") {
				payload = map[string]any{"returnObj": payload}
			}
			got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, Failure: w.Failure}, payload)
			if err != nil || got != want {
				t.Fatalf("%s value %s: %s %v", id, value, got, err)
			}
		}
	}
}

// TestExpandedProductNestedFailures rejects inner operation failures even when
// the EOP envelope itself reports success.
func TestExpandedProductNestedFailures(t *testing.T) {
	for _, tc := range []struct{ product, operation, body string }{
		{"cda", "cda.static-route.create", `{"statusCode":800,"returnObj":{"result":"0","errorMsg":"failed"}}`},
		{"research-assistant", "research-assistant.queue.create", `{"statusCode":200,"returnObj":{"status":{"code":"failed"}}}`},
		{"vod", "vod.task.cancel", `{"statusCode":0,"returnObj":{"code":0,"data":{"code":1,"successful":false}}}`},
		{"clickhouse", "clickhouse.sql.execute", `{"code":0,"data":{"success":false,"result":"failed"}}`},
		{"htap", "htap.instance.renew", `{"statusCode":200,"returnObj":{"data":{"submitted":false}}}`},
		{"ec", "ec.bandwidth-package.price.create", `{"statusCode":800,"returnObj":{"isSucceed":false,"totalPrice":0}}`},
	} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.product), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		op, ok := bundle.APIs.Operations[tc.operation]
		if !ok {
			t.Fatalf("missing operation %s", tc.operation)
		}
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body))}
		if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response}); err == nil {
			t.Errorf("%s accepted nested failure", tc.operation)
		}
	}
}

// TestNetworkPricingSafety distinguishes price queries from the purchase or
// resize operations named in their paths.
func TestNetworkPricingSafety(t *testing.T) {
	for _, name := range []string{"ec", "vpce", "traffic-package"} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, cmd := range bundle.Commands.Commands {
			if !strings.Contains(cmd.ID, ".price.") {
				continue
			}
			found++
			if !bundle.APIs.Operations[cmd.Operation].Retryable || cmd.Dangerous.Confirm == "yes" {
				t.Errorf("price query treated as mutation: %s", cmd.ID)
			}
		}
		if found == 0 {
			t.Errorf("missing pricing commands for %s", name)
		}
	}
}

// TestTrafficPackageActivationStates preserves the documented Chinese wire
// states and keeps initial and unknown states pending.
func TestTrafficPackageActivationStates(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/traffic-package"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	w, ok := bundle.Waiters.Waiters["traffic-package.package.active"]
	if !ok {
		t.Fatal("missing activation waiter")
	}
	for value, want := range map[string]waiter.State{"初始": waiter.Pending, "有效": waiter.Success, "退订": waiter.Failure, "过期": waiter.Failure, "销毁": waiter.Failure, "unknown": waiter.Pending} {
		got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, FailureValues: w.FailureValues}, map[string]any{"returnObj": map[string]any{"status": value}})
		if err != nil || got != want {
			t.Fatalf("state %s: %s %v", value, got, err)
		}
	}
}

// TestTokenHubQueryBinding preserves the query location shown in the official
// request URL despite its misplaced path-parameter documentation table.
func TestTokenHubQueryBinding(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/maas/modelService/billing/productDetails" || r.URL.Query().Get("modelId") != "model/one" {
			t.Fatalf("incorrect model query: %s %s", r.Method, r.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":200,"returnObj":{"modelId":"model/one"}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"tokenhub", "model", "product", "show", "--model-id", "model/one"}, Config: []byte(`{}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestCCEOnePathIdentity verifies that numeric-looking resource IDs keep their
// precision and that path delimiters in caller input remain within one segment.
func TestCCEOnePathIdentity(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.EscapedPath() != "/v1/clusters/1891792064037486593%2Fpart" {
			t.Fatalf("incorrect cluster path: %s", r.URL.EscapedPath())
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"clusterId":"1891792064037486593/part","businessStatus":110}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"cce-one", "cluster", "show", "1891792064037486593/part", "--region", "test-region"}, Config: []byte(`{"region":"test-region"}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestDatabaseGETMutationSafety prevents upstream GET lifecycle endpoints from
// being retried or executed without the normal mutation confirmation.
func TestDatabaseGETMutationSafety(t *testing.T) {
	for _, name := range []string{"clickhouse", "htap"} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		for _, cmd := range bundle.Commands.Commands {
			op := bundle.APIs.Operations[cmd.Operation]
			if op.Method != "GET" || strings.Contains(cmd.ID, ".price.") {
				continue
			}
			for _, suffix := range []string{".restart", ".start", ".stop", ".renew", ".unsubscribe"} {
				if strings.HasSuffix(cmd.ID, suffix) && (op.Retryable || cmd.Dangerous.Confirm != "yes") {
					t.Errorf("unsafe GET mutation %s", cmd.ID)
				}
			}
		}
	}
}

// TestHTAPWaiterSelectsNestedIdentity verifies the real bundle's nested identity
// and state paths against multiple rows, missing instances and terminal failures.
func TestHTAPWaiterSelectsNestedIdentity(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/htap"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	w, ok := bundle.Waiters.Waiters["htap.instance.running"]
	if !ok || w.Selector == nil || w.Selector.Value != "$param.spu_inst_id" {
		t.Fatal("missing exact instance binding")
	}
	selector := *w.Selector
	selector.Value = "target"
	spec := waiter.Spec{Path: w.Path, Success: w.Success, FailureValues: w.FailureValues, Selector: &selector}
	for _, tc := range []struct {
		body string
		want waiter.State
	}{
		{`{"returnObj":[{"prodInstInfo":{"spuInstId":"other","status":2}},{"prodInstInfo":{"spuInstId":"target","status":0}}]}`, waiter.Success},
		{`{"returnObj":[{"prodInstInfo":{"spuInstId":"other","status":0}}]}`, waiter.Pending},
		{`{"returnObj":[]}`, waiter.Pending},
		{`{"returnObj":[{"prodInstInfo":{"spuInstId":"target","status":1}}]}`, waiter.Pending},
		{`{"returnObj":[{"prodInstInfo":{"spuInstId":"target","status":2}}]}`, waiter.Failure},
	} {
		var payload map[string]any
		if err := json.Unmarshal([]byte(tc.body), &payload); err != nil {
			t.Fatal(err)
		}
		got, err := waiter.Evaluate(spec, payload)
		if err != nil || got != tc.want {
			t.Fatalf("%s: %s %v", tc.body, got, err)
		}
	}
}

// TestResearchStorageZoneIdentity prevents product-local numeric availability
// zone IDs from being replaced by the general string region profile.
func TestResearchStorageZoneIdentity(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.URL.Path != "/api/bc/v2/listStorageResearchSpecs" || body["regionId"] != float64(194) {
			t.Fatalf("incorrect zone binding: %s %#v", r.URL, body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":"200","returnObj":{"status":{"code":"ok"},"specs":[]}}`))}, nil
	})
	cfg := cli.Config{Args: []string{"research-assistant", "storage", "specification", "list", "--zone-id", "194"}, Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"general-region"}}}`), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }}
	if err := cli.Run(cfg); err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	cfg.Args = cfg.Args[:4]
	if err := cli.Run(cfg); err == nil || calls != 1 {
		t.Fatalf("zone fell back to general region: calls=%d err=%v", calls, err)
	}
}
