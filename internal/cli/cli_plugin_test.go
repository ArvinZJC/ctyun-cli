/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

func TestPluginInstallAndList(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()

	if _, err := installPluginSource(bundleDir, pluginRoot); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var listOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "plugin", "list"},
		Stdout:     &listOut,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if !strings.Contains(listOut.String(), "ecs") {
		t.Fatalf("plugin list output = %q, want ecs", listOut.String())
	}
	for _, want := range []string{"Name", "Plugin", "Product", "Version", "Channel", "Quality", "Commands", "Operations", "Elastic Cloud Server", "ecs", "0.1.0", "stable", "reviewed"} {
		if !strings.Contains(listOut.String(), want) {
			t.Fatalf("plugin list output missing %q:\n%s", want, listOut.String())
		}
	}
	for _, unwanted := range []string{"Source", "bundled", "region"} {
		if strings.Contains(listOut.String(), unwanted) {
			t.Fatalf("plugin list output contains %q:\n%s", unwanted, listOut.String())
		}
	}
}

func TestPluginListShowsOnlyInstalledPlugins(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("plugin list returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Name", "Plugin", "Product", "Version", "Channel", "Quality", "Commands", "Operations"} {
		if !strings.Contains(got, want) {
			t.Fatalf("plugin list output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"ecs", "region", "bundled", "Source"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("plugin list output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestPluginListFollowsOutputOptions(t *testing.T) {
	pluginRoot := t.TempDir()
	if _, err := plugin.InstallVerifiedLocalBundle(testBundleDir(t), pluginRoot, version.Version); err != nil {
		t.Fatalf("install ecs bundle: %v", err)
	}
	vpcDir := filepath.Join(t.TempDir(), "vpc")
	writeVPCBundle(t, vpcDir)
	if _, err := plugin.InstallVerifiedLocalBundle(vpcDir, pluginRoot, version.Version); err != nil {
		t.Fatalf("install vpc bundle: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "--table", "plain", "--cols", "plugin,commands,operations", "--filter", "plugin=vpc", "--sort", "-plugin", "--no-header", "plugin", "list"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin list returned error: %v", err)
	}
	got := stdout.String()
	if strings.Contains(got, "Plugin") || strings.Contains(got, "Name") || strings.Contains(got, "Channel") {
		t.Fatalf("plugin list ignored header/column options:\n%s", got)
	}
	if !strings.Contains(got, "vpc") || !strings.Contains(got, "1") {
		t.Fatalf("plugin list output missing selected installed plugin counts:\n%s", got)
	}
	if strings.Contains(got, "ecs") {
		t.Fatalf("plugin list output was not filtered by plugin:\n%s", got)
	}

	stdout.Reset()
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "--output", "json", "--filter", "plugin=vpc", "plugin", "list"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin list json returned error: %v", err)
	}
	got = stdout.String()
	for _, want := range []string{`"name": "vpc"`, `"plugin": "vpc"`, `"commands": "1"`, `"operations": "1"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("plugin list json output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"plugin": "ecs"`) {
		t.Fatalf("plugin list json output was not filtered by plugin:\n%s", got)
	}
}

func TestPluginInstallRejectsLocalPath(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()

	err := Run(Config{
		Args:       []string{"plugin", "install", bundleDir},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("plugin install accepted a local path")
	}
	if !strings.Contains(err.Error(), "unsupported plugin source") && !strings.Contains(err.Error(), "use --bundled") {
		t.Fatalf("error = %v, want hosted source or bundled guidance", err)
	}
	if _, statErr := os.Stat(filepath.Join(pluginRoot, "ecs")); !os.IsNotExist(statErr) {
		t.Fatalf("local path plugin was copied, stat err: %v", statErr)
	}
}

func TestInstallPluginSourceRejectsInvalidBundleBeforeCopy(t *testing.T) {
	bundleDir := filepath.Join(t.TempDir(), "vpc")
	writeVPCBundle(t, bundleDir)
	mustWrite(t, filepath.Join(bundleDir, "tables.json"), `{"tables": {}}`)
	pluginRoot := t.TempDir()

	_, err := installPluginSource(bundleDir, pluginRoot)
	if err == nil {
		t.Fatal("installPluginSource returned nil error for invalid bundle")
	}
	if !strings.Contains(err.Error(), "missing table") {
		t.Fatalf("error = %v, want missing table validation", err)
	}
	if _, statErr := os.Stat(filepath.Join(pluginRoot, "vpc")); !os.IsNotExist(statErr) {
		t.Fatalf("invalid plugin was copied, stat err: %v", statErr)
	}
}

func TestPluginRemove(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()

	if _, err := installPluginSource(bundleDir, pluginRoot); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "remove", "ecs"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("remove returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Removed ecs.") {
		t.Fatalf("remove output = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(pluginRoot, "ecs")); !os.IsNotExist(err) {
		t.Fatalf("installed plugin dir still exists or unexpected stat error: %v", err)
	}
}

func TestPluginRemoveRejectsUnsafeName(t *testing.T) {
	parent := t.TempDir()
	pluginRoot := filepath.Join(parent, "plugins")
	outside := filepath.Join(parent, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("create outside dir: %v", err)
	}

	err := Run(Config{
		Args:       []string{"plugin", "remove", "../outside"},
		Stdout:     io.Discard,
		PluginRoot: pluginRoot,
	})
	if err == nil {
		t.Fatal("plugin remove returned nil error for unsafe name")
	}
	if !strings.Contains(err.Error(), "invalid plugin name") {
		t.Fatalf("error = %v, want invalid plugin name", err)
	}
	if _, statErr := os.Stat(outside); statErr != nil {
		t.Fatalf("outside directory was touched: %v", statErr)
	}
}

func TestPluginLintValidatesBundle(t *testing.T) {
	bundleDir := testBundleDir(t)

	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"plugin", "lint", bundleDir},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("plugin lint returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Valid plugin ecs") {
		t.Fatalf("lint output = %q, want Valid plugin ecs", stdout.String())
	}
}

func TestPluginLintIsDevOnly(t *testing.T) {
	bundleDir := testBundleDir(t)
	restoreVersion := patchVersion("0.1.0")
	t.Cleanup(restoreVersion)

	if err := Run(Config{Args: []string{"plugin", "lint", bundleDir}, Stdout: io.Discard}); err == nil {
		t.Fatal("released build accepted plugin lint")
	}
}

func TestPluginLintRejectsInvalidBundle(t *testing.T) {
	bundleDir := testBundleDir(t)
	mustWrite(t, filepath.Join(bundleDir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "nightly",
  "quality": "reviewed",
  "requires": {"ctyun": ">=0.1.0 <1.0.0"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "docs_version": "81"}
}`)

	err := Run(Config{
		Args:   []string{"plugin", "lint", bundleDir},
		Stdout: io.Discard,
	})
	if err == nil {
		t.Fatal("plugin lint returned nil error for invalid bundle")
	}
	if !strings.Contains(err.Error(), "unsupported channel") {
		t.Fatalf("error = %v, want unsupported channel validation", err)
	}
}

func TestPluginListUpdatesUsesRegistryIndex(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()
	index := []byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0.tar.gz"}
  ]
}`)
	publicKey, transport := hostedPluginRegistry(t, index, nil)

	if _, err := installPluginSource(bundleDir, pluginRoot); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "list", "--updates", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("list --updates returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Update available for ecs: 0.1.0 -> 0.2.0.") {
		t.Fatalf("updates output = %q", stdout.String())
	}
}

func TestPluginListUpdatesUsesSelectedChannel(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()
	index := []byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-stable.tar.gz"},
    {"name": "ecs", "version": "0.3.0", "channel": "beta", "quality": "generated", "url": "ecs-beta.tar.gz"}
  ]
}`)
	publicKey, transport := hostedPluginRegistry(t, index, nil)

	if _, err := installPluginSource(bundleDir, pluginRoot); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "list", "--updates", "--source", "github", "--channel", "beta"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("list --updates returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Update available for ecs: 0.1.0 -> 0.3.0.") {
		t.Fatalf("updates output = %q", stdout.String())
	}
}

func TestPluginSearchUsesRegistryStorefront(t *testing.T) {
	index := []byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "generated", "url": "ecs-generated.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-reviewed.tar.gz"},
    {"name": "vpc", "version": "0.1.0", "channel": "stable", "quality": "curated", "url": "vpc.tar.gz"}
  ]
}`)
	publicKey, transport := hostedPluginRegistry(t, index, nil)

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "search", "--source", "github", "ec"},
		Stdout:        &stdout,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("plugin search returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "ecs") || !strings.Contains(got, "0.2.0") {
		t.Fatalf("search output missing reviewed ecs:\n%s", got)
	}
	if strings.Contains(got, "generated") || strings.Contains(got, "vpc") {
		t.Fatalf("search output exposed generated or unrelated plugins:\n%s", got)
	}
}

