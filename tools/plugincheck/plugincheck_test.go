/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestRepoPluginsLoadAndRunOfflineFixtures verifies that every repository
// plugin can load and render its fixture-backed commands through the CLI.
func TestRepoPluginsLoadAndRunOfflineFixtures(t *testing.T) {
	pluginsRoot := repoPath(t, "plugins")
	for _, pluginDir := range pluginDirs(t, pluginsRoot) {
		bundle, err := plugin.LoadBundle(pluginDir, version.Version)
		if err != nil {
			t.Fatalf("load plugin %s: %v", filepath.Base(pluginDir), err)
		}
		if len(bundle.Commands.Commands) == 0 {
			t.Fatalf("plugin %s has no commands", bundle.Manifest.Name)
		}

		for _, command := range bundle.Commands.Commands {
			if command.FixtureResponse == "" {
				continue
			}
			t.Run(command.ID, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				args := []string{"--offline", "--lang", "en-US", "--table", "plain"}
				if command.Dangerous.Confirm != "" {
					args = append(args, "--yes")
				}
				args = append(args, commandSmokeArgs(command)...)
				if err := cli.Run(cli.Config{
					Args:       args,
					Stdout:     &stdout,
					Stderr:     &stderr,
					PluginRoot: t.TempDir(),
				}); err != nil {
					t.Fatalf("offline command %q returned error: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
				}
				if strings.TrimSpace(stdout.String()) == "" {
					t.Fatalf("offline command %q produced empty output", strings.Join(args, " "))
				}
			})
		}
	}
}

// TestRegionPluginCoversResourcePoolAPIs keeps the region plugin aligned with
// the ECS public resource-pool APIs that use /v4/region paths.
func TestRegionPluginCoversResourcePoolAPIs(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", "region")), version.Version)
	if err != nil {
		t.Fatalf("load region plugin: %v", err)
	}

	operations := make(map[string]bool, len(bundle.APIs.Operations))
	for id := range bundle.APIs.Operations {
		operations[id] = true
	}
	commands := make(map[string]bool, len(bundle.Commands.Commands))
	for _, command := range bundle.Commands.Commands {
		commands[command.ID] = true
	}

	for _, expected := range []struct {
		command   string
		operation string
	}{
		{command: "region.list", operation: "v4.region.list"},
		{command: "region.show", operation: "v4.region.show"},
		{command: "region.zone.list", operation: "v4.region.zone.list"},
		{command: "region.product.show", operation: "v4.region.product.show"},
		{command: "region.demand.check", operation: "v4.region.demand.check"},
	} {
		if !commands[expected.command] {
			t.Fatalf("region plugin missing command %s", expected.command)
		}
		if !operations[expected.operation] {
			t.Fatalf("region plugin missing operation %s", expected.operation)
		}
	}
}

// TestRealPluginCommandSmokesStayOutOfCore enforces that real plugin release
// checks live under tools/plugincheck instead of internal/cli.
func TestRealPluginCommandSmokesStayOutOfCore(t *testing.T) {
	pluginsRoot := repoPath(t, "plugins")
	pluginNames := make(map[string]bool)
	for _, pluginDir := range pluginDirs(t, pluginsRoot) {
		pluginNames[filepath.Base(pluginDir)] = true
	}

	entries, err := os.ReadDir(repoPath(t, filepath.Join("internal", "cli")))
	if err != nil {
		t.Fatalf("read internal/cli: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "cli_command_") || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		product := strings.TrimSuffix(strings.TrimPrefix(name, "cli_command_"), "_test.go")
		if pluginNames[product] {
			t.Fatalf("%s contains a real-plugin smoke test; keep plugin release checks under tools/plugincheck", filepath.Join("internal", "cli", name))
		}
	}
}

// commandSmokeArgs returns a minimal command line for one fixture-backed
// plugin command.
func commandSmokeArgs(command plugin.Command) []string {
	args := make([]string, 0, len(command.Path)+len(command.Parameters)*2)
	for _, segment := range command.Path {
		args = append(args, pathSegmentValue(segment))
	}
	for _, parameter := range command.Parameters {
		if !parameter.Required {
			continue
		}
		args = append(args, "--"+parameter.Flag, parameterValue(parameter))
	}
	return args
}

// pathSegmentValue replaces metadata path placeholders with deterministic
// sample values.
func pathSegmentValue(segment string) string {
	if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
		return segment
	}
	return sampleValue(strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}"))
}

// parameterValue returns a deterministic value for one required command
// parameter.
func parameterValue(parameter plugin.Parameter) string {
	if len(parameter.AllowedValues) > 0 {
		return parameter.AllowedValues[0]
	}
	return sampleValue(parameter.Name)
}

// sampleValue returns known-good placeholder values for current repository
// plugin command paths.
func sampleValue(name string) string {
	switch name {
	case "instance_id":
		return "ins-demo-1"
	case "region_id":
		return "81f7728662dd11ec810800155d307d5b"
	default:
		return "sample-" + strings.ReplaceAll(name, "_", "-")
	}
}

// pluginDirs lists plugin bundle directories under root.
func pluginDirs(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read plugins root: %v", err)
	}
	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(root, entry.Name()))
		}
	}
	return dirs
}

// repoPath finds a repository path from the current test working directory or
// one of its parents.
func repoPath(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		} else if err != nil && !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", candidate, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("cannot find %s from working directory", name)
		}
		dir = parent
	}
}
