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

// TestNewInstallRejectsUnrelatedWinner keeps a different manifest from satisfying a pending installation.
func TestNewInstallRejectsUnrelatedWinner(t *testing.T) {
	root := t.TempDir()
	requested := writeBundle(t, "requested", testCompatibleCoreConstraint())
	other := writeBundle(t, "other", testCompatibleCoreConstraint())
	target := filepath.Join(root, "requested")
	if err := os.Rename(other, target); err != nil {
		t.Fatal(err)
	}
	_, err := InstallNewVerifiedLocalBundle(requested, root, testCoreVersion())
	if err == nil || errors.Is(err, ErrAlreadyInstalled) {
		t.Fatalf("unrelated bundle was accepted as the installed target: %v", err)
	}
	manifest, err := LoadManifest(target)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "other" {
		t.Fatalf("existing bundle was changed: %q", manifest.Name)
	}
}

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
	if _, err := copyBundleIntoPlace(source, root, "ecs", replaceDir); err == nil {
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

// TestConcurrentNewBundleInstall publishes one complete bundle and leaves the winner in place.
func TestConcurrentNewBundleInstall(t *testing.T) {
	source, root := writeBundle(t, "demo", testCompatibleCoreConstraint()), t.TempDir()
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := InstallNewVerifiedLocalBundle(source, root, testCoreVersion())
			results <- err
		}()
	}
	installed, skipped := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			installed++
		case errors.Is(err, ErrAlreadyInstalled):
			skipped++
		default:
			t.Errorf("install error: %v", err)
		}
	}
	if installed != 1 || skipped != 1 {
		t.Fatalf("installed=%d skipped=%d", installed, skipped)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "demo" {
		t.Fatalf("unexpected remaining entries: %v, %v", entries, err)
	}
}

// TestReplaceDirReportsRollbackFailure verifies both filesystem errors are visible when restoration fails.
func TestReplaceDirReportsRollbackFailure(t *testing.T) {
	src, dest := t.TempDir(), filepath.Join(t.TempDir(), "installed")
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatal(err)
	}
	original := renameBundlePath
	t.Cleanup(func() { renameBundlePath = original })
	replaceErr, restoreErr := errors.New("replacement failed"), errors.New("rollback failed")
	renameBundlePath = func(from, to string) error {
		if from == dest {
			if ValidName(filepath.Base(to)) {
				t.Errorf("backup %q can be mistaken for an installed plugin", filepath.Base(to))
			}
			return os.Rename(from, to)
		}
		if from == src {
			return replaceErr
		}
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: restoreErr}
	}
	err := replaceDir(src, dest)
	if !errors.Is(err, replaceErr) || !errors.Is(err, restoreErr) {
		t.Fatalf("missing replacement or rollback cause: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(dest))
	if err != nil || len(entries) != 1 || IsBundleDir(entries[0]) {
		t.Fatalf("expected one recoverable backup outside plugin discovery, got %v, %v", entries, err)
	}
}

// TestReplacementChecksInstalledManifest rejects missing or changed targets under the store lock.
func TestReplacementChecksInstalledManifest(t *testing.T) {
	source, root := writeBundle(t, "demo", testCompatibleCoreConstraint()), t.TempDir()
	replace := func(expected string) error {
		_, err := ReplaceInstalledBundle(source, root, testCoreVersion(), expected)
		return err
	}
	if err := replace("0.1.0"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing installation: %v", err)
	}
	if _, err := InstallNewVerifiedLocalBundle(source, root, testCoreVersion()); err != nil {
		t.Fatal(err)
	}
	if err := replace("0.0.1"); err == nil {
		t.Fatal("replaced an installation whose version had changed")
	}
	if err := replace("0.1.0"); err != nil {
		t.Fatalf("matching install: %v", err)
	}
	if err := WithStoreLock(filepath.Join(root, "demo", "plugin.json", "invalid"), false, func() error { return nil }); err == nil {
		t.Fatal("accepted non-directory parent")
	}
	if err := WithStoreLock(filepath.Join(root, "absent"), false, func() error { return os.ErrNotExist }); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty inventory: %v", err)
	}
}
