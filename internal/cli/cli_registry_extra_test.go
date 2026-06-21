/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

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

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
)

func TestRegistryArtifactHelpersAndHTTPUtilities(t *testing.T) {
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`)
	if _, _, err := findRegistryArtifact(registryRoot, "missing", "", nil, ""); err == nil {
		t.Fatal("findRegistryArtifact returned nil error for missing plugin")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if _, _, err := findRegistryArtifact(badRegistry, "ecs", "", nil, ""); err == nil {
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

	if err := updateAllPlugins(io.Discard, filepath.Join(t.TempDir(), "missing"), registryRoot, "", nil, "", "en-US"); err != nil {
		t.Fatalf("updateAllPlugins missing root returned error: %v", err)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := updateAllPlugins(io.Discard, rootFile, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for file root")
	}
	if err := updateAllPlugins(io.Discard, root, filepath.Join(t.TempDir(), "missing-registry"), "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for missing registry")
	}
	if err := updateAllPlugins(io.Discard, root, "", "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error without registry")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := updateAllPlugins(io.Discard, root, badRegistry, "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for malformed registry")
	}

	mustWrite(t, filepath.Join(root, "note.txt"), "ignore")
	if err := updateAllPlugins(io.Discard, root, registryRoot, "", nil, "", "en-US"); err != nil {
		t.Fatalf("updateAllPlugins non-dir entry returned error: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if err := updateAllPlugins(io.Discard, root, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for invalid installed bundle")
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.2.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs-0.1.0"}]}`)
	var stdout bytes.Buffer
	if err := updateAllPlugins(&stdout, root, registryRoot, "", nil, "", "en-US"); err != nil {
		t.Fatalf("updateAllPlugins up-to-date returned error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("updateAllPlugins no-op output = %q, want empty", stdout.String())
	}

	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"missing-artifact"}]}`)
	if err := updateAllPlugins(io.Discard, root, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for missing artifact source")
	}
	artifactDir := filepath.Join(registryRoot, "ecs-0.3.0")
	writeVersionedBundle(t, artifactDir, "ecs", "0.3.0")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0","sha256":"bad"}]}`)
	if err := updateAllPlugins(io.Discard, root, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for bad checksum")
	}

	if err := updateOnePlugin(io.Discard, root, "", "ecs", "", nil, "", "en-US"); err == nil {
		t.Fatal("updateOnePlugin returned nil error without registry")
	}
	if err := updateOnePlugin(io.Discard, root, registryRoot, "missing", "", nil, "", "en-US"); err == nil {
		t.Fatal("updateOnePlugin returned nil error for missing installed plugin")
	}
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"ecs-0.2.0"}]}`)
	stdout.Reset()
	if err := updateOnePlugin(&stdout, root, registryRoot, "ecs", "", nil, "", "en-US"); err != nil {
		t.Fatalf("updateOnePlugin up-to-date returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "up to date") {
		t.Fatalf("updateOnePlugin output = %q, want up to date", stdout.String())
	}
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"missing-artifact"}]}`)
	if err := updateOnePlugin(io.Discard, root, registryRoot, "ecs", "", nil, "", "en-US"); err == nil {
		t.Fatal("updateOnePlugin returned nil error for missing update artifact")
	}
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0","sha256":"bad"}]}`)
	if err := updateOnePlugin(io.Discard, root, registryRoot, "ecs", "", nil, "", "en-US"); err == nil {
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
	if err := updateAllPlugins(&stdout, root, registryRoot, "", nil, "", "en-US"); err != nil {
		t.Fatalf("updateAllPlugins install returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.2.0.") {
		t.Fatalf("updateAllPlugins output = %q", stdout.String())
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	stdout.Reset()
	if err := updateOnePlugin(&stdout, root, registryRoot, "ecs", "", nil, "", "en-US"); err != nil {
		t.Fatalf("updateOnePlugin install returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.2.0.") {
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

	if err := updateAllPlugins(io.Discard, root, "https://registry.example.test/root", "", transport, publicKeyText, "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for HTTP artifact without sha256")
	}
	if err := updateOnePlugin(io.Discard, root, t.TempDir(), "ecs", "", nil, "", "en-US"); err == nil {
		t.Fatal("updateOnePlugin returned nil error for missing registry artifact")
	}
	if err := updateOnePlugin(io.Discard, root, "https://registry.example.test/root", "ecs", "", transport, publicKeyText, "en-US"); err == nil {
		t.Fatal("updateOnePlugin returned nil error for HTTP artifact without sha256")
	}
}

func TestPluginDistributionHelperEdges(t *testing.T) {
	if !isHTTPURL("https://example.test") || isHTTPURL("file:///tmp/plugin") {
		t.Fatal("isHTTPURL result mismatch")
	}
	if _, err := registrySource(distribution.Source{}); err == nil {
		t.Fatal("registrySource returned nil error for empty distribution source")
	}
	if _, err := registrySource(123); err == nil {
		t.Fatal("registrySource returned nil error for unsupported source type")
	}
	if err := updateAllBundledPlugins(io.Discard, filepath.Join(t.TempDir(), "missing"), "en-US"); err != nil {
		t.Fatalf("updateAllBundledPlugins missing root returned error: %v", err)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := updateAllBundledPlugins(io.Discard, rootFile, "en-US"); err == nil {
		t.Fatal("updateAllBundledPlugins returned nil error for file root")
	}

	restore := patchVersion("0.1.0-dev")
	defer restore()
	if _, err := bundledPluginSource("../bad"); err == nil {
		t.Fatal("bundledPluginSource returned nil error for invalid name")
	}
	if _, err := bundledPluginSource("missing"); err == nil {
		t.Fatal("bundledPluginSource returned nil error for missing bundled plugin")
	}
}

func TestPluginHostedInstallReportsInstallFailure(t *testing.T) {
	registryRoot := t.TempDir()
	root := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "broken"), "not a plugin bundle")
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"broken"}]}`)

	if err := installPluginFromHostedSource(io.Discard, root, distribution.Source{Name: "test", URL: registryRoot}, "ecs", "", nil, "", "en-US"); err == nil {
		t.Fatal("installPluginFromHostedSource returned nil error for invalid artifact bundle")
	}
}

func TestPluginBundledUpdateErrorPaths(t *testing.T) {
	restore := patchVersion("0.1.0-dev")
	defer restore()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bad"), 0o755); err != nil {
		t.Fatalf("create bad installed plugin dir: %v", err)
	}
	if err := updateOneBundledPlugin(io.Discard, root, "bad", "en-US"); err == nil {
		t.Fatal("updateOneBundledPlugin returned nil error for invalid installed bundle")
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "missing"), "missing", "0.1.0")
	if err := updateOneBundledPlugin(io.Discard, root, "missing", "en-US"); err == nil {
		t.Fatal("updateOneBundledPlugin returned nil error for missing bundled source")
	}

	root = t.TempDir()
	fixtureRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(fixtureRoot, "plugins", "ecs"), 0o755); err != nil {
		t.Fatalf("create invalid bundled plugin: %v", err)
	}
	t.Chdir(fixtureRoot)
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	if err := updateOneBundledPlugin(io.Discard, root, "ecs", "en-US"); err == nil {
		t.Fatal("updateOneBundledPlugin returned nil error for invalid bundled source")
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	if err := os.RemoveAll(filepath.Join(fixtureRoot, "plugins", "ecs")); err != nil {
		t.Fatalf("remove invalid bundled plugin: %v", err)
	}
	writeVersionedBundle(t, filepath.Join(fixtureRoot, "plugins", "ecs"), "ecs", "0.2.0")
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatalf("chmod plugin root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	if err := updateOneBundledPlugin(io.Discard, root, "ecs", "en-US"); err == nil {
		t.Fatal("updateOneBundledPlugin returned nil error for unwritable plugin root")
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatalf("restore plugin root permissions: %v", err)
	}

	root = t.TempDir()
	mustWrite(t, filepath.Join(root, "note.txt"), "ignore")
	if err := updateAllBundledPlugins(io.Discard, root, "en-US"); err != nil {
		t.Fatalf("updateAllBundledPlugins non-dir entry returned error: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if err := updateAllBundledPlugins(io.Discard, root, "en-US"); err == nil {
		t.Fatal("updateAllBundledPlugins returned nil error for invalid installed bundle")
	}
}

func TestListPluginUpdatesCoverNoopAndErrorPaths(t *testing.T) {
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[]}`)
	if err := listPluginUpdates(io.Discard, filepath.Join(t.TempDir(), "missing"), registryRoot, "", nil, "", "en-US"); err != nil {
		t.Fatalf("listPluginUpdates missing root returned error: %v", err)
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := listPluginUpdates(io.Discard, rootFile, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("listPluginUpdates returned nil error for file root")
	}
	if err := listPluginUpdates(io.Discard, t.TempDir(), filepath.Join(t.TempDir(), "missing-registry"), "", nil, "", "en-US"); err == nil {
		t.Fatal("listPluginUpdates returned nil error for missing registry")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := listPluginUpdates(io.Discard, t.TempDir(), badRegistry, "", nil, "", "en-US"); err == nil {
		t.Fatal("listPluginUpdates returned nil error for malformed registry")
	}

	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "note.txt"), "ignore")
	if err := listPluginUpdates(io.Discard, root, registryRoot, "", nil, "", "en-US"); err != nil {
		t.Fatalf("listPluginUpdates non-dir entry returned error: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "bad"), 0o755); err != nil {
		t.Fatalf("create bad plugin dir: %v", err)
	}
	if err := listPluginUpdates(io.Discard, root, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("listPluginUpdates returned nil error for invalid installed bundle")
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.2.0")
	var stdout bytes.Buffer
	if err := listPluginUpdates(&stdout, root, registryRoot, "", nil, "", "en-US"); err != nil {
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
	if _, got, err := readRegistryIndex("https://registry.example.test/root", transport, base64.StdEncoding.EncodeToString(publicKey)); err != nil || string(got) != string(index) {
		t.Fatalf("readRegistryIndex HTTP = %q, %v", string(got), err)
	}
	if _, _, err := readRegistryIndex("https://registry.example.test/root", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "index.sig") {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("missing"))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(index))}, nil
	}), "key"); err == nil {
		t.Fatal("readRegistryIndex returned nil error for missing signature")
	}
	if _, _, err := readRegistryIndex("https://registry.example.test/root", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	}), "key"); err == nil {
		t.Fatal("readRegistryIndex returned nil error for index fetch failure")
	}
	if _, _, err := readRegistryIndex("https://registry.example.test/root", transport, "bad-key"); err == nil {
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

	originalCreateTempArtifactFile := createTempArtifactFile
	t.Cleanup(func() { createTempArtifactFile = originalCreateTempArtifactFile })
	createTempArtifactFile = func() (tempArtifactFile, error) {
		return nil, errors.New("create temp")
	}
	if _, _, err := downloadRegistryArtifact("https://registry.example.test/plugin.tar.gz", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("artifact"))}, nil
	})); err == nil {
		t.Fatal("downloadRegistryArtifact returned nil error for temp file creation failure")
	}

	createTempArtifactFile = func() (tempArtifactFile, error) {
		return fakeTempArtifactFile{name: filepath.Join(t.TempDir(), "artifact.tar.gz"), writeErr: errors.New("write")}, nil
	}
	if _, _, err := downloadRegistryArtifact("https://registry.example.test/plugin.tar.gz", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("artifact"))}, nil
	})); err == nil {
		t.Fatal("downloadRegistryArtifact returned nil error for temp file write failure")
	}

	createTempArtifactFile = func() (tempArtifactFile, error) {
		return fakeTempArtifactFile{name: filepath.Join(t.TempDir(), "artifact.tar.gz"), closeErr: errors.New("close")}, nil
	}
	if _, _, err := downloadRegistryArtifact("https://registry.example.test/plugin.tar.gz", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader("artifact"))}, nil
	})); err == nil {
		t.Fatal("downloadRegistryArtifact returned nil error for temp file close failure")
	}
	createTempArtifactFile = originalCreateTempArtifactFile

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
