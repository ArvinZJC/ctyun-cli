/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
)

// TestFirstConfigSavePublishesOnlyCompletedData checks readers, partial write failures, and a competing creator.
func TestFirstConfigSavePublishesOnlyCompletedData(t *testing.T) {
	for _, outcome := range []string{"save", "partial write", "competing save"} {
		t.Run(outcome, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "settings.json")
			realWrite := writeConfigAt
			t.Cleanup(func() { writeConfigAt = realWrite })
			writeConfigAt = func(file *os.File, data []byte, offset int64) (int, error) {
				writeConfigAt = realWrite
				raw, err := loadConfigBytes(nil, path)
				if err != nil || raw != nil {
					t.Errorf("reader before first publication: bytes=%q, err=%v", raw, err)
				}
				if outcome == "partial write" {
					n, err := realWrite(file, data[:4], offset)
					return n, errors.Join(err, io.ErrShortWrite)
				}
				if outcome == "competing save" {
					err := Run(Config{ConfigPath: path, Stdout: io.Discard, Args: []string{"config", "profile", "set", "winner", "region", "other"}})
					if err != nil {
						t.Errorf("competing creator: %v", err)
					}
				}
				return realWrite(file, data, offset)
			}
			err := Run(Config{ConfigPath: path, Stdout: io.Discard, Args: []string{"config", "profile", "set", "initial", "region", "first"}})
			switch outcome {
			case "partial write":
				if !errors.Is(err, io.ErrShortWrite) {
					t.Errorf("save error = %v, want short write", err)
				}
				entries, readErr := os.ReadDir(dir)
				if readErr != nil || len(entries) != 0 {
					t.Errorf("failed first save left files: %v, %v", entries, readErr)
				}
			case "competing save":
				if err == nil || !strings.Contains(err.Error(), errConfigChanged.Error()) {
					t.Errorf("stale first creation = %v, want conflict", err)
				}
				cfg := readConfigForTest(t, path)
				if len(cfg.Profiles) != 1 || cfg.Profiles["winner"].Region != "other" {
					t.Errorf("lost competing save: %#v", cfg.Profiles)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				if cfg := readConfigForTest(t, path); cfg.Profiles["initial"].Region != "first" {
					t.Error("published config lacks the selected profile")
				}
			}
		})
	}
}

// TestFirstConfigSaveWithoutExclusiveRename checks no-overwrite publication on a filesystem using hard links.
func TestFirstConfigSaveWithoutExclusiveRename(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	oldMove := moveNewConfig
	t.Cleanup(func() { moveNewConfig = oldMove })
	moveNewConfig = func(string, string) error { return errors.ErrUnsupported }
	if err := writeConfigFile("settings.json", coreconfig.Config{}, nil); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, "settings.json", "concurrent")
	err := writeConfigFile("settings.json", coreconfig.Config{}, nil)
	if err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("concurrent creation = %v, want a conflict", err)
	}
	data, err := os.ReadFile("settings.json")
	if err != nil || string(data) != "concurrent" {
		t.Fatalf("concurrent file = %q, %v", data, err)
	}
}

