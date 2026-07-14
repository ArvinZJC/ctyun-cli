/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
)

// pluginSubcommandSummaries returns public plugin-manager help definitions.
func pluginSubcommandSummaries() []subcommandHelp {
	return []subcommandHelp{
		{
			Name:           "install",
			DescriptionKey: "plugin.install.description",
			Usage: []string{
				globalUsage("plugin install {name...} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugin install --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins install {name...} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins install --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
			},
			Arguments: []commandArgumentSummary{{Name: "{name...}", Key: "argument.plugin_names"}},
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.install"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "list",
			DescriptionKey: "plugin.list.description",
			Usage: []string{
				globalUsage("plugin list [--available|--updates] [--source <auto|github|gitee>] [--channel <stable|beta|alpha|all>]"),
				globalUsage("plugins list [--available|--updates] [--source <auto|github|gitee>] [--channel <stable|beta|alpha|all>]"),
			},
			Options: []pluginOptionSummary{
				{Name: "--available", Key: "plugin.option.available"},
				{Name: "--updates", Key: "plugin.option.updates"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha|all>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "remove",
			DescriptionKey: "plugin.remove.description",
			Usage: []string{
				globalUsage("plugin remove {name...}"),
				globalUsage("plugin remove --all"),
				globalUsage("plugins remove {name...}"),
				globalUsage("plugins remove --all"),
			},
			Arguments: []commandArgumentSummary{{Name: "{name...}", Key: "argument.plugin_names"}},
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.remove"},
			},
		},
		{
			Name:           "reinstall",
			DescriptionKey: "plugin.reinstall.description",
			Usage: []string{
				globalUsage("plugin reinstall {name...} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugin reinstall --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins reinstall {name...} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins reinstall --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
			},
			Arguments: []commandArgumentSummary{{Name: "{name...}", Key: "argument.plugin_names"}},
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.reinstall"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "search",
			DescriptionKey: "plugin.search.description",
			Usage: []string{
				globalUsage("plugin search {query} [--source <auto|github|gitee>] [--channel <stable|beta|alpha|all>]"),
				globalUsage("plugins search {query} [--source <auto|github|gitee>] [--channel <stable|beta|alpha|all>]"),
			},
			Arguments: []commandArgumentSummary{{Name: "{query}", Key: "argument.plugin_query"}},
			Options: []pluginOptionSummary{
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha|all>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "update",
			Aliases:        []string{"upgrade"},
			DescriptionKey: "plugin.update.description",
			Usage: []string{
				globalUsage("plugin update {name} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugin update --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugin upgrade {name} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugin upgrade --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins update {name} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins update --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins upgrade {name} [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
				globalUsage("plugins upgrade --all [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]"),
			},
			Arguments: []commandArgumentSummary{{Name: "{name}", Key: "argument.plugin_name"}},
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.update"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
	}
}

// printPluginHelp prints plugin-manager overview or subcommand help.
func printPluginHelp(stdout io.Writer, args []string, language string) (bool, error) {
	writer := newOutputWriter(stdout)
	if len(args) == 1 {
		writer.Line(helpPageText("plugin.description", language))
		writer.Format("\n%s:\n", helpText("usage.heading", language))
		writeUsageLines(writer, pluginOverviewUsageLines())
		writer.Format("\n%s:\n", helpText("subcommands.heading", language))
		writeAlignedHelpRows(writer, subcommandHelpRows(pluginSubcommandSummaries(), language), "  ")
		return true, writer.Err()
	}
	for _, command := range pluginSubcommandSummaries() {
		if subcommandMatches(command, args[1]) {
			if err := validatePositionalArguments(args[2:], nil, 0, 0); err != nil {
				return true, err
			}
			writeSubcommandHelpPage(writer, command, language)
			return true, writer.Err()
		}
	}
	return false, nil
}

// pluginOverviewUsageLines returns command-group usage forms for the plugin
// help overview.
func pluginOverviewUsageLines() []string {
	return []string{
		"ctyun [global options] plugin <subcommand>",
		"ctyun [global options] plugins <subcommand>",
		"ctyun help plugin <subcommand>",
		"ctyun help plugins <subcommand>",
	}
}
