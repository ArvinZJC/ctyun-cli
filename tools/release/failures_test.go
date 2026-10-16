/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
)

// releaseTestFile writes controlled inputs without relying on repository files.
func releaseTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// releaseTestDir creates a directory used either as input or to block file writes.
func releaseTestDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// releaseTestKey supplies a deterministic signing key for packaging tests.
func releaseTestKey() ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
}

// TestMergeReplacesOnlySelectedRelease verifies completing a platform set,
// updating its artifact, and advancing a channel without losing other channels.
func TestMergeReplacesOnlySelectedRelease(t *testing.T) {
	old := corerelease.Index{Releases: []corerelease.Release{
		{Version: "0.1.0", Channel: "stable", Artifacts: []corerelease.Artifact{{OS: "linux", Arch: "amd64", URL: "old"}, {OS: "darwin", Arch: "arm64", URL: "retained"}}},
		{Version: "0.2.0-beta.1", Channel: "beta"},
	}}
	merged := upsertCoreRelease(old, corerelease.Release{Version: "0.1.0", Channel: "stable", PublishedAt: "now", Artifacts: []corerelease.Artifact{{OS: "linux", Arch: "amd64", URL: "replacement"}, {OS: "windows", Arch: "amd64", URL: "added"}}})
	if len(merged.Releases) != 2 || len(merged.Releases[0].Artifacts) != 3 || merged.Releases[0].Artifacts[0].URL != "replacement" || merged.Releases[0].Artifacts[1].URL != "retained" || merged.Releases[0].Artifacts[2].URL != "added" || merged.Releases[0].PublishedAt != "now" {
		t.Fatalf("bad merge: %#v", merged)
	}
	updated := upsertCoreRelease(merged, corerelease.Release{Version: "0.2.0", Channel: "stable"})
	if updated.Releases[0].Version != "0.2.0" || len(updated.Releases[0].Artifacts) != 0 || updated.Releases[1].Channel != "beta" {
		t.Fatalf("bad replacement: %#v", updated)
	}
	registryIndex := upsertRegistryArtifact(registry.Index{Plugins: []registry.Artifact{{Name: "demo", Channel: "beta", Version: "0.1.0-beta.1"}, {Name: "demo", Channel: "stable", Version: "0.1.0"}}}, registry.Artifact{Name: "demo", Channel: "stable", Version: "0.2.0"})
	if len(registryIndex.Plugins) != 2 || registryIndex.Plugins[0].Version != "0.1.0-beta.1" || registryIndex.Plugins[1].Version != "0.2.0" {
		t.Fatalf("bad plugin replacement: %#v", registryIndex)
	}
	if got := pluginDisplayName(plugin.Bundle{Manifest: plugin.Manifest{Name: "demo"}}); got != "demo" {
		t.Fatalf("fallback name %q", got)
	}
}

// TestMergeRejectsUnreadableOrInvalidIndexes protects retained channel metadata
// from being silently discarded when a previous index cannot be loaded.
func TestMergeRejectsUnreadableOrInvalidIndexes(t *testing.T) {
	for _, name := range []string{"core-index.json", "index.json"} {
		for _, kind := range []string{"directory", "invalid"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, name)
				if kind == "directory" {
					releaseTestDir(t, path)
				} else {
					releaseTestFile(t, path, "{")
				}
				var err error
				if name == "index.json" {
					_, err = mergeRegistryIndex(root, registry.Index{})
				} else {
					_, err = mergeCoreIndex(root, corerelease.Index{})
				}
				if err == nil {
					t.Fatal("accepted unreadable index")
				}
			})
		}
	}
}

// failAfterWrites simulates a destination that stops accepting key output.
type failAfterWrites struct{ remaining int }

// Write fails at the selected output line and accepts preceding lines.
func (w *failAfterWrites) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, io.ErrClosedPipe
	}
	w.remaining--
	return len(p), nil
}

// TestGenerateKeyPropagatesOutputFailures prevents successful reporting after
// only part of the generated signing-key instructions reached the destination.
func TestGenerateKeyPropagatesOutputFailures(t *testing.T) {
	for n := range 3 {
		if err := generateKey(&failAfterWrites{remaining: n}); !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("write %d: %v", n, err)
		}
	}
}

