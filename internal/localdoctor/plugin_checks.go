/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package localdoctor

import (
	"errors"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
)

// evaluatePlugins inspects an installed-plugin root and every immediate bundle independently.
func evaluatePlugins(input Input, dependencies Dependencies) []Finding {
	rootFinding := Finding{Key: CheckPluginDirectory, Target: input.PluginRoot}
	info, err := dependencies.Stat(input.PluginRoot)
	if errors.Is(err, fs.ErrNotExist) {
		rootFinding.Status = doctor.StatusSkipped
		rootFinding.DetailKey = "doctor.local.detail.plugin_directory_missing"
		return []Finding{rootFinding}
	}
	if err != nil {
		rootFinding.Status = doctor.StatusFailed
		rootFinding.Category = FailureFilesystem
		rootFinding.DetailKey = "doctor.local.detail.plugin_directory_unreadable"
		return []Finding{rootFinding}
	}
	if !info.IsDir() {
		rootFinding.Status = doctor.StatusFailed
		rootFinding.Category = FailureFilesystem
		rootFinding.DetailKey = "doctor.local.detail.plugin_directory_not_directory"
		return []Finding{rootFinding}
	}
	entries, err := dependencies.ReadDir(input.PluginRoot)
	if err != nil {
		rootFinding.Status = doctor.StatusFailed
		rootFinding.Category = FailureFilesystem
		rootFinding.DetailKey = "doctor.local.detail.plugin_directory_unreadable"
		return []Finding{rootFinding}
	}
	rootFinding.Status = doctor.StatusPassed
	rootFinding.DetailKey = "doctor.local.detail.plugin_directory_passed"
	directories := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}
	slices.Sort(directories)
	findings := []Finding{rootFinding}
	for _, name := range directories {
		bundle, loadErr := dependencies.LoadBundle(filepath.Join(input.PluginRoot, name), input.CoreVersion)
		finding := Finding{Key: CheckPlugin, Target: name}
		if loadErr != nil {
			finding.Status = doctor.StatusFailed
			finding.Category = FailurePlugin
			finding.DetailKey = "doctor.local.detail.plugin_invalid"
		} else {
			finding.Status = doctor.StatusPassed
			finding.DetailKey = "doctor.local.detail.plugin_passed"
			finding.DetailArgs = []any{bundle.Manifest.Version}
		}
		findings = append(findings, finding)
	}
	return findings
}
