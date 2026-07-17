/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

func TestPluginCommandParsingAndPayloadErrors(t *testing.T) {
	command := plugin.Command{
		ID:   "ecs.instance.list",
		Path: []string{"ecs", "instance", "list"},
		Parameters: []plugin.Parameter{{
			Name: "name", Flag: "name", Target: "displayName", Required: true, Pattern: "[",
		}},
	}
	_, err := parseCommandParameters(command, []string{"unexpected"}, "zh-CN")
	requireDiagnosticKey(t, err, "error.unexpected_argument")
	_, err = parseCommandParameters(command, []string{"--name"}, "zh-CN")
	requireDiagnosticKey(t, err, "error.option_requires_value")
	_, err = parseCommandParameters(command, []string{"--bad", "x"}, "zh-CN")
	requireDiagnosticKey(t, err, "error.unknown_option")
	if _, err := parseCommandParameters(command, []string{"--name", "x"}, "zh-CN"); err == nil || !strings.Contains(err.Error(), "校验表达式无效") {
		t.Fatalf("invalid pattern error = %v", err)
	}

	command.Parameters[0].Pattern = "^[a-z]+$"
	if _, err := parseCommandParameters(command, nil, "zh-CN"); err == nil || !strings.Contains(err.Error(), "缺少必填选项 --name") {
		t.Fatalf("missing required error = %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--name=bad name"}, "zh-CN"); err == nil || !strings.Contains(err.Error(), "不匹配") {
		t.Fatalf("pattern mismatch error = %v", err)
	}

	command = plugin.Command{
		ID: "region.demand.check",
		Parameters: []plugin.Parameter{
			{Name: "productType", Flag: "product-type", Target: "productType", Required: true},
			{Name: "flavorID", Flag: "flavor-id", Target: "flavorID"},
			{Name: "specName", Flag: "spec-name", Target: "specName"},
			{Name: "ebsType", Flag: "ebs-type", Target: "ebsType"},
			{Name: "ebsSize", Flag: "ebs-size", Target: "ebsSize"},
		},
		ConditionalRequirements: []plugin.ConditionalRequirement{
			{
				When:  plugin.ParameterCondition{Parameter: "productType", Equals: "ecs"},
				AnyOf: []string{"flavorID", "specName"},
			},
			{
				When:     plugin.ParameterCondition{Parameter: "productType", Equals: "ebs"},
				Required: []string{"ebsType", "ebsSize"},
			},
		},
	}
	if _, err := parseCommandParameters(command, []string{"--product-type", "ecs"}, "en-US"); err == nil || !strings.Contains(err.Error(), "requires one of --flavor-id, --spec-name when --product-type is ecs") {
		t.Fatalf("missing conditional any-of error = %v", err)
	} else if strings.Contains(err.Error(), "region.demand.check") {
		t.Fatalf("missing conditional any-of error leaked command id: %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--product-type", "ebs", "--ebs-type", "SATA"}, "en-US"); err == nil || !strings.Contains(err.Error(), "requires --ebs-size when --product-type is ebs") {
		t.Fatalf("missing conditional flag error = %v", err)
	} else if strings.Contains(err.Error(), "region.demand.check") {
		t.Fatalf("missing conditional flag error leaked command id: %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--product-type", "ecs", "--spec-name", "s7.small.1"}, "en-US"); err != nil {
		t.Fatalf("parseCommandParameters returned error with conditional any-of satisfied: %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--product-type", "ebs", "--ebs-type", "SATA", "--ebs-size", "30"}, "en-US"); err != nil {
		t.Fatalf("parseCommandParameters returned error with conditional required fields satisfied: %v", err)
	}
	if parameterConditionMatches(plugin.ParameterCondition{In: []string{"ecs"}}, "") {
		t.Fatal("parameterConditionMatches matched empty value")
	}
	if !parameterConditionMatches(plugin.ParameterCondition{In: []string{"ecs", "ebs"}}, "ecs") {
		t.Fatal("parameterConditionMatches did not match listed value")
	}

	table := plugin.Table{RowPath: "items", Columns: []plugin.TableColumn{{Key: "id", Path: "id"}}}
	if _, err := rowsFromPayload(map[string]any{"items": "not-array"}, table); err == nil {
		t.Fatal("rowsFromPayload returned nil error for non-array rows")
	}
	if _, err := rowsFromPayload(map[string]any{}, table); err == nil {
		t.Fatal("rowsFromPayload returned nil error for missing row path")
	}
	if _, err := rowsFromPayload(map[string]any{"items": []any{"not-object"}}, table); err == nil {
		t.Fatal("rowsFromPayload returned nil error for non-object row")
	}
	if _, err := valueAtPath(map[string]any{"items": "not-object"}, "items.id"); err == nil {
		t.Fatal("valueAtPath returned nil error for non-object")
	}
	if _, err := valueAtPath(map[string]any{"items": map[string]any{}}, "items.id"); err == nil {
		t.Fatal("valueAtPath returned nil error for missing key")
	}
	rootTable := plugin.Table{RowPath: "$", Columns: []plugin.TableColumn{{Key: "status", Path: "status"}}}
	rootRows, err := rowsFromPayload(map[string]any{"status": "success"}, rootTable)
	if err != nil || len(rootRows) != 1 || rootRows[0]["status"] != "success" {
		t.Fatalf("rowsFromPayload root object = %#v, %v", rootRows, err)
	}
}