// TestArchiveFailurePaths checks source reads, invalid tar names, unsupported
// symlinks, truncated compressed input, and archive output failures.
func TestArchiveFailurePaths(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		if err := writeTarGzArchive(t.TempDir(), func(*tar.Writer) error { return nil }); err == nil {
			t.Fatal("created archive over directory")
		}
	})
	t.Run("missing binary", func(t *testing.T) {
		if err := writeArchive(filepath.Join(t.TempDir(), "out"), "missing", "ctyun"); err == nil {
			t.Fatal("missing binary accepted")
		}
	})
	t.Run("missing input", func(t *testing.T) {
		if err := writeDirectoryArchive(filepath.Join(t.TempDir(), "out"), filepath.Join(t.TempDir(), "missing"), "plugin"); err == nil {
			t.Fatal("missing directory accepted")
		}
	})
	t.Run("missing output parent", func(t *testing.T) {
		if err := writeDirectoryArchive(filepath.Join(t.TempDir(), "missing", "out"), t.TempDir(), "plugin"); err == nil {
			t.Fatal("missing parent accepted")
		}
	})
	t.Run("bad published gzip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "out")
		releaseTestFile(t, path, "broken gzip")
		if err := writeDirectoryArchive(path, t.TempDir(), "plugin"); err == nil {
			t.Fatal("corrupt published archive replaced")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Symlink("absent", filepath.Join(root, "link")); err != nil {
			t.Fatal(err)
		}
		if err := writeDirectoryArchive(filepath.Join(t.TempDir(), "out"), root, "plugin"); err == nil {
			t.Fatal("symlink archived")
		}
	})
	t.Run("bad tar name", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		releaseTestFile(t, file, "content")
		writer := tar.NewWriter(io.Discard)
		defer func() { _ = writer.Close() }()
		if err := addFileToArchive(writer, file, "bad\x00name", 0o644); err == nil {
			t.Fatal("invalid tar name accepted")
		}
	})
	t.Run("document read", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		releaseTestDir(t, filepath.Join(root, "README.md"))
		releaseTestFile(t, filepath.Join(root, "ctyun"), "binary")
		if err := writeArchive(filepath.Join(root, "out"), filepath.Join(root, "ctyun"), "ctyun"); err == nil {
			t.Fatal("unreadable documentation accepted")
		}
	})
	t.Run("document stat", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		if err := os.Symlink("README.md", filepath.Join(root, "README.md")); err != nil {
			t.Fatal(err)
		}
		releaseTestFile(t, filepath.Join(root, "ctyun"), "binary")
		if err := writeArchive(filepath.Join(root, "out"), filepath.Join(root, "ctyun"), "ctyun"); err == nil {
			t.Fatal("unreadable documentation path accepted")
		}
	})
	t.Run("documents absent", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		releaseTestFile(t, filepath.Join(root, "ctyun"), "binary")
		if err := writeArchive(filepath.Join(root, "out"), filepath.Join(root, "ctyun"), "ctyun"); err != nil {
			t.Fatal(err)
		}
	})
}

// TestInstallerCopyRejectsFilesystemFailures requires both installer scripts to
// be found, read and written before packaging can report success.
func TestInstallerCopyRejectsFilesystemFailures(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "output"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			out := t.TempDir()
			if kind == "directory" {
				releaseTestDir(t, filepath.Join(root, "install.sh"))
			} else if kind == "output" {
				releaseTestFile(t, filepath.Join(root, "install.sh"), "script")
				releaseTestDir(t, filepath.Join(out, "install.sh"))
			}
			if err := copyInstallerScripts(out); err == nil {
				t.Fatal("installer copy succeeded despite filesystem error")
			}
		})
	}
}

// TestPathAndChecksumFailures preserves failures resolving source paths and
// reading checksum input, including a working directory deleted by another process.
func TestPathAndChecksumFailures(t *testing.T) {
	if _, _, err := checksumAndSize(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing checksum input accepted")
	}
	if _, _, err := checksumAndSize(t.TempDir()); err == nil {
		t.Fatal("directory checksum input accepted")
	}
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("PWD", "")
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if _, err := projectFilePath("anything"); err == nil {
		t.Fatal("deleted working directory accepted")
	}
}

// TestCloseWithErrorPreservesBothFailures requires the original operation and
// failed cleanup to remain inspectable by errors.Is.
func TestCloseWithErrorPreservesBothFailures(t *testing.T) {
	for _, first := range []error{nil, io.ErrUnexpectedEOF} {
		got := closeWithError(first, func() error { return io.ErrClosedPipe })
		if !errors.Is(got, io.ErrClosedPipe) || first != nil && !errors.Is(got, first) {
			t.Fatalf("lost failures: %v", got)
		}
	}
}

