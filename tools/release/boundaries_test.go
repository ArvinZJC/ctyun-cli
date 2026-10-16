/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"crypto/ed25519"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMainReportsExitStatus invokes the real entry point while intercepting only
// process termination so failures and successful key generation remain covered.
func TestMainReportsExitStatus(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[invalid], func(t *testing.T) {
			oldArgs, oldExit, oldOut, oldErr := os.Args, exitProcess, os.Stdout, os.Stderr
			t.Cleanup(func() { os.Args, exitProcess, os.Stdout, os.Stderr = oldArgs, oldExit, oldOut, oldErr })
			out, err := os.CreateTemp(t.TempDir(), "output")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = out.Close() })
			os.Stdout, os.Stderr = out, out
			os.Args = []string{"release", "--generate-key"}
			if invalid {
				os.Args = []string{"release", "--invalid"}
			}
			status := -1
			exitProcess = func(code int) { status = code }
			main()
			data, err := os.ReadFile(out.Name())
			if err != nil {
				t.Fatal(err)
			}
			if invalid {
				if status != 1 || !strings.Contains(string(data), "flag provided but not defined") {
					t.Fatalf("status=%d output=%s", status, data)
				}
			} else if status != -1 || !strings.Contains(string(data), "CTYUN_RELEASE_PUBLIC_KEY=") {
				t.Fatalf("status=%d output=%s", status, data)
			}
		})
	}
}

// TestKeyGenerationFailureProducesNoPartialKey ensures an entropy/key provider
// failure reaches the caller without printing misleading key assignments.
func TestKeyGenerationFailureProducesNoPartialKey(t *testing.T) {
	original := createSigningKey
	t.Cleanup(func() { createSigningKey = original })
	createSigningKey = func(io.Reader) (ed25519.PublicKey, ed25519.PrivateKey, error) { return nil, nil, io.ErrUnexpectedEOF }
	if err := generateKey(io.Discard); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("lost key provider failure: %v", err)
	}
}

// TestReleaseEncodingAndWriteFailures checks that serialization and a destination
// replaced during publication both fail before a signed success is reported.
func TestReleaseEncodingAndWriteFailures(t *testing.T) {
	for _, pluginIndex := range []bool{false, true} {
		for _, writeFailure := range []bool{false, true} {
			t.Run(map[bool]string{false: "core", true: "plugins"}[pluginIndex]+map[bool]string{false: " encoding", true: " write"}[writeFailure], func(t *testing.T) {
				original := encodeReleaseJSON
				t.Cleanup(func() { encodeReleaseJSON = original })
				root := t.TempDir()
				t.Chdir(root)
				releaseTestDir(t, filepath.Join(root, "plugins"))
				out := t.TempDir()
				name := "core-index.json"
				if pluginIndex {
					name = "index.json"
				}
				encodeReleaseJSON = func(value any) ([]byte, error) {
					if !writeFailure {
						return nil, io.ErrClosedPipe
					}
					releaseTestDir(t, filepath.Join(out, name))
					return indentedJSON(value)
				}
				var err error
				opts := releaseOptions{Version: "0.5.1", Channel: "stable", OutDir: out}
				if pluginIndex {
					err = writePluginRegistry(opts, releaseTestKey())
				} else {
					err = writeRelease(opts, releaseTestKey())
				}
				if err == nil {
					t.Fatal("publication ignored encoder/output failure")
				}
				if !writeFailure && !errors.Is(err, io.ErrClosedPipe) {
					t.Fatalf("lost encoding failure: %v", err)
				}
				if _, err := os.Stat(filepath.Join(out, strings.TrimSuffix(name, ".json")+".sig")); !os.IsNotExist(err) {
					t.Fatal("failed index was signed")
				}
			})
		}
	}
}

// TestCoreArchiveChecksumFailureStopsPublication ensures an archive unreadable
// after writing cannot enter the release index with an absent or invalid hash.
func TestCoreArchiveChecksumFailureStopsPublication(t *testing.T) {
	original := checksumArchive
	t.Cleanup(func() { checksumArchive = original })
	checksumArchive = func(string) (string, int64, error) { return "", 0, io.ErrUnexpectedEOF }
	defer patchBuildBinary(func(opts buildOptions) error { return os.WriteFile(opts.Output, []byte("binary"), 0o755) })()
	if _, err := buildCoreArtifact(releaseOptions{Version: "0.5.1", OutDir: t.TempDir()}, "", "linux/amd64"); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("lost checksum failure: %v", err)
	}
}

// TestCandidateArchiveReadFailurePreservesPublishedArchive checks the second
// read boundary independently: failed candidate verification must not replace
// the previously published compressed bytes.
func TestCandidateArchiveReadFailurePreservesPublishedArchive(t *testing.T) {
	root := t.TempDir()
	releaseTestFile(t, filepath.Join(root, "plugin.json"), "{}")
	path := filepath.Join(t.TempDir(), "published.gz")
	if err := writeDirectoryArchive(path, root, "demo"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	original := loadArchivePayload
	t.Cleanup(func() { loadArchivePayload = original })
	loadArchivePayload = func(candidate string) ([]byte, error) {
		if candidate != path {
			return nil, io.ErrUnexpectedEOF
		}
		return readArchivePayload(candidate)
	}
	if err := writeDirectoryArchive(path, root, "demo"); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("lost candidate read error: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatalf("published archive changed: %v", err)
	}
}

// TestPluginArchiveFailuresStopIndexPublication checks real archive creation and
// checksum failures after loading a valid isolated plugin bundle.
func TestPluginArchiveFailuresStopIndexPublication(t *testing.T) {
	source, err := projectFilePath("plugins/region")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"archive", "checksum"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			if err := os.CopyFS(filepath.Join(root, "plugins", "region"), os.DirFS(source)); err != nil {
				t.Fatal(err)
			}
			out := t.TempDir()
			if kind == "archive" {
				if err := os.Symlink("absent", filepath.Join(root, "plugins", "region", "unreadable")); err != nil {
					t.Fatal(err)
				}
			} else {
				original := checksumArchive
				t.Cleanup(func() { checksumArchive = original })
				checksumArchive = func(string) (string, int64, error) { return "", 0, io.ErrUnexpectedEOF }
			}
			if err := writePluginRegistry(releaseOptions{Version: "0.5.1", OutDir: out}, releaseTestKey()); err == nil {
				t.Fatal("failed artifact entered index")
			}
			if _, err := os.Stat(filepath.Join(out, "index.json")); !os.IsNotExist(err) {
				t.Fatal("index published after artifact failure")
			}
		})
	}
}

// TestProjectLookupPropagatesWorkingDirectoryFailure keeps path-resolution
// failures distinct from a missing project file on platforms that retain a
// deleted directory's pathname in Getwd.
func TestProjectLookupPropagatesWorkingDirectoryFailure(t *testing.T) {
	original := workingDirectory
	t.Cleanup(func() { workingDirectory = original })
	workingDirectory = func() (string, error) { return "", os.ErrPermission }
	if _, err := projectFilePath("plugins"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("lost working-directory failure: %v", err)
	}
}
