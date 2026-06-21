/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// helpCatalog contains localized core help, help-only hints, and plugin-manager
// help labels.
var helpCatalog = map[string]map[string]string{
	"title": {
		"en-US": "ctyun - plugin-based CTyun CLI",
		"en-GB": "ctyun - plugin-based CTyun CLI",
		"zh-CN": "ctyun - 非官方插件化天翼云 CLI",
	},
	"description.heading":                   {"en-US": "Description", "en-GB": "Description", "zh-CN": "描述"},
	"description.line1":                     {"en-US": "ctyun is an unofficial, plugin-based CLI for CTyun.", "en-GB": "ctyun is an unofficial, plugin-based CLI for CTyun.", "zh-CN": "ctyun 是非官方插件化天翼云 CLI。"},
	"description.line2":                     {"en-US": "It prioritizes terminal-friendly cloud workflows because CTyun has no official CLI.", "en-GB": "It prioritises terminal-friendly cloud workflows because CTyun has no official CLI.", "zh-CN": "它体验优先，面向终端里的资源查询和管理；目前天翼云没有官方 CLI。"},
	"usage.heading":                         {"en-US": "Usage", "en-GB": "Usage", "zh-CN": "用法"},
	"usage.command":                         {"en-US": "command", "en-GB": "command", "zh-CN": "命令"},
	"usage.global":                          {"en-US": "global options", "en-GB": "global options", "zh-CN": "全局选项"},
	"usage.command_opts":                    {"en-US": "command options", "en-GB": "command options", "zh-CN": "命令选项"},
	"core.heading":                          {"en-US": "Core Commands", "en-GB": "Core Commands", "zh-CN": "核心命令"},
	"plugins.heading":                       {"en-US": "Plugin Commands", "en-GB": "Plugin Commands", "zh-CN": "插件命令"},
	"commands.heading":                      {"en-US": "Commands", "en-GB": "Commands", "zh-CN": "命令"},
	"subcommands.heading":                   {"en-US": "Subcommands", "en-GB": "Subcommands", "zh-CN": "子命令"},
	"global.heading":                        {"en-US": "Global Options", "en-GB": "Global Options", "zh-CN": "全局选项"},
	"command.heading":                       {"en-US": "Command Options", "en-GB": "Command Options", "zh-CN": "命令选项"},
	"columns.heading":                       {"en-US": "Columns", "en-GB": "Columns", "zh-CN": "列"},
	"examples.heading":                      {"en-US": "Examples", "en-GB": "Examples", "zh-CN": "示例"},
	"docs.heading":                          {"en-US": "Docs", "en-GB": "Docs", "zh-CN": "文档"},
	"product.label":                         {"en-US": "Product", "en-GB": "Product", "zh-CN": "产品"},
	"description.label":                     {"en-US": "Description", "en-GB": "Description", "zh-CN": "描述"},
	"required":                              {"en-US": "required", "en-GB": "required", "zh-CN": "必填"},
	"validation.allowed":                    {"en-US": "one of %s", "en-GB": "one of %s", "zh-CN": "可选值 %s"},
	"validation.pattern":                    {"en-US": "matches %s", "en-GB": "matches %s", "zh-CN": "匹配 %s"},
	"core.config":                           {"en-US": "Show and update CLI configuration", "en-GB": "Show and update CLI configuration", "zh-CN": "显示并更新 CLI 配置"},
	"core.completion":                       {"en-US": "Print a shell completion script", "en-GB": "Print a shell completion script", "zh-CN": "输出 shell 补全脚本"},
	"core.doctor":                           {"en-US": "Inspect local network and registry configuration", "en-GB": "Inspect local network and registry configuration", "zh-CN": "检查本地网络和插件源配置"},
	"core.help":                             {"en-US": "Show CLI or command help", "en-GB": "Show CLI or command help", "zh-CN": "显示 CLI 或命令帮助"},
	"core.plugin":                           {"en-US": "Manage plugins", "en-GB": "Manage plugins", "zh-CN": "管理插件"},
	"core.upgrade":                          {"en-US": "Update or upgrade the core ctyun binary", "en-GB": "Update or upgrade the core ctyun binary", "zh-CN": "更新或升级核心 ctyun 二进制文件"},
	"core.upgrade.plugins":                  {"en-US": "For plugin updates, run ctyun plugin|plugins update|upgrade", "en-GB": "For plugin updates, run ctyun plugin|plugins update|upgrade", "zh-CN": "如需更新插件，请运行 ctyun plugin|plugins update|upgrade"},
	"core.upgrade.option.check":             {"en-US": "Check signed release metadata without changing the binary", "en-GB": "Check signed release metadata without changing the binary", "zh-CN": "检查签名发布元数据，不修改二进制文件"},
	"core.upgrade.option.source":            {"en-US": "Use auto, github, or gitee", "en-GB": "Use auto, github, or gitee", "zh-CN": "使用 auto、github 或 gitee"},
	"core.upgrade.option.channel":           {"en-US": "Select the core release channel", "en-GB": "Select the core release channel", "zh-CN": "选择核心发布通道"},
	"core.version":                          {"en-US": "Print the CLI version", "en-GB": "Print the CLI version", "zh-CN": "输出 CLI 版本"},
	"config.description":                    {"en-US": "Show, update, and reset ctyun config files", "en-GB": "Show, update, and reset ctyun config files", "zh-CN": "显示、更新和重置 ctyun 配置文件"},
	"config.path.description":               {"en-US": "Print the resolved config file path", "en-GB": "Print the resolved config file path", "zh-CN": "输出解析后的配置文件路径"},
	"config.show.description":               {"en-US": "Show config JSON with masked credentials", "en-GB": "Show config JSON with masked credentials", "zh-CN": "显示配置 JSON，并对凭证做掩码处理"},
	"config.set.description":                {"en-US": "Set a global config key, or a profile key with --profile", "en-GB": "Set a global config key, or a profile key with --profile", "zh-CN": "设置全局配置键，或配合 --profile 设置配置档案键"},
	"config.unset.description":              {"en-US": "Unset a global config key, or a profile key with --profile", "en-GB": "Unset a global config key, or a profile key with --profile", "zh-CN": "取消全局配置键，或配合 --profile 取消配置档案键"},
	"config.reset.description":              {"en-US": "Back up and remove the current config file", "en-GB": "Back up and remove the current config file", "zh-CN": "备份并删除当前配置文件"},
	"config.option.from_stdin":              {"en-US": "Read the credential value from stdin", "en-GB": "Read the credential value from stdin", "zh-CN": "从标准输入读取凭证值"},
	"config.profile.description":            {"en-US": "Manage named config profiles", "en-GB": "Manage named config profiles", "zh-CN": "管理命名配置档案"},
	"config.profile.list.description":       {"en-US": "List configured profiles", "en-GB": "List configured profiles", "zh-CN": "列出已有配置档案"},
	"config.profile.use.description":        {"en-US": "Select the active profile", "en-GB": "Select the active profile", "zh-CN": "选择当前配置档案"},
	"config.profile.set.description":        {"en-US": "Set a profile config key", "en-GB": "Set a profile config key", "zh-CN": "设置配置档案键"},
	"config.profile.unset.description":      {"en-US": "Unset a profile config key", "en-GB": "Unset a profile config key", "zh-CN": "取消配置档案键"},
	"config.profile.set_secret.description": {"en-US": "Set a profile AK/SK value from stdin", "en-GB": "Set a profile AK/SK value from stdin", "zh-CN": "从标准输入设置配置档案 AK/SK"},
	"config.profile.reset.description":      {"en-US": "Remove one profile from the config file", "en-GB": "Remove one profile from the config file", "zh-CN": "从配置文件删除一个配置档案"},
	"doctor.description":                    {"en-US": "Inspect local environment details that affect ctyun connectivity", "en-GB": "Inspect local environment details that affect ctyun connectivity", "zh-CN": "检查影响 ctyun 连接的本地环境信息"},
	"doctor.network.description":            {"en-US": "Inspect local network and registry configuration", "en-GB": "Inspect local network and registry configuration", "zh-CN": "检查本地网络和插件源配置"},
	"plugin.description":                    {"en-US": "Manage plugin bundles and discover metadata-defined product commands", "en-GB": "Manage plugin bundles and discover metadata-defined product commands", "zh-CN": "管理插件包，并发现由元数据定义的产品命令"},
	"plugin.hint.list":                      {"en-US": "List installed plugins", "en-GB": "List installed plugins", "zh-CN": "列出已安装插件"},
	"plugin.hint.help":                      {"en-US": "Show commands provided by a plugin", "en-GB": "Show commands provided by a plugin", "zh-CN": "显示某个插件提供的命令"},
	"plugin.install.description":            {"en-US": "Install a plugin from a hosted source", "en-GB": "Install a plugin from a hosted source", "zh-CN": "从托管源安装插件"},
	"plugin.list.description":               {"en-US": "List installed plugins", "en-GB": "List installed plugins", "zh-CN": "列出已安装插件"},
	"plugin.remove.description":             {"en-US": "Remove an installed plugin", "en-GB": "Remove an installed plugin", "zh-CN": "删除已安装插件"},
	"plugin.search.description":             {"en-US": "Search hosted plugin metadata", "en-GB": "Search hosted plugin metadata", "zh-CN": "搜索托管插件元数据"},
	"plugin.update.description":             {"en-US": "Update or upgrade one or all installed plugins", "en-GB": "Update or upgrade one or all installed plugins", "zh-CN": "更新或升级一个或全部已安装插件"},
	"plugin.option.source":                  {"en-US": "Use auto, github, or gitee", "en-GB": "Use auto, github, or gitee", "zh-CN": "使用 auto、github 或 gitee"},
	"plugin.option.channel":                 {"en-US": "Select the registry channel", "en-GB": "Select the registry channel", "zh-CN": "选择插件源通道"},
	"plugin.option.updates":                 {"en-US": "Check installed plugins against hosted metadata", "en-GB": "Check installed plugins against hosted metadata", "zh-CN": "根据托管元数据检查已安装插件更新"},
	"plugin.option.all":                     {"en-US": "Update or upgrade every installed plugin with an available update", "en-GB": "Update or upgrade every installed plugin with an available update", "zh-CN": "更新或升级所有有可用更新的插件"},
	"plugin.column.name":                    {"en-US": "Name", "en-GB": "Name", "zh-CN": "名称"},
	"plugin.column.plugin":                  {"en-US": "Plugin", "en-GB": "Plugin", "zh-CN": "插件"},
	"plugin.column.version":                 {"en-US": "Version", "en-GB": "Version", "zh-CN": "版本"},
	"plugin.column.product":                 {"en-US": "Product", "en-GB": "Product", "zh-CN": "产品"},
	"plugin.column.channel":                 {"en-US": "Channel", "en-GB": "Channel", "zh-CN": "通道"},
	"plugin.column.quality":                 {"en-US": "Quality", "en-GB": "Quality", "zh-CN": "质量"},
	"plugin.column.commands":                {"en-US": "Commands", "en-GB": "Commands", "zh-CN": "命令数"},
	"plugin.column.operations":              {"en-US": "Operations", "en-GB": "Operations", "zh-CN": "接口数"},
	"option.output":                         {"en-US": "Render output as a table or raw JSON", "en-GB": "Render output as a table or raw JSON", "zh-CN": "以表格或原始 JSON 输出"},
	"option.cols":                           {"en-US": "Select output columns by stable column key", "en-GB": "Select output columns by stable column key", "zh-CN": "按稳定列键选择输出列"},
	"option.no_header":                      {"en-US": "Hide the table header", "en-GB": "Hide the table header", "zh-CN": "隐藏表头"},
	"option.filter":                         {"en-US": "Filter table rows by stable column key", "en-GB": "Filter table rows by stable column key", "zh-CN": "按稳定列键过滤表格行"},
	"option.sort":                           {"en-US": "Sort table rows by stable column key", "en-GB": "Sort table rows by stable column key", "zh-CN": "按稳定列键排序表格行"},
	"option.lang":                           {"en-US": "Choose help and output language", "en-GB": "Choose help and output language", "zh-CN": "选择帮助和输出语言"},
	"option.yes":                            {"en-US": "Confirm dangerous operations without prompting", "en-GB": "Confirm dangerous operations without prompting", "zh-CN": "无需提示直接确认危险操作"},
	"option.wait":                           {"en-US": "Evaluate a command waiter after the request", "en-GB": "Evaluate a command waiter after the request", "zh-CN": "请求后执行命令等待器"},
	"option.table":                          {"en-US": "Choose table style", "en-GB": "Choose table style", "zh-CN": "选择表格样式"},
	"option.timeout":                        {"en-US": "Set the per-request HTTP timeout", "en-GB": "Set the per-request HTTP timeout", "zh-CN": "设置单次 HTTP 请求超时"},
	"option.config":                         {"en-US": "Read profile configuration from a file", "en-GB": "Read profile configuration from a file", "zh-CN": "从文件读取配置档案"},
	"option.profile":                        {"en-US": "Select a named profile from the config file", "en-GB": "Select a named profile from the config file", "zh-CN": "从配置文件选择指定档案"},
	"option.debug":                          {"en-US": "Print HTTP request diagnostics to stderr", "en-GB": "Print HTTP request diagnostics to stderr", "zh-CN": "向 stderr 输出 HTTP 请求诊断信息"},
	"option.help":                           {"en-US": "Show help for the command", "en-GB": "Show help for the command", "zh-CN": "显示命令帮助"},
}