// TestFirstConfigSaveTracksDanglingLinks checks that failure or retargeting preserves the selected symlink.
func TestFirstConfigSaveTracksDanglingLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unprivileged symlink creation")
	}
	dir := t.TempDir()
	link := filepath.Join(dir, "settings.json")
	target := filepath.Join(dir, "target")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	oldWrite := writeConfigAt
	t.Cleanup(func() { writeConfigAt = oldWrite })
	writeConfigAt = func(*os.File, []byte, int64) (int, error) { return 0, io.ErrShortWrite }
	if err := writeConfigFile(link, coreconfig.Config{}, nil); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("failed first save: %v", err)
	}
	if actual, err := os.Readlink(link); err != nil || actual != target {
		t.Fatalf("selected link = %q, %v", actual, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("failed save published a target: %v", err)
	}
	other := filepath.Join(dir, "other")
	writeConfigAt = func(file *os.File, data []byte, offset int64) (int, error) {
		if err := os.Remove(link); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(other, link); err != nil {
			t.Fatal(err)
		}
		return oldWrite(file, data, offset)
	}
	err := writeConfigFile(link, coreconfig.Config{}, nil)
	if err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("redirected first save = %v, want a conflict", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(link) {
		t.Fatalf("redirected creation left files: %v, %v", entries, err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-parent/target", link); err != nil {
		t.Fatal(err)
	}
	if err := writeConfigFile(link, coreconfig.Config{}, nil); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing target directory = %v", err)
	}
}

// TestConfigSetPreservesOriginalAfterPartialWrite simulates a disk filling during an update.
func TestConfigSetPreservesOriginalAfterPartialWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	original := `{"profiles":{"test":{"region":"before"}}}` + "\n"
	mustWrite(t, path, original)
	oldWrite := writeConfigAt
	t.Cleanup(func() { writeConfigAt = oldWrite })
	failure := errors.New("injected partial write")
	for _, limit := range []int{10, len(original) + 4} {
		writeConfigAt = func(file *os.File, data []byte, offset int64) (int, error) {
			writeConfigAt = oldWrite
			_, err := file.WriteAt(data[:limit], offset)
			return 0, errors.Join(err, failure)
		}
		var stdout bytes.Buffer
		err := Run(Config{ConfigPath: path, Stdout: &stdout, Args: []string{"config", "profile", "set", "test", "region", strings.Repeat("new", 30)}})
		if !errors.Is(err, failure) || stdout.Len() != 0 {
			t.Fatalf("config set after %d bytes = %v, stdout %q", limit, err, stdout.String())
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != original {
			t.Fatalf("config after %d bytes = %q, %v", limit, got, err)
		}
	}
}

// TestReplaceConfigContentsReportsRecoveryErrors retains both the save failure and any rollback failure.
func TestReplaceConfigContentsReportsRecoveryErrors(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "config")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte("original"), 0); err != nil {
		t.Fatal(err)
	}
	oldWrite := writeConfigAt
	t.Cleanup(func() { writeConfigAt = oldWrite })
	restoreFailure := errors.New("restore failed")
	writeConfigAt = func(f *os.File, data []byte, offset int64) (int, error) {
		n, err := f.WriteAt(data, offset)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		writeConfigAt = func(*os.File, []byte, int64) (int, error) { return 0, restoreFailure }
		return n, nil
	}
	if err := replaceConfigContents(file, []byte("{}"), []byte("original")); !errors.Is(err, os.ErrClosed) || !errors.Is(err, restoreFailure) {
		t.Fatalf("failed truncate and restore = %v", err)
	}

}

// TestConfigSetRetainsFileIdentity verifies links, file permissions, and write protection.
func TestConfigSetRetainsFileIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes and unprivileged symlink creation")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	runConfigForTest(t, path, nil, "config", "profile", "set", "work", "region", "before")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("new config mode = %v, %v", info, err)
	}
	link := filepath.Join(dir, "selected.json")
	alias := filepath.Join(dir, "other.json")
	if err := os.Symlink(filepath.Base(path), link); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, alias); err != nil {
		t.Fatal(err)
	}
	runConfigForTest(t, link, nil, "config", "profile", "set", "work", "region", "ok")
	if readConfigForTest(t, alias).Profiles["work"].Region != "ok" {
		t.Fatal("hard link did not see the shorter config written via the symlink")
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	probe, openErr := os.OpenFile(link, os.O_RDWR, 0)
	if openErr == nil {
		if err := probe.Close(); err != nil {
			t.Fatal(err)
		}
	}
	err = Run(Config{ConfigPath: link, Args: []string{"config", "profile", "set", "work", "region", "new"}})
	if (err == nil) != (openErr == nil) {
		t.Fatalf("config update through read-only link: %v; opening target for writing: %v", err, openErr)
	}
}

