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
	"path/filepath"
	"strings"
	"testing"

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
	if err := corerelease.VerifyIndexSignature(indexBytes, signature, base64.StdEncoding.EncodeToString(publicKey)); err != nil {
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
	if err := corerelease.VerifySHA256(filepath.Join(outDir, artifact.URL), artifact.SHA256); err != nil {
		t.Fatalf("artifact checksum invalid: %v", err)
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

func patchBuildBinary(fn func(buildOptions) error) func() {
	original := buildBinary
	buildBinary = fn
	return func() {
		buildBinary = original
	}
}