// runHelp routes help requests to core, plugin manager, or product-command help.
func runHelp(stdout io.Writer, args []string, installedRoot, language string) error {
	if len(args) == 0 {
		return printMainHelp(stdout, installedRoot, language)
	}
	if printCoreHelp(stdout, args, language) {
		return nil
	}
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return err
	}
	bundle, command, ok := matchPluginCommandForHelp(args, bundles)
	if !ok {
		bundle, commands, groupOK := matchPluginCommandGroupForHelp(args, bundles)
		if !groupOK {
			return fmt.Errorf("unknown command %q", strings.Join(args, " "))
		}
		printPluginCommandIndex(stdout, bundle, commands, language)
		return nil
	}
	if description := localizedPluginText(bundle, language, "command."+command.ID+".description", ""); description != "" {
		fmt.Fprintf(stdout, "%s\n", helpPageDescription(description, language))
	}
	if productName := localizedPluginText(bundle, language, "name", ""); productName != "" {
		if description := localizedPluginText(bundle, language, "command."+command.ID+".description", ""); description != "" {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "%s: %s\n", helpText("product.label", language), productName)
	}
	fmt.Fprintf(stdout, "\n%s:\n", helpText("usage.heading", language))
	fmt.Fprintf(stdout, "  ctyun [%s] %s [%s]\n", helpText("usage.global", language), strings.Join(command.Path, " "), helpText("usage.command_opts", language))
	if len(command.Parameters) > 0 {
		fmt.Fprintf(stdout, "\n%s:\n", helpText("command.heading", language))
		for _, parameter := range command.Parameters {
			required := ""
			if parameter.Required {
				required = " (" + helpText("required", language) + ")"
			}
			description := localizedPluginText(bundle, language, "parameter."+command.ID+"."+parameter.Name+".description", parameter.Description)
			if description != "" {
				description = "  " + description
			}
			validation := parameterValidationHint(parameter, language)
			fmt.Fprintf(stdout, "  --%s <value>%s%s%s\n", parameter.Flag, required, description, validation)
		}
	}
	printGlobalOptions(stdout, language)
	if table, ok := bundle.Tables.Tables[command.Table]; ok && len(table.Columns) > 0 {
		keys := make([]string, 0, len(table.Columns))
		for _, column := range table.Columns {
			keys = append(keys, column.Key)
		}
		fmt.Fprintf(stdout, "\n%s:\n  %s\n", helpText("columns.heading", language), strings.Join(keys, ","))
	}
	examples := visibleExamples(command.Examples)
	if len(examples) > 0 {
		fmt.Fprintf(stdout, "\n%s:\n", helpText("examples.heading", language))
		for _, example := range examples {
			fmt.Fprintf(stdout, "  %s\n", example)
		}
	}
	if command.DocsURL != "" {
		fmt.Fprintf(stdout, "\n%s:\n  %s\n", helpText("docs.heading", language), command.DocsURL)
	}
	return nil
}

