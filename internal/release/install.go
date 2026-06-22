/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// InstallOptions controls replacement of the current ctyun executable.
type InstallOptions struct {
	CurrentExecutable string
	ArchivePath       string
	BinaryName        string
	TempDir           string
}

var renamePath = os.Rename

// ExtractBinary extracts binaryName from archivePath into destDir while
// rejecting unsafe archive entries.
func ExtractBinary(archivePath, destDir, binaryName string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	var binaryPath string
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		target := filepath.Join(destDir, filepath.Clean(header.Name))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return "", diagnostic.New("error.archive_path_escapes_destination", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return "", err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				return "", err
			}
			if err := out.Close(); err != nil {
				return "", err
			}
			if filepath.Base(target) == binaryName {
				binaryPath = target
			}
		default:
			return "", diagnostic.New("error.unsupported_archive_entry", header.Name)
		}
	}
	if binaryPath == "" {
		return "", diagnostic.New("error.archive_missing_binary", binaryName)
	}
	return binaryPath, nil
}

// InstallArtifact extracts an artifact archive and replaces the current
// executable, restoring the old binary if final replacement fails.
func InstallArtifact(opts InstallOptions) error {
	if opts.CurrentExecutable == "" {
		return diagnostic.New("error.current_executable_required")
	}
	if opts.ArchivePath == "" {
		return diagnostic.New("error.archive_path_required")
	}
	if opts.BinaryName == "" {
		opts.BinaryName = filepath.Base(opts.CurrentExecutable)
	}

	tempParent := opts.TempDir
	if tempParent == "" {
		tempParent = filepath.Dir(opts.CurrentExecutable)
	}
	extractDir, err := os.MkdirTemp(tempParent, ".ctyun-upgrade-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	newBinary, err := ExtractBinary(opts.ArchivePath, extractDir, opts.BinaryName)
	if err != nil {
		return err
	}
	if err := os.Chmod(newBinary, 0o755); err != nil {
		return err
	}

	backup := opts.CurrentExecutable + ".old-" + fmt.Sprint(time.Now().UnixNano())
	if err := renamePath(opts.CurrentExecutable, backup); err != nil {
		return err
	}
	if err := renamePath(newBinary, opts.CurrentExecutable); err != nil {
		_ = renamePath(backup, opts.CurrentExecutable)
		return err
	}
	return os.Remove(backup)
}
