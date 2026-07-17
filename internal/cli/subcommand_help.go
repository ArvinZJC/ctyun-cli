/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"slices"
	"strings"
)

// subcommandHelp describes one built-in command subcommand.
type subcommandHelp struct {
	Name           string
	Aliases        []string
	DescriptionKey string
	Usage          []string
	Arguments      []commandArgumentSummary
	Options        []pluginOptionSummary
}

// writeSubcommandHelpPage writes the shared description, usage, argument, and
// command-option sections for one built-in subcommand.
func writeSubcommandHelpPage(writer *outputWriter, command subcommandHelp, language string) {
	writer.Line(helpPageText(command.DescriptionKey, language))
	writer.Format("\n%s:\n", helpText("usage.heading", language))
	writeUsageLines(writer, command.Usage)
	if len(command.Arguments) > 0 {
		writer.Format("\n%s:\n", helpText("arguments.heading", language))
		writeArgumentHelpRows(writer, command.Arguments, language)
	}
	if len(command.Options) > 0 {
		writer.Format("\n%s:\n", helpText("command.heading", language))
		writeAlignedHelpRows(writer, pluginOptionHelpRows(command.Options, language), "  ")
	}
}

// subcommandHelpRows converts built-in subcommands to aligned help rows.
func subcommandHelpRows(commands []subcommandHelp, language string) []helpRow {
	rows := make([]helpRow, 0, len(commands))
	for _, command := range commands {
		rows = append(rows, helpRow{
			Name:        subcommandNames(command),
			Description: helpText(command.DescriptionKey, language),
		})
	}
	sortHelpRows(rows)
	return rows
}

// subcommandNames joins a built-in subcommand and aliases for display.
func subcommandNames(command subcommandHelp) string {
	names := append([]string{command.Name}, command.Aliases...)
	return strings.Join(names, "|")
}

// subcommandMatches reports whether name selects a subcommand or one of its
// aliases.
func subcommandMatches(command subcommandHelp, name string) bool {
	return command.Name == name || slices.Contains(command.Aliases, name)
}