// matchPluginCommandForHelp finds an exact plugin command for help output.
func matchPluginCommandForHelp(args []string, bundles []plugin.Bundle) (plugin.Bundle, plugin.Command, bool) {
	for _, bundle := range bundles {
		if command, ok := plugin.FindCommand(bundle, args); ok {
			return bundle, command, true
		}
	}
	return plugin.Bundle{}, plugin.Command{}, false
}

// matchPluginCommandGroupForHelp finds product commands below a shared prefix.
func matchPluginCommandGroupForHelp(args []string, bundles []plugin.Bundle) (plugin.Bundle, []plugin.Command, bool) {
	for _, bundle := range bundles {
		commands := make([]plugin.Command, 0)
		for _, command := range bundle.Commands.Commands {
			if pathHasPrefix(command.Path, args) {
				commands = append(commands, command)
			}
		}
		if len(commands) > 0 {
			return bundle, commands, true
		}
	}
	return plugin.Bundle{}, nil, false
}

// pathHasPrefix reports whether prefix identifies a non-leaf command group.
func pathHasPrefix(path, prefix []string) bool {
	if len(prefix) == 0 || len(prefix) >= len(path) {
		return false
	}
	for i, part := range prefix {
		if path[i] != part {
			return false
		}
	}
	return true
}

