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
	if !strings.Contains(stdout.String(), "Self-upgrade is unavailable for development builds") {
		t.Fatalf("stdout = %q, want development-build guidance", stdout.String())
	}
	if strings.Contains(stdout.String(), "path-or-url") || strings.Contains(stdout.String(), "URL") {
		t.Fatalf("stdout = %q, want named hosted source guidance only", stdout.String())
	}
}

func TestUpgradeCheckUsesExplicitSignedSource(t *testing.T) {
	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()
	index := []byte(`{"schema":1,"releases":[{"version":"0.3.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`)
	publicKey, transport := signedReleaseTransport(t, index, nil)
	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"upgrade", "--check", "--source", "github"},
		Stdout:        &stdout,
		HTTPTransport: transport,
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
	if !strings.Contains(stdout.String(), "0.3.0") {
		t.Fatalf("stdout = %q, want available version", stdout.String())
	}
}

func TestUpgradeCheckUsesEmbeddedReleasePublicKey(t *testing.T) {
	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()
	index := []byte(`{"schema":1,"releases":[{"version":"0.3.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`)
	publicKey, transport := signedReleaseTransport(t, index, nil)
	restoreKey := patchReleasePublicKey(publicKey)
	defer restoreKey()

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"upgrade", "--check", "--source", "github"},
		Stdout:        &stdout,
		HTTPTransport: transport,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "0.3.0") {
		t.Fatalf("stdout = %q, want available version", stdout.String())
	}
}

func TestUpgradeInstallsExplicitSignedSource(t *testing.T) {
	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()

	root := t.TempDir()
	current := filepath.Join(root, upgradeBinaryNameForTest())
	if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "ctyun.tar.gz")
	writeUpgradeArchive(t, archive, upgradeBinaryNameForTest(), "new")
	archiveBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBytes)
	index := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + hex.EncodeToString(sum[:]) + `"}]}]}`
	publicKey, transport := signedReleaseTransport(t, []byte(index), map[string][]byte{"ctyun.tar.gz": archiveBytes})

	var stdout bytes.Buffer
	err = Run(Config{
		Args:              []string{"upgrade", "--source", "github"},
		Stdout:            &stdout,
		HTTPTransport:     transport,
		CurrentExecutable: func() (string, error) { return current, nil },
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
	if !strings.Contains(stdout.String(), "Upgraded ctyun: 0.2.0-dev -> 0.2.0.") {
		t.Fatalf("stdout = %q, want upgrade summary", stdout.String())
	}
}

func TestUpgradeApplyReportsProgressPhases(t *testing.T) {
	t.Cleanup(patchVersion("0.2.0-dev"))
	root := t.TempDir()
	current := filepath.Join(root, upgradeBinaryNameForTest())
	if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "ctyun.tar.gz")
	writeUpgradeArchive(t, archive, upgradeBinaryNameForTest(), "new")
	archiveBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBytes)
	index := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + hex.EncodeToString(sum[:]) + `"}]}]}`
	publicKey, transport := signedReleaseTransport(t, []byte(index), map[string][]byte{"ctyun.tar.gz": archiveBytes})
	display := &recordingOperationDisplay{}
	originalFactory := operationProgressFactory
	t.Cleanup(func() { operationProgressFactory = originalFactory })
	operationProgressFactory = func(io.Writer) operationDisplay { return display }

	var stdout bytes.Buffer
	err = Run(Config{
		Args:          []string{"upgrade", "--source", "github"},
		Stdout:        &stdout,
		Stderr:        &bytes.Buffer{},
		HTTPTransport: transport,
		CurrentExecutable: func() (string, error) {
			return current, nil
		},
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("upgrade returned error: %v", err)
	}
	for _, want := range []string{"Downloading ctyun", "Verifying ctyun", "Installing ctyun"} {
		found := false
		for _, got := range display.updates {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("progress updates %v missing %q", display.updates, want)
		}
	}
	if !display.cleared {
		t.Fatal("core upgrade progress was not cleared")
	}
	if got := strings.TrimSpace(stdout.String()); got != "Upgraded ctyun: 0.2.0-dev -> 0.2.0." {
		t.Fatalf("upgrade summary = %q", got)
	}
}