func TestPluginSourceCanComeFromEnv(t *testing.T) {
	bundleDir := testBundleDir(t)
	pluginRoot := t.TempDir()
	index := []byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs-0.2.0.tar.gz"}
  ]
}`)
	publicKey, transport := hostedPluginRegistry(t, index, nil)

	if _, err := installPluginSource(bundleDir, pluginRoot); err != nil {
		t.Fatalf("install old bundle: %v", err)
	}

	var envOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list", "--updates"},
		Stdout:     &envOut,
		PluginRoot: pluginRoot,
		Env: func(key string) string {
			switch key {
			case "CTYUN_PLUGIN_SOURCE":
				return "github"
			case "CTYUN_RELEASE_PUBLIC_KEY":
				return publicKey
			}
			return ""
		},
		HTTPTransport: transport,
	}); err != nil {
		t.Fatalf("list updates from env source: %v", err)
	}
	if !strings.Contains(envOut.String(), "Update available for ecs: 0.1.0 -> 0.2.0.") {
		t.Fatalf("env registry updates output = %q", envOut.String())
	}
}

func TestPluginInstallByNameFromRegistry(t *testing.T) {
	pluginRoot := t.TempDir()
	artifactName, artifactBytes, checksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + artifactName + `","sha256":"` + checksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{artifactName: artifactBytes})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("install from registry returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Installed ecs.") {
		t.Fatalf("install output = %q", stdout.String())
	}

	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), "0.1.0")
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.2.0" {
		t.Fatalf("installed version = %q, want 0.2.0", installed.Manifest.Version)
	}
}

func TestPluginInstallByNameFromRegistryVerifiesChecksum(t *testing.T) {
	pluginRoot := t.TempDir()
	artifactName, artifactBytes, checksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + artifactName + `","sha256":"` + checksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{artifactName: artifactBytes})

	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github"},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("install with checksum returned error: %v", err)
	}

	badIndex := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + artifactName + `","sha256":"bad"}]}`)
	publicKey, transport = hostedPluginRegistry(t, badIndex, map[string][]byte{artifactName: artifactBytes})
	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github"},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err == nil {
		t.Fatal("install with bad checksum returned nil error")
	}
}