// printMainHelp prints top-level CLI usage and command groups.
func printMainHelp(stdout io.Writer, installedRoot, language string) error {
	fmt.Fprintln(stdout, helpText("description.line1", language))
	fmt.Fprintln(stdout, helpText("description.line2", language))
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%s:\n", helpText("usage.heading", language))
	fmt.Fprintf(stdout, "  ctyun [%s] <%s> [%s]\n", helpText("usage.global", language), helpText("usage.command", language), helpText("usage.command_opts", language))
	fmt.Fprintf(stdout, "  ctyun help <%s>\n", helpText("usage.command", language))
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%s:\n", helpText("core.heading", language))
	coreCommands := coreCommandSummaries(language)
	maxNameWidth := 0
	for _, command := range coreCommands {
		if len(command.Name) > maxNameWidth {
			maxNameWidth = len(command.Name)
		}
	}
	for _, command := range coreCommands {
		fmt.Fprintf(stdout, "  %-*s  %s\n", maxNameWidth, command.Name, command.Description)
	}
	printPluginCommandHints(stdout, language)
	printGlobalOptions(stdout, language)
	return nil
}

// commandSummary is one row in the core command summary.
type commandSummary struct {
	Name        string
	Description string
}

// pluginOptionSummary is one plugin-manager option row.
type pluginOptionSummary struct {
	Name string
	Key  string
}

