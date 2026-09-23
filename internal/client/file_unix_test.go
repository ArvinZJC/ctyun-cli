//go:build !windows

/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"path/filepath"
	"syscall"
	"testing"
)

// TestFileUploadRejectsFIFOWithoutPeer prevents a local special file from hanging preparation.
func TestFileUploadRejectsFIFOWithoutPeer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fifo")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenRegularFile(path); err == nil {
		t.Fatal("FIFO accepted")
	}
}
