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
	"errors"
	"io"
	"net/http"
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

func TestUpgradeCheckAutoFallsBackToGitee(t *testing.T) {
	index := []byte(`{"schema":1,"releases":[{"version":"0.3.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`)
	publicKey, signature := signReleaseIndexForTest(t, index)
	restoreVersion := patchVersion("0.2.0")
	defer restoreVersion()
	var stdout bytes.Buffer

	err := Run(Config{
		Args:   []string{"upgrade", "--check"},
		Stdout: &stdout,
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
		HTTPTransport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Host, "github.com") {
				return httpStringResponse(http.StatusInternalServerError, ""), nil
			}
			switch filepath.Base(req.URL.Path) {
			case "core-index.json":
				return httpStringResponse(http.StatusOK, string(index)), nil
			case "core-index.sig":
				return httpStringResponse(http.StatusOK, signature), nil
			default:
				return httpStringResponse(http.StatusNotFound, ""), nil
			}
		}),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "from gitee") {
		t.Fatalf("stdout = %q, want gitee fallback", stdout.String())
	}
}

func TestUpgradeCheckReportsUpToDateAndMissingArtifact(t *testing.T) {
	source, publicKey := writeSignedReleaseSource(t, `{"schema":1,"releases":[{"version":"0.1.0","channel":"stable","artifacts":[{"os":"`+runtime.GOOS+`","arch":"`+runtime.GOARCH+`","url":"ctyun.tar.gz","sha256":"`+strings.Repeat("0", 64)+`"}]}]}`)
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"upgrade", "--check", "--source", source},
		Stdout: &stdout,
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	}); err != nil {
		t.Fatalf("up-to-date check returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "already up to date") {
		t.Fatalf("stdout = %q, want up-to-date message", stdout.String())
	}

	if err := Run(Config{
		Args: []string{"upgrade", "--check", "--source", source, "--channel", "beta"},
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	}); err == nil || !strings.Contains(err.Error(), "no ctyun release found") {
		t.Fatalf("missing artifact error = %v", err)
	}
}

func TestUpgradeRejectsInvalidOptions(t *testing.T) {
	for _, args := range [][]string{
		{"upgrade", "--source"},
		{"upgrade", "--channel"},
		{"upgrade", "--bogus"},
	} {
		if err := Run(Config{Args: args}); err == nil {
			t.Fatalf("Run(%v) returned nil error", args)
		}
	}
}

func TestUpgradePropagatesReleaseSourceErrors(t *testing.T) {
	source, _ := writeSignedReleaseSource(t, `{"schema":1,"releases":[]}`)
	if err := Run(Config{Args: []string{"upgrade", "--check", "--source", source}}); err == nil {
		t.Fatal("Run returned nil error without release public key")
	}
	malformed, key := writeSignedReleaseSource(t, `{`)
	if err := Run(Config{
		Args: []string{"upgrade", "--check", "--source", malformed},
		Env: func(name string) string {
			if name == "CTYUN_RELEASE_PUBLIC_KEY" {
				return key
			}
			return ""
		},
	}); err == nil {
		t.Fatal("Run returned nil error for malformed signed index")
	}
}

func TestUpgradePropagatesArtifactAndInstallErrors(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(source, "ctyun.tar.gz")
	writeUpgradeArchive(t, archive, upgradeBinaryNameForTest(), "new")
	badIndex := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`
	key := writeSignedReleaseIndex(t, source, badIndex)
	env := func(name string) string {
		if name == "CTYUN_RELEASE_PUBLIC_KEY" {
			return key
		}
		return ""
	}
	if err := Run(Config{Args: []string{"upgrade", "--source", source}, Env: env}); err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("bad checksum error = %v", err)
	}

	goodIndex := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + sha256FileForTest(t, archive) + `"}]}]}`
	key = writeSignedReleaseIndex(t, source, goodIndex)
	restoreExecutable := patchCurrentExecutable(func() (string, error) {
		return "", errors.New("no executable")
	})
	if err := Run(Config{Args: []string{"upgrade", "--source", source}, Env: env}); err == nil || !strings.Contains(err.Error(), "no executable") {
		t.Fatalf("current executable error = %v", err)
	}
	restoreExecutable()

	restoreExecutable = patchCurrentExecutable(func() (string, error) {
		return filepath.Join(root, "missing-ctyun"), nil
	})
	defer restoreExecutable()
	if err := Run(Config{Args: []string{"upgrade", "--source", source}, Env: env}); err == nil {
		t.Fatal("Run returned nil error for install failure")
	}
}

func TestUpgradePropagatesArtifactDownloadError(t *testing.T) {
	source, key := writeSignedReleaseSource(t, `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"`+runtime.GOOS+`","arch":"`+runtime.GOARCH+`","url":"https://artifacts.example.test/ctyun.tar.gz","sha256":"`+strings.Repeat("0", 64)+`"}]}]}`)
	err := Run(Config{
		Args: []string{"upgrade", "--source", source},
		Env: func(name string) string {
			if name == "CTYUN_RELEASE_PUBLIC_KEY" {
				return key
			}
			return ""
		},
		HTTPTransport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return httpStringResponse(http.StatusNotFound, ""), nil
		}),
	})
	if err == nil {
		t.Fatal("Run returned nil error for artifact download failure")
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
	publicKey, signature := signReleaseIndexForTest(t, []byte(index))
	if err := os.WriteFile(filepath.Join(root, "core-index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core-index.sig"), []byte(signature), 0o644); err != nil {
		t.Fatal(err)
	}
	return publicKey
}

func signReleaseIndexForTest(t *testing.T, index []byte) (string, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, index)
	return base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(signature)
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

func patchVersion(value string) func() {
	original := version.Version
	version.Version = value
	return func() {
		version.Version = original
	}
}

func httpStringResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