// pluginSubcommandHelp describes one plugin-manager subcommand.
type pluginSubcommandHelp struct {
	Name           string
	Aliases        []string
	DescriptionKey string
	Usage          string
	Options        []pluginOptionSummary
}

// coreCommandSummaries returns localized summaries for built-in commands.
func coreCommandSummaries(language string) []commandSummary {
	return []commandSummary{
		{Name: "config", Description: helpText("core.config", language)},
		{Name: "completion", Description: helpText("core.completion", language)},
		{Name: "doctor", Description: helpText("core.doctor", language)},
		{Name: "help", Description: helpText("core.help", language)},
		{Name: "plugin|plugins", Description: helpText("core.plugin", language)},
		{Name: "update|upgrade", Description: helpText("core.upgrade", language) + " " + helpText("core.upgrade.plugins", language)},
		{Name: "version", Description: helpText("core.version", language)},
	}
}

// pluginSubcommandSummaries returns public plugin-manager help definitions.
func pluginSubcommandSummaries() []pluginSubcommandHelp {
	return []pluginSubcommandHelp{
		{
			Name:           "install",
			DescriptionKey: "plugin.install.description",
			Usage:          "ctyun plugin install <name> [--source auto|github|gitee] [--channel name]",
			Options: []pluginOptionSummary{
				{Name: "--source name", Key: "plugin.option.source"},
				{Name: "--channel name", Key: "plugin.option.channel"},
			},
		},
		{
			Name:           "list",
			DescriptionKey: "plugin.list.description",
			Usage:          "ctyun plugin list [--updates] [--source auto|github|gitee]",
			Options: []pluginOptionSummary{
				{Name: "--updates", Key: "plugin.option.updates"},
				{Name: "--source name", Key: "plugin.option.source"},
			},
		},
		{
			Name:           "remove",
			DescriptionKey: "plugin.remove.description",
			Usage:          "ctyun plugin remove <name>",
		},
		{
			Name:           "search",
			DescriptionKey: "plugin.search.description",
			Usage:          "ctyun plugin search <query> [--source auto|github|gitee] [--channel name]",
			Options: []pluginOptionSummary{
				{Name: "--source name", Key: "plugin.option.source"},
				{Name: "--channel name", Key: "plugin.option.channel"},
			},
		},
		{
			Name:           "update",
			Aliases:        []string{"upgrade"},
			DescriptionKey: "plugin.update.description",
			Usage:          "ctyun plugin update <name|--all> [--source auto|github|gitee]\n  ctyun plugin upgrade <name|--all> [--source auto|github|gitee]\n  ctyun plugins update <name|--all> [--source auto|github|gitee]\n  ctyun plugins upgrade <name|--all> [--source auto|github|gitee]",
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all"},
				{Name: "--source name", Key: "plugin.option.source"},
			},
		},
	}
}

