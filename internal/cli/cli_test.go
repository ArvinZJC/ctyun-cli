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

	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestVersionCommand renders the shipped command name and current version.
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
	want := version.Name + " " + version.Version
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("version output = %q, want %s", got, want)
	}
}

// TestVersionGlobalOptions supports both long and short top-level version options.
func TestVersionGlobalOptions(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"-v"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run(Config{
				Args:   args,
				Stdout: &stdout,
				Stderr: &stderr,
				Env: func(string) string {
					t.Fatal("version option should not read environment")
					return ""
				},
			})
			if err != nil {
				t.Fatalf("Run returned error: %v, stderr=%s", err, stderr.String())
			}
			if !strings.Contains(stdout.String(), "ctyun") {
				t.Fatalf("version output = %q, want ctyun", stdout.String())
			}
		})
	}
}

// TestVersionOptionDoesNotOverrideCommands rejects version options inside a resolved command.
func TestVersionOptionDoesNotOverrideCommands(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	for _, args := range [][]string{
		{"help", "region", "-v"},
		{"region", "list", "--version"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout bytes.Buffer
			err := Run(Config{PluginRoot: pluginRoot,
				Args:   args,
				Stdout: &stdout,
			})
			if err == nil {
				t.Fatalf("Run returned nil error and stdout %q", stdout.String())
			}
			requireDiagnosticKey(t, err, "error.unknown_option")
			if strings.Contains(stdout.String(), version.Name+" "+version.Version) {
				t.Fatalf("version output took precedence over command: %q", stdout.String())
			}
		})
	}
}

// TestExecuteFormatsErrors renders a diagnostic and returns a failing exit code.
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

// TestExecuteFormatsLocalizedErrors renders diagnostics in the selected language.
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

// TestExecuteLocalizesMissingCommand localizes the missing-command diagnostic.
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

// TestExecuteLocalizesPluginCompatibilityErrors localizes incompatible plugin requirements.
func TestExecuteLocalizesPluginCompatibilityErrors(t *testing.T) {
	bundleDir := testBundleDir(t)
	mustWrite(t, filepath.Join(bundleDir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=99.0.0 <100.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
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
	if !strings.Contains(got, "错误：插件 \"ecs\" 需要 ctyun \">=99.0.0 <100.0.0\"") {
		t.Fatalf("stderr = %q, want localized compatibility error", got)
	}
	if strings.Contains(got, "requires ctyun") {
		t.Fatalf("stderr still contains English compatibility text: %q", got)
	}
}

// TestExecuteRedactsCredentialMaterialInErrors checks that formatted service errors retain context while redacting credentials.
func TestExecuteRedactsCredentialMaterialInErrors(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`ak-test sk-test Signature=abc`)),
		}, nil
	})

	var stderr bytes.Buffer
	code := Execute(Config{PluginRoot: pluginRoot,
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
      "region": "81f7728662dd11ec810800155d307d5b",
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
	if !strings.Contains(got, "Error: CTyun API returned HTTP \"400\"") {
		t.Fatalf("stderr = %q, want formatted HTTP error", got)
	}
}

// TestCompletionCommand generates a shell completion script.
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
	for _, want := range []string{"ctyun", "__complete", "compadd"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("completion output does not include %q: %q", want, stdout.String())
		}
	}
}

// TestUpgradeCommandWithoutSourceLocalizesDevelopmentGuidance localizes the development-build upgrade guidance.
func TestUpgradeCommandWithoutSourceLocalizesDevelopmentGuidance(t *testing.T) {
	err := Run(Config{
		Args:   []string{"--lang", "zh-CN", "upgrade"},
		Stdout: io.Discard,
	})
	requireDiagnosticKey(t, err, "error.upgrade_dev_apply")
	got := formatError(err, "zh-CN")
	if !strings.Contains(got, "开发构建不可执行自升级") || !strings.Contains(got, "ctyun update --check") {
		t.Fatalf("upgrade output = %q, want localized development-build guidance", got)
	}
	if strings.Contains(got, "Self-upgrade") || strings.Contains(got, "development builds") {
		t.Fatalf("upgrade output contains untranslated English:\n%s", got)
	}
}

