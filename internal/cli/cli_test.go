package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
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
		"ctyun - unofficial command line tool for CTyun",
		"Description:",
		"ctyun is an unofficial command line tool for CTyun.",
		"Core Commands:",
		"Product Commands:",
		"ecs instance list",
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
	for _, unwanted := range []string{"O&M", "live retrieval", "--offline", "--fixture"} {
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
	for _, want := range []string{"非官方天翼云命令行工具", "描述:", "核心命令:", "产品命令:", "全局选项:", "选择帮助和输出语言"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Global Options") || strings.Contains(got, "Description:") {
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
	for _, want := range []string{"ctyun plugin install", "--registry URL", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("core subcommand help output missing %q:\n%s", want, got)
		}
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

func TestPluginCommandDispatchUsesMetadataWithoutProductBranch(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVPCBundle(t, filepath.Join(pluginRoot, "vpc"))

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"--offline", "--lang", "en-US", "vpc", "subnet", "list", "--cols", "subnet_id,name"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := stdout.String()
	for _, want := range []string{"Subnet ID", "Name", "subnet-demo-1", "app-subnet"} {
		if !strings.Contains(got, want) {
			t.Fatalf("generic plugin output missing %q:\n%s", want, got)
		}
	}
}

func TestDangerousCommandRequiresConfirmation(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDangerBundle(t, filepath.Join(pluginRoot, "ecs"))

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"ecs", "instance", "delete", "ins-demo-1"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("dangerous command without --yes returned nil error")
	}

	err = Run(Config{
		Args:       []string{"--offline", "--yes", "ecs", "instance", "delete", "ins-demo-1"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("dangerous command with --yes returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "job-demo-1") {
		t.Fatalf("confirmed dangerous output missing fixture response:\n%s", stdout.String())
	}
}

func TestDangerousCommandLocalizesConfirmationError(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDangerBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "ecs", "instance", "delete", "ins-demo-1"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("dangerous command without --yes returned nil error")
	}
	if !strings.Contains(err.Error(), "需要确认") || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("error = %v, want localized confirmation requirement", err)
	}
}

func TestWaitFlagEvaluatesWaiterMetadata(t *testing.T) {
	pluginRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"--offline", "--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "waiter ecs.instance.running: success") {
		t.Fatalf("waiter output missing success:\n%s", stdout.String())
	}
}

func TestWaitFlagPollsUntilWaiterSucceeds(t *testing.T) {
	pluginRoot := t.TempDir()
	writePollingWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	attempts := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		status := "starting"
		if attempts == 2 {
			status = "running"
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
  "returnObj": {
    "status": "` + status + `",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1", "--cols", "instance_id"},
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if !strings.Contains(stdout.String(), "waiter ecs.instance.running: success") {
		t.Fatalf("waiter output missing success:\n%s", stdout.String())
	}
}

func TestWaitFlagReportsTimeoutAfterMaxAttempts(t *testing.T) {
	pluginRoot := t.TempDir()
	writePollingWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	attempts := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
  "returnObj": {
    "status": "starting",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--wait", "ecs.instance.running", "ecs", "instance", "show", "ins-demo-1", "--cols", "instance_id"},
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want timeout after max attempts", attempts)
	}
	if !strings.Contains(stdout.String(), "waiter ecs.instance.running: timeout") {
		t.Fatalf("waiter output missing timeout:\n%s", stdout.String())
	}
}

func TestTimeoutFlagOverridesProfileRequestTimeout(t *testing.T) {
	pluginRoot := t.TempDir()
	writePollingWaitBundle(t, filepath.Join(pluginRoot, "ecs"))

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		if !ok {
			t.Fatal("request context has no timeout deadline")
		}
		remaining := time.Until(deadline)
		if remaining < 18*time.Second || remaining > 22*time.Second {
			t.Fatalf("request timeout = %v, want about 20s", remaining)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
  "returnObj": {
    "status": "running",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--timeout", "20", "ecs", "instance", "show", "ins-demo-1", "--cols", "instance_id"},
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "timeout_seconds": 7
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestPluginCommandWithoutFixtureUsesAPIMetadataAndEnvCredentials(t *testing.T) {
	pluginRoot := t.TempDir()
	writeIMSBundleWithoutFixture(t, filepath.Join(pluginRoot, "ims"))

	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Eop-Authorization") == "" {
			t.Fatal("request was not signed")
		}
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
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[{"imageID":"img-demo-1","name":"base"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ims", "image", "list", "--cols", "image_id,name"},
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test",
      "timeout_seconds": 7
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(seenBody, `"regionID":"cn-huadong1"`) {
		t.Fatalf("request body did not include profile region: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "img-demo-1") {
		t.Fatalf("output missing API response row:\n%s", stdout.String())
	}
}

func TestPluginCommandBindsPathArgumentsIntoAPIBody(t *testing.T) {
	pluginRoot := t.TempDir()
	writeArgumentBundle(t, filepath.Join(pluginRoot, "ims"))

	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"images":[{"imageID":"img-demo-1","name":"base"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"ims", "image", "show", "img-demo-1", "--cols", "image_id,name"},
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(seenBody, `"imageID":"img-demo-1"`) {
		t.Fatalf("request body did not bind path argument: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "img-demo-1") {
		t.Fatalf("output missing response row:\n%s", stdout.String())
	}
}

func TestPluginCommandBindsMetadataFlagsIntoAPIBody(t *testing.T) {
	pluginRoot := t.TempDir()
	writeFlagBundle(t, filepath.Join(pluginRoot, "ecs"))

	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"instances":[{"instanceID":"ins-demo-1","displayName":"demo-web","status":"running"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"ecs", "instance", "list", "--name", "demo-web", "--cols", "instance_id,name"},
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(seenBody, `"displayName":"demo-web"`) {
		t.Fatalf("request body did not bind metadata flag: %s", seenBody)
	}
	if !strings.Contains(stdout.String(), "demo-web") {
		t.Fatalf("output missing response row:\n%s", stdout.String())
	}
}

func TestPluginCommandBindsQueryAndHeaderMetadata(t *testing.T) {
	pluginRoot := t.TempDir()
	writeQueryHeaderBundle(t, filepath.Join(pluginRoot, "ecs"))

	var seenQuery, seenHeader, seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seenQuery = req.URL.RawQuery
		seenHeader = req.Header.Get("x-ctyun-resource")
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"instances":[{"instanceID":"ins-demo-1","displayName":"demo-web"}]}}`)),
		}, nil
	})

	err := Run(Config{
		Args:          []string{"ecs", "instance", "list", "--page", "2", "--cols", "instance_id"},
		Stdout:        io.Discard,
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
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if seenQuery != "pageNo=2&regionID=cn-huadong1" {
		t.Fatalf("query = %q, want pageNo=2&regionID=cn-huadong1", seenQuery)
	}
	if seenHeader != "ecs" {
		t.Fatalf("header = %q, want ecs", seenHeader)
	}
	if seenBody != "" {
		t.Fatalf("body = %q, want empty body for query/header-only operation", seenBody)
	}
}

func TestPluginCommandRequiresMetadataFlags(t *testing.T) {
	pluginRoot := t.TempDir()
	writeFlagBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"ecs", "instance", "list"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
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
	if err == nil {
		t.Fatal("Run returned nil error for missing required flag")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Fatalf("error = %v, want missing --name", err)
	}
}

func TestPluginCommandValidatesAllowedParameterValues(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"--lang", "en-US", "ecs", "instance", "list", "--status", "paused", "--name", "demo-web"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("Run returned nil error for invalid status")
	}
	if !strings.Contains(err.Error(), "--status must be one of running,stopped") {
		t.Fatalf("error = %v, want allowed-values validation", err)
	}
}

func TestPluginCommandValidatesParameterPattern(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"--lang", "en-US", "ecs", "instance", "list", "--status", "running", "--name", "bad name"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("Run returned nil error for invalid name")
	}
	if !strings.Contains(err.Error(), "--name does not match") {
		t.Fatalf("error = %v, want pattern validation", err)
	}
}

func TestPluginCommandLocalizesValidationErrors(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "ecs", "instance", "list", "--status", "paused", "--name", "demo-web"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("Run returned nil error for invalid localized status")
	}
	if !strings.Contains(err.Error(), "--status 必须是以下值之一 running,stopped") {
		t.Fatalf("error = %v, want localized allowed-values validation", err)
	}
}

func TestDefaultExecutionBypassesFixtureForRetrievalCommand(t *testing.T) {
	var seen bool
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seen = true
		if req.URL.Path != "/v4/ecs/list-instances" {
			t.Fatalf("path = %q, want /v4/ecs/list-instances", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"results":[{"instanceID":"ins-live-1","displayName":"live","instanceStatus":"running"}]}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ecs", "instance", "list", "--cols", "instance_id,name"},
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
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !seen {
		t.Fatal("HTTP transport was not used")
	}
	if !strings.Contains(stdout.String(), "ins-live-1") {
		t.Fatalf("live output missing response row:\n%s", stdout.String())
	}
}

func TestOptionalMetadataFlagIsOmittedWhenUnset(t *testing.T) {
	var seenBody string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		seenBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"results":[{"instanceID":"ins-live-1","displayName":"live","instanceStatus":"running"}]}}`)),
		}, nil
	})

	err := Run(Config{
		Args:          []string{"ecs", "instance", "list", "--cols", "instance_id,name"},
		Stdout:        io.Discard,
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
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if strings.Contains(seenBody, "displayName") {
		t.Fatalf("optional displayName was sent without --name: %s", seenBody)
	}
	if !strings.Contains(seenBody, `"regionID":"cn-huadong1"`) {
		t.Fatalf("request body missing region: %s", seenBody)
	}
}

func TestDebugFlagWritesRedactedHTTPDetailsToStderr(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`ak-test sk-test Signature=abc`)),
		}, nil
	})

	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:          []string{"--debug", "ecs", "instance", "list"},
		Stdout:        &stdout,
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
	if err == nil {
		t.Fatal("Run returned nil error for HTTP 400")
	}
	got := stderr.String()
	for _, secret := range []string{"ak-test", "sk-test", "abc"} {
		if strings.Contains(got, secret) {
			t.Fatalf("debug stderr still contains %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "request POST https://ctapi.example.test/v4/ecs/list-instances") {
		t.Fatalf("debug stderr missing request line: %s", got)
	}
	if !strings.Contains(got, "response 400") {
		t.Fatalf("debug stderr missing response status: %s", got)
	}
}

func TestPluginInstallAndList(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()

	var installOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "install", bundleDir},
		Stdout:     &installOut,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install returned error: %v", err)
	}
	if !strings.Contains(installOut.String(), "installed ecs") {
		t.Fatalf("install output = %q, want installed ecs", installOut.String())
	}

	var listOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list"},
		Stdout:     &listOut,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if !strings.Contains(listOut.String(), "ecs") {
		t.Fatalf("plugin list output = %q, want ecs", listOut.String())
	}
}

func TestPluginInstallRejectsInvalidBundleBeforeCopy(t *testing.T) {
	bundleDir := filepath.Join(t.TempDir(), "vpc")
	writeVPCBundle(t, bundleDir)
	mustWrite(t, filepath.Join(bundleDir, "tables.json"), `{"tables": {}}`)
	pluginRoot := t.TempDir()

	err := Run(Config{
		Args:       []string{"plugin", "install", bundleDir},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("plugin install returned nil error for invalid bundle")
	}
	if !strings.Contains(err.Error(), "missing table") {
		t.Fatalf("error = %v, want missing table validation", err)
	}
	if _, statErr := os.Stat(filepath.Join(pluginRoot, "vpc")); !os.IsNotExist(statErr) {
		t.Fatalf("invalid plugin was copied, stat err: %v", statErr)
	}
}

func TestPluginRemove(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()

	if err := Run(Config{
		Args:       []string{"plugin", "install", bundleDir},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "remove", "ecs"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("remove returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "removed ecs") {
		t.Fatalf("remove output = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(pluginRoot, "ecs")); !os.IsNotExist(err) {
		t.Fatalf("installed plugin dir still exists or unexpected stat error: %v", err)
	}
}

func TestPluginRemoveRejectsUnsafeName(t *testing.T) {
	parent := t.TempDir()
	pluginRoot := filepath.Join(parent, "plugins")
	outside := filepath.Join(parent, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}

	err := Run(Config{
		Args:       []string{"plugin", "remove", "../outside"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("plugin remove returned nil error for unsafe name")
	}
	if !strings.Contains(err.Error(), "invalid plugin name") {
		t.Fatalf("error = %v, want invalid plugin name", err)
	}
	if _, statErr := os.Stat(outside); statErr != nil {
		t.Fatalf("outside directory was touched: %v", statErr)
	}
}

func TestPluginLintValidatesBundle(t *testing.T) {
	bundleDir := testBundleDir(t)

	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"plugin", "lint", bundleDir},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("plugin lint returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "valid ecs") {
		t.Fatalf("lint output = %q, want valid ecs", stdout.String())
	}
}

func TestPluginLintRejectsInvalidBundle(t *testing.T) {
	bundleDir := testBundleDir(t)
	mustWrite(t, filepath.Join(bundleDir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "nightly",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)

	err := Run(Config{
		Args:   []string{"plugin", "lint", bundleDir},
		Stdout: io.Discard,
	})
	if err == nil {
		t.Fatal("plugin lint returned nil error for invalid bundle")
	}
	if !strings.Contains(err.Error(), "unsupported channel") {
		t.Fatalf("error = %v, want unsupported channel validation", err)
	}
}

func TestPluginListUpdatesUsesRegistryIndex(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0.tar.gz"}
  ]
}`)

	if err := Run(Config{
		Args:       []string{"plugin", "install", bundleDir},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list", "--updates", "--registry", registryRoot},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("list --updates returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("updates output = %q", stdout.String())
	}
}

func TestPluginSearchUsesRegistryStorefront(t *testing.T) {
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "generated", "url": "ecs-generated.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-reviewed.tar.gz"},
    {"name": "vpc", "version": "0.1.0", "channel": "stable", "quality": "curated", "url": "vpc.tar.gz"}
  ]
}`)

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"plugin", "search", "--registry", registryRoot, "ec"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("plugin search returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "ecs") || !strings.Contains(got, "0.2.0") {
		t.Fatalf("search output missing reviewed ecs:\n%s", got)
	}
	if strings.Contains(got, "generated") || strings.Contains(got, "vpc") {
		t.Fatalf("search output exposed generated or unrelated plugins:\n%s", got)
	}
}

func TestPluginRegistryCanComeFromEnvOrProfile(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0"}
  ]
}`)
	writeVersionedBundle(t, filepath.Join(registryRoot, "ecs-0.2.0"), "ecs", "0.2.0")

	if err := Run(Config{
		Args:       []string{"plugin", "install", bundleDir},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install old bundle: %v", err)
	}

	var envOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list", "--updates"},
		Stdout:     &envOut,
		PluginRoot: pluginRoot,
		Env: func(key string) string {
			if key == "CTYUN_REGISTRY_URL" {
				return registryRoot
			}
			return ""
		},
	}); err != nil {
		t.Fatalf("list updates from env registry: %v", err)
	}
	if !strings.Contains(envOut.String(), "ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("env registry updates output = %q", envOut.String())
	}

	var profileOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list", "--updates"},
		Stdout:     &profileOut,
		PluginRoot: pluginRoot,
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {"registry_url": "` + registryRoot + `"}
  }
}`),
	}); err != nil {
		t.Fatalf("list updates from profile registry: %v", err)
	}
	if !strings.Contains(profileOut.String(), "ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("profile registry updates output = %q", profileOut.String())
	}

	var nestedProfileOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list", "--updates"},
		Stdout:     &nestedProfileOut,
		PluginRoot: pluginRoot,
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {"registry": {"url": "` + registryRoot + `"}}
  }
}`),
	}); err != nil {
		t.Fatalf("list updates from nested profile registry: %v", err)
	}
	if !strings.Contains(nestedProfileOut.String(), "ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("nested profile registry updates output = %q", nestedProfileOut.String())
	}
}

func TestPluginInstallByNameFromRegistry(t *testing.T) {
	pluginRoot := t.TempDir()
	registryRoot := t.TempDir()
	artifactDir := filepath.Join(registryRoot, "ecs-0.2.0")
	writeVersionedBundle(t, artifactDir, "ecs", "0.2.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0"}
  ]
}`)

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "install", "ecs", "--registry", registryRoot},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install from registry returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "installed ecs") {
		t.Fatalf("install output = %q", stdout.String())
	}

	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), "0.1.0")
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.2.0" {
		t.Fatalf("installed version = %q, want 0.2.0", installed.Manifest.Version)
	}
}

