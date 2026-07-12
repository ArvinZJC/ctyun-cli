/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

func TestPluginInstallSkipsInstalledPluginBeforeRegistryRead(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0-beta.2")
	publicKey, _ := hostedPluginRegistry(t, []byte(`{"plugins":[]}`), nil)
	requested := false
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		requested = true
		return nil, errors.New("unexpected request")
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github", "--channel", "alpha"},
		Stdout:        &stdout,
		Stderr:        &bytes.Buffer{},
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	})
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}
	if requested {
		t.Fatal("install read registry for already-installed explicit target")
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin install complete: installed 0; already installed 1; failed 0." {
		t.Fatalf("install summary = %q", got)
	}
}

func TestPluginInstallAllSkipsInstalledPlugins(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	ecsArtifact, ecsBytes, ecsChecksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	vpcArtifact, vpcBytes, vpcChecksum := hostedPluginArtifact(t, "vpc", "0.1.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + ecsArtifact + `","sha256":"` + ecsChecksum + `"},{"name":"vpc","version":"0.1.0","channel":"stable","quality":"reviewed","url":"` + vpcArtifact + `","sha256":"` + vpcChecksum + `"}]}`)
	publicKey, baseTransport := hostedPluginRegistry(t, index, map[string][]byte{ecsArtifact: ecsBytes, vpcArtifact: vpcBytes})
	ecsDownloaded := false
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if strings.HasSuffix(request.URL.Path, "/"+ecsArtifact) {
			ecsDownloaded = true
		}
		return baseTransport.RoundTrip(request)
	})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "install", "--all", "--source", "github"},
		Stdout:        &stdout,
		Stderr:        &bytes.Buffer{},
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("install --all returned error: %v", err)
	}
	if ecsDownloaded {
		t.Fatal("install --all downloaded an already-installed plugin")
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin install complete: installed 1; already installed 1; failed 0." {
		t.Fatalf("install --all summary = %q", got)
	}
	installed, err := loadInstalledPluginForTest(pluginRoot, "ecs")
	if err != nil {
		t.Fatal(err)
	}
	if installed != "0.1.0" {
		t.Fatalf("installed ecs version = %q, want original 0.1.0", installed)
	}
}

func TestPluginBundledInstallSkipsInstalledPlugin(t *testing.T) {
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	t.Cleanup(patchVersion("0.3.1-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "install", "ecs", "--bundled"},
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("bundled install returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin install complete: installed 0; already installed 1; failed 0." {
		t.Fatalf("bundled install summary = %q", got)
	}
	installed, err := loadInstalledPluginForTest(pluginRoot, "ecs")
	if err != nil {
		t.Fatal(err)
	}
	if installed != "0.1.0" {
		t.Fatalf("installed ecs version = %q, want original 0.1.0", installed)
	}
}

func TestPluginReinstallRejectsMissingTargetBeforeRegistryRead(t *testing.T) {
	publicKey, _ := hostedPluginRegistry(t, []byte(`{"plugins":[]}`), nil)
	requested := false
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		requested = true
		return nil, errors.New("unexpected request")
	})
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:          []string{"plugin", "reinstall", "ecs", "--source", "github"},
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginRoot:    t.TempDir(),
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	})
	if err == nil {
		t.Fatal("reinstall missing plugin returned nil error")
	}
	requireDiagnosticKey(t, err, "error.operation_batch_failed")
	if requested {
		t.Fatal("reinstall read registry for an absent explicit target")
	}
	if got := stderr.String(); !strings.Contains(got, "ecs: plugin ecs is not installed.") || strings.Contains(got, "plugin.json") || strings.Contains(got, t.TempDir()) {
		t.Fatalf("missing reinstall stderr = %q", got)
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin reinstall complete: reinstalled 0; failed 1." {
		t.Fatalf("missing reinstall summary = %q", got)
	}
}

func TestPluginReinstallCanDowngradeSelectedChannel(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0-beta.2")
	artifact, artifactBytes, checksum := hostedPluginArtifact(t, "ecs", "0.1.0-alpha.1")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.1.0-alpha.1","channel":"alpha","quality":"generated","url":"` + artifact + `","sha256":"` + checksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{artifact: artifactBytes})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "reinstall", "ecs", "--source", "github", "--channel", "alpha"},
		Stdout:        &stdout,
		Stderr:        &bytes.Buffer{},
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("reinstall downgrade returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin reinstall complete: reinstalled 1; failed 0." {
		t.Fatalf("reinstall summary = %q", got)
	}
	installed, err := loadInstalledPluginForTest(pluginRoot, "ecs")
	if err != nil {
		t.Fatal(err)
	}
	if installed != "0.1.0-alpha.1" {
		t.Fatalf("reinstalled ecs version = %q, want alpha downgrade", installed)
	}
}