func TestPluginInstallByNameFromHTTPRegistry(t *testing.T) {
	pluginRoot := t.TempDir()
	artifactName, artifactBytes, checksum := hostedPluginArtifact(t, "ecs", "0.3.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"https://artifacts.example.test/` + artifactName + `","sha256":"` + checksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{artifactName: artifactBytes})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("install from HTTP registry returned error: %v", err)
	}

	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), "0.1.0")
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.3.0" {
		t.Fatalf("installed version = %q, want 0.3.0", installed.Manifest.Version)
	}
}

func TestPluginInstallFromLocalRegistryDownloadsHTTPArtifact(t *testing.T) {
	pluginRoot := t.TempDir()
	artifactName, artifactBytes, checksum := hostedPluginArtifact(t, "ecs", "0.3.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"https://artifacts.example.test/` + artifactName + `","sha256":"` + checksum + `"}]}`)
	publicKey, baseTransport := hostedPluginRegistry(t, index, map[string][]byte{artifactName: artifactBytes})
	downloaded := false
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "artifacts.example.test" && filepath.Base(req.URL.Path) == artifactName {
			downloaded = true
		}
		return baseTransport.RoundTrip(req)
	})

	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github"},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("install from hosted registry HTTP artifact returned error: %v", err)
	}
	if !downloaded {
		t.Fatal("HTTP artifact was not downloaded")
	}
	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), "0.1.0")
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.3.0" {
		t.Fatalf("installed version = %q, want 0.3.0", installed.Manifest.Version)
	}
}

