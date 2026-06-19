/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

func TestUpgradeCheckDevelopmentBuildWithoutSource(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{Args: []string{"upgrade", "--check"}, Stdout: &stdout})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "self-upgrade is unavailable for development builds") {
		t.Fatalf("stdout = %q, want development-build guidance", stdout.String())
	}
}

func TestUpgradeCheckUsesExplicitSignedSource(t *testing.T) {
	source, publicKey := writeSignedReleaseSource(t, `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"`+runtime.GOOS+`","arch":"`+runtime.GOARCH+`","url":"ctyun.tar.gz","sha256":"`+strings.Repeat("0", 64)+`"}]}]}`)
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"upgrade", "--check", "--source", source},
		Stdout: &stdout,
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "0.2.0") {
		t.Fatalf("stdout = %q, want available version", stdout.String())
	}
}

func TestUpgradeCheckUsesEmbeddedReleasePublicKey(t *testing.T) {
	source, publicKey := writeSignedReleaseSource(t, `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"`+runtime.GOOS+`","arch":"`+runtime.GOARCH+`","url":"ctyun.tar.gz","sha256":"`+strings.Repeat("0", 64)+`"}]}]}`)
	restoreKey := patchReleasePublicKey(publicKey)
	defer restoreKey()

	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"upgrade", "--check", "--source", source},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "0.2.0") {
		t.Fatalf("stdout = %q, want available version", stdout.String())
	}
}

func TestUpgradeInstallsExplicitSignedSource(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, upgradeBinaryNameForTest())
	if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(source, "ctyun.tar.gz")
	writeUpgradeArchive(t, archive, upgradeBinaryNameForTest(), "new")
	sum := sha256FileForTest(t, archive)
	index := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + sum + `"}]}]}`
	publicKey := writeSignedReleaseIndex(t, source, index)
	restoreExecutable := patchCurrentExecutable(func() (string, error) {
		return current, nil
	})
	defer restoreExecutable()

	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"upgrade", "--source", source},
		Stdout: &stdout,
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	data, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("installed binary = %q, want new", data)
	}
	if !strings.Contains(stdout.String(), "upgraded ctyun 0.1.0-dev -> 0.2.0") {
		t.Fatalf("stdout = %q, want upgrade summary", stdout.String())
	}
}

func writeSignedReleaseSource(t *testing.T, index string) (string, string) {
	t.Helper()
	root := t.TempDir()
	publicKey := writeSignedReleaseIndex(t, root, index)
	return root, publicKey
}

func writeSignedReleaseIndex(t *testing.T, root string, index string) string {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, []byte(index))
	if err := os.WriteFile(filepath.Join(root, "core-index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core-index.sig"), []byte(base64.StdEncoding.EncodeToString(signature)), 0o644); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(publicKey)
}

func writeUpgradeArchive(t *testing.T, path string, name string, body string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func sha256FileForTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func upgradeBinaryNameForTest() string {
	if runtime.GOOS == "windows" {
		return "ctyun.exe"
	}
	return "ctyun"
}

func patchCurrentExecutable(fn func() (string, error)) func() {
	original := currentExecutable
	currentExecutable = fn
	return func() {
		currentExecutable = original
	}
}

func patchReleasePublicKey(key string) func() {
	original := version.ReleasePublicKey
	version.ReleasePublicKey = key
	return func() {
		version.ReleasePublicKey = original
	}
}
