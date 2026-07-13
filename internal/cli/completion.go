/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"strings"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// runCompletion implements the user-facing "ctyun completion <shell>" command.
// It prints installable shell glue; the glue calls the hidden __complete command
// so every shell shares the same Go resolver.
func runCompletion(stdout io.Writer, args []string) error {
	if err := validatePositionalArguments(args, []string{"shell"}, 1, 1); err != nil {
		return err
	}
	switch args[0] {
	case "zsh":
		return writeLines(stdout,
			"#compdef ctyun",
			"_ctyun() {",
			"  local -a completions",
			"  completions=(${(f)\"$(ctyun __complete \"${words[@]:2}\")\"})",
			"  compadd -- $completions",
			"}",
			"_ctyun \"$@\"",
		)
	case "bash":
		return writeLines(stdout,
			"_ctyun_completion() {",
			"  local IFS=$'\\n'",
			"  COMPREPLY=($(ctyun __complete \"${COMP_WORDS[@]:1}\"))",
			"}",
			"complete -F _ctyun_completion ctyun",
		)
	case "fish":
		return writeLines(stdout,
			"function __ctyun_complete",
			"  set -l words (commandline -opc)",
			"  if test (count $words) -gt 0",
			"    set -e words[1]",
			"  end",
			"  set -l current (commandline -ct)",
			"  if test -z \"$current\"",
			"    set -a words \"\"",
			"  else if test (count $words) -eq 0; or test \"$words[-1]\" != \"$current\"",
			"    set -a words \"$current\"",
			"  end",
			"  ctyun __complete $words",
			"end",
			"complete -c ctyun -f -a '(__ctyun_complete)'",
		)
	case "powershell":
		return writeLines(stdout,
			"Register-ArgumentCompleter -Native -CommandName ctyun -ScriptBlock {",
			"  param($wordToComplete, $commandAst, $cursorPosition)",
			"  $arguments = @()",
			"  foreach ($element in $commandAst.CommandElements) { $arguments += $element.Extent.Text }",
			"  if ($arguments.Count -gt 1) { $arguments = @($arguments[1..($arguments.Count - 1)]) } else { $arguments = @() }",
			"  $line = $commandAst.Extent.Text",
			"  $relativeCursor = [Math]::Min([Math]::Max($cursorPosition - $commandAst.Extent.StartOffset, 0), $line.Length)",
			"  if ($relativeCursor -gt 0 -and $line.Substring(0, $relativeCursor).EndsWith(' ')) { $arguments += '' }",
			"  ctyun __complete @arguments | Where-Object { $_ -like \"$wordToComplete*\" } | ForEach-Object {",
			"    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)",
			"  }",
			"}",
		)
	default:
		return diagnostic.New("error.unsupported_shell", args[0])
	}
}

// runComplete implements the hidden "__complete" command used by shell glue.
// It writes one already-filtered candidate per line for the active cursor.
func runComplete(stdout io.Writer, args []string, installedRoot string) error {
	for _, candidate := range completeArgs(args, installedRoot) {
		if err := writeLine(stdout, candidate); err != nil {
			return err
		}
	}
	return nil
}

// completeArgs is the shell-neutral completion resolver. It receives argv after
// "ctyun", including an empty final token when the cursor is after a space.
func completeArgs(args []string, installedRoot string) []string {
	tokens, prefix := splitCompletionInput(args)
	context := completionContextFor(tokens, installedRoot)
	if values, ok := inlineOptionValueCompletions(prefix, context); ok {
		return values
	}
	if values, ok := pendingOptionValueCompletions(tokens, prefix, context); ok {
		return values
	}
	if strings.HasPrefix(prefix, "-") {
		return optionCompletions(tokens, prefix, context)
	}
	path := completionPathTokens(tokens, installedRoot)
	return filterByPrefix(commandCompletions(path, context), prefix)
}

