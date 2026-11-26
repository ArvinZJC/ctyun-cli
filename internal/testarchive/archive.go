/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package testarchive provides archive helpers for tests that need local plugin
// bundles.
package testarchive

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Reporter is the testing boundary used by archive helpers. A fatal diagnostic
// must stop the calling test, as testing.T.Fatal does.
type Reporter interface {
	Helper()
	Fatalf(string, ...any)
}

// WriteTarGzFromDir writes srcDir contents to archivePath as a gzipped tar archive.
func WriteTarGzFromDir(t Reporter, archivePath, srcDir string) {
	t.Helper()
	WriteTarGzFromDirWithPrefix(t, archivePath, srcDir, "")
}

// WriteTarGzFromDirWithPrefix writes a gzipped tar archive with an optional
// prefix, reporting failures only after closing the archive resources.
func WriteTarGzFromDirWithPrefix(t Reporter, archivePath, srcDir, prefix string) {
	t.Helper()
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
		return
	}
	if err := writeTarGz(file, srcDir, prefix); err != nil {
		t.Fatalf("write archive: %v", err)
	}
}

// writeTarGz owns the destination and closes every encoder even after a failed
// source read, retaining both the original failure and finalisation errors.
func writeTarGz(file io.WriteCloser, srcDir, prefix string) (err error) {
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		err = errors.Join(err, tarWriter.Close(), gzipWriter.Close(), file.Close())
	}()
	return WriteTarEntries(tarWriter, srcDir, prefix)
}

// WriteTarEntries writes all non-directory srcDir entries to tarWriter, placing
// each entry under prefix when prefix is non-empty.
func WriteTarEntries(tarWriter *tar.Writer, srcDir, prefix string) error {
	return filepath.WalkDir(srcDir, func(path string, entry os.DirEntry, err error) error {
		return writeTarEntry(tarWriter, srcDir, path, prefix, entry, err)
	})
}

// writeTarEntry preserves filesystem entry names and contents while propagating
// traversal, metadata, and stream errors to the archive owner.
func writeTarEntry(tarWriter *tar.Writer, srcDir, path, prefix string, entry os.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() {
		return nil
	}
	rel, err := filepath.Rel(srcDir, path)
	if err != nil {
		return err
	}
	if prefix != "" {
		rel = filepath.Join(prefix, rel)
	}
	info, err := entry.Info()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = rel
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = tarWriter.Write(data)
	return err
}