// printCoreHelp prints help for a built-in command when args match one.
func printCoreHelp(stdout io.Writer, args []string, language string) bool {
	switch args[0] {
	case "config":
		if !printConfigHelp(stdout, args, language) {
			return false
		}
	case "completion":
		fmt.Fprintln(stdout, helpPageText("core.completion", language))
		fmt.Fprintf(stdout, "\n%s:\n  ctyun completion <bash|zsh|fish|powershell>\n", helpText("usage.heading", language))
	case "doctor":
		if !printDoctorHelp(stdout, args, language) {
			return false
		}
	case "help":
		fmt.Fprintln(stdout, helpPageText("core.help", language))
		fmt.Fprintf(stdout, "\n%s:\n  ctyun help [command]\n", helpText("usage.heading", language))
	case "plugin", "plugins":
		if !printPluginHelp(stdout, args, language) {
			return false
		}
	case "upgrade", "update":
		fmt.Fprintln(stdout, helpPageText("core.upgrade", language))
		fmt.Fprintln(stdout, helpPageText("core.upgrade.plugins", language))
		fmt.Fprintf(stdout, "\n%s:\n", helpText("usage.heading", language))
		fmt.Fprintln(stdout, "  ctyun update [--check] [--source auto|github|gitee] [--channel name]")
		fmt.Fprintln(stdout, "  ctyun upgrade [--check] [--source auto|github|gitee] [--channel name]")
		fmt.Fprintf(stdout, "\n%s:\n", helpText("command.heading", language))
		fmt.Fprintf(stdout, "  %-16s  %s\n", "--check", helpText("core.upgrade.option.check", language))
		fmt.Fprintf(stdout, "  %-16s  %s\n", "--source value", helpText("core.upgrade.option.source", language))
		fmt.Fprintf(stdout, "  %-16s  %s\n", "--channel name", helpText("core.upgrade.option.channel", language))
	case "version":
		fmt.Fprintln(stdout, helpPageText("core.version", language))
		fmt.Fprintf(stdout, "\n%s:\n  ctyun version\n", helpText("usage.heading", language))
	default:
		return false
	}
	printGlobalOptions(stdout, language)
	return true
}

// printDoctorHelp prints help for doctor and doctor subcommands.
func printDoctorHelp(stdout io.Writer, args []string, language string) bool {
	if len(args) == 1 {
		fmt.Fprintln(stdout, helpPageText("doctor.description", language))
		fmt.Fprintf(stdout, "\n%s:\n", helpText("usage.heading", language))
		fmt.Fprintln(stdout, "  ctyun doctor <subcommand>")
		fmt.Fprintln(stdout, "  ctyun help doctor <subcommand>")
		fmt.Fprintf(stdout, "\n%s:\n", helpText("subcommands.heading", language))
		fmt.Fprintf(stdout, "  %-8s  %s\n", "network", helpText("doctor.network.description", language))
		return true
	}
	if len(args) == 2 && args[1] == "network" {
		fmt.Fprintln(stdout, helpPageText("doctor.network.description", language))
		fmt.Fprintf(stdout, "\n%s:\n  ctyun doctor network\n", helpText("usage.heading", language))
		return true
	}
	return false
}

