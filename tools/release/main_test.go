/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// TestGenerateKeyPrintsBase64Keys checks the environment assignments emitted for a new signing key.
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

// TestReleaseToolWritesSignedIndexAndArchive checks signed artifacts and additive badge maps.
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
		"--version", "0.5.1",
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
	assertPublishedVersion(t, indexBytes, []string{"stable"}, "0.5.1")
	index, err := corerelease.LoadIndex(indexBytes)
	if err != nil {
		t.Fatal(err)
	}
	rel, artifact, ok := index.FindLatest("stable", "darwin", "arm64")
	if !ok {
		t.Fatal("signed index did not contain darwin/arm64 artifact")
	}
	if rel.Version != "0.5.1" {
		t.Fatalf("version = %s, want 0.5.1", rel.Version)
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
	var storefront struct {
		Plugins []struct {
			Name         string
			DisplayName  string            `json:"display_name"`
			DisplayNames map[string]string `json:"display_names"`
		}
	}
	if err := json.Unmarshal(registryIndexBytes, &storefront); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range storefront.Plugins {
		for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
			path, err := projectFilePath(filepath.Join("plugins", artifact.Name, "i18n", language+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var catalog map[string]string
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &catalog); err != nil {
				t.Fatal(err)
			}
			if want := catalog["name"]; want == "" || artifact.DisplayNames[language] != want || (language == "en-US" && artifact.DisplayName != want) {
				t.Fatalf("%s %s storefront names = %q, %v; want %q", artifact.Name, language, artifact.DisplayName, artifact.DisplayNames, want)
			}
		}
	}
	registryIndex, err := registry.LoadIndex(registryIndexBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, bundledPlugin := range []struct {
		name    string
		channel string
	}{
		{name: "ecs", channel: "beta"},
		{name: "region", channel: "stable"},
	} {
		artifact, ok := registryIndex.Find(bundledPlugin.name, bundledPlugin.channel)
		if !ok {
			t.Fatalf("registry index missing %s %s artifact", bundledPlugin.name, bundledPlugin.channel)
		}
		if err := distribution.VerifySHA256(filepath.Join(outDir, artifact.URL), artifact.SHA256); err != nil {
			t.Fatalf("%s artifact checksum invalid: %v", bundledPlugin.name, err)
		}
		assertPublishedVersion(t, registryIndexBytes, []string{artifact.Name, artifact.Channel}, artifact.Version)
		if artifact.Product == "" || artifact.DisplayName == "" {
			t.Fatalf("%s artifact missing storefront metadata: %#v", bundledPlugin.name, artifact)
		}
	}
}

// TestReleaseToolMergesExistingIndexes checks retained channels and updated artifact pointers.
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
		"--version", "0.5.1",
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
	if rel, _, ok := coreIndex.FindLatest("stable", "darwin", "arm64"); !ok || rel.Version != "0.5.1" {
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

// TestBashInstallerSelection exercises JSON decoding and platform/channel selection against a local release source.
func TestBashInstallerSelection(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" || runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skipf("install.sh does not support %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	// Expose only the bootstrap tools. Extra JSON parsers on a developer machine
	// must not mask an undeclared installer dependency.
	bin := t.TempDir()
	for _, name := range strings.Fields("bash uname tr mktemp rm awk mkdir tar gzip cp chmod mv sed curl wget sha256sum shasum") {
		if path, err := exec.LookPath(name); err == nil {
			if err := os.Symlink(path, filepath.Join(bin, name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("PATH", bin)
	root := t.TempDir()
	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer server.Close()

	artifact := func(channel, arch string) string {
		path := filepath.Join(root, channel+".tar.gz")
		writeTestArchive(t, path, channel)
		sum, _, err := checksumAndSize(path)
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf(`{"sha256":%q,"url":%q,"arch":%q,"os":%q}`, sum, channel+".tar.gz", arch, runtime.GOOS)
	}
	// Object keys may appear in any order; release and artifact arrays retain their order.
	const releaseIndex = `{"schema":1,"releases":[{"artifacts":[%s],"version":"0.2.0-beta.1","channel":"beta"},{"artifacts":[%s],"version":"0.1.0","channel":"stable"}]}`
	beta := artifact("beta", runtime.GOARCH)
	compact := fmt.Sprintf(releaseIndex, beta, artifact("stable", runtime.GOARCH))
	otherPlatform := fmt.Sprintf(releaseIndex, beta, artifact("other", "other"))
	escaped := strings.NewReplacer(
		`"releases"`, `"rele\u0061ses"`,
		`"stable.tar.gz"`, `"\u0073table.tar.gz?tag=a\u0026b=c"`,
		`"schema":1`, `"metadata":{"releases":[],"note":"\uD83D\uDE80 {,} \"url\"","values":[true,null,-1.2e+3]},"schema":1`,
	).Replace(compact)
	unicodeURL := fmt.Sprintf(releaseIndex, beta, artifact("镜像🚀", runtime.GOARCH))
	unicodeURL = strings.Replace(unicodeURL, "镜像🚀.tar.gz", `\u955c\u50cf\ud83d\ude80.tar.gz`, 1)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(compact), "", "  "); err != nil {
		t.Fatal(err)
	}
	script, err := projectFilePath("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, index, channel, want, diagnostic string
	}{
		{name: "pretty", index: pretty.String(), want: "stable"},
		{name: "compact", index: compact, want: "stable"},
		{name: "escaped and nested", index: escaped, want: "stable"},
		{name: "Unicode URL", index: unicodeURL, want: "镜像🚀"},
		{name: "pinned", index: compact, channel: "beta", want: "beta"},
		{name: "fallback", index: otherPlatform, want: "beta"},
		{name: "unavailable", index: compact, channel: "alpha", diagnostic: "no ctyun release found"},
		{name: "invalid JSON", index: compact + "!", diagnostic: "invalid core-index.json"},
		{name: "truncated", index: compact[:len(compact)-2], diagnostic: "invalid core-index.json"},
		{name: "trailing comma", index: strings.Replace(compact, "]}", ",]}", 1), diagnostic: "invalid core-index.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(root, "core-index.json"), []byte(tc.index), 0o600); err != nil {
				t.Fatal(err)
			}
			// Some checksum tools escape printed filenames containing a backslash.
			tmp := filepath.Join(t.TempDir(), `temporary\path`)
			if err := os.Mkdir(tmp, 0o700); err != nil {
				t.Fatal(err)
			}
			dest := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			t.Setenv("CTYUN_INSTALL_BASE_URL", server.URL)
			t.Setenv("CTYUN_INSTALL_DIR", dest)
			t.Setenv("CTYUN_INSTALL_CHANNEL", tc.channel)
			t.Setenv("CTYUN_INSTALL_SOURCE", "auto")
			out, err := exec.CommandContext(t.Context(), "bash", script).CombinedOutput()
			if tc.diagnostic != "" {
				if err == nil || !strings.Contains(string(out), tc.diagnostic) {
					t.Fatalf("expected %q, got %v\n%s", tc.diagnostic, err, out)
				}
				if _, err := os.Stat(filepath.Join(dest, "ctyun")); !os.IsNotExist(err) {
					t.Fatalf("unexpected install target after failure: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("install.sh: %v\n%s", err, out)
			}
			got, err := os.ReadFile(filepath.Join(dest, "ctyun"))
			if err != nil || string(got) != tc.want {
				t.Fatalf("installed %q, want %q: %v\n%s", got, tc.want, err, out)
			}
		})
	}
}

// TestRunRejectsInvalidInputs checks required arguments and signing key validation.
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

// TestReleaseRejectsInvalidInputsBeforeWriting checks that a stray operand cannot
// hide later build flags and a damaged signing key cannot produce release files.
func TestReleaseRejectsInvalidInputsBeforeWriting(t *testing.T) {
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	damaged := bytes.Clone(key)
	damaged[ed25519.SeedSize] ^= 1
	for _, tc := range []struct {
		name  string
		key   []byte
		extra []string
	}{
		{"stray operand", key, []string{"extra", "--channel", "beta"}},
		{"inconsistent signing key", damaged, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "release")
			args := []string{"--version", "0.2.0", "--out", out, "--platform", "darwin/arm64"}
			err := run(append(args, tc.extra...), func(string) string {
				return base64.StdEncoding.EncodeToString(tc.key)
			}, &bytes.Buffer{})
			if err == nil {
				t.Fatal("packaging reported success")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("invalid input touched output directory, stat error = %v", err)
			}
		})
	}
}

// TestValidateReleaseOptionsRequiresSemanticVersion checks release version syntax.
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
	for _, tt := range []struct {
		version, channel string
	}{
		{"0.2.0-beta.1", "stable"},
		{"0.2.0+release-note", "beta"},
	} {
		opts := valid
		opts.Version, opts.Channel = tt.version, tt.channel
		if err := validateReleaseOptions(opts); err == nil {
			t.Errorf("accepted incompatible version %s and channel %s", opts.Version, opts.Channel)
		}
	}

	invalid := valid
	invalid.Version = "v0.2"
	if err := validateReleaseOptions(invalid); err == nil {
		t.Fatal("validateReleaseOptions accepted non-SemVer version")
	}
}

// TestDefaultBuildBinaryUsesTrimpath checks reproducible build arguments.
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

// TestBuildFailureIncludesCompilerDiagnostic checks the real Go tool's platform failure on every host.
func TestBuildFailureIncludesCompilerDiagnostic(t *testing.T) {
	err := defaultBuildBinary(buildOptions{GOOS: "ctyun-test-os", GOARCH: "amd64", Output: filepath.Join(t.TempDir(), "ctyun")})
	if err == nil || !strings.Contains(err.Error(), "build ctyun for ctyun-test-os/amd64") || !strings.Contains(err.Error(), "unsupported GOOS/GOARCH") {
		t.Fatalf("build failure lost platform or compiler diagnostic: %v", err)
	}
}

// TestReleaseToolWritesWindowsArtifactAndPropagatesBuildError checks Windows archive naming and build failures.
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
	err = run([]string{"--version", "0.5.1", "--out", outDir, "--platform", "windows/amd64"}, func(string) string {
		return base64.StdEncoding.EncodeToString(privateKey)
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run windows release returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "ctyun_0.5.1_windows_amd64.tar.gz")); err != nil {
		t.Fatal(err)
	}

	restoreFail := patchBuildBinary(func(buildOptions) error {
		return os.ErrPermission
	})
	defer restoreFail()
	if err := run([]string{"--version", "0.5.1", "--out", t.TempDir(), "--platform", "linux/amd64"}, func(string) string {
		return base64.StdEncoding.EncodeToString(privateKey)
	}, &bytes.Buffer{}); err == nil {
		t.Fatal("run returned nil error for build failure")
	}
}

// TestRepeatedPlatformBuildsOnce verifies that repeating a target retains the first
// requested order without adding another build or index entry.
func TestRepeatedPlatformBuildsOnce(t *testing.T) {
	opts, err := parseOptions([]string{"--platform", "linux/amd64", "--platform", "darwin/arm64", "--platform", "linux/amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(opts.Platforms, ","); got != "linux/amd64,darwin/arm64" {
		t.Fatalf("build targets = %q", got)
	}
}

// TestReleaseHelpersCoverUtilityBranches checks flag and filesystem helpers.
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

// writeTestArchive writes a minimal archive fixture.
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

// patchBuildBinary replaces compilation for a test and returns its restoration function.
func patchBuildBinary(fn func(buildOptions) error) func() {
	original := buildBinary
	buildBinary = fn
	return func() {
		buildBinary = original
	}
}

// TestFailedCoreArchiveRebuildPreservesPreviousBytes protects an archive still referenced by the current signed index.
func TestFailedCoreArchiveRebuildPreservesPreviousBytes(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "core.tar.gz")
	binary := filepath.Join(dir, "ctyun")
	if err := os.WriteFile(binary, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeArchive(archive, binary, "ctyun"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeArchive(archive, filepath.Join(dir, "missing"), "ctyun"); err == nil {
		t.Fatal("expected archive build failure")
	}
	after, err := os.ReadFile(archive)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("previous archive changed after failed rebuild: %v", err)
	}
}

// TestCoreArchiveRejectsMissingParent checks the staging boundary before creating an archive.
func TestCoreArchiveRejectsMissingParent(t *testing.T) {
	if err := writeArchive(filepath.Join(t.TempDir(), "missing", "core.tar.gz"), "ctyun", "ctyun"); err == nil {
		t.Fatal("expected missing parent error")
	}
}
