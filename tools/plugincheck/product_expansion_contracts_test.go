/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
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

// TestAIResultAndFailureBoundaries distinguishes negative detection results from
// failed processing and prevents JSON errors from becoming downloaded files.
func TestAIResultAndFailureBoundaries(t *testing.T) {
	for _, tc := range []struct {
		product, operation, body, media string
		success                         bool
	}{
		{"ai-capability", "ai-capability.image.violence.detect", `{"statusCode":0,"message":{"success":0,"fail":1},"returnObj":[{"err_code":4008}]}`, "application/json", false},
		{"ai-capability", "ai-capability.image.violence.detect", `{"statusCode":0,"message":{"success":1,"fail":0},"returnObj":{"violence":[{"label":1}]}}`, "application/json", true},
		{"ai-security", "ai-security.guard.check", `{"code":"200","success":true,"data":{"checkResult":"fail"}}`, "application/json", true},
		{"ai-security", "ai-security.file.check", `{"code":"200","success":true,"data":{"result":true}}`, "application/json", true},
		{"ai-security", "ai-security.watermark.add", `{"code":"200","data":{"Data":{"success":false}}}`, "application/json", false},
		{"ai-security", "ai-security.watermark.download", `{"code":"401","success":false}`, "application/json", false},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+tc.product), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			op, ok := bundle.APIs.Operations[tc.operation]
			if !ok {
				t.Fatal("operation missing")
			}
			response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{tc.media}}, Body: io.NopCloser(strings.NewReader(tc.body))}
			_, err = client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
			if (err == nil) != tc.success {
				t.Fatalf("success=%v err=%v", tc.success, err)
			}
		})
	}
}

// TestDTSOverallCompletionRequiresFinalState prevents migration phase completion
// from being reported as completion of the entire transmission job.
func TestDTSOverallCompletionRequiresFinalState(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/dts"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	w, ok := bundle.Waiters.Waiters["dts.job.finished"]
	if !ok {
		t.Fatal("completion waiter missing")
	}
	for _, tc := range []struct {
		value string
		want  waiter.State
	}{
		{"FULLOVER", waiter.Pending}, {"MIGRATED", waiter.Pending}, {"INC_DISPATCH_DONE", waiter.Pending},
		{"FINISH", waiter.Success}, {"PAUSE", waiter.Failure}, {"PRECHECKFAILURE", waiter.Failure},
	} {
		got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, SuccessValues: w.SuccessValues, FailureValues: w.FailureValues}, map[string]any{"data": map[string]any{"jobStatus": tc.value}})
		if err != nil || got != tc.want {
			t.Fatalf("state %s: %s %v", tc.value, got, err)
		}
	}
}

// TestEdgeWAFPreservesConditionGroups keeps separate nested rule groups intact
// when the upstream table incorrectly describes a flat array of objects.
func TestEdgeWAFPreservesConditionGroups(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/api-common/ctapi/api/adProtect/update" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
		}
		var body struct {
			PublicRange [][]map[string]any `json:"publicRange"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.PublicRange) != 2 || len(body.PublicRange[0]) != 1 || len(body.PublicRange[1]) != 1 || body.PublicRange[0][0]["publicContent"] != "/first" || body.PublicRange[1][0]["publicContent"] != "/second" {
			t.Fatalf("condition groups changed: %#v", body.PublicRange)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":100000,"returnObj":{}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "edge-waf", "ad-protection", "update", "--domain", "example.com", "--product-code", "010", "--mod", "CLOSE", "--ad-switch", "LOG", "--advanced-detect", "CLOSE", "--public-range", `[[{"zone":"PATH","publicContent":"/first"}],[{"zone":"PATH","publicContent":"/second"}]]`}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestAPMBatchDeletePreservesTopLevelArray sends large numeric IDs as their
// original file bytes, without introducing a named object wrapper or rounding.
func TestAPMBatchDeletePreservesTopLevelArray(t *testing.T) {
	const payload = "[1054661322597744600,1054661322597744601]\n"
	path := filepath.Join(t.TempDir(), "ids.json")
	if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/v1/alert/rule/batchDelete" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("regionId") != "test-region" {
			t.Fatalf("unexpected request: %s %s %#v", r.Method, r.URL, r.Header)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != payload {
			t.Fatalf("body=%s err=%v", body, err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":2}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "apm", "alert", "rule", "batch", "delete", "--region", "test-region", "--ids-file", path}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestLiveStreamingErrorExampleAndGETMutations prevents a published error example
// from becoming a success fixture and keeps GET-based stop operations guarded.
func TestLiveStreamingErrorExampleAndGETMutations(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/live-streaming"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	op := bundle.APIs.Operations["live-streaming.pull-config.update"]
	response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":200002,"error":"CDN_200002","errorMessage":"parameter validation failed"}`))}
	if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response}); err == nil {
		t.Fatal("published parameter-validation failure accepted")
	}
	wanted := map[string]bool{"live-streaming.snapshot.stop": false, "live-streaming.relay.stop": false, "live-streaming.recording.stop": false, "live-streaming.domain.ownership.verify": false}
	for _, command := range bundle.Commands.Commands {
		if _, ok := wanted[command.ID]; !ok {
			continue
		}
		wanted[command.ID] = true
		operation := bundle.APIs.Operations[command.Operation]
		if operation.Method != "GET" || operation.Retryable || command.Dangerous.Confirm != "yes" {
			t.Errorf("unsafe GET mutation %s", command.ID)
		}
	}
	for id, found := range wanted {
		if !found {
			t.Errorf("missing command %s", id)
		}
	}
}

