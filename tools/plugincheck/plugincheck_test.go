/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
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
				args := []string{"--lang", "en-US", "--table", "plain"}
				if command.Dangerous.Confirm != "" {
					args = append(args, "--yes")
				}
				args = append(args, commandSmokeArgs(command)...)
				args = append(args, "--offline")
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
		{command: "region.resource.show", operation: "v4.region.resource.show"},
		{command: "region.quota.show", operation: "v4.region.quota.show"},
	} {
		if !commands[expected.command] {
			t.Fatalf("region plugin missing command %s", expected.command)
		}
		if !operations[expected.operation] {
			t.Fatalf("region plugin missing operation %s", expected.operation)
		}
	}
}

// TestRegionPluginCoversDocumentedSummaryObjects keeps wide single-object
// region tables complete by exposing documented top-level response objects as
// selectable fields while default_columns can keep the normal view compact.
func TestRegionPluginCoversDocumentedSummaryObjects(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", "region")), version.Version)
	if err != nil {
		t.Fatalf("load region plugin: %v", err)
	}

	assertTableKeys(t, bundle.Tables.Tables["region.product.show"], []string{
		"ebs", "ecs", "pm", "image", "monitor", "oss", "paas", "saas", "caas", "hpc", "sdwan", "order", "lb",
		"scaling", "eip", "sfs", "dec", "ch", "vpc", "cnssl", "security_safe", "rds", "kms", "acl", "cda", "efs", "other", "az_list",
	})
	assertTableKeys(t, bundle.Tables.Tables["region.resource.show"], []string{
		"vm", "volume", "volume_snapshot", "vpc", "public_ip", "bms", "nat", "disk_backup", "vm_group", "snapshot", "acl_list",
		"ip_pool", "image", "lb_listener", "loadbalancer", "os_backup", "traffic_mirror_flow", "traffic_mirror_filter", "cbr", "cert", "cbr_vbs",
	})
	assertTableKeys(t, bundle.Tables.Tables["region.quota.show"], []string{"quotas", "global_quota"})
}

// TestRegionPluginUsesReadableFieldSummaries protects the default single-row
// views from regressing to long object cells while keeping object selectors
// available through explicit --cols.
func TestRegionPluginUsesReadableFieldSummaries(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", "region")), version.Version)
	if err != nil {
		t.Fatalf("load region plugin: %v", err)
	}

	product := bundle.Tables.Tables["region.product.show"]
	assertTableDefaultKeys(t, product, []string{
		"ecs_flavor_types",
		"ebs_storage_types",
		"ebs_share",
		"ebs_auto_snapshot_policy",
		"monitor_available",
		"paas_pay_as_you_go",
		"paas_upgrade",
		"az_names",
	})
	for _, key := range []string{"ecs", "ebs", "monitor", "az_list"} {
		if slices.Contains(product.DefaultColumns, key) {
			t.Fatalf("product default columns include long object selector %s", key)
		}
	}

	resource := bundle.Tables.Tables["region.resource.show"]
	assertTableColumnLabel(t, resource, "cbr", "Cloud Backup (CBR) Summary")
	assertTableColumnLabel(t, resource, "cbr_vbs", "Volume Backup (VBS) Summary")
	assertTableColumnLabel(t, resource, "ecs_backup_total", "Cloud Backup (CBR) Total")
	assertTableColumnLabel(t, resource, "ip_pool", "Shared Bandwidth Summary")
	assertTableColumnLabel(t, resource, "ip_pool_total", "Shared Bandwidth Total")
	assertUniqueTableLabels(t, resource)
}

// TestJobPluginUsesProfileRegionAndReadableColumns keeps the generic Job
// command aligned with profile-scoped region handling while avoiding raw
// response-object selectors as table columns.
func TestJobPluginUsesProfileRegionAndReadableColumns(t *testing.T) {
	bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", "job")), version.Version)
	if err != nil {
		t.Fatalf("load job plugin: %v", err)
	}

	command, ok := plugin.FindCommand(bundle, []string{"job", "info", "job-demo-1"})
	if !ok {
		t.Fatal("job info command does not accept job_id argument")
	}
	if !slices.Equal(command.Path, []string{"job", "info", "{job_id}"}) {
		t.Fatalf("job command path = %#v", command.Path)
	}
	if len(command.Parameters) != 1 || command.Parameters[0].Name != "region" || command.Parameters[0].Flag != "region" || command.Parameters[0].Target != "regionID" || command.Parameters[0].Required {
		t.Fatalf("job region option = %#v", command.Parameters)
	}
	operation := bundle.APIs.Operations[command.Operation]
	if operation.Query["regionID"] != "$profile.region" || operation.Query["jobID"] != "$arg.job_id" {
		t.Fatalf("job query bindings = %#v", operation.Query)
	}

	table := bundle.Tables.Tables[command.Table]
	assertTableKeys(t, table, []string{"job_id", "status", "job_status", "resource_id", "task_name"})
	for _, column := range table.Columns {
		if column.Key == "fields" || column.Path == "fields" || column.Labels["en-US"] == "Fields" {
			t.Fatalf("job table exposes raw fields object: %#v", column)
		}
	}
}

