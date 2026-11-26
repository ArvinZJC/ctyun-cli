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
)

// TestRefreshedAPIRequestLocations protects the wire contract where the portal
// now documents JSON bodies for POSTs and query parameters for GETs.
func TestRefreshedAPIRequestLocations(t *testing.T) {
	for _, tc := range []struct{ product, operation, location, field, binding string }{
		{"ims", "v4.ims.image.deactivate", "body", "imageID", "$arg.image_id"},
		{"ims", "v4.ims.image.reactivate", "body", "imageID", "$arg.image_id"},
		{"ims", "v4.ims.import-task.delete", "body", "taskID", "$arg.task_id"},
		{"ecs", "v4.ecs.dedicated-host.ecs-flavor", "query", "dedicatedHostID", "$arg.dedicated_host_id"},
		{"ecs", "v4.ecs.dedicated-host.flavor-list", "query", "regionID", "$profile.region"},
		{"ecs", "v4.ecs.dedicated-host.list", "body", "regionID", "$profile.region"},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			ctx := loadStorageReviewContext(t, tc.product)
			op := ctx.bundle.APIs.Operations[tc.operation]
			want, other := op.Body, op.Query
			if tc.location == "query" {
				want, other = op.Query, op.Body
			}
			if want[tc.field] != tc.binding || other[tc.field] != "" {
				t.Fatalf("%s %s mapping: body=%v query=%v", tc.location, tc.field, op.Body, op.Query)
			}
		})
	}
}

// TestECPCRenewPriceRequiresBillingCycle prevents a renewal-price request from
// reaching execution with the removed bandwidth input and no billing cycle.
func TestECPCRenewPriceRequiresBillingCycle(t *testing.T) {
	ctx := loadStorageReviewContext(t, "ecpc")
	const op = "v4.ecpc.floating-ip.describe-floating-ip-renew-price"
	for _, target := range []string{"cycleType", "cycleCnt"} {
		if !ctx.commandParameter(op, target).Required {
			t.Errorf("%s must be required", target)
		}
	}
	if _, ok := ctx.bundle.APIs.Operations[op].Body["band"]; ok {
		t.Error("renewal pricing still sends bandwidth")
	}
}

// TestCloudAssistantDocumentationStatusKeepsCommandAvailable distinguishes
// unavailable public documentation from the available command API.
func TestCloudAssistantDocumentationStatusKeepsCommandAvailable(t *testing.T) {
	ctx := loadStorageReviewContext(t, "cloud-assistant")
	op := ctx.bundle.APIs.Operations["v4.cloud-assistant.command.run"]
	if op.Deprecation != nil {
		t.Fatal("available API marked deprecated")
	}
	for _, tc := range []struct{ language, notice string }{{"en-US", "API is available, but its public documentation is currently unavailable"}, {"en-GB", "API is available, but its public documentation is currently unavailable"}, {"zh-CN", "API 可用，但公开文档目前不可用"}} {
		var stdout bytes.Buffer
		if err := cli.Run(cli.Config{Args: []string{"--lang", tc.language, "help", "cloud-assistant", "command", "run"}, Stdout: &stdout, PluginRoot: t.TempDir()}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout.String(), tc.notice) {
			t.Fatalf("%s help does not distinguish API and documentation availability:\n%s", tc.language, stdout.String())
		}
	}
}

// TestScopedShutdownNoticesKeepFieldsUsable verifies that an offline flavor
// state and a retiring modification action preserve normal field and option use.
func TestScopedShutdownNoticesKeepFieldsUsable(t *testing.T) {
	var warnings bytes.Buffer
	err := cli.Run(cli.Config{
		Args:       []string{"api-gateway", "flavor", "list", "--region", "test-region", "--cols", "status,az", "--offline"},
		PluginRoot: t.TempDir(),
		Stdout:     io.Discard,
		Stderr:     &warnings,
	})
	if err != nil || warnings.Len() != 0 {
		t.Fatalf("flavor fields: %v, %s", err, warnings.String())
	}
	for _, name := range []string{"cdn", "whole-site-acceleration"} {
		context := loadStorageReviewContext(t, name)
		operation := name + ".domain.create"
		if context.commandParameter(operation, "action").Deprecation.Active() {
			t.Errorf("%s marks all actions deprecated", operation)
		}
		for _, locale := range []string{"en-GB", "en-US", "zh-CN"} {
			help := context.bundle.I18N[locale]["parameter."+operation+".action.description"]
			if !strings.Contains(help, "2") || !strings.Contains(help, name+" domain update") {
				t.Errorf("%s %s lost the value-specific replacement: %q", operation, locale, help)
			}
		}
	}
}

