package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
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
	if err := runCompletion(io.Discard, []string{"powershell"}, t.TempDir()); err == nil {
		t.Fatal("runCompletion returned nil error for unsupported shell")
	}

	badRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(badRoot, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if words := completionWords(badRoot); len(words) == 0 {
		t.Fatal("completionWords returned no core words when plugin loading failed")
	}
}

func TestCompletionSkipsArgumentPlaceholdersInAliases(t *testing.T) {
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

	words := completionWords(root)
	for _, word := range words {
		if strings.Contains(word, "{") {
			t.Fatalf("completionWords exposed placeholder %q in %v", word, words)
		}
	}
}

func TestHelpHelpersCoverCoreAndFallbackText(t *testing.T) {
	for _, command := range []string{"completion", "doctor", "help", "plugin", "upgrade", "version"} {
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
	if err := printProductCommands(&stdout, t.TempDir(), "en-US"); err != nil {
		t.Fatalf("printProductCommands empty returned error: %v", err)
	}
	badRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(badRoot, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if err := printProductCommands(io.Discard, badRoot, "en-US"); err == nil {
		t.Fatal("printProductCommands returned nil error for invalid plugin")
	}
	if err := printMainHelp(io.Discard, badRoot, "en-US"); err == nil {
		t.Fatal("printMainHelp returned nil error for invalid plugin root")
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

func TestParseGlobalOptionsErrorsAndDeprecatedLive(t *testing.T) {
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
	opts, rest, err := parseGlobalOptions([]string{"--live", "--no-header", "--debug", "--help", "version"})
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
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := runPlugin(io.Discard, rootFile, []string{"list"}, coreconfig.Profile{}, func(string) string { return "" }, nil); err == nil {
		t.Fatal("plugin list returned nil error for file root")
	}
}

func TestRegistryArtifactHelpersAndHTTPUtilities(t *testing.T) {
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`)
	if _, err := findRegistryArtifact(registryRoot, "missing", "", nil, ""); err == nil {
		t.Fatal("findRegistryArtifact returned nil error for missing plugin")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if _, err := findRegistryArtifact(badRegistry, "ecs", "", nil, ""); err == nil {
		t.Fatal("findRegistryArtifact returned nil error for malformed registry")
	}
	if err := searchPlugins(io.Discard, "", "", "ecs", nil, ""); err == nil {
		t.Fatal("searchPlugins returned nil error without registry")
	}
	if err := searchPlugins(io.Discard, filepath.Join(t.TempDir(), "missing"), "", "ecs", nil, ""); err == nil {
		t.Fatal("searchPlugins returned nil error for missing registry")
	}
	if err := searchPlugins(io.Discard, badRegistry, "", "ecs", nil, ""); err == nil {
		t.Fatal("searchPlugins returned nil error for malformed registry")
	}

	artifact := registry.Artifact{Name: "ecs", URL: "https://registry.example.test/ecs.tar.gz"}
	if _, _, err := prepareRegistryArtifact("", artifact, nil); err == nil {
		t.Fatal("prepareRegistryArtifact returned nil error for HTTP artifact without sha256")
	}
	artifact = registry.Artifact{Name: "ecs", URL: "ecs.tar.gz", SHA256: "bad"}
	if _, _, err := prepareRegistryArtifact("https://registry.example.test", artifact, nil); err == nil {
		t.Fatal("prepareRegistryArtifact returned nil error for HTTP registry relative artifact without transport")
	}
	if _, _, err := prepareRegistryArtifact("https://registry.example.test", registry.Artifact{Name: "ecs", URL: "ecs.tar.gz"}, nil); err == nil {
		t.Fatal("prepareRegistryArtifact returned nil error for HTTP registry artifact without sha256")
	}
	if _, _, err := prepareRegistryArtifact("https://registry.example.test", artifact, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	})); err == nil {
		t.Fatal("prepareRegistryArtifact returned nil error for failed download")
	}
	if err := verifyArtifact(filepath.Join(t.TempDir(), "missing"), registry.Artifact{SHA256: "bad"}); err == nil {
		t.Fatal("verifyArtifact returned nil error for missing path")
	}

	if _, err := httpGetBytes("://bad", nil); err == nil {
		t.Fatal("httpGetBytes returned nil error for bad URL")
	}
	if _, _, err := downloadRegistryArtifact("://bad", nil); err == nil {
		t.Fatal("downloadRegistryArtifact returned nil error for bad URL")
	}
	if _, err := httpGetBytes("https://registry.example.test/index.json", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	})); err == nil {
		t.Fatal("httpGetBytes returned nil error for transport error")
	}
	if _, err := httpGetBytes("https://registry.example.test/index.json", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("missing"))}, nil
	})); err == nil {
		t.Fatal("httpGetBytes returned nil error for HTTP 404")
	}
	if got := joinRegistryURL("://bad", "index.json"); got != "://bad/index.json" {
		t.Fatalf("joinRegistryURL bad root = %q", got)
	}
}

func TestPluginUpdateHelpersCoverNoopAndErrorPaths(t *testing.T) {
	root := t.TempDir()
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[]}`)

	if err := updateAllPlugins(io.Discard, filepath.Join(t.TempDir(), "missing"), registryRoot, nil, ""); err != nil {
		t.Fatalf("updateAllPlugins missing root returned error: %v", err)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := updateAllPlugins(io.Discard, rootFile, registryRoot, nil, ""); err == nil {
		t.Fatal("updateAllPlugins returned nil error for file root")
	}
	if err := updateAllPlugins(io.Discard, root, filepath.Join(t.TempDir(), "missing-registry"), nil, ""); err == nil {
		t.Fatal("updateAllPlugins returned nil error for missing registry")
	}
	if err := updateAllPlugins(io.Discard, root, "", nil, ""); err == nil {
		t.Fatal("updateAllPlugins returned nil error without registry")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := updateAllPlugins(io.Discard, root, badRegistry, nil, ""); err == nil {
		t.Fatal("updateAllPlugins returned nil error for malformed registry")
	}

	mustWrite(t, filepath.Join(root, "note.txt"), "ignore")
	if err := updateAllPlugins(io.Discard, root, registryRoot, nil, ""); err != nil {
		t.Fatalf("updateAllPlugins non-dir entry returned error: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if err := updateAllPlugins(io.Discard, root, registryRoot, nil, ""); err == nil {
		t.Fatal("updateAllPlugins returned nil error for invalid installed bundle")
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.2.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs-0.1.0"}]}`)
	var stdout bytes.Buffer
	if err := updateAllPlugins(&stdout, root, registryRoot, nil, ""); err != nil {
		t.Fatalf("updateAllPlugins up-to-date returned error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("updateAllPlugins no-op output = %q, want empty", stdout.String())
	}

	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"missing-artifact"}]}`)
	if err := updateAllPlugins(io.Discard, root, registryRoot, nil, ""); err == nil {
		t.Fatal("updateAllPlugins returned nil error for missing artifact source")
	}
	artifactDir := filepath.Join(registryRoot, "ecs-0.3.0")
	writeVersionedBundle(t, artifactDir, "ecs", "0.3.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0","sha256":"bad"}]}`)
	if err := updateAllPlugins(io.Discard, root, registryRoot, nil, ""); err == nil {
		t.Fatal("updateAllPlugins returned nil error for bad checksum")
	}

	if err := updateOnePlugin(io.Discard, root, "", "ecs", nil, ""); err == nil {
		t.Fatal("updateOnePlugin returned nil error without registry")
	}
	if err := updateOnePlugin(io.Discard, root, registryRoot, "missing", nil, ""); err == nil {
		t.Fatal("updateOnePlugin returned nil error for missing installed plugin")
	}
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"ecs-0.2.0"}]}`)
	stdout.Reset()
	if err := updateOnePlugin(&stdout, root, registryRoot, "ecs", nil, ""); err != nil {
		t.Fatalf("updateOnePlugin up-to-date returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "up to date") {
		t.Fatalf("updateOnePlugin output = %q, want up to date", stdout.String())
	}
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"missing-artifact"}]}`)
	if err := updateOnePlugin(io.Discard, root, registryRoot, "ecs", nil, ""); err == nil {
		t.Fatal("updateOnePlugin returned nil error for missing update artifact")
	}
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0","sha256":"bad"}]}`)
	if err := updateOnePlugin(io.Discard, root, registryRoot, "ecs", nil, ""); err == nil {
		t.Fatal("updateOnePlugin returned nil error for bad checksum")
	}
}