func TestPluginInstallByNameFromRegistryVerifiesChecksum(t *testing.T) {
	pluginRoot := t.TempDir()
	registryRoot := t.TempDir()
	artifactDir := filepath.Join(registryRoot, "ecs-0.2.0")
	writeVersionedBundle(t, artifactDir, "ecs", "0.2.0")
	checksum := sha256Path(t, filepath.Join(artifactDir, "plugin.json"))
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0", "sha256": "`+checksum+`"}
  ]
}`)

	if err := Run(Config{
		Args:       []string{"plugin", "install", "ecs", "--registry", registryRoot},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install with checksum returned error: %v", err)
	}

	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0", "sha256": "bad"}
  ]
}`)
	if err := Run(Config{
		Args:       []string{"plugin", "install", "ecs", "--registry", registryRoot},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	}); err == nil {
		t.Fatal("install with bad checksum returned nil error")
	}
}

func TestPluginInstallByNameFromHTTPRegistry(t *testing.T) {
	pluginRoot := t.TempDir()
	bundleDir := filepath.Join(t.TempDir(), "ecs-0.3.0")
	writeVersionedBundle(t, bundleDir, "ecs", "0.3.0")
	archivePath := filepath.Join(t.TempDir(), "ecs-0.3.0.tar.gz")
	writeTarGz(t, archivePath, bundleDir)
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])
	index := `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"` + "https://registry.example.test" + `/ecs-0.3.0.tar.gz","sha256":"` + checksum + `"}]}`
	publicKey, signature := signedRegistryIndex(t, []byte(index))

	registryURL := "https://registry.example.test"
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/index.json":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(index)),
			}, nil
		case "/index.sig":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(signature)),
			}, nil
		case "/ecs-0.3.0.tar.gz":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}
	})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--registry", registryURL},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			if key == "CTYUN_REGISTRY_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	}); err != nil {
		t.Fatalf("install from HTTP registry returned error: %v", err)
	}

	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), "0.1.0")
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.3.0" {
		t.Fatalf("installed version = %q, want 0.3.0", installed.Manifest.Version)
	}
}