// TestCDNNestedConditionsAndDefaultRequirements preserves condition arrays and
// enforces the documented default task lookup mode before any HTTP request.
func TestCDNNestedConditionsAndDefaultRequirements(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/v1/domain/update-domain" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		var body struct {
			Conditions map[string][]struct {
				Mode    int    `json:"mode"`
				Content string `json:"content"`
			} `json:"black_referer_condition"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		conditions := body.Conditions["black_referer"]
		if len(conditions) != 1 || conditions[0].Mode != 1 || conditions[0].Content != "/protected" {
			t.Fatalf("condition structure changed: %#v", body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":100000,"returnObj":{}}`))}, nil
	})
	run := func(args []string) error {
		return cli.Run(cli.Config{Args: args, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	}
	if err := run([]string{"--yes", "cdn", "domain", "update", "--domain", "example.com", "--black-referer-condition", `{"black_referer":[{"mode":1,"content":"/protected"}]}`}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if err := run([]string{"cdn", "refresh", "list"}); err == nil {
		t.Fatal("default time lookup accepted without time bounds")
	}
	if calls != 1 {
		t.Fatal("invalid lookup reached HTTP transport")
	}
}

// TestEdgeSecurityNestedSuccess distinguishes logged HTTP failures from API
// failures and rejects an order that the server did not submit.
func TestEdgeSecurityNestedSuccess(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/edge-security-acceleration"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		operation, body string
		success         bool
	}{
		{"edge-security-acceleration.waf.attack-log.list", `{"returnObj":{"statusCode":"100000","results":[{"statusCode":"403"}]}}`, true},
		{"edge-security-acceleration.waf.attack-log.list", `{"returnObj":{"statusCode":"200002","results":[]}}`, false},
		{"edge-security-acceleration.service.subscribe", `{"statusCode":"100000","returnObj":{"submitted":false,"errorMessage":"order rejected"}}`, false},
	} {
		op, ok := bundle.APIs.Operations[tc.operation]
		if !ok {
			t.Fatal("operation missing")
		}
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body))}
		_, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
		if (err == nil) != tc.success {
			t.Errorf("%s success=%v err=%v", tc.operation, tc.success, err)
		}
	}
}

// TestYunxiaoRejectsPublishedAuthorizationFailures keeps status 900 examples
// from becoming successful command results, even when returnObj is populated.
func TestYunxiaoRejectsPublishedAuthorizationFailures(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/yunxiao"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for id, op := range bundle.APIs.Operations {
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":900,"error":"Cwai.Task.UnAuthorized","returnObj":{"id":"example"}}`))}
		if _, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response}); err == nil {
			t.Errorf("%s accepted an authorization failure", id)
		}
	}
}

