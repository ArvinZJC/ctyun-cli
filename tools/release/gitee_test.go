/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
)

// TestWriteGiteePluginRegistryUsesImmutableVersionReleases verifies signed
// output, immutable URLs, and deterministic ordering across channel ties.
func TestWriteGiteePluginRegistryUsesImmutableVersionReleases(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	canonical := registry.Index{Plugins: []registry.Artifact{
		{Name: "ecs", Version: "0.1.0-alpha.1", Channel: "alpha", Quality: "generated", URL: "ctyun_plugin_ecs_0.1.0-alpha.1.tar.gz", SHA256: strings.Repeat("1", 64)},
		{Name: "ecs", Version: "0.1.0-alpha.1", Channel: "beta", Quality: "generated", URL: "ctyun_plugin_ecs_0.1.0-alpha.1-beta.tar.gz", SHA256: strings.Repeat("3", 64)},
		{Name: "zos", Version: "0.1.0-beta.2", Channel: "beta", Quality: "generated", URL: "ctyun_plugin_zos_0.1.0-beta.2.tar.gz", SHA256: strings.Repeat("2", 64)},
	}}
	canonicalBytes, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "index.json"), canonicalBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeGiteePluginRegistry(outDir, "https://gitee.example.test/owner/repo/releases/download", privateKey); err != nil {
		t.Fatalf("write Gitee registry: %v", err)
	}
	giteeDir := filepath.Join(outDir, "gitee")
	indexBytes, err := os.ReadFile(filepath.Join(giteeDir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	signature, err := os.ReadFile(filepath.Join(giteeDir, "index.sig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := distribution.VerifyIndexSignature(indexBytes, signature, base64.StdEncoding.EncodeToString(publicKey), "Gitee registry"); err != nil {
		t.Fatalf("Gitee index signature: %v", err)
	}
	index, err := registry.LoadIndex(indexBytes)
	if err != nil {
		t.Fatal(err)
	}
	wantURL := "https://gitee.example.test/owner/repo/releases/download/releases%2Fplugins%2Fzos%2F0.1.0-beta.2/ctyun_plugin_zos_0.1.0-beta.2.tar.gz"
	artifact, ok := index.Find("zos", "beta")
	if !ok || artifact.URL != wantURL {
		t.Fatalf("ZOS Gitee artifact = %#v, want URL %s", artifact, wantURL)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(giteeDir, "releases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest giteeReleaseManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != 1 || len(manifest.Releases) != 3 {
		t.Fatalf("Gitee release manifest = %#v", manifest)
	}
	if manifest.Releases[0].Channel != "alpha" || manifest.Releases[1].Channel != "beta" {
		t.Fatalf("ECS release channel order = %#v", manifest.Releases[:2])
	}
	if got := manifest.Releases[2]; got.Tag != "releases/plugins/zos/0.1.0-beta.2" || got.Archive != "ctyun_plugin_zos_0.1.0-beta.2.tar.gz" || got.URL != wantURL {
		t.Fatalf("ZOS release entry = %#v", got)
	}
}

// TestWriteGiteePluginRegistryCoversDefaultsAndFailures verifies that every
// publication stage returns its underlying read, validation, encoding, or
// filesystem failure.
func TestWriteGiteePluginRegistryCoversDefaultsAndFailures(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	canonical := registry.Index{Plugins: []registry.Artifact{{
		Name: "common", Version: "0.1.0", Channel: "stable", Quality: "curated",
		URL: "ctyun_plugin_common_0.1.0.tar.gz", SHA256: strings.Repeat("4", 64),
	}}}
	canonicalBytes, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	writeCanonical := func(t *testing.T, outDir string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(outDir, "index.json"), canonicalBytes, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("default download root", func(t *testing.T) {
		outDir := t.TempDir()
		writeCanonical(t, outDir)
		if err := writeGiteePluginRegistry(outDir, "", privateKey); err != nil {
			t.Fatalf("write default Gitee registry: %v", err)
		}
		indexBytes, err := os.ReadFile(filepath.Join(outDir, "gitee", "index.json"))
		if err != nil {
			t.Fatal(err)
		}
		index, err := registry.LoadIndex(indexBytes)
		if err != nil {
			t.Fatal(err)
		}
		if got := index.Plugins[0].URL; !strings.HasPrefix(got, defaultGiteePluginDownloadRoot+"/") {
			t.Fatalf("default Gitee URL = %q", got)
		}
	})

	tests := []struct {
		name    string
		prepare func(*testing.T, string)
	}{
		{name: "missing canonical index"},
		{name: "invalid canonical index", prepare: func(t *testing.T, outDir string) {
			if err := os.WriteFile(filepath.Join(outDir, "index.json"), []byte(`{`), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "Gitee directory blocked", prepare: func(t *testing.T, outDir string) {
			writeCanonical(t, outDir)
			if err := os.WriteFile(filepath.Join(outDir, "gitee"), []byte("blocked"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "index output blocked", prepare: func(t *testing.T, outDir string) {
			writeCanonical(t, outDir)
			if err := os.MkdirAll(filepath.Join(outDir, "gitee", "index.json"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "signature output blocked", prepare: func(t *testing.T, outDir string) {
			writeCanonical(t, outDir)
			if err := os.MkdirAll(filepath.Join(outDir, "gitee", "index.sig"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "manifest output blocked", prepare: func(t *testing.T, outDir string) {
			writeCanonical(t, outDir)
			if err := os.MkdirAll(filepath.Join(outDir, "gitee", "releases.json"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "index encoding failure", prepare: func(t *testing.T, outDir string) {
			writeCanonical(t, outDir)
			encodeGiteeJSON = func(any) ([]byte, error) { return nil, errors.New("encode index") }
		}},
		{name: "manifest encoding failure", prepare: func(t *testing.T, outDir string) {
			writeCanonical(t, outDir)
			calls := 0
			encodeGiteeJSON = func(value any) ([]byte, error) {
				calls++
				if calls == 2 {
					return nil, errors.New("encode manifest")
				}
				return indentedJSON(value)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalEncoder := encodeGiteeJSON
			t.Cleanup(func() { encodeGiteeJSON = originalEncoder })
			outDir := t.TempDir()
			if test.prepare != nil {
				test.prepare(t, outDir)
			}
			if err := writeGiteePluginRegistry(outDir, "https://gitee.example.test/releases/download", privateKey); err == nil {
				t.Fatal("writeGiteePluginRegistry returned nil error")
			}
		})
	}

	if _, err := indentedJSON(make(chan int)); err == nil {
		t.Fatal("indentedJSON accepted an unsupported value")
	}
}

// TestValidateReleaseOptionsRejectsUnsafeGiteeDownloadRoot verifies that the
// release tool accepts only credential-free HTTPS roots.
func TestValidateReleaseOptionsRejectsUnsafeGiteeDownloadRoot(t *testing.T) {
	valid := releaseOptions{
		Version:                 "0.4.0",
		Channel:                 "stable",
		OutDir:                  t.TempDir(),
		Platforms:               []string{"darwin/arm64"},
		GiteePluginDownloadRoot: "https://gitee.example.test/owner/repo/releases/download",
	}
	if err := validateReleaseOptions(valid); err != nil {
		t.Fatalf("valid options: %v", err)
	}
	for _, root := range []string{"http://gitee.example.test/releases/download", "file:///tmp/releases", "://bad"} {
		t.Run(root, func(t *testing.T) {
			invalid := valid
			invalid.GiteePluginDownloadRoot = root
			if err := validateReleaseOptions(invalid); err == nil || !strings.Contains(err.Error(), "--gitee-plugin-download-root") {
				t.Fatalf("validate error = %v", err)
			}
		})
	}
}