// printPluginCommandHints prints discovery hints for product commands.
func printPluginCommandHints(stdout io.Writer, language string) {
	fmt.Fprintf(stdout, "\n%s:\n", helpText("plugins.heading", language))
	fmt.Fprintf(stdout, "  %-20s %s\n", "ctyun plugin list", helpText("plugin.hint.list", language))
	fmt.Fprintf(stdout, "  %-20s %s\n", "ctyun plugins list", helpText("plugin.hint.list", language))
	fmt.Fprintf(stdout, "  %-20s %s\n", "ctyun help <plugin>", helpText("plugin.hint.help", language))
}

// printPluginCommandIndex prints a command index for one plugin group.
func printPluginCommandIndex(stdout io.Writer, bundle plugin.Bundle, commands []plugin.Command, language string) {
	if productName := localizedPluginText(bundle, language, "name", ""); productName != "" {
		fmt.Fprintf(stdout, "%s\n\n", productName)
	}
	type commandHelp struct {
		Path        string
		Description string
	}
	index := make([]commandHelp, 0, len(commands))
	maxPathWidth := 0
	for _, command := range commands {
		pathText := strings.Join(command.Path, " ")
		if len(pathText) > maxPathWidth {
			maxPathWidth = len(pathText)
		}
		index = append(index, commandHelp{
			Path:        pathText,
			Description: localizedPluginText(bundle, language, "command."+command.ID+".description", command.ID),
		})
	}
	fmt.Fprintf(stdout, "%s:\n", helpText("commands.heading", language))
	for _, command := range index {
		fmt.Fprintf(stdout, "  %-*s %s\n", maxPathWidth, command.Path, command.Description)
	}
	if len(commands) > 0 {
		fmt.Fprintf(stdout, "\n%s:\n", helpText("examples.heading", language))
		for _, command := range commands {
			fmt.Fprintf(stdout, "  ctyun help %s\n", strings.Join(command.Path, " "))
		}
	}
}

// printPluginHelp prints plugin-manager overview or subcommand help.
func printPluginHelp(stdout io.Writer, args []string, language string) bool {
	if len(args) == 1 {
		fmt.Fprintln(stdout, helpPageText("plugin.description", language))
		fmt.Fprintf(stdout, "\n%s:\n", helpText("usage.heading", language))
		fmt.Fprintln(stdout, "  ctyun plugin <subcommand> [options]")
		fmt.Fprintln(stdout, "  ctyun plugins <subcommand> [options]")
		fmt.Fprintln(stdout, "  ctyun help plugin <subcommand>")
		fmt.Fprintln(stdout, "  ctyun help plugins <subcommand>")
		fmt.Fprintf(stdout, "\n%s:\n", helpText("subcommands.heading", language))
		maxNameWidth := 0
		for _, command := range pluginSubcommandSummaries() {
			if len(pluginSubcommandNames(command)) > maxNameWidth {
				maxNameWidth = len(pluginSubcommandNames(command))
			}
		}
		for _, command := range pluginSubcommandSummaries() {
			fmt.Fprintf(stdout, "  %-*s  %s\n", maxNameWidth, pluginSubcommandNames(command), helpText(command.DescriptionKey, language))
		}
		return true
	}
	if len(args) != 2 {
		return false
	}
	for _, command := range pluginSubcommandSummaries() {
		if pluginSubcommandMatches(command, args[1]) {
			fmt.Fprintln(stdout, helpPageText(command.DescriptionKey, language))
			fmt.Fprintf(stdout, "\n%s:\n  %s\n", helpText("usage.heading", language), command.Usage)
			if len(command.Options) > 0 {
				fmt.Fprintf(stdout, "\n%s:\n", helpText("command.heading", language))
				maxNameWidth := 0
				for _, option := range command.Options {
					if len(option.Name) > maxNameWidth {
						maxNameWidth = len(option.Name)
					}
				}
				for _, option := range command.Options {
					fmt.Fprintf(stdout, "  %-*s  %s\n", maxNameWidth, option.Name, helpText(option.Key, language))
				}
			}
			return true
		}
	}
	return false
}

