/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestPluginArchivePreservesEquivalentCompressedBytes protects immutable archive checksums across compressor changes.
func TestPluginArchivePreservesEquivalentCompressedBytes(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(source, "plugin.json")
	if err := os.WriteFile(file, []byte(`{"version":"0.1.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "plugin.tar.gz")
	if err := writeDirectoryArchive(archive, source, "example"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	var previous bytes.Buffer
	writer := gzip.NewWriter(&previous)
	writer.Name = "previous-toolchain"
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, previous.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeDirectoryArchive(archive, source, "example"); err != nil {
		t.Fatal(err)
	}
	retained, err := os.ReadFile(archive)
	if err != nil || !bytes.Equal(retained, previous.Bytes()) {
		t.Fatalf("equivalent published archive was rewritten: %v", err)
	}
	if err := os.WriteFile(file, []byte(`{"version":"0.2.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeDirectoryArchive(archive, source, "example"); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(archive)
	if err != nil || bytes.Equal(updated, retained) {
		t.Fatalf("changed bundle archive was not rebuilt: %v", err)
	}
}