func TestPluginUpdateHelpersInstallSuccessfulUpdates(t *testing.T) {
	root := t.TempDir()
	registryRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	writeVersionedBundle(t, filepath.Join(registryRoot, "ecs-0.2.0"), "ecs", "0.2.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"ecs-0.2.0"}]}`)

	var stdout bytes.Buffer
	if err := updateAllPlugins(&stdout, root, registryRoot, nil, ""); err != nil {
		t.Fatalf("updateAllPlugins install returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("updateAllPlugins output = %q", stdout.String())
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	stdout.Reset()
	if err := updateOnePlugin(&stdout, root, registryRoot, "ecs", nil, ""); err != nil {
		t.Fatalf("updateOnePlugin install returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("updateOnePlugin output = %q", stdout.String())
	}
}

func TestPluginUpdateHelpersCoverSignedHTTPPrepareErrors(t *testing.T) {
	root := t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"ecs-0.2.0"}]}`)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate registry key: %v", err)
	}
	signature := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, index)))
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := index
		if strings.HasSuffix(req.URL.Path, "index.sig") {
			body = signature
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(body))}, nil
	})
	publicKeyText := base64.StdEncoding.EncodeToString(publicKey)

	if err := updateAllPlugins(io.Discard, root, "https://registry.example.test/root", transport, publicKeyText); err == nil {
		t.Fatal("updateAllPlugins returned nil error for HTTP artifact without sha256")
	}
	if err := updateOnePlugin(io.Discard, root, t.TempDir(), "ecs", nil, ""); err == nil {
		t.Fatal("updateOnePlugin returned nil error for missing registry artifact")
	}
	if err := updateOnePlugin(io.Discard, root, "https://registry.example.test/root", "ecs", transport, publicKeyText); err == nil {
		t.Fatal("updateOnePlugin returned nil error for HTTP artifact without sha256")
	}
}

func TestListPluginUpdatesCoverNoopAndErrorPaths(t *testing.T) {
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[]}`)
	if err := listPluginUpdates(io.Discard, filepath.Join(t.TempDir(), "missing"), registryRoot, nil, ""); err != nil {
		t.Fatalf("listPluginUpdates missing root returned error: %v", err)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := listPluginUpdates(io.Discard, rootFile, registryRoot, nil, ""); err == nil {
		t.Fatal("listPluginUpdates returned nil error for file root")
	}
	if err := listPluginUpdates(io.Discard, t.TempDir(), filepath.Join(t.TempDir(), "missing-registry"), nil, ""); err == nil {
		t.Fatal("listPluginUpdates returned nil error for missing registry")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := listPluginUpdates(io.Discard, t.TempDir(), badRegistry, nil, ""); err == nil {
		t.Fatal("listPluginUpdates returned nil error for malformed registry")
	}

	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "note.txt"), "ignore")
	if err := listPluginUpdates(io.Discard, root, registryRoot, nil, ""); err != nil {
		t.Fatalf("listPluginUpdates non-dir entry returned error: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if err := listPluginUpdates(io.Discard, root, registryRoot, nil, ""); err == nil {
		t.Fatal("listPluginUpdates returned nil error for invalid installed bundle")
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.2.0")
	var stdout bytes.Buffer
	if err := listPluginUpdates(&stdout, root, registryRoot, nil, ""); err != nil {
		t.Fatalf("listPluginUpdates no update returned error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("listPluginUpdates no-op output = %q, want empty", stdout.String())
	}
}

func TestHTTPRegistryIndexAndDownloadHelpers(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate registry key: %v", err)
	}
	index := []byte(`{"plugins":[]}`)
	signature := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, index)))
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := index
		if strings.HasSuffix(req.URL.Path, "index.sig") {
			body = signature
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
	})
	if got, err := readRegistryIndex("https://registry.example.test/root", transport, base64.StdEncoding.EncodeToString(publicKey)); err != nil || string(got) != string(index) {
		t.Fatalf("readRegistryIndex HTTP = %q, %v", string(got), err)
	}
	if _, err := readRegistryIndex("https://registry.example.test/root", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "index.sig") {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("missing"))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(index))}, nil
	}), "key"); err == nil {
		t.Fatal("readRegistryIndex returned nil error for missing signature")
	}
	if _, err := readRegistryIndex("https://registry.example.test/root", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	}), "key"); err == nil {
		t.Fatal("readRegistryIndex returned nil error for index fetch failure")
	}
	if _, err := readRegistryIndex("https://registry.example.test/root", transport, "bad-key"); err == nil {
		t.Fatal("readRegistryIndex returned nil error for bad signature key")
	}

	path, cleanup, err := downloadRegistryArtifact("https://registry.example.test/plugin.tar.gz", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("artifact"))}, nil
	}))
	if err != nil {
		t.Fatalf("downloadRegistryArtifact returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("downloaded artifact missing: %v", err)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup did not remove artifact, stat err: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "tmp-file")
	mustWrite(t, tmpFile, "not-dir")
	t.Setenv("TMPDIR", tmpFile)
	if _, _, err := downloadRegistryArtifact("https://registry.example.test/plugin.tar.gz", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("artifact"))}, nil
	})); err == nil {
		t.Fatal("downloadRegistryArtifact returned nil error when TMPDIR is a file")
	}
	t.Setenv("TMPDIR", t.TempDir())

	sum := sha256.Sum256([]byte("artifact"))
	path, cleanup, err = prepareRegistryArtifact("https://registry.example.test/root", registry.Artifact{Name: "ecs", URL: "ecs.tar.gz", SHA256: hex.EncodeToString(sum[:])}, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("artifact"))}, nil
	}))
	if err != nil {
		t.Fatalf("prepareRegistryArtifact HTTP download returned error: %v", err)
	}
	cleanup()
	if path == "" {
		t.Fatal("prepareRegistryArtifact returned empty path")
	}
}

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
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Offline: true}, coreconfig.Profile{}, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error without fixture")
	}
	command.FixtureResponse = "missing.json"
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Offline: true}, coreconfig.Profile{}, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error for missing fixture")
	}
	mustWrite(t, filepath.Join(bundle.Dir, "bad.json"), `{`)
	command.FixtureResponse = "bad.json"
	if _, err := loadCommandResponse(bundle, command, nil, nil, globalOptions{Offline: true}, coreconfig.Profile{}, nil, nil, nil); err == nil {
		t.Fatal("loadCommandResponse returned nil error for malformed fixture")
	}

	command = plugin.Command{ID: "ecs.instance.list", Operation: "missing"}
	if _, err := executeAPICommand(bundle, command, nil, nil, coreconfig.Profile{EndpointURL: "https://ctapi.example.test"}, func(string) string { return "" }, nil, nil); err == nil {
		t.Fatal("executeAPICommand returned nil error for missing operation")
	}
	bundle.APIs = plugin.APIs{Operations: map[string]plugin.Operation{"op": {Method: http.MethodGet, Path: "/v4/demo"}}}
	command.Operation = "op"
	if _, err := executeAPICommand(plugin.Bundle{APIs: bundle.APIs}, command, nil, nil, coreconfig.Profile{}, func(string) string { return "" }, nil, nil); err == nil {
		t.Fatal("executeAPICommand returned nil error without endpoint")
	}
	if _, err := executeAPICommand(bundle, command, nil, nil, coreconfig.Profile{EndpointURL: "https://ctapi.example.test"}, func(string) string { return "" }, nil, nil); err == nil {
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
	}, transport, nil); err != nil {
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