// TestYunxiaoWorkspaceLocationsAndEndpoint preserves distinct header and body
// workspace values and sends caller-prepared command content unchanged.
func TestYunxiaoWorkspaceLocationsAndEndpoint(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "cwai-global.ctapi.ctyun.cn" || r.URL.Path != "/v4/cwai/central/task-service/task/create" || r.Header.Get("workspaceID") != "header-space" {
			t.Fatalf("unexpected request %s headers=%v", r.URL, r.Header)
		}
		var body struct {
			Workspace string   `json:"workspaceID"`
			Commands  []string `json:"commands"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Workspace != "body-space" || len(body.Commands) != 1 || body.Commands[0] != "prepared-ciphertext" {
			t.Fatalf("body changed: %#v", body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "yunxiao", "task", "create", "--header-workspace-id", "header-space", "--workspace-id", "body-space", "--commands", `["prepared-ciphertext"]`, "--image-name", "example-image", "--queue-id", "queue", "--resources", `[]`}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestVPCECreateAcceptsOnlyGuardedOrderProgress distinguishes an accepted order
// still being processed from ordinary failures sharing status code 900.
func TestVPCECreateAcceptsOnlyGuardedOrderProgress(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/vpce"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	op, ok := bundle.APIs.Operations["vpce.endpoint.create"]
	if !ok {
		t.Fatal("endpoint create operation missing")
	}
	spec := client.RequestSpec{Method: op.Method, Response: op.Response}
	for _, rule := range op.AcceptedStatuses {
		spec.AcceptedStatuses = append(spec.AcceptedStatuses, client.AcceptedStatusRule{Code: rule.Code, RequiredPath: rule.RequiredPath})
	}
	for _, tc := range []struct {
		body    string
		success bool
	}{
		{`{"statusCode":800,"returnObj":{"masterOrderID":"order"}}`, true},
		{`{"statusCode":900,"errorCode":"endpoint.order.inProgress","returnObj":{"masterOrderID":"order"}}`, true},
		{`{"statusCode":900,"errorCode":"Openapi.Parameter.Error"}`, false},
		{`{"statusCode":900,"returnObj":{}}`, false},
	} {
		response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body))}
		_, err := client.DecodeHTTPResponse(response, spec)
		if (err == nil) != tc.success {
			t.Errorf("response %s success=%v err=%v", tc.body, tc.success, err)
		}
	}
}

// TestModelTrainingPathAndPaginationBindings preserves escaped resource identity
// and the independently documented query strings and integer JSON body fields.
func TestModelTrainingPathAndPaginationBindings(t *testing.T) {
	for _, tc := range []struct {
		args       []string
		path       string
		pagination bool
	}{
		{[]string{"model-training", "model", "version", "list", "--page-num", "1", "--page-size", "10", "--query-page-num", "2", "--query-page-size", "20"}, "/api/v3/aipaas/model/train/modelManage/versions/page", true},
		{[]string{"model-training", "task", "show", "task/with space"}, "/api/v3/aipaas/model/train/api/v3/task/detail/task%2Fwith%20space", false},
	} {
		calls := 0
		transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "GET" || r.URL.EscapedPath() != tc.path || r.URL.Host != "ctxuntui-global.ctapi.ctyun.cn" {
				t.Fatalf("unexpected request %s %s", r.Method, r.URL)
			}
			if tc.pagination {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 2 || body["pageNum"] != float64(1) || body["pageSize"] != float64(10) || r.URL.Query().Get("pageNum") != "2" || r.URL.Query().Get("pageSize") != "20" {
					t.Fatalf("pagination changed: %v %s", body, r.URL.RawQuery)
				}
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":200,"returnObj":{"list":[],"status":"RUNNING"}}`))}, nil
		})
		err := cli.Run(cli.Config{Args: tc.args, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
			return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk", "CTYUN_REGION": "test-region"}[k]
		}})
		if err != nil || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}

// TestModelTrainingGETMutationsRequireConfirmation prevents retrieval-looking
// verbs from retrying or silently changing training and storage resources.
func TestModelTrainingGETMutationsRequireConfirmation(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/model-training"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"model-training.task.start", "model-training.task.stop", "model-training.ide.start", "model-training.v2.storage.delegate.authorize"} {
		op, ok := bundle.APIs.Operations[id]
		if !ok || op.Method != "GET" || op.Retryable {
			t.Fatalf("unsafe GET operation %s: %#v", id, op)
		}
		found := false
		for _, command := range bundle.Commands.Commands {
			if command.Operation == id {
				found = true
				if command.Dangerous.Confirm != "yes" {
					t.Fatalf("%s lacks confirmation", id)
				}
			}
		}
		if !found {
			t.Fatalf("command missing for %s", id)
		}
	}
}

