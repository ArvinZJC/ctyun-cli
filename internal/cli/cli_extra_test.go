/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

type fakeTempArtifactFile struct {
	name     string
	writeErr error
	closeErr error
}

func (file fakeTempArtifactFile) Name() string {
	return file.name
}

func (file fakeTempArtifactFile) Write([]byte) (int, error) {
	if file.writeErr != nil {
		return 0, file.writeErr
	}
	return 1, nil
}

func (file fakeTempArtifactFile) Close() error {
	return file.closeErr
}

func TestExecuteSuccessAndErrorLanguageFallbacks(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Execute(Config{Args: []string{"version"}, Stdout: &stdout, Stderr: &stderr}); code != 0 {
		t.Fatalf("Execute version code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	getenv := func(key string) string {
		if key == "LANG" {
			return "en_GB.UTF-8"
		}
		return ""
	}
	if got := errorLanguage(Config{Args: []string{"--lang"}}, getenv); got != "en-GB" {
		t.Fatalf("errorLanguage parse fallback = %q, want en-GB", got)
	}
	if got := errorLanguage(Config{Args: []string{"--config", filepath.Join(t.TempDir(), "missing-dir")}}, getenv); got != "en-GB" {
		t.Fatalf("errorLanguage config fallback = %q, want en-GB", got)
	}
	if got := errorLanguage(Config{Args: []string{"--config", t.TempDir()}}, getenv); got != "en-GB" {
		t.Fatalf("errorLanguage config read error fallback = %q, want en-GB", got)
	}
	if got := localizedErrorText("plain", "zh-CN"); got != "plain" {
		t.Fatalf("localizedErrorText plain = %q", got)
	}
	if got := localizedErrorText("plain", "en-US"); got != "plain" {
		t.Fatalf("localizedErrorText English = %q", got)
	}
	if got := formatError(errors.New("plain"), "en-US"); got != "Error: plain" {
		t.Fatalf("formatError = %q", got)
	}
}

func TestExecuteUsesDefaultStderrOnErrors(t *testing.T) {
	if code := Execute(Config{Args: []string{"--lang", "en-US", "nope"}}); code != 1 {
		t.Fatalf("Execute code = %d, want 1", code)
	}
}

func TestRunReturnsEarlyParseAndConfigErrors(t *testing.T) {
	if err := Run(Config{Args: []string{"--output"}}); err == nil {
		t.Fatal("Run returned nil error for missing --output value")
	}
	if err := Run(Config{Args: []string{"version"}, ConfigPath: t.TempDir()}); err == nil {
		t.Fatal("Run returned nil error for unreadable config path")
	}
}

func TestDetectOSLocaleFallsBackToLangOnOtherPlatforms(t *testing.T) {
	restoreOS := runtimeGOOS
	t.Cleanup(func() { runtimeGOOS = restoreOS })
	runtimeGOOS = "linux"

	got := detectOSLocale(func(key string) string {
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	})
	if got != "C.UTF-8" {
		t.Fatalf("detectOSLocale = %q, want LANG fallback", got)
	}
}

func TestLocaleReadersCoverUnavailablePlatformHelpers(t *testing.T) {
	t.Setenv("PATH", "")
	if got := readDarwinAppleLocale(); got != "" {
		t.Fatalf("readDarwinAppleLocale with missing defaults = %q, want empty", got)
	}
	if got := readWindowsUserLocale(); got != "" {
		t.Fatalf("readWindowsUserLocale fallback = %q, want empty", got)
	}
}

func TestCompletionVariantsAndErrors(t *testing.T) {
	for _, shell := range []string{"bash", "fish"} {
		var stdout bytes.Buffer
		if err := runCompletion(&stdout, []string{shell}, t.TempDir()); err != nil {
			t.Fatalf("runCompletion %s returned error: %v", shell, err)
		}
		if !strings.Contains(stdout.String(), "ctyun") {
			t.Fatalf("completion for %s missing ctyun: %s", shell, stdout.String())
		}
	}
	if err := runCompletion(io.Discard, nil, t.TempDir()); err == nil {
		t.Fatal("runCompletion returned nil error without shell")
	}
	if err := runCompletion(io.Discard, []string{"nushell"}, t.TempDir()); err == nil {
		t.Fatal("runCompletion returned nil error for unsupported shell")
	}

	badRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(badRoot, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if words := allCompletionWords(badRoot); len(words) == 0 {
		t.Fatal("allCompletionWords returned no core words when plugin loading failed")
	}
}

func TestCompletionIgnoresProductCommandAliases(t *testing.T) {
	root := t.TempDir()
	writeArgumentBundle(t, filepath.Join(root, "ims"))
	mustWrite(t, filepath.Join(root, "ims", "commands.json"), `{
  "commands": [
    {
      "id": "ims.image.show",
      "path": ["ims", "image", "show", "{image_id}"],
      "aliases": [["ims", "img", "{image_id}"]],
      "operation": "v4.ims.image.show",
      "table": "ims.image.show"
    }
  ]
}`)

	words := allCompletionWords(root)
	for _, word := range words {
		switch word {
		case "img", "{image_id}":
			t.Fatalf("allCompletionWords exposed unsupported alias word %q in %v", word, words)
		}
	}
}

func TestHelpHelpersCoverCoreAndFallbackText(t *testing.T) {
	for _, command := range []string{"completion", "doctor", "help", "plugin", "plugins", "upgrade", "update", "version"} {
		var stdout bytes.Buffer
		if !printCoreHelp(&stdout, []string{command}, "en-US") {
			t.Fatalf("printCoreHelp did not handle %s", command)
		}
		if !strings.Contains(stdout.String(), "Global Options") {
			t.Fatalf("core help for %s missing global options: %s", command, stdout.String())
		}
	}
	if printCoreHelp(io.Discard, []string{"unknown"}, "en-US") {
		t.Fatal("printCoreHelp handled unknown command")
	}
	if printCoreHelp(io.Discard, []string{"plugin", "install", "extra"}, "en-US") {
		t.Fatal("printCoreHelp handled plugin help with too many arguments")
	}
	if printCoreHelp(io.Discard, []string{"plugin", "missing"}, "en-US") {
		t.Fatal("printCoreHelp handled unknown plugin subcommand")
	}
	if got := helpText("missing.key", "en-US"); got != "missing.key" {
		t.Fatalf("missing helpText = %q", got)
	}
	if got := helpText("title", "en-AU"); !strings.Contains(got, "ctyun") {
		t.Fatalf("English helpText fallback = %q", got)
	}
	if got := helpText("title", "fr-FR"); !strings.Contains(got, "天翼云") {
		t.Fatalf("Chinese helpText fallback = %q", got)
	}
	helpCatalog["test.no.fallback"] = map[string]string{"fr-FR": "bonjour"}
	t.Cleanup(func() { delete(helpCatalog, "test.no.fallback") })
	if got := helpText("test.no.fallback", "de-DE"); got != "test.no.fallback" {
		t.Fatalf("helpText final fallback = %q", got)
	}

	bundle := plugin.Bundle{I18N: map[string]map[string]string{"zh-CN": {"name": "弹性云主机"}}}
	if got := localizedPluginText(bundle, "", "name", "fallback"); got != "弹性云主机" {
		t.Fatalf("localizedPluginText zh default = %q", got)
	}
	if got := localizedPluginText(bundle, "en-US", "name", "fallback"); got != "fallback" {
		t.Fatalf("localizedPluginText fallback = %q", got)
	}

	var stdout bytes.Buffer
	printPluginCommandHints(&stdout, "en-US")
	if !strings.Contains(stdout.String(), "ctyun plugin list") || !strings.Contains(stdout.String(), "ctyun help <plugin>") {
		t.Fatalf("printPluginCommandHints output = %q", stdout.String())
	}
	if pathHasPrefix([]string{"ecs"}, nil) {
		t.Fatal("pathHasPrefix matched an empty prefix")
	}
	if pathHasPrefix([]string{"ecs"}, []string{"ecs"}) {
		t.Fatal("pathHasPrefix matched a complete command path")
	}
}

func TestRunHelpErrorsForInvalidOrUnknownPluginCommands(t *testing.T) {
	badRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(badRoot, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if err := runHelp(io.Discard, []string{"bad"}, badRoot, "en-US"); err == nil {
		t.Fatal("runHelp returned nil error for invalid plugin root")
	}
	if err := runHelp(io.Discard, []string{"missing"}, t.TempDir(), "en-US"); err == nil {
		t.Fatal("runHelp returned nil error for unknown command")
	}
}

func TestRunHelpShowsRequiredParameters(t *testing.T) {
	root := t.TempDir()
	bundleDir := filepath.Join(root, "ecs")
	writeValidationBundle(t, bundleDir)
	mustWrite(t, filepath.Join(bundleDir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "table": "ecs.instance.list",
      "fixture_response": "fixtures/instance-list.json",
      "parameters": [
        {"name": "status", "flag": "status", "target": "status", "allowed_values": ["running", "stopped"], "description": "Status"},
        {"name": "name", "flag": "name", "target": "displayName", "required": true, "pattern": "^[A-Za-z0-9-]+$", "description": "Instance name"}
      ]
    }
  ]
}`)
	var stdout bytes.Buffer
	if err := runHelp(&stdout, []string{"ecs", "instance", "list"}, root, "en-US"); err != nil {
		t.Fatalf("runHelp returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "(required)") {
		t.Fatalf("help output missing required marker:\n%s", stdout.String())
	}
}

func TestParameterValidationHintAndDoctorErrors(t *testing.T) {
	parameter := plugin.Parameter{AllowedValues: []string{"running", "stopped"}, Pattern: "^[a-z]+$"}
	if got := parameterValidationHint(parameter, "en-US"); !strings.Contains(got, "one of running,stopped") || !strings.Contains(got, "matches") {
		t.Fatalf("English validation hint = %q", got)
	}
	if got := parameterValidationHint(parameter, "zh-CN"); !strings.Contains(got, "可选值") || !strings.Contains(got, "匹配") {
		t.Fatalf("Chinese validation hint = %q", got)
	}
	if got := parameterValidationHint(plugin.Parameter{}, "en-US"); got != "" {
		t.Fatalf("empty validation hint = %q", got)
	}
	if err := runDoctor(io.Discard, nil); err == nil {
		t.Fatal("runDoctor returned nil error for missing subcommand")
	}
}

func TestParseGlobalOptionsErrorsAndFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--output"}, {"--cols"}, {"--wait"}, {"--table"}, {"--timeout"}, {"--config"}, {"--profile"}, {"--filter"}, {"--sort"}, {"--lang"},
	} {
		if _, _, err := parseGlobalOptions(args); err == nil {
			t.Fatalf("parseGlobalOptions returned nil error for %v", args)
		}
	}
	if _, _, err := parseGlobalOptions([]string{"--timeout", "0"}); err == nil {
		t.Fatal("parseGlobalOptions returned nil error for zero timeout")
	}
	opts, rest, err := parseGlobalOptions([]string{"--no-header", "--debug", "--help", "version"})
	if err != nil {
		t.Fatalf("parseGlobalOptions returned error: %v", err)
	}
	if !opts.NoHeader || !opts.Debug || !opts.Help || opts.Offline || strings.Join(rest, " ") != "version" {
		t.Fatalf("parsed opts = %+v rest=%v", opts, rest)
	}
}

func TestConfigLoadingPathsAndActiveProfileErrors(t *testing.T) {
	injected := []byte(`{"profiles":{}}`)
	got, err := loadConfigBytes(injected, filepath.Join(t.TempDir(), "ignored"))
	if err != nil || string(got) != string(injected) {
		t.Fatalf("loadConfigBytes injected = %q, %v", string(got), err)
	}
	if got, err := loadConfigBytes(nil, filepath.Join(t.TempDir(), "missing")); err != nil || got != nil {
		t.Fatalf("loadConfigBytes missing = %q, %v", string(got), err)
	}
	if got, err := loadConfigBytes(nil, ""); err != nil || got != nil {
		t.Fatalf("loadConfigBytes empty path = %q, %v", string(got), err)
	}
	if _, err := loadConfigBytes(nil, t.TempDir()); err == nil {
		t.Fatal("loadConfigBytes returned nil error for directory path")
	}

	if got := configPath("flag.json", "configured.json", func(string) string { return "env.json" }); got != "flag.json" {
		t.Fatalf("configPath flag = %q", got)
	}
	if got := configPath("", "configured.json", func(string) string { return "env.json" }); got != "configured.json" {
		t.Fatalf("configPath configured = %q", got)
	}
	if got := configPath("", "", func(key string) string {
		if key == "CTYUN_CONFIG" {
			return "env.json"
		}
		return ""
	}); got != "env.json" {
		t.Fatalf("configPath env = %q", got)
	}
	if got := configPath("", "", func(string) string { return "" }); got == "" {
		t.Fatal("configPath did not use user home fallback")
	}
	t.Setenv("HOME", "")
	if got := configPath("", "", func(string) string { return "" }); got != "" {
		t.Fatalf("configPath without home = %q, want empty", got)
	}

	if _, err := activeProfile([]byte(`{"profiles":{"dev":{"region":"cn-dev"}}}`), "missing"); err == nil {
		t.Fatal("activeProfile returned nil error for missing profile")
	}
}

func TestRunPluginReportsOptionAndSubcommandErrors(t *testing.T) {
	profile := coreconfig.Profile{}
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		nil,
		{"install"},
		{"install", "one", "two"},
		{"install", "ecs", "--registry"},
		{"install", "ecs", "--channel"},
		{"list", "--bad"},
		{"list", "--updates"},
		{"list", "--updates", "--registry"},
		{"search", "ecs"},
		{"search", "ecs", "vpc"},
		{"search", "--registry"},
		{"search", "--channel"},
		{"remove"},
		{"lint"},
		{"update"},
		{"update", "one", "two"},
		{"update", "ecs", "--all"},
		{"update", "ecs", "--registry"},
		{"unknown"},
	} {
		if err := runPlugin(io.Discard, t.TempDir(), args, profile, getenv, nil); err == nil {
			t.Fatalf("runPlugin returned nil error for args %v", args)
		}
	}
}

func TestRunPluginRemovePropagatesRemoveErrors(t *testing.T) {
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := runPlugin(io.Discard, rootFile, []string{"remove", "ecs"}, coreconfig.Profile{}, func(string) string { return "" }, nil); err == nil {
		t.Fatal("plugin remove returned nil error when root is a file")
	}
}

func TestParsePluginOptionsRejectDuplicateSourcesAndQueries(t *testing.T) {
	if _, err := parsePluginInstallOptions([]string{"one", "two"}); err == nil {
		t.Fatal("parsePluginInstallOptions returned nil error for duplicate source")
	}
	if opts, err := parsePluginInstallOptions([]string{"--channel", "beta", "ecs"}); err != nil || opts.Channel != "beta" || opts.Source != "ecs" {
		t.Fatalf("parsePluginInstallOptions channel = %+v, %v", opts, err)
	}
	if _, err := parsePluginSearchOptions([]string{"one", "two"}); err == nil {
		t.Fatal("parsePluginSearchOptions returned nil error for duplicate query")
	}
	if opts, err := parsePluginSearchOptions([]string{"--channel", "stable", "ecs"}); err != nil || opts.Channel != "stable" || opts.Query != "ecs" {
		t.Fatalf("parsePluginSearchOptions channel = %+v, %v", opts, err)
	}
}

func TestRunPluginListHandlesMissingAndUnreadableRoots(t *testing.T) {
	if err := runPlugin(io.Discard, filepath.Join(t.TempDir(), "missing"), []string{"list"}, coreconfig.Profile{}, func(string) string { return "" }, nil); err != nil {
		t.Fatalf("plugin list missing root returned error: %v", err)
	}
	if err := runPluginWithOptions(io.Discard, filepath.Join(t.TempDir(), "missing"), []string{"list"}, coreconfig.Profile{}, func(string) string { return "" }, nil, globalOptions{Language: "en-US"}); err != nil {
		t.Fatalf("plugin list default output returned error: %v", err)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := runPlugin(io.Discard, rootFile, []string{"list"}, coreconfig.Profile{}, func(string) string { return "" }, nil); err == nil {
		t.Fatal("plugin list returned nil error for file root")
	}
}

func TestListPluginsReportsInvalidInstalledBundle(t *testing.T) {
	root := t.TempDir()
	badPlugin := filepath.Join(root, "broken")
	if err := os.Mkdir(badPlugin, 0o755); err != nil {
		t.Fatalf("mkdir broken plugin: %v", err)
	}
	if err := listPlugins(io.Discard, root, globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("listPlugins returned nil error for invalid installed plugin")
	}
}

func TestListPluginsRejectsInvalidTableControls(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts globalOptions
		want string
	}{
		{name: "unknown-filter-key", opts: globalOptions{Output: "table", Language: "en-US", Filter: "missing=value"}, want: "unknown filter key"},
		{name: "unknown-sort-key", opts: globalOptions{Output: "table", Language: "en-US", Sort: "missing"}, want: "unknown sort key"},
		{name: "invalid-filter", opts: globalOptions{Output: "table", Language: "en-US", Filter: "broken"}, want: "invalid filter"},
		{name: "invalid-sort", opts: globalOptions{Output: "table", Language: "en-US", Sort: "-"}, want: "invalid sort"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := listPlugins(io.Discard, t.TempDir(), tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("listPlugins error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestListPluginsPropagatesHelperErrors(t *testing.T) {
	t.Run("render", func(t *testing.T) {
		originalRenderOutputTable := renderOutputTable
		t.Cleanup(func() { renderOutputTable = originalRenderOutputTable })
		renderOutputTable = func([]map[string]string, []output.Column, output.TableOptions) (string, error) {
			return "", errors.New("render failed")
		}
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "table", Language: "en-US"})
		if err == nil || !strings.Contains(err.Error(), "render failed") {
			t.Fatalf("listPlugins render error = %v, want render failed", err)
		}
	})
	t.Run("json", func(t *testing.T) {
		originalRenderOutputJSON := renderOutputJSON
		t.Cleanup(func() { renderOutputJSON = originalRenderOutputJSON })
		renderOutputJSON = func(any) (string, error) {
			return "", errors.New("json failed")
		}
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "json", Language: "en-US"})
		if err == nil || !strings.Contains(err.Error(), "json failed") {
			t.Fatalf("listPlugins json error = %v, want json failed", err)
		}
	})
	t.Run("output", func(t *testing.T) {
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "yaml", Language: "en-US"})
		if err == nil || !strings.Contains(err.Error(), `unsupported output "yaml"`) {
			t.Fatalf("listPlugins output error = %v, want unsupported output", err)
		}
	})
	t.Run("stat", func(t *testing.T) {
		originalOSStat := osStat
		t.Cleanup(func() { osStat = originalOSStat })
		osStat = func(string) (os.FileInfo, error) {
			return nil, errors.New("stat failed")
		}
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "table", Language: "en-US"})
		if err == nil || !strings.Contains(err.Error(), "stat failed") {
			t.Fatalf("listPlugins stat error = %v, want stat failed", err)
		}
	})
}
