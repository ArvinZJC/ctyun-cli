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
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

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
	got := formatError(diagnostic.New("error.api_status", "900", `{"message":"regionID cannot be null"}`), "en-US")
	for _, want := range []string{
		"Error: ctyun API returned statusCode 900",
		"https://github.com/ArvinZJC/ctyun-cli/issues",
		"https://gitee.com/ArvinZJC/ctyun-cli/issues",
		"CTyun work order",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatError API hint missing %q:\n%s", want, got)
		}
	}
	got = localizedErrorText("ctyun API returned HTTP 403: forbidden", "zh-CN")
	for _, want := range []string{"ctyun API 返回 HTTP 403", "GitHub Issue", "天翼云工单"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localizedErrorText API hint missing %q:\n%s", want, got)
		}
	}
	got = localizedErrorText("ctyun API returned statusCode 900: bad request", "en-US")
	for _, want := range []string{"ctyun API returned statusCode 900", "CTyun work order"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localizedErrorText statusCode hint missing %q:\n%s", want, got)
		}
	}
}

func TestMessageTextUsesCatalogFallbacks(t *testing.T) {
	messageCatalog["test.catalog"] = map[string]string{"en-US": "catalog", "en-GB": "catalogue", "zh-CN": "目录"}
	t.Cleanup(func() { delete(messageCatalog, "test.catalog") })
	if got := messageText("test.catalog", "en-GB"); got != "catalogue" {
		t.Fatalf("messageText exact en-GB = %q, want catalogue", got)
	}
	if got := messageText("test.catalog", "en-AU"); got != "catalog" {
		t.Fatalf("messageText English fallback = %q, want catalog", got)
	}
	if got := messageText("test.catalog", "fr-FR"); got != "目录" {
		t.Fatalf("messageText non-English fallback = %q, want zh-CN text", got)
	}
	if got := messageText("test.missing", "zh-CN"); got != "test.missing" {
		t.Fatalf("messageText missing key = %q", got)
	}
}

func TestCatalogScopesSeparateHelpRuntimeAndCommonText(t *testing.T) {
	if _, ok := helpCatalog["warning.config_credentials"]; ok {
		t.Fatal("runtime warning lives in helpCatalog")
	}
	if got := messageText("warning.config_credentials", "zh-CN"); !strings.Contains(got, "警告") {
		t.Fatalf("warning.config_credentials message = %q, want localized warning", got)
	}
	if _, ok := messageCatalog["validation.allowed"]; ok {
		t.Fatal("help-only validation hint lives in messageCatalog")
	}
	if got := helpText("validation.allowed", "en-US"); got != "one of %s" {
		t.Fatalf("validation.allowed help text = %q", got)
	}
	if got := commonText("sentence.terminator", "en-AU"); got != "." {
		t.Fatalf("English common sentence terminator = %q", got)
	}
	if got := commonText("sentence.terminator", "zh-CN"); got != "。" {
		t.Fatalf("Chinese common sentence terminator = %q", got)
	}
}

func TestExecuteLocalizesPluginManagerErrors(t *testing.T) {
	var stderr bytes.Buffer
	code := Execute(Config{Args: []string{"--lang", "zh-CN", "plugin"}, Stderr: &stderr})
	if code != 1 {
		t.Fatalf("Execute code = %d, want 1", code)
	}
	got := stderr.String()
	if !strings.Contains(got, "错误：plugin 需要子命令") {
		t.Fatalf("stderr = %q, want localized plugin error", got)
	}
	if strings.Contains(got, "requires a subcommand") {
		t.Fatalf("stderr contains untranslated English: %q", got)
	}
}

