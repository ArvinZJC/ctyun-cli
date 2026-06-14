/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"version"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ctyun") {
		t.Fatalf("version output = %q, want ctyun", stdout.String())
	}
}

func TestExecuteFormatsErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"--lang", "en-US", "nope"},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error: unknown command") {
		t.Fatalf("stderr = %q, want formatted error", stderr.String())
	}
}

func TestExecuteFormatsLocalizedErrors(t *testing.T) {
	var stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"--lang", "zh-CN", "nope"},
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "错误：未知命令") {
		t.Fatalf("stderr = %q, want localized unknown-command error", stderr.String())
	}
	if strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr still contains English unknown-command text: %q", stderr.String())
	}
}

func TestExecuteLocalizesMissingCommand(t *testing.T) {
	var stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"--lang", "zh-CN"},
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "错误：缺少命令") {
		t.Fatalf("stderr = %q, want localized missing-command error", stderr.String())
	}
}

func TestExecuteLocalizesPluginCompatibilityErrors(t *testing.T) {
	bundleDir := testBundleDir(t)
	mustWrite(t, filepath.Join(bundleDir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=99.0.0 <100.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)

	var stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"--lang", "zh-CN", "plugin", "lint", bundleDir},
		Stderr: &stderr,
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	got := stderr.String()
	if !strings.Contains(got, "错误：插件 ecs 需要 ctyun >=99.0.0 <100.0.0") {
		t.Fatalf("stderr = %q, want localized compatibility error", got)
	}
	if strings.Contains(got, "requires ctyun") {
		t.Fatalf("stderr still contains English compatibility text: %q", got)
	}
}

func TestExecuteRedactsCredentialMaterialInErrors(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`ak-test sk-test Signature=abc`)),
		}, nil
	})

	var stderr bytes.Buffer
	code := Execute(Config{
		Args:          []string{"--lang", "en-US", "ecs", "instance", "list"},
		Stderr:        &stderr,
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	got := stderr.String()
	for _, secret := range []string{"ak-test", "sk-test", "abc"} {
		if strings.Contains(got, secret) {
			t.Fatalf("stderr still contains %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "Error: ctyun API returned HTTP 400") {
		t.Fatalf("stderr = %q, want formatted HTTP error", got)
	}
}

func TestCompletionCommand(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"completion", "zsh"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("completion returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "#compdef ctyun") {
		t.Fatalf("completion output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "instance") {
		t.Fatalf("completion output does not include plugin metadata words: %q", stdout.String())
	}
	for _, want := range []string{"install", "update", "lint", "--name", "-o", "-h"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("completion output does not include %q: %q", want, stdout.String())
		}
	}
}

func TestMainHelpShowsDescriptionCommandsAndGlobalOptions(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"ctyun - plugin-based CTyun CLI",
		"Description:",
		"ctyun is an unofficial, plugin-based CLI for CTyun.",
		"It prioritizes terminal-friendly cloud workflows because CTyun has no official CLI.",
		"Core Commands:",
		"Plugin Commands:",
		"ctyun plugin list",
		"ctyun help <plugin>",
		"Global Options:",
		"-o, --output <table|json>",
		"-l, --lang <locale>",
		"-t, --table <bordered|compact|plain>",
		"-C, --config <path>",
		"-P, --profile <name>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Product Commands:", "Available Plugins:", "Elastic Cloud Server", "region  Region", "ecs instance list", "O&M", "live retrieval", "--offline", "--fixture"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestMainHelpUsesI18N(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "zh-CN", "help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"非官方插件化天翼云 CLI", "目前天翼云没有官方 CLI", "描述:", "核心命令:", "插件命令:", "全局选项:", "选择帮助和输出语言"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Global Options") || strings.Contains(got, "Description:") || strings.Contains(got, "产品命令:") || strings.Contains(got, "可用插件:") {
		t.Fatalf("localized help output still contains English section text:\n%s", got)
	}
}

func TestHelpFlagShowsCommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "ecs", "instance", "list", "--help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("command help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"ecs.instance.list", "Command Options:", "--name <value>", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("command help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "--offline") || strings.Contains(got, "--fixture") {
		t.Fatalf("command help output exposes fixture options:\n%s", got)
	}
}

func TestHelpFlagShowsCoreSubcommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "plugin", "install", "--help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("core subcommand help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"plugin install", "Install a plugin from a local bundle or registry", "ctyun plugin install <bundle-or-name>", "Plugin Options:", "--registry URL", "--channel name", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("core subcommand help output missing %q:\n%s", want, got)
		}
	}
}

