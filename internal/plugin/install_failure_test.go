/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// archiveCloseFailure closes the real destination while reporting a storage error.
type archiveCloseFailure struct {
	io.WriteCloser
	err error
}

// Close releases its destination before returning the injected failure.
func (f archiveCloseFailure) Close() error { _ = f.WriteCloser.Close(); return f.err }

// TestArchiveCloseFailure rejects an archive whose extracted file did not close cleanly.
func TestArchiveCloseFailure(t *testing.T) {
	original := openArchiveOutput
	t.Cleanup(func() { openArchiveOutput = original })
	failure := errors.New("close failure")
	openArchiveOutput = func(path string, flag int, mode os.FileMode) (io.WriteCloser, error) {
		f, err := os.OpenFile(path, flag, mode)
		if err != nil {
			return nil, err
		}
		return archiveCloseFailure{f, failure}, nil
	}
	archive := filepath.Join(t.TempDir(), "plugin.tar.gz")
	writeCustomTarGz(t, archive, []tarEntry{{name: "plugin.json", body: minimalManifest("ecs")}})
	root := t.TempDir()
	if _, err := InstallLocalBundle(archive, root); !errors.Is(err, failure) {
		t.Fatalf("install error=%v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed extraction installed files: %v, %v", entries, err)
	}
}

// TestCopyBundleReplacementFailureCleansStaging preserves a broken destination link on failure.
func TestCopyBundleReplacementFailureCleansStaging(t *testing.T) {
	source := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	root := t.TempDir()
	dest := filepath.Join(root, "ecs")
	if err := os.Symlink(filepath.Join(root, "missing"), dest); err != nil {
		t.Fatal(err)
	}
	if _, err := copyBundleIntoPlace(source, root, "ecs"); err == nil {
		t.Fatal("replacement succeeded over broken link")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "ecs" {
		t.Fatalf("staging remains: %v, %v", entries, err)
	}
	if target, err := os.Readlink(dest); err != nil || target != filepath.Join(root, "missing") {
		t.Fatalf("destination changed: %q, %v", target, err)
	}
}