func TestCommandDisplayPath(t *testing.T) {
	command := plugin.Command{Path: []string{"ecs", "instance", "delete", "{instance_id}"}}
	if got := commandDisplayPath(command); got != "ecs instance delete {instance_id}" {
		t.Fatalf("commandDisplayPath() = %q, want visible path", got)
	}
	if got := commandDisplayPath(plugin.Command{}); got != "plugin command" {
		t.Fatalf("commandDisplayPath() = %q, want fallback subject", got)
	}
}

func TestParameterConditionalHintUsesInValues(t *testing.T) {
	command := plugin.Command{
		Parameters: []plugin.Parameter{
			{Name: "productType", Flag: "product-type", Target: "productType"},
			{Name: "flavorID", Flag: "flavor-id", Target: "flavorID"},
		},
		ConditionalRequirements: []plugin.ConditionalRequirement{{
			When:     plugin.ParameterCondition{Parameter: "productType", In: []string{"ecs", "gpu"}},
			Required: []string{"flavorID"},
		}},
	}

	got := parameterConditionalHint(command, command.Parameters[1], "en-US")
	if !strings.Contains(got, "--product-type is ecs|gpu") {
		t.Fatalf("parameterConditionalHint = %q, want joined in-values", got)
	}
}

func TestParameterConditionalHintSkipsIncompleteMetadata(t *testing.T) {
	parameter := plugin.Parameter{Name: "name", Flag: "name"}
	tests := []plugin.Command{
		{
			Parameters: []plugin.Parameter{parameter},
			ConditionalRequirements: []plugin.ConditionalRequirement{{
				When:     plugin.ParameterCondition{Parameter: "missing", Equals: "ecs"},
				Required: []string{"name"},
			}},
		},
		{
			Parameters: []plugin.Parameter{{Name: "productType", Flag: "product-type"}, parameter},
			ConditionalRequirements: []plugin.ConditionalRequirement{{
				When:     plugin.ParameterCondition{Parameter: "productType"},
				Required: []string{"name"},
			}},
		},
		{
			Parameters: []plugin.Parameter{{Name: "productType", Flag: "product-type"}, parameter},
			ConditionalRequirements: []plugin.ConditionalRequirement{{
				When:     plugin.ParameterCondition{Parameter: "productType", Equals: "ecs"},
				Required: []string{"other"},
			}},
		},
	}
	for _, command := range tests {
		if got := parameterConditionalHint(command, parameter, "en-US"); got != "" {
			t.Fatalf("parameterConditionalHint = %q, want empty", got)
		}
	}
	if value := parameterConditionValue(plugin.ParameterCondition{}); value != "" {
		t.Fatalf("empty parameter condition value = %q", value)
	}
	if _, ok := commandParameterByName(plugin.Command{}, "missing"); ok {
		t.Fatal("commandParameterByName found missing parameter")
	}
}

