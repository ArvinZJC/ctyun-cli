/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
)

// TestRetentionRequiresVerifiedIndexes checks new and historical signature failures and legacy unsigned inputs.
func TestRetentionRequiresVerifiedIndexes(t *testing.T) {
	key := releaseTestKey()
	pub := base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
	stage, output := t.TempDir(), t.TempDir()
	sign := func(dir, stem, data string) {
		t.Helper()
		releaseTestFile(t, filepath.Join(dir, stem+".json"), data)
		releaseTestFile(t, filepath.Join(dir, stem+".sig"), base64.StdEncoding.EncodeToString(ed25519.Sign(key, []byte(data))))
	}
	sign(stage, "core-index", `{"schema":1,"releases":[]}`)
	sign(stage, "index", `{"plugins":[]}`)
	sign(stage, "gitee/index", `{"plugins":[]}`)
	missing := filepath.Join(output, "missing")
	if _, err := planArchiveRetention(stage, missing, pub); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing output: %v", err)
	}
	for _, stem := range []string{"core-index", "index", "gitee/index"} {
		t.Run(stem, func(t *testing.T) {
			file := filepath.Join(stage, stem+".sig")
			sig, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			releaseTestFile(t, file, "invalid")
			if _, err := planArchiveRetention(stage, output, pub); err == nil {
				t.Fatalf("accepted invalid %s signature", stem)
			}
			releaseTestFile(t, file, string(sig))
		})
	}
	for _, stem := range []string{"core-index", "index"} {
		t.Run("invalid historical "+stem, func(t *testing.T) {
			dir := t.TempDir()
			sign(dir, stem, "{")
			if _, err := planArchiveRetention(stage, dir, pub); err == nil {
				t.Fatal("accepted malformed signed index")
			}
			sig := filepath.Join(dir, stem+".sig")
			releaseTestFile(t, sig, "invalid")
			if _, err := planArchiveRetention(stage, dir, pub); err == nil {
				t.Fatal("accepted invalid historical signature")
			}
			if err := os.Remove(sig); err != nil {
				t.Fatal(err)
			}
			plan, err := planArchiveRetention(stage, dir, pub)
			if err != nil || len(plan.superseded) != 0 {
				t.Fatalf("unsigned history authorised removal: %v %v", plan, err)
			}
		})
	}
}

// TestRetirementRollback restores the previous upload set if a later superseded archive cannot be moved.
func TestRetirementRollback(t *testing.T) {
	output, stage := t.TempDir(), t.TempDir()
	releaseTestFile(t, filepath.Join(stage, "index.json"), "new")
	releaseTestFile(t, filepath.Join(output, "index.json"), "old")
	old := []string{"a.tar.gz", "b.tar.gz"}
	for _, name := range old {
		releaseTestFile(t, filepath.Join(output, name), name)
	}
	rename := renamePublicationFile
	t.Cleanup(func() { renamePublicationFile = rename })
	renamePublicationFile = func(from, to string) error {
		if from == filepath.Join(output, old[1]) {
			return io.ErrClosedPipe
		}
		return rename(from, to)
	}
	if err := publishReleaseFiles(stage, output, archiveRetention{superseded: old}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("retirement failure = %v", err)
	}
	for name, want := range map[string]string{"index.json": "old", old[0]: old[0], old[1]: old[1]} {
		raw, err := os.ReadFile(filepath.Join(output, name))
		if err != nil || string(raw) != want {
			t.Fatalf("%s after rollback = %q, %v", name, raw, err)
		}
	}
}

// setUpRetentionRelease provides a minimal project with version-specific binary contents.
func setUpRetentionRelease(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	releaseTestDir(t, "plugins")
	releaseTestFile(t, "install.sh", "shell")
	releaseTestFile(t, "install.ps1", "powershell")
	t.Cleanup(patchBuildBinary(func(opts buildOptions) error {
		return os.WriteFile(opts.Output, []byte(opts.Version), 0o755)
	}))
}

// TestReleaseRejectsUnverifiedHistoryBeforeReplacement exercises the normal packaging entry point.
func TestReleaseRejectsUnverifiedHistoryBeforeReplacement(t *testing.T) {
	setUpRetentionRelease(t)
	opts := releaseOptions{OutDir: t.TempDir(), Version: "1.0.0", Channel: "stable", Platforms: []string{"linux/amd64"}}
	if err := writeRelease(opts, releaseTestKey()); err != nil {
		t.Fatal(err)
	}
	releaseTestFile(t, filepath.Join(opts.OutDir, "index.sig"), "damaged")
	opts.Version = "2.0.0"
	if err := writeRelease(opts, releaseTestKey()); err == nil {
		t.Fatal("published over unverifiable history")
	}
	if _, err := os.Stat(filepath.Join(opts.OutDir, "ctyun_1.0.0_linux_amd64.tar.gz")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.OutDir, "ctyun_2.0.0_linux_amd64.tar.gz")); !os.IsNotExist(err) {
		t.Fatalf("unverified replacement on disk: %v", err)
	}
}

