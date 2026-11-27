/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// publishedFile records how to undo one file replacement if a later destination fails.
type publishedFile struct {
	path   string
	backup string
}

// renamePublicationFile moves files on the output filesystem, including backup restoration.
var renamePublicationFile = os.Rename

// createReleaseStage is the workspace allocation boundary before any release outputs change.
var createReleaseStage = os.MkdirTemp

// publishReleaseFiles installs the verified upload set and retires superseded archives in the same rollback transaction.
// If a filesystem error also prevents recovery, the backup directory remains available to the publisher.
func publishReleaseFiles(stage, output string, plan archiveRetention) error {
	backupDir, err := os.MkdirTemp(output, ".ctyun-previous-*")
	if err != nil {
		return err
	}
	var changes []publishedFile
	// Back up obsolete files by their original spelling before installing replacements.
	// Rollback then removes any case-only replacement before restoring the original name.
	for _, name := range plan.superseded {
		change := publishedFile{path: filepath.Join(output, name), backup: filepath.Join(backupDir, url.PathEscape(name))}
		if err = renamePublicationFile(change.path, change.backup); err != nil {
			break
		}
		changes = append(changes, change)
	}
	if err == nil {
		err = fs.WalkDir(os.DirFS(stage), ".", func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			target := filepath.Join(output, filepath.FromSlash(name))
			if entry.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			if strings.HasSuffix(name, ".tar.gz") && !plan.current[name] {
				// A build requested out of order may have lost every platform in the merged index.
				return nil
			}
			change := publishedFile{path: target}
			info, err := os.Lstat(target)
			if err == nil {
				if !info.Mode().IsRegular() {
					return fmt.Errorf("cannot replace non-regular release output %q", target)
				}
				change.backup = filepath.Join(backupDir, url.PathEscape(name))
				if err := renamePublicationFile(target, change.backup); err != nil {
					return err
				}
			} else if !os.IsNotExist(err) {
				return err
			}
			err = renamePublicationFile(filepath.Join(stage, filepath.FromSlash(name)), target)
			if err == nil || change.backup != "" {
				changes = append(changes, change)
			}
			return err
		})
	}

	if err != nil {
		var recoveryErr error
		for _, change := range slices.Backward(changes) {
			if change.backup == "" {
				recoveryErr = errors.Join(recoveryErr, os.Remove(change.path))
			} else {
				recoveryErr = errors.Join(recoveryErr, renamePublicationFile(change.backup, change.path))
			}
		}
		if recoveryErr != nil {
			return errors.Join(err, fmt.Errorf("restore release files from %q: %w", backupDir, recoveryErr))
		}
	}
	return closeWithError(err, func() error { return os.RemoveAll(backupDir) })
}
