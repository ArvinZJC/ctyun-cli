/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// TestBareCommandGroupsRenderHelp treats each valid incomplete group as successful help output.
func TestBareCommandGroupsRenderHelp(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	for _, args := range [][]string{
		{"config"},
		{"config", "profile"},
		{"doctor"},
		{"plugin"},
		{"plugins"},
		{"ecs"},
		{"ecs", "instance"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			if err := Run(Config{PluginRoot: pluginRoot, Args: args, Stdout: &stdout, Stderr: io.Discard}); err != nil {
				t.Fatalf("Run(%v) returned error: %v", args, err)
			}
			if !strings.Contains(stdout.String(), "Usage:") {
				t.Fatalf("Run(%v) did not render help:\n%s", args, stdout.String())
			}
		})
	}
}

// TestInvalidLeafInputUsesGenericDiagnostics classifies leaf arity and option failures consistently across command owners.
func TestInvalidLeafInputUsesGenericDiagnostics(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	tests := []struct {
		name string
		args []string
		key  string
	}{
		{name: "version positional", args: []string{"version", "extra"}, key: "error.unexpected_argument"},
		{name: "version option", args: []string{"version", "-e"}, key: "error.unknown_option"},
		{name: "completion missing shell", args: []string{"completion"}, key: "error.missing_required_argument"},
		{name: "completion extra shell", args: []string{"completion", "bash", "extra"}, key: "error.unexpected_argument"},
		{name: "config path positional", args: []string{"config", "path", "extra"}, key: "error.unexpected_argument"},
		{name: "config reset positional", args: []string{"config", "reset", "extra", "--yes"}, key: "error.unexpected_argument"},
		{name: "config list positional", args: []string{"config", "profile", "list", "extra"}, key: "error.unexpected_argument"},
		{name: "config set-secret option", args: []string{"config", "profile", "set-secret", "prod", "ak", "--bad"}, key: "error.unknown_option"},
		{name: "config set-secret positional", args: []string{"config", "profile", "set-secret", "prod", "--from-stdin"}, key: "error.missing_required_argument"},
		{name: "doctor positional", args: []string{"doctor", "network", "test"}, key: "error.unexpected_argument"},
		{name: "doctor option", args: []string{"doctor", "network", "--bad"}, key: "error.unknown_option"},
		{name: "upgrade option", args: []string{"upgrade", "--bad"}, key: "error.unknown_option"},
		{name: "upgrade positional", args: []string{"upgrade", "extra"}, key: "error.unexpected_argument"},
		{name: "plugin list option", args: []string{"plugin", "list", "--bad"}, key: "error.unknown_option"},
		{name: "plugin list positional", args: []string{"plugin", "list", "extra"}, key: "error.unexpected_argument"},
		{name: "plugin install option", args: []string{"plugin", "install", "--bad"}, key: "error.unknown_option"},
		{name: "plugin lint option", args: []string{"plugin", "lint", "-e"}, key: "error.unknown_option"},
		{name: "plugin lint missing path", args: []string{"plugin", "lint"}, key: "error.missing_required_argument"},
		{name: "plugin lint extra path", args: []string{"plugin", "lint", ".", "extra"}, key: "error.unexpected_argument"},
		{name: "plugin remove bundled", args: []string{"plugin", "remove", "ecs", "--bundled"}, key: "error.unknown_option"},
		{name: "product long option", args: []string{"ecs", "instance", "list", "--bad"}, key: "error.unknown_option"},
		{name: "product inline long option", args: []string{"ecs", "instance", "list", "--bad=value"}, key: "error.unknown_option"},
		{name: "product short option", args: []string{"ecs", "instance", "list", "-e"}, key: "error.unknown_option"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Run(Config{PluginRoot: pluginRoot, Args: tc.args, Stdout: io.Discard, Stderr: io.Discard})
			requireDiagnosticKey(t, err, tc.key)
		})
	}
}

