/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"io"
	"os"
	"testing"
)

// faultSnapshot injects disk write, stat and close failures while retaining real cleanup paths.
type faultSnapshot struct {
	*os.File
	writeErr, statErr, closeErr error
}

// Write emulates a failed spool write when configured.
func (f faultSnapshot) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.File.Write(p)
}

// Stat emulates a lost descriptor when configured.
func (f faultSnapshot) Stat() (os.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.File.Stat()
}

// Close releases the real descriptor before reporting a configured close failure.
func (f faultSnapshot) Close() error {
	err := f.File.Close()
	if f.closeErr != nil {
		return f.closeErr
	}
	return err
}

// TestSnapshotDiskFailures removes partial snapshots after every filesystem failure.
func TestSnapshotDiskFailures(t *testing.T) {
	original := createBodySnapshot
	defer func() { createBodySnapshot = original }()
	for _, kind := range []string{"write", "stat", "close"} {
		t.Run(kind, func(t *testing.T) {
			var path string
			createBodySnapshot = func() (snapshotFile, error) {
				f, err := os.CreateTemp(t.TempDir(), "spool")
				if err != nil {
					return nil, err
				}
				path = f.Name()
				wrapped := faultSnapshot{File: f}
				switch kind {
				case "write":
					wrapped.writeErr = io.ErrShortWrite
				case "stat":
					wrapped.statErr = io.ErrClosedPipe
				case "close":
					wrapped.closeErr = io.ErrClosedPipe
				}
				return wrapped, nil
			}
			if _, err := PrepareBody(BodyInput{Encoding: "multipart", Parts: []BodyPart{{Name: "value", Value: "data"}}}); err == nil {
				t.Fatal("disk failure ignored")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("partial spool retained")
			}
		})
	}
}

// TestRegularSourceStatFailureClosesDescriptor propagates source descriptor failures.
func TestRegularSourceStatFailureClosesDescriptor(t *testing.T) {
	original := openBodySource
	defer func() { openBodySource = original }()
	openBodySource = func(string, int, os.FileMode) (*os.File, error) {
		file, err := os.CreateTemp(t.TempDir(), "closed")
		if err != nil {
			return nil, err
		}
		if closeErr := file.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		return file, nil
	}
	if _, err := OpenRegularFile("source"); err == nil {
		t.Fatal("closed descriptor accepted")
	}
}
