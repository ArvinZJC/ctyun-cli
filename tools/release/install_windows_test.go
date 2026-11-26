/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"archive/tar"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestWindowsInstallerLiteralPaths downloads through Windows PowerShell into bracketed directories.
func TestWindowsInstallerLiteralPaths(t *testing.T) {
	root := t.TempDir()
	temp := filepath.Join(root, "temporary [1]")
	install := filepath.Join(root, "install [1]")
	if err := os.Mkdir(temp, 0o700); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(root, "source.exe")
	if err := os.WriteFile(fixture, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "core.tar.gz")
	if err := writeTarGzArchive(archive, func(w *tar.Writer) error {
		return addFileToArchive(w, fixture, "ctyun.exe", 0o755)
	}); err != nil {
		t.Fatal(err)
	}
	sum, _, err := checksumAndSize(archive)
	if err != nil {
		t.Fatal(err)
	}
	index := fmt.Sprintf(`{"releases":[{"channel":"stable","version":"1.0.0","artifacts":[{"os":"windows","arch":%q,"url":"core.tar.gz","sha256":%q}]}]}`, runtime.GOARCH, sum)
	if err := os.WriteFile(filepath.Join(root, "core-index.json"), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer server.Close()
	for _, name := range []string{"TEMP", "TMP", "SystemTemp"} {
		t.Setenv(name, temp)
	}
	t.Setenv("CTYUN_INSTALL_BASE_URL", server.URL)
	t.Setenv("CTYUN_INSTALL_CHANNEL", "stable")
	t.Setenv("CTYUN_INSTALL_SOURCE", "auto")
	t.Setenv("CTYUN_INSTALL_DIR", install)
	t.Setenv("PROCESSOR_ARCHITEW6432", runtime.GOARCH)
	script, err := projectFilePath("install.ps1")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install.ps1: %v\n%s", err, out)
	}
	if data, err := os.ReadFile(filepath.Join(install, "ctyun.exe")); err != nil || string(data) != "fixture" {
		t.Fatalf("installed binary = %q, %v", data, err)
	}
	if entries, err := os.ReadDir(temp); err != nil || len(entries) != 0 {
		t.Fatalf("temporary files after installation = %v, %v", entries, err)
	}
}
