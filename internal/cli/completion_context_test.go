/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestCompletionScriptSupportsPowerShell emits PowerShell registration that delegates to the hidden resolver.
func TestCompletionScriptSupportsPowerShell(t *testing.T) {
	var stdout bytes.Buffer
	if err := runCompletion(&stdout, []string{"powershell"}); err != nil {
		t.Fatalf("runCompletion powershell returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Register-ArgumentCompleter", "ctyun", "__complete"} {
		if !strings.Contains(got, want) {
			t.Fatalf("PowerShell completion script missing %q:\n%s", want, got)
		}
	}
}

// TestHiddenCompletionUsesContextAwareCommandTree combines independent synthetic command groups.
func TestHiddenCompletionUsesContextAwareCommandTree(t *testing.T) {
	pluginRoot := t.TempDir()
	writeSyntheticNetworkBundle(t, filepath.Join(pluginRoot, "sample-network"))
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	top := completeArgs(nil, pluginRoot)
	assertHasCompletions(t, top, "doctor", "ecs", "sample-network")
	assertNoCompletions(t, top, "instance", "subnet")

	ecsChildren := completeArgs([]string{"ecs", ""}, pluginRoot)
	assertHasCompletions(t, ecsChildren, "instance")
	assertNoCompletions(t, ecsChildren, "subnet", "doctor")

	instanceChildren := completeArgs([]string{"ecs", "instance", ""}, pluginRoot)
	assertHasCompletions(t, instanceChildren, "list")
	assertNoCompletions(t, instanceChildren, "ecs", "doctor")
}

// TestHiddenCompletionSuggestsOptionsForResolvedCommand offers applicable global and metadata options after a complete leaf.
func TestHiddenCompletionSuggestsOptionsForResolvedCommand(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	got := completeArgs([]string{"ecs", "instance", "list", ""}, pluginRoot)
	assertHasCompletions(t, got, "--name", "--cols", "--filter", "--sort", "--output", "-o")
	assertNoCompletions(t, got, "show", "start")
}

// TestHiddenCompletionSuggestsOptionValuesFromMetadata uses declared enums, table selectors, and waiter bindings for option values.
func TestHiddenCompletionSuggestsOptionValuesFromMetadata(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	waitRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(waitRoot, "ecs"))

	assertEqualCompletions(t, completeArgs([]string{"--output", ""}, pluginRoot), []string{"json", "table"})
	assertEqualCompletions(t, completeArgs([]string{"--output=t"}, pluginRoot), []string{"--output=table"})
	assertEqualCompletions(t, completeArgs([]string{"--table", ""}, pluginRoot), []string{"bordered", "compact", "plain"})
	assertEqualCompletions(t, completeArgs([]string{"--lang", "en"}, pluginRoot), []string{"en-GB", "en-US"})
	assertHasCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--status", ""}, pluginRoot), "running", "stopped")
	assertHasCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--cols", ""}, pluginRoot), "instance_id")
	assertHasCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--filter", ""}, pluginRoot), "instance_id=")
	assertHasCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--sort", ""}, pluginRoot), "instance_id", "-instance_id")
	assertHasCompletions(t, completeArgs([]string{"ecs", "instance", "show", "ins-demo-1", "--wait", ""}, waitRoot), "ecs.instance.running")
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--missing=x"}, pluginRoot), nil)
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--config", ""}, pluginRoot), nil)
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--status=running"}, pluginRoot), []string{"--status=running"})
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--status=missing"}, pluginRoot), nil)
	assertNoCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--status=running", ""}, pluginRoot), "--status")
	assertNoCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--status=running", "--"}, pluginRoot), "--status")
}

// TestHiddenCompletionCommandPrintsOneCandidatePerLine keeps hidden completion output consumable by shell scripts.
func TestHiddenCompletionCommandPrintsOneCandidatePerLine(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"__complete", "completion", ""},
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("__complete returned error: %v", err)
	}
	got := strings.Fields(stdout.String())
	assertHasCompletions(t, got, "bash", "zsh", "fish", "powershell")
}