// TestCoreArtifactInputAndOutputFailures ensures malformed platform names,
// unavailable temporary storage, and failed archive creation stop packaging.
func TestCoreArtifactInputAndOutputFailures(t *testing.T) {
	if _, err := buildCoreArtifact(releaseOptions{}, "", "bad"); err == nil {
		t.Fatal("bad platform accepted")
	}
	t.Run("temporary storage", func(t *testing.T) {
		t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
		if _, err := buildCoreArtifact(releaseOptions{}, "", "linux/amd64"); err == nil {
			t.Fatal("missing temp directory accepted")
		}
	})
	t.Run("archive output", func(t *testing.T) {
		defer patchBuildBinary(func(opts buildOptions) error { return os.WriteFile(opts.Output, []byte("binary"), 0o755) })()
		if _, err := buildCoreArtifact(releaseOptions{OutDir: filepath.Join(t.TempDir(), "missing"), Version: "0.5.1"}, "", "linux/amd64"); err == nil {
			t.Fatal("missing output directory accepted")
		}
	})
}

// TestPluginRegistryFilesystemFailures verifies failures locating, loading,
// archiving and persisting bundles without modifying the checked-in plugins.
func TestPluginRegistryFilesystemFailures(t *testing.T) {
	for _, kind := range []string{"missing plugins", "plugins file", "invalid bundle", "invalid index", "blocked index", "blocked signature", "skip file"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			out := t.TempDir()
			if kind == "plugins file" {
				releaseTestFile(t, filepath.Join(root, "plugins"), "file")
			} else if kind != "missing plugins" {
				releaseTestDir(t, filepath.Join(root, "plugins"))
			}
			switch kind {
			case "invalid bundle":
				releaseTestDir(t, filepath.Join(root, "plugins", "demo"))
			case "invalid index":
				releaseTestFile(t, filepath.Join(out, "index.json"), "{")
			case "blocked index":
				releaseTestDir(t, filepath.Join(out, "index.json"))
			case "blocked signature":
				releaseTestDir(t, filepath.Join(out, "index.sig"))
			case "skip file":
				releaseTestFile(t, filepath.Join(root, "plugins", "README.md"), "not a bundle")
			}
			err := writePluginRegistry(releaseOptions{OutDir: out, Version: "0.5.1"}, releaseTestKey())
			if kind == "skip file" {
				if err != nil {
					t.Fatal(err)
				}
				raw, e := os.ReadFile(filepath.Join(out, "index.json"))
				if e != nil || !bytes.Contains(raw, []byte(`"plugins": []`)) {
					t.Fatalf("unexpected empty index %s %v", raw, e)
				}
			} else if err == nil {
				t.Fatal("filesystem failure ignored")
			}
		})
	}
}

// TestReleaseOutputFailures exercises orchestration failure propagation after a
// successful binary build, without performing compiler or network operations.
func TestReleaseOutputFailures(t *testing.T) {
	defer patchBuildBinary(func(opts buildOptions) error { return os.WriteFile(opts.Output, []byte("binary"), 0o755) })()
	for _, kind := range []string{"output directory", "invalid previous index", "core signature", "installer", "plugins"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			out := filepath.Join(root, "out")
			releaseTestDir(t, out)
			switch kind {
			case "output directory":
				out = filepath.Join(root, "file")
				releaseTestFile(t, out, "file")
			case "invalid previous index":
				releaseTestFile(t, filepath.Join(out, "core-index.json"), "{")
			case "core signature":
				releaseTestDir(t, filepath.Join(out, "core-index.sig"))
			case "plugins":
				releaseTestFile(t, filepath.Join(root, "install.sh"), "script")
				releaseTestFile(t, filepath.Join(root, "install.ps1"), "script")
			}
			if err := writeRelease(releaseOptions{Version: "0.5.1", Channel: "stable", OutDir: out, Platforms: []string{"linux/amd64"}}, releaseTestKey()); err == nil {
				t.Fatal("release ignored output failure")
			}
		})
	}
}

// TestReadArchivePayloadRejectsTruncation checks gzip headers as well as payload
// integrity instead of accepting any file with a gzip prefix.
func TestReadArchivePayloadRejectsTruncation(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte("archive payload")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, content := range [][]byte{[]byte("invalid gzip header"), compressed.Bytes()[:compressed.Len()-4]} {
		path := filepath.Join(t.TempDir(), "broken.gz")
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readArchivePayload(path); err == nil {
			t.Fatal("corrupt gzip accepted")
		}
	}
}