func TestRowsFromPayloadFormatsArrayCells(t *testing.T) {
	table := plugin.Table{
		RowPath: "items",
		Columns: []plugin.TableColumn{
			{Key: "zones", Path: "zones"},
		},
	}
	rows, err := rowsFromPayload(map[string]any{
		"items": []any{
			map[string]any{"zones": []any{"az1", "az2", "az3"}},
		},
	}, table)
	if err != nil {
		t.Fatalf("rowsFromPayload returned error: %v", err)
	}
	if got := rows[0]["zones"]; got != "az1, az2, az3" {
		t.Fatalf("array cell = %q, want comma-separated values", got)
	}
	if got := formatTableCell(map[string]any{"enabled": true, "id": "az1"}); got != "enabled=true; id=az1" {
		t.Fatalf("object cell = %q, want readable scalar fields", got)
	}
	if got := formatTableCell(map[string]any{"c": []any{"c6", "c7"}, "s": []any{"s6", "s7"}}); got != "c=c6, c7; s=s6, s7" {
		t.Fatalf("object array cell = %q, want readable scalar array fields", got)
	}
	if got := formatTableCell(map[string]any{}); got != "{}" {
		t.Fatalf("empty object cell = %q, want JSON object", got)
	}
	if got := formatTableCell(map[string]any{"mixed": []any{"az1", map[string]any{"id": "az2"}}}); got != "mixed=az1, {id=az2}" {
		t.Fatalf("mixed array cell = %q, want readable nested values", got)
	}
	if got := formatTableCell(map[string]any{"nested": map[string]any{"id": "az1"}}); got != "nested={id=az1}" {
		t.Fatalf("nested object cell = %q, want readable nested object", got)
	}
	if got := formatTableCell(map[string]any{"detail": map[string]any{"bb9fdb42056f11eda1610242ac110002": float64(2)}, "outer_pool_count": float64(0), "total_count": float64(2)}); got != "detail={bb9fdb42056f11eda1610242ac110002=2}; outer_pool_count=0; total_count=2" {
		t.Fatalf("nested count object cell = %q, want consistent key=value formatting", got)
	}
	if got := formatTableCell(map[string]any{"nested": map[string]any{"bad": func() {}}}); got == "" {
		t.Fatal("nested object cell with marshal error rendered empty")
	}
	if got, err := valueAtPathParts("leaf", nil, ""); err != nil || got != "leaf" {
		t.Fatalf("valueAtPathParts terminal = %#v, %v; want leaf, nil", got, err)
	}
	if _, err := valueAtPath([]any{map[string]any{}}, "id"); err == nil {
		t.Fatal("valueAtPath returned nil error for missing projected key")
	}
}

// TestRowsFromPayloadRendersNullCellAsEmpty verifies that an explicit JSON
// null is presented like an absent optional value in table output.
func TestRowsFromPayloadRendersNullCellAsEmpty(t *testing.T) {
	table := plugin.Table{
		RowPath: "items",
		Columns: []plugin.TableColumn{
			{Key: "name", Path: "name"},
		},
	}
	rows, err := rowsFromPayload(map[string]any{
		"items": []any{map[string]any{"name": nil}},
	}, table)
	if err != nil {
		t.Fatalf("rowsFromPayload returned error: %v", err)
	}
	if got := rows[0]["name"]; got != "" {
		t.Fatalf("null table cell = %q, want empty", got)
	}
}