// TestProfileRegionOperationsExposeRegionOption keeps generated API plugins
// consistent: profile-sourced regionID can come from config, or from a visible
// optional --region command override when profiles are not configured.
func TestProfileRegionOperationsExposeRegionOption(t *testing.T) {
	pluginsRoot := repoPath(t, "plugins")
	for _, pluginDir := range pluginDirs(t, pluginsRoot) {
		bundle, err := plugin.LoadBundle(pluginDir, version.Version)
		if err != nil {
			t.Fatalf("load plugin %s: %v", filepath.Base(pluginDir), err)
		}
		for _, command := range bundle.Commands.Commands {
			operation, ok := bundle.APIs.Operations[command.Operation]
			if !ok {
				continue
			}
			for _, target := range profileRegionTargets(operation) {
				if !commandHasOptionalRegionOption(command, target) {
					t.Fatalf("%s command %s maps %s from profile region but does not expose optional --region", bundle.Manifest.Name, command.ID, target)
				}
			}
		}
	}
}

// TestRegionArgumentCommandsDoNotAlsoExposeRegionOption keeps the command
// surface unambiguous: commands with a region_id argument can fall back to the
// active profile, but they must not also expose a duplicate --region option.
func TestRegionArgumentCommandsDoNotAlsoExposeRegionOption(t *testing.T) {
	pluginsRoot := repoPath(t, "plugins")
	for _, pluginDir := range pluginDirs(t, pluginsRoot) {
		bundle, err := plugin.LoadBundle(pluginDir, version.Version)
		if err != nil {
			t.Fatalf("load plugin %s: %v", filepath.Base(pluginDir), err)
		}
		for _, command := range bundle.Commands.Commands {
			if !commandPathHasArgument(command, "region_id") {
				continue
			}
			if commandHasOptionalRegionOption(command, "regionID") {
				t.Fatalf("%s command %s exposes both {region_id} and --region", bundle.Manifest.Name, command.ID)
			}
		}
	}
}

// assertTableKeys checks that table exposes all stable keys.
func assertTableKeys(t *testing.T, table plugin.Table, keys []string) {
	t.Helper()
	seen := make(map[string]bool, len(table.Columns))
	for _, column := range table.Columns {
		seen[column.Key] = true
	}
	for _, key := range keys {
		if !seen[key] {
			t.Fatalf("table missing documented key %s", key)
		}
	}
}

// assertTableDefaultKeys checks a table's exact default selector order.
func assertTableDefaultKeys(t *testing.T, table plugin.Table, keys []string) {
	t.Helper()
	if !slices.Equal(table.DefaultColumns, keys) {
		t.Fatalf("default columns = %#v, want %#v", table.DefaultColumns, keys)
	}
}

// assertTableColumnLabel checks the English label for a table selector.
func assertTableColumnLabel(t *testing.T, table plugin.Table, key, label string) {
	t.Helper()
	for _, column := range table.Columns {
		if column.Key == key {
			if got := column.Labels["en-US"]; got != label {
				t.Fatalf("label for %s = %q, want %q", key, got, label)
			}
			return
		}
	}
	t.Fatalf("table missing key %s", key)
}

// assertUniqueTableLabels checks that visible English labels are not ambiguous.
func assertUniqueTableLabels(t *testing.T, table plugin.Table) {
	t.Helper()
	seen := map[string]string{}
	for _, column := range table.Columns {
		label := column.Labels["en-US"]
		if previous := seen[label]; previous != "" {
			t.Fatalf("duplicate label %q for %s and %s", label, previous, column.Key)
		}
		seen[label] = column.Key
	}
}

// profileRegionTargets returns request fields sourced from profile region.
func profileRegionTargets(operation plugin.Operation) []string {
	var targets []string
	for _, values := range []map[string]string{operation.Body, operation.Query, operation.Headers} {
		for target, source := range values {
			if source == "$profile.region" {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

// commandHasOptionalRegionOption reports whether command exposes target as an
// optional --region override.
func commandHasOptionalRegionOption(command plugin.Command, target string) bool {
	for _, parameter := range command.Parameters {
		if parameter.Name == "region" && parameter.Flag == "region" && parameter.Target == target && !parameter.Required {
			return true
		}
	}
	return false
}

// commandPathHasArgument reports whether command path declares argument.
func commandPathHasArgument(command plugin.Command, argument string) bool {
	for _, segment := range command.Path {
		if segment == "{"+argument+"}" {
			return true
		}
	}
	return false
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
	if len(command.ConditionalRequirements) > 0 && len(command.Examples) > 0 {
		return exampleSmokeArgs(command.Examples[0])
	}
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

// exampleSmokeArgs returns a smoke command from a curated ctyun example.
func exampleSmokeArgs(example string) []string {
	fields := strings.Fields(example)
	if len(fields) > 0 && fields[0] == "ctyun" {
		return fields[1:]
	}
	return fields
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
