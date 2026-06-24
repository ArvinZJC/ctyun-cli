/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

func TestPluginListAvailableShowsInstallStatusAndOutputOptions(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	index := []byte(`{
  "plugins": [
    {"name": "ecs", "product": "ecs", "display_name": "Elastic Cloud Server", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs.tar.gz"},
    {"name": "vpc", "product": "vpc", "display_name": "Virtual Private Cloud", "version": "0.1.0", "channel": "stable", "quality": "curated", "url": "vpc.tar.gz"}
  ]
}`)
	publicKey, transport := hostedPluginRegistry(t, index, nil)

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"--lang", "en-US", "--table", "plain", "--cols", "plugin,status,quality,installed_version,version", "--filter", "status=available", "--no-header", "plugin", "list", "--available", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("plugin list --available returned error: %v", err)
	}
	got := stdout.String()
	if strings.Contains(got, "Plugin") || strings.Contains(got, "Status") {
		t.Fatalf("available list ignored output controls:\n%s", got)
	}
	if !strings.Contains(got, "vpc") || !strings.Contains(got, "available") || !strings.Contains(got, "curated") || !strings.Contains(got, "0.1.0") {
		t.Fatalf("available list missing vpc status:\n%s", got)
	}
	if strings.Contains(got, "maintained") {
		t.Fatalf("available list changed English quality label:\n%s", got)
	}
	if strings.Contains(got, "ecs") || strings.Contains(got, "outdated") {
		t.Fatalf("available list was not filtered by status:\n%s", got)
	}
}

func TestPluginListAvailableLocalizesQualityAndStatusAfterFiltering(t *testing.T) {
	index := []byte(`{
  "plugins": [
    {"name": "ecs", "product": "ecs", "display_name": "Elastic Cloud Server", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs.tar.gz"},
    {"name": "vpc", "product": "vpc", "display_name": "Virtual Private Cloud", "version": "0.1.0", "channel": "stable", "quality": "curated", "url": "vpc.tar.gz"}
  ]
}`)
	publicKey, transport := hostedPluginRegistry(t, index, nil)

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"--lang", "zh-CN", "--table", "plain", "--cols", "插件,质量,状态", "--filter", "状态=可安装", "--sort", "质量", "--no-header", "plugin", "list", "--available", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    t.TempDir(),
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("plugin list --available returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"ecs", "vpc", "已复核", "持续维护", "可安装"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized available list missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"reviewed", "curated", "available", "not-installed"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("localized available list leaked raw value %q:\n%s", unwanted, got)
		}
	}
}

func TestPluginListLocalizesQualityAfterFiltering(t *testing.T) {
	pluginRoot := t.TempDir()
	if _, err := plugin.InstallVerifiedLocalBundle(testBundleDir(t), pluginRoot, version.Version); err != nil {
		t.Fatalf("install ecs bundle: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "--table", "plain", "--cols", "插件,质量", "--filter", "质量=工具生成", "--no-header", "plugin", "list"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin list returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "ecs") || !strings.Contains(got, "工具生成") {
		t.Fatalf("localized plugin list missing quality:\n%s", got)
	}
	if strings.Contains(got, "generated") {
		t.Fatalf("localized plugin list leaked raw quality:\n%s", got)
	}
}

func TestPluginListAndSearchUseBundledRegistryInDevelopmentBuild(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.0.1")

	var listOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--table", "plain", "--cols", "plugin,status,installed_version,version", "--no-header", "plugin", "list", "--available", "--bundled"},
		Stdout:     &listOut,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin list --available --bundled returned error: %v", err)
	}
	gotList := listOut.String()
	if !strings.Contains(gotList, "ecs") || !strings.Contains(gotList, "outdated") || !strings.Contains(gotList, "0.0.1") {
		t.Fatalf("bundled available list missing ecs status:\n%s", gotList)
	}
	if !strings.Contains(gotList, "region") || !strings.Contains(gotList, "available") {
		t.Fatalf("bundled available list missing region status:\n%s", gotList)
	}

	var searchOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--output", "json", "plugin", "search", "reg", "--bundled"},
		Stdout:     &searchOut,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin search --bundled returned error: %v", err)
	}
	gotSearch := searchOut.String()
	if !strings.Contains(gotSearch, `"plugin": "region"`) || strings.Contains(gotSearch, `"plugin": "ecs"`) {
		t.Fatalf("bundled search output mismatch:\n%s", gotSearch)
	}

	var updatesOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "list", "--updates", "--bundled"},
		Stdout:     &updatesOut,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin list --updates --bundled returned error: %v", err)
	}
	if got := updatesOut.String(); !strings.Contains(got, "Update available for ecs: 0.0.1 -> 0.1.0-alpha.1.") {
		t.Fatalf("bundled updates output mismatch:\n%s", got)
	}

	var alphaOut bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--output", "json", "plugin", "search", "reg", "--bundled", "--channel", "alpha"},
		Stdout:     &alphaOut,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin search --bundled --channel alpha returned error: %v", err)
	}
	if got := alphaOut.String(); !strings.Contains(got, `"plugin": "region"`) {
		t.Fatalf("bundled search with explicit channel output mismatch:\n%s", got)
	}
}

