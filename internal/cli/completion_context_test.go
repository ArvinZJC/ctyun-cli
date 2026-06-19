/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionScriptSupportsPowerShell(t *testing.T) {
	var stdout bytes.Buffer
	if err := runCompletion(&stdout, []string{"powershell"}, t.TempDir()); err != nil {
		t.Fatalf("runCompletion powershell returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Register-ArgumentCompleter", "ctyun", "__complete"} {
		if !strings.Contains(got, want) {
			t.Fatalf("PowerShell completion script missing %q:\n%s", want, got)
		}
	}
}

func TestHiddenCompletionUsesContextAwareCommandTree(t *testing.T) {
	pluginRoot := t.TempDir()
	writeVPCBundle(t, filepath.Join(pluginRoot, "vpc"))

	top := completeArgs(nil, pluginRoot)
	assertHasCompletions(t, top, "doctor", "ecs", "vpc")
	assertNoCompletions(t, top, "instance", "subnet")

	ecsChildren := completeArgs([]string{"ecs", ""}, pluginRoot)
	assertHasCompletions(t, ecsChildren, "instance")
	assertNoCompletions(t, ecsChildren, "subnet", "doctor")

	instanceChildren := completeArgs([]string{"ecs", "instance", ""}, pluginRoot)
	assertHasCompletions(t, instanceChildren, "list", "show", "start")
	assertNoCompletions(t, instanceChildren, "ecs", "doctor")
}

func TestHiddenCompletionSuggestsOptionsForResolvedCommand(t *testing.T) {
	got := completeArgs([]string{"ecs", "instance", "list", ""}, defaultPluginRoot())
	assertHasCompletions(t, got, "--name", "--cols", "--filter", "--sort", "--output", "-o")
	assertNoCompletions(t, got, "show", "start")
}

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

func TestHiddenCompletionCommandPrintsOneCandidatePerLine(t *testing.T) {
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

func TestHiddenCompletionCoversCoreBranchesAndPluginOptions(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	waitRoot := t.TempDir()
	writeWaitBundle(t, filepath.Join(waitRoot, "ecs"))

	assertHasCompletions(t, completeArgs([]string{"doctor", ""}, pluginRoot), "network")
	assertEqualCompletions(t, completeArgs([]string{"doctor", "network", ""}, pluginRoot), nil)
	assertEqualCompletions(t, completeArgs([]string{"completion", "bash", ""}, pluginRoot), nil)
	assertHasCompletions(t, completeArgs([]string{"help", ""}, pluginRoot), "ecs", "plugin")
	assertHasCompletions(t, completeArgs([]string{"help", "ecs", ""}, pluginRoot), "instance")
	assertHasCompletions(t, completeArgs([]string{"plugin", ""}, pluginRoot), "install", "list", "upgrade")
	assertHasCompletions(t, completeArgs([]string{"plugin", "install", ""}, pluginRoot), "--source", "--channel", "--bundled")
	assertHasCompletions(t, completeArgs([]string{"plugin", "install", "--channel", ""}, pluginRoot), "beta", "edge", "stable")
	assertHasCompletions(t, completeArgs([]string{"plugin", "install", "--source", ""}, pluginRoot), "auto", "gitee", "github")
	assertHasCompletions(t, completeArgs([]string{"plugin", "list", ""}, pluginRoot), "--updates", "--source")
	assertHasCompletions(t, completeArgs([]string{"plugin", "update", ""}, pluginRoot), "--all", "--source", "--bundled")
	assertHasCompletions(t, completeArgs([]string{"upgrade", ""}, pluginRoot), "--check", "--source", "--channel")
	assertHasCompletions(t, completeArgs([]string{"upgrade", "--source", ""}, pluginRoot), "auto", "gitee", "github")
	assertHasCompletions(t, completeArgs([]string{"upgrade", "--channel", ""}, pluginRoot), "beta", "edge", "stable")
	assertEqualCompletions(t, completeArgs([]string{"plugin", "unknown", ""}, pluginRoot), nil)
	assertEqualCompletions(t, completeArgs([]string{"version", ""}, pluginRoot), nil)
	assertHasCompletions(t, completeArgs([]string{"ecs", "instance", "show", "ins-demo-1", ""}, waitRoot), "--cols", "--wait")
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", "show", ""}, waitRoot), nil)
	assertEqualCompletions(t, completeArgs([]string{"ecs", "instance", ""}, defaultPluginRoot()), []string{"list", "show", "start"})
	assertEqualCompletions(t, completeArgs([]string{"ecs", "wrong", ""}, pluginRoot), nil)
}

func TestPluginCompletionOptionsCoverAllSubcommands(t *testing.T) {
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
}

func TestCompletionInternalsCoverNoSuggestionPaths(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))

	if err := runComplete(failingWriter{}, []string{"completion", ""}, pluginRoot); err == nil {
		t.Fatal("runComplete returned nil error for writer failure")
	}
	assertEqualCompletions(t, completeArgs([]string{"ecs"}, pluginRoot), []string{"ecs"})
	assertEqualCompletions(t, completeArgs([]string{"--output=table", "ecs", ""}, pluginRoot), []string{"instance"})
	assertNoCompletions(t, completeArgs([]string{"ecs", "instance", "list", "--name", "demo", ""}, pluginRoot), "--name")
	if got, ok := pendingOptionValueCompletions([]string{"--output=table"}, "", completionContext{}); ok || len(got) != 0 {
		t.Fatalf("pendingOptionValueCompletions inline option = %v, %v; want no match", got, ok)
	}
	if got, ok := pendingOptionValueCompletions(nil, "", completionContext{}); ok || len(got) != 0 {
		t.Fatalf("pendingOptionValueCompletions nil tokens = %v, %v; want no match", got, ok)
	}
	assertEqualCompletions(t, tableColumnKeys(completionContext{}, ""), nil)
	assertEqualCompletions(t, tableColumnKeys(completionContext{CommandFound: true}, ""), nil)
	assertNoCompletions(t, usedOptionKeysForTest([]string{"value"}), "value")
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
	if _, ok := pendingOptionValueCompletions([]string{"plain"}, "", completionContext{}); ok {
		t.Fatal("pendingOptionValueCompletions treated a plain word as an option")
	}
	assertHasCompletions(t, allCompletionWords(pluginRoot), "--all", "--updates", "ecs")
}

func usedOptionKeysForTest(tokens []string) []string {
	used := usedCompletionOptions(tokens)
	keys := make([]string, 0, len(used))
	for key := range used {
		keys = append(keys, key)
	}
	sortStrings(keys)
	return keys
}

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
