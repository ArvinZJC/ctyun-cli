/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InstallLocalBundle installs a local plugin directory or archive after basic
// manifest name validation.
func InstallLocalBundle(srcDir, destRoot string) (string, error) {
	return installLocalBundle(srcDir, destRoot, "")
}

// InstallVerifiedLocalBundle installs a local plugin directory or archive only
// after full bundle validation against coreVersion.
func InstallVerifiedLocalBundle(srcDir, destRoot, coreVersion string) (string, error) {
	return installLocalBundle(srcDir, destRoot, coreVersion)
}

// installLocalBundle validates and installs a directory or compressed plugin
// bundle.
func installLocalBundle(srcDir, destRoot, coreVersion string) (string, error) {
	if strings.HasSuffix(srcDir, ".tar.gz") || strings.HasSuffix(srcDir, ".tgz") {
		tmpDir, err := os.MkdirTemp("", "ctyun-plugin-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(tmpDir)
		if err := extractTarGz(srcDir, tmpDir); err != nil {
			return "", err
		}
		// Archives may wrap the bundle in one top-level directory; normalize to
		// the directory that actually contains plugin.json before validation.
		bundleRoot, err := findExtractedBundleRoot(tmpDir)
		if err != nil {
			return "", err
		}
		srcDir = bundleRoot
	}

	if coreVersion != "" {
		bundle, err := LoadBundle(srcDir, coreVersion)
		if err != nil {
			return "", err
		}
		return copyBundleIntoPlace(srcDir, destRoot, bundle.Manifest.Name)
	}

	manifest := Manifest{}
	if err := readJSON(filepath.Join(srcDir, "plugin.json"), &manifest); err != nil {
		return "", err
	}
	if manifest.Name == "" {
		return "", fmt.Errorf("plugin manifest is missing name")
	}
	if !ValidName(manifest.Name) {
		return "", fmt.Errorf("invalid plugin name %q", manifest.Name)
	}

	return copyBundleIntoPlace(srcDir, destRoot, manifest.Name)
}

// findExtractedBundleRoot locates plugin.json after an archive has been
// unpacked.
func findExtractedBundleRoot(root string) (string, error) {
	if _, err := os.Stat(filepath.Join(root, "plugin.json")); err == nil {
		return root, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(candidate, "plugin.json")); err == nil {
			candidates = append(candidates, candidate)
		} else if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("archive contains multiple plugin roots")
	}
	return "", fmt.Errorf("archive does not contain plugin.json")
}

// copyBundleIntoPlace copies a validated bundle into its final plugin
// directory.
func copyBundleIntoPlace(srcDir, destRoot, name string) (string, error) {
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return "", err
	}
	destDir := filepath.Join(destRoot, name)
	tmpDir, err := os.MkdirTemp(destRoot, "."+name+"-tmp-*")
	if err != nil {
		return "", err
	}
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.RemoveAll(tmpDir)
		}
	}()
	if err := copyDir(srcDir, tmpDir); err != nil {
		return "", err
	}
	// Copy into a sibling temp directory first so a failed install does not
	// leave a partially replaced plugin.
	if err := replaceDir(tmpDir, destDir); err != nil {
		return "", err
	}
	cleanupTmp = false
	return destDir, nil
}

// replaceDir atomically swaps src into dest while preserving the old dest on
// rename failure.
func replaceDir(src, dest string) error {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return os.Rename(src, dest)
	} else if err != nil {
		return err
	}

	backup := dest + ".old-" + fmt.Sprint(time.Now().UnixNano())
	if err := os.Rename(dest, backup); err != nil {
		return err
	}
	if err := os.Rename(src, dest); err != nil {
		_ = os.Rename(backup, dest)
		return err
	}
	return os.RemoveAll(backup)
}

// extractTarGz extracts a plugin archive while rejecting unsafe entry types and
// paths.
func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, filepath.Clean(header.Name))
		// filepath.Join cleans the path, so check the final location instead of
		// trusting the archive entry name.
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("archive path escapes destination: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry %s", header.Name)
		}
	}
}

// copyDir recursively copies regular files and directories from src to dest.
func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type() != 0 {
			return fmt.Errorf("unsupported bundle entry %s", rel)
		}
		return copyFile(path, target)
	})
}

// copyFile copies one regular file to dest with plugin-bundle permissions.
func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
