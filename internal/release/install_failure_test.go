/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// failingArchiveOutput retains the real destination and models deferred storage failures.
type failingArchiveOutput struct {
	io.WriteCloser
	writeErr, closeErr error
	closed             bool
}

// Write injects a destination write failure when configured.
func (f *failingArchiveOutput) Write(data []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.WriteCloser.Write(data)
}

// Close closes the destination and optionally reports failure flushing its data.
func (f *failingArchiveOutput) Close() error {
	f.closed = true
	err := f.WriteCloser.Close()
	if f.closeErr != nil {
		return f.closeErr
	}
	return err
}

// TestExtractBinaryFilesystemConflicts rejects collisions while preserving existing files.
func TestExtractBinaryFilesystemConflicts(t *testing.T) {
	for _, kind := range []string{"directory parent", "file parent", "output directory"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			blocked := filepath.Join(root, "blocked")
			entries := []tarEntry{{name: "blocked/ctyun", body: "new"}}
			if kind == "output directory" {
				if err := os.Mkdir(blocked, 0700); err != nil {
					t.Fatal(err)
				}
				entries = []tarEntry{{name: "blocked", body: "new"}}
			} else {
				if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
				if kind == "directory parent" {
					entries = []tarEntry{{name: "blocked/subdir", dir: true}}
				}
			}
			if _, err := ExtractBinary(writeTarGz(t, entries), root, "ctyun"); err == nil {
				t.Fatal("conflicting extraction succeeded")
			}
			if kind != "output directory" {
				data, err := os.ReadFile(blocked)
				if err != nil || string(data) != "keep" {
					t.Fatalf("existing file changed: %q, %v", data, err)
				}
			}
		})
	}
}

// TestExtractBinaryInvalidTar rejects valid gzip data that cannot form a tar header.
func TestExtractBinaryInvalidTar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(f)
	if _, err := writer.Write([]byte("invalid tar")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractBinary(path, t.TempDir(), "ctyun"); err == nil {
		t.Fatal("invalid tar accepted")
	}
}

// TestInstallArtifactStorageFailures preserves the current executable and removes staging.
func TestInstallArtifactStorageFailures(t *testing.T) {
	originalOpen, originalChmod := openArchiveOutput, chmodInstalledBinary
	t.Cleanup(func() { openArchiveOutput = originalOpen; chmodInstalledBinary = originalChmod })
	failure := errors.New("storage failure")
	for _, stage := range []string{"create output", "write", "close", "chmod", "temporary directory"} {
		t.Run(stage, func(t *testing.T) {
			openArchiveOutput = originalOpen
			chmodInstalledBinary = originalChmod
			root := t.TempDir()
			current := filepath.Join(root, "ctyun")
			if err := os.WriteFile(current, []byte("old"), 0700); err != nil {
				t.Fatal(err)
			}
			archive := writeTarGz(t, []tarEntry{{name: "ctyun", body: "new"}})
			var output *failingArchiveOutput
			options := InstallOptions{CurrentExecutable: current, ArchivePath: archive, BinaryName: "ctyun"}
			switch stage {
			case "temporary directory":
				options.TempDir = filepath.Join(root, "missing")
			case "chmod":
				chmodInstalledBinary = func(string, os.FileMode) error { return failure }
			default:
				openArchiveOutput = func(path string, flag int, mode os.FileMode) (io.WriteCloser, error) {
					if stage == "create output" {
						return nil, failure
					}
					f, err := os.OpenFile(path, flag, mode)
					if err != nil {
						return nil, err
					}
					output = &failingArchiveOutput{WriteCloser: f}
					if stage == "write" {
						output.writeErr = failure
					} else {
						output.closeErr = failure
					}
					return output, nil
				}
			}
			err := InstallArtifact(options)
			if err == nil || stage != "temporary directory" && !errors.Is(err, failure) {
				t.Fatalf("install error=%v", err)
			}
			data, readErr := os.ReadFile(current)
			if readErr != nil || string(data) != "old" {
				t.Fatalf("current binary=%q, %v", data, readErr)
			}
			entries, readErr := os.ReadDir(root)
			if readErr != nil || len(entries) != 1 || entries[0].Name() != "ctyun" {
				t.Fatalf("staging remains: %v, %v", entries, readErr)
			}
			if output != nil && !output.closed {
				t.Error("destination handle leaked")
			}
		})
	}
}
