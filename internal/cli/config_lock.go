/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// errConfigChanged asks callers to repeat a mutation against the current snapshot.
var errConfigChanged = diagnostic.New("error.config_changed")

// withLockedConfig locks the file itself so symbolic and hard links share one lock.
// Closing the handle releases the lock, including when the process exits unexpectedly.
func withLockedConfig(path string, flags int, exclusive bool, action func(*os.File) error) error {
	file, err := openConfigHandle(path, flags, 0)
	if err != nil {
		return err
	}
	err = withConfigHandleLock(file, exclusive, action)
	return errors.Join(err, file.Close())
}

// createConfigFile prepares a private sibling before publishing it without replacing a concurrent creator.
func createConfigFile(path string, data []byte) error {
	target := newConfigTarget(path)
	dir, _ := filepath.Split(target)
	if dir == "" {
		dir = "."
	}
	file, err := os.CreateTemp(dir, ".ctyun-config-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	_, writeErr := writeConfigAt(file, data, 0)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	if newConfigTarget(path) != target {
		return errConfigChanged
	}
	err = moveNewConfig(file.Name(), target)
	if errors.Is(err, errors.ErrUnsupported) || errors.Is(err, syscall.EINVAL) {
		// Filesystems may reject an unsupported rename flag with EINVAL.
		err = os.Link(file.Name(), target)
	}
	if errors.Is(err, os.ErrExist) {
		return errConfigChanged
	}
	return err
}

// newConfigTarget follows dangling final symlinks while preserving relative traversal through directory links.
// An existing file, working link or symlink loop is left for no-replace publication to reject.
func newConfigTarget(path string) string {
	for {
		link, err := os.Readlink(path)
		if err != nil {
			return path
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return path
		}
		if !filepath.IsAbs(link) {
			dir, _ := filepath.Split(path)
			link = dir + link
		}
		path = link
	}
}

// moveNewConfig is the no-replace publication boundary for a fully written sibling config.
var moveNewConfig = renameConfigNoReplace

// readLockedConfig reads a complete snapshot for configuration resolution and local diagnostics.
func readLockedConfig(path string) ([]byte, error) {
	var data []byte
	err := withLockedConfig(path, os.O_RDONLY, false, func(file *os.File) error {
		var readErr error
		data, readErr = io.ReadAll(file)
		return readErr
	})
	return data, err
}

// diagnosticConfigBytes prefers the command's snapshot. For early errors, only a regular file
// may supply language and redaction settings; opening a FIFO must not wait for a writer.
func diagnosticConfigBytes(snapshot []byte, path string) ([]byte, error) {
	if snapshot != nil {
		return snapshot, nil
	}
	var raw []byte
	err := withLockedConfig(path, diagnosticConfigReadFlags, false, func(file *os.File) error {
		info, err := file.Stat()
		if err == nil && info.Mode().IsRegular() {
			raw, err = io.ReadAll(file)
		}
		return err
	})
	return raw, err
}

// checkConfigSnapshot detects a replaced pathname and a change to the locked file's contents.
func checkConfigSnapshot(file *os.File, path string, expected []byte) error {
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	current, err := os.Stat(path)
	if err != nil || !os.SameFile(opened, current) {
		return errConfigChanged
	}
	data, err := io.ReadAll(io.NewSectionReader(file, 0, math.MaxInt64))
	if err == nil && !bytes.Equal(data, expected) {
		return errConfigChanged
	}
	return err
}