// TestCommandBoundariesClassifyOptionsAndWordsConsistently distinguishes unknown options from unknown visible command paths.
func TestCommandBoundariesClassifyOptionsAndWordsConsistently(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	optionCases := [][]string{
		{"config", "-e"},
		{"config", "profile", "-e"},
		{"config", "profiles", "-e"},
		{"doctor", "-e"},
		{"plugin", "-e"},
		{"plugins", "-e"},
		{"help", "-e"},
		{"completion", "-e"},
		{"ecs", "-e"},
		{"ecs", "instance", "-e"},
	}
	for _, args := range optionCases {
		t.Run("option_"+strings.Join(args, "_"), func(t *testing.T) {
			err := Run(Config{PluginRoot: pluginRoot, Args: args, Stdout: io.Discard, Stderr: io.Discard})
			requireDiagnosticKey(t, err, "error.unknown_option")
			if got := formatError(err, "en-US"); got != `Error: unknown option "-e"` {
				t.Fatalf("formatted option error = %q", got)
			}
		})
	}

	wordCases := [][]string{
		{"config", "missing"},
		{"config", "profile", "missing"},
		{"config", "profiles", "missing"},
		{"doctor", "missing"},
		{"plugin", "missing"},
		{"plugins", "missing"},
		{"help", "missing"},
		{"ecs", "missing"},
		{"ecs", "instance", "missing"},
	}
	for _, args := range wordCases {
		t.Run("word_"+strings.Join(args, "_"), func(t *testing.T) {
			err := Run(Config{PluginRoot: pluginRoot, Args: args, Stdout: io.Discard, Stderr: io.Discard})
			requireDiagnosticKey(t, err, "error.unknown_command")
			path := strings.Join(args, " ")
			if args[0] == "help" {
				path = strings.Join(args[1:], " ")
			}
			if got := formatError(err, "en-US"); got != `Error: unknown command "`+path+`"` {
				t.Fatalf("formatted command error = %q", got)
			}
		})
	}
}

// TestHelpLeafInputUsesGenericDiagnostics applies normal leaf argument validation while rendering help.
func TestHelpLeafInputUsesGenericDiagnostics(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	tests := []struct {
		name string
		args []string
		key  string
	}{
		{name: "config option", args: []string{"help", "config", "show", "-e"}, key: "error.unknown_option"},
		{name: "config positional", args: []string{"help", "config", "show", "extra"}, key: "error.unexpected_argument"},
		{name: "profile option", args: []string{"help", "config", "profile", "list", "-e"}, key: "error.unknown_option"},
		{name: "profile positional", args: []string{"help", "config", "profile", "list", "extra"}, key: "error.unexpected_argument"},
		{name: "doctor option", args: []string{"help", "doctor", "network", "-e"}, key: "error.unknown_option"},
		{name: "doctor positional", args: []string{"help", "doctor", "network", "extra"}, key: "error.unexpected_argument"},
		{name: "plugin option", args: []string{"help", "plugin", "list", "-e"}, key: "error.unknown_option"},
		{name: "plugin positional", args: []string{"help", "plugin", "list", "extra"}, key: "error.unexpected_argument"},
		{name: "completion option", args: []string{"help", "completion", "-e"}, key: "error.unknown_option"},
		{name: "completion positional", args: []string{"help", "completion", "extra"}, key: "error.unexpected_argument"},
		{name: "help option", args: []string{"help", "help", "-e"}, key: "error.unknown_option"},
		{name: "help positional", args: []string{"help", "help", "extra"}, key: "error.unexpected_argument"},
		{name: "upgrade option", args: []string{"help", "upgrade", "-e"}, key: "error.unknown_option"},
		{name: "upgrade positional", args: []string{"help", "upgrade", "extra"}, key: "error.unexpected_argument"},
		{name: "version option", args: []string{"help", "version", "-e"}, key: "error.unknown_option"},
		{name: "version positional", args: []string{"help", "version", "extra"}, key: "error.unexpected_argument"},
		{name: "product option", args: []string{"help", "ecs", "instance", "list", "-e"}, key: "error.unknown_option"},
		{name: "product positional", args: []string{"help", "ecs", "instance", "list", "extra"}, key: "error.unexpected_argument"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Run(Config{PluginRoot: pluginRoot, Args: tc.args, Stdout: io.Discard, Stderr: io.Discard})
			requireDiagnosticKey(t, err, tc.key)
		})
	}
}

// TestDevelopmentFixtureOptionsAreProductOwnedAndLongOnly accepts fixture switches only after a complete product path.
func TestDevelopmentFixtureOptionsAreProductOwnedAndLongOnly(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	mustWrite(t, filepath.Join(pluginRoot, "ecs", "fixtures", "list.json"), ` {"returnObj":{"instances":[{"instanceID":"synthetic-fixture"}]}} `)
	for _, args := range [][]string{
		{"--offline", "ecs", "instance", "list"},
		{"--fixture", "ecs", "instance", "list"},
		{"-O", "ecs", "instance", "list"},
		{"ecs", "instance", "list", "-O"},
		{"plugin", "list", "--offline"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			err := Run(Config{PluginRoot: pluginRoot, Args: args, Stdout: io.Discard, Stderr: io.Discard})
			requireDiagnosticKey(t, err, "error.unknown_option")
		})
	}

	for _, option := range []string{"--offline", "--fixture"} {
		var stdout bytes.Buffer
		if err := Run(Config{
			PluginRoot: pluginRoot,
			Args:       []string{"ecs", "instance", "list", option},
			Stdout:     &stdout,
			Stderr:     io.Discard,
		}); err != nil {
			t.Fatalf("product command %s returned error: %v", option, err)
		}
		if !strings.Contains(stdout.String(), "synthetic-fixture") {
			t.Fatalf("product command %s did not use fixture:\n%s", option, stdout.String())
		}
	}
}