// completionContext keeps the original tokens and the option-stripped command
// path together, because option de-duplication and command matching need
// different views of the same partial argv.
type completionContext struct {
	InstalledRoot    string
	Bundles          []plugin.Bundle
	Tokens           []string
	Path             []string
	Bundle           plugin.Bundle
	Command          plugin.Command
	CommandFound     bool
	PluginSubcommand string
}

// completionOption describes one completable option and optional value source.
type completionOption struct {
	Names         []string
	RequiresValue bool
	Values        func(completionContext) []string
}

// splitCompletionInput separates completed tokens from the active prefix.
func splitCompletionInput(args []string) ([]string, string) {
	if len(args) == 0 {
		return nil, ""
	}
	if args[len(args)-1] == "" {
		return args[:len(args)-1], ""
	}
	return args[:len(args)-1], args[len(args)-1]
}

// completionContextFor builds the command and option context for tokens.
func completionContextFor(tokens []string, installedRoot string) completionContext {
	bundles := mustLoadBundlesForCompletion(installedRoot)
	path := completionPathTokens(tokens, installedRoot)
	context := completionContext{InstalledRoot: installedRoot, Bundles: bundles, Tokens: tokens, Path: path}
	if len(path) >= 2 && (path[0] == "plugin" || path[0] == "plugins") {
		for _, command := range pluginSubcommandSummaries() {
			if pluginSubcommandMatches(command, path[1]) {
				context.PluginSubcommand = path[1]
				break
			}
		}
	}
	for _, bundle := range bundles {
		if command, _, ok := plugin.FindCommandWithArgs(bundle, path); ok {
			context.Bundle = bundle
			context.Command = command
			context.CommandFound = true
			break
		}
	}
	return context
}

// inlineOptionValueCompletions completes values for --flag=value prefixes.
func inlineOptionValueCompletions(prefix string, context completionContext) ([]string, bool) {
	name, valuePrefix, ok := strings.Cut(prefix, "=")
	if !ok || !strings.HasPrefix(name, "-") {
		return nil, false
	}
	option, ok := findCompletionOption(name, context)
	if !ok || !option.RequiresValue || option.Values == nil {
		return nil, true
	}
	values := make([]string, 0)
	for _, value := range option.Values(context) {
		if strings.HasPrefix(value, valuePrefix) {
			values = append(values, name+"="+value)
		}
	}
	sortStrings(values)
	return values, true
}

// pendingOptionValueCompletions completes values after a separate --flag token.
func pendingOptionValueCompletions(tokens []string, prefix string, context completionContext) ([]string, bool) {
	if len(tokens) == 0 {
		return nil, false
	}
	last := tokens[len(tokens)-1]
	if strings.Contains(last, "=") {
		return nil, false
	}
	option, ok := findCompletionOption(last, context)
	if !ok || !option.RequiresValue {
		return nil, false
	}
	if option.Values == nil {
		return nil, true
	}
	return filterByPrefix(option.Values(context), prefix), true
}

