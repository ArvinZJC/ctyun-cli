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

func InstallLocalBundle(srcDir, destRoot string) (string, error) {
	return installLocalBundle(srcDir, destRoot, "")
}

func InstallVerifiedLocalBundle(srcDir, destRoot, coreVersion string) (string, error) {
	return installLocalBundle(srcDir, destRoot, coreVersion)
}

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
		if bundle.Manifest.Name == "" {
			return "", fmt.Errorf("plugin manifest is missing name")
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
	if err := replaceDir(tmpDir, destDir); err != nil {
		return "", err
	}
	cleanupTmp = false
	return destDir, nil
}

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

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
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