func TestLocalizedErrorTextCoversCommandManagerMessages(t *testing.T) {
	cases := []string{
		"missing command",
		"doctor supports: network",
		"plugin requires a subcommand",
		"plugin install requires a plugin name",
		"plugin remove requires a plugin name",
		"plugin lint requires a bundle path",
		"plugin lint is only available in development builds",
		"plugin update/upgrade --bundled requires a plugin name or --all",
		"plugin update/upgrade requires a plugin name or --all",
		"hosted plugin metadata is unavailable for development builds; use --bundled",
		"plugin install accepts one plugin name",
		"plugin install accepts either --bundled or --source",
		"plugin search accepts one query",
		"plugin update accepts one plugin name",
		"plugin update accepts either --all or one plugin name",
		"plugin update accepts either --bundled or --source",
		"plugin source is empty",
		"--source requires a value",
		"--channel requires a value",
		"--bundled is only available in development builds",
		"fixture mode is only available in development builds",
		"--output requires a value",
		"unknown upgrade option \"--bad\"",
		"unknown plugin subcommand \"bad\"",
		"invalid plugin name \"../bad\"",
		"plugin ecs requires ctyun >=0.2.0, current version is 0.1.0",
		"unknown command \"bad\"",
	}
	for _, tc := range cases {
		got := localizedErrorText(tc, "zh-CN")
		if got == tc {
			t.Fatalf("localizedErrorText(%q) returned untranslated text", tc)
		}
	}
	if got := localizedErrorText("not translated", "zh-CN"); got != "not translated" {
		t.Fatalf("localizedErrorText fallback = %q", got)
	}
}

func TestLocalizedErrorTextCoversDistributionFetchMessages(t *testing.T) {
	cases := []string{
		"read registry index: Get \"https://example.com/index.json\": network",
		"read registry index signature: Get \"https://example.com/index.sig\": network",
		"registry index signature: registry index signature verification failed",
		"registry index requires a trusted public key",
		"decode registry public key: illegal base64 data at input byte 0",
		"registry public key has length 3, want 32",
		"decode registry signature: illegal base64 data at input byte 0",
		"registry index signature verification failed",
		"GET https://example.com/index.json returned 404 Not Found",
		"ecs requires sha256",
		"sha256 mismatch for /tmp/ecs.tar.gz: got abc, want def",
	}
	for _, tc := range cases {
		got := localizedErrorText(tc, "zh-CN")
		if got == tc {
			t.Fatalf("localizedErrorText(%q) returned untranslated text", tc)
		}
	}
}

func TestExecuteLocalizesHostedRegistryFetchErrors(t *testing.T) {
	var stderr bytes.Buffer
	code := Execute(Config{
		Args: []string{"--lang", "zh-CN", "plugin", "search", "ecs", "--source", "github"},
		HTTPTransport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return httpStringResponse(http.StatusNotFound, "missing"), nil
		}),
		Env:    hostedPluginEnv("bad-public-key"),
		Stderr: &stderr,
	})
	if code != 1 {
		t.Fatalf("Execute code = %d, want 1", code)
	}
	got := stderr.String()
	for _, want := range []string{"错误：", "读取 registry 索引失败", "GET", "返回 Not Found"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"read registry index", "returned"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("stderr leaked untranslated text %q:\n%s", unwanted, got)
		}
	}
}

func TestLocalizedErrorTextCoversCommonRuntimeMessages(t *testing.T) {
	cases := []string{
		"unsupported output \"yaml\"",
		"unknown filter key \"id\"; use a visible column or field label or stable key",
		"unknown sort key \"id\"; use a visible column or field label or stable key",
		"unknown waiter \"ready\"",
		"plugin root /tmp/plugins is not a directory",
		"plugin ecs not found in registry",
		"completion requires one shell: bash, zsh, fish, or powershell",
		"unsupported shell \"tcsh\"",
		"unknown config subcommand \"bad\"",
		"config path is unavailable",
		"Usage: ctyun config show [--profile name]",
		"Usage: ctyun config set <key> <value> [--profile name]",
		"Usage: ctyun config unset <key> [--profile name]",
		"Usage: ctyun config profile <list|use|set|unset|set-secret|reset>",
		"Usage: ctyun config profile use <name>",
		"Usage: ctyun config profile set <name> <key=value|key value>",
		"Usage: ctyun config profile unset <name> <key>",
		"Usage: ctyun config profile set-secret <name> <ak|sk> --from-stdin",
		"Usage: ctyun config profile reset <name>",
		"profile \"prod\" not found",
		"unsupported secret key \"token\"",
		"config profile reset confirmation required",
		"config reset confirmation required",
		"active_profile \"prod\" does not exist",
		"profile name cannot be empty",
		"profile \"prod\" timeout_seconds cannot be negative",
		"profile \"prod\" language \"fr-FR\" is not supported",
		"unsupported global config key \"bad\"",
		"unsupported profile config key \"bad\"",
		"parse timeout_seconds: invalid syntax",
		"parse warn_config_credentials: invalid syntax",
		"validate config: invalid",
		"parse response JSON: invalid character",
		"ctyun API returned HTTP 403: forbidden",
		"ctyun API request failed",
		"no ctyun release found for darwin/arm64 on channel stable",
		"unsupported release index schema 2",
		"release 0 is missing version",
		"release 0 has invalid version \"bad\"",
		"release 0 0.1.0 has unsupported channel \"dev\"",
		"release 0 0.1.0 has no artifacts",
		"release 0 0.1.0 artifact 0 is missing os",
		"release 0 0.1.0 artifact 0 is missing arch",
		"release 0 0.1.0 artifact 0 is missing url",
		"release 0 0.1.0 artifact 0 has invalid artifact url \"../bad\"",
		"release 0 0.1.0 artifact 0 has invalid sha256",
	}
	for _, tc := range cases {
		got := localizedErrorText(tc, "zh-CN")
		if got == tc {
			t.Fatalf("localizedErrorText(%q) returned untranslated text", tc)
		}
	}
}