func TestHelpShowsPluginManagementSubcommands(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{args: []string{"help", "plugin"}, want: []string{"plugin", "Description:", "Usage:", "ctyun plugin <subcommand> [options]", "Plugin Commands:", "install", "Install a plugin from a local bundle or registry", "update", "Update one or all installed plugins", "Global Options:"}},
		{args: []string{"help", "plugin", "list"}, want: []string{"plugin list", "List installed plugins", "ctyun plugin list [--updates] [--registry URL]", "--updates"}},
		{args: []string{"help", "plugin", "lint"}, want: []string{"plugin lint", "Validate a plugin bundle", "ctyun plugin lint <bundle-path>"}},
		{args: []string{"help", "plugin", "remove"}, want: []string{"plugin remove", "Remove an installed plugin", "ctyun plugin remove <name>"}},
		{args: []string{"help", "plugin", "search"}, want: []string{"plugin search", "Search a plugin registry", "ctyun plugin search <query>", "--channel name"}},
		{args: []string{"help", "plugin", "update"}, want: []string{"plugin update", "Update one or all installed plugins", "ctyun plugin update <name|--all>", "--all"}},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			if err := Run(Config{
				Args:   append([]string{"--lang", "en-US"}, tc.args...),
				Stdout: &stdout,
			}); err != nil {
				t.Fatalf("help returned error: %v", err)
			}
			got := stdout.String()
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("plugin help output missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestHelpShowsDoctorSubcommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "doctor"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("doctor help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"doctor", "Description:", "Usage:", "ctyun doctor <subcommand>", "Core Commands:", "network", "Inspect local network and registry configuration", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor help output missing %q:\n%s", want, got)
		}
	}
}

func TestHelpShowsDoctorNetworkDetailAndRejectsUnknownSubcommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "doctor", "network"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("doctor network help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"doctor network", "Description:", "Inspect local network and registry configuration", "Usage:", "ctyun doctor network", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor network help output missing %q:\n%s", want, got)
		}
	}
	if printCoreHelp(io.Discard, []string{"doctor", "unknown"}, "en-US") {
		t.Fatal("printCoreHelp returned true for unknown doctor subcommand")
	}
	if printCoreHelp(io.Discard, []string{"doctor", "network", "extra"}, "en-US") {
		t.Fatal("printCoreHelp returned true for extra doctor help argument")
	}
}

func TestDoctorNetworkCommand(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"doctor", "network"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("doctor network returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "registry") {
		t.Fatalf("doctor output = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "registry.url") {
		t.Fatalf("doctor output = %q, want profile registry.url guidance", stdout.String())
	}
}

func TestUpgradeCommandIsExplicitlyDeferred(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"upgrade"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("upgrade returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "package manager") {
		t.Fatalf("upgrade output = %q", stdout.String())
	}
}

func TestECSInstanceListDefaultsToTable(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "--lang", "en-US", "ecs", "instance", "list", "--cols", "instance_id,name,status"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"Instance ID", "Name", "Status", "ins-demo-1", "running"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Private IP") {
		t.Fatalf("table output included unselected Private IP column:\n%s", got)
	}
}

func TestGlobalOptionShorthands(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"-O", "-l", "en-US", "-o", "table", "-t", "compact", "-c", "instance_id,status", "-f", "status=running", "-s", "-instance_id", "ecs", "instance", "list"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "ins-demo-1") || strings.Contains(got, "ins-demo-2") {
		t.Fatalf("shorthand output did not apply filter/sort/columns:\n%s", got)
	}
	if strings.Contains(got, "Name") {
		t.Fatalf("shorthand columns included unselected Name column:\n%s", got)
	}
}

func TestResolveCLILanguageUsesDarwinLocaleWhenEnvIsCLocale(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreLocale := readDarwinAppleLocale
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		readDarwinAppleLocale = restoreLocale
	})
	runtimeGOOS = "darwin"
	readDarwinAppleLocale = func() string {
		return "en_GB"
	}
	getenv := func(key string) string {
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	}

	if got := resolveCLILanguage(getenv, ""); got != "en-GB" {
		t.Fatalf("resolveCLILanguage() = %q, want en-GB", got)
	}
}