// optionCompletions returns unused options matching prefix.
func optionCompletions(tokens []string, prefix string, context completionContext) []string {
	used := usedCompletionOptions(tokens)
	candidates := make([]string, 0)
	for _, option := range completionOptions(context) {
		skip := false
		for _, name := range option.Names {
			if used[name] {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		candidates = append(candidates, option.Names...)
	}
	return filterByPrefix(candidates, prefix)
}

// commandCompletions returns command words or option names for the current path.
func commandCompletions(path []string, context completionContext) []string {
	if len(path) == 0 {
		return topLevelCompletionCommands(context.Bundles)
	}
	switch path[0] {
	case "config":
		return configCommandCompletions(path)
	case "completion":
		if len(path) == 1 {
			return completionShells()
		}
		return nil
	case "doctor":
		if len(path) == 1 {
			return []string{"local", "network"}
		}
		if len(path) == 2 && (path[1] == "local" || path[1] == "network") {
			return optionCompletions(context.Tokens, "", context)
		}
		return nil
	case "help":
		return helpCompletionCommands(path[1:], context)
	case "plugin", "plugins":
		if len(path) == 1 {
			return pluginCompletionSubcommands()
		}
		if context.PluginSubcommand != "" {
			return optionCompletions(context.Tokens, "", context)
		}
		return nil
	case "update", "upgrade":
		return optionCompletions(context.Tokens, "", context)
	case "version":
		return nil
	}
	if context.CommandFound {
		return optionCompletions(context.Tokens, "", context)
	}
	return nextPluginPathCompletions(path, context.Bundles)
}

// configCommandCompletions returns static completions for config subcommands.
func configCommandCompletions(path []string) []string {
	if len(path) == 1 {
		return configCompletionSubcommands()
	}
	switch path[1] {
	case "explain":
		if len(path) == 2 {
			keys := coreconfig.SettingKeys()
			sortStrings(keys)
			return keys
		}
	case "profile", "profiles":
		if len(path) == 2 {
			return configProfileCompletionSubcommands()
		}
		if len(path) == 4 && path[2] == "set-secret" {
			return configSecretKeys()
		}
		if len(path) == 5 && path[2] == "set-secret" && validConfigSecretKey(path[4]) {
			return configProfileCompletionOptionNames(path[2])
		}
	}
	return nil
}

// configCompletionSubcommands returns config subcommands and aliases.
func configCompletionSubcommands() []string {
	return configCompletionNames(configSubcommandSummaries())
}

// configProfileCompletionSubcommands returns config profile subcommands and
// aliases.
func configProfileCompletionSubcommands() []string {
	return configCompletionNames(configProfileSubcommandSummaries())
}

// configCompletionNames returns command names and aliases from help metadata.
func configCompletionNames(commands []configSubcommandHelp) []string {
	seen := make(map[string]struct{})
	for _, command := range commands {
		seen[command.Name] = struct{}{}
		for _, alias := range command.Aliases {
			seen[alias] = struct{}{}
		}
	}
	return sortedCompletionSet(seen)
}

// configCompletionOptionNames returns option names from config help metadata.
func configCompletionOptionNames() []string {
	seen := make(map[string]struct{})
	addConfigCompletionOptionNames(seen, configSubcommandSummaries())
	addConfigCompletionOptionNames(seen, configProfileSubcommandSummaries())
	return sortedCompletionSet(seen)
}

// configProfileCompletionOptionNames returns options for one profile
// subcommand.
func configProfileCompletionOptionNames(subcommand string) []string {
	for _, command := range configProfileSubcommandSummaries() {
		if configSubcommandMatches(command, subcommand) {
			return configOptionNames(command.Options)
		}
	}
	return nil
}

// addConfigCompletionOptionNames adds option names from config commands.
func addConfigCompletionOptionNames(seen map[string]struct{}, commands []configSubcommandHelp) {
	for _, command := range commands {
		for _, name := range configOptionNames(command.Options) {
			seen[name] = struct{}{}
		}
	}
}

// configOptionNames returns names from option summary metadata.
func configOptionNames(options []pluginOptionSummary) []string {
	names := make([]string, 0, len(options))
	for _, option := range options {
		names = append(names, option.Name)
	}
	sortStrings(names)
	return names
}

// helpCompletionCommands returns command paths valid after ctyun help.
func helpCompletionCommands(path []string, context completionContext) []string {
	if len(path) == 0 {
		return topLevelCompletionCommands(context.Bundles)
	}
	switch path[0] {
	case "config":
		return configCommandCompletions(path)
	case "doctor":
		if len(path) == 1 {
			return []string{"local", "network"}
		}
		return nil
	case "plugin", "plugins":
		if len(path) == 1 {
			return pluginCompletionSubcommands()
		}
		return nil
	}
	return nextPluginPathCompletions(path, context.Bundles)
}

// topLevelCompletionCommands returns core commands plus top-level plugin words.
func topLevelCompletionCommands(bundles []plugin.Bundle) []string {
	seen := completionSet(coreCompletionCommands()...)
	for _, bundle := range bundles {
		for _, command := range bundle.Commands.Commands {
			if len(command.Path) > 0 && !isPathPlaceholder(command.Path[0]) {
				seen[command.Path[0]] = struct{}{}
			}
		}
	}
	return sortedCompletionSet(seen)
}

// nextPluginPathCompletions returns the next literal plugin path segment.
func nextPluginPathCompletions(path []string, bundles []plugin.Bundle) []string {
	seen := make(map[string]struct{})
	for _, bundle := range bundles {
		for _, command := range bundle.Commands.Commands {
			if len(path) >= len(command.Path) || !completionPathMatches(command.Path, path) {
				continue
			}
			next := command.Path[len(path)]
			if !isPathPlaceholder(next) {
				seen[next] = struct{}{}
			}
		}
	}
	return sortedCompletionSet(seen)
}

// completionPathMatches reports whether path can still match pattern.
func completionPathMatches(pattern, path []string) bool {
	if len(path) > len(pattern) {
		return false
	}
	for i, part := range path {
		if isPathPlaceholder(pattern[i]) {
			if part == "" {
				return false
			}
			continue
		}
		if pattern[i] != part {
			return false
		}
	}
	return true
}

// completionPathTokens strips options and option values from completion tokens.
func completionPathTokens(tokens []string, installedRoot string) []string {
	requiresValue := completionOptionValueNames(installedRoot)
	path := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if strings.HasPrefix(token, "-") {
			name := completionOptionName(token)
			if requiresValue[name] && !strings.Contains(token, "=") && i+1 < len(tokens) {
				i++
			}
			continue
		}
		path = append(path, token)
	}
	return path
}

// completionOptionValueNames returns option names that consume a following
// value.
func completionOptionValueNames(installedRoot string) map[string]bool {
	context := completionContext{Bundles: mustLoadBundlesForCompletion(installedRoot)}
	names := make(map[string]bool)
	for _, option := range completionOptions(context) {
		if option.RequiresValue {
			for _, name := range option.Names {
				names[name] = true
			}
		}
	}
	for _, subcommand := range pluginCompletionSubcommands() {
		for _, option := range pluginCompletionOptions(subcommand) {
			if option.RequiresValue {
				for _, name := range option.Names {
					names[name] = true
				}
			}
		}
	}
	for _, bundle := range context.Bundles {
		for _, command := range bundle.Commands.Commands {
			for _, parameter := range command.Parameters {
				names["--"+parameter.Flag] = true
			}
		}
	}
	return names
}

// completionOptions returns global, plugin-manager, and product-command options
// for context.
func completionOptions(context completionContext) []completionOption {
	options := make([]completionOption, 0, len(globalOptionsHelp)+8)
	for _, option := range globalOptionsHelp {
		if option.Long == "--version" && len(context.Path) > 0 {
			continue
		}
		if len(context.Path) > 0 && !globalOptionAllowed(context.Path, strings.TrimPrefix(option.Long, "--")) {
			continue
		}
		options = append(options, completionOption{
			Names:         globalCompletionOptionNames(option),
			RequiresValue: option.Value != "",
			Values:        globalCompletionOptionValues(option.Long),
		})
	}
	if context.PluginSubcommand != "" {
		options = append(options, pluginCompletionOptions(context.PluginSubcommand)...)
	}
	if len(context.Path) > 0 && (context.Path[0] == "update" || context.Path[0] == "upgrade") {
		options = append(options, upgradeCompletionOptions()...)
	}
	if len(context.Path) >= 2 && context.Path[0] == "doctor" && context.Path[1] == "network" {
		options = append(options, completionOption{
			Names:         []string{"--source"},
			RequiresValue: true,
			Values:        func(completionContext) []string { return []string{"auto", "gitee", "github"} },
		})
	}
	if context.CommandFound {
		for _, parameter := range context.Command.Parameters {
			values := parameter.AllowedValues
			options = append(options, completionOption{
				Names:         []string{"--" + parameter.Flag},
				RequiresValue: true,
				Values: func(completionContext) []string {
					return append([]string(nil), values...)
				},
			})
		}
	}
	return options
}

// globalCompletionOptionNames returns long, short, and alias names for a global
// option.
func globalCompletionOptionNames(option globalOptionHelp) []string {
	names := []string{option.Long}
	if option.Short != "" {
		names = append(names, option.Short)
	}
	names = append(names, option.Aliases...)
	return names
}

// globalCompletionOptionValues returns known value completions for a global
// option.
func globalCompletionOptionValues(name string) func(completionContext) []string {
	switch name {
	case "--output":
		return func(completionContext) []string { return []string{"json", "table"} }
	case "--table":
		return func(completionContext) []string { return []string{"bordered", "compact", "plain"} }
	case "--lang":
		return func(completionContext) []string { return []string{"en-GB", "en-US", "zh-CN"} }
	case "--cols":
		return func(context completionContext) []string { return tableColumnKeys(context, "") }
	case "--filter":
		return func(context completionContext) []string { return tableColumnKeys(context, "=") }
	case "--sort":
		return func(context completionContext) []string {
			keys := tableColumnKeys(context, "")
			values := make([]string, 0, len(keys)*2)
			for _, key := range keys {
				values = append(values, key, "-"+key)
			}
			return values
		}
	case "--wait":
		return func(context completionContext) []string {
			if !context.CommandFound {
				return nil
			}
			ids := make([]string, 0, len(context.Bundle.Waiters.Waiters))
			for id := range context.Bundle.Waiters.Waiters {
				ids = append(ids, id)
			}
			sortStrings(ids)
			return ids
		}
	default:
		return nil
	}
}

// upgradeCompletionOptions returns core self-upgrade command options.
func upgradeCompletionOptions() []completionOption {
	return []completionOption{
		{Names: []string{"--check"}},
		{Names: []string{"--source"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"auto", "gitee", "github"} }},
		{Names: []string{"--channel"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"alpha", "beta", "stable"} }},
	}
}

