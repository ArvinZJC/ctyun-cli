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

func patchBuildBinary(fn func(buildOptions) error) func() {
	original := buildBinary
	buildBinary = fn
	return func() {
		buildBinary = original
	}
}
