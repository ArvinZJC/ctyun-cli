/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package output

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestWriteFilePreservesRacingDestination pins no-clobber publication.
func TestWriteFilePreservesRacingDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "download")
	err := WriteFile(path, false, func(w io.Writer) error {
		if err := os.WriteFile(path, []byte("other"), 0600); err != nil {
			return err
		}
		_, err := io.WriteString(w, "download")
		return err
	})
	if err == nil {
		t.Fatal("overwrote racing destination")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "other" {
		t.Fatalf("%q %v", got, err)
	}
}

// TestWriteFilePublicationAndFailures preserves existing files and cleans incomplete downloads.
func TestWriteFilePublicationAndFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	write := func(w io.Writer) error { _, err := io.WriteString(w, "complete"); return err }
	if err := WriteFile(path, false, write); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, false, write); err == nil {
		t.Fatal("existing destination accepted")
	}
	if err := WriteFile(path, true, write); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(dir, true, write); err == nil {
		t.Fatal("directory accepted")
	}
	if err := WriteFile(filepath.Join(dir, "missing", "file"), false, write); err == nil {
		t.Fatal("missing parent accepted")
	}
	if err := WriteFile(filepath.Join(path, "file"), false, write); err == nil {
		t.Fatal("invalid parent accepted")
	}
	link := filepath.Join(dir, "symlink")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(link, true, write); err == nil {
		t.Fatal("symlink accepted")
	}
	if err := WriteFile(path, true, func(io.Writer) error { return io.ErrUnexpectedEOF }); err != io.ErrUnexpectedEOF {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "complete" {
		t.Fatalf("%q %v", got, err)
	}
	if err := WriteFile(filepath.Join(dir, "closed"), false, func(w io.Writer) error { return w.(*os.File).Close() }); err == nil {
		t.Fatal("sync of closed file succeeded")
	}
	race := filepath.Join(dir, "race")
	if err := WriteFile(race, true, func(w io.Writer) error { return os.Mkdir(race, 0700) }); err == nil {
		t.Fatal("racing directory accepted")
	}
	files, err := filepath.Glob(filepath.Join(dir, ".ctyun-download-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary files %v %v", files, err)
	}
}

// TestWriteFileCloseAndRecheckFailures preserves incomplete and racing destinations.
func TestWriteFileCloseAndRecheckFailures(t *testing.T) {
	original := closeDownloadFile
	defer func() { closeDownloadFile = original }()
	closeDownloadFile = func(file *os.File) error { file.Close(); return io.ErrClosedPipe }
	path := filepath.Join(t.TempDir(), "destination")
	if err := WriteFile(path, false, func(io.Writer) error { return nil }); err != io.ErrClosedPipe {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("incomplete result published")
	}
	closeDownloadFile = original
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(parent, "destination")
	err := WriteFile(path, true, func(io.Writer) error {
		if err := os.Rename(parent, parent+"-moved"); err != nil {
			return err
		}
		return os.WriteFile(parent, []byte("replacement"), 0600)
	})
	if err == nil {
		t.Fatal("invalid destination parent accepted")
	}
}