// TestServerSecurityLastScanVulnerabilityTypes checks the localized heading
// for the scan categories returned in osType by the last-scan detail API.
func TestServerSecurityLastScanVulnerabilityTypes(t *testing.T) {
	for _, tc := range []struct{ locale, label string }{
		{"en-US", "Vulnerability Types"}, {"en-GB", "Vulnerability Types"}, {"zh-CN", "漏洞类型"},
	} {
		var stdout bytes.Buffer
		err := cli.Run(cli.Config{
			Args:       []string{"--lang", tc.locale, "server-security", "vulnerability", "last-scan", "show", "--offline", "--cols", tc.label},
			PluginRoot: t.TempDir(),
			Stdout:     &stdout,
		})
		if err != nil || !strings.Contains(stdout.String(), tc.label) || !strings.Contains(stdout.String(), "LINUX,WINDOWS") {
			t.Errorf("%s rendered %q, %v", tc.locale, stdout.String(), err)
		}
	}
}

// refreshTransport captures signed API requests without making network calls.
type refreshTransport struct{ body map[string]any }

// RoundTrip records the request body and returns a successful API envelope.
func (transport *refreshTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := json.NewDecoder(request.Body).Decode(&transport.body); err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{}}`))}, nil
}

// TestECSLegacyBillingConversionIsNotOverridden verifies that optional new
// inputs do not change the wire meaning of an existing cancellation request.
func TestECSLegacyBillingConversionIsNotOverridden(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		transport := &refreshTransport{}
		args := []string{"ecs", "instance", "convert-to-ondemand", "--region", "test-region", "--instance-idlist", "test-instance", "--auto-to-need", "false", "--yes", "--output", "json"}
		if explicit {
			args = append(args, "--conversion-type", "2")
		}
		err := cli.Run(cli.Config{Args: args, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "test-ak"
			case "CTYUN_SK":
				return "test-sk"
			}
			return ""
		}})
		if err != nil {
			t.Fatal(err)
		}
		if transport.body["autoToNeed"] != false {
			t.Fatalf("legacy cancellation lost: %v", transport.body)
		}
		value, exists := transport.body["conversionType"]
		if explicit {
			if value != float64(2) {
				t.Fatalf("explicit conversion lost: %v", transport.body)
			}
		} else if exists {
			t.Fatalf("legacy cancellation overridden by injected conversionType: %v", transport.body)
		}
	}
}

// TestRefreshedPluginCompletion exposes new groups, typed choices, and waiters
// through the same metadata used by command execution.
func TestRefreshedPluginCompletion(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"ecs", "instance", "convert-to-ondemand", "--conversion-type", ""}, "2"},
		{[]string{"ecpc", "settings", "two-factor-auth", "update", "--two-factor-auth-type", ""}, "VirtualMfa"},
		{[]string{"evs", "volume", "label", "update", "--action", ""}, "UNBIND"},
		{[]string{"acs", "instance", ""}, "reset-password"},
		{[]string{"acs", "instance", "show", "test-instance", "--wait", ""}, "acs.instance.running"},
	} {
		var stdout bytes.Buffer
		if err := cli.Run(cli.Config{Args: append([]string{"__complete"}, tc.args...), Stdout: &stdout, PluginRoot: t.TempDir()}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains("\n"+stdout.String(), "\n"+tc.want+"\n") {
			t.Errorf("completion for %v omitted %q: %s", tc.args, tc.want, stdout.String())
		}
	}
}
