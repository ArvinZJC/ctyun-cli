/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestProductCommandOutputControlsAcceptLocalizedColumnLabels verifies the promoted bundle through the public CLI.
func TestProductCommandOutputControlsAcceptLocalizedColumnLabels(t *testing.T) {
	var stdout bytes.Buffer
	if err := cli.Run(cli.Config{
		Args:       []string{"--lang", "zh-CN", "--table", "plain", "--cols", "实例 ID,名称", "--filter", "名称=api-test01", "--sort", "实例 ID", "--no-header", "ecs", "instance", "list", "--offline"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("product command with localized output controls returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "c5a7966a-88e7-362b-6e11-c2d8fbfc07ca") || !strings.Contains(got, "api-test01") {
		t.Fatalf("localized output controls missing expected row:\n%s", got)
	}
	if strings.Contains(got, "状态") {
		t.Fatalf("localized output controls did not filter/select columns:\n%s", got)
	}
}

// TestECSInstanceListDefaultsToTable renders selected columns from the promoted ECS fixture.
func TestECSInstanceListDefaultsToTable(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(cli.Config{
		Args:   []string{"--lang", "en-US", "ecs", "instance", "list", "--offline", "--cols", "instance_id,name,status"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"Instance ID", "Name", "Status", "c5a7966a-88e7-362b-6e11-c2d8fbfc07ca", "api-test01", "running"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Private IP") {
		t.Fatalf("table output included unselected Private IP column:\n%s", got)
	}
}

// TestECSInstanceListSupportsJSONPassthrough preserves the captured ECS response envelope.
func TestECSInstanceListSupportsJSONPassthrough(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(cli.Config{
		Args:   []string{"ecs", "instance", "list", "--offline", "--output", "json"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, `"returnObj"`) || !strings.Contains(got, `"instanceID"`) {
		t.Fatalf("json output did not preserve CTyun-like payload:\n%s", got)
	}
}

// TestECSInstanceListSupportsFilterAndSort applies table controls to the promoted ECS fixture.
func TestECSInstanceListSupportsFilterAndSort(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(cli.Config{
		Args:   []string{"--lang", "en-US", "ecs", "instance", "list", "--offline", "--filter", "status=running", "--sort", "-instance_id", "--cols", "instance_id,status"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "c5a7966a-88e7-362b-6e11-c2d8fbfc07ca") {
		t.Fatalf("filtered output missing running instance:\n%s", got)
	}
}

// TestECSInstanceListAppliesParameterFiltersToFixtureRows applies the real name option binding to fixture rows.
func TestECSInstanceListAppliesParameterFiltersToFixtureRows(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(cli.Config{
		Args:   []string{"--lang", "en-US", "ecs", "instance", "list", "--offline", "--name", "api-test01", "--cols", "instance_id,name"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "api-test01") {
		t.Fatalf("parameter-filtered output missing api-test01:\n%s", got)
	}
}

// TestECSInstanceListRejectsUnknownFilterOrSortKeys rejects selectors absent from the promoted table definition.
func TestECSInstanceListRejectsUnknownFilterOrSortKeys(t *testing.T) {
	err := cli.Run(cli.Config{
		Args: []string{"ecs", "instance", "list", "--offline", "--filter", "实例ID=ins-demo-1"},
	})
	if err == nil {
		t.Fatal("Run returned nil error for translated filter key")
	}
	requireBundleDiagnosticKey(t, err, "error.unknown_filter_key")

	err = cli.Run(cli.Config{
		Args: []string{"ecs", "instance", "list", "--offline", "--sort", "displayName"},
	})
	if err == nil {
		t.Fatal("Run returned nil error for response-field sort key")
	}
	requireBundleDiagnosticKey(t, err, "error.unknown_sort_key")
}

// TestECSInstanceListLocalizesHeaders uses Chinese labels from the promoted ECS table.
func TestECSInstanceListLocalizesHeaders(t *testing.T) {
	var stdout bytes.Buffer
	err := cli.Run(cli.Config{
		Args:   []string{"--lang", "zh-CN", "ecs", "instance", "list", "--offline", "--cols", "instance_id,status"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "实例 ID") || !strings.Contains(got, "状态") {
		t.Fatalf("localized table output missing Chinese headers:\n%s", got)
	}
}

// TestShippedECSIncludesStateChangingStartWithConfirmation requires confirmation for the promoted start command.
func TestShippedECSIncludesStateChangingStartWithConfirmation(t *testing.T) {
	err := cli.Run(cli.Config{
		Args:  []string{"--lang", "en-US", "ecs", "instance", "start", "ins-demo-1", "--offline"},
		Stdin: strings.NewReader("n\n"),
	})
	if err == nil {
		t.Fatal("shipped ECS start without confirmation returned nil error")
	}
	requireBundleDiagnosticKey(t, err, "error.confirmation_cancelled")

	var stdout bytes.Buffer
	err = cli.Run(cli.Config{
		Args:   []string{"--lang", "en-US", "--yes", "ecs", "instance", "start", "ins-demo-1", "--offline"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("confirmed shipped ECS start returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Job ID") {
		t.Fatalf("confirmed start output missing job id table:\n%s", stdout.String())
	}
}

// TestShippedRegionListUsesOfficialPublicAPI preserves the promoted Region endpoint and request contract.
func TestShippedRegionListUsesOfficialPublicAPI(t *testing.T) {
	var seenHost, seenMethod, seenPath, seenQuery string
	transport := nativeWireTransport(func(req *http.Request) (*http.Response, error) {
		seenHost = req.URL.Host
		seenMethod = req.Method
		seenPath = req.URL.Path
		seenQuery = req.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
  "statusCode": 800,
  "message": "SUCCESS",
  "returnObj": {
    "regionList": [
      {
        "regionID": "41f64827f25f468595ffa3a5deb5d15d",
        "regionParent": "内蒙",
        "regionName": "内蒙",
        "regionType": "openstack",
        "isMultiZones": false,
        "regionCode": "cn-neimeng-8",
        "openapiAvailable": true
      }
    ]
  }
}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := cli.Run(cli.Config{
		Args:          []string{"--lang", "en-US", "region", "list", "--name", "内蒙", "--cols", "region_id,region_name,region_code"},
		Stdout:        &stdout,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {}
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if seenHost != "ctecs-global.ctapi.ctyun.cn" {
		t.Fatalf("host = %q, want plugin default endpoint", seenHost)
	}
	if seenMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", seenMethod)
	}
	if seenPath != "/v4/region/list-regions" {
		t.Fatalf("path = %q, want /v4/region/list-regions", seenPath)
	}
	if seenQuery != "regionName=%E5%86%85%E8%92%99" {
		t.Fatalf("query = %q, want encoded regionName", seenQuery)
	}
	if !strings.Contains(stdout.String(), "cn-neimeng-8") {
		t.Fatalf("stdout = %q, want region row", stdout.String())
	}
}

// requireBundleDiagnosticKey checks the public diagnostic contract for a bundle failure.
func requireBundleDiagnosticKey(t *testing.T, err error, want string) {
	t.Helper()
	diagnostic, ok := err.(interface{ MessageKey() string })
	if !ok || diagnostic.MessageKey() != want {
		t.Fatalf("error %v does not expose diagnostic key %q", err, want)
	}
}
