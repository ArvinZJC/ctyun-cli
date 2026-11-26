/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"slices"
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
			"  completions=(${(f)\"$(ctyun __complete \"${words[@]:1:$((CURRENT - 1))}\")\"})",
			"  compadd -- $completions",
			"}",
			"_ctyun \"$@\"",
		)
	case "bash":
		return writeLines(stdout, `_ctyun_completion() {
  local -a arguments=()
  local remaining=${COMP_LINE-} word last i current fragment kept= candidate
  for ((i=0; i<=COMP_CWORD; i++)); do
    word=${COMP_WORDS[i]}
    last=$((${#arguments[@]} - 1))
    # Bash versions differ in whether they split inline options at word breaks.
    # Join only adjacent pieces of an option; whitespace still separates arguments.
    if (( last >= 0 )) && [[ -n $word && $remaining == "$word"* && ${arguments[last]} == -* && ( $word == '=' || ${arguments[last]} == *=* ) ]]; then
      arguments[last]+=$word
    elif (( i > 0 )); then
      arguments+=("$word")
    fi
    remaining=${remaining#*"$word"}
  done
  if (( ${#arguments[@]} )); then
    current=${arguments[${#arguments[@]}-1]}
    fragment=${2-$current}
    # Readline has already retained the part before its replacement fragment.
    if [[ $current == -*=* && $current == *"$fragment" ]]; then
      kept=${current%"$fragment"}
    fi
  fi
  COMPREPLY=()
  while IFS= read -r candidate; do
    COMPREPLY+=("${candidate#"$kept"}")
  done < <(ctyun __complete "${arguments[@]}")
}
complete -F _ctyun_completion ctyun`)
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
		return writeLines(stdout, `Register-ArgumentCompleter -Native -CommandName ctyun -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  $arguments = @()
  foreach ($element in @($commandAst.CommandElements | Select-Object -Skip 1)) {
    if ($element.Extent.EndOffset -ge $cursorPosition) { break }
    if ($element -is [System.Management.Automation.Language.StringConstantExpressionAst]) { $arguments += $element.Value } else { $arguments += $element.Extent.Text }
  }
  $arguments += $wordToComplete
  $passing = $ExecutionContext.SessionState.PSVariable.GetValue('PSNativeCommandArgumentPassing')
  $candidates = if ($null -eq $passing -or $passing -eq 'Legacy') {
    # Bypass the legacy binder, which drops empty strings and reparses embedded quotes.
    $start = New-Object System.Diagnostics.ProcessStartInfo
    $start.FileName = @(Get-Command ctyun -CommandType Application)[0].Path
    $start.WorkingDirectory = $ExecutionContext.SessionState.Path.CurrentFileSystemLocation.ProviderPath
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.StandardOutputEncoding = [System.Text.Encoding]::UTF8
    $quoted = foreach ($argument in $arguments) {
      # In a quoted native argument, double backslashes before a quote or the closing delimiter.
      $escaped = [regex]::Replace([string]$argument, '(\\*)"', '$1$1\"')
      '"' + [regex]::Replace($escaped, '(\\+)\z', '$1$1') + '"'
    }
    $start.Arguments = '__complete ' + ($quoted -join ' ')
    $process = [System.Diagnostics.Process]::Start($start)
    try {
      while ($null -ne ($line = $process.StandardOutput.ReadLine())) { $line }
      $process.WaitForExit()
    } finally { $process.Dispose() }
  } else { ctyun __complete @arguments }
  $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
  }
}`)
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
	if context.PluginSubcommand != "" {
		if _, err := subcommandInvocationHelpPath(context.Path, 1, pluginSubcommandSummaries()); err != nil {
			return nil
		}
	}
	if context.PendingOption != "" {
		option, ok := findCompletionOption(context.PendingOption, context)
		if !ok || option.Values == nil {
			return nil
		}
		return filterByPrefix(option.Values(context), prefix)
	}
	candidates := commandCompletions(context.Path, context)
	if candidates == nil {
		return nil
	}
	if values, ok := inlineOptionValueCompletions(prefix, context); ok {
		return values
	}
	if prefix == "" || strings.HasPrefix(prefix, "-") {
		candidates = append(candidates, optionCompletions(prefix, context)...)
	}
	return filterByPrefix(candidates, prefix)
}

// completionContext records command operands separately from consumed options and their values.
type completionContext struct {
	Bundles          []plugin.Bundle
	UsedOptions      map[string]bool
	PendingOption    string
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
	context := completionContext{Bundles: bundles, UsedOptions: make(map[string]bool)}
	path := context.consumeOptions(tokens, completionOptions(context))
	context.Path = path
	if len(path) >= 2 && (path[0] == "plugin" || path[0] == "plugins") {
		for _, command := range pluginSubcommandSummaries() {
			if subcommandMatches(command, path[1]) {
				context.PluginSubcommand = path[1]
				break
			}
		}
	}
	if len(path) == 0 {
		return context
	}
	if slices.Contains(coreCompletionCommands(), path[0]) {
		context.Path = context.consumeOptions(path, completionOptions(context))
	} else if bundle, command, _, rest, ok := matchProductCommand(bundles, path); ok {
		context.Bundle = bundle
		context.Command = command
		context.CommandFound = true
		if operands := context.consumeOptions(rest, completionOptions(context)); len(operands) != 0 {
			// The execution parser also recognizes build-specific switches omitted from completion.
			_, err := parseSuppliedCommandParameters(command, operands, "")
			context.CommandFound = err == nil
		}
	}
	return context
}

