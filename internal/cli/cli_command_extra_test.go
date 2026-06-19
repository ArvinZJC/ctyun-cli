/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestPluginCommandParsingAndPayloadErrors(t *testing.T) {
	command := plugin.Command{
		ID:   "ecs.instance.list",
		Path: []string{"ecs", "instance", "list"},
		Parameters: []plugin.Parameter{{
			Name: "name", Flag: "name", Target: "displayName", Required: true, Pattern: "[",
		}},
	}
	if _, err := parseCommandParameters(command, []string{"unexpected"}, "zh-CN"); err == nil || !strings.Contains(err.Error(), "不支持参数") {
		t.Fatalf("unexpected argument error = %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--name"}, "zh-CN"); err == nil || !strings.Contains(err.Error(), "需要一个值") {
		t.Fatalf("requires value error = %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--bad", "x"}, "zh-CN"); err == nil || !strings.Contains(err.Error(), "不支持选项") {
		t.Fatalf("unknown option error = %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--name", "x"}, "zh-CN"); err == nil || !strings.Contains(err.Error(), "校验表达式无效") {
		t.Fatalf("invalid pattern error = %v", err)
	}

	command.Parameters[0].Pattern = "^[a-z]+$"
	if _, err := parseCommandParameters(command, nil, "zh-CN"); err == nil || !strings.Contains(err.Error(), "需要 --name") {
		t.Fatalf("missing required error = %v", err)
	}
	if _, err := parseCommandParameters(command, []string{"--name=bad name"}, "zh-CN"); err == nil || !strings.Contains(err.Error(), "不匹配") {
		t.Fatalf("pattern mismatch error = %v", err)
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
}

func TestRunPluginCommandWriterWaiterAndOutputErrors(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))
	profile := coreconfig.Profile{}
	getenv := func(string) string { return "" }

	if err := runPluginCommand(failingWriter{}, io.Discard, globalOptions{Output: "json", Offline: true}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for JSON writer failure")
	}
	if err := runPluginCommand(failingWriter{}, io.Discard, globalOptions{Output: "table", Offline: true}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for table writer failure")
	}
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "yaml", Offline: true}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
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
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "table", Offline: true}, []string{"ecs", "instance", "delete", "ins-demo-1"}, dangerRoot, profile, getenv, nil); err == nil || !strings.Contains(err.Error(), "ecs.instance.delete") {
		t.Fatalf("runPluginCommand default dangerous message error = %v", err)
	}
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "table", Offline: true, Waiter: "missing"}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for unknown waiter")
	}
	if err := renderWaiter(io.Discard, plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"bad": {Path: "missing.path", Success: "ok"}}}}, "bad", map[string]any{}, func() (map[string]any, error) {
		return nil, nil
	}); err == nil {
		t.Fatal("renderWaiter returned nil error for missing waiter path")
	}
	if err := renderWaiter(io.Discard, plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"bad": {Path: "returnObj.status", Success: "ok", MaxAttempts: 2}}}}, "bad", map[string]any{"returnObj": map[string]any{"status": "pending"}}, func() (map[string]any, error) {
		return nil, errors.New("reload failed")
	}); err == nil {
		t.Fatal("renderWaiter returned nil error for reload failure")
	}
	reloaded := false
	if err := renderWaiter(io.Discard, plugin.Bundle{Waiters: plugin.Waiters{Waiters: map[string]plugin.Waiter{"slow": {Path: "returnObj.status", Success: "ok", MaxAttempts: 2, IntervalSeconds: 1}}}}, "slow", map[string]any{"returnObj": map[string]any{"status": "pending"}}, func() (map[string]any, error) {
		reloaded = true
		return map[string]any{"returnObj": map[string]any{"status": "ok"}}, nil
	}); err != nil {
		t.Fatalf("renderWaiter interval reload returned error: %v", err)
	}
	if !reloaded {
		t.Fatal("renderWaiter did not reload pending payload")
	}
}

func TestRunPluginCommandDataErrors(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))
	profile := coreconfig.Profile{}
	getenv := func(string) string { return "" }

	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "table", Offline: true}, []string{"missing", "command"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for unknown command")
	}
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "table", Offline: true, Filter: "bad"}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for invalid table filter")
	}
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "table", Offline: true, Sort: "-"}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for invalid table sort")
	}
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "table", Offline: true, Columns: []string{"missing"}}, []string{"ecs", "instance", "show", "ins-demo-1"}, pluginRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for unknown column")
	}

	badRoot := t.TempDir()
	writeMalformedRowsBundle(t, filepath.Join(badRoot, "ecs"), `{"returnObj":{"items":"not-array"}}`)
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "table", Offline: true}, []string{"ecs", "item", "list"}, badRoot, profile, getenv, nil); err == nil {
		t.Fatal("runPluginCommand returned nil error for non-array rows")
	}
	badRoot = t.TempDir()
	writeMalformedRowsBundle(t, filepath.Join(badRoot, "ecs"), `{"returnObj":{"items":[{"id":"one"}]}}`)
	if err := runPluginCommand(io.Discard, io.Discard, globalOptions{Output: "json", Offline: true}, []string{"ecs", "item", "list"}, badRoot, profile, getenv, nil); err != nil {
		t.Fatalf("runPluginCommand JSON control returned error: %v", err)
	}
}

func TestLoadCommandResponseAndExecuteAPICommandErrors(t *testing.T) {
	bundle := plugin.Bundle{Dir: t.TempDir(), Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://ctapi.example.test"}}}
	command := plugin.Command{ID: "ecs.instance.list"}
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Offline: true}, coreconfig.Profile{}, nil, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error without fixture")
	}
	command.FixtureResponse = "missing.json"
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Offline: true}, coreconfig.Profile{}, nil, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error for missing fixture")
	}
	mustWrite(t, filepath.Join(bundle.Dir, "bad.json"), `{`)
	command.FixtureResponse = "bad.json"
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Offline: true}, coreconfig.Profile{}, nil, nil, nil, nil); err == nil {
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
	if compareVersion("0.1.0", "0.2.0") >= 0 || compareVersion("0.3.0", "0.2.0") <= 0 || compareVersion("0.2.0", "0.2") != 0 {
		t.Fatal("compareVersion ordering failed")
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
	if got := filterKey("status"); got != "" {
		t.Fatalf("filterKey invalid = %q, want empty", got)
	}
}

func writeMalformedRowsBundle(t *testing.T, dir, fixture string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create bundle fixtures: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
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
