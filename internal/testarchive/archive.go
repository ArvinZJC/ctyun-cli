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
	"os"
	"path/filepath"
	"testing"
)

// WriteTarGzFromDir writes srcDir contents to archivePath as a gzipped tar
// archive.
func WriteTarGzFromDir(t *testing.T, archivePath, srcDir string) {
	t.Helper()
	WriteTarGzFromDirWithPrefix(t, archivePath, srcDir, "")
}

// WriteTarGzFromDirWithPrefix writes srcDir contents to archivePath as a
// gzipped tar archive, placing each entry under prefix when prefix is non-empty.
func WriteTarGzFromDirWithPrefix(t *testing.T, archivePath, srcDir, prefix string) {
	t.Helper()
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	if err := WriteTarEntries(tarWriter, srcDir, prefix); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
}

// WriteTarEntries writes all non-directory srcDir entries to tarWriter, placing
// each entry under prefix when prefix is non-empty.
func WriteTarEntries(tarWriter *tar.Writer, srcDir, prefix string) error {
	if err := filepath.WalkDir(srcDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
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
	}); err != nil {
		return err
	}
	return nil
}