func TestErrorCredentialsReturnsResolvedConfigCredentials(t *testing.T) {
	creds := errorCredentials(Config{
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "ak": "profile-ak",
      "sk": "profile-sk"
    }
  }
}`),
	}, func(string) string { return "" })

	if creds.AccessKey != "profile-ak" || creds.SecretKey != "profile-sk" {
		t.Fatalf("errorCredentials = %#v, want profile credentials", creds)
	}
	if !creds.UsesConfig() {
		t.Fatal("errorCredentials did not mark profile credentials as config-backed")
	}
}

func TestErrorCredentialsFallsBackToEnvWhenResolutionCannotContinue(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "CTYUN_AK":
			return "env-ak"
		case "CTYUN_SK":
			return "env-sk"
		default:
			return ""
		}
	}
	cases := []struct {
		name string
		cfg  Config
	}{
		{name: "invalid option", cfg: Config{Args: []string{"--config"}}},
		{name: "config load error", cfg: Config{Args: []string{"--config", t.TempDir()}}},
		{name: "active profile error", cfg: Config{Config: []byte(`{"profiles":{"dev":{},"prod":{}}}`)}},
		{name: "missing resolved credentials", cfg: Config{Config: []byte(`{"active_profile":"default","profiles":{"default":{}}}`)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			creds := errorCredentials(tc.cfg, getenv)
			if creds.AccessKey != "env-ak" || creds.SecretKey != "env-sk" {
				t.Fatalf("errorCredentials = %#v, want env fallback", creds)
			}
		})
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
}

func TestCompletionVariantsAndErrors(t *testing.T) {
	for _, shell := range []string{"bash", "fish"} {
		var stdout bytes.Buffer
		if err := runCompletion(&stdout, []string{shell}); err != nil {
			t.Fatalf("runCompletion %s returned error: %v", shell, err)
		}
		if !strings.Contains(stdout.String(), "ctyun") {
			t.Fatalf("completion for %s missing ctyun: %s", shell, stdout.String())
		}
	}
	if err := runCompletion(io.Discard, nil); err == nil {
		t.Fatal("runCompletion returned nil error without shell")
	}
	if err := runCompletion(io.Discard, []string{"nushell"}); err == nil {
		t.Fatal("runCompletion returned nil error for unsupported shell")
	}
	if err := runCompletion(failingWriter{}, []string{"bash"}); err == nil {
		t.Fatal("runCompletion returned nil error for writer failure")
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
	for _, command := range []string{"completion", "config", "doctor", "help", "plugin", "plugins", "upgrade", "update", "version"} {
		var stdout bytes.Buffer
		handled, err := printCoreHelp(&stdout, []string{command}, "en-US")
		if err != nil {
			t.Fatalf("printCoreHelp %s returned error: %v", command, err)
		}
		if !handled {
			t.Fatalf("printCoreHelp did not handle %s", command)
		}
		if !strings.Contains(stdout.String(), "Global Options") {
			t.Fatalf("core help for %s missing global options: %s", command, stdout.String())
		}
	}
	handled, err := printCoreHelp(io.Discard, []string{"unknown"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp unknown returned error: %v", err)
	}
	if handled {
		t.Fatal("printCoreHelp handled unknown command")
	}
	handled, err = printCoreHelp(io.Discard, []string{"plugin", "install", "extra"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp plugin extra returned error: %v", err)
	}
	if handled {
		t.Fatal("printCoreHelp handled plugin help with too many arguments")
	}
	handled, err = printCoreHelp(io.Discard, []string{"plugin", "missing"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp missing plugin returned error: %v", err)
	}
	if handled {
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
	if err := printPluginCommandHints(&stdout, "en-US"); err != nil {
		t.Fatalf("printPluginCommandHints returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "ctyun plugin list") || !strings.Contains(stdout.String(), "ctyun help <plugin>") {
		t.Fatalf("printPluginCommandHints output = %q", stdout.String())
	}
	if pathHasPrefix([]string{"ecs"}, nil) {
		t.Fatal("pathHasPrefix matched an empty prefix")
	}
	if pathHasPrefix([]string{"ecs"}, []string{"ecs"}) {
		t.Fatal("pathHasPrefix matched a complete command path")
	}

	argumentBundle := plugin.Bundle{
		I18N: map[string]map[string]string{"en-US": {
			"argument.region.show.region_id.description": "Resource pool ID",
		}},
		Tables: plugin.Tables{Tables: map[string]plugin.Table{
			"result": {Columns: []plugin.TableColumn{{Key: "ok", Labels: map[string]string{"en-US": "OK"}}}},
			"region": {Columns: []plugin.TableColumn{{Key: "region_id", Labels: map[string]string{"en-US": "Region ID"}}}},
		}},
	}
	if got := pluginCommandArgumentDescription(argumentBundle, plugin.Command{ID: "region.show", Table: "missing"}, "region_id", "en-US"); got != "Resource pool ID" {
		t.Fatalf("pluginCommandArgumentDescription i18n = %q, want Resource pool ID", got)
	}
	argumentCommand := plugin.Command{ID: "region.demand.check", Table: "result"}
	if got := pluginCommandArgumentDescription(argumentBundle, argumentCommand, "region_id", "en-US"); got != "Region ID" {
		t.Fatalf("pluginCommandArgumentDescription fallback = %q, want Region ID", got)
	}
	argumentRows := pluginCommandArgumentHelpRows(plugin.Bundle{}, plugin.Command{Path: []string{"demo", "{id}"}}, "en-US")
	if len(argumentRows) != 1 || argumentRows[0].Name != "{id}" || argumentRows[0].Description != "" {
		t.Fatalf("pluginCommandArgumentHelpRows empty fallback = %#v", argumentRows)
	}
	groupRows := pluginCommandGroupHelpRows(plugin.Bundle{}, []string{"demo"}, []plugin.Command{
		{ID: "demo", Path: []string{"demo"}},
		{ID: "demo.list", Path: []string{"demo", "list"}},
		{ID: "demo.list.extra", Path: []string{"demo", "list", "extra"}},
	}, "en-US")
	if len(groupRows) != 1 || groupRows[0].Name != "list" || groupRows[0].Description != "Show demo list subcommands" {
		t.Fatalf("pluginCommandGroupHelpRows filtered rows = %#v", groupRows)
	}
}

func TestShortHelpDescriptionsDoNotEndWithPunctuation(t *testing.T) {
	for key, translations := range helpCatalog {
		if !shortHelpDescriptionKey(key) {
			continue
		}
		for language, text := range translations {
			if strings.ContainsRune(".。!?！？", lastHelpRune(text)) {
				t.Fatalf("%s %s help description ends with punctuation: %q", key, language, text)
			}
		}
	}
}

func shortHelpDescriptionKey(key string) bool {
	if strings.HasPrefix(key, "core.") {
		return key != "core.heading"
	}
	if strings.HasPrefix(key, "config.") && strings.Contains(key, "description") {
		return true
	}
	if strings.HasPrefix(key, "doctor.") && strings.Contains(key, "description") {
		return true
	}
	if strings.HasPrefix(key, "plugin.") && (strings.Contains(key, "description") || strings.HasPrefix(key, "plugin.option.")) {
		return true
	}
	return strings.HasPrefix(key, "option.")
}

func lastHelpRune(text string) rune {
	trimmed := strings.TrimSpace(text)
	var last rune
	for _, char := range trimmed {
		last = char
	}
	return last
}

func TestRunHelpErrorsForInvalidOrUnknownPluginCommands(t *testing.T) {
	if err := runHelp(failingWriter{}, nil, t.TempDir(), "en-US"); err == nil {
		t.Fatal("runHelp returned nil error for main help writer failure")
	}
	if err := runHelp(failingWriter{}, []string{"completion"}, t.TempDir(), "en-US"); err == nil {
		t.Fatal("runHelp returned nil error for core help writer failure")
	}
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
	if got := parameterValidationHint(parameter, "en-US"); strings.Contains(got, "one of running,stopped") || !strings.Contains(got, "matches") {
		t.Fatalf("English validation hint = %q", got)
	}
	if got := parameterValidationHint(parameter, "zh-CN"); strings.Contains(got, "可选值") || !strings.Contains(got, "匹配") {
		t.Fatalf("Chinese validation hint = %q", got)
	}
	if got := parameterValidationHint(plugin.Parameter{}, "en-US"); got != "" {
		t.Fatalf("empty validation hint = %q", got)
	}
	if err := runDoctor(io.Discard, nil, "en-US"); err == nil {
		t.Fatal("runDoctor returned nil error for missing subcommand")
	}
	if err := runDoctor(failingWriter{}, []string{"network"}, "en-US"); err == nil {
		t.Fatal("runDoctor returned nil error for writer failure")
	}
}

func TestWriteSelectorHelpRowsWithoutDescriptions(t *testing.T) {
	var stdout bytes.Buffer
	writer := newOutputWriter(&stdout)
	writeSelectorHelpRows(writer, []helpRow{{Name: "First"}, {Name: "Second"}})
	if err := writer.Err(); err != nil {
		t.Fatalf("writeSelectorHelpRows returned error: %v", err)
	}
	if got := stdout.String(); got != "  First\n  Second\n" {
		t.Fatalf("selector rows = %q", got)
	}
}

func TestMissingCommandReturnsUsageWriterError(t *testing.T) {
	err := Run(Config{Stderr: failingWriter{}})
	if err == nil {
		t.Fatal("Run returned nil error for missing command writer failure")
	}
	if err.Error() != "write failed" {
		t.Fatalf("Run error = %q, want write failed", err.Error())
	}
}

func TestOutputWriterRecordsFirstError(t *testing.T) {
	writer := newOutputWriter(failingWriter{})
	writer.Line("first")
	writer.Format("%s", "second")
	writer.String("third")
	writer.Lines("fourth")
	if writer.Err() == nil {
		t.Fatal("outputWriter returned nil error for writer failure")
	}

	var stdout bytes.Buffer
	writer = newOutputWriter(&stdout)
	writer.Line("first")
	writer.Format("%s", "second")
	writer.String(" third")
	writer.Lines("fourth")
	if err := writer.Err(); err != nil {
		t.Fatalf("outputWriter returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "first") || !strings.Contains(got, "second thirdfourth") {
		t.Fatalf("outputWriter output = %q", got)
	}
}

func TestPrintGlobalOptionsReturnsWriterError(t *testing.T) {
	if err := printGlobalOptions(failingWriter{}, "en-US"); err == nil {
		t.Fatal("printGlobalOptions returned nil error for writer failure")
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

	if _, err := activeProfile([]byte(`{"profiles":{"dev":{"region":"41f64827f25f468595ffa3a5deb5d15d"}}}`), "missing"); err == nil {
		t.Fatal("activeProfile returned nil error for missing profile")
	}
}

func TestRunPluginReportsOptionAndSubcommandErrors(t *testing.T) {
	profile := coreconfig.Profile{}
	getenv := func(string) string { return "" }
	for _, args := range [][]string{
		nil,
		{"install"},
		{"install", "ecs"},
		{"install", "--all", "ecs"},
		{"install", "ecs", "--source"},
		{"install", "ecs", "--channel"},
		{"list", "--bad"},
		{"list", "--available"},
		{"list", "--updates"},
		{"list", "--updates", "--source"},
		{"list", "--updates", "--channel"},
		{"search", "ecs"},
		{"search", "ecs", "vpc"},
		{"search", "--source"},
		{"search", "--channel"},
		{"remove"},
		{"lint"},
		{"update"},
		{"update", "--bundled"},
		{"update", "--source", "github"},
		{"update", "one", "two"},
		{"update", "ecs", "--all"},
		{"update", "ecs", "--source"},
		{"update", "ecs", "--channel"},
		{"unknown"},
	} {
		if err := runPluginWithOptions(io.Discard, io.Discard, strings.NewReader(""), t.TempDir(), args, profile, getenv, nil, globalOptions{Output: "table", Language: "en-US"}); err == nil {
			t.Fatalf("runPlugin returned nil error for args %v", args)
		}
	}
}

func TestRunPluginBundledInstallPropagatesInstallErrors(t *testing.T) {
	t.Cleanup(patchVersion("0.3.1-dev"))
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := runPluginWithOptions(io.Discard, io.Discard, strings.NewReader(""), rootFile, []string{"install", "ecs", "--bundled"}, coreconfig.Profile{}, func(string) string { return "" }, nil, globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("plugin install --bundled returned nil error when plugin root is a file")
	}
}

func TestRunPluginRemovePropagatesRemoveErrors(t *testing.T) {
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := runPluginWithOptions(io.Discard, io.Discard, strings.NewReader(""), rootFile, []string{"remove", "ecs"}, coreconfig.Profile{}, func(string) string { return "" }, nil, globalOptions{Output: "table", Language: "en-US", Yes: true}); err == nil {
		t.Fatal("plugin remove returned nil error when root is a file")
	}
}

func TestParsePluginOptionsRejectDuplicateSourcesAndQueries(t *testing.T) {
	if _, err := parsePluginInstallOptions([]string{"--all", "ecs"}); err == nil {
		t.Fatal("parsePluginInstallOptions returned nil error for all/name conflict")
	}
	if opts, err := parsePluginInstallOptions([]string{"--channel", "beta", "ecs"}); err != nil || opts.Channel != "beta" || len(opts.Names) != 1 || opts.Names[0] != "ecs" {
		t.Fatalf("parsePluginInstallOptions channel = %+v, %v", opts, err)
	}
	if _, err := parsePluginInstallOptions([]string{"--channel", "all", "ecs"}); err == nil {
		t.Fatal("parsePluginInstallOptions returned nil error for all channel")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_channel")
	}
	if _, err := parsePluginInstallOptions([]string{"--source", "bad", "ecs"}); err == nil {
		t.Fatal("parsePluginInstallOptions returned nil error for bad source")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_source")
	}
	if _, err := parsePluginInstallOptions([]string{"--bundled", "--source", "auto", "ecs"}); err == nil {
		t.Fatal("parsePluginInstallOptions returned nil error for bundled source conflict")
	}
	if _, err := parsePluginSearchOptions([]string{"one", "two"}); err == nil {
		t.Fatal("parsePluginSearchOptions returned nil error for duplicate query")
	}
	if opts, err := parsePluginSearchOptions([]string{"--channel", "stable", "ecs"}); err != nil || opts.Channel != "stable" || opts.Query != "ecs" {
		t.Fatalf("parsePluginSearchOptions channel = %+v, %v", opts, err)
	}
	if opts, err := parsePluginSearchOptions([]string{"--channel", "all", "ecs"}); err != nil || opts.Channel != "all" || opts.Query != "ecs" {
		t.Fatalf("parsePluginSearchOptions all channel = %+v, %v", opts, err)
	}
	if _, err := parsePluginSearchOptions([]string{"--channel", "nightly", "ecs"}); err == nil {
		t.Fatal("parsePluginSearchOptions returned nil error for bad channel")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_plugin_discovery_channel")
	}
	if _, err := parsePluginSearchOptions([]string{"--source", "bad", "ecs"}); err == nil {
		t.Fatal("parsePluginSearchOptions returned nil error for bad source")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_source")
	}
	if opts, err := parsePluginListOptions([]string{"--updates", "--channel", "alpha"}); err != nil || !opts.Updates || opts.Channel != "alpha" {
		t.Fatalf("parsePluginListOptions channel = %+v, %v", opts, err)
	}
	if opts, err := parsePluginListOptions([]string{"--available", "--channel", "all"}); err != nil || !opts.Available || opts.Channel != "all" {
		t.Fatalf("parsePluginListOptions available all channel = %+v, %v", opts, err)
	}
	if _, err := parsePluginListOptions([]string{"--updates", "--channel", "all"}); err == nil {
		t.Fatal("parsePluginListOptions returned nil error for updates all channel")
	} else {
		requireDiagnosticKey(t, err, "error.plugin_list_all_channel_available")
	}
	if _, err := parsePluginListOptions([]string{"--channel", "all"}); err == nil {
		t.Fatal("parsePluginListOptions returned nil error for installed all channel")
	} else {
		requireDiagnosticKey(t, err, "error.plugin_list_all_channel_available")
	}
	if _, err := parsePluginListOptions([]string{"--available", "--channel", "nightly"}); err == nil {
		t.Fatal("parsePluginListOptions returned nil error for bad channel")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_plugin_discovery_channel")
	}
	if _, err := parsePluginListOptions([]string{"--available", "--source", "bad"}); err == nil {
		t.Fatal("parsePluginListOptions returned nil error for bad source")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_source")
	}
	if _, err := parsePluginListOptions([]string{"--available", "--updates"}); err == nil {
		t.Fatal("parsePluginListOptions returned nil error for available/update conflict")
	}
	if _, err := parsePluginListOptions([]string{"--channel"}); err == nil {
		t.Fatal("parsePluginListOptions returned nil error for missing channel value")
	}
	if opts, err := parsePluginUpdateOptions([]string{"--channel", "beta", "ecs"}); err != nil || opts.Channel != "beta" || opts.Name != "ecs" {
		t.Fatalf("parsePluginUpdateOptions channel = %+v, %v", opts, err)
	}
	if _, err := parsePluginUpdateOptions([]string{"--channel", "all", "ecs"}); err == nil {
		t.Fatal("parsePluginUpdateOptions returned nil error for all channel")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_channel")
	}
	if _, err := parsePluginUpdateOptions([]string{"--channel"}); err == nil {
		t.Fatal("parsePluginUpdateOptions returned nil error for missing channel value")
	}
	if _, err := parsePluginUpdateOptions([]string{"--bundled", "--source", "auto", "ecs"}); err == nil {
		t.Fatal("parsePluginUpdateOptions returned nil error for bundled source conflict")
	}
}

func TestRunPluginListHandlesMissingAndUnreadableRoots(t *testing.T) {
	if err := runPluginWithOptions(io.Discard, io.Discard, strings.NewReader(""), filepath.Join(t.TempDir(), "missing"), []string{"list"}, coreconfig.Profile{}, func(string) string { return "" }, nil, globalOptions{Output: "table", Language: "en-US"}); err != nil {
		t.Fatalf("plugin list missing root returned error: %v", err)
	}
	if err := runPluginWithOptions(io.Discard, io.Discard, strings.NewReader(""), filepath.Join(t.TempDir(), "missing"), []string{"list"}, coreconfig.Profile{}, func(string) string { return "" }, nil, globalOptions{Language: "en-US"}); err != nil {
		t.Fatalf("plugin list default output returned error: %v", err)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := runPluginWithOptions(io.Discard, io.Discard, strings.NewReader(""), rootFile, []string{"list"}, coreconfig.Profile{}, func(string) string { return "" }, nil, globalOptions{Output: "table", Language: "en-US"}); err == nil {
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
		{name: "unknown-filter-key", opts: globalOptions{Output: "table", Language: "en-US", Filter: "missing=value"}, want: "error.unknown_filter_key"},
		{name: "unknown-sort-key", opts: globalOptions{Output: "table", Language: "en-US", Sort: "missing"}, want: "error.unknown_sort_key"},
		{name: "invalid-filter", opts: globalOptions{Output: "table", Language: "en-US", Filter: "broken"}, want: "error.invalid_filter_syntax"},
		{name: "invalid-sort", opts: globalOptions{Output: "table", Language: "en-US", Sort: "-"}, want: "error.invalid_sort_missing_key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := listPlugins(io.Discard, t.TempDir(), tc.opts)
			if err == nil {
				t.Fatal("listPlugins returned nil error")
			}
			requireDiagnosticKey(t, err, tc.want)
		})
	}
}

func requireDiagnosticKey(t *testing.T, err error, want string) {
	t.Helper()
	got, ok := err.(interface{ MessageKey() string })
	if !ok {
		t.Fatalf("error %T does not expose a diagnostic key: %v", err, err)
	}
	if got.MessageKey() != want {
		t.Fatalf("diagnostic key = %q, want %q", got.MessageKey(), want)
	}
}
