/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
)

// TestPublicationRejectsCrossChannelTagReuse protects both core and plugin archives referenced by retained channels.
func TestPublicationRejectsCrossChannelTagReuse(t *testing.T) {
	defer patchBuildBinary(func(opts buildOptions) error {
		return os.WriteFile(opts.Output, []byte(opts.Channel), 0o755)
	})()
	for _, product := range []string{"core", "plugin"} {
		t.Run(product, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			releaseTestFile(t, "install.sh", "sh")
			releaseTestFile(t, "install.ps1", "ps")
			manifest := `{"name":"sample","version":"0.1.0-rc.1","channel":"alpha","quality":"reviewed","requires":{"ctyun":">=0.3.1 <1.0.0"},"api":{"product":"sample","ctyun_product_id":1}}`
			manifestPath := filepath.Join("plugins", "sample", "plugin.json")
			releaseTestFile(t, manifestPath, manifest)
			releaseTestFile(t, filepath.Join("plugins", "sample", "commands.json"), `{"commands":[]}`)
			releaseTestFile(t, filepath.Join("plugins", "sample", "tables.json"), `{"tables":{}}`)
			key := releaseTestKey()
			opts := releaseOptions{OutDir: t.TempDir(), Version: "0.5.3-rc.1", Channel: "alpha", Platforms: []string{"darwin/arm64"}}
			if err := writeRelease(opts, key); err != nil {
				t.Fatal(err)
			}
			newer := opts
			newer.Version, newer.Channel = "0.5.3-rc.9", "beta"
			releaseTestFile(t, manifestPath, strings.NewReplacer("0.1.0-rc.1", "0.1.0-rc.9", "alpha", "beta").Replace(manifest))
			if err := writeRelease(newer, key); err != nil {
				t.Fatal(err)
			}
			releaseTestFile(t, manifestPath, manifest)
			archive := "ctyun_0.5.3-rc.1_darwin_arm64.tar.gz"
			if product == "core" {
				opts.Channel = "beta"
			} else {
				archive = "ctyun_plugin_sample_0.1.0-rc.1.tar.gz"
				releaseTestFile(t, manifestPath, strings.Replace(manifest, `"alpha"`, `"beta"`, 1))
			}
			path := filepath.Join(opts.OutDir, archive)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := writeRelease(opts, key); err == nil || !strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), "beta") {
				t.Fatalf("expected conflicting release channels, got %v", err)
			}
			if after, err := os.ReadFile(path); err != nil || !bytes.Equal(before, after) {
				t.Fatalf("changed retained archive %s: %v", archive, err)
			}
		})
	}
}

// TestStagedReleaseKeepsEachPlatformsNewestBuild publishes partial and out-of-order release histories.
func TestStagedReleaseKeepsEachPlatformsNewestBuild(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	releaseTestFile(t, "install.sh", "sh")
	releaseTestFile(t, "install.ps1", "ps")
	path := filepath.Join("plugins", "demo")
	releaseTestFile(t, filepath.Join(path, "commands.json"), `{"commands":[]}`)
	releaseTestFile(t, filepath.Join(path, "tables.json"), `{"tables":{}}`)
	defer patchBuildBinary(func(opts buildOptions) error {
		return os.WriteFile(opts.Output, []byte(opts.Version), 0o755)
	})()
	key, output := releaseTestKey(), t.TempDir()
	publish := func(version, bundleVersion string, targets []string, want map[string]string, latestCore, latestBundle string, wantReleases int) {
		t.Helper()
		manifest := fmt.Sprintf(`{"name":"demo","version":%q,"channel":"stable","quality":"reviewed","requires":{"ctyun":">=0.3.1 <1.0.0"},"api":{"product":"demo","ctyun_product_id":1}}`, bundleVersion)
		releaseTestFile(t, filepath.Join(path, "plugin.json"), manifest)
		if err := writeRelease(releaseOptions{OutDir: output, Version: version, Channel: "stable", Platforms: targets}, key); err != nil {
			t.Fatal(err)
		}
		publicKey := base64.StdEncoding.EncodeToString(key[32:])
		raw, err := corerelease.ReadSignedIndex(output, publicKey, nil)
		if err != nil {
			t.Fatal(err)
		}
		index, err := corerelease.LoadIndex(raw)
		if err != nil {
			t.Fatal(err)
		}
		assertPublishedVersion(t, raw, []string{"stable"}, latestCore)
		first := make(map[string]string)
		for _, rel := range index.Releases {
			for _, art := range rel.Artifacts {
				if first[art.OS] == "" {
					first[art.OS] = rel.Version
				}
			}
		}
		for target, v := range want {
			rel, _, ok := index.FindLatest("stable", target, "arm64")
			if !ok || rel.Version != v || first[target] != v {
				t.Fatalf("after %s, %s: runtime %q, bootstrap %q; want %q", version, target, rel.Version, first[target], v)
			}
		}
		if len(index.Releases) != wantReleases {
			t.Fatalf("after %s, retained %d releases, want %d", version, len(index.Releases), wantReleases)
		}
		for _, dir := range []string{output, filepath.Join(output, "gitee")} {
			data, err := os.ReadFile(filepath.Join(dir, "index.json"))
			if err != nil {
				t.Fatal(err)
			}
			sig, err := os.ReadFile(filepath.Join(dir, "index.sig"))
			if err != nil {
				t.Fatal(err)
			}
			if err := distribution.VerifyIndexSignature(data, sig, publicKey, "registry"); err != nil {
				t.Fatal(err)
			}
			assertPublishedVersion(t, data, []string{"demo", "stable"}, latestBundle)
		}
	}
	publish("0.5.2", "0.2.0", []string{"darwin/arm64", "windows/arm64"}, map[string]string{"darwin": "0.5.2", "windows": "0.5.2"}, "0.5.2", "0.2.0", 1)
	publish("0.5.3", "0.3.0", []string{"windows/arm64"}, map[string]string{"darwin": "0.5.2", "windows": "0.5.3"}, "0.5.3", "0.3.0", 2)
	publish("0.5.1", "0.1.0", []string{"linux/arm64"}, map[string]string{"darwin": "0.5.2", "windows": "0.5.3", "linux": "0.5.1"}, "0.5.3", "0.3.0", 3)
	publish("0.5.3", "0.3.0", []string{"darwin/arm64", "linux/arm64"}, map[string]string{"darwin": "0.5.3", "windows": "0.5.3", "linux": "0.5.3"}, "0.5.3", "0.3.0", 1)
	publish("0.5.1", "0.1.0", []string{"linux/arm64"}, map[string]string{"darwin": "0.5.3", "windows": "0.5.3", "linux": "0.5.3"}, "0.5.3", "0.3.0", 1)
	publish("0.5.3+b", "0.3.0+b", []string{"windows/arm64"}, map[string]string{"darwin": "0.5.3", "windows": "0.5.3+b", "linux": "0.5.3"}, "0.5.3+b", "0.3.0+b", 2)
}