func TestConfigSetWritesOnlyConfiguredProfileValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	runConfigForTest(t, configPath, nil, "config", "set", "region", "bb9fdb42056f11eda1610242ac110002", "--profile", "huadong1")

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "profiles": {
    "huadong1": {
      "region": "bb9fdb42056f11eda1610242ac110002"
    }
  }
}
`
	if string(raw) != want {
		t.Fatalf("config JSON =\n%s\nwant:\n%s", raw, want)
	}
}

// TestConfigMutationKeepsInterveningProfile rejects stale changes after another command succeeds during input.
func TestConfigMutationKeepsInterveningProfile(t *testing.T) {
	for _, args := range [][]string{{"config", "profile", "set-secret", "old", "ak", "--from-stdin"}, {"config", "reset"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			mustWrite(t, path, `{}`)
			input := configInputAction(func() {
				runConfigForTest(t, path, nil, "config", "profile", "set", "other", "region", "new")
			})
			err := Run(Config{Args: args, ConfigPath: path, Stdin: &input, Stdout: io.Discard, Stderr: io.Discard})
			if err == nil || !strings.Contains(err.Error(), "error.config_changed") {
				t.Fatalf("stale mutation returned %v", err)
			}
			if profile := readConfigForTest(t, path).Profiles["other"]; profile.Region != "new" {
				t.Fatalf("independent profile was lost: %#v", profile)
			}
		})
	}
}

// configInputAction runs one concurrent command after the first command has loaded its configuration.
type configInputAction func()

// Read supplies input and EOF after the intervening change has completed.
func (action *configInputAction) Read(p []byte) (int, error) {
	(*action)()
	return copy(p, "yes\n"), io.EOF
}

// TestConfigLocksProtectAliasesAndReleaseOnClose checks shared reads, write contention,
// dangling-link creation, and conflicts through different names for the same config.
func TestConfigLocksProtectAliasesAndReleaseOnClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	alias := filepath.Join(dir, "alias.json")
	mustWrite(t, path, "{}")
	if err := os.Link(path, alias); err != nil {
		t.Fatal(err)
	}
	checkContended := func(file *os.File) error {
		if _, err := loadConfigBytes(nil, alias); err != nil {
			t.Fatal(err)
		}
		if err := writeConfigFile(alias, coreconfig.Config{}, []byte("{}")); err == nil || err.Error() != errConfigChanged.Error() {
			t.Fatalf("write through an alias with a shared lock: %v", err)
		}
		return nil
	}
	if err := withLockedConfig(path, os.O_RDONLY, false, checkContended); err != nil {
		t.Fatal(err)
	}
	if err := writeConfigFile(alias, coreconfig.Config{}, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	link := filepath.Join(dir, "link.json")
	missing := filepath.Join(dir, "new.json")
	if err := os.Symlink(filepath.Base(missing), link); err != nil {
		t.Fatal(err)
	}
	if err := writeConfigFile(link, coreconfig.Config{}, nil); err != nil {
		t.Fatalf("write through dangling link: %v", err)
	}
	if err := writeConfigFile(link, coreconfig.Config{}, nil); err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("stale creation through a link: %v", err)
	}
}

// TestConfigSnapshotChecksFileIdentity rejects a closed handle and a path redirected to identical bytes.
func TestConfigSnapshotChecksFileIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	other := filepath.Join(dir, "other.json")
	mustWrite(t, path, "{}")
	mustWrite(t, other, "{}")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkConfigSnapshot(file, other, []byte("{}")); err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("replaced config path: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := checkConfigSnapshot(file, path, []byte("{}")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("closed config handle: %v", err)
	}
	err = withConfigHandleLock(file, false, func(closed *os.File) error {
		_, statErr := closed.Stat()
		return statErr
	})
	if err == nil {
		t.Fatal("locked a closed file")
	}
}

// TestResetKeepsChangesDuringBackup checks path removal, empty-file creation, and
// an external edit made after a successful backup and before removal.
func TestResetKeepsChangesDuringBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	reset := func(expected []byte) error {
		return runConfigReset(io.Discard, io.Discard, nil, expected, path, globalOptions{Yes: true})
	}
	if err := writeConfigFile(path, coreconfig.Config{}, []byte("{}")); err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("write after reset: %v", err)
	}
	if err := reset([]byte("{}")); err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("concurrently removed config: %v", err)
	}
	mustWrite(t, path, "")
	if err := reset(nil); err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("new empty config: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.Size() != 0 {
		t.Fatalf("new empty config changed: %v %v", info, err)
	}
	oldBackup := writeConfigBackup
	t.Cleanup(func() { writeConfigBackup = oldBackup })
	writeConfigBackup = func(file *os.File, data []byte) error {
		return errors.Join(io.ErrShortWrite, file.Close())
	}
	if err := reset([]byte{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("failed reset backup: %v", err)
	}
	writeConfigBackup = func(file *os.File, data []byte) error {
		mustWrite(t, path, "{}")
		return oldBackup(file, data)
	}
	if err := reset([]byte{}); err == nil || err.Error() != errConfigChanged.Error() {
		t.Fatalf("edit during backup: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "{}" {
		t.Fatalf("edited file = %q, %v", raw, err)
	}
}