// TestHiddenCompletionCoversCoreBranchesAndPluginOptions keeps each command group limited to its applicable completions.
func TestHiddenCompletionCoversCoreBranchesAndPluginOptions(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	waitRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(waitRoot, "ecs"))

	assertHasCompletions(t, completeArgs([]string{"doctor", ""}, pluginRoot), "network")
	doctorOptions := completeArgs([]string{"doctor", "network", ""}, pluginRoot)
	assertHasCompletions(t, doctorOptions, "--source", "--timeout")
	assertNoCompletions(t, doctorOptions, "--offline", "--fixture", "-O")
	assertHasCompletions(t, completeArgs([]string{"doctor", "network", "--source", ""}, pluginRoot), "auto", "gitee", "github")
	assertHasCompletions(t, completeArgs([]string{"completion", "bash", ""}, pluginRoot), "--help")
	assertHasCompletions(t, completeArgs([]string{"help", ""}, pluginRoot), "ecs", "plugin")
	assertHasCompletions(t, completeArgs([]string{"help", "ecs", ""}, pluginRoot), "instance")
	pluginCompletions := completeArgs([]string{"plugin", ""}, pluginRoot)
	assertHasCompletions(t, pluginCompletions, "install", "list", "reinstall", "upgrade")
	assertNoCompletions(t, pluginCompletions, "lint")
	installCompletions := completeArgs([]string{"plugin", "install", ""}, pluginRoot)
	assertHasCompletions(t, installCompletions, "--source", "--channel")
	assertNoCompletions(t, installCompletions, "--bundled")
	assertHasCompletions(t, completeArgs([]string{"plugin", "install", "--channel", ""}, pluginRoot), "alpha", "beta", "stable")
	assertHasCompletions(t, completeArgs([]string{"plugin", "install", "--source", ""}, pluginRoot), "auto", "gitee", "github")
	assertHasCompletions(t, completeArgs([]string{"plugin", "list", ""}, pluginRoot), "--updates", "--source")
	updateCompletions := completeArgs([]string{"plugin", "update", ""}, pluginRoot)
	assertHasCompletions(t, updateCompletions, "--all", "--source")
	assertNoCompletions(t, updateCompletions, "--bundled")
	reinstallCompletions := completeArgs([]string{"plugin", "reinstall", ""}, pluginRoot)
	assertHasCompletions(t, reinstallCompletions, "--all", "--source")
	assertNoCompletions(t, reinstallCompletions, "--bundled")
	assertHasCompletions(t, completeArgs([]string{"upgrade", ""}, pluginRoot), "--check", "--source", "--channel")
	assertHasCompletions(t, completeArgs([]string{"upgrade", "--source", ""}, pluginRoot), "auto", "gitee", "github")
	assertHasCompletions(t, completeArgs([]string{"upgrade", "--channel", ""}, pluginRoot), "alpha", "beta", "stable")
	assertEqualCompletions(t, completeArgs([]string{"plugin", "unknown", ""}, pluginRoot), nil)
	assertHasCompletions(t, completeArgs([]string{"version", ""}, pluginRoot), "--help")
	assertHasCompletions(t, completeArgs([]string{"ecs", "instance", "show", "ins-demo-1", ""}, waitRoot), "--cols", "--wait")
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", "show", ""}, waitRoot), nil)
	defaultInstanceChildren := completeArgs([]string{"ecs", "instance", ""}, pluginRoot)
	assertHasCompletions(t, defaultInstanceChildren, "list")
	assertEqualCompletions(t, completeArgs([]string{"ecs", "wrong", ""}, pluginRoot), nil)
}

// TestPluginCompletionOptionsCoverAllSubcommands exposes declared plugin-manager choices without development-only flags.
func TestPluginCompletionOptionsCoverAllSubcommands(t *testing.T) {
	pluginRoot := t.TempDir()
	disableDevelopmentBundledPluginsForTest(t)
	for _, subcommand := range []string{"install", "search", "list", "update", "upgrade"} {
		options := pluginCompletionOptions(subcommand)
		if len(options) == 0 {
			t.Fatalf("pluginCompletionOptions(%q) returned no options", subcommand)
		}
		for _, option := range options {
			if option.Values == nil {
				continue
			}
			values := option.Values(completionContext{})
			if len(values) == 0 {
				t.Fatalf("pluginCompletionOptions(%q) option %v returned no values", subcommand, option.Names)
			}
		}
	}
	if got := pluginCompletionOptions("missing"); got != nil {
		t.Fatalf("pluginCompletionOptions missing = %#v, want nil", got)
	}
	assertHasCompletions(t, completeArgs([]string{"plugin", "list", "--channel", ""}, pluginRoot), "all", "alpha", "beta", "stable")
	assertHasCompletions(t, completeArgs([]string{"plugin", "search", "--channel", ""}, pluginRoot), "all", "alpha", "beta", "stable")
	assertNoCompletions(t, completeArgs([]string{"plugin", "install", "--channel", ""}, pluginRoot), "all")
}