// TestReleaseRetiresOnlyHistoricalBytes protects repurposed archive names and requires a readable checksum.
func TestReleaseRetiresOnlyHistoricalBytes(t *testing.T) {
	setUpRetentionRelease(t)
	opts := releaseOptions{OutDir: t.TempDir(), Version: "1.0.0", Channel: "stable", Platforms: []string{"linux/amd64"}}
	if err := writeRelease(opts, releaseTestKey()); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(opts.OutDir, "ctyun_1.0.0_linux_amd64.tar.gz")
	releaseTestFile(t, old, "local replacement")
	original := checksumArchive
	t.Cleanup(func() { checksumArchive = original })
	checksumArchive = func(path string) (string, int64, error) {
		if path == old {
			return "", 0, io.ErrUnexpectedEOF
		}
		return original(path)
	}
	opts.Version = "2.0.0"
	if err := writeRelease(opts, releaseTestKey()); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("unreadable historical archive: %v", err)
	}
	checksumArchive = original
	if err := writeRelease(opts, releaseTestKey()); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(old); err != nil || string(data) != "local replacement" {
		t.Fatalf("repurposed archive after packaging: %q %v", data, err)
	}
}

// TestCaseSensitivePrereleasePublication checks both directions of a case-only version change.
func TestCaseSensitivePrereleasePublication(t *testing.T) {
	setUpRetentionRelease(t)
	opts := releaseOptions{OutDir: t.TempDir(), Channel: "alpha", Platforms: []string{"linux/amd64"}}
	key := releaseTestKey()
	publicKey := base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey))
	var newest []byte
	for i, suffix := range []string{"A", "a", "A"} {
		opts.Version = "1.0.0-rc." + suffix
		if err := writeRelease(opts, key); err != nil {
			t.Fatal(err)
		}
		data, err := corerelease.ReadSignedIndex(opts.OutDir, publicKey, nil)
		if err != nil {
			t.Fatal(err)
		}
		var index corerelease.Index
		if err := json.Unmarshal(data, &index); err != nil {
			t.Fatal(err)
		}
		rel, art, ok := index.FindLatest("alpha", "linux", "amd64")
		if !ok || i > 0 && rel.Version != "1.0.0-rc.a" {
			t.Fatalf("incorrect current release: %v", rel)
		}
		if err := distribution.VerifySHA256(filepath.Join(opts.OutDir, art.URL), art.SHA256); err != nil {
			t.Fatal(err)
		}
		archives, err := filepath.Glob(filepath.Join(opts.OutDir, "*.tar.gz"))
		if err != nil || len(archives) != 1 || filepath.Base(archives[0]) != art.URL {
			t.Fatalf("archives = %v, want %s: %v", archives, art.URL, err)
		}
		if i == 1 {
			newest = data
		}
		if i == 2 && !bytes.Equal(newest, data) {
			t.Fatal("older case variant displaced the signed release")
		}
	}
}

// TestLocalArchiveRetentionCasePolicies exercises case-sensitive and case-insensitive
// directory semantics without requiring two different volumes on the test host.
func TestLocalArchiveRetentionCasePolicies(t *testing.T) {
	stage, output := t.TempDir(), t.TempDir()
	upper, lower := "release-A.tar.gz", "release-a.tar.gz"
	releaseTestFile(t, filepath.Join(output, upper), "local")
	if _, err := localArchiveRetention(filepath.Join(stage, "missing"), output, nil, nil, true); err == nil {
		t.Fatal("accepted missing stage")
	}
	both := map[string]bool{upper: true, lower: true}
	if _, err := localArchiveRetention(stage, output, both, nil, true); err == nil {
		t.Fatal("accepted colliding current upload names")
	}
	if _, err := localArchiveRetention(stage, output, both, nil, false); err != nil {
		t.Fatal(err)
	}
	current := map[string]bool{lower: true}
	plan, err := localArchiveRetention(stage, output, current, nil, true)
	if err != nil || len(plan.superseded) != 0 {
		t.Fatalf("retired reused archive: %v, %v", plan, err)
	}
	releaseTestFile(t, filepath.Join(stage, lower), "new")
	if _, err := localArchiveRetention(stage, output, current, nil, true); err == nil {
		t.Fatal("overwrote unrelated case variant")
	}
	if _, err := localArchiveRetention(stage, output, current, nil, false); err != nil {
		t.Fatal(err)
	}
}

// TestCaseOnlyRollbackRestoresUploadNames fails after the new archive has been installed
// and checks the exact directory entry and the bytes needed to retry the old upload.
func TestCaseOnlyRollbackRestoresUploadNames(t *testing.T) {
	setUpRetentionRelease(t)
	opts := releaseOptions{OutDir: t.TempDir(), Version: "1.0.0-rc.A", Channel: "alpha", Platforms: []string{"linux/amd64"}}
	key := releaseTestKey()
	if err := writeRelease(opts, key); err != nil {
		t.Fatal(err)
	}
	oldName := "ctyun_1.0.0-rc.A_linux_amd64.tar.gz"
	prior, err := os.ReadFile(filepath.Join(opts.OutDir, oldName))
	if err != nil {
		t.Fatal(err)
	}
	rename := renamePublicationFile
	t.Cleanup(func() { renamePublicationFile = rename })
	renamePublicationFile = func(from, to string) error {
		if to == filepath.Join(opts.OutDir, "install.sh") && strings.HasPrefix(filepath.Base(filepath.Dir(from)), ".ctyun-release-") {
			return io.ErrClosedPipe
		}
		return rename(from, to)
	}
	opts.Version = "1.0.0-rc.a"
	if err := writeRelease(opts, key); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("publish error: %v", err)
	}
	archives, err := filepath.Glob(filepath.Join(opts.OutDir, "*.tar.gz"))
	if err != nil || len(archives) != 1 || filepath.Base(archives[0]) != oldName {
		t.Fatalf("rollback filenames: %v, %v", archives, err)
	}
	if content, err := os.ReadFile(archives[0]); err != nil || !bytes.Equal(content, prior) {
		t.Fatalf("rollback lost original archive bytes: %v", err)
	}
}
