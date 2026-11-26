/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// coreInvocationHelpPath removes valid command-owned inputs after a core leaf for flag-form help.
// Unresolved groups retain their tokens for the shared help path diagnostics.
func coreInvocationHelpPath(args []string) ([]string, error) {
	switch args[0] {
	case "completion":
		if err := validatePositionalArguments(args[1:], nil, 0, 1); err != nil {
			return nil, err
		}
		if len(args) == 2 && !slices.Contains(completionShells(), args[1]) {
			return nil, diagnostic.New("error.unsupported_shell", args[1])
		}
		return args[:1], nil
	case "update", "upgrade":
		_, err := parseUpgradeOptions(args[1:])
		return args[:1], err
	case "doctor":
		if len(args) >= 2 && args[1] == "network" {
			_, err := parseDoctorNetworkOptions(args[1:])
			return args[:2], err
		}
	case "plugin", "plugins":
		return subcommandInvocationHelpPath(args, 1, pluginSubcommandSummaries())
	case "config":
		if len(args) >= 2 && (args[1] == "profile" || args[1] == "profiles") {
			return subcommandInvocationHelpPath(args, 2, configProfileSubcommandSummaries())
		}
		return subcommandInvocationHelpPath(args, 1, configSubcommandSummaries())
	}
	return args, nil
}

// subcommandInvocationHelpPath verifies supplied tokens while allowing help to omit required inputs.
func subcommandInvocationHelpPath(args []string, index int, commands []subcommandHelp) ([]string, error) {
	if len(args) <= index+1 {
		return args, nil
	}
	for _, command := range commands {
		if !subcommandMatches(command, args[index]) {
			continue
		}
		tail := args[index+1:]
		var parsed parsedCommandTokens
		var err error
		if args[0] == "config" {
			parsed, err = parseCommandTokens(tail, command.tokenOptions())
		} else {
			parsed, err = parsePluginCommandTokens(command.Name, tail)
			if err == nil {
				err = validatePluginSourceOption(parsed.Options["source"])
			}
			if err == nil {
				err = validatePluginChannelOption(parsed.Options["channel"], command.Name == "list" || command.Name == "search")
			}
		}
		if err != nil {
			return nil, err
		}
		maximum := len(command.Arguments)
		if maximum > 0 && strings.HasSuffix(command.Arguments[maximum-1].Name, "...}") {
			maximum = -1
		}
		return args[:index+1], rejectUnexpectedPositionals(parsed.Positionals, maximum)
	}
	return args, nil
}