// pluginCompletionOptions returns options for a plugin-manager subcommand.
func pluginCompletionOptions(subcommand string) []completionOption {
	switch subcommand {
	case "install":
		return []completionOption{
			{Names: []string{"--all"}},
			{Names: []string{"--source"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"auto", "gitee", "github"} }},
			{Names: []string{"--channel"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"alpha", "beta", "stable"} }},
		}
	case "search":
		return []completionOption{
			{Names: []string{"--source"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"auto", "gitee", "github"} }},
			{Names: []string{"--channel"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"all", "alpha", "beta", "stable"} }},
		}
	case "list":
		return []completionOption{
			{Names: []string{"--available"}},
			{Names: []string{"--updates"}},
			{Names: []string{"--source"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"auto", "gitee", "github"} }},
			{Names: []string{"--channel"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"all", "alpha", "beta", "stable"} }},
		}
	case "remove":
		return []completionOption{
			{Names: []string{"--all"}},
		}
	case "reinstall", "update", "upgrade":
		return []completionOption{
			{Names: []string{"--all"}},
			{Names: []string{"--source"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"auto", "gitee", "github"} }},
			{Names: []string{"--channel"}, RequiresValue: true, Values: func(completionContext) []string { return []string{"alpha", "beta", "stable"} }},
		}
	default:
		return nil
	}
}

