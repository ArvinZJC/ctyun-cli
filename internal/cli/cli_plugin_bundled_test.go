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

// TestPluginBundledInstallAndUpdateAreDevOnly installs and updates synthetic development bundles while rejecting release access.
func TestPluginBundledInstallAndUpdateAreDevOnly(t *testing.T) {
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")

	restoreVersion := patchVersion(testCoreVersion() + "-dev")
	defer func() {
		restoreVersion()
	}()
	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"plugin", "install", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled install returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Plugin install complete: installed 1; already installed 0; failed 0.") {
		t.Fatalf("bundled install output = %q", stdout.String())
	}

	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.3.0")
	stdout.Reset()
	if err := Run(Config{Args: []string{"plugin", "update", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled update returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Plugin update complete: updated 1; already current 0; failed 0.") {
		t.Fatalf("bundled update output = %q", stdout.String())
	}

	restoreVersion()
	restoreVersion = patchVersion("0.1.0")
	if err := Run(Config{Args: []string{"plugin", "install", "ecs", "--bundled"}, Stdout: io.Discard, PluginRoot: t.TempDir()}); err == nil {
		t.Fatal("released build accepted --bundled plugin install")
	}
}

// TestPluginBundledStatusMessagesUseLanguage localizes bundled install and update summaries.
func TestPluginBundledStatusMessagesUseLanguage(t *testing.T) {
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")
	t.Cleanup(patchVersion(testCoreVersion() + "-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"--lang", "zh-CN", "plugin", "install", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled install returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "插件安装完成：已安装 1 个；已存在 0 个；失败 0 个。" {
		t.Fatalf("bundled install output = %q", got)
	}

	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.3.0")
	stdout.Reset()
	if err := Run(Config{Args: []string{"--lang", "zh-CN", "plugin", "update", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled update returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "插件更新完成：已更新 1 个；已是最新 0 个；失败 0 个。" {
		t.Fatalf("bundled update output = %q", got)
	}
}

// TestPluginBundledUpdateAll updates only synthetic installed bundles with newer available versions.
func TestPluginBundledUpdateAll(t *testing.T) {
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.3.0")
	writeVersionedBundle(t, filepath.Join(bundledRoot, "vpc"), "vpc", "0.2.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.1.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "vpc"), "vpc", "0.2.0")
	t.Cleanup(patchVersion(testCoreVersion() + "-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"plugin", "update", "--all", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled update --all returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Plugin update complete: updated 1; already current 1; failed 0.") {
		t.Fatalf("bundled update --all output = %q", stdout.String())
	}
}

// TestPluginBundledReinstallRefreshesSameVersion replaces stale metadata even when the selected bundle version is unchanged.
func TestPluginBundledReinstallRefreshesSameVersion(t *testing.T) {
	bundledRoot, pluginRoot := prepareBundledPluginRoots(t)
	writeVersionedBundle(t, filepath.Join(bundledRoot, "ecs"), "ecs", "0.2.0")
	writeVersionedBundle(t, filepath.Join(pluginRoot, "ecs"), "ecs", "0.2.0")
	if err := os.MkdirAll(filepath.Join(pluginRoot, "ecs", "i18n"), 0o755); err != nil {
		t.Fatalf("create stale i18n dir: %v", err)
	}
	mustWrite(t, filepath.Join(pluginRoot, "ecs", "i18n", "en-US.json"), `{"name": "Stale ECS"}`)
	t.Cleanup(patchVersion(testCoreVersion() + "-dev"))

	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"plugin", "reinstall", "ecs", "--bundled"}, Stdout: &stdout, PluginRoot: pluginRoot}); err != nil {
		t.Fatalf("bundled reinstall returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Plugin reinstall complete: reinstalled 1; failed 0.") {
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
// tests, enters it, and points runtimeCaller at its source layout.
func prepareBundledPluginRoots(t *testing.T) (string, string) {
	t.Helper()

	repoRoot := t.TempDir()
	bundledRoot := filepath.Join(repoRoot, "plugins")
	if err := os.Mkdir(bundledRoot, 0o755); err != nil {
		t.Fatalf("create bundled root: %v", err)
	}
	t.Chdir(repoRoot)
	pluginRoot := t.TempDir()
	originalCaller := runtimeCaller
	t.Cleanup(func() { runtimeCaller = originalCaller })
	runtimeCaller = func(int) (uintptr, string, int, bool) {
		return 0, filepath.Join(repoRoot, "internal", "cli", "cli.go"), 1, true
	}
	return bundledRoot, pluginRoot
}

// TestBundledMutationsRespectChannel checks that explicit and batch operations honour their selected source channel.
func TestBundledMutationsRespectChannel(t *testing.T) {
	source, _ := prepareBundledPluginRoots(t)
	t.Cleanup(patchVersion(testCoreVersion() + "-dev"))
	writeVersionedBundle(t, filepath.Join(source, "demo"), "demo", "0.3.0")
	for _, command := range []string{"install", "reinstall", "update"} {
		for _, target := range []string{"demo", "--all"} {
			t.Run(command+"_"+target, func(t *testing.T) {
				root := t.TempDir()
				if command != "install" {
					writeVersionedBundle(t, filepath.Join(root, "demo"), "demo", "0.1.0")
				}
				var out bytes.Buffer
				err := Run(Config{Args: []string{"--lang", "en-US", "plugin", command, target, "--bundled", "--channel", "beta"}, PluginRoot: root, Stdout: &out, Stderr: io.Discard})
				wantError := target == "demo" || command == "reinstall"
				if (err != nil) != wantError {
					t.Errorf("channel selection returned %v, want error %v", err, wantError)
				}
				if command == "update" && target == "--all" && !strings.Contains(out.String(), "already current 1") {
					t.Errorf("update summary for unavailable source: %s", out.String())
				}
				if command == "install" {
					if _, err := os.Stat(filepath.Join(root, "demo")); !os.IsNotExist(err) {
						t.Errorf("installed stable plugin from beta channel: %v", err)
					}
				} else if got, err := loadInstalledPluginForTest(root, "demo"); err != nil || got != "0.1.0" {
					t.Errorf("changed plugin from wrong channel: %q, %v", got, err)
				}
			})
		}
	}
}

// TestBundledReinstallRejectsMisdirectedNames preserves unrelated installed plugins when metadata names the wrong directory.
func TestBundledReinstallRejectsMisdirectedNames(t *testing.T) {
	for _, invalid := range []string{"source", "installed"} {
		t.Run(invalid, func(t *testing.T) {
			source, root := prepareBundledPluginRoots(t)
			t.Cleanup(patchVersion(testCoreVersion() + "-dev"))
			sourceName, installedName := "selected", "selected"
			if invalid == "source" {
				sourceName = "other"
			} else {
				installedName = "other"
			}
			writeVersionedBundle(t, filepath.Join(source, "selected"), sourceName, "0.3.0")
			writeVersionedBundle(t, filepath.Join(source, "other"), "other", "0.3.0")
			writeVersionedBundle(t, filepath.Join(root, "selected"), installedName, "0.1.0")
			writeVersionedBundle(t, filepath.Join(root, "other"), "other", "0.1.0")
			args := []string{"plugin", "reinstall", "selected", "--bundled"}
			err := Run(Config{Args: args, PluginRoot: root, Stdout: io.Discard, Stderr: io.Discard})
			if err == nil {
				t.Error("reinstalled a differently named plugin")
			}
			bundle, err := plugin.LoadBundle(filepath.Join(root, "other"), version.Version)
			if err != nil || bundle.Manifest.Version != "0.1.0" {
				t.Errorf("unrelated installed plugin changed: %q, %v", bundle.Manifest.Version, err)
			}
		})
	}
}

// TestBundledInstallFiltersStableQualityAndSkipsInstalled checks the default, explicit channel and repeated install.
func TestBundledInstallFiltersStableQualityAndSkipsInstalled(t *testing.T) {
	source, root := prepareBundledPluginRoots(t)
	t.Cleanup(patchVersion(testCoreVersion() + "-dev"))
	for name, replacement := range map[string][2]string{"preview": {"stable", "beta"}, "draft": {"reviewed", "generated"}} {
		dir := filepath.Join(source, name)
		writeVersionedBundle(t, dir, name, "1.0.0")
		path := filepath.Join(dir, "plugin.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, path, strings.Replace(string(raw), replacement[0], replacement[1], 1))
	}
	for _, tc := range []struct {
		args    []string
		summary string
	}{
		{[]string{"--all"}, "installed 0; already installed 0; failed 0"},
		{[]string{"--all", "--channel", "beta"}, "installed 1; already installed 0; failed 0"},
		{[]string{"preview"}, "installed 0; already installed 1; failed 0"},
	} {
		var out bytes.Buffer
		args := append([]string{"--lang", "en-US", "plugin", "install", "--bundled"}, tc.args...)
		err := Run(Config{Args: args, PluginRoot: root, Stdout: &out, Stderr: io.Discard})
		if err != nil || !strings.Contains(out.String(), tc.summary) {
			t.Fatalf("%v returned %q, %v", args, out.String(), err)
		}
	}
}

// TestPluginManagementReportsUnreadableRoots distinguishes inaccessible inventories from empty ones.
func TestPluginManagementReportsUnreadableRoots(t *testing.T) {
	bundled, installed := prepareBundledPluginRoots(t)
	t.Cleanup(patchVersion(testCoreVersion() + "-dev"))
	for _, dir := range []string{bundled, installed} {
		if err := os.Chmod(dir, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
		if _, err := os.ReadDir(dir); !os.IsPermission(err) {
			t.Skipf("directory permission checks unavailable: %v", err)
		}
	}
	readable := t.TempDir()
	for _, tc := range []struct {
		root string
		args []string
	}{
		{installed, []string{"plugin", "list"}},
		{readable, []string{"plugin", "list", "--available", "--bundled"}},
		{readable, []string{"plugin", "search", "demo", "--bundled"}},
		{readable, []string{"plugin", "install", "--all", "--bundled"}},
		{installed, []string{"plugin", "reinstall", "--all", "--bundled"}},
		{installed, []string{"plugin", "update", "--all", "--bundled"}},
		{installed, []string{"plugin", "reinstall", "--all", "--source", "github"}},
		{installed, []string{"plugin", "update", "--all", "--source", "github"}},
		{installed, []string{"--yes", "plugin", "remove", "--all"}},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			err := Run(Config{Args: tc.args, PluginRoot: tc.root, Stdout: io.Discard, Stderr: io.Discard})
			if err == nil || !strings.Contains(err.Error(), "permission denied") {
				t.Fatalf("expected an unreadable inventory error, got %v", err)
			}
		})
	}
}
