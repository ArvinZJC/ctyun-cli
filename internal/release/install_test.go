/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExtractBinaryAcceptsSafeArchives(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry string
	}{
		{name: "direct", entry: "ctyun"},
		{name: "wrapped", entry: "ctyun_0.2.0/ctyun"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archive := writeTarGz(t, []tarEntry{{name: tc.entry, body: "new-binary"}})
			got, err := ExtractBinary(archive, t.TempDir(), "ctyun")
			if err != nil {
				t.Fatalf("ExtractBinary returned error: %v", err)
			}
			data, err := os.ReadFile(got)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "new-binary" {
				t.Fatalf("binary = %q, want new-binary", data)
			}
		})
	}
}

func TestExtractBinaryRejectsUnsafeArchives(t *testing.T) {
	tests := []struct {
		name    string
		entries []tarEntry
		want    string
	}{
		{name: "traversal", entries: []tarEntry{{name: "../ctyun", body: "bad"}}, want: "error.archive_path_escapes_destination"},
		{name: "symlink", entries: []tarEntry{{name: "ctyun", link: "target"}}, want: "error.unsupported_archive_entry"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			archive := writeTarGz(t, tc.entries)
			_, err := ExtractBinary(archive, t.TempDir(), "ctyun")
			if err == nil {
				t.Fatal("ExtractBinary returned nil error")
			}
			requireDiagnosticKey(t, err, tc.want)
		})
	}
}

func TestExtractBinaryRejectsInvalidOrIncompleteArchives(t *testing.T) {
	if _, err := ExtractBinary(filepath.Join(t.TempDir(), "missing.tar.gz"), t.TempDir(), "ctyun"); err == nil {
		t.Fatal("ExtractBinary returned nil error for missing archive")
	}
	badGzip := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := os.WriteFile(badGzip, []byte("not gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractBinary(badGzip, t.TempDir(), "ctyun"); err == nil {
		t.Fatal("ExtractBinary returned nil error for bad gzip")
	}
	archive := writeTarGz(t, []tarEntry{{name: "README.md", body: "docs"}})
	if _, err := ExtractBinary(archive, t.TempDir(), "ctyun"); err == nil {
		t.Fatalf("ExtractBinary missing binary error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.archive_missing_binary")
	}
	dirArchive := writeTarGz(t, []tarEntry{{name: "bin", dir: true}, {name: "bin/ctyun", body: "new"}})
	if _, err := ExtractBinary(dirArchive, t.TempDir(), "ctyun"); err != nil {
		t.Fatalf("ExtractBinary with directory returned error: %v", err)
	}
}

func TestInstallArtifactReplacesCurrentExecutable(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, binaryNameForTest())
	if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	archive := writeTarGz(t, []tarEntry{{name: binaryNameForTest(), body: "new"}})

	if err := InstallArtifact(InstallOptions{CurrentExecutable: current, ArchivePath: archive, BinaryName: binaryNameForTest()}); err != nil {
		t.Fatalf("InstallArtifact returned error: %v", err)
	}
	data, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("installed binary = %q, want new", data)
	}
}

func TestInstallArtifactRestoresOldBinaryOnRenameFailure(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, binaryNameForTest())
	if err := os.WriteFile(current, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	archive := writeTarGz(t, []tarEntry{{name: binaryNameForTest(), body: "new"}})
	failed := false
	restore := patchRename(func(oldPath, newPath string) error {
		if !failed && oldPath != current && newPath == current {
			failed = true
			return errors.New("replace failed")
		}
		return os.Rename(oldPath, newPath)
	})
	defer restore()

	err := InstallArtifact(InstallOptions{CurrentExecutable: current, ArchivePath: archive, BinaryName: binaryNameForTest()})
	if err == nil || err.Error() != "replace failed" {
		t.Fatalf("InstallArtifact error = %v, want replace failure", err)
	}
	data, readErr := os.ReadFile(current)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "old" {
		t.Fatalf("restored binary = %q, want old", data)
	}
}

func TestInstallArtifactRejectsInvalidInputs(t *testing.T) {
	if err := InstallArtifact(InstallOptions{}); err == nil {
		t.Fatal("InstallArtifact returned nil error without current executable")
	}
	if err := InstallArtifact(InstallOptions{CurrentExecutable: filepath.Join(t.TempDir(), "ctyun")}); err == nil {
		t.Fatal("InstallArtifact returned nil error without archive")
	}
	if err := InstallArtifact(InstallOptions{CurrentExecutable: filepath.Join(t.TempDir(), "ctyun"), ArchivePath: filepath.Join(t.TempDir(), "missing.tar.gz")}); err == nil {
		t.Fatal("InstallArtifact returned nil error for missing archive")
	}
}

func TestInstallArtifactPropagatesCurrentRenameFailure(t *testing.T) {
	current := filepath.Join(t.TempDir(), "missing-ctyun")
	archive := writeTarGz(t, []tarEntry{{name: binaryNameForTest(), body: "new"}})
	if err := InstallArtifact(InstallOptions{CurrentExecutable: current, ArchivePath: archive, BinaryName: binaryNameForTest()}); err == nil {
		t.Fatal("InstallArtifact returned nil error for missing current executable")
	}
}

func binaryNameForTest() string {
	if runtime.GOOS == "windows" {
		return "ctyun.exe"
	}
	return "ctyun"
}

type tarEntry struct {
	name string
	body string
	link string
	dir  bool
}

func writeTarGz(t *testing.T, entries []tarEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ctyun.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		if entry.dir {
			if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if entry.link != "" {
			if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Typeflag: tar.TypeSymlink, Linkname: entry.link}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		header := &tar.Header{Name: entry.name, Mode: 0o755, Size: int64(len(entry.body)), Typeflag: tar.TypeReg}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func patchRename(fn func(string, string) error) func() {
	original := renamePath
	renamePath = fn
	return func() {
		renamePath = original
	}
}