// TestPluginSearchRequiresQuery rejects an empty search before reading bundled metadata.
func TestPluginSearchRequiresQuery(t *testing.T) {
	err := Run(Config{
		Args:   []string{"plugin", "search", "--bundled"},
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	requireDiagnosticKey(t, err, "error.missing_required_argument")
}

// TestInlineOptionValuesAreAcceptedConsistently accepts equals-separated values across shared command parsers.
func TestInlineOptionValuesAreAcceptedConsistently(t *testing.T) {
	global, rest, err := parseGlobalOptions([]string{"--lang=en-US", "version"})
	if err != nil || global.Language != "en-US" || len(rest) != 1 || rest[0] != "version" {
		t.Fatalf("parseGlobalOptions inline = %+v, %v, %v", global, rest, err)
	}
	doctor, err := parseDoctorNetworkOptions([]string{"network", "--source=github"})
	if err != nil || doctor.Source != "github" {
		t.Fatalf("parseDoctorNetworkOptions inline = %+v, %v", doctor, err)
	}
	upgrade, err := parseUpgradeOptions([]string{"--source=github", "--channel=beta"})
	if err != nil || upgrade.Source != "github" || upgrade.Channel != "beta" {
		t.Fatalf("parseUpgradeOptions inline = %+v, %v", upgrade, err)
	}
	plugin, err := parsePluginListOptions([]string{"--available", "--source=github", "--channel=beta"})
	if err != nil || plugin.Source != "github" || plugin.Channel != "beta" {
		t.Fatalf("parsePluginListOptions inline = %+v, %v", plugin, err)
	}
}

// TestSharedCommandTokenParserClassifiesOptions checks option aliases, missing values, and positional validation.
func TestSharedCommandTokenParserClassifiesOptions(t *testing.T) {
	parsed, err := parseCommandTokens([]string{"-S", "github", "query"}, []commandOption{
		{Name: "source", Aliases: []string{"-S"}, TakesValue: true},
	})
	if err != nil || parsed.Options["source"] != "github" || len(parsed.Positionals) != 1 || parsed.Positionals[0] != "query" {
		t.Fatalf("parseCommandTokens alias = %+v, %v", parsed, err)
	}
	if _, err := parseCommandTokens([]string{"--all=true"}, []commandOption{{Name: "all"}}); err == nil {
		t.Fatal("boolean option with inline value returned nil error")
	}
	if err := validatePositionalArguments([]string{"first"}, []string{"value"}, 2, 2); err == nil {
		t.Fatal("missing positional with a shorter name list returned nil error")
	}
	if opts, err := parseDoctorNetworkOptions(nil); err != nil || opts.Source != "" {
		t.Fatalf("parseDoctorNetworkOptions(nil) = %+v, %v", opts, err)
	}
	if globalOptionAllowed(nil, "output") {
		t.Fatal("output option applies without a command")
	}
	var marker interface{ silentExit() } = silentExitError{}
	marker.silentExit()
}

// TestHelpCompletionTraversesCoreCommandGroups offers child commands only at valid help command boundaries.
func TestHelpCompletionTraversesCoreCommandGroups(t *testing.T) {
	root := t.TempDir()
	disableDevelopmentBundledPluginsForTest(t)
	assertHasCompletions(t, completeArgs([]string{"help", "config", ""}, root), "path", "profile", "show")
	assertHasCompletions(t, completeArgs([]string{"help", "config", "profile", ""}, root), "list", "set", "use")
	assertHasCompletions(t, completeArgs([]string{"help", "doctor", ""}, root), "network")
	assertHasCompletions(t, completeArgs([]string{"help", "plugin", ""}, root), "install", "list", "update")
	if got := completeArgs([]string{"help", "doctor", "network", ""}, root); len(got) != 0 {
		t.Fatalf("completion below doctor network = %v, want none", got)
	}
	if got := completeArgs([]string{"help", "plugin", "list", ""}, root); len(got) != 0 {
		t.Fatalf("completion below plugin list = %v, want none", got)
	}
}

// TestVersionIsNotAdvertisedAsACommandOption keeps the top-level version switch out of command help and completion.
func TestVersionIsNotAdvertisedAsACommandOption(t *testing.T) {
	pluginRoot := t.TempDir()
	disableDevelopmentBundledPluginsForTest(t)
	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"config", "show", "--help"}, Stdout: &stdout}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "--version") {
		t.Fatalf("config show help advertises --version:\n%s", stdout.String())
	}
	assertNoCompletions(t, completeArgs([]string{"config", "show", "-"}, pluginRoot), "--version", "-v")
}

