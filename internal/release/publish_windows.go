/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/dirlock"
)

// previousImagePrefix identifies the content-addressed Windows updater backup.
const previousImagePrefix = "ctyun-updater-v1-"

// publishExecutable retains one previous image because Windows can rename a running
// executable but cannot unlink it. The next upgrade clears its slot before publication.
func publishExecutable(current, staged string, renamePath func(string, string) error) error {
	parent := filepath.Dir(current)
	return dirlock.With(parent, true, func() error {
		name, err := previousImageName(current)
		if err != nil {
			return err
		}
		dir := filepath.Join(parent, "."+filepath.Base(current)+".previous")
		err = os.Mkdir(dir, 0o700)
		created := err == nil
		if !created {
			if !errors.Is(err, fs.ErrExist) {
				return err
			}
			if err := clearPreviousImage(dir); err != nil {
				return err
			}
		}
		err = replaceWithBackup(current, staged, filepath.Join(dir, name), renamePath)
		if created && err != nil {
			// Remove the empty directory only when no recovery image remains.
			_ = os.Remove(dir)
		}
		return err
	})
}

// clearPreviousImage removes only the regular file identified by its original digest.
func clearPreviousImage(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return diagnostic.New("error.upgrade_backup_conflict", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return err
	}
	if len(entries) != 1 || !entries[0].Type().IsRegular() || !strings.HasPrefix(entries[0].Name(), previousImagePrefix) {
		return diagnostic.New("error.upgrade_backup_conflict", dir)
	}
	previous := filepath.Join(dir, entries[0].Name())
	name, err := previousImageName(previous)
	if err != nil {
		return err
	}
	if name != entries[0].Name() {
		return diagnostic.New("error.upgrade_backup_conflict", dir)
	}
	return os.Remove(previous)
}

// previousImageName binds a backup filename to the exact bytes moved at replacement.
func previousImageName(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err := errors.Join(err, file.Close()); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%x.exe", previousImagePrefix, hash.Sum(nil)), nil
}