func TestResolveCLILanguagePrefersEnvAndProfile(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreLocale := readDarwinAppleLocale
	restoreWindowsLocale := readWindowsUserLocale
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		readDarwinAppleLocale = restoreLocale
		readWindowsUserLocale = restoreWindowsLocale
	})
	runtimeGOOS = "darwin"
	readDarwinAppleLocale = func() string {
		return "en_GB"
	}

	envLanguage := func(key string) string {
		if key == "CTYUN_LANGUAGE" {
			return "zh-CN"
		}
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	}
	if got := resolveCLILanguage(envLanguage, "en-GB"); got != "zh-CN" {
		t.Fatalf("env language precedence = %q, want zh-CN", got)
	}

	profileLanguage := func(key string) string {
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	}
	if got := resolveCLILanguage(profileLanguage, "en-US"); got != "en-US" {
		t.Fatalf("profile language precedence = %q, want en-US", got)
	}
}

func TestResolveCLILanguageUsesWindowsUserLocaleWhenEnvIsCLocale(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreLocale := readWindowsUserLocale
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		readWindowsUserLocale = restoreLocale
	})
	runtimeGOOS = "windows"
	readWindowsUserLocale = func() string {
		return "en-GB"
	}
	getenv := func(key string) string {
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	}

	if got := resolveCLILanguage(getenv, ""); got != "en-GB" {
		t.Fatalf("resolveCLILanguage() = %q, want en-GB", got)
	}
}

func TestResolveCLILanguageUsesWindowsUserLocaleWhenEnvIsMissing(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreLocale := readWindowsUserLocale
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		readWindowsUserLocale = restoreLocale
	})
	runtimeGOOS = "windows"
	readWindowsUserLocale = func() string {
		return "zh-CN"
	}

	if got := resolveCLILanguage(func(string) string { return "" }, ""); got != "zh-CN" {
		t.Fatalf("resolveCLILanguage() = %q, want zh-CN", got)
	}
}

func TestECSInstanceListSupportsJSONPassthrough(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "ecs", "instance", "list", "--output", "json"},
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

func TestJSONOutputWithWaiterKeepsStdoutMachineReadable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "--output", "json", "--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid JSON:\n%s\nerr=%v", stdout.String(), err)
	}
	if strings.Contains(stdout.String(), "waiter ecs.instance.running") {
		t.Fatalf("stdout contains waiter status:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "waiter ecs.instance.running: success") {
		t.Fatalf("stderr missing waiter status:\n%s", stderr.String())
	}
}

func TestECSInstanceShowSupportsWaiter(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "--lang", "en-US", "--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1", "--cols", "instance_id,status"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"Instance ID", "Status", "ins-demo-1", "running", "waiter ecs.instance.running: success"} {
		if !strings.Contains(got, want) {
			t.Fatalf("show output missing %q:\n%s", want, got)
		}
	}
}

func TestECSInstanceListSupportsFilterAndSort(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "--lang", "en-US", "ecs", "instance", "list", "--filter", "status=running", "--sort", "-instance_id", "--cols", "instance_id,status"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "ins-demo-1") {
		t.Fatalf("filtered output missing running instance:\n%s", got)
	}
	if strings.Contains(got, "ins-demo-2") {
		t.Fatalf("filtered output includes stopped instance:\n%s", got)
	}
}

func TestECSInstanceListAppliesParameterFiltersToFixtureRows(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "--lang", "en-US", "ecs", "instance", "list", "--name", "demo-web", "--cols", "instance_id,name"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "demo-web") {
		t.Fatalf("parameter-filtered output missing demo-web:\n%s", got)
	}
	if strings.Contains(got, "demo-worker") {
		t.Fatalf("parameter-filtered output includes demo-worker:\n%s", got)
	}
}

