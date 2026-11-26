/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package testarchive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// archiveSource creates nested source files and a directory that must be skipped.
func archiveSource(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "file"), []byte("contents"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestArchiveRoundTrip checks paths and exact file bytes for prefixed and plain archives.
func TestArchiveRoundTrip(t *testing.T) {
	src := archiveSource(t)
	for _, prefix := range []string{"", "bundle"} {
		path := filepath.Join(t.TempDir(), "bundle.tar.gz")
		if prefix == "" {
			WriteTarGzFromDir(t, path, src)
		} else {
			WriteTarGzFromDirWithPrefix(t, path, src, prefix)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = file.Close() })
		gz, err := gzip.NewReader(file)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = gz.Close() })
		tr := tar.NewReader(gz)
		header, err := tr.Next()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		if header.Name != filepath.Join(prefix, "nested", "file") || string(data) != "contents" {
			t.Fatalf("header=%+v data=%q", header, data)
		}
		if _, err := tr.Next(); !errors.Is(err, io.EOF) {
			t.Fatalf("extra entry: %v", err)
		}
	}
}

// panicReporter records fatal helper diagnostics without terminating the parent test.
type panicReporter struct {
	message string
	helpers int
}

// Helper records that exported helpers identify themselves to the testing reporter.
func (r *panicReporter) Helper() { r.helpers++ }

// Fatalf simulates testing.T termination so resources must be closed by defers.
func (r *panicReporter) Fatalf(format string, args ...any) {
	r.message = fmt.Sprintf(format, args...)
	panic(r)
}

// TestArchiveFatalErrors reports creation and source errors through the testing boundary.
func TestArchiveFatalErrors(t *testing.T) {
	for _, tc := range []struct {
		name, src   string
		invalidPath bool
		want        string
	}{{"create", archiveSource(t), true, "create archive"}, {"source", filepath.Join(t.TempDir(), "missing"), false, "write archive"}} {
		t.Run(tc.name, func(t *testing.T) {
			r := &panicReporter{}
			path := filepath.Join(t.TempDir(), "archive.gz")
			if tc.invalidPath {
				path = filepath.Join(path, "missing.gz")
			}
			func() {
				defer func() {
					if recovered := recover(); recovered != r {
						t.Fatalf("panic=%v", recovered)
					}
				}()
				WriteTarGzFromDir(r, path, tc.src)
			}()
			if r.helpers < 2 || !strings.Contains(r.message, tc.want) {
				t.Fatalf("reporter=%+v", r)
			}
		})
	}
}

// trackedArchiveWriter injects real stream write/close failures and records cleanup.
type trackedArchiveWriter struct {
	bytes.Buffer
	calls, failAt int
	closeErr      error
	closed        bool
}

// Write rejects a selected write while retaining successful compressed bytes.
func (w *trackedArchiveWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.failAt == w.calls {
		return 0, io.ErrClosedPipe
	}
	return w.Buffer.Write(p)
}

// Close records ownership release and propagates an optional storage failure.
func (w *trackedArchiveWriter) Close() error { w.closed = true; return w.closeErr }

// TestArchiveStreamFailures verifies failed encoding and finalisation always close the output.
func TestArchiveStreamFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		failAt   int
		closeErr error
		missing  bool
	}{{"header", 1, nil, false}, {"gzip finalisation", 2, nil, false}, {"file close", 0, io.ErrClosedPipe, false}, {"source", 0, nil, true}} {
		t.Run(tc.name, func(t *testing.T) {
			src := archiveSource(t)
			if tc.missing {
				src = filepath.Join(src, "missing")
			}
			writer := &trackedArchiveWriter{failAt: tc.failAt, closeErr: tc.closeErr}
			err := writeTarGz(writer, src, "")
			if !writer.closed || err == nil {
				t.Fatalf("closed=%v error=%v", writer.closed, err)
			}
			if !tc.missing && !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("lost write/close failure: %v", err)
			}
			if tc.missing && !os.IsNotExist(err) && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("lost source failure: %v", err)
			}
		})
	}
}

// entryFailure retains real entry behavior except for a deliberately failed metadata read.
type entryFailure struct {
	fs.DirEntry
	info fs.FileInfo
	err  error
}

// Info supplies an explicit filesystem metadata result.
func (e entryFailure) Info() (fs.FileInfo, error) { return e.info, e.err }

// unsupportedInfo represents a file type unsupported by the tar format.
type unsupportedInfo struct{ fs.FileInfo }

// Mode identifies an irregular entry instead of a regular source file.
func (unsupportedInfo) Mode() fs.FileMode { return fs.ModeIrregular }

// truncatedInfo simulates a source growing after its metadata was read.
type truncatedInfo struct{ fs.FileInfo }

// Size declares the earlier empty source length.
func (truncatedInfo) Size() int64 { return 0 }

// TestArchiveEntryFailures exercises walk, relative-path, metadata, header, and content errors.
func TestArchiveEntryFailures(t *testing.T) {
	src := archiveSource(t)
	path := filepath.Join(src, "nested", "file")
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	entry := entries[0]
	info, err := entry.Info()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, base, path string
		entry            fs.DirEntry
		walkErr          error
		closed           bool
	}{{"walk", src, path, nil, io.ErrUnexpectedEOF, false}, {"relative", ".", path, entry, nil, false}, {"metadata", src, path, entryFailure{DirEntry: entry, err: io.ErrUnexpectedEOF}, nil, false}, {"unsupported", src, path, entryFailure{DirEntry: entry, info: unsupportedInfo{info}}, nil, false}, {"header", src, path, entry, nil, true}, {"read", src, filepath.Join(src, "missing"), entry, nil, false}, {"body", src, path, entryFailure{DirEntry: entry, info: truncatedInfo{info}}, nil, false}} {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := tar.NewWriter(&buffer)
			if tc.closed {
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
			}
			err := writeTarEntry(writer, tc.base, tc.path, "", tc.entry, tc.walkErr)
			if err == nil {
				t.Fatal("entry failure was lost")
			}
			if tc.name == "body" && !errors.Is(err, tar.ErrWriteTooLong) {
				t.Fatalf("body write error=%v", err)
			}
			if (tc.name == "walk" || tc.name == "metadata") && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("filesystem error=%v", err)
			}
			if tc.name == "header" && !errors.Is(err, tar.ErrWriteAfterClose) {
				t.Fatalf("header error=%v", err)
			}
			if tc.name == "read" && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("read error=%v", err)
			}
		})
	}
}
