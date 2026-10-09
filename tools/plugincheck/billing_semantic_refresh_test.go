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
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
)

// TestBillingProductPeriodCurrentEnvelope accepts the published statusCode
// contract and rejects an HTTP-success business failure even when rows exist.
func TestBillingProductPeriodCurrentEnvelope(t *testing.T) {
	ctx := loadStorageReviewContext(t, "billing")
	op := ctx.bundle.APIs.Operations["billing.product.period.list"]
	for _, tc := range []struct {
		name, body string
		fails      bool
	}{
		{"current success", `{"statusCode":"800","msg":"成功","result":{"result":[{"consumeDate":"2025-08-21"}]}}`, false},
		{"current failure", `{"statusCode":"900","msg":"failed","result":{"result":[{"consumeDate":"2025-08-21"}]}}`, true},
		{"missing status", `{"msg":"成功","result":{"result":[]}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(tc.body))}
			_, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: op.Method, Response: op.Response})
			if (err != nil) != tc.fails {
				t.Fatalf("err=%v, want failure=%v", err, tc.fails)
			}
		})
	}
}

// TestBillingUsagePeriodExampleOnlyIDsStayOptional prevents a required-input
// error for account and user IDs no longer declared in the current request table.
func TestBillingUsagePeriodExampleOnlyIDsStayOptional(t *testing.T) {
	calls := 0
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" || r.URL.Path != "/acct/acct-openapi/queryUsageCycleBill" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		for _, field := range []string{"accountId", "userId"} {
			if _, exists := body[field]; exists {
				t.Errorf("omitted optional input sent: %s", field)
			}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"success":true,"errorCode":"800","result":{"result":[]}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"billing", "usage", "period", "list", "--billing-cycle-id", "202508", "--page-no", "1", "--page-size", "1", "--output", "json"}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
		return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k]
	}})
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

// TestVPCMirrorFilterCurrentDestinationPort renders distinct source and
// destination values from the published destPort field through the real bundle.
func TestVPCMirrorFilterCurrentDestinationPort(t *testing.T) {
	var out bytes.Buffer
	transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"mirrorFilterRules":[{"srcPort":"1-2","destPort":"8080"}]}}`))}, nil
	})
	err := cli.Run(cli.Config{Args: []string{"--lang", "en-US", "vpc", "mirror", "filter", "rule", "list", "--region", "region", "--mirror-filter-id", "filter", "--direction", "IN", "--cols", "src_port,desc_port"}, Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
		return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k]
	}})
	if err != nil || !strings.Contains(out.String(), "8080") || !strings.Contains(out.String(), "Source Port") || !strings.Contains(out.String(), "1-2") {
		t.Fatalf("err=%v output=%s", err, out.String())
	}
}

// TestOfficialStorageDeprecationsPreserveScopeAndCommands keeps verified API
// notices and replacements without changing resource-lifecycle commands.
func TestOfficialStorageDeprecationsPreserveScopeAndCommands(t *testing.T) {
	for _, tc := range []struct {
		plugin, operation, replacement, scope string
	}{
		{"evs", "v4.evs.volume.set-snapshot-delete-policy", "ctyun evs volume update", "即将废弃"},
		{"ims", "v4.ims.import-task.list", "ctyun ims image list", "已弃用，目前仍可调用"},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			ctx := loadStorageReviewContext(t, tc.plugin)
			api := ctx.bundle.APIs.Operations[tc.operation]
			if !api.Deprecation.Active() || !strings.Contains(api.Deprecation.Notice, tc.scope) {
				t.Fatalf("API lifecycle notice = %#v, want scope %q", api.Deprecation, tc.scope)
			}
			if api.Deprecation.Replacement == nil || api.Deprecation.Replacement.Kind != "command" || api.Deprecation.Replacement.Label != tc.replacement {
				t.Fatalf("replacement = %#v, want %q", api.Deprecation.Replacement, tc.replacement)
			}
		})
	}
	traffic := loadStorageReviewContext(t, "traffic-package")
	shutdown := traffic.bundle.APIs.Operations["traffic-package.package.metric.query"].Deprecation
	if !shutdown.Active() || !strings.Contains(shutdown.Notice, "本接口已下线") || shutdown.Replacement != nil {
		t.Fatalf("shutdown notice = %#v, want published shutdown without guessed replacement", shutdown)
	}
	ims := loadStorageReviewContext(t, "ims")
	if api := ims.bundle.APIs.Operations["v4.ims.import-task.delete"]; api.Deprecation.Active() {
		t.Fatalf("multi-AZ-only notice became unconditional runtime deprecation: %#v", api.Deprecation)
	}
	for _, command := range ims.bundle.Commands.Commands {
		if command.Operation != "v4.ims.import-task.delete" {
			continue
		}
		if command.Deprecation.Active() || command.Recommendation == nil || command.Recommendation.TargetCommand.Plugin != "ims" || !strings.Contains(command.Recommendation.Applicability, "multi-AZ") {
			t.Fatalf("scoped deletion guidance = %#v, deprecation = %#v", command.Recommendation, command.Deprecation)
		}
	}
	for _, id := range []string{"v4.ims.image.disable", "v4.ims.image.enable"} {
		if operation, ok := ims.bundle.APIs.Operations[id]; ok && operation.Deprecation.Active() {
			t.Errorf("resource lifecycle command %s was incorrectly deprecated", id)
		}
	}
}