// tableColumnKeys returns stable table keys for the matched product command.
func tableColumnKeys(context completionContext, suffix string) []string {
	if !context.CommandFound {
		return nil
	}
	table, ok := context.Bundle.Tables.Tables[context.Command.Table]
	if !ok {
		return nil
	}
	values := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		values = append(values, column.Key+suffix)
	}
	sortStrings(values)
	return values
}

// findCompletionOption looks up an option by any accepted name.
func findCompletionOption(name string, context completionContext) (completionOption, bool) {
	name = completionOptionName(name)
	for _, option := range completionOptions(context) {
		for _, candidate := range option.Names {
			if candidate == name {
				return option, true
			}
		}
	}
	return completionOption{}, false
}

// usedCompletionOptions records options already present in the current command
// line.
func usedCompletionOptions(tokens []string) map[string]bool {
	used := make(map[string]bool)
	for _, token := range tokens {
		if strings.HasPrefix(token, "-") {
			used[completionOptionName(token)] = true
		}
	}
	return used
}

// completionOptionName strips an inline value from an option token.
func completionOptionName(token string) string {
	name, _, _ := strings.Cut(token, "=")
	return name
}

// coreCompletionCommands returns built-in command words.
func coreCompletionCommands() []string {
	return []string{"completion", "config", "doctor", "help", "plugin", "plugins", "update", "upgrade", "version"}
}

