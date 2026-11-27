/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
)

// TestConcurrentPublishersPreserveChannelsAndPlatforms interleaves an indexed platform rebuild with a new channel.
func TestConcurrentPublishersPreserveChannelsAndPlatforms(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	releaseTestDir(t, filepath.Join(root, "plugins"))
	releaseTestFile(t, filepath.Join(root, "install.sh"), "sh")
	releaseTestFile(t, filepath.Join(root, "install.ps1"), "ps")
	defer patchBuildBinary(func(opts buildOptions) error {
		return os.WriteFile(opts.Output, []byte(opts.GOOS+opts.GOARCH+opts.Version), 0o755)
	})()
	stable := releaseOptions{OutDir: t.TempDir(), Version: "0.5.2", Channel: "stable", Platforms: []string{"darwin/amd64"}}
	key := releaseTestKey()
	if err := writeRelease(stable, key); err != nil {
		t.Fatal(err)
	}
	stable.Platforms = []string{"darwin/arm64"}
	beta := releaseOptions{OutDir: stable.OutDir, Version: "0.6.0-beta.1", Channel: "beta", Platforms: []string{"linux/amd64"}}
	readingStable, resume := make(chan struct{}), make(chan struct{})
	encode := encodeReleaseJSON
	t.Cleanup(func() { encodeReleaseJSON = encode })
	encodeReleaseJSON = func(value any) ([]byte, error) {
		if index, ok := value.(corePublicationIndex); ok && len(index.Releases) == 1 {
			close(readingStable)
			<-resume
		}
		return encode(value)
	}
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { first <- writeRelease(stable, key) }()
	<-readingStable
	go func() { second <- writeRelease(beta, key) }()
	var betaErr error
	select {
	case betaErr = <-second:
		close(resume)
	case <-time.After(100 * time.Millisecond):
		close(resume)
		betaErr = <-second
	}
	if stableErr := <-first; betaErr != nil || stableErr != nil {
		t.Fatalf("stable=%v beta=%v", stableErr, betaErr)
	}
	publicKey := base64.StdEncoding.EncodeToString(key[32:])
	raw, err := corerelease.ReadSignedIndex(stable.OutDir, publicKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	index, err := corerelease.LoadIndex(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range [][3]string{{"stable", "darwin", "amd64"}, {"stable", "darwin", "arm64"}, {"beta", "linux", "amd64"}} {
		_, artifact, ok := index.FindLatest(target[0], target[1], target[2])
		if !ok {
			t.Fatalf("successful concurrent publication lost %v", target)
		}
		if err := distribution.VerifySHA256(filepath.Join(stable.OutDir, artifact.URL), artifact.SHA256); err != nil {
			t.Fatal(err)
		}
	}
}

// TestPublicationRollsBackFiles covers failed backup, replacement and restoration moves.
func TestPublicationRollsBackFiles(t *testing.T) {
	for _, fail := range []string{"backup", "replace", "restore", "directory"} {
		t.Run(fail, func(t *testing.T) {
			stage, output := t.TempDir(), t.TempDir()
			for _, name := range []string{"a", "b", "c"} {
				releaseTestFile(t, filepath.Join(stage, name), "new "+name)
			}
			releaseTestFile(t, filepath.Join(output, "a"), "original a")
			target := filepath.Join(output, "c")
			if fail == "directory" {
				releaseTestDir(t, target)
			} else {
				releaseTestFile(t, target, "original c")
			}
			original := renamePublicationFile
			t.Cleanup(func() { renamePublicationFile = original })
			renamePublicationFile = func(from, to string) error {
				switch {
				case fail == "backup" && from == target,
					(fail == "replace" || fail == "restore") && from == filepath.Join(stage, "c"):
					return io.ErrClosedPipe
				case fail == "restore" && to == target:
					return io.ErrUnexpectedEOF
				}
				return os.Rename(from, to)
			}
			err := publishReleaseFiles(stage, output, archiveRetention{})
			if err == nil || fail != "directory" && !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("unexpected publication result: %v", err)
			}
			for _, name := range []string{"a", "c"} {
				p := filepath.Join(output, name)
				if name == "c" {
					if fail == "directory" {
						info, err := os.Stat(target)
						if err != nil || !info.IsDir() {
							t.Fatalf("existing directory changed: %v", err)
						}
						continue
					}
					if fail == "restore" {
						if !errors.Is(err, io.ErrUnexpectedEOF) {
							t.Fatalf("missing rollback error: %v", err)
						}
						backups, _ := filepath.Glob(filepath.Join(output, ".ctyun-previous-*", "c"))
						if len(backups) != 1 || !strings.Contains(err.Error(), filepath.Dir(backups[0])) {
							t.Fatalf("no recoverable backup: %v, %v", backups, err)
						}
						p = backups[0]
					}
				}
				raw, err := os.ReadFile(p)
				if err != nil || string(raw) != "original "+name {
					t.Fatalf("original %s not recoverable: %q %v", name, raw, err)
				}
			}
			if _, err := os.Stat(filepath.Join(output, "b")); !os.IsNotExist(err) {
				t.Fatalf("new artifact survived rollback: %v", err)
			}
		})
	}
}

// TestPublicationStorageFailures checks failures creating staging, reading files or traversing a destination.
func TestPublicationStorageFailures(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := publishReleaseFiles(t.TempDir(), missing, archiveRetention{}); err == nil {
		t.Fatal("created backups without output directory")
	}
	if err := publishReleaseFiles(missing, t.TempDir(), archiveRetention{}); err == nil {
		t.Fatal("published missing stage")
	}
	t.Run("staging", func(t *testing.T) {
		original := createReleaseStage
		t.Cleanup(func() { createReleaseStage = original })
		createReleaseStage = func(string, string) (string, error) { return "", os.ErrPermission }
		if err := writeRelease(releaseOptions{OutDir: t.TempDir()}, releaseTestKey()); !errors.Is(err, os.ErrPermission) {
			t.Fatalf("staging error = %v", err)
		}
	})
	t.Run("destination directory replaced", func(t *testing.T) {
		stage, output := t.TempDir(), t.TempDir()
		releaseTestFile(t, filepath.Join(stage, "gitee", "index.json"), "{}")
		releaseTestFile(t, filepath.Join(stage, "gitee", "index.sig"), "signature")
		dir := filepath.Join(output, "gitee")
		original := renamePublicationFile
		t.Cleanup(func() { renamePublicationFile = original })
		renamePublicationFile = func(from, to string) error {
			if err := os.Rename(from, to); err != nil {
				return err
			}
			if to == filepath.Join(dir, "index.json") {
				// Simulate an external directory replacement between the index and signature moves.
				if err := os.Rename(dir, dir+"-moved"); err != nil {
					t.Fatal(err)
				}
				releaseTestFile(t, dir, "external file")
			}
			return nil
		}
		if err := publishReleaseFiles(stage, output, archiveRetention{}); err == nil {
			t.Fatal("ignored replaced destination directory")
		}
	})
	t.Run("signature", func(t *testing.T) {
		stage := t.TempDir()
		releaseTestDir(t, filepath.Join(stage, "core-index.sig"))
		if err := writeReleaseMetadata(releaseOptions{OutDir: stage}, releaseTestKey(), t.TempDir(), corerelease.Index{}); err == nil {
			t.Fatal("wrote signature over existing directory")
		}
	})
}