// TestTrainingWaitersKeepRestartableStatesPending avoids premature failures
// while a newly started task still reports its pre-start or recovery state.
func TestTrainingWaitersKeepRestartableStatesPending(t *testing.T) {
	for _, name := range []string{"model-training", "huiju"} {
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		for _, suffix := range []string{"running", "completed"} {
			w := bundle.Waiters.Waiters[name+".task."+suffix]
			for _, state := range []string{"WAITING", "CREATING"} {
				got, err := waiter.Evaluate(waiter.Spec{Path: w.Path, Success: w.Success, FailureValues: w.FailureValues}, map[string]any{"returnObj": map[string]any{"status": state}})
				if err != nil || got != waiter.Pending {
					t.Errorf("%s %s for %s: %v %v", name, suffix, state, got, err)
				}
			}
		}
	}
}

// TestHuijuMultipartFileStaysOutOfQuery sends the documented binary form part
// with a generated boundary while retaining the scalar storage query fields.
func TestHuijuMultipartFileStaysOutOfQuery(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.json")
	const content = `{"message":"exact file contents"}`
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.Host != "huijuaip-global.ctapi.ctyun.cn" || r.URL.Path != "/huiju/api/v2/user/storage/personal/pvc/file" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		if r.URL.Query().Has("file") || r.URL.Query().Get("filePath") != "/work/home/data.json" || r.URL.Query().Get("clusterId") != "1" {
			t.Fatalf("incorrect query %s", r.URL.RawQuery)
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(part)
		if err != nil || part.FormName() != "file" || string(data) != content {
			t.Fatalf("file changed: %s %q %v", part.FormName(), data, err)
		}
		if _, err := reader.NextPart(); err != io.EOF {
			t.Fatalf("unexpected extra part: %v", err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":200,"returnObj":true}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--yes", "huiju", "storage", "file", "import", "--file", file, "--file-path", "/work/home/data.json", "--cluster-id", "1", "--region", "region"}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestKafkaQuotaRejectsNestedFailure rejects an unsuccessful Boolean result
// even when the outer Kafka envelope reports status 800.
func TestKafkaQuotaRejectsNestedFailure(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, "plugins/kafka"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for id, op := range bundle.APIs.Operations {
		if !strings.HasPrefix(id, "kafka.quota.") || !strings.HasSuffix(id, ".create") && !strings.HasSuffix(id, ".update") && !strings.HasSuffix(id, ".delete") {
			continue
		}
		for _, value := range []string{"true", "false"} {
			response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"data":` + value + `}}`))}
			_, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
			if (err == nil) != (value == "true") {
				t.Errorf("%s data=%s err=%v", id, value, err)
			}
		}
	}
}

// TestKafkaRebalancePreservesNestedBrokerAssignments keeps manual partition
// maps intact and enforces the documented automatic-mode default before calls.
func TestKafkaRebalancePreservesNestedBrokerAssignments(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		var assignments map[string][]int
		if err := json.Unmarshal(body["partitionBrokers"], &assignments); err != nil {
			t.Fatal(err)
		}
		if len(assignments) != 2 || len(assignments["0"]) != 2 || assignments["0"][1] != 2 || string(body["type"]) != "1" {
			t.Fatalf("assignments changed: %s", body["partitionBrokers"])
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"data":"create success"}}`))}, nil
	})
	base := []string{"--yes", "kafka", "v3", "topic", "partition", "rebalance", "--prod-inst-id", "instance", "--topic-name", "topic", "--region", "region"}
	run := func(extra ...string) error {
		return cli.Run(cli.Config{Args: append(append([]string{}, base...), extra...), Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }})
	}
	if err := run(); err == nil || calls != 0 {
		t.Fatalf("automatic default omitted brokers: calls=%d err=%v", calls, err)
	}
	if err := run("--type", "1"); err == nil || calls != 0 {
		t.Fatalf("manual mode omitted assignments: calls=%d err=%v", calls, err)
	}
	if err := run("--type", "1", "--partition-brokers", `{"0":[1,2],"1":[2,3]}`); err != nil || calls != 1 {
		t.Fatalf("manual mode failed: calls=%d err=%v", calls, err)
	}
}