// TestCompletionInternalsCoverNoSuggestionPaths returns no candidates for invalid paths and consumed options.
func TestCompletionInternalsCoverNoSuggestionPaths(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	if err := runComplete(failingWriter{}, []string{"completion", ""}, pluginRoot); err == nil {
		t.Fatal("runComplete returned nil error for writer failure")
	}
	assertEqualCompletions(t, completeArgs([]string{"ecs"}, pluginRoot), []string{"ecs"})
	group := completeArgs([]string{"--output=table", "ecs", ""}, pluginRoot)
	assertHasCompletions(t, group, "instance", "--help")
	assertNoCompletions(t, group, "--output", "-o")
	assertNoCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--name", "demo", ""}, pluginRoot), "--name")
	assertEqualCompletions(t, tableColumnKeys(completionContext{}, ""), nil)
	assertEqualCompletions(t, tableColumnKeys(completionContext{CommandFound: true}, ""), nil)
	if completionPathMatches([]string{"ecs"}, []string{"ecs", "instance"}) {
		t.Fatal("completionPathMatches matched a path longer than the pattern")
	}
	if completionPathMatches([]string{"ecs", "{instance_id}"}, []string{"ecs", ""}) {
		t.Fatal("completionPathMatches matched an empty placeholder")
	}
	if !completionPathMatches([]string{"ecs", "{instance_id}"}, []string{"ecs", "ins-demo-1"}) {
		t.Fatal("completionPathMatches did not match a non-empty placeholder")
	}
	if _, ok := inlineOptionValueCompletions("plain", completionContext{}); ok {
		t.Fatal("inlineOptionValueCompletions treated a plain word as an option")
	}
	if _, ok := inlineOptionValueCompletions("--unknown=value", completionContext{}); !ok {
		t.Fatal("inlineOptionValueCompletions should handle unknown inline options")
	}
	assertHasCompletions(t, allCompletionWords(pluginRoot), "--all", "--updates", "ecs")
}

// assertHasCompletions requires every requested candidate in a completion result.
func assertHasCompletions(t *testing.T, got []string, wants ...string) {
	t.Helper()
	seen := make(map[string]bool, len(got))
	for _, value := range got {
		seen[value] = true
	}
	for _, want := range wants {
		if !seen[want] {
			t.Fatalf("completion candidates missing %q in %v", want, got)
		}
	}
}

// assertNoCompletions rejects each forbidden candidate from a completion result.
func assertNoCompletions(t *testing.T, got []string, unwanted ...string) {
	t.Helper()
	seen := make(map[string]bool, len(got))
	for _, value := range got {
		seen[value] = true
	}
	for _, value := range unwanted {
		if seen[value] {
			t.Fatalf("completion candidates unexpectedly include %q in %v", value, got)
		}
	}
}

// assertEqualCompletions compares complete ordered completion candidate lists.
func assertEqualCompletions(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("completion candidates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("completion candidates = %v, want %v", got, want)
		}
	}
}

// TestCompletionConfigSecretOptionPrefix keeps command-owned options available while typing.
func TestCompletionConfigSecretOptionPrefix(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	for _, group := range []string{"profile", "profiles"} {
		got := completeArgs([]string{"config", group, "set-secret", "demo", "ak", "--from"}, t.TempDir())
		if len(got) != 1 || got[0] != "--from-stdin" {
			t.Fatalf("completion = %v", got)
		}
		got = completeArgs([]string{"config", group, "set-secret", "demo", "ak", "--from-stdin", "--from"}, t.TempDir())
		if len(got) != 0 {
			t.Fatalf("used option repeated: %v", got)
		}
	}
}