func TestPluginBundledDiscoveryValidationErrors(t *testing.T) {
	rootFile := filepath.Join(t.TempDir(), "plugins")
	if err := os.WriteFile(rootFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write plugin root file: %v", err)
	}
	if err := Run(Config{Args: []string{"plugin", "list", "--available", "--bundled"}, Stdout: io.Discard, PluginRoot: rootFile}); err == nil {
		t.Fatal("plugin list --available --bundled returned nil error for file root")
	}
	if err := Run(Config{Args: []string{"plugin", "search", "ecs", "--bundled"}, Stdout: io.Discard, PluginRoot: rootFile}); err == nil {
		t.Fatal("plugin search --bundled returned nil error for file root")
	}

	restoreVersion := patchVersion("0.1.0")
	defer restoreVersion()
	if err := Run(Config{Args: []string{"plugin", "search", "ecs", "--bundled"}, Stdout: io.Discard, PluginRoot: t.TempDir()}); err == nil {
		t.Fatal("released build accepted bundled plugin search")
	}
	if err := Run(Config{Args: []string{"plugin", "list", "--available", "--bundled"}, Stdout: io.Discard, PluginRoot: t.TempDir()}); err == nil {
		t.Fatal("released build accepted bundled plugin list")
	}
	if err := Run(Config{Args: []string{"plugin", "list", "--updates", "--bundled"}, Stdout: io.Discard, PluginRoot: t.TempDir()}); err == nil {
		t.Fatal("released build accepted bundled plugin update listing")
	}
	restoreVersion()

	repoRoot := t.TempDir()
	bundledRoot := filepath.Join(repoRoot, "plugins")
	if err := os.MkdirAll(filepath.Join(bundledRoot, "ecs"), 0o755); err != nil {
		t.Fatalf("create malformed bundled plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundledRoot, "ecs", "plugin.json"), []byte(`{"name":"ecs"}`), 0o644); err != nil {
		t.Fatalf("write malformed bundled manifest: %v", err)
	}
	originalCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(repoRoot, "internal", "cli", "cli.go"), 1, true
	}
	if err := Run(Config{Args: []string{"plugin", "search", "ecs", "--bundled"}, Stdout: io.Discard, PluginRoot: t.TempDir()}); err == nil {
		t.Fatal("plugin search --bundled returned nil error for malformed bundled plugin")
	}

	if _, err := parsePluginSearchOptions([]string{"ecs", "--bundled", "--source", "auto"}); err == nil {
		t.Fatal("parsePluginSearchOptions returned nil error for bundled/source conflict")
	}
	if _, err := parsePluginListOptions([]string{"--available", "--bundled", "--source", "auto"}); err == nil {
		t.Fatal("parsePluginListOptions returned nil error for bundled/source conflict")
	}
}

func TestPluginRemoveMultiple(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "vpc"), "vpc", "0.1.0")

	var stderr bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "remove", "ecs", "vpc"},
		Stdout:     io.Discard,
		Stderr:     &stderr,
		Stdin:      strings.NewReader("n\n"),
		PluginRoot: pluginRoot,
	}); err == nil {
		t.Fatal("declined remove multiple returned nil error")
	}
	if !strings.Contains(stderr.String(), "Continue? [y/N]:") {
		t.Fatalf("confirmation prompt missing from stderr:\n%s", stderr.String())
	}
	for _, name := range []string{"ecs", "vpc"} {
		if _, err := os.Stat(filepath.Join(pluginRoot, name)); err != nil {
			t.Fatalf("%s was removed after declined confirmation: %v", name, err)
		}
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "remove", "ecs", "vpc"},
		Stdout:     &stdout,
		Stdin:      strings.NewReader("yes\n"),
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("remove multiple returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Removed ecs.", "Removed vpc."} {
		if !strings.Contains(got, want) {
			t.Fatalf("remove output missing %q:\n%s", want, got)
		}
	}
	for _, name := range []string{"ecs", "vpc"} {
		if _, err := os.Stat(filepath.Join(pluginRoot, name)); !os.IsNotExist(err) {
			t.Fatalf("%s dir still exists or unexpected stat error: %v", name, err)
		}
	}
}