func TestUpgradeApplyHandlesProgressDisplayErrors(t *testing.T) {
	t.Cleanup(patchVersion("0.2.0-dev"))
	root := t.TempDir()
	archive := filepath.Join(root, "ctyun.tar.gz")
	writeUpgradeArchive(t, archive, upgradeBinaryNameForTest(), "new")
	archiveBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archiveBytes)
	index := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + hex.EncodeToString(sum[:]) + `"}]}]}`
	publicKey, transport := signedReleaseTransport(t, []byte(index), map[string][]byte{"ctyun.tar.gz": archiveBytes})

	for _, testCase := range []struct {
		name        string
		display     operationDisplay
		wantCleared bool
	}{
		{name: "start", display: &recordingOperationDisplay{err: errors.New("start failed")}},
		{name: "download", display: &failingUpdateOperationDisplay{failCall: 1}, wantCleared: true},
		{name: "verify", display: &failingUpdateOperationDisplay{failCall: 2}, wantCleared: true},
		{name: "install", display: &failingUpdateOperationDisplay{failCall: 3}, wantCleared: true},
		{name: "complete", display: &failingUpdateOperationDisplay{failCall: 4}, wantCleared: true},
		{name: "clear", display: &failingUpdateOperationDisplay{clearErr: errors.New("clear failed")}, wantCleared: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			current := filepath.Join(t.TempDir(), upgradeBinaryNameForTest())
			if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
				t.Fatal(err)
			}
			originalFactory := operationProgressFactory
			t.Cleanup(func() { operationProgressFactory = originalFactory })
			operationProgressFactory = func(io.Writer) operationDisplay { return testCase.display }
			err := Run(Config{
				Args:          []string{"upgrade", "--source", "github"},
				Stdout:        io.Discard,
				Stderr:        io.Discard,
				HTTPTransport: transport,
				CurrentExecutable: func() (string, error) {
					return current, nil
				},
				Env: func(key string) string {
					if key == "CTYUN_RELEASE_PUBLIC_KEY" {
						return publicKey
					}
					return ""
				},
			})
			if err == nil {
				t.Fatal("progress display failure returned nil error")
			}
			if display, ok := testCase.display.(*failingUpdateOperationDisplay); ok && testCase.wantCleared && !display.cleared {
				t.Fatal("progress display failure did not clear the line")
			}
		})
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
	t.Cleanup(patchVersion("0.1.0"))
	index := []byte(`{"schema":1,"releases":[{"version":"0.1.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`)
	publicKey, transport := signedReleaseTransport(t, index, nil)
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"upgrade", "--check", "--source", "github"},
		Stdout:        &stdout,
		HTTPTransport: transport,
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
		Args:          []string{"upgrade", "--check", "--source", "github", "--channel", "beta"},
		HTTPTransport: transport,
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	}); err == nil {
		t.Fatalf("missing artifact error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.release_not_found")
	}
}

func TestUpgradeCheckDefaultsToReleaseBuildChannel(t *testing.T) {
	restoreVersion := patchVersion("0.1.0")
	version.Channel = "beta"
	t.Cleanup(restoreVersion)
	index := []byte(`{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"stable.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]},{"version":"0.3.0","channel":"beta","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"beta.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`)
	publicKey, transport := signedReleaseTransport(t, index, nil)
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"upgrade", "--check", "--source", "github"},
		Stdout:        &stdout,
		HTTPTransport: transport,
		Env:           releasePublicKeyEnv(publicKey),
	}); err != nil {
		t.Fatalf("upgrade check returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "ctyun 0.3.0 is available") || !strings.Contains(stdout.String(), "beta.tar.gz") {
		t.Fatalf("stdout = %q, want beta release", stdout.String())
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
	index := []byte(`{"schema":1,"releases":[]}`)
	_, transport := signedReleaseTransport(t, index, nil)
	if err := Run(Config{Args: []string{"upgrade", "--check", "--source", "github"}, HTTPTransport: transport}); err == nil {
		t.Fatal("Run returned nil error without release public key")
	}
	malformedIndex := []byte(`{`)
	key, malformedTransport := signedReleaseTransport(t, malformedIndex, nil)
	if err := Run(Config{
		Args:          []string{"upgrade", "--check", "--source", "github"},
		HTTPTransport: malformedTransport,
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
	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()

	root := t.TempDir()
	archive := filepath.Join(root, "ctyun.tar.gz")
	writeUpgradeArchive(t, archive, upgradeBinaryNameForTest(), "new")
	archiveBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	badIndex := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`
	key, badTransport := signedReleaseTransport(t, []byte(badIndex), map[string][]byte{"ctyun.tar.gz": archiveBytes})
	env := func(name string) string {
		if name == "CTYUN_RELEASE_PUBLIC_KEY" {
			return key
		}
		return ""
	}
	var stderr bytes.Buffer
	if err := Run(Config{Args: []string{"upgrade", "--source", "github"}, Stderr: &stderr, Env: env, HTTPTransport: badTransport}); err == nil {
		t.Fatalf("bad checksum error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.operation_batch_failed")
		if !strings.Contains(stderr.String(), "sha256 mismatch") {
			t.Fatalf("checksum stderr = %q", stderr.String())
		}
	}

	goodIndex := `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"ctyun.tar.gz","sha256":"` + sha256FileForTest(t, archive) + `"}]}]}`
	key, goodTransport := signedReleaseTransport(t, []byte(goodIndex), map[string][]byte{"ctyun.tar.gz": archiveBytes})
	stderr.Reset()
	if err := Run(Config{
		Args:              []string{"upgrade", "--source", "github"},
		Stderr:            &stderr,
		Env:               env,
		HTTPTransport:     goodTransport,
		CurrentExecutable: func() (string, error) { return "", errors.New("no executable") },
	}); err == nil || !strings.Contains(stderr.String(), "no executable") {
		t.Fatalf("current executable error = %v", err)
	}

	stderr.Reset()
	if err := Run(Config{
		Args:              []string{"upgrade", "--source", "github"},
		Stderr:            &stderr,
		Env:               env,
		HTTPTransport:     goodTransport,
		CurrentExecutable: func() (string, error) { return filepath.Join(root, "missing-ctyun"), nil },
	}); err == nil {
		t.Fatal("Run returned nil error for install failure")
	}
}

func TestUpgradePropagatesArtifactDownloadError(t *testing.T) {
	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()

	index := []byte(`{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"` + runtime.GOOS + `","arch":"` + runtime.GOARCH + `","url":"https://artifacts.example.test/ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`)
	key, transport := signedReleaseTransport(t, index, nil)
	err := Run(Config{
		Args: []string{"upgrade", "--source", "github"},
		Env: func(name string) string {
			if name == "CTYUN_RELEASE_PUBLIC_KEY" {
				return key
			}
			return ""
		},
		HTTPTransport: transport,
	})
	if err == nil {
		t.Fatal("Run returned nil error for artifact download failure")
	}
}

