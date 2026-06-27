/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapi

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PromoteDraft copies runtime plugin metadata into targetDir and advances the
// accepted OpenAPI baseline after a reviewed or curated promotion.
func (workspace Workspace) PromoteDraft(product, targetDir string) error {
	source, err := workspace.ReadSource(product)
	if err != nil {
		return err
	}
	report, err := workspace.ReviewDraft(product)
	if err != nil {
		return err
	}
	if !report.Ready {
		return fmt.Errorf("review is not ready for %s", product)
	}
	if report.Quality != "reviewed" && report.Quality != "curated" {
		return fmt.Errorf("promotion requires reviewed or curated quality, got %s", report.Quality)
	}
	draftDir := workspace.ProductPath(product, "draft")
	for _, name := range []string{"plugin.json", "apis.json", "commands.json", "tables.json", "waiters.json"} {
		if err := copyFile(filepath.Join(draftDir, name), filepath.Join(targetDir, name)); err != nil {
			return err
		}
	}
	for _, dir := range []string{"fixtures", "i18n"} {
		target := filepath.Join(targetDir, dir)
		if err := os.RemoveAll(target); err != nil {
			return err
		}
		if err := copyTree(filepath.Join(draftDir, dir), target); err != nil {
			return err
		}
	}
	return workspace.WriteCatalog(workspace.ProductPath(product, "baseline.json"), source)
}

// copyTree recursively copies a generated draft directory.
func copyTree(src, dest string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

// copyFile copies one generated metadata file into the promoted plugin.
func copyFile(src, dest string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return errors.Join(err, input.Close())
	}
	output, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return errors.Join(err, input.Close())
	}
	_, copyErr := io.Copy(output, input)
	return errors.Join(copyErr, output.Close(), input.Close())
}