// TestGlobalOptionShorthands applies short output, column, filter, and sort options.
func TestGlobalOptionShorthands(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	var stdout bytes.Buffer
	err := Run(Config{PluginRoot: pluginRoot,
		Args:   []string{"-l", "en-US", "-o", "table", "-t", "compact", "-c", "instance_id,status", "-f", "status=running", "-s", "-instance_id", "ecs", "instance", "list", "--offline"},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "ins-demo-1") {
		t.Fatalf("shorthand output did not apply filter/sort/columns:\n%s", got)
	}
	if strings.Contains(got, "Name") {
		t.Fatalf("shorthand columns included unselected Name column:\n%s", got)
	}
}

// TestResolveCLILanguageUsesDarwinLocaleWhenEnvIsCLocale falls back from the C locale to the macOS user locale.
func TestResolveCLILanguageUsesDarwinLocaleWhenEnvIsCLocale(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreReaders := platformLocaleReaders
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		platformLocaleReaders = restoreReaders
	})
	runtimeGOOS = "darwin"
	platformLocaleReaders.darwin = func() string {
		return "en_GB"
	}
	getenv := func(key string) string {
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	}

	if got := i18n.ResolveLanguage(i18n.LanguageOptions{Env: getenv("CTYUN_LANGUAGE"), OSLocale: detectOSLocale(getenv)}); got != "en-GB" {
		t.Fatalf("ResolveLanguage() = %q, want en-GB", got)
	}
}

// TestResolveCLILanguagePrefersEnvAndProfile preserves language precedence over OS defaults.
func TestResolveCLILanguagePrefersEnvAndProfile(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreReaders := platformLocaleReaders
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		platformLocaleReaders = restoreReaders
	})
	runtimeGOOS = "darwin"
	platformLocaleReaders.darwin = func() string {
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
	if got := i18n.ResolveLanguage(i18n.LanguageOptions{Env: envLanguage("CTYUN_LANGUAGE"), Profile: "en-GB", OSLocale: detectOSLocale(envLanguage)}); got != "zh-CN" {
		t.Fatalf("env language precedence = %q, want zh-CN", got)
	}

	profileLanguage := func(key string) string {
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	}
	if got := i18n.ResolveLanguage(i18n.LanguageOptions{Env: profileLanguage("CTYUN_LANGUAGE"), Profile: "en-US", OSLocale: detectOSLocale(profileLanguage)}); got != "en-US" {
		t.Fatalf("profile language precedence = %q, want en-US", got)
	}
}

// TestDetectOSLocaleUsesWindowsUserLocaleWhenEnvIsCLocale uses the Windows user locale when the environment is C.
func TestDetectOSLocaleUsesWindowsUserLocaleWhenEnvIsCLocale(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreReaders := platformLocaleReaders
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		platformLocaleReaders = restoreReaders
	})
	runtimeGOOS = "windows"
	platformLocaleReaders.windows = func() string {
		return "en-GB"
	}
	getenv := func(key string) string {
		if key == "LANG" {
			return "C.UTF-8"
		}
		return ""
	}

	got := detectOSLocale(getenv)
	if got != "en-GB" {
		t.Fatalf("detectOSLocale() = %q, want en-GB", got)
	}
}

// TestDetectOSLocaleUsesWindowsUserLocaleWhenEnvIsMissing uses the Windows user locale when locale variables are absent.
func TestDetectOSLocaleUsesWindowsUserLocaleWhenEnvIsMissing(t *testing.T) {
	restoreOS := runtimeGOOS
	restoreReaders := platformLocaleReaders
	t.Cleanup(func() {
		runtimeGOOS = restoreOS
		platformLocaleReaders = restoreReaders
	})
	runtimeGOOS = "windows"
	platformLocaleReaders.windows = func() string {
		return "zh-CN"
	}

	got := detectOSLocale(func(string) string { return "" })
	if got != "zh-CN" {
		t.Fatalf("detectOSLocale() = %q, want zh-CN", got)
	}
}

// TestJSONOutputWithWaiterKeepsStdoutMachineReadable keeps waiter progress off JSON stdout.
func TestJSONOutputWithWaiterKeepsStdoutMachineReadable(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:       []string{"--output", "json", "--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1", "--offline"},
		Stdout:     &stdout,
		Stderr:     &stderr,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid JSON:\n%s\nerr=%v", stdout.String(), err)
	}
	if strings.Contains(stdout.String(), "waiter \"ecs.instance.running\"") {
		t.Fatalf("stdout contains waiter status:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "waiter \"ecs.instance.running\": success") {
		t.Fatalf("stderr missing waiter status:\n%s", stderr.String())
	}
}