func TestUpgradePropagatesSourceAndDevelopmentWriterErrors(t *testing.T) {
	if err := runUpgrade(io.Discard, io.Discard, []string{"--source", "bad"}, func(string) string { return "" }, nil, "en-US", os.Executable); err == nil {
		t.Fatal("runUpgrade returned nil error for invalid source")
	}
	getenv := func(key string) string {
		if key == "CTYUN_UPGRADE_SOURCE" {
			return "bad"
		}
		return ""
	}
	if err := runUpgrade(io.Discard, io.Discard, nil, getenv, nil, "en-US", os.Executable); err == nil {
		t.Fatal("runUpgrade returned nil error for invalid env source")
	}

	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()
	if err := runUpgrade(failingWriter{}, io.Discard, nil, func(string) string { return "" }, nil, "en-US", os.Executable); err == nil {
		t.Fatal("runUpgrade returned nil error for development message writer failure")
	}
}

func TestUpgradeBinaryNameForGOOS(t *testing.T) {
	if got := upgradeBinaryNameForGOOS("/usr/local/bin/ctyun", "linux"); got != "ctyun" {
		t.Fatalf("upgradeBinaryNameForGOOS linux = %q", got)
	}
	if got := upgradeBinaryNameForGOOS(`C:\bin\ctyun.exe`, "windows"); got != "ctyun.exe" {
		t.Fatalf("upgradeBinaryNameForGOOS windows = %q", got)
	}
}

func signedReleaseTransport(t *testing.T, index []byte, artifacts map[string][]byte) (string, http.RoundTripper) {
	t.Helper()
	publicKey, signature := signReleaseIndexForTest(t, index)
	return publicKey, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch filepath.Base(req.URL.Path) {
		case "core-index.json":
			return httpStringResponse(http.StatusOK, string(index)), nil
		case "core-index.sig":
			return httpStringResponse(http.StatusOK, signature), nil
		default:
			if body, ok := artifacts[filepath.Base(req.URL.Path)]; ok {
				return &http.Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK), Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
			}
			return httpStringResponse(http.StatusNotFound, ""), nil
		}
	})
}

func releasePublicKeyEnv(publicKey string) func(string) string {
	return func(key string) string {
		if key == "CTYUN_RELEASE_PUBLIC_KEY" {
			return publicKey
		}
		return ""
	}
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

func patchReleasePublicKey(key string) func() {
	original := version.ReleasePublicKey
	version.ReleasePublicKey = key
	return func() {
		version.ReleasePublicKey = original
	}
}

func patchVersion(value string) func() {
	original := version.Version
	originalChannel := version.Channel
	version.Version = value
	if strings.HasSuffix(value, "-dev") {
		version.Channel = "dev"
	} else {
		version.Channel = "stable"
	}
	return func() {
		version.Version = original
		version.Channel = originalChannel
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