func TestPluginBundledReinstallRejectsMissingTarget(t *testing.T) {
	prepareBundledPluginRoots(t)
	t.Cleanup(patchVersion("0.3.1-dev"))
	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:       []string{"plugin", "reinstall", "ecs", "--bundled"},
		Stdout:     &stdout,
		Stderr:     &stderr,
		PluginRoot: t.TempDir(),
	})
	if err == nil {
		t.Fatal("bundled reinstall missing plugin returned nil error")
	}
	if !strings.Contains(stderr.String(), "plugin ecs is not installed") {
		t.Fatalf("bundled missing stderr = %q", stderr.String())
	}
}

func TestPluginUpdateAllChangesOnlyStrictlyNewerVersions(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "region"), "region", "0.3.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "vpc"), "vpc", "0.2.0")
	ecsArtifact, ecsBytes, ecsChecksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	regionArtifact, regionBytes, regionChecksum := hostedPluginArtifact(t, "region", "0.2.0")
	vpcArtifact, vpcBytes, vpcChecksum := hostedPluginArtifact(t, "vpc", "0.2.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + ecsArtifact + `","sha256":"` + ecsChecksum + `"},{"name":"region","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + regionArtifact + `","sha256":"` + regionChecksum + `"},{"name":"vpc","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + vpcArtifact + `","sha256":"` + vpcChecksum + `"}]}`)
	publicKey, baseTransport := hostedPluginRegistry(t, index, map[string][]byte{ecsArtifact: ecsBytes, regionArtifact: regionBytes, vpcArtifact: vpcBytes})
	downloads := make(map[string]bool)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		for _, artifact := range []string{ecsArtifact, regionArtifact, vpcArtifact} {
			if strings.HasSuffix(request.URL.Path, "/"+artifact) {
				downloads[artifact] = true
			}
		}
		return baseTransport.RoundTrip(request)
	})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "update", "--all", "--source", "github"},
		Stdout:        &stdout,
		Stderr:        &bytes.Buffer{},
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("update --all returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin update complete: updated 1; already current 2; failed 0." {
		t.Fatalf("update summary = %q", got)
	}
	if !downloads[ecsArtifact] || downloads[regionArtifact] || downloads[vpcArtifact] {
		t.Fatalf("artifact downloads = %v", downloads)
	}
}

func TestPluginRemoveContinuesAfterTargetFailure(t *testing.T) {
	pluginRoot := t.TempDir()
	for _, name := range []string{"ecs", "region", "vpc"} {
		writeVersionedBundle(t, filepath.Join(pluginRoot, name), name, "0.1.0")
	}
	originalRemoveAll := removeAll
	t.Cleanup(func() { removeAll = originalRemoveAll })
	removeAll = func(path string) error {
		if filepath.Base(path) == "region" {
			return errors.New("permission denied")
		}
		return originalRemoveAll(path)
	}

	var stdout, stderr bytes.Buffer
	err := Run(Config{
		Args:       []string{"--yes", "plugin", "remove", "ecs", "region", "vpc"},
		Stdout:     &stdout,
		Stderr:     &stderr,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("remove batch returned nil error")
	}
	requireDiagnosticKey(t, err, "error.operation_batch_failed")
	if got := strings.TrimSpace(stdout.String()); got != "Plugin removal complete: removed 2; failed 1." {
		t.Fatalf("remove summary = %q", got)
	}
	if !strings.Contains(stderr.String(), "region: permission denied.") {
		t.Fatalf("remove stderr = %q", stderr.String())
	}
	for _, name := range []string{"ecs", "vpc"} {
		if _, statErr := os.Stat(filepath.Join(pluginRoot, name)); !os.IsNotExist(statErr) {
			t.Fatalf("removed plugin %s still exists or returned unexpected error: %v", name, statErr)
		}
	}
	if _, statErr := loadInstalledPluginForTest(pluginRoot, "region"); statErr != nil {
		t.Fatalf("failed removal should preserve region: %v", statErr)
	}
}

func TestPluginOperationPrecheckAndBundleErrors(t *testing.T) {
	if _, _, err := loadInstalledPlugin(t.TempDir(), "../ecs"); err == nil {
		t.Fatal("loadInstalledPlugin accepted unsafe name")
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "ecs"), "not a directory")
	if _, _, err := loadInstalledPlugin(root, "ecs"); err == nil {
		t.Fatal("loadInstalledPlugin accepted file target")
	}

	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	if err := os.MkdirAll(filepath.Join(bundledRoot, "ecs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(bundledRoot, "ecs", "plugin.json"), `{`)
	t.Cleanup(patchVersion("0.3.1-dev"))
	if err := installBundledPlugins(io.Discard, pluginRoot, []string{"ecs"}, false, "en-US"); err == nil {
		t.Fatal("bundled install accepted invalid source bundle")
	}
}

func TestHostedPluginOperationSharedErrorPaths(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	missingRegistry := filepath.Join(t.TempDir(), "missing")
	if err := reinstallPluginsFromHostedSource(io.Discard, pluginRoot, distribution.Source{Name: "test", URL: missingRegistry}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("reinstall accepted missing registry")
	}
	badRegistry := t.TempDir()
	mustWrite(t, filepath.Join(badRegistry, "index.json"), `{`)
	if err := reinstallPluginsFromHostedSource(io.Discard, pluginRoot, distribution.Source{Name: "test", URL: badRegistry}, []string{"ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("reinstall accepted malformed registry")
	}
	emptyRegistry := t.TempDir()
	mustWrite(t, filepath.Join(emptyRegistry, "index.json"), `{"plugins":[]}`)
	if err := updateOnePlugin(io.Discard, pluginRoot, emptyRegistry, "ecs", "", nil, "", "en-US"); err == nil {
		t.Fatal("update accepted missing registry artifact")
	}
	if err := installPluginsFromHostedSource(io.Discard, t.TempDir(), distribution.Source{Name: "test", URL: emptyRegistry}, []string{"../ecs"}, false, "", nil, "", "en-US"); err == nil {
		t.Fatal("install accepted unsafe precheck target")
	}
	installRoot := t.TempDir()
	mustWrite(t, filepath.Join(installRoot, "ecs"), "not a directory")
	indexRegistry := t.TempDir()
	mustWrite(t, filepath.Join(indexRegistry, "index.json"), `{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"ecs"}]}`)
	if err := installPluginsFromHostedSource(io.Discard, installRoot, distribution.Source{Name: "test", URL: indexRegistry}, nil, true, "", nil, "", "en-US"); err == nil {
		t.Fatal("install --all accepted invalid installed target")
	}
}

func loadInstalledPluginForTest(root, name string) (string, error) {
	bundle, err := plugin.LoadBundle(filepath.Join(root, name), version.Version)
	if err != nil {
		return "", err
	}
	return bundle.Manifest.Version, nil
}
