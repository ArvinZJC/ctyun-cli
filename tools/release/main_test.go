/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
)

func TestGenerateKeyPrintsBase64Keys(t *testing.T) {
	var stdout bytes.Buffer
	if err := run([]string{"--generate-key"}, func(string) string { return "" }, &stdout); err != nil {
		t.Fatalf("run generate-key returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"CTYUN_RELEASE_PRIVATE_KEY=", "CTYUN_RELEASE_PUBLIC_KEY="} {
		if !strings.Contains(got, want) {
			t.Fatalf("generate-key output missing %q:\n%s", want, got)
		}
	}
}

func TestReleaseToolWritesSignedIndexAndArchive(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	restore := patchBuildBinary(func(opts buildOptions) error {
		return os.WriteFile(opts.Output, []byte("fake ctyun "+opts.Version), 0o755)
	})
	defer restore()

	outDir := t.TempDir()
	err = run([]string{
		"--version", "0.2.0",
		"--channel", "stable",
		"--out", outDir,
		"--private-key-env", "PRIVATE_KEY",
		"--platform", "darwin/arm64",
	}, func(key string) string {
		if key == "PRIVATE_KEY" {
			return base64.StdEncoding.EncodeToString(privateKey)
		}
		return ""
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run release returned error: %v", err)
	}

	indexPath := filepath.Join(outDir, "core-index.json")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := os.ReadFile(filepath.Join(outDir, "core-index.sig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := distribution.VerifyIndexSignature(indexBytes, signature, base64.StdEncoding.EncodeToString(publicKey), "release"); err != nil {
		t.Fatalf("index signature invalid: %v", err)
	}
	index, err := corerelease.LoadIndex(indexBytes)
	if err != nil {
		t.Fatal(err)
	}
	rel, artifact, ok := index.FindLatest("stable", "darwin", "arm64")
	if !ok {
		t.Fatal("signed index did not contain darwin/arm64 artifact")
	}
	if rel.Version != "0.2.0" {
		t.Fatalf("version = %s, want 0.2.0", rel.Version)
	}
	if err := distribution.VerifySHA256(filepath.Join(outDir, artifact.URL), artifact.SHA256); err != nil {
		t.Fatalf("artifact checksum invalid: %v", err)
	}
	for _, name := range []string{"install.sh", "install.ps1", "index.json", "index.sig"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("release output missing %s: %v", name, err)
		}
	}
	registryIndexBytes, err := os.ReadFile(filepath.Join(outDir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	registrySignature, err := os.ReadFile(filepath.Join(outDir, "index.sig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := distribution.VerifyIndexSignature(registryIndexBytes, registrySignature, base64.StdEncoding.EncodeToString(publicKey), "registry"); err != nil {
		t.Fatalf("registry signature invalid: %v", err)
	}
	registryIndex, err := registry.LoadIndex(registryIndexBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, bundledPlugin := range []struct {
		name    string
		channel string
	}{
		{name: "ecs", channel: "stable"},
		{name: "region", channel: "stable"},
	} {
		artifact, ok := registryIndex.Find(bundledPlugin.name, bundledPlugin.channel)
		if !ok {
			t.Fatalf("registry index missing %s %s artifact", bundledPlugin.name, bundledPlugin.channel)
		}
		if err := distribution.VerifySHA256(filepath.Join(outDir, artifact.URL), artifact.SHA256); err != nil {
			t.Fatalf("%s artifact checksum invalid: %v", bundledPlugin.name, err)
		}
		if artifact.Product == "" || artifact.DisplayName == "" {
			t.Fatalf("%s artifact missing storefront metadata: %#v", bundledPlugin.name, artifact)
		}
	}
}

func TestReleaseToolMergesExistingIndexes(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	restore := patchBuildBinary(func(opts buildOptions) error {
		return os.WriteFile(opts.Output, []byte("fake ctyun "+opts.Version), 0o755)
	})
	defer restore()

	outDir := t.TempDir()
	oldCore := `{"schema":1,"releases":[{"version":"0.1.0-alpha.1","channel":"alpha","published_at":"2026-06-20T00:00:00Z","artifacts":[{"os":"darwin","arch":"arm64","url":"old-alpha-core.tar.gz","sha256":"` + strings.Repeat("1", 64) + `"}]},{"version":"0.1.0-beta.1","channel":"beta","published_at":"2026-06-20T00:00:00Z","artifacts":[{"os":"darwin","arch":"arm64","url":"old-beta-core.tar.gz","sha256":"` + strings.Repeat("2", 64) + `"}]}]}`
	if err := os.WriteFile(filepath.Join(outDir, "core-index.json"), []byte(oldCore), 0o644); err != nil {
		t.Fatal(err)
	}
	oldRegistry := `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"old-ecs-stable.tar.gz","sha256":"` + strings.Repeat("3", 64) + `"},{"name":"ecs","version":"0.1.0-alpha.0","channel":"alpha","quality":"reviewed","url":"old-ecs-alpha.tar.gz","sha256":"` + strings.Repeat("4", 64) + `"}]}`
	if err := os.WriteFile(filepath.Join(outDir, "index.json"), []byte(oldRegistry), 0o644); err != nil {
		t.Fatal(err)
	}

	err = run([]string{
		"--version", "0.2.0",
		"--channel", "stable",
		"--out", outDir,
		"--private-key-env", "PRIVATE_KEY",
		"--platform", "darwin/arm64",
	}, func(key string) string {
		if key == "PRIVATE_KEY" {
			return base64.StdEncoding.EncodeToString(privateKey)
		}
		return ""
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run release returned error: %v", err)
	}

	coreBytes, err := os.ReadFile(filepath.Join(outDir, "core-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	coreSignature, err := os.ReadFile(filepath.Join(outDir, "core-index.sig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := distribution.VerifyIndexSignature(coreBytes, coreSignature, base64.StdEncoding.EncodeToString(publicKey), "release"); err != nil {
		t.Fatalf("merged core index signature invalid: %v", err)
	}
	coreIndex, err := corerelease.LoadIndex(coreBytes)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := coreIndex.FindLatest("alpha", "darwin", "arm64"); !ok {
		t.Fatal("merged core index lost existing alpha release")
	}
	if rel, _, ok := coreIndex.FindLatest("beta", "darwin", "arm64"); !ok || rel.Version != "0.1.0-beta.1" {
		t.Fatalf("merged core index beta release = %s, %v", rel.Version, ok)
	}
	if rel, _, ok := coreIndex.FindLatest("stable", "darwin", "arm64"); !ok || rel.Version != "0.2.0" {
		t.Fatalf("merged core index stable release = %s, %v", rel.Version, ok)
	}
	betaReleases := 0
	for _, release := range coreIndex.Releases {
		if release.Channel == "beta" {
			betaReleases++
		}
	}
	if betaReleases != 1 {
		t.Fatalf("merged core index has %d beta releases, want 1", betaReleases)
	}

	registryBytes, err := os.ReadFile(filepath.Join(outDir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	registrySignature, err := os.ReadFile(filepath.Join(outDir, "index.sig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := distribution.VerifyIndexSignature(registryBytes, registrySignature, base64.StdEncoding.EncodeToString(publicKey), "registry"); err != nil {
		t.Fatalf("merged registry signature invalid: %v", err)
	}
	registryIndex, err := registry.LoadIndex(registryBytes)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registryIndex.Find("ecs", "stable"); !ok {
		t.Fatal("merged registry index lost existing stable plugin")
	}
	if _, ok := registryIndex.Find("ecs", "alpha"); !ok {
		t.Fatal("merged registry index lost existing alpha plugin")
	}
	ecsAlphaArtifacts := 0
	for _, artifact := range registryIndex.Plugins {
		if artifact.Name == "ecs" && artifact.Channel == "alpha" {
			ecsAlphaArtifacts++
			if artifact.Version != "0.1.0-alpha.0" {
				t.Fatalf("merged registry index kept ecs alpha %s, want 0.1.0-alpha.0", artifact.Version)
			}
		}
	}
	if ecsAlphaArtifacts != 1 {
		t.Fatalf("merged registry index has %d ecs alpha artifacts, want 1", ecsAlphaArtifacts)
	}
}

func TestInstallScriptPrefersStableByDefault(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("install.sh does not support %s", runtime.GOOS)
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skipf("install.sh does not support %s", runtime.GOARCH)
	}
	root := t.TempDir()
	stableArchive := filepath.Join(root, "stable.tar.gz")
	betaArchive := filepath.Join(root, "beta.tar.gz")
	writeTestArchive(t, stableArchive, "stable")
	writeTestArchive(t, betaArchive, "beta")
	stableSHA, _, err := checksumAndSize(stableArchive)
	if err != nil {
		t.Fatal(err)
	}
	betaSHA, _, err := checksumAndSize(betaArchive)
	if err != nil {
		t.Fatal(err)
	}
	index := `{
  "schema": 1,
  "releases": [
    {
      "version": "0.2.0-beta.1",
      "channel": "beta",
      "artifacts": [
        {
          "os": "` + runtime.GOOS + `",
          "arch": "` + runtime.GOARCH + `",
          "url": "beta.tar.gz",
          "sha256": "` + betaSHA + `"
        }
      ]
    },
    {
      "version": "0.1.0",
      "channel": "stable",
      "artifacts": [
        {
          "os": "` + runtime.GOOS + `",
          "arch": "` + runtime.GOARCH + `",
          "url": "stable.tar.gz",
          "sha256": "` + stableSHA + `"
        }
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(root, "core-index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeBin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeCurl := `#!/bin/sh
out=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) shift; out="$1" ;;
    https://*) url="$1" ;;
  esac
  shift
done
case "$url" in
  */core-index.json) cp "$CTYUN_TEST_ROOT/core-index.json" "$out" ;;
  */stable.tar.gz) cp "$CTYUN_TEST_ROOT/stable.tar.gz" "$out" ;;
  */beta.tar.gz) cp "$CTYUN_TEST_ROOT/beta.tar.gz" "$out" ;;
  *) exit 22 ;;
esac
`
	if err := os.WriteFile(filepath.Join(fakeBin, "curl"), []byte(fakeCurl), 0o755); err != nil {
		t.Fatal(err)
	}

	scriptPath, err := projectFilePath("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(t.TempDir(), "install")
	cmd := exec.Command("bash", scriptPath)
	cmd.Env = append(os.Environ(),
		"CTYUN_INSTALL_BASE_URL=https://example.test/core",
		"CTYUN_INSTALL_DIR="+installDir,
		"CTYUN_TEST_ROOT="+root,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, output)
	}
	installed, err := os.ReadFile(filepath.Join(installDir, "ctyun"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != "stable" {
		t.Fatalf("installed binary = %q, want stable\n%s", installed, output)
	}
}

func TestRunRejectsInvalidInputs(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	validKey := base64.StdEncoding.EncodeToString(privateKey)
	tests := []struct {
		name string
		args []string
		env  string
	}{
		{name: "bad flag", args: []string{"--missing"}},
		{name: "missing version", args: []string{"--out", t.TempDir(), "--platform", "darwin/arm64"}, env: validKey},
		{name: "bad version", args: []string{"--version", "v0.2", "--out", t.TempDir(), "--platform", "darwin/arm64"}, env: validKey},
		{name: "missing out", args: []string{"--version", "0.2.0", "--platform", "darwin/arm64"}, env: validKey},
		{name: "bad channel", args: []string{"--version", "0.2.0", "--channel", "nightly", "--out", t.TempDir(), "--platform", "darwin/arm64"}, env: validKey},
		{name: "missing platform", args: []string{"--version", "0.2.0", "--out", t.TempDir()}, env: validKey},
		{name: "bad platform", args: []string{"--version", "0.2.0", "--out", t.TempDir(), "--platform", "darwin"}},
		{name: "missing key", args: []string{"--version", "0.2.0", "--out", t.TempDir(), "--platform", "darwin/arm64"}},
		{name: "bad key base64", args: []string{"--version", "0.2.0", "--out", t.TempDir(), "--platform", "darwin/arm64"}, env: "bad"},
		{name: "short key", args: []string{"--version", "0.2.0", "--out", t.TempDir(), "--platform", "darwin/arm64"}, env: base64.StdEncoding.EncodeToString([]byte("short"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args, func(string) string { return tt.env }, &bytes.Buffer{})
			if err == nil {
				t.Fatal("run returned nil error")
			}
		})
	}
}

func TestValidateReleaseOptionsRequiresSemanticVersion(t *testing.T) {
	valid := releaseOptions{
		Version:   "0.2.0-beta.1",
		Channel:   "beta",
		OutDir:    t.TempDir(),
		Platforms: []string{"darwin/arm64"},
	}
	if err := validateReleaseOptions(valid); err != nil {
		t.Fatalf("validateReleaseOptions rejected SemVer prerelease: %v", err)
	}

	invalid := valid
	invalid.Version = "v0.2"
	if err := validateReleaseOptions(invalid); err == nil {
		t.Fatal("validateReleaseOptions accepted non-SemVer version")
	}
}

func TestDefaultBuildBinaryUsesTrimpath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake PATH executable uses a shell script")
	}
	binDir := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	goPath := filepath.Join(binDir, "go")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsPath + "\"\n"
	if err := os.WriteFile(goPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := defaultBuildBinary(buildOptions{
		Version:          "0.2.0",
		Channel:          "stable",
		ReleasePublicKey: "public-key",
		GOOS:             runtime.GOOS,
		GOARCH:           runtime.GOARCH,
		Output:           filepath.Join(t.TempDir(), "ctyun"),
	})
	if err != nil {
		t.Fatalf("defaultBuildBinary returned error: %v", err)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read fake go args: %v", err)
	}
	if !strings.Contains(string(args), "-trimpath\n") {
		t.Fatalf("go build args missing -trimpath:\n%s", string(args))
	}
}

func TestReleaseToolWritesWindowsArtifactAndPropagatesBuildError(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	restore := patchBuildBinary(func(opts buildOptions) error {
		if opts.GOOS != "windows" || !strings.HasSuffix(opts.Output, "ctyun.exe") {
			t.Fatalf("build options = %#v, want windows ctyun.exe", opts)
		}
		return os.WriteFile(opts.Output, []byte("fake"), 0o755)
	})
	defer restore()
	outDir := t.TempDir()
	err = run([]string{"--version", "0.2.0", "--out", outDir, "--platform", "windows/amd64"}, func(string) string {
		return base64.StdEncoding.EncodeToString(privateKey)
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run windows release returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "ctyun_0.2.0_windows_amd64.tar.gz")); err != nil {
		t.Fatal(err)
	}

	restoreFail := patchBuildBinary(func(buildOptions) error {
		return os.ErrPermission
	})
	defer restoreFail()
	if err := run([]string{"--version", "0.2.0", "--out", t.TempDir(), "--platform", "linux/amd64"}, func(string) string {
		return base64.StdEncoding.EncodeToString(privateKey)
	}, &bytes.Buffer{}); err == nil {
		t.Fatal("run returned nil error for build failure")
	}
}

func TestReleaseHelpersCoverUtilityBranches(t *testing.T) {
	if got := (*multiFlag)(nil).String(); got != "" {
		t.Fatalf("nil multiFlag String = %q", got)
	}
	var platforms multiFlag
	if err := platforms.Set("linux/amd64"); err != nil {
		t.Fatal(err)
	}
	if got := platforms.String(); got != "linux/amd64" {
		t.Fatalf("multiFlag String = %q", got)
	}
	if _, err := privateKeyFromEnv(nil, "MISSING_RELEASE_KEY"); err == nil {
		t.Fatal("privateKeyFromEnv returned nil error for missing env")
	}
	if _, _, err := splitPlatform("bad"); err == nil {
		t.Fatal("splitPlatform returned nil error for bad platform")
	}
}

func writeTestArchive(t *testing.T, archivePath, content string) {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "ctyun")
	if err := os.WriteFile(binaryPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeArchive(archivePath, binaryPath, "ctyun"); err != nil {
		t.Fatal(err)
	}
}

func patchBuildBinary(fn func(buildOptions) error) func() {
	original := buildBinary
	buildBinary = fn
	return func() {
		buildBinary = original
	}
}