func TestPluginInstallFromHTTPRegistryRequiresChecksum(t *testing.T) {
	pluginRoot := t.TempDir()
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0.tar.gz"}]}`)
	publicKey, baseTransport := hostedPluginRegistry(t, index, map[string][]byte{"ecs-0.3.0.tar.gz": []byte("archive")})
	downloaded := false
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if filepath.Base(req.URL.Path) == "ecs-0.3.0.tar.gz" {
			downloaded = true
		}
		return baseTransport.RoundTrip(req)
	})

	err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github"},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	})
	if err == nil {
		t.Fatal("install from HTTP registry without checksum returned nil error")
	}
	if !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("error = %v, want sha256 requirement", err)
	}
	if downloaded {
		t.Fatal("artifact was downloaded before checksum requirement was enforced")
	}
}

func TestPluginHTTPRegistryRequiresSignedIndex(t *testing.T) {
	pluginRoot := t.TempDir()
	downloaded := false
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch filepath.Base(req.URL.Path) {
		case "index.json":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0.tar.gz","sha256":"abc"}]}`)),
			}, nil
		case "ecs-0.3.0.tar.gz":
			downloaded = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("archive")),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}
	})

	err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "github"},
		Stdout:        io.Discard,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
	})
	if err == nil {
		t.Fatal("install from unsigned HTTP registry returned nil error")
	}
	if !strings.Contains(err.Error(), "public key") && !strings.Contains(err.Error(), "signature") {
		t.Fatalf("error = %v, want signed registry requirement", err)
	}
	if downloaded {
		t.Fatal("artifact was downloaded before signed index verification")
	}
}

func TestPluginInstallFromHostedSourceFallsBackToGitee(t *testing.T) {
	t.Cleanup(patchVersion("0.1.0"))
	pluginRoot := t.TempDir()
	bundleDir := filepath.Join(t.TempDir(), "ecs-0.3.0")
	writeVersionedBundle(t, bundleDir, "ecs", "0.3.0")
	archivePath := filepath.Join(t.TempDir(), "ecs-0.3.0.tar.gz")
	writeTarGz(t, archivePath, bundleDir)
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	sum := sha256.Sum256(archiveBytes)
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.3.0","channel":"stable","quality":"reviewed","url":"ecs-0.3.0.tar.gz","sha256":"` + hex.EncodeToString(sum[:]) + `"}]}`)
	publicKey, signature := signedRegistryIndex(t, index)
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "github.com" {
			return httpStringResponse(http.StatusNotFound, "not found"), nil
		}
		switch req.URL.Path {
		case "/ArvinZJC/ctyun-cli/releases/download/ctyun-plugins/index.json":
			return httpStringResponse(http.StatusOK, string(index)), nil
		case "/ArvinZJC/ctyun-cli/releases/download/ctyun-plugins/index.sig":
			return httpStringResponse(http.StatusOK, signature), nil
		case "/ArvinZJC/ctyun-cli/releases/download/ctyun-plugins/ecs-0.3.0.tar.gz":
			return httpStringResponse(http.StatusOK, string(archiveBytes)), nil
		default:
			return httpStringResponse(http.StatusNotFound, "not found"), nil
		}
	})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "--source", "auto"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	}); err != nil {
		t.Fatalf("install from hosted source returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Installed ecs.") {
		t.Fatalf("stdout = %q, want install summary", stdout.String())
	}
	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), "0.1.0")
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.3.0" {
		t.Fatalf("installed version = %q, want 0.3.0", installed.Manifest.Version)
	}
}