// TestECSInstanceShowSupportsWaiter polls the waiter declared by synthetic command metadata.
func TestECSInstanceShowSupportsWaiter(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))
	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"--lang", "en-US", "--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1", "--offline", "--cols", "instance_id"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"Instance ID", "ins-demo-1", "waiter \"ecs.instance.running\": success"} {
		if !strings.Contains(got, want) {
			t.Fatalf("show output missing %q:\n%s", want, got)
		}
	}
}

// TestFixtureModeIsDevOnly rejects development-only fixture options in released builds.
func TestFixtureModeIsDevOnly(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	restoreVersion := patchVersion("0.5.1")
	t.Cleanup(restoreVersion)

	for _, flag := range []string{"--offline", "--fixture", "-O"} {
		t.Run(flag, func(t *testing.T) {
			err := Run(Config{Args: []string{"region", "list", flag}, Stdout: io.Discard, PluginRoot: pluginRoot})
			if err == nil {
				t.Fatalf("released build accepted %s fixture mode", flag)
			}
			requireDiagnosticKey(t, err, "error.unknown_option")
		})
	}
}

// TestPluginCommandDefaultsToLiveRequest performs an HTTP request unless fixture mode is selected.
func TestPluginCommandDefaultsToLiveRequest(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
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
	err := Run(Config{PluginRoot: pluginRoot,
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
    "default": {"region": "81f7728662dd11ec810800155d307d5b"}
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

// TestProfileLanguageIsUsedWhenFlagAndEnvAreUnset uses the selected profile language without a supported environment override.
func TestProfileLanguageIsUsedWhenFlagAndEnvAreUnset(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	var stdout bytes.Buffer
	err := Run(Config{PluginRoot: pluginRoot,
		Args:   []string{"ecs", "instance", "list", "--offline", "--cols", "instance_id,status"},
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

// TestConfigWithMultipleProfilesRequiresSelection rejects ambiguous profile selection.
func TestConfigWithMultipleProfilesRequiresSelection(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	err := Run(Config{PluginRoot: pluginRoot,
		Args: []string{"version"},
		Config: []byte(`{
  "profiles": {
    "dev": {"region": "41f64827f25f468595ffa3a5deb5d15d"},
    "prod": {"region": "100054c0416811e9a6690242ac110002"}
  }
}`),
	})
	if err == nil {
		t.Fatal("Run returned nil error for ambiguous profiles")
	}
	requireDiagnosticKey(t, err, "error.config_multiple_profiles")
}

// TestConfigFileAndProfileFlagsFeedLiveCommand uses credentials and region from the explicitly selected profile.
func TestConfigFileAndProfileFlagsFeedLiveCommand(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{
  "active_profile": "dev",
  "profiles": {
    "dev": {
      "region": "41f64827f25f468595ffa3a5deb5d15d",
      "endpoint_url": "https://dev.example.test",
      "language": "zh-CN"
    },
    "prod": {
      "region": "100054c0416811e9a6690242ac110002",
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
	err := Run(Config{PluginRoot: pluginRoot,
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
	if !strings.Contains(seenBody, `"regionID":"100054c0416811e9a6690242ac110002"`) {
		t.Fatalf("request body did not use selected profile region: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "Instance ID") {
		t.Fatalf("profile language did not use en-GB table headers:\n%s", stdout.String())
	}
}

// TestConfigFileRejectsUnsupportedPersistedSecrets rejects secret fields unsupported by persisted configuration.
func TestConfigFileRejectsUnsupportedPersistedSecrets(t *testing.T) {
	pluginRoot := writeSyntheticHelpBundles(t)
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "100054c0416811e9a6690242ac110002",
      "secret_token": "must-not-be-here"
    }
  }
}`)

	err := Run(Config{PluginRoot: pluginRoot,
		Args: []string{"--config", configPath, "version"},
	})
	if err == nil {
		t.Fatal("Run returned nil error for config containing unsupported secret material")
	}
	requireDiagnosticKey(t, err, "error.config_unsupported_secret")
}