func TestPluginInstallFromLocalRegistryDownloadsHTTPArtifact(t *testing.T) {
	pluginRoot := t.TempDir()
	registryRoot := t.TempDir()
	bundleDir := filepath.Join(t.TempDir(), "ecs-0.3.0")
	writeVersionedBundle(t, bundleDir, "ecs", "0.3.0")
	archivePath := filepath.Join(t.TempDir(), "ecs-0.3.0.tar.gz")
	writeTarGz(t, archivePath, bundleDir)
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.3.0", "channel": "stable", "quality": "reviewed", "url": "https://artifacts.example.test/ecs-0.3.0.tar.gz", "sha256": "`+checksum+`"}
  ]
}`)

	downloaded := false
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "artifacts.example.test" && req.URL.Path == "/ecs-0.3.0.tar.gz" {
			downloaded = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("not found")),
		}, nil
	})

	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--registry", registryRoot},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
	}); err != nil {
		t.Fatalf("install from local registry HTTP artifact returned error: %v", err)
	}
	if !downloaded {
		t.Fatal("HTTP artifact was not downloaded")
	}
	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), "0.1.0")
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.3.0" {
		t.Fatalf("installed version = %q, want 0.3.0", installed.Manifest.Version)
	}
}

func TestPluginInstallFromHTTPRegistryRequiresChecksum(t *testing.T) {
	pluginRoot := t.TempDir()
	registryURL := "https://registry.example.test"
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"` + registryURL + `/ecs-0.3.0.tar.gz"}]}`)
	publicKey, signature := signedRegistryIndex(t, index)
	downloaded := false
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/index.json":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(index)),
			}, nil
		case "/index.sig":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(signature)),
			}, nil
		case "/ecs-0.3.0.tar.gz":
			downloaded = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("archive")),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}
	})

	err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--registry", registryURL},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			if key == "CTYUN_REGISTRY_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	})
	if err == nil {
		t.Fatal("install from HTTP registry without checksum returned nil error")
	}
	if !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("error = %v, want sha256 requirement", err)
	}
	if downloaded {
		t.Fatal("artifact was downloaded before checksum requirement was enforced")
	}
}

