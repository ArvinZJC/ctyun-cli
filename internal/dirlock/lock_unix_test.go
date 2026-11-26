//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package dirlock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// TestDirectoryLockFailure checks interruption, lock failures and inaccessible store paths.
func TestDirectoryLockFailure(t *testing.T) {
	root := t.TempDir()
	original := flock
	t.Cleanup(func() { flock = original })
	attempts := 0
	flock = func(fd, mode int) error {
		attempts++
		if attempts == 1 {
			return unix.EINTR
		}
		return unix.EBADF
	}
	action := func() error { t.Fatal("failed lock invoked mutation"); return nil }
	if err := With(root, true, action); !errors.Is(err, unix.EBADF) || attempts != 2 {
		t.Fatalf("lock attempts=%d, error=%v", attempts, err)
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := With(filepath.Join(file, "child"), false, action); err == nil {
		t.Fatal("inspect below a regular file succeeded")
	}
	failure := errors.New("operation error")
	if err := With(root, false, func() error { return failure }); !errors.Is(err, unix.EBADF) {
		t.Fatalf("unexpected lock error: %v", err)
	}
	flock = original
	if err := With(root, false, func() error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("lost reader error: %v", err)
	}
}
