/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package localdoctor

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestPluginChecksInspectBundlesIndependentlyInStableOrder(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"healthy", "broken"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	deps := DefaultDependencies()
	deps.LoadBundle = func(dir, coreVersion string) (plugin.Bundle, error) {
		if filepath.Base(dir) == "broken" {
			return plugin.Bundle{}, errors.New("broken metadata")
		}
		return plugin.Bundle{Manifest: plugin.Manifest{Name: filepath.Base(dir), Version: "1.0.0"}}, nil
	}
	report := Run(Input{UseInjectedConfig: true, Getenv: completeEnvironment, PluginRoot: root, CoreVersion: "1.0.0"}, deps)
	assertFindingStatus(t, report, CheckPluginDirectory, doctor.StatusPassed)
	plugins := pluginFindings(report.Findings)
	if got := []string{plugins[0].Target, plugins[1].Target}; !slices.Equal(got, []string{"broken", "healthy"}) {
		t.Fatalf("plugin targets = %v", got)
	}
	if plugins[0].Status != doctor.StatusFailed || plugins[1].Status != doctor.StatusPassed {
		t.Fatalf("plugin findings = %#v", plugins)
	}
}

func TestPluginChecksHandleMissingAndEmptyRoots(t *testing.T) {
	missing := Run(Input{UseInjectedConfig: true, Getenv: completeEnvironment, PluginRoot: filepath.Join(t.TempDir(), "missing")}, DefaultDependencies())
	assertFindingStatus(t, missing, CheckPluginDirectory, doctor.StatusSkipped)
	if got := pluginFindings(missing.Findings); len(got) != 0 {
		t.Fatalf("missing-root plugins = %#v", got)
	}
	empty := Run(Input{UseInjectedConfig: true, Getenv: completeEnvironment, PluginRoot: t.TempDir()}, DefaultDependencies())
	assertFindingStatus(t, empty, CheckPluginDirectory, doctor.StatusPassed)
}

func TestPluginChecksCoverUnusableRoots(t *testing.T) {
	statError := DefaultDependencies()
	statError.Stat = func(string) (fs.FileInfo, error) { return nil, errors.New("stat") }
	assertFindingStatus(t, Run(Input{UseInjectedConfig: true, Getenv: completeEnvironment, PluginRoot: "/plugins"}, statError), CheckPluginDirectory, doctor.StatusFailed)

	file := filepath.Join(t.TempDir(), "plugins")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	assertFindingStatus(t, Run(Input{UseInjectedConfig: true, Getenv: completeEnvironment, PluginRoot: file}, DefaultDependencies()), CheckPluginDirectory, doctor.StatusFailed)

	root := t.TempDir()
	readError := DefaultDependencies()
	readError.ReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("read") }
	assertFindingStatus(t, Run(Input{UseInjectedConfig: true, Getenv: completeEnvironment, PluginRoot: root}, readError), CheckPluginDirectory, doctor.StatusFailed)
}

// completeEnvironment supplies a complete non-secret test credential pair.
func completeEnvironment(key string) string {
	if key == "CTYUN_AK" || key == "CTYUN_SK" {
		return "configured"
	}
	return ""
}

// pluginFindings selects per-bundle rows from a complete local report.
func pluginFindings(findings []Finding) []Finding {
	var plugins []Finding
	for _, finding := range findings {
		if finding.Key == CheckPlugin {
			plugins = append(plugins, finding)
		}
	}
	return plugins
}