// TestConfigArityUsesGenericDiagnostics reports config argument counts through shared diagnostics.
func TestConfigArityUsesGenericDiagnostics(t *testing.T) {
	pluginRoot := t.TempDir()
	disableDevelopmentBundledPluginsForTest(t)
	tests := []struct {
		args []string
		key  string
	}{
		{args: []string{"config", "show", "-e"}, key: "error.unknown_option"},
		{args: []string{"config", "show", "extra"}, key: "error.unexpected_argument"},
		{args: []string{"config", "set", "region"}, key: "error.missing_required_argument"},
		{args: []string{"config", "unset"}, key: "error.missing_required_argument"},
		{args: []string{"config", "profile", "use"}, key: "error.missing_required_argument"},
		{args: []string{"config", "profile", "list", "--bad"}, key: "error.unknown_option"},
	}
	for _, tc := range tests {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			err := Run(Config{PluginRoot: pluginRoot, Args: tc.args, Stdout: io.Discard, Stderr: io.Discard})
			requireDiagnosticKey(t, err, tc.key)
		})
	}
}

// TestInapplicableSharedOptionsAreUnknown rejects shared flags outside their declared command scope.
func TestInapplicableSharedOptionsAreUnknown(t *testing.T) {
	pluginRoot := t.TempDir()
	disableDevelopmentBundledPluginsForTest(t)
	for _, args := range [][]string{
		{"version", "--output", "json"},
		{"completion", "bash", "--table", "plain"},
		{"config", "path", "--wait", "anything"},
	} {
		err := Run(Config{PluginRoot: pluginRoot, Args: args, Stdout: io.Discard, Stderr: io.Discard})
		requireDiagnosticKey(t, err, "error.unknown_option")
	}
}

// TestUnknownOptionWordingIsConsistentAcrossLanguages keeps the unknown option value quoted in localized diagnostics.
func TestUnknownOptionWordingIsConsistentAcrossLanguages(t *testing.T) {
	err := Run(Config{Args: []string{"config", "show", "-e"}, Stdout: io.Discard, Stderr: io.Discard})
	if got := formatError(err, "zh-CN"); got != `错误：未知选项 "-e"` {
		t.Fatalf("Chinese unknown-option diagnostic = %q", got)
	}
}

// TestReleasedBuildDevelopmentOptionsAreUnknown treats development switches as ordinary unknown options in releases.
func TestReleasedBuildDevelopmentOptionsAreUnknown(t *testing.T) {
	pluginRoot := t.TempDir()
	writeValidationBundle(t, filepath.Join(pluginRoot, "ecs"))
	restoreVersion := patchVersion("0.5.1")
	t.Cleanup(restoreVersion)
	for _, args := range [][]string{
		{"ecs", "instance", "list", "--offline"},
		{"ecs", "instance", "list", "--fixture"},
		{"plugin", "list", "--available", "--bundled"},
	} {
		err := Run(Config{PluginRoot: pluginRoot, Args: args, Stdout: io.Discard, Stderr: io.Discard})
		requireDiagnosticKey(t, err, "error.unknown_option")
	}
}

// TestBundledHelpersRejectReleasedBuilds rejects repository bundle access from released builds.
func TestBundledHelpersRejectReleasedBuilds(t *testing.T) {
	restoreVersion := patchVersion("0.5.1")
	t.Cleanup(restoreVersion)

	if _, err := bundledPluginSource("ecs"); err == nil {
		t.Fatal("bundledPluginSource returned nil error in a released build")
	}
	if _, err := bundledPluginIndex("en-US"); err == nil {
		t.Fatal("bundledPluginIndex returned nil error in a released build")
	}
	if err := listBundledAvailablePlugins(io.Discard, t.TempDir(), "", globalOptions{Language: "en-US"}); err == nil {
		t.Fatal("listBundledAvailablePlugins returned nil error in a released build")
	}
	if err := listBundledPluginUpdates(io.Discard, t.TempDir(), "", "en-US"); err == nil {
		t.Fatal("listBundledPluginUpdates returned nil error in a released build")
	}
	if err := installBundledPluginsWithProgress(io.Discard, io.Discard, t.TempDir(), []string{"ecs"}, false, "en-US"); err == nil {
		t.Fatal("installBundledPluginsWithProgress returned nil error in a released build")
	}
}