func TestPluginBundledInstallAndUpdateAreDevOnly(t *testing.T) {
	repoRoot := t.TempDir()
	bundledRoot := filepath.Join(repoRoot, "plugins")
	if err := os.Mkdir(bundledRoot, 0o755); err != nil {
		t.Fatalf("create bundled root: %v", err)
	}
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")
	pluginRoot := t.TempDir()
	originalCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(repoRoot, "internal", "cli", "cli.go"), 1, true
	}

	restoreDevVersion := patchVersion("0.1.0-dev")
	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"plugin", "install", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled install returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Installed ecs.") {
		t.Fatalf("bundled install output = %q", stdout.String())
	}

	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.3.0")
	stdout.Reset()
	if err := Run(Config{Args: []string{"plugin", "update", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled update returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.2.0 -> 0.3.0.") {
		t.Fatalf("bundled update output = %q", stdout.String())
	}

	restoreDevVersion()
	t.Cleanup(patchVersion("0.1.0"))
	if err := Run(Config{Args: []string{"plugin", "install", "ecs", "--bundled"}, Stdout: io.Discard, PluginRoot: t.TempDir()}); err == nil {
		t.Fatal("released build accepted --bundled plugin install")
	}
}

func TestPluginBundledStatusMessagesUseLanguage(t *testing.T) {
	repoRoot := t.TempDir()
	bundledRoot := filepath.Join(repoRoot, "plugins")
	if err := os.Mkdir(bundledRoot, 0o755); err != nil {
		t.Fatalf("create bundled root: %v", err)
	}
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")
	pluginRoot := t.TempDir()
	originalCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(repoRoot, "internal", "cli", "cli.go"), 1, true
	}
	t.Cleanup(patchVersion("0.1.0-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"--lang", "zh-CN", "plugin", "install", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled install returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "已安装 ecs。" {
		t.Fatalf("bundled install output = %q", got)
	}

	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.3.0")
	stdout.Reset()
	if err := Run(Config{Args: []string{"--lang", "zh-CN", "plugin", "update", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled update returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "已更新 ecs：0.2.0 -> 0.3.0。" {
		t.Fatalf("bundled update output = %q", got)
	}
}

func TestPluginBundledUpdateAll(t *testing.T) {
	repoRoot := t.TempDir()
	bundledRoot := filepath.Join(repoRoot, "plugins")
	if err := os.Mkdir(bundledRoot, 0o755); err != nil {
		t.Fatalf("create bundled root: %v", err)
	}
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.3.0")
	writeVersionedBundle(t, filepath.Join(bundledRoot, "vpc"), "vpc", "0.2.0")
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "vpc"), "vpc", "0.2.0")
	originalCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(repoRoot, "internal", "cli", "cli.go"), 1, true
	}
	t.Cleanup(patchVersion("0.1.0-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"plugin", "update", "--all", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled update --all returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.3.0.") || !strings.Contains(stdout.String(), "vpc is already up to date.") {
		t.Fatalf("bundled update --all output = %q", stdout.String())
	}
}

func TestPluginUpdateRejectsCustomLocalSource(t *testing.T) {
	t.Cleanup(patchVersion("0.1.0"))
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	if err := Run(Config{Args: []string{"plugin", "update", "ecs", "--source", t.TempDir()}, Stdout: io.Discard, PluginRoot: pluginRoot}); err == nil {
		t.Fatal("plugin update accepted a local source")
	}
	if err := Run(Config{Args: []string{"plugin", "update", "ecs", "--source", "https://registry.example.test"}, Stdout: io.Discard, PluginRoot: pluginRoot}); err == nil {
		t.Fatal("plugin update accepted a custom URL source")
	}
}

func TestPluginUpdateAllFromRegistry(t *testing.T) {
	oldBundle := testBundleDir(t)
	pluginRoot := t.TempDir()
	artifactName, artifactBytes, checksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + artifactName + `","sha256":"` + checksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{artifactName: artifactBytes})

	if _, err := installPluginSource(oldBundle, pluginRoot); err != nil {
		t.Fatalf("install old bundle: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "update", "--all", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("update --all returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.2.0.") {
		t.Fatalf("update output = %q", stdout.String())
	}
}

func TestPluginUpdateOneFromRegistry(t *testing.T) {
	oldBundle := testBundleDir(t)
	pluginRoot := t.TempDir()
	ecsArtifact, ecsBytes, ecsChecksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	vpcArtifact, vpcBytes, vpcChecksum := hostedPluginArtifact(t, "vpc", "0.2.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + ecsArtifact + `","sha256":"` + ecsChecksum + `"},{"name":"vpc","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + vpcArtifact + `","sha256":"` + vpcChecksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{ecsArtifact: ecsBytes, vpcArtifact: vpcBytes})

	if _, err := installPluginSource(oldBundle, pluginRoot); err != nil {
		t.Fatalf("install old bundle: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "update", "ecs", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("update ecs returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.2.0.") {
		t.Fatalf("update output = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(pluginRoot, "vpc")); !os.IsNotExist(err) {
		t.Fatalf("update one installed unrelated plugin or unexpected stat error: %v", err)
	}
}

func TestPluginUpdateUsesSelectedChannel(t *testing.T) {
	oldBundle := testBundleDir(t)
	pluginRoot := t.TempDir()
	stableArtifact, stableBytes, stableChecksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	betaArtifact, betaBytes, betaChecksum := hostedPluginArtifact(t, "ecs", "0.3.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + stableArtifact + `","sha256":"` + stableChecksum + `"},{"name":"ecs","version":"0.3.0","channel":"beta","quality":"generated","url":"` + betaArtifact + `","sha256":"` + betaChecksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{stableArtifact: stableBytes, betaArtifact: betaBytes})

	if _, err := installPluginSource(oldBundle, pluginRoot); err != nil {
		t.Fatalf("install old bundle: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "update", "ecs", "--source", "github", "--channel", "beta"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("update ecs returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.3.0.") {
		t.Fatalf("update output = %q", stdout.String())
	}
	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), version.Version)
	if err != nil {
		t.Fatalf("load installed bundle: %v", err)
	}
	if installed.Manifest.Version != "0.3.0" {
		t.Fatalf("installed version = %q, want 0.3.0", installed.Manifest.Version)
	}
}

func TestPluginAndPluginsUpgradeAliasesUpdatePlugins(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "plugin_upgrade_all", args: []string{"plugin", "upgrade", "--all"}},
		{name: "plugins_update_all", args: []string{"plugins", "update", "--all"}},
		{name: "plugins_upgrade_all", args: []string{"plugins", "upgrade", "--all"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldBundle := testBundleDir(t)
			pluginRoot := t.TempDir()
			artifactName, artifactBytes, checksum := hostedPluginArtifact(t, "ecs", "0.2.0")
			index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + artifactName + `","sha256":"` + checksum + `"}]}`)
			publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{artifactName: artifactBytes})

			if _, err := installPluginSource(oldBundle, pluginRoot); err != nil {
				t.Fatalf("install old bundle: %v", err)
			}

			var stdout bytes.Buffer
			args := append(append([]string{}, tc.args...), "--source", "github")
			if err := Run(Config{
				Args:          args,
				Stdout:        &stdout,
				PluginRoot:    pluginRoot,
				HTTPTransport: transport,
				Env:           hostedPluginEnv(publicKey),
			}); err != nil {
				t.Fatalf("%s returned error: %v", strings.Join(args, " "), err)
			}
			if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.2.0.") {
				t.Fatalf("upgrade alias output = %q", stdout.String())
			}
		})
	}
}

