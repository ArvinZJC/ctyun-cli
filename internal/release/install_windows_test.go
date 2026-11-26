/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestWindowsUpgradeOfRunningExecutable runs the installer inside the exact image it replaces.
func TestWindowsUpgradeOfRunningExecutable(t *testing.T) {
	current, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if archive := os.Getenv("CTYUN_TEST_RUNNING_UPGRADE_ARCHIVE"); archive != "" {
		if err := InstallArtifact(InstallOptions{CurrentExecutable: current, ArchivePath: archive, BinaryName: "ctyun.exe"}); err != nil {
			t.Fatal(err)
		}
		fmt.Println("installed")
		if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
			t.Fatal(err)
		}
		return
	}
	original, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "ctyun.exe")
	if err := os.WriteFile(target, original, 0o755); err != nil {
		t.Fatal(err)
	}
	archive := writeTarGz(t, []tarEntry{{name: "ctyun.exe", body: "new release"}})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, target, "-test.run=^TestWindowsUpgradeOfRunningExecutable$")
	command.Env = append(os.Environ(), "CTYUN_TEST_RUNNING_UPGRADE_ARCHIVE="+archive)
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if command.ProcessState == nil {
			cancel()
			_ = command.Wait()
		}
	}()
	line, err := bufio.NewReader(output).ReadString('\n')
	if err != nil || line != "installed\n" {
		t.Fatalf("self-upgrade: %q, %v", line, err)
	}
	next := InstallOptions{CurrentExecutable: target, ArchivePath: writeTarGz(t, []tarEntry{{name: "ctyun.exe", body: "third release"}})}
	if err := InstallArtifact(next); err == nil {
		t.Fatal("replaced the current binary while the previous executable was still running")
	}
	backupDir := filepath.Join(dir, ".ctyun.exe.previous")
	backups, err := os.ReadDir(backupDir)
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %v, %v", backups, err)
	}
	previous, err := os.ReadFile(filepath.Join(backupDir, backups[0].Name()))
	if err != nil || !bytes.Equal(original, previous) {
		t.Fatalf("original image was not retained: %v", err)
	}
	updated, err := os.ReadFile(target)
	if err != nil || string(updated) != "new release" {
		t.Fatalf("updated image = %q, %v", updated, err)
	}
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := InstallArtifact(next); err != nil {
		t.Fatalf("retry after previous process exited: %v", err)
	}
}

// TestWindowsUpgradeRotatesOnlyItsPreviousImage checks busy, modified and unrelated backup files before replacement.
func TestWindowsUpgradeRotatesOnlyItsPreviousImage(t *testing.T) {
	for _, condition := range []string{"rotate", "busy", "modified", "unrelated", "extra", "directory collision"} {
		t.Run(condition, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "custom.exe")
			backupDir := filepath.Join(dir, ".custom.exe.previous")
			if err := os.WriteFile(target, []byte("first"), 0o755); err != nil {
				t.Fatal(err)
			}
			install := func(content string) error {
				return InstallArtifact(InstallOptions{CurrentExecutable: target, BinaryName: "ctyun.exe", ArchivePath: writeTarGz(t, []tarEntry{{name: "ctyun.exe", body: content}})})
			}
			if condition == "directory collision" {
				if err := os.WriteFile(backupDir, []byte("user file"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := install("second"); err == nil {
					t.Fatal("unrelated destination accepted")
				}
				if data, err := os.ReadFile(backupDir); err != nil || string(data) != "user file" {
					t.Fatalf("unrelated file changed: %q, %v", data, err)
				}
				return
			}
			if err := install("second"); err != nil {
				t.Fatal(err)
			}
			previous := filepath.Join(backupDir, fmt.Sprintf("ctyun-updater-v1-%x.exe", sha256.Sum256([]byte("first"))))
			original := "first"
			switch condition {
			case "busy":
				f, err := os.Open(previous)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = f.Close() })
			case "modified":
				original = "edited by user"
				if err := os.WriteFile(previous, []byte(original), 0o600); err != nil {
					t.Fatal(err)
				}
			case "unrelated":
				other := filepath.Join(backupDir, "user.exe")
				if err := os.Rename(previous, other); err != nil {
					t.Fatal(err)
				}
				previous = other
			case "extra":
				if err := os.WriteFile(filepath.Join(backupDir, "notes.txt"), []byte("keep"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := install("third")
			want := "second"
			if condition == "rotate" {
				want = "third"
				if err != nil {
					t.Fatal(err)
				}
				entries, err := os.ReadDir(backupDir)
				if err != nil || len(entries) != 1 {
					t.Fatalf("retained %v backups: %v", entries, err)
				}
				previous = filepath.Join(backupDir, entries[0].Name())
				original = "second"
			} else if err == nil {
				t.Fatal("replaced the current binary despite an occupied backup")
			}
			if data, err := os.ReadFile(previous); err != nil || string(data) != original {
				t.Fatalf("previous=%q, want %q: %v", data, original, err)
			}
			if data, err := os.ReadFile(target); err != nil || string(data) != want {
				t.Fatalf("current=%q, want %q: %v", data, want, err)
			}
		})
	}
}