// pluginSubcommandNames joins a plugin-manager command and aliases for display.
func pluginSubcommandNames(command pluginSubcommandHelp) string {
	if len(command.Aliases) == 0 {
		return command.Name
	}
	return command.Name + "|" + strings.Join(command.Aliases, "|")
}

// pluginSubcommandMatches reports whether name is a plugin-manager command or
// alias.
func pluginSubcommandMatches(command pluginSubcommandHelp, name string) bool {
	if command.Name == name {
		return true
	}
	for _, alias := range command.Aliases {
		if alias == name {
			return true
		}
	}
	return false
}

// printGlobalOptions prints localized global option help.
func printGlobalOptions(stdout io.Writer, language string) {
	fmt.Fprintf(stdout, "\n%s:\n", helpText("global.heading", language))
	options := globalOptionsHelp
	maxNameWidth := 0
	for _, option := range options {
		if width := len(formatGlobalOptionNames(option)); width > maxNameWidth {
			maxNameWidth = width
		}
	}
	for _, option := range options {
		fmt.Fprintf(stdout, "  %-*s  %s\n", maxNameWidth, formatGlobalOptionNames(option), helpText(option.Key, language))
	}
}

// formatGlobalOptionNames formats short, long, alias, and value markers.
func formatGlobalOptionNames(option globalOptionHelp) string {
	name := option.Long
	if option.Short != "" {
		name = option.Short + ", " + option.Long
	}
	for _, alias := range option.Aliases {
		name += ", " + alias
	}
	if option.Value != "" {
		name += " <" + option.Value + ">"
	}
	return name
}

// helpPageText resolves and sentence-formats text used as a help page lead.
func helpPageText(key, language string) string {
	return helpPageDescription(helpText(key, language), language)
}

// helpPageDescription formats standalone help descriptions as sentences.
func helpPageDescription(text, language string) string {
	text = strings.TrimSpace(text)
	if text == "" || strings.ContainsRune(".。!?！？", lastRune(text)) {
		return text
	}
	return text + commonText("sentence.terminator", language)
}

// lastRune returns the last rune from a non-empty string.
func lastRune(text string) rune {
	var last rune
	for _, char := range text {
		last = char
	}
	return last
}

// helpText resolves a localized core help string with fallbacks.
func helpText(key, language string) string {
	return localizedCatalogText(helpCatalog, key, language)
}

// helpf formats localized core help text with fallbacks.
func helpf(key, language string, args ...any) string {
	return fmt.Sprintf(helpText(key, language), args...)
}

// visibleExamples hides fixture-only examples from user-facing help.
func visibleExamples(examples []string) []string {
	visible := make([]string, 0, len(examples))
	for _, example := range examples {
		if strings.Contains(example, "--offline") || strings.Contains(example, "--fixture") {
			continue
		}
		visible = append(visible, example)
	}
	return visible
}

// localizedPluginText resolves plugin-provided localized text.
func localizedPluginText(bundle plugin.Bundle, language, key, fallback string) string {
	if catalog, ok := bundle.I18N[language]; ok {
		if value := catalog[key]; value != "" {
			return value
		}
	}
	if catalog, ok := bundle.I18N["zh-CN"]; ok && language == "" {
		if value := catalog[key]; value != "" {
			return value
		}
	}
	return fallback
}

// parameterValidationHint formats allowed-value and regex hints for help.
func parameterValidationHint(parameter plugin.Parameter, language string) string {
	parts := make([]string, 0, 2)
	if len(parameter.AllowedValues) > 0 {
		parts = append(parts, helpf("validation.allowed", language, strings.Join(parameter.AllowedValues, ",")))
	}
	if parameter.Pattern != "" {
		parts = append(parts, helpf("validation.pattern", language, parameter.Pattern))
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, "; ") + "]"
}