func TestRowsFromPayloadProjectsArrayObjectCells(t *testing.T) {
	table := plugin.Table{
		RowPath: "items",
		Columns: []plugin.TableColumn{
			{Key: "storage_types", Path: "storage.type"},
			{Key: "nested_types", Path: "zones.details.storageType.type"},
		},
	}

	rows, err := rowsFromPayload(map[string]any{
		"items": []any{
			map[string]any{
				"storage": []any{
					map[string]any{"type": "SATA"},
					map[string]any{"type": "SSD"},
				},
				"zones": []any{
					map[string]any{"details": map[string]any{"storageType": []any{
						map[string]any{"type": "SAS"},
					}}},
					map[string]any{"details": map[string]any{"storageType": []any{
						map[string]any{"type": "SATA-KUNPENG"},
					}}},
				},
			},
		},
	}, table)
	if err != nil {
		t.Fatalf("rowsFromPayload returned error: %v", err)
	}
	if got := rows[0]["storage_types"]; got != "SATA, SSD" {
		t.Fatalf("projected array cell = %q, want leaf values", got)
	}
	if got := rows[0]["nested_types"]; got != "SAS, SATA-KUNPENG" {
		t.Fatalf("nested projected array cell = %q, want flattened leaf values", got)
	}
}

func TestRunPluginCommandWriterWaiterAndOutputErrors(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))
	profile := coreconfig.Profile{}
	getenv := func(string) string { return "" }

	if err := runPluginCommand(failingWriter{}, io.Discard, strings.NewReader(""), globalOptions{Output: "json", Fixture: true}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for JSON writer failure")
	}
	if err := runPluginCommand(failingWriter{}, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for table writer failure")
	}
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "yaml", Fixture: true}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for unsupported output")
	}
	dangerRoot := t.TempDir()
	writeDangerBundle(t, filepath.Join(dangerRoot, "ecs"))
	mustWrite(t, filepath.Join(dangerRoot, "ecs", "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.delete",
      "path": ["ecs", "instance", "delete", "{instance_id}"],
      "operation": "v4.ecs.instance.delete",
      "table": "ecs.instance.delete",
      "fixture_response": "fixtures/delete.json",
      "dangerous": {"confirm": "yes"}
    }
  ]
}`)
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true}, []string{"ecs", "instance", "delete", "ins-demo-1"}, dangerRoot, profile, getenv, nil); err == nil {
		t.Fatalf("runPluginCommand default dangerous message error = %v", err)
	}
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true, Waiter: "missing"}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for unknown waiter")
	}
	if err := renderWaiter(io.Discard, plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"bad": {Path: "missing.path", Success: "ok"}}}}, "bad", map[string]any{}, func() (map[string]any, error) {
		return nil, nil
	}, "en-US"); err == nil {
		t.Fatal("renderWaiter returned nil error for missing waiter path")
	}
	if err := renderWaiter(io.Discard, plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"bad": {Path: "returnObj.status", Success: "ok", MaxAttempts: 2}}}}, "bad", map[string]any{"returnObj": map[string]any{"status": "pending"}}, func() (map[string]any, error) {
		return nil, errors.New("reload failed")
	}, "en-US"); err == nil {
		t.Fatal("renderWaiter returned nil error for reload failure")
	}
	reloaded := false
	if err := renderWaiter(io.Discard, plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"slow": {Path: "returnObj.status", Success: "ok", MaxAttempts: 2, IntervalSeconds: 1}}}}, "slow", map[string]any{"returnObj": map[string]any{"status": "pending"}}, func() (map[string]any, error) {
		reloaded = true
		return map[string]any{"returnObj": map[string]any{"status": "ok"}}, nil
	}, "en-US"); err != nil {
		t.Fatalf("renderWaiter interval reload returned error: %v", err)
	}
	if !reloaded {
		t.Fatal("renderWaiter did not reload pending payload")
	}
	if err := renderWaiter(failingWriter{}, plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"ok": {Path: "returnObj.status", Success: "ok"}}}}, "ok", map[string]any{"returnObj": map[string]any{"status": "ok"}}, func() (map[string]any, error) {
		return nil, nil
	}, "en-US"); err == nil {
		t.Fatal("renderWaiter returned nil error for writer failure")
	}
}

func TestPluginCommandAcceptsGuardedOperationStatus(t *testing.T) {
	pluginRoot := t.TempDir()
	bundleDir := filepath.Join(pluginRoot, "ims")
	writeIMSBundleWithoutFixture(t, bundleDir)
	mustWrite(t, filepath.Join(bundleDir, "apis.json"), `{
  "operations": {
    "v4.ims.image.list": {
      "method": "POST",
      "path": "/v4/ims/image/list",
      "content_type": "application/json",
      "accepted_statuses": [{"code": "900", "required_path": "returnObj.images"}],
      "body": {"regionID": "$profile.region"}
    }
  }
}`)

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"statusCode":900,"returnObj":{"images":[{"imageID":"img-demo-1","name":"base"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"ims", "image", "list", "--cols", "image_id"},
		Stdout:        &stdout,
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
      "region": "81f7728662dd11ec810800155d307d5b",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error for accepted statusCode 900: %v", err)
	}
	if !strings.Contains(stdout.String(), "img-demo-1") {
		t.Fatalf("output missing API response row:\n%s", stdout.String())
	}
}

func TestWarnConfigCredentialsReturnsWriterError(t *testing.T) {
	profile := coreconfig.Profile{AccessKey: "ak-test", SecretKey: "sk-test"}
	creds, err := coreconfig.ResolveCredentials(func(string) string { return "" }, profile)
	if err != nil {
		t.Fatalf("ResolveCredentials returned error: %v", err)
	}
	err = warnConfigCredentials(failingWriter{}, creds, func(string) string { return "" }, profile, "en-US")
	if err == nil {
		t.Fatal("warnConfigCredentials returned nil error for writer failure")
	}
}

func TestConfirmDangerousOperationCoversInputBranches(t *testing.T) {
	var stderr bytes.Buffer
	if err := confirmDangerousOperation(&stderr, strings.NewReader("YES"), globalOptions{Language: "en-US"}, "delete instance"); err != nil {
		t.Fatalf("YES confirmation returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "delete instance requires confirmation") {
		t.Fatalf("prompt missing subject: %q", stderr.String())
	}

	stderr.Reset()
	if err := confirmDangerousOperation(&stderr, strings.NewReader(""), globalOptions{Language: "en-US"}, "delete instance"); err == nil {
		t.Fatal("empty confirmation returned nil error")
	}

	if err := confirmDangerousOperation(failingWriter{}, strings.NewReader("y\n"), globalOptions{Language: "en-US"}, "delete instance"); err == nil {
		t.Fatal("failing prompt writer returned nil error")
	}

	stderr.Reset()
	if err := confirmDangerousOperation(&stderr, strings.NewReader(""), globalOptions{Language: "en-US", Yes: true}, "delete instance"); err != nil {
		t.Fatalf("--yes confirmation returned error: %v", err)
	}
	if stderr.String() != "" {
		t.Fatalf("--yes wrote prompt: %q", stderr.String())
	}
}

func TestRunPluginCommandDataErrors(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))
	profile := coreconfig.Profile{}
	getenv := func(string) string { return "" }

	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true}, []string{"missing", "command"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for unknown command")
	}
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true, Language: "en-US"}, []string{"ecs", "instance", "show"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for missing path argument")
	} else {
		requireDiagnosticKey(t, err, "error.missing_path_argument")
	}
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true, Filter: "bad"}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for invalid table filter")
	}
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true, Sort: "-"}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for invalid table sort")
	}
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true, Columns: []string{"missing"}}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for unknown column")
	}

	badRoot := t.TempDir()
	writeMalformedRowsBundle(t, filepath.Join(badRoot, "ecs"), `{"returnObj":{"items":"not-array"}}`)
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true}, []string{"ecs", "item", "list"}, badRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for non-array rows")
	}
	badRoot = t.TempDir()
	writeMalformedRowsBundle(t, filepath.Join(badRoot, "ecs"), `{"returnObj":{"items":[{"id":"one"}]}}`)
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "json", Fixture: true}, []string{"ecs", "item", "list"}, badRoot, profile, getenv, nil); err != nil {
		t.Fatalf("runPluginCommand JSON control returned error: %v", err)
	}
}

func TestLoadCommandResponseAndExecuteAPICommandErrors(t *testing.T) {
	bundle := plugin.Bundle{Dir: t.TempDir(), Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://ctapi.example.test"}}}
	command := plugin.Command{ID: "ecs.instance.list"}
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Fixture: true}, coreconfig.Profile{}, nil, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error without fixture")
	}
	command.FixtureResponse = "missing.json"
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Fixture: true}, coreconfig.Profile{}, nil, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error for missing fixture")
	}
	mustWrite(t, filepath.Join(bundle.Dir, "bad.json"), `{`)
	command.FixtureResponse = "bad.json"
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Fixture: true}, coreconfig.Profile{}, nil, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error for malformed fixture")
	}

	command = plugin.Command{ID: "ecs.instance.list", Operation: "missing"}
	if _, err := executeAPICommand(bundle, command, nil, nil, coreconfig.Profile{EndpointURL: "https://ctapi.example.test"}, func(string) string { return "" }, nil, nil, nil, "en-US"); err == nil {
		t.Fatal("executeAPICommand returned nil error for missing operation")
	}
	bundle.APIs = plugin.APIs{Operations: map[string]plugin.Operation{"op": {Method: http.MethodGet, Path: "/v4/demo"}}}
	command.Operation = "op"
	if _, err := executeAPICommand(plugin.Bundle{APIs: bundle.APIs}, command, nil, nil, coreconfig.Profile{}, func(string) string { return "" }, nil, nil, nil, "en-US"); err == nil {
		t.Fatal("executeAPICommand returned nil error without endpoint")
	}
	if _, err := executeAPICommand(bundle, command, nil, nil, coreconfig.Profile{EndpointURL: "https://ctapi.example.test"}, func(string) string { return "" }, nil, nil, nil, "en-US"); err == nil {
		t.Fatal("executeAPICommand returned nil error without credentials")
	}
	if _, err := executeAPICommand(bundle, command, nil, nil, coreconfig.Profile{EndpointURL: "https://ctapi.example.test", AccessKey: "ak-test", SecretKey: "sk-test"}, func(string) string { return "" }, nil, failingWriter{}, nil, "en-US"); err == nil {
		t.Fatal("executeAPICommand returned nil error for credential warning writer failure")
	}

	seenContentType := ""
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seenContentType = req.Header.Get("Content-Type")
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})
	if _, err := executeAPICommand(bundle, command, nil, nil, coreconfig.Profile{EndpointURL: "https://ctapi.example.test"}, func(key string) string {
		switch key {
		case "CTYUN_AK":
			return "ak-test"
		case "CTYUN_SK":
			return "sk-test"
		default:
			return ""
		}
	}, transport, nil, nil, "en-US"); err != nil {
		t.Fatalf("executeAPICommand default content type returned error: %v", err)
	}
	if seenContentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", seenContentType)
	}
}

func TestPluginRootsAndVersionComparison(t *testing.T) {
	configured := t.TempDir()
	if got := pluginRoot(configured); got != configured {
		t.Fatalf("pluginRoot configured = %q", got)
	}
	t.Setenv("HOME", "")
	if got := pluginRoot(""); got != ".ctyun/plugins" {
		t.Fatalf("pluginRoot no-home = %q", got)
	}
	chdirRepoRootForTest(t)
	if got := defaultPluginRoot(); got != "plugins" {
		t.Fatalf("defaultPluginRoot from repo root = %q, want plugins", got)
	}
	if got := defaultPluginRoot(); got == "" {
		t.Fatal("defaultPluginRoot returned empty path")
	}
	emptyRoot := t.TempDir()
	if err := os.Chdir(emptyRoot); err != nil {
		t.Fatalf("chdir empty root: %v", err)
	}
	originalRuntimeCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalRuntimeCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, "", 0, false
	}
	if got := defaultPluginRoot(); got != "plugins" {
		t.Fatalf("defaultPluginRoot without caller = %q, want plugins", got)
	}
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(emptyRoot, "nested", "cli.go"), 0, true
	}
	if got := defaultPluginRoot(); got != "plugins" {
		t.Fatalf("defaultPluginRoot without discoverable plugins = %q, want plugins", got)
	}
	runtimeCaller = originalRuntimeCaller
	if version.CompareSemanticVersions("0.1.0", "0.2.0") >= 0 ||
		version.CompareSemanticVersions("0.3.0", "0.2.0") <= 0 ||
		version.CompareSemanticVersions("0.2.0", "0.2.0-beta.1") <= 0 {
		t.Fatal("CompareSemanticVersions ordering failed")
	}
}

func TestLoadBundlesUsesBundledPluginsOnlyForDevelopmentBuilds(t *testing.T) {
	chdirRepoRootForTest(t)

	originalChannel := version.Channel
	t.Cleanup(func() { version.Channel = originalChannel })

	version.Channel = "dev"
	installedRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(installedRoot, "ecs"), "ecs", "9.9.9")
	bundles, err := loadBundles(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("loadBundles dev returned error: %v", err)
	}
	if len(bundles) == 0 {
		t.Fatal("loadBundles dev did not include bundled repo plugins")
	}
	bundles, err = loadBundles(installedRoot)
	if err != nil {
		t.Fatalf("loadBundles dev with installed conflict returned error: %v", err)
	}
	foundBundledECS := false
	for _, bundle := range bundles {
		if bundle.Manifest.Name == "ecs" {
			if bundle.Manifest.Version == "9.9.9" || len(bundle.Commands.Commands) == 0 {
				t.Fatalf("loadBundles dev preferred installed ecs bundle: %#v", bundle.Manifest)
			}
			foundBundledECS = true
			break
		}
	}
	if !foundBundledECS {
		t.Fatal("loadBundles dev did not include bundled ecs")
	}

	version.Channel = "stable"
	bundles, err = loadBundles(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("loadBundles stable returned error: %v", err)
	}
	if len(bundles) != 0 {
		t.Fatalf("loadBundles stable = %d bundles, want only installed plugins", len(bundles))
	}
}

func chdirRepoRootForTest(t *testing.T) {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
}

func TestPluginDirsIgnoresMissingAndUnreadableRoots(t *testing.T) {
	if dirs := pluginDirs(filepath.Join(t.TempDir(), "missing")); dirs != nil {
		t.Fatalf("pluginDirs missing = %#v, want nil", dirs)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if dirs := pluginDirs(rootFile); dirs != nil {
		t.Fatalf("pluginDirs file = %#v, want nil", dirs)
	}
	badRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(badRoot, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if _, _, _, _, _, err := findPluginCommand([]string{"bad"}, badRoot, "en-US"); err == nil {
		t.Fatal("findPluginCommand returned nil error for invalid plugin root")
	}
}

func TestLocalizedEnglishValidationErrors(t *testing.T) {
	if got := localizedUnexpectedArgument("extra", "cmd", "en-US").Error(); !strings.Contains(got, "unexpected argument") {
		t.Fatalf("localizedUnexpectedArgument English = %q", got)
	}
	if got := localizedFlagRequiresValue("name", "en-US").Error(); !strings.Contains(got, "requires a value") {
		t.Fatalf("localizedFlagRequiresValue English = %q", got)
	}
	if got := localizedUnknownOption("bad", "cmd", "en-US").Error(); !strings.Contains(got, "unknown option") {
		t.Fatalf("localizedUnknownOption English = %q", got)
	}
	if got := localizedInvalidPattern("cmd", "name", errors.New("bad"), "en-US").Error(); !strings.Contains(got, "invalid validation pattern") {
		t.Fatalf("localizedInvalidPattern English = %q", got)
	}
}

func TestFilteringHelpersCoverEmptyAndNoTargetCases(t *testing.T) {
	rows := []map[string]string{{"id": "one"}}
	if got := filterRowsByParameters(nil, plugin.Table{}, nil, nil); got != nil {
		t.Fatalf("filterRowsByParameters nil rows = %#v", got)
	}
	table := plugin.Table{Columns: []plugin.TableColumn{{Key: "id", Path: "id"}}}
	parameters := []plugin.Parameter{{Name: "name", Target: "missingPath"}}
	values := map[string]string{"name": "", "other": "ignored"}
	filtered := filterRowsByParameters(rows, table, parameters, values)
	if len(filtered) != 1 || filtered[0]["id"] != "one" {
		t.Fatalf("filterRowsByParameters = %#v", filtered)
	}

	table = plugin.Table{Columns: []plugin.TableColumn{{Key: "name", Path: "displayName"}}}
	parameters = []plugin.Parameter{{Name: "name", Target: "displayName"}}
	values = map[string]string{"name": "keep"}
	rows = []map[string]string{{"name": "keep"}, {"name": "drop"}}
	filtered = filterRowsByParameters(rows, table, parameters, values)
	if len(filtered) != 1 || filtered[0]["name"] != "keep" {
		t.Fatalf("filterRowsByParameters did not drop non-matching row: %#v", filtered)
	}
}

func TestVisibleExamplesHidesFixtureOnlyExamples(t *testing.T) {
	got := visibleExamples([]string{
		"ctyun ecs instance list",
		"ctyun --offline ecs instance list",
		"ctyun --fixture ecs instance list",
	})
	if len(got) != 1 || got[0] != "ctyun ecs instance list" {
		t.Fatalf("visibleExamples = %#v", got)
	}
}

func TestPluginCommandUsesTableDefaultColumns(t *testing.T) {
	pluginRoot := t.TempDir()
	bundleDir := filepath.Join(pluginRoot, "demo")
	if err := os.MkdirAll(filepath.Join(bundleDir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	mustWrite(t, filepath.Join(bundleDir, "plugin.json"), `{
  "name": "demo",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "demo", "ctyun_product_id": 25}
}`)
	mustWrite(t, filepath.Join(bundleDir, "commands.json"), `{"commands":[{"id":"demo.show","path":["demo","show"],"table":"demo.show","fixture_response":"fixtures/show.json"}]}`)
	mustWrite(t, filepath.Join(bundleDir, "tables.json"), `{"tables":{"demo.show":{"row_path":"returnObj","default_columns":["name"],"columns":[{"key":"name","path":"name","labels":{"zh-CN":"名称","en-US":"Name","en-GB":"Name"}},{"key":"details","path":"details","labels":{"zh-CN":"详情","en-US":"Details","en-GB":"Details"}}]}}}`)
	mustWrite(t, filepath.Join(bundleDir, "fixtures", "show.json"), `{"statusCode":800,"returnObj":{"name":"demo","details":"hidden by default"}}`)

	var stdout bytes.Buffer
	if err := runPluginCommand(&stdout, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true, Table: "plain", Language: "en-US"}, []string{"demo", "show"}, pluginRoot, coreconfig.Profile{}, func(string) string { return "" }, nil); err != nil {
		t.Fatalf("runPluginCommand returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Name") || strings.Contains(got, "Details") {
		t.Fatalf("default column output =\n%s", got)
	}

	stdout.Reset()
	if err := runPluginCommand(&stdout, io.Discard, strings.NewReader(""), globalOptions{Output: "table", Fixture: true, Table: "plain", Language: "en-US", Columns: []string{"details"}}, []string{"demo", "show"}, pluginRoot, coreconfig.Profile{}, func(string) string { return "" }, nil); err != nil {
		t.Fatalf("runPluginCommand with --cols returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Details") || strings.Contains(got, "Name") {
		t.Fatalf("explicit column output =\n%s", got)
	}
}

func writeMalformedRowsBundle(t *testing.T, dir, fixture string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create bundle fixtures: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[{"id":"ecs.item.list","path":["ecs","item","list"],"table":"ecs.items","fixture_response":"fixtures/items.json"}]}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{"tables":{"ecs.items":{"row_path":"returnObj.items","columns":[{"key":"id","path":"id","labels":{"zh-CN":"ID","en-US":"ID","en-GB":"ID"}}]}}}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "items.json"), fixture)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
