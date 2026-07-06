/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestRegionSourceHelperBranches(t *testing.T) {
	bundle := plugin.Bundle{
		Commands: plugin.Commands{Commands: []plugin.Command{
			{ID: "ecs.list", Path: []string{"ecs", "list"}},
			{ID: "region.missing", Path: []string{"missing", "show", "{region_id}"}, Operation: "missing"},
			{ID: "region.zone", Path: []string{"zone", "show", "{region_id}"}, Operation: "v4.zone.show"},
			{
				ID:        "region.show",
				Path:      []string{"region", "show", "{region_id}"},
				Operation: "v4.region.show",
			},
		}},
		APIs: plugin.APIs{Operations: map[string]plugin.Operation{
			"v4.zone.show":   {Query: map[string]string{"zoneID": "$arg.zone_id"}},
			"v4.region.show": {Query: map[string]string{"regionID": "$arg.region_id"}},
		}},
	}

	if _, _, _, ok := findOptionalRegionCommand(bundle, []string{"region", "show", "81f7728662dd11ec810800155d307d5b"}); ok {
		t.Fatal("findOptionalRegionCommand matched when positional region_id was supplied")
	}
	command, args, rest, ok := findOptionalRegionCommand(bundle, []string{"region", "show", "--region", "81f7728662dd11ec810800155d307d5b"})
	if !ok || command.ID != "region.show" || len(args) != 0 || !slices.Equal(rest, []string{"--region", "81f7728662dd11ec810800155d307d5b"}) {
		t.Fatalf("findOptionalRegionCommand = %#v %#v %#v %v", command, args, rest, ok)
	}
	if _, _, ok := pluginPathPrefix([]string{"region", "show"}, []string{"region"}); ok {
		t.Fatal("pluginPathPrefix matched too-short input")
	}
	if _, _, ok := pluginPathPrefix([]string{"region", "show"}, []string{"region", "bad"}); ok {
		t.Fatal("pluginPathPrefix matched mismatched literal")
	}
	prefixArgs, rest, ok := pluginPathPrefix([]string{"region", "{name}"}, []string{"region", "show", "--region", "81f7728662dd11ec810800155d307d5b"})
	if !ok || prefixArgs["name"] != "show" || !slices.Equal(rest, []string{"--region", "81f7728662dd11ec810800155d307d5b"}) {
		t.Fatalf("pluginPathPrefix placeholder = %#v %#v %v", prefixArgs, rest, ok)
	}

	profileOperation := plugin.Operation{Query: map[string]string{"regionID": "$profile.region"}}
	regionParameter := []plugin.Parameter{{Name: "region", Target: "regionID"}}
	if operationMissingProfileRegion(plugin.Operation{}, nil, nil, nil) {
		t.Fatal("operationMissingProfileRegion required region for operation without region source")
	}
	if operationMissingProfileRegion(profileOperation, nil, regionParameter, map[string]string{"region": "81f7728662dd11ec810800155d307d5b"}) {
		t.Fatal("operationMissingProfileRegion ignored region option for profile source")
	}
	argOperation := plugin.Operation{Query: map[string]string{"regionID": "$arg.region_id"}}
	if !operationMissingProfileRegion(argOperation, nil, nil, nil) {
		t.Fatal("operationMissingProfileRegion ignored missing argument and missing option")
	}
	if operationMissingProfileRegion(argOperation, map[string]string{"region_id": "81f7728662dd11ec810800155d307d5b"}, nil, nil) {
		t.Fatal("operationMissingProfileRegion ignored positional region_id")
	}
	if !operationMissingProfileRegion(argOperation, nil, regionParameter, map[string]string{"region": "81f7728662dd11ec810800155d307d5b"}) {
		t.Fatal("operationMissingProfileRegion accepted --region for argument source")
	}
	if got := pluginCommandArgumentDescription(plugin.Bundle{}, plugin.Command{ID: "demo"}, "missing", "en-US"); got != "" {
		t.Fatalf("pluginCommandArgumentDescription empty fallback = %q", got)
	}
	fallbackBundle := plugin.Bundle{Tables: plugin.Tables{Tables: map[string]plugin.Table{
		"current": {Columns: []plugin.TableColumn{{Key: "region_id", Labels: map[string]string{"en-US": "Current Region ID"}}}},
	}}}
	if got := pluginCommandArgumentDescription(fallbackBundle, plugin.Command{ID: "demo", Table: "current"}, "region_id", "en-US"); got != "Current Region ID" {
		t.Fatalf("pluginCommandArgumentDescription command table fallback = %q", got)
	}
	scanBundle := plugin.Bundle{Tables: plugin.Tables{Tables: map[string]plugin.Table{
		"fallback": {Columns: []plugin.TableColumn{{Key: "region_id", Labels: map[string]string{"en-US": "Region ID"}}}},
	}}}
	if got := pluginCommandArgumentDescription(scanBundle, plugin.Command{ID: "demo", Table: "missing"}, "region_id", "en-US"); got != "Region ID" {
		t.Fatalf("pluginCommandArgumentDescription table scan fallback = %q", got)
	}
}