func TestHelpCommandUsesPluginMetadata(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"help", "ecs", "instance", "list"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"List cloud servers", "Product: Elastic Cloud Server", "ctyun ecs instance list", "https://"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
	if first := firstNonEmptyLine(got); first != "List cloud servers." {
		t.Fatalf("plugin command help first line = %q", first)
	}
	for _, unwanted := range []string{"ecs.instance.list", "Description:", "ctyun ecs server ls"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestHelpPluginPrefixListsPluginCommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "ecs"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Elastic Cloud Server",
		"Commands:",
		"ecs instance list                List cloud servers",
		"ecs instance show {instance_id}  Show cloud server details",
		"ctyun help ecs instance list",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plugin help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "region list") {
		t.Fatalf("plugin help output exposed unrelated commands:\n%s", got)
	}
	if strings.Contains(got, "Available Commands:") {
		t.Fatalf("plugin help output contains old command heading:\n%s", got)
	}
	if strings.Contains(got, "ecs server ls") {
		t.Fatalf("plugin help output exposed unsupported alias:\n%s", got)
	}
}

func TestHelpNestedPrefixListsMatchingPluginCommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "ecs", "instance"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Commands:",
		"ecs instance list",
		"ecs instance show {instance_id}",
		"ecs instance start {instance_id}",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("nested plugin help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Available Commands:") {
		t.Fatalf("nested plugin help output contains old command heading:\n%s", got)
	}
}

func TestHelpUsesSentenceCaseForEnglishDescriptions(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "ecs", "instance", "list"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"List cloud servers", "Filter by instance name", "Render output as a table or raw JSON", "Show help for the command"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing sentence-case text %q:\n%s", want, got)
		}
	}
}

func TestHelpCommandUsesPluginI18N(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "zh-CN", "help", "ecs", "instance", "list"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"弹性云主机", "列出云主机", "命令选项:", "全局选项:", "--name <value>  按云主机名称过滤", "[匹配 ^[A-Za-z0-9._-]+$]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Filter by instance name") || strings.Contains(got, "matches ^") {
		t.Fatalf("localized help output still contains English option description:\n%s", got)
	}
}