// inlineOptionValueCompletions completes values for --flag=value prefixes.
func inlineOptionValueCompletions(prefix string, context completionContext) ([]string, bool) {
	name, valuePrefix, ok := strings.Cut(prefix, "=")
	if !ok || !strings.HasPrefix(name, "--") {
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
	slices.Sort(values)
	return values, true
}

// optionCompletions returns unused options matching prefix.
func optionCompletions(prefix string, context completionContext) []string {
	candidates := make([]string, 0)
	for _, option := range completionOptions(context) {
		skip := false
		for _, name := range option.Names {
			if context.UsedOptions[name] {
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

// commandCompletions returns command words for a valid cursor position. An empty slice
// allows applicable options at a leaf; nil means the path is unknown or still needs a positional value.
func commandCompletions(path []string, context completionContext) []string {
	help := len(path) > 0 && path[0] == "help"
	if help {
		path = path[1:]
	}
	if len(path) == 0 {
		return topLevelCompletionCommands(context.Bundles)
	}
	switch path[0] {
	case "config":
		return configCommandCompletions(path, help)
	case "completion":
		if !help && len(path) == 1 {
			return completionShells()
		}
		if help && len(path) == 1 || !help && len(path) == 2 && slices.Contains(completionShells(), path[1]) {
			return []string{}
		}
	case "doctor":
		subcommands := []string{"local", "network"}
		if len(path) == 1 {
			return subcommands
		}
		if len(path) == 2 && slices.Contains(subcommands, path[1]) {
			return []string{}
		}
	case "plugin", "plugins":
		subcommands := pluginCompletionSubcommands()
		if len(path) == 1 {
			return subcommands
		}
		if slices.Contains(subcommands, path[1]) && (!help || len(path) == 2) {
			return []string{}
		}
	case "help", "update", "upgrade", "version":
		if len(path) == 1 {
			return []string{}
		}
	default:
		words := nextPluginPathCompletions(path, context.Bundles)
		if len(words) != 0 || context.CommandFound {
			return words
		}
		if help {
			if _, _, ok := matchPluginCommandForHelp(path, context.Bundles); ok {
				return []string{}
			}
		}
	}
	return nil
}

// configCommandCompletions resolves config groups, declared operands and supported value choices.
func configCommandCompletions(path []string, help bool) []string {
	commands := configSubcommandSummaries()
	index := 1
	if len(path) > 1 && (path[1] == "profile" || path[1] == "profiles") {
		commands = configProfileSubcommandSummaries()
		index++
	}
	if len(path) == index {
		return configCompletionNames(commands)
	}
	for _, command := range commands {
		if !subcommandMatches(command, path[index]) {
			continue
		}
		arguments := len(path) - index - 1
		if arguments > len(command.Arguments) || help && arguments > 0 {
			return nil
		}
		if !help && command.Name == "explain" && arguments == 0 {
			keys := coreconfig.SettingKeys()
			slices.Sort(keys)
			return keys
		}
		if !help && command.Name == "set-secret" && arguments == 1 {
			return configSecretKeys()
		}
		return []string{}
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
func configCompletionNames(commands []subcommandHelp) []string {
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

// addConfigCompletionOptionNames adds option names from config commands.
func addConfigCompletionOptionNames(seen map[string]struct{}, commands []subcommandHelp) {
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
	slices.Sort(names)
	return names
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

// consumeOptions returns operands in order, recording only options recognized at this parsing stage.
// As in execution, global options are consumed first and product options only after the matched command path.
func (context *completionContext) consumeOptions(tokens []string, options []completionOption) []string {
	byName := make(map[string]completionOption)
	for _, option := range options {
		for _, name := range option.Names {
			byName[name] = option
		}
	}
	var operands []string
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		name, _, inline := strings.Cut(token, "=")
		option, ok := byName[name]
		if !ok || inline && (!option.RequiresValue || !strings.HasPrefix(name, "--")) {
			operands = append(operands, token)
			continue
		}
		context.UsedOptions[name] = true
		if option.RequiresValue && !inline {
			index++
			if index == len(tokens) && context.PendingOption == "" {
				context.PendingOption = name
			}
		}
	}
	return operands
}

// completionOptions returns global, plugin-manager, and product-command options
// for context.
func completionOptions(context completionContext) []completionOption {
	options := make([]completionOption, 0, len(globalOptionsHelp)+8)
	for _, option := range globalOptionsHelp {
		if context.CommandFound && !productGlobalOptionAllowed(context.Bundle.APIs.Operations[context.Command.Operation], strings.TrimPrefix(option.Long, "--")) {
			continue
		}
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
	if len(context.Path) >= 3 && context.Path[0] == "config" && (context.Path[1] == "profile" || context.Path[1] == "profiles") {
		for _, command := range configProfileSubcommandSummaries() {
			if subcommandMatches(command, context.Path[2]) {
				for _, option := range command.Options {
					options = append(options, completionOption{Names: []string{option.Name}})
				}
				break
			}
		}
	}
	if context.CommandFound {
		for _, option := range productTransferOptions(context.Command) {
			options = append(options, completionOption{Names: []string{"--" + option.Name}, RequiresValue: option.TakesValue})
		}
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
		return func(context completionContext) []string {
			if context.CommandFound {
				return plugin.OutputFormats(context.Bundle.APIs.Operations[context.Command.Operation])
			}
			return []string{"json", "table"}
		}
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
			return plugin.CommandWaiters(context.Bundle, context.Command)
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
	slices.Sort(values)
	return values
}

// findCompletionOption looks up an option by any accepted name.
func findCompletionOption(name string, context completionContext) (completionOption, bool) {
	name, _, _ = strings.Cut(name, "=")
	for _, option := range completionOptions(context) {
		if slices.Contains(option.Names, name) {
			return option, true
		}
	}
	return completionOption{}, false
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
	slices.Sort(values)
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