func TestPluginHTTPRegistryRequiresSignedIndex(t *testing.T) {
	pluginRoot := t.TempDir()
	registryURL := "https://registry.example.test"
	downloaded := false
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/index.json":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0.tar.gz","sha256":"abc"}]}`)),
			}, nil
		case "/ecs-0.3.0.tar.gz":
			downloaded = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("archive")),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}
	})

	err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--registry", registryURL},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
	})
	if err == nil {
		t.Fatal("install from unsigned HTTP registry returned nil error")
	}
	if !strings.Contains(err.Error(), "public key") && !strings.Contains(err.Error(), "signature") {
		t.Fatalf("error = %v, want signed registry requirement", err)
	}
	if downloaded {
		t.Fatal("artifact was downloaded before signed index verification")
	}
}

func TestPluginUpdateAllFromRegistry(t *testing.T) {
	oldBundle := testBundleDir(t)
	pluginRoot := t.TempDir()
	registryRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(registryRoot, "ecs-0.2.0"), "ecs", "0.2.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0"}
  ]
}`)

	if err := Run(Config{
		Args:       []string{"plugin", "install", oldBundle},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install old bundle: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "update", "--all", "--registry", registryRoot},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("update --all returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("update output = %q", stdout.String())
	}
}

func TestPluginUpdateOneFromRegistry(t *testing.T) {
	oldBundle := testBundleDir(t)
	pluginRoot := t.TempDir()
	registryRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(registryRoot, "ecs-0.2.0"), "ecs", "0.2.0")
	writeVersionedBundle(t, filepath.Join(registryRoot, "vpc-0.2.0"), "vpc", "0.2.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0"},
    {"name": "vpc", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "vpc-0.2.0"}
  ]
}`)

	if err := Run(Config{
		Args:       []string{"plugin", "install", oldBundle},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("install old bundle: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "update", "ecs", "--registry", registryRoot},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("update ecs returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "updated ecs 0.1.0 -> 0.2.0") {
		t.Fatalf("update output = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(pluginRoot, "vpc")); !os.IsNotExist(err) {
		t.Fatalf("update one installed unrelated plugin or unexpected stat error: %v", err)
	}
}

func TestHelpCommandUsesPluginMetadata(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"help", "ecs", "instance", "list"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"ecs.instance.list", "ctyun ecs instance list", "https://"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
}

func TestHelpUsesSentenceCaseForEnglishDescriptions(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "ecs", "instance", "list"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"List cloud servers", "Filter by instance name", "Render output as a table or raw JSON", "Show help for the command"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing sentence-case text %q:\n%s", want, got)
		}
	}
}

func TestHelpCommandUsesPluginI18N(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "zh-CN", "help", "ecs", "instance", "list"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"弹性云主机", "列出云主机", "命令选项:", "全局选项:", "--name <value>  按云主机名称过滤", "[匹配 ^[A-Za-z0-9._-]+$]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Filter by instance name") || strings.Contains(got, "matches ^") {
		t.Fatalf("localized help output still contains English option description:\n%s", got)
	}
}

func writeArgumentBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create argument bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ims",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ims", "ctyun_product_id": 23, "docs_version": "89"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ims.image.show",
      "path": ["ims", "image", "show", "{image_id}"],
      "operation": "v4.ims.image.show",
      "table": "ims.image.show"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ims.image.show": {
      "method": "POST",
      "path": "/v4/ims/image/show",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "imageID": "$arg.image_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ims.image.show": {
      "row_path": "returnObj.images",
      "columns": [
        {"key": "image_id", "path": "imageID", "labels": {"zh-CN": "镜像ID", "en-US": "Image ID", "en-GB": "Image ID"}},
        {"key": "name", "path": "name", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
}`)
}

func writeFlagBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create flag bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "parameters": [
        {"name": "name", "flag": "name", "target": "displayName", "required": true, "description": "Filter by instance name"}
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "POST",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "displayName": "$param.name"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}},
        {"key": "name", "path": "displayName", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
}`)
}

func writeQueryHeaderBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create query/header bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "parameters": [
        {"name": "page", "flag": "page", "target": "pageNo", "required": true, "description": "Page number"}
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "GET",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "query": {"regionID": "$profile.region", "pageNo": "$param.page"},
      "headers": {"x-ctyun-resource": "ecs"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
}

func writeValidationBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create validation bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "fixture_response": "fixtures/list.json",
      "parameters": [
        {"name": "status", "flag": "status", "target": "status", "allowed_values": ["running", "stopped"], "description": "Status"},
        {"name": "name", "flag": "name", "target": "displayName", "pattern": "^[A-Za-z0-9-]+$", "description": "Instance name"}
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "POST",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "body": {"status": "$param.status", "displayName": "$param.name"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "list.json"), `{"returnObj":{"instances":[]}}`)
}

func writeIMSBundleWithoutFixture(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create ims bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ims",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ims", "ctyun_product_id": 23, "docs_version": "89"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ims.image.list",
      "path": ["ims", "image", "list"],
      "operation": "v4.ims.image.list",
      "table": "ims.image.list"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ims.image.list": {
      "method": "POST",
      "path": "/v4/ims/image/list",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ims.image.list": {
      "row_path": "returnObj.images",
      "columns": [
        {"key": "image_id", "path": "imageID", "labels": {"zh-CN": "镜像ID", "en-US": "Image ID", "en-GB": "Image ID"}},
        {"key": "name", "path": "name", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
}`)
}