// completionShells returns shells supported by ctyun completion.
func completionShells() []string {
	return []string{"bash", "fish", "powershell", "zsh"}
}

// pluginCompletionSubcommands returns plugin-manager subcommands and aliases.
func pluginCompletionSubcommands() []string {
	seen := make(map[string]struct{})
	for _, command := range pluginSubcommandSummaries() {
		seen[command.Name] = struct{}{}
		for _, alias := range command.Aliases {
			seen[alias] = struct{}{}
		}
	}
	return sortedCompletionSet(seen)
}

// allCompletionWords returns every static word the completion system can emit.
func allCompletionWords(installedRoot string) []string {
	seen := completionSet(coreCompletionCommands()...)
	for _, word := range pluginCompletionSubcommands() {
		seen[word] = struct{}{}
	}
	for _, word := range configCompletionSubcommands() {
		seen[word] = struct{}{}
	}
	for _, word := range configProfileCompletionSubcommands() {
		seen[word] = struct{}{}
	}
	for _, word := range configSecretKeys() {
		seen[word] = struct{}{}
	}
	for _, word := range configCompletionOptionNames() {
		seen[word] = struct{}{}
	}
	seen["network"] = struct{}{}
	context := completionContext{Bundles: mustLoadBundlesForCompletion(installedRoot)}
	for _, option := range completionOptions(context) {
		for _, name := range option.Names {
			seen[name] = struct{}{}
		}
	}
	for _, subcommand := range pluginCompletionSubcommands() {
		for _, option := range pluginCompletionOptions(subcommand) {
			for _, name := range option.Names {
				seen[name] = struct{}{}
			}
		}
	}
	for _, bundle := range context.Bundles {
		for _, command := range bundle.Commands.Commands {
			for _, part := range command.Path {
				if !isPathPlaceholder(part) {
					seen[part] = struct{}{}
				}
			}
			for _, parameter := range command.Parameters {
				seen["--"+parameter.Flag] = struct{}{}
			}
		}
	}
	return sortedCompletionSet(seen)
}

// completionSet builds a deduplication set from values.
func completionSet(values ...string) map[string]struct{} {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	return seen
}

// sortedCompletionSet returns deterministic set contents.
func sortedCompletionSet(seen map[string]struct{}) []string {
	values := make([]string, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	sortStrings(values)
	return values
}

// filterByPrefix returns sorted unique values matching prefix.
func filterByPrefix(values []string, prefix string) []string {
	seen := make(map[string]struct{})
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			seen[value] = struct{}{}
		}
	}
	return sortedCompletionSet(seen)
}

// isPathPlaceholder reports whether a command path segment is an argument
// placeholder.
func isPathPlaceholder(part string) bool {
	return strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")
}

// mustLoadBundlesForCompletion loads bundles for completion and falls back to
// none on invalid metadata.
func mustLoadBundlesForCompletion(installedRoot string) []plugin.Bundle {
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return nil
	}
	return bundles
}