func TestPluginRemoveAllRequiresYes(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "vpc"), "vpc", "0.1.0")

	var stderr bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "remove", "--all"},
		Stdout:     io.Discard,
		Stderr:     &stderr,
		Stdin:      strings.NewReader("n\n"),
		PluginRoot: pluginRoot,
	}); err == nil {
		t.Fatal("declined plugin remove --all returned nil error")
	}
	if !strings.Contains(stderr.String(), "Continue? [y/N]:") {
		t.Fatalf("confirmation prompt missing from stderr:\n%s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(pluginRoot, "ecs")); err != nil {
		t.Fatalf("ecs was removed after declined confirmation: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"plugin", "remove", "--all", "--yes"},
		Stdout:     &stdout,
		PluginRoot: pluginRoot,
	}); err != nil {
		t.Fatalf("plugin remove --all --yes returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Removed ecs.", "Removed vpc."} {
		if !strings.Contains(got, want) {
			t.Fatalf("remove all output missing %q:\n%s", want, got)
		}
	}
}

func TestPluginSearchUsesFuzzyMatchingAndOutputOptions(t *testing.T) {
	index := []byte(`{
  "plugins": [
    {"name": "ecs", "product": "ecs", "display_name": "Elastic Cloud Server", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs.tar.gz"},
    {"name": "vpc", "product": "vpc", "display_name": "Virtual Private Cloud", "version": "0.1.0", "channel": "stable", "quality": "curated", "url": "vpc.tar.gz"}
  ]
}`)
	publicKey, transport := hostedPluginRegistry(t, index, nil)

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"--lang", "en-US", "--output", "json", "--filter", "plugin=ecs", "plugin", "search", "--source", "github", "elc"},
		Stdout:        &stdout,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("plugin search returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{`"name": "Elastic Cloud Server"`, `"plugin": "ecs"`, `"product": "ecs"`, `"version": "0.2.0"`, `"status": "available"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("search json output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"plugin": "vpc"`) {
		t.Fatalf("search output included unrelated plugin:\n%s", got)
	}
}

func TestPluginInstallMultipleFromRegistry(t *testing.T) {
	pluginRoot := t.TempDir()
	ecsArtifact, ecsBytes, ecsChecksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	vpcArtifact, vpcBytes, vpcChecksum := hostedPluginArtifact(t, "vpc", "0.1.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + ecsArtifact + `","sha256":"` + ecsChecksum + `"},{"name":"vpc","version":"0.1.0","channel":"stable","quality":"reviewed","url":"` + vpcArtifact + `","sha256":"` + vpcChecksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{ecsArtifact: ecsBytes, vpcArtifact: vpcBytes})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "install", "ecs", "vpc", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("install multiple returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Installed ecs.", "Installed vpc."} {
		if !strings.Contains(got, want) {
			t.Fatalf("install output missing %q:\n%s", want, got)
		}
	}
	for _, name := range []string{"ecs", "vpc"} {
		if _, err := plugin.LoadBundle(filepath.Join(pluginRoot, name), testCoreVersion()); err != nil {
			t.Fatalf("load installed %s bundle: %v", name, err)
		}
	}
}

func TestPluginInstallAllFromRegistry(t *testing.T) {
	pluginRoot := t.TempDir()
	ecsArtifact, ecsBytes, ecsChecksum := hostedPluginArtifact(t, "ecs", "0.2.0")
	vpcArtifact, vpcBytes, vpcChecksum := hostedPluginArtifact(t, "vpc", "0.1.0")
	index := []byte(`{"plugins":[{"name":"ecs","version":"0.2.0","channel":"stable","quality":"reviewed","url":"` + ecsArtifact + `","sha256":"` + ecsChecksum + `"},{"name":"vpc","version":"0.1.0","channel":"stable","quality":"curated","url":"` + vpcArtifact + `","sha256":"` + vpcChecksum + `"}]}`)
	publicKey, transport := hostedPluginRegistry(t, index, map[string][]byte{ecsArtifact: ecsBytes, vpcArtifact: vpcBytes})

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:          []string{"plugin", "install", "--all", "--source", "github"},
		Stdout:        &stdout,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           hostedPluginEnv(publicKey),
	}); err != nil {
		t.Fatalf("install --all returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Installed ecs.", "Installed vpc."} {
		if !strings.Contains(got, want) {
			t.Fatalf("install all output missing %q:\n%s", want, got)
		}
	}
}
