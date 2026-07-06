/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// PromoteDraft copies review-ready runtime plugin metadata into targetDir and
// advances the accepted OpenAPI baseline.
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
	draftDir := workspace.ProductPath(product, "draft")
	for _, name := range []string{"plugin.json", "apis.json", "commands.json", "tables.json", "waiters.json"} {
		if err := copyJSONFileIfChanged(filepath.Join(draftDir, name), filepath.Join(targetDir, name)); err != nil {
			return err
		}
	}
	for _, dir := range []string{"fixtures", "i18n"} {
		if err := syncJSONTree(filepath.Join(draftDir, dir), filepath.Join(targetDir, dir)); err != nil {
			return err
		}
	}
	return workspace.WriteCatalog(workspace.ProductPath(product, "baseline.json"), source)
}

// syncJSONTree copies changed JSON files while preserving equivalent target
// formatting and removing files no longer present in the generated draft.
func syncJSONTree(sourceDir, destinationDir string) error {
	seen := map[string]bool{}
	if err := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(sourceDir, path)
		seen[rel] = true
		target := filepath.Join(destinationDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyJSONFileIfChanged(path, target)
	}); err != nil {
		return err
	}
	var stale []string
	if err := filepath.WalkDir(destinationDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(destinationDir, path)
		if !seen[rel] {
			stale = append(stale, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for index := len(stale) - 1; index >= 0; index-- {
		if err := os.RemoveAll(stale[index]); err != nil {
			return err
		}
	}
	return nil
}

// copyJSONFileIfChanged preserves an existing target file when its JSON content
// is equivalent to the generated draft.
func copyJSONFileIfChanged(sourcePath, destinationPath string) error {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return err
	}
	target, err := os.ReadFile(destinationPath)
	switch {
	case err == nil:
		if bytes.Equal(source, target) || equivalentJSON(source, target) {
			return nil
		}
	case os.IsNotExist(err):
	default:
		return err
	}
	return os.WriteFile(destinationPath, source, 0o644)
}

// equivalentJSON reports whether two JSON documents decode to the same value.
func equivalentJSON(left, right []byte) bool {
	var leftValue any
	var rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}