func writeVersionedBundle(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "`+name+`",
  "version": "`+version+`",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "`+name+`", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands": []}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{"tables": {}}`)
}

func writeDangerBundle(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create danger fixture dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.delete",
      "path": ["ecs", "instance", "delete", "{instance_id}"],
      "operation": "v4.ecs.instance.delete",
      "table": "ecs.instance.delete",
      "fixture_response": "fixtures/delete.json",
      "dangerous": {"confirm": "yes", "message": "delete instance"}
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.delete": {
      "method": "POST",
      "path": "/v4/ecs/delete-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "instanceID": "$arg.instance_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.delete": {
      "row_path": "returnObj.jobs",
      "columns": [
        {"key": "job_id", "path": "jobID", "labels": {"zh-CN": "任务ID", "en-US": "Job ID", "en-GB": "Job ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "delete.json"), `{
  "returnObj": {
    "jobs": [{"jobID": "job-demo-1"}]
  }
}`)
}

func writeWaitBundle(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create wait fixture dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.show",
      "path": ["ecs", "instance", "show", "{instance_id}"],
      "operation": "v4.ecs.instance.show",
      "table": "ecs.instance.show",
      "fixture_response": "fixtures/show.json"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.show": {
      "method": "POST",
      "path": "/v4/ecs/show-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "instanceID": "$arg.instance_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.show": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{
  "waiters": {
    "ecs.instance.running": {"path": "returnObj.status", "success": "running", "failure": "error"}
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "show.json"), `{
  "returnObj": {
    "status": "running",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)
}

func writePollingWaitBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create polling wait bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.show",
      "path": ["ecs", "instance", "show", "{instance_id}"],
      "operation": "v4.ecs.instance.show",
      "table": "ecs.instance.show"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.show": {
      "method": "POST",
      "path": "/v4/ecs/show-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "instanceID": "$arg.instance_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.show": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{
  "waiters": {
    "ecs.instance.running": {
      "path": "returnObj.status",
      "success": "running",
      "failure": "error",
      "max_attempts": 3,
      "interval_seconds": 0
    }
  }
}`)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func writeVPCBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create vpc fixture dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "vpc",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "vpc", "ctyun_product_id": 18, "docs_version": "94"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "vpc.subnet.list",
      "path": ["vpc", "subnet", "list"],
      "operation": "v4.vpc.subnet.list",
      "table": "vpc.subnet.list",
      "fixture_response": "fixtures/subnet-list.json"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.vpc.subnet.list": {
      "method": "POST",
      "path": "/v4/vpc/list-subnet",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "vpc.subnet.list": {
      "row_path": "returnObj.subnets",
      "columns": [
        {"key": "subnet_id", "path": "subnetID", "labels": {"zh-CN": "子网ID", "en-US": "Subnet ID", "en-GB": "Subnet ID"}},
        {"key": "name", "path": "name", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "subnet-list.json"), `{
  "statusCode": 800,
  "message": "success",
  "returnObj": {
    "subnets": [
      {"subnetID": "subnet-demo-1", "name": "app-subnet"}
    ]
  }
}`)
}

func testBundleDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands": []}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{"tables": {}}`)
	return dir
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func sha256Path(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func signedRegistryIndex(t *testing.T, index []byte) (string, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate registry signing key: %v", err)
	}
	signature := ed25519.Sign(privateKey, index)
	return base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(signature)
}

func writeTarGz(t *testing.T, archivePath, srcDir string) {
	t.Helper()
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	if err := filepath.WalkDir(srcDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tarWriter.Write(data)
		return err
	}); err != nil {
		t.Fatalf("write archive: %v", err)
	}
}
