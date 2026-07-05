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
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
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
	if err := searchPlugins(io.Discard, t.TempDir(), "", "", "ecs", nil, "", globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("searchPlugins returned nil error without registry")
	}
	if err := searchPlugins(io.Discard, t.TempDir(), filepath.Join(t.TempDir(), "missing"), "", "ecs", nil, "", globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("searchPlugins returned nil error for missing registry")
	}
	if err := searchPlugins(io.Discard, t.TempDir(), badRegistry, "", "ecs", nil, "", globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("searchPlugins returned nil error for malformed registry")
	}
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := searchPlugins(io.Discard, rootFile, registryRoot, "", "ecs", nil, "", globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("searchPlugins returned nil error for invalid plugin root")
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

}

func TestPluginReinstallRegistryErrorPaths(t *testing.T) {
	if err := reinstallPluginsFromHostedSource(io.Discard, t.TempDir(), distribution.Source{Name: "test", URL: filepath.Join(t.TempDir(), "missing")}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("reinstallPluginsFromHostedSource returned nil error for missing registry")
	}

	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := reinstallPluginsFromHostedSource(io.Discard, t.TempDir(), distribution.Source{Name: "test", URL: badRegistry}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("reinstallPluginsFromHostedSource returned nil error for malformed registry")
	}

	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[]}`)
	if err := reinstallPluginsFromHostedSource(io.Discard, t.TempDir(), distribution.Source{Name: "test", URL: registryRoot}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("reinstallPluginsFromHostedSource returned nil error for missing installed plugin")
	}

	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.2.0")
	if err := reinstallPluginsFromHostedSource(io.Discard, pluginRoot, distribution.Source{Name: "test", URL: registryRoot}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("reinstallPluginsFromHostedSource returned nil error for missing registry artifact")
	}
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz","sha256":"bad"}]}`)
	mustWrite(t, filepath.Join(registryRoot, "ecs.tar.gz"), "not the expected artifact")
	if err := reinstallPluginsFromHostedSource(io.Discard, pluginRoot, distribution.Source{Name: "test", URL: registryRoot}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("reinstallPluginsFromHostedSource returned nil error for artifact reinstall failure")
	}

	if err := reinstallRegistryArtifact(io.Discard, t.TempDir(), distribution.Source{Name: "test"}, registry.Artifact{Name: "ecs", URL: "https://registry.example.test/ecs.tar.gz"}, nil, "en-US"); err == nil {
		t.Fatal("reinstallRegistryArtifact returned nil error for unverified HTTP artifact")
	}

	artifactRoot := t.TempDir()
	mustWrite(t, filepath.Join(artifactRoot, "ecs.tar.gz"), "not the expected artifact")
	if err := reinstallRegistryArtifact(io.Discard, t.TempDir(), distribution.Source{Name: "test", URL: artifactRoot}, registry.Artifact{Name: "ecs", URL: "ecs.tar.gz", SHA256: "bad"}, nil, "en-US"); err == nil {
		t.Fatal("reinstallRegistryArtifact returned nil error for checksum mismatch")
	}
	if err := reinstallRegistryArtifact(io.Discard, t.TempDir(), distribution.Source{Name: "test", URL: artifactRoot}, registry.Artifact{Name: "ecs", URL: "missing.tar.gz"}, nil, "en-US"); err == nil {
		t.Fatal("reinstallRegistryArtifact returned nil error for missing local artifact")
	}
}