func TestPluginCommandDefaultsToLiveRequest(t *testing.T) {
	var seen bool
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seen = true
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"results":[{"instanceID":"ins-live-1","displayName":"live","instanceStatus":"running"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ecs", "instance", "list", "--cols", "instance_id,status"},
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
    "default": {"region": "cn-huadong1"}
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !seen {
		t.Fatal("HTTP transport was not used by default")
	}
	if !strings.Contains(stdout.String(), "ins-live-1") {
		t.Fatalf("live output missing response row:\n%s", stdout.String())
	}
}

func TestECSInstanceListRejectsUnknownFilterOrSortKeys(t *testing.T) {
	err := Run(Config{
		Args: []string{"--offline", "ecs", "instance", "list", "--filter", "实例ID=ins-demo-1"},
	})
	if err == nil {
		t.Fatal("Run returned nil error for translated filter key")
	}
	if !strings.Contains(err.Error(), "unknown filter key") {
		t.Fatalf("error = %v, want unknown filter key", err)
	}

	err = Run(Config{
		Args: []string{"--offline", "ecs", "instance", "list", "--sort", "displayName"},
	})
	if err == nil {
		t.Fatal("Run returned nil error for response-field sort key")
	}
	if !strings.Contains(err.Error(), "unknown sort key") {
		t.Fatalf("error = %v, want unknown sort key", err)
	}
}

func TestECSInstanceListLocalizesHeaders(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "--lang", "zh-CN", "ecs", "instance", "list", "--cols", "instance_id,status"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "实例ID") || !strings.Contains(got, "状态") {
		t.Fatalf("localized table output missing Chinese headers:\n%s", got)
	}
}

func TestShippedECSInstanceStartRequiresConfirmation(t *testing.T) {
	err := Run(Config{
		Args: []string{"--lang", "en-US", "ecs", "instance", "start", "ins-demo-1"},
	})
	if err == nil {
		t.Fatal("start without --yes returned nil error")
	}
	if !strings.Contains(err.Error(), "confirmation required") {
		t.Fatalf("error = %v, want confirmation requirement", err)
	}

	var stdout bytes.Buffer
	err = Run(Config{
		Args:   []string{"--offline", "--lang", "en-US", "--yes", "ecs", "instance", "start", "ins-demo-1"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("confirmed start returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Job ID", "job-start-demo-1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("confirmed start output missing %q:\n%s", want, got)
		}
	}
}

func TestShippedRegionListUsesOfficialPublicAPI(t *testing.T) {
	var seenHost, seenMethod, seenPath, seenQuery string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
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
	err := Run(Config{
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

func TestProfileLanguageIsUsedWhenFlagAndEnvAreUnset(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"--offline", "ecs", "instance", "list", "--cols", "instance_id,status"},
		Stdout: &stdout,
		Env: func(key string) string {
			if key == "LANG" {
				return "fr-FR"
			}
			return ""
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {"language": "en-GB"}
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "Instance ID") {
		t.Fatalf("profile language was not used:\n%s", stdout.String())
	}
}

func TestConfigWithMultipleProfilesRequiresSelection(t *testing.T) {
	err := Run(Config{
		Args: []string{"version"},
		Config: []byte(`{
  "profiles": {
    "dev": {"region": "cn-dev"},
    "prod": {"region": "cn-prod"}
  }
}`),
	})
	if err == nil {
		t.Fatal("Run returned nil error for ambiguous profiles")
	}
	if !strings.Contains(err.Error(), "active_profile") {
		t.Fatalf("error = %v, want active_profile guidance", err)
	}
}

func TestConfigFileAndProfileFlagsFeedLiveCommand(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{
  "active_profile": "dev",
  "profiles": {
    "dev": {
      "region": "cn-dev",
      "endpoint_url": "https://dev.example.test",
      "language": "zh-CN"
    },
    "prod": {
      "region": "cn-prod",
      "endpoint_url": "https://prod.example.test",
      "language": "en-GB",
      "timeout_seconds": 9
    }
  }
}`)

	var seenHost, seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seenHost = req.URL.Host
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("request context has no profile timeout deadline")
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"results":[{"instanceID":"ins-prod-1","displayName":"prod","instanceStatus":"running"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--config", configPath, "--profile", "prod", "ecs", "instance", "list", "--cols", "instance_id,status"},
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
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if seenHost != "prod.example.test" {
		t.Fatalf("request host = %q, want prod.example.test", seenHost)
	}
	if !strings.Contains(seenBody, `"regionID":"cn-prod"`) {
		t.Fatalf("request body did not use selected profile region: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "Instance ID") {
		t.Fatalf("profile language did not use en-GB table headers:\n%s", stdout.String())
	}
}

func TestConfigFileRejectsPersistedSecrets(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-prod",
      "ak": "must-not-be-here"
    }
  }
}`)

	err := Run(Config{
		Args: []string{"--config", configPath, "version"},
	})
	if err == nil {
		t.Fatal("Run returned nil error for config containing AK")
	}
	if !strings.Contains(err.Error(), "must not contain") {
		t.Fatalf("error = %v, want persisted secret rejection", err)
	}
}
