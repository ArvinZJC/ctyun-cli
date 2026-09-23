/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package output

import (
	"io"
	"os"
	"path/filepath"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// closeDownloadFile is the close boundary used to verify failed publication cleanup.
var closeDownloadFile = (*os.File).Close

// WriteFile publishes a completed sibling temporary file without clobbering by default.
func WriteFile(path string, overwrite bool, write func(io.Writer) error) (err error) {
	if info, statErr := os.Lstat(path); statErr == nil {
		if !overwrite || !info.Mode().IsRegular() {
			return diagnostic.New("error.output_exists", path)
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".ctyun-download-*")
	if err != nil {
		return err
	}
	// Successful publication checks Close below; fallback cleanup is best-effort.
	defer func() { _ = temporary.Close(); _ = os.Remove(temporary.Name()) }()
	if err = write(temporary); err != nil {
		return err
	}
	if err = temporary.Sync(); err != nil {
		return err
	}
	if err = closeDownloadFile(temporary); err != nil {
		return err
	}
	if overwrite {
		if info, statErr := os.Lstat(path); statErr == nil && !info.Mode().IsRegular() {
			return diagnostic.New("error.output_exists", path)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return statErr
		}
		return os.Rename(temporary.Name(), path)
	}
	return os.Link(temporary.Name(), path)
}