func TestPluginReinstallBundledErrorPaths(t *testing.T) {
	pluginRoot := t.TempDir()
	if _, err := reinstallTargets(pluginRoot, []string{"../ecs"}, false); err == nil {
		t.Fatal("reinstallTargets returned nil error for unsafe name")
	}
	if _, err := reinstallTargets(pluginRoot, []string{"ecs"}, false); err == nil {
		t.Fatal("reinstallTargets returned nil error for missing installed plugin")
	}
	if err := reinstallBundledPlugins(io.Discard, pluginRoot, []string{"ecs"}, false, "en-US"); err == nil {
		t.Fatal("reinstallBundledPlugins returned nil error for missing installed plugin")
	}

	badRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(badRoot, "broken"), 0o755); err != nil {
		t.Fatalf("create broken plugin dir: %v", err)
	}
	if _, err := reinstallTargets(badRoot, nil, true); err == nil {
		t.Fatal("reinstallTargets returned nil error for broken installed plugin")
	}

	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.2.0")
	restoreRelease := patchVersion("0.1.0")
	if err := reinstallBundledPlugins(io.Discard, pluginRoot, []string{"ecs"}, false, "en-US"); err == nil {
		t.Fatal("reinstallBundledPlugins returned nil error in released build")
	}
	restoreRelease()

	repoRoot := t.TempDir()
	bundledRoot := filepath.Join(repoRoot, "plugins")
	if err := os.MkdirAll(filepath.Join(bundledRoot, "ecs"), 0o755); err != nil {
		t.Fatalf("create bundled plugin dir: %v", err)
	}
	mustWrite(t, filepath.Join(bundledRoot, "ecs", "plugin.json"), `{`)
	originalCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(repoRoot, "internal", "cli", "cli.go"), 1, true
	}
	t.Cleanup(patchVersion("0.2.0-dev"))
	if err := reinstallBundledPlugins(io.Discard, pluginRoot, []string{"ecs"}, false, "en-US"); err == nil {
		t.Fatal("reinstallBundledPlugins returned nil error for invalid bundled plugin")
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
	if err := updateAllPlugins(failingWriter{}, root, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("updateAllPlugins returned nil error for writer failure")
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

func TestPluginRegistrySourceEdges(t *testing.T) {
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

	restore := patchVersion("0.2.0-dev")
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

	if err := installPluginsFromHostedSource(io.Discard, root, distribution.Source{Name: "test", URL: registryRoot}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("installPluginsFromHostedSource returned nil error for invalid artifact bundle")
	}

	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := installPluginsFromHostedSource(io.Discard, root, distribution.Source{Name: "test", URL: badRegistry}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("installPluginsFromHostedSource returned nil error for malformed registry")
	}
	emptyRegistry := t.TempDir()
	mustWrite(t, filepath.Join(emptyRegistry, "index.json"), `{"plugins":[]}`)
	if err := installPluginsFromHostedSource(io.Discard, root, distribution.Source{Name: "test", URL: emptyRegistry}, []string{"missing"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("installPluginsFromHostedSource returned nil error for missing named plugin")
	}
}

func TestPluginStorefrontHelperEdges(t *testing.T) {
	registryRoot := t.TempDir()
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`)
	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := listAvailablePlugins(io.Discard, rootFile, registryRoot, "", nil, "", globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("listAvailablePlugins returned nil error for file plugin root")
	}
	if err := listAvailablePlugins(io.Discard, t.TempDir(), filepath.Join(t.TempDir(), "missing"), "", nil, "", globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("listAvailablePlugins returned nil error for missing registry")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := listAvailablePlugins(io.Discard, t.TempDir(), badRegistry, "", nil, "", globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("listAvailablePlugins returned nil error for malformed registry")
	}

	root := t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	badRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(badRoot, "broken"), 0o755); err != nil {
		t.Fatalf("create broken plugin dir: %v", err)
	}
	if _, err := availablePluginRows(badRoot, []registry.Artifact{{Name: "ecs", Version: "0.1.0", Channel: "stable", Quality: "reviewed"}}); err == nil {
		t.Fatal("availablePluginRows returned nil error for invalid installed bundle")
	}
	rows, err := availablePluginRows(root, []registry.Artifact{{Name: "ecs", Version: "0.1.0", Channel: "stable", Quality: "reviewed"}, {Name: "vpc", Version: "0.2.0", Channel: "stable", Quality: "reviewed"}})
	if err != nil {
		t.Fatalf("availablePluginRows returned error: %v", err)
	}
	if rows[0]["status"] != "installed" || rows[1]["status"] != "available" {
		t.Fatalf("available rows status = %#v", rows)
	}
	if got := pluginInstallStatus("0.1.0", "0.2.0"); got != "outdated" {
		t.Fatalf("pluginInstallStatus outdated = %q", got)
	}
}

func TestRenderPluginRowsPropagatesOutputErrors(t *testing.T) {
	columns := []output.Column{{Key: "plugin", Label: "Plugin"}}
	rows := []map[string]string{{"plugin": "ecs"}}
	if err := renderPluginRows(io.Discard, rows, columns, globalOptions{Output: "table", Language: "en-US", Filter: "missing=value"}); err == nil {
		t.Fatal("renderPluginRows returned nil error for invalid filter key")
	}
	if err := renderPluginRows(io.Discard, rows, columns, globalOptions{Output: "table", Language: "en-US", Filter: "plugin"}); err == nil {
		t.Fatal("renderPluginRows returned nil error for invalid filter expression")
	}
	if err := renderPluginRows(io.Discard, rows, columns, globalOptions{Output: "table", Language: "en-US", Sort: "-"}); err == nil {
		t.Fatal("renderPluginRows returned nil error for invalid sort expression")
	}
	if err := renderPluginRows(io.Discard, rows, columns, globalOptions{Output: "yaml", Language: "en-US"}); err == nil {
		t.Fatal("renderPluginRows returned nil error for unsupported output")
	}

	originalRenderOutputTable := renderOutputTable
	t.Cleanup(func() { renderOutputTable = originalRenderOutputTable })
	renderOutputTable = func([]map[string]string, []output.Column, output.TableOptions) (string, error) {
		return "", errors.New("render failed")
	}
	if err := renderPluginRows(io.Discard, rows, columns, globalOptions{Output: "table", Language: "en-US"}); err == nil || !strings.Contains(err.Error(), "render failed") {
		t.Fatalf("renderPluginRows table error = %v, want render failed", err)
	}

	renderOutputTable = originalRenderOutputTable
	originalRenderOutputJSON := renderOutputJSON
	t.Cleanup(func() { renderOutputJSON = originalRenderOutputJSON })
	renderOutputJSON = func(any) (string, error) {
		return "", errors.New("json failed")
	}
	if err := renderPluginRows(io.Discard, rows, columns, globalOptions{Output: "json", Language: "en-US"}); err == nil || !strings.Contains(err.Error(), "json failed") {
		t.Fatalf("renderPluginRows json error = %v, want json failed", err)
	}
}

func TestPluginRemoveHelperEdges(t *testing.T) {
	if _, err := parsePluginRemoveOptions([]string{"--all", "ecs"}); err == nil {
		t.Fatal("parsePluginRemoveOptions returned nil error for all/name conflict")
	}

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatalf("create hidden plugin dir: %v", err)
	}
	if err := removePlugins(io.Discard, io.Discard, strings.NewReader(""), root, pluginRemoveOptions{All: true}, globalOptions{Yes: true, Language: "en-US"}); err == nil {
		t.Fatal("removePlugins returned nil error for invalid discovered plugin name")
	}

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	if err := removePlugins(failingWriter{}, io.Discard, strings.NewReader(""), root, pluginRemoveOptions{Names: []string{"ecs"}}, globalOptions{Yes: true, Language: "en-US"}); err == nil {
		t.Fatal("removePlugins returned nil error for writer failure")
	}
}

func TestInstallBundledPluginsAllRequiresDevelopmentBuild(t *testing.T) {
	restoreVersion := patchVersion("0.1.0")
	defer restoreVersion()
	if err := installBundledPlugins(io.Discard, t.TempDir(), nil, true, "en-US"); err == nil {
		t.Fatal("installBundledPlugins --all returned nil error for released build")
	}
}

func TestInstallBundledPluginsAllInstallsDevelopmentBundles(t *testing.T) {
	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()

	root := t.TempDir()
	var stdout bytes.Buffer
	if err := installBundledPlugins(&stdout, root, nil, true, "en-US"); err != nil {
		t.Fatalf("installBundledPlugins --all returned error: %v", err)
	}
	got := stdout.String()
	for _, name := range []string{"ecs", "region"} {
		if !strings.Contains(got, "Installed "+name+".") {
			t.Fatalf("install bundled all output missing %s:\n%s", name, got)
		}
		if _, err := plugin.LoadBundle(filepath.Join(root, name), version.Version); err != nil {
			t.Fatalf("load installed bundled %s: %v", name, err)
		}
	}
	if err := installBundledPlugins(failingWriter{}, t.TempDir(), nil, true, "en-US"); err == nil {
		t.Fatal("installBundledPlugins --all returned nil error for writer failure")
	}
	if err := installBundledPlugins(failingWriter{}, t.TempDir(), []string{"ecs"}, false, "en-US"); err == nil {
		t.Fatal("installBundledPlugins single returned nil error for writer failure")
	}
}

func TestInstallBundledPluginsAllReportsBundleAndInstallErrors(t *testing.T) {
	restoreVersion := patchVersion("0.2.0-dev")
	defer restoreVersion()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	tempRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempRoot, "plugins", "broken"), 0o755); err != nil {
		t.Fatalf("create broken bundled plugin: %v", err)
	}
	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	if err := installBundledPlugins(io.Discard, t.TempDir(), nil, true, "en-US"); err == nil {
		t.Fatal("installBundledPlugins --all returned nil error for malformed bundled plugin")
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("restore cwd: %v", err)
	}

	rootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, rootFile, "not a directory")
	if err := installBundledPlugins(io.Discard, rootFile, nil, true, "en-US"); err == nil {
		t.Fatal("installBundledPlugins --all returned nil error for file plugin root")
	}
}

func TestPluginBundledUpdateErrorPaths(t *testing.T) {
	restore := patchVersion("0.2.0-dev")
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

	root = t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "ecs"), "ecs", "0.1.0")
	if err := reinstallBundledPlugins(failingWriter{}, root, []string{"ecs"}, false, "en-US"); err == nil {
		t.Fatal("reinstallBundledPlugins returned nil error for writer failure")
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
	mustWrite(t, filepath.Join(registryRoot, "index.json"), `{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"unused"}]}`)
	if err := listPluginUpdates(failingWriter{}, root, registryRoot, "", nil, "", "en-US"); err == nil {
		t.Fatal("listPluginUpdates returned nil error for writer failure")
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

	sum := sha256.Sum256([]byte("artifact"))
	path, cleanup, err := prepareRegistryArtifact("https://registry.example.test/root", registry.Artifact{Name: "ecs", URL: "ecs.tar.gz", SHA256: hex.EncodeToString(sum[:])}, roundTripFunc(func(*http.Request) (*http.Response, error) {
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