func TestRegionArgumentCommandRejectsRegionOption(t *testing.T) {
	pluginRoot := t.TempDir()
	writeArgumentRegionOptionBundle(t, filepath.Join(pluginRoot, "region"))

	_, _, _, _, _, err := findPluginCommand([]string{"region", "show", "--region"}, pluginRoot, "en-US")
	if err == nil {
		t.Fatal("findPluginCommand returned nil error for missing --region value")
	}
}

func TestPluginCommandRequiresProfileRegionWhenOperationUsesProfileRegion(t *testing.T) {
	pluginRoot := t.TempDir()
	writeIMSBundleWithoutFixture(t, filepath.Join(pluginRoot, "ims"))
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("HTTP transport should not be called without profile region")
		return nil, nil
	})

	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ims", "image", "list"},
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		PluginRoot:    pluginRoot,
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
    "default": {
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err == nil {
		t.Fatal("Run returned nil error for missing profile region")
	}
	requireDiagnosticKey(t, err, "error.missing_profile_region")
}

func TestPluginCommandUsesRegionOptionWhenProfileRegionIsMissing(t *testing.T) {
	pluginRoot := t.TempDir()
	writeIMSBundleWithoutFixture(t, filepath.Join(pluginRoot, "ims"))
	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(req.Body)
		seenBody = string(data)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[]}}`)),
		}, nil
	})

	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ims", "image", "list", "--region", "100054c0416811e9a6690242ac110002"},
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		PluginRoot:    pluginRoot,
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
    "default": {
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(seenBody, `"regionID":"100054c0416811e9a6690242ac110002"`) {
		t.Fatalf("request body did not use region option: %s", seenBody)
	}
}

func TestTrailingRegionArgumentFallsBackToProfileRegion(t *testing.T) {
	pluginRoot := t.TempDir()
	writeArgumentRegionOptionBundle(t, filepath.Join(pluginRoot, "region"))
	var seenQuery string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seenQuery = req.URL.RawQuery
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"regionID":"81f7728662dd11ec810800155d307d5b"}}`)),
		}, nil
	})

	err := Run(Config{
		Args:          []string{"--lang", "en-US", "region", "show"},
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           regionTestCredentials,
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "endpoint_url": "https://ctapi.example.test",
      "region": "81f7728662dd11ec810800155d307d5b"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(seenQuery, "regionID=81f7728662dd11ec810800155d307d5b") {
		t.Fatalf("request query did not use profile region: %s", seenQuery)
	}
}

func TestTrailingRegionArgumentRejectsRegionOption(t *testing.T) {
	pluginRoot := t.TempDir()
	writeArgumentRegionOptionBundle(t, filepath.Join(pluginRoot, "region"))
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("HTTP transport should not be called when region argument command receives --region")
		return nil, nil
	})

	err := Run(Config{
		Args:          []string{"--lang", "en-US", "region", "show", "--region", "100054c0416811e9a6690242ac110002"},
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           regionTestCredentials,
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err == nil {
		t.Fatal("Run returned nil error for duplicate region option")
	}
}

func regionTestCredentials(key string) string {
	switch key {
	case "CTYUN_AK":
		return "ak-test"
	case "CTYUN_SK":
		return "sk-test"
	default:
		return ""
	}
}
