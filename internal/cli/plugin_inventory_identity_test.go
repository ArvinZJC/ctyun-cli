/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
	"github.com/ArvinZJC/ctyun-cli/internal/localdoctor"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestPluginInventoryRejectsMismatchedDirectory checks the installed layout separately from standalone bundle validity.
func TestPluginInventoryRejectsMismatchedDirectory(t *testing.T) {
	t.Cleanup(patchVersion(testCoreVersion()))
	root := t.TempDir()
	dir := filepath.Join(root, "renamed")
	writeVersionedBundle(t, dir, "sample", "0.1.0")
	if _, err := plugin.LoadBundle(dir, testCoreVersion()); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"plugin", "list"}, {"sample", "--help"}} {
		err := Run(Config{Args: args, Config: []byte("{}"), PluginRoot: root, Stdout: io.Discard, Stderr: io.Discard})
		if finding, ok := errors.AsType[diagnostic.Error](err); !ok || finding.Key != "error.registry_bundle_mismatch" {
			t.Errorf("%v returned %v for a mismatched installed directory", args, err)
		}
	}
	report := localdoctor.Run(localdoctor.Input{PluginRoot: root, CoreVersion: testCoreVersion(), UseInjectedConfig: true}, localdoctor.DefaultDependencies())
	for _, finding := range report.Findings {
		if finding.Key == localdoctor.CheckPlugin && finding.Target == "renamed" {
			if finding.Status != doctor.StatusFailed || finding.Category != localdoctor.FailurePlugin {
				t.Fatalf("installed identity finding: %+v", finding)
			}
			return
		}
	}
	t.Fatal("renamed bundle absent from local diagnostics")
}

// TestPluginRemoveCountsOnlyExistingTargets preserves idempotent removal and reports actual changes.
func TestPluginRemoveCountsOnlyExistingTargets(t *testing.T) {
	root := t.TempDir()
	writeVersionedBundle(t, filepath.Join(root, "sample"), "sample", "0.1.0")
	var stdout bytes.Buffer
	err := removePlugins(&stdout, io.Discard, strings.NewReader(""), root,
		pluginRemoveOptions{Names: []string{"sample", "sample", "absent"}}, globalOptions{Yes: true, Language: "en-US"})
	if err != nil || stdout.String() != "Plugin removal complete: removed 1; failed 0.\n" {
		t.Fatalf("summary %q, error %v", stdout.String(), err)
	}
}