// TestCompletionPreservesProductOperands checks paths and option values against the executable command grammar.
func TestCompletionPreservesProductOperands(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "region")
	writeArgumentRegionOptionBundle(t, dir)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[
		{"id":"region.show","path":["region","show","{region_id}"],"operation":"v4.region.show","table":"region.show","parameters":[{"name":"view","flag":"view","target":"view","allowed_values":["short","long"]}]},
		{"id":"resource.show","path":["region","resource","{id}","show"],"operation":"v4.region.show","table":"region.show","parameters":[{"name":"view","flag":"view","target":"view","allowed_values":["short","long"]}]},
		{"id":"resource","path":["region","resource","{id}"],"operation":"v4.region.show","table":"region.show","parameters":[{"name":"view","flag":"view","target":"view","allowed_values":["short","long"]},{"name":"label","flag":"label","target":"label"}]}
	]}`)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"region", "show", "--vi"}, "--view"},
		{[]string{"region", "show", ""}, "--view"},
		{[]string{"region", "show", "--view", "l"}, "long"},
		{[]string{"region", "show", "id", "--view=s"}, "--view=short"},
		{[]string{"region", "resource", "-12", "--vi"}, "--view"},
		{[]string{"region", "resource", "--view", "--vi"}, "--view"},
		{[]string{"region", "resource", "--view", ""}, "show"},
		{[]string{"region", "resource", "plain", "s"}, "show"},
		{[]string{"help", "region", "resource", "--view", "s"}, "show"},
		{[]string{"region", "resource", "--view", "show", "--view", "l"}, "long"},
		{[]string{"region", "resource", "id", "--label", "--view", "--vi"}, "--view"},
		{[]string{"region", "resource", "id", "--label", "--view", ""}, "--view"},
		{[]string{"--config", "--view", "region", "resource", "id", "--vi"}, "--view"},
		{[]string{"region", "resource", "id", "--label", "--view", "--view", "s"}, "short"},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var out bytes.Buffer
			if err := Run(Config{PluginRoot: root, Args: append([]string{"__complete"}, tc.args...), Stdout: &out}); err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(strings.Fields(out.String()), tc.want) {
				t.Errorf("%q: missing %q in %q", tc.args, tc.want, out.String())
			}
		})
	}
	for _, args := range [][]string{
		{"region", "resource", "id", "extra", "--vi"},
		{"region", "resource", "id", "--view=long", "--vi"},
		{"region", "resource", "id", "--label", "--view", "s"},
	} {
		assertEqualCompletions(t, completeArgs(args, root), nil)
	}
}

// TestCompletionUsesOnlyExecutableOptionForms checks inline spelling and irreparable plugin command tails.
func TestCompletionUsesOnlyExecutableOptionForms(t *testing.T) {
	root := t.TempDir()
	disableDevelopmentBundledPluginsForTest(t)
	for _, args := range [][]string{
		{"version", "-l=en"}, {"version", "-l=en-US", ""},
		{"plugin", "list", "extra", "--"}, {"plugin", "list", "extra", "--channel", ""},
		{"plugins", "search", "first", "second", ""}, {"plugin", "upgrade", "first", "second", "--"},
		{"plugin", "install", "--unknown", ""}, {"plugin", "list", "--available=true", ""},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			assertEqualCompletions(t, completeArgs(args, root), nil)
		})
	}
	for _, args := range [][]string{{"version", "--lang=en"}, {"version", "-l", "en"}} {
		got := completeArgs(args, root)
		if len(got) != 2 || !strings.HasSuffix(got[1], "en-US") {
			t.Fatalf("valid inline or separate spelling %q: %q", args, got)
		}
	}
	for _, subcommand := range []string{"install", "reinstall", "remove"} {
		assertHasCompletions(t, completeArgs([]string{"plugins", subcommand, "a", "b", "--"}, root), "--help")
	}
	assertHasCompletions(t, completeArgs([]string{"plugins", "list", "--bundled", "--"}, root), "--channel")
	assertHasCompletions(t, completeArgs([]string{"plugin", "search", "--channel", ""}, root), "beta")
	t.Cleanup(patchVersion(testCoreVersion()))
	assertEqualCompletions(t, completeArgs([]string{"plugins", "list", "--bundled", "--"}, root), nil)
}

// TestProductCompletionAfterDevelopmentSwitch keeps normal options available after an accepted hidden switch.
func TestProductCompletionAfterDevelopmentSwitch(t *testing.T) {
	root := t.TempDir()
	writeValidationBundle(t, filepath.Join(root, "ecs"))
	for _, option := range []string{"--offline", "--fixture"} {
		for _, prefix := range []string{"", "--"} {
			got := completeArgs([]string{"ecs", "instance", "list", option, prefix}, root)
			assertHasCompletions(t, got, "--status", "--output")
			assertNoCompletions(t, got, "--offline", "--fixture")
		}
	}
	for _, path := range [][]string{{"--offline", "ecs", "instance", "list", "--"}, {"ecs", "--fixture", "instance", "list", "--"}} {
		assertEqualCompletions(t, completeArgs(path, root), nil)
	}
	t.Cleanup(patchVersion(testCoreVersion()))
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--fixture", "--"}, root), nil)
}
