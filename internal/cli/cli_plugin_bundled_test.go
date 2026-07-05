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

func TestPluginBundledInstallAndUpdateAreDevOnly(t *testing.T) {
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")

	restoreDevVersion := patchVersion("0.2.0-dev")
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
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")
	t.Cleanup(patchVersion("0.2.0-dev"))

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
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.3.0")
	writeVersionedBundle(t, filepath.Join(bundledRoot, "vpc"), "vpc", "0.2.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "vpc"), "vpc", "0.2.0")
	t.Cleanup(patchVersion("0.2.0-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"plugin", "update", "--all", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled update --all returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Updated ecs: 0.1.0 -> 0.3.0.") || !strings.Contains(stdout.String(), "vpc is already up to date.") {
		t.Fatalf("bundled update --all output = %q", stdout.String())
	}
}

func TestPluginBundledReinstallRefreshesSameVersion(t *testing.T) {
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.2.0")
	if err := os.MkdirAll(filepath.Join(pluginRoot, "ecs", "i18n"), 0o755); err != nil {
		t.Fatalf("create stale i18n dir: %v", err)
	}
	mustWrite(t, filepath.Join(pluginRoot, "ecs", "i18n", "en-US.json"), `{"name": "Stale ECS"}`)
	t.Cleanup(patchVersion("0.2.0-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"plugin", "reinstall", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled reinstall returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Reinstalled ecs.") {
		t.Fatalf("bundled reinstall output = %q", stdout.String())
	}
	installed, err := plugin.LoadBundle(filepath.Join(pluginRoot, "ecs"), version.Version)
	if err != nil {
		t.Fatalf("load reinstalled bundle: %v", err)
	}
	if got := localizedPluginText(installed, "en-US", "name", installed.Manifest.Name); got == "Stale ECS" {
		t.Fatalf("bundled reinstall kept stale metadata")
	}
}

// prepareBundledPluginRoots creates a fake repository root for bundled plugin
// tests and points runtimeCaller at it.
func prepareBundledPluginRoots(t *testing.T) (string, string) {
	t.Helper()

	repoRoot := t.TempDir()
	bundledRoot := filepath.Join(repoRoot, "plugins")
	if err := os.Mkdir(bundledRoot, 0o755); err != nil {
		t.Fatalf("create bundled root: %v", err)
	}
	pluginRoot := t.TempDir()
	originalCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(repoRoot, "internal", "cli", "cli.go"), 1, true
	}
	return bundledRoot, pluginRoot
}