// TestPublishedHistoryServesNewestEligiblePlatform checks the order consumed by both bootstrap installers.
func TestPublishedHistoryServesNewestEligiblePlatform(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	releaseTestDir(t, "plugins")
	releaseTestFile(t, "install.sh", "sh")
	releaseTestFile(t, "install.ps1", "ps")
	previous, output := t.TempDir(), t.TempDir()
	artifact := func(goos string) corerelease.Artifact {
		return corerelease.Artifact{OS: goos, Arch: "arm64", URL: "core-" + goos + ".tar.gz", SHA256: strings.Repeat("a", 64)}
	}
	old := corerelease.Index{Schema: 1, Releases: []corerelease.Release{
		{Version: "0.5.2", Channel: "stable", Artifacts: []corerelease.Artifact{artifact("darwin")}},
		{Version: "0.5.11", Channel: "stable", Artifacts: []corerelease.Artifact{artifact("linux")}},
		{Version: "0.5.10", Channel: "stable", Artifacts: []corerelease.Artifact{artifact("darwin")}},
	}}
	raw, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	releaseTestFile(t, filepath.Join(previous, "core-index.json"), string(raw))
	current := corerelease.Index{Schema: 1, Releases: []corerelease.Release{{Version: "0.6.0-rc.1", Channel: "alpha", Artifacts: []corerelease.Artifact{artifact("darwin")}}}}
	key := releaseTestKey()
	if err := writeReleaseMetadata(releaseOptions{OutDir: output}, key, previous, current); err != nil {
		t.Fatal(err)
	}
	published, err := corerelease.ReadSignedIndex(output, base64.StdEncoding.EncodeToString(key[32:]), nil)
	if err != nil {
		t.Fatal(err)
	}
	assertPublishedVersion(t, published, []string{"stable"}, "0.5.11")
	index, err := corerelease.LoadIndex(published)
	if err != nil {
		t.Fatal(err)
	}
	first := make(map[string]string)
	for _, rel := range index.Releases {
		if rel.Channel == "stable" {
			for _, target := range rel.Artifacts {
				if _, ok := first[target.OS]; !ok {
					first[target.OS] = rel.Version
				}
			}
		}
	}
	if first["darwin"] != "0.5.10" || first["linux"] != "0.5.11" {
		t.Fatalf("bootstrap versions by platform = %v", first)
	}
}

// TestRebuildingIndexedHistoryKeepsNewerVersion verifies an older platform or plugin rebuild leaves the newer pointer available.
func TestRebuildingIndexedHistoryKeepsNewerVersion(t *testing.T) {
	t.Run("core", func(t *testing.T) {
		history := corerelease.Index{Releases: []corerelease.Release{
			{Version: "0.5.10", Channel: "stable", Artifacts: []corerelease.Artifact{{OS: "darwin", Arch: "arm64"}}},
			{Version: "0.5.2", Channel: "stable", Artifacts: []corerelease.Artifact{{OS: "darwin", Arch: "arm64"}}},
		}}
		merged := upsertCoreRelease(history, corerelease.Release{Version: "0.5.2", Channel: "stable", Artifacts: []corerelease.Artifact{{OS: "linux", Arch: "arm64"}}})
		if merged.Releases[0].Version != "0.5.10" || len(merged.Releases[1].Artifacts) != 2 {
			t.Fatalf("wrong release rebuilt: %#v", merged)
		}
	})
	t.Run("plugin", func(t *testing.T) {
		history := registry.Index{Plugins: []registry.Artifact{
			{Name: "sample", Version: "0.1.0-beta.10", Channel: "beta"},
			{Name: "sample", Version: "0.1.0-beta.1", Channel: "beta"},
		}}
		merged := upsertRegistryArtifact(history, registry.Artifact{Name: "sample", Version: "0.1.0-beta.1", Channel: "beta", URL: "rebuilt"})
		if merged.Plugins[0].Version != "0.1.0-beta.10" || merged.Plugins[1].URL != "rebuilt" {
			t.Fatalf("wrong plugin rebuilt: %#v", merged)
		}
		merged = upsertRegistryArtifact(merged, registry.Artifact{Name: "sample", Version: "0.1.0-beta.2", Channel: "beta"})
		latest, ok := merged.Find("sample", "beta")
		if !ok || latest.Version != "0.1.0-beta.10" {
			t.Fatalf("an unindexed historical build rolled the channel back to %q", latest.Version)
		}
	})
}
