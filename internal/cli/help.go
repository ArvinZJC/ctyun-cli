/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// helpCatalog contains localized core help, help-only hints, and plugin-manager
// help labels.
//
//goland:noinspection SqlNoDataSourceInspection
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
	"plugins.heading":                       {"en-US": "Plugin Discovery", "en-GB": "Plugin Discovery", "zh-CN": "插件发现"},
	"commands.heading":                      {"en-US": "Commands", "en-GB": "Commands", "zh-CN": "命令"},
	"subcommands.heading":                   {"en-US": "Subcommands", "en-GB": "Subcommands", "zh-CN": "子命令"},
	"global.heading":                        {"en-US": "Global Options", "en-GB": "Global Options", "zh-CN": "全局选项"},
	"arguments.heading":                     {"en-US": "Arguments", "en-GB": "Arguments", "zh-CN": "参数"},
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
	"core.upgrade.option.source":            {"en-US": "Select the core release source", "en-GB": "Select the core release source", "zh-CN": "选择核心发布源"},
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
	"plugin.list.description":               {"en-US": "List installed or available plugins", "en-GB": "List installed or available plugins", "zh-CN": "列出已安装或可用插件"},
	"plugin.remove.description":             {"en-US": "Remove installed plugins", "en-GB": "Remove installed plugins", "zh-CN": "删除已安装插件"},
	"plugin.reinstall.description":          {"en-US": "Reinstall one or all installed plugins", "en-GB": "Reinstall one or all installed plugins", "zh-CN": "重新安装一个或全部已安装插件"},
	"plugin.search.description":             {"en-US": "Search hosted plugin metadata", "en-GB": "Search hosted plugin metadata", "zh-CN": "搜索托管插件元数据"},
	"plugin.update.description":             {"en-US": "Update or upgrade one or all installed plugins", "en-GB": "Update or upgrade one or all installed plugins", "zh-CN": "更新或升级一个或全部已安装插件"},
	"plugin.option.source":                  {"en-US": "Select the hosted plugin source", "en-GB": "Select the hosted plugin source", "zh-CN": "选择托管插件源"},
	"plugin.option.channel":                 {"en-US": "Select the registry channel", "en-GB": "Select the registry channel", "zh-CN": "选择插件源通道"},
	"plugin.option.updates":                 {"en-US": "Check installed plugins against hosted metadata", "en-GB": "Check installed plugins against hosted metadata", "zh-CN": "根据托管元数据检查已安装插件更新"},
	"plugin.option.available":               {"en-US": "List hosted plugins with installation status", "en-GB": "List hosted plugins with installation status", "zh-CN": "列出托管插件及安装状态"},
	"plugin.option.all.install":             {"en-US": "Install every available plugin", "en-GB": "Install every available plugin", "zh-CN": "安装全部可用插件"},
	"plugin.option.all.remove":              {"en-US": "Remove every installed plugin", "en-GB": "Remove every installed plugin", "zh-CN": "删除全部已安装插件"},
	"plugin.option.all.reinstall":           {"en-US": "Reinstall every installed plugin", "en-GB": "Reinstall every installed plugin", "zh-CN": "重新安装全部已安装插件"},
	"plugin.option.all.update":              {"en-US": "Update every installed plugin", "en-GB": "Update every installed plugin", "zh-CN": "更新全部已安装插件"},
	"plugin.column.name":                    {"en-US": "Name", "en-GB": "Name", "zh-CN": "名称"},
	"plugin.column.plugin":                  {"en-US": "Plugin", "en-GB": "Plugin", "zh-CN": "插件"},
	"plugin.column.version":                 {"en-US": "Version", "en-GB": "Version", "zh-CN": "版本"},
	"plugin.column.product":                 {"en-US": "Product", "en-GB": "Product", "zh-CN": "产品"},
	"plugin.column.channel":                 {"en-US": "Channel", "en-GB": "Channel", "zh-CN": "通道"},
	"plugin.column.quality":                 {"en-US": "Quality", "en-GB": "Quality", "zh-CN": "质量"},
	"plugin.column.status":                  {"en-US": "Status", "en-GB": "Status", "zh-CN": "状态"},
	"plugin.column.installed_version":       {"en-US": "Installed", "en-GB": "Installed", "zh-CN": "已安装版本"},
	"plugin.column.commands":                {"en-US": "Commands", "en-GB": "Commands", "zh-CN": "命令数"},
	"plugin.column.operations":              {"en-US": "Operations", "en-GB": "Operations", "zh-CN": "接口数"},
	"option.output":                         {"en-US": "Render output as a table or raw JSON", "en-GB": "Render output as a table or raw JSON", "zh-CN": "以表格或原始 JSON 输出"},
	"option.cols":                           {"en-US": "Select output columns by visible label or stable key", "en-GB": "Select output columns by visible label or stable key", "zh-CN": "按可见列名或稳定列键选择输出列"},
	"option.no_header":                      {"en-US": "Hide the table header", "en-GB": "Hide the table header", "zh-CN": "隐藏表头"},
	"option.filter":                         {"en-US": "Filter table rows by visible label or stable key", "en-GB": "Filter table rows by visible label or stable key", "zh-CN": "按可见列名或稳定列键过滤表格行"},
	"option.sort":                           {"en-US": "Sort table rows by visible label or stable key", "en-GB": "Sort table rows by visible label or stable key", "zh-CN": "按可见列名或稳定列键排序表格行"},
	"option.lang":                           {"en-US": "Choose help and output language", "en-GB": "Choose help and output language", "zh-CN": "选择帮助和输出语言"},
	"option.yes":                            {"en-US": "Confirm dangerous operations without prompting", "en-GB": "Confirm dangerous operations without prompting", "zh-CN": "无需提示直接确认危险操作"},
	"option.wait":                           {"en-US": "Evaluate a command waiter after the request", "en-GB": "Evaluate a command waiter after the request", "zh-CN": "请求后执行命令等待器"},
	"option.table":                          {"en-US": "Choose table style", "en-GB": "Choose table style", "zh-CN": "选择表格样式"},
	"option.timeout":                        {"en-US": "Set the per-request HTTP timeout", "en-GB": "Set the per-request HTTP timeout", "zh-CN": "设置单次 HTTP 请求超时"},
	"option.config":                         {"en-US": "Read profile configuration from a file", "en-GB": "Read profile configuration from a file", "zh-CN": "从文件读取配置档案"},
	"option.profile":                        {"en-US": "Select a named profile from the config file", "en-GB": "Select a named profile from the config file", "zh-CN": "从配置文件选择指定档案"},
	"option.debug":                          {"en-US": "Print HTTP request diagnostics to stderr", "en-GB": "Print HTTP request diagnostics to stderr", "zh-CN": "向 stderr 输出 HTTP 请求诊断信息"},
	"option.version":                        {"en-US": "Print the CLI version", "en-GB": "Print the CLI version", "zh-CN": "输出 CLI 版本"},
	"option.help":                           {"en-US": "Show help for the command", "en-GB": "Show help for the command", "zh-CN": "显示命令帮助"},
	"option.default_hint":                   {"en-US": " (default: %s)", "en-GB": " (default: %s)", "zh-CN": "（默认：%s）"},
}

// runHelp routes help requests to core, plugin manager, or product-command help.
func runHelp(stdout io.Writer, args []string, installedRoot, language string) error {
	if len(args) == 0 {
		return printMainHelp(stdout, language)
	}
	handled, err := printCoreHelp(stdout, args, language)
	if err != nil {
		return err
	}
	if handled {
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
			return diagnostic.New("error.unknown_command", strings.Join(args, " "))
		}
		return printPluginCommandIndex(stdout, bundle, args, commands, language)
	}
	writer := newOutputWriter(stdout)
	if description := localizedPluginText(bundle, language, "command."+command.ID+".description", ""); description != "" {
		writer.Format("%s\n", helpPageDescription(description, language))
	}
	if productName := localizedPluginText(bundle, language, "name", ""); productName != "" {
		if description := localizedPluginText(bundle, language, "command."+command.ID+".description", ""); description != "" {
			writer.Line()
		}
		writer.Format("%s: %s\n", helpText("product.label", language), productName)
	}
	writer.Format("\n%s:\n", helpText("usage.heading", language))
	writer.Line(pluginCommandUsage(command, language))
	if rows := pluginCommandArgumentHelpRows(bundle, command, language); len(rows) > 0 {
		writer.Format("\n%s:\n", helpText("arguments.heading", language))
		writeAlignedHelpRows(writer, rows, "  ")
	}
	if len(command.Parameters) > 0 {
		writer.Format("\n%s:\n", helpText("command.heading", language))
		writeAlignedHelpRows(writer, pluginCommandParameterHelpRows(bundle, command, language), "  ")
	}
	printGlobalOptionsTo(writer, language)
	if table, ok := bundle.Tables.Tables[command.Table]; ok && len(table.Columns) > 0 {
		labels := make([]string, 0, len(table.Columns))
		for _, column := range tableColumns(table, language) {
			labels = append(labels, column.Label)
		}
		writer.Format("\n%s:\n  %s\n", helpText("columns.heading", language), strings.Join(labels, ","))
	}
	examples := visibleExamples(command.Examples)
	if len(examples) > 0 {
		writer.Format("\n%s:\n", helpText("examples.heading", language))
		columns := tableColumns(bundle.Tables.Tables[command.Table], language)
		for _, example := range examples {
			writer.Format("  %s\n", localizedExampleSelectors(example, columns))
		}
	}
	if command.DocsURL != "" {
		writer.Format("\n%s:\n  %s\n", helpText("docs.heading", language), command.DocsURL)
	}
	return writer.Err()
}

// pluginCommandUsage formats the runtime usage line for one product command.
func pluginCommandUsage(command plugin.Command, language string) string {
	usage := fmt.Sprintf("  ctyun [%s] %s", helpText("usage.global", language), strings.Join(command.Path, " "))
	for _, parameter := range command.Parameters {
		usage += " " + parameterUsageToken(parameter)
	}
	return usage
}

// pluginCommandParameterHelpRows returns help rows for command-specific flags.
func pluginCommandParameterHelpRows(bundle plugin.Bundle, command plugin.Command, language string) []helpRow {
	rows := make([]helpRow, 0, len(command.Parameters))
	for _, parameter := range command.Parameters {
		rows = append(rows, helpRow{
			Name:        parameterOptionToken(parameter),
			Description: parameterHelpDescription(bundle, command, parameter, language),
		})
	}
	return rows
}

// parameterUsageToken formats one command flag for the usage line.
func parameterUsageToken(parameter plugin.Parameter) string {
	token := parameterOptionToken(parameter)
	if parameter.Required {
		return token
	}
	return "[" + token + "]"
}

// parameterOptionToken formats one command flag and its value placeholder.
func parameterOptionToken(parameter plugin.Parameter) string {
	return "--" + parameter.Flag + " <" + parameterValuePlaceholder(parameter) + ">"
}

// parameterValuePlaceholder returns the help placeholder for one command flag.
func parameterValuePlaceholder(parameter plugin.Parameter) string {
	if len(parameter.AllowedValues) > 0 {
		return strings.Join(parameter.AllowedValues, "|")
	}
	return parameter.Flag
}

// parameterHelpDescription returns localized help text for one command flag.
func parameterHelpDescription(bundle plugin.Bundle, command plugin.Command, parameter plugin.Parameter, language string) string {
	description := localizedPluginText(bundle, language, "parameter."+command.ID+"."+parameter.Name+".description", parameter.Description)
	if parameter.Required {
		if description != "" {
			description += " "
		}
		description += "(" + helpText("required", language) + ")"
	}
	if validation := strings.TrimSpace(parameterValidationHint(parameter, language)); validation != "" {
		if description != "" {
			description += " "
		}
		description += validation
	}
	return description
}

// pluginCommandArgumentHelpRows returns help rows for required path arguments.
func pluginCommandArgumentHelpRows(bundle plugin.Bundle, command plugin.Command, language string) []helpRow {
	arguments := commandPathArguments(command.Path)
	rows := make([]helpRow, 0, len(arguments))
	for _, argument := range arguments {
		description := pluginCommandArgumentDescription(bundle, command, argument, language)
		rows = append(rows, helpRow{Name: "{" + argument + "}", Description: description})
	}
	return rows
}

// commandPathArguments extracts placeholder names from a command path.
func commandPathArguments(path []string) []string {
	arguments := make([]string, 0)
	for _, segment := range path {
		if name, ok := commandPathArgumentName(segment); ok {
			arguments = append(arguments, name)
		}
	}
	return arguments
}

// commandPathArgumentName returns the placeholder name from one path segment.
func commandPathArgumentName(segment string) (string, bool) {
	if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
	return name, name != ""
}

// pluginCommandArgumentDescription resolves a localized path argument label.
func pluginCommandArgumentDescription(bundle plugin.Bundle, command plugin.Command, argument, language string) string {
	if text := localizedPluginText(bundle, language, "argument."+command.ID+"."+argument+".description", ""); text != "" {
		return text
	}
	if table, ok := bundle.Tables.Tables[command.Table]; ok {
		if label := tableColumnLabel(table, argument, language); label != "" {
			return label
		}
	}
	for tableID, table := range bundle.Tables.Tables {
		if tableID == command.Table {
			continue
		}
		if label := tableColumnLabel(table, argument, language); label != "" {
			return label
		}
	}
	return ""
}

// tableColumnLabel resolves one localized table column label by stable key.
func tableColumnLabel(table plugin.Table, key, language string) string {
	for _, column := range tableColumns(table, language) {
		if column.Key == key {
			return column.Label
		}
	}
	return ""
}

// matchPluginCommandForHelp finds an exact plugin command for help output.
func matchPluginCommandForHelp(args []string, bundles []plugin.Bundle) (plugin.Bundle, plugin.Command, bool) {
	for _, bundle := range bundles {
		if command, ok := plugin.FindCommand(bundle, args); ok {
			return bundle, command, true
		}
		if command, _, ok := plugin.FindCommandMissingPathArgs(bundle, args); ok {
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
func printMainHelp(stdout io.Writer, language string) error {
	writer := newOutputWriter(stdout)
	writer.Lines(
		helpText("description.line1", language),
		helpText("description.line2", language),
		"",
	)
	writer.Format("%s:\n", helpText("usage.heading", language))
	writer.Format("  ctyun [%s] <%s> [%s]\n", helpText("usage.global", language), helpText("usage.command", language), helpText("usage.command_opts", language))
	writer.Format("  ctyun help <%s>\n", helpText("usage.command", language))
	writer.Line()
	writer.Format("%s:\n", helpText("core.heading", language))
	writeAlignedHelpRows(writer, commandSummaryHelpRows(coreCommandSummaries(language)), "  ")
	printPluginCommandHintsTo(writer, language)
	printGlobalOptionsTo(writer, language)
	return writer.Err()
}

// helpRow is one already-localized two-column help row.
type helpRow struct {
	Name        string
	Description string
}

// commandSummary is one row in the core command summary.
type commandSummary struct {
	Name        string
	Description string
}

// pluginOptionSummary is one plugin-manager option row.
type pluginOptionSummary struct {
	Name    string
	Key     string
	Default string
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
			Usage:          "ctyun plugin install <name...|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]\n  ctyun plugins install <name...|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]",
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.install"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "list",
			DescriptionKey: "plugin.list.description",
			Usage:          "ctyun plugin list [--available|--updates] [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]\n  ctyun plugins list [--available|--updates] [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]",
			Options: []pluginOptionSummary{
				{Name: "--available", Key: "plugin.option.available"},
				{Name: "--updates", Key: "plugin.option.updates"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "remove",
			DescriptionKey: "plugin.remove.description",
			Usage:          "ctyun plugin remove <name...|--all>\n  ctyun plugins remove <name...|--all>",
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.remove"},
			},
		},
		{
			Name:           "reinstall",
			DescriptionKey: "plugin.reinstall.description",
			Usage:          "ctyun plugin reinstall <name...|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]\n  ctyun plugins reinstall <name...|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]",
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.reinstall"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "search",
			DescriptionKey: "plugin.search.description",
			Usage:          "ctyun plugin search <query> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]\n  ctyun plugins search <query> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]",
			Options: []pluginOptionSummary{
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
		{
			Name:           "update",
			Aliases:        []string{"upgrade"},
			DescriptionKey: "plugin.update.description",
			Usage:          "ctyun plugin update <name|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]\n  ctyun plugin upgrade <name|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]\n  ctyun plugins update <name|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]\n  ctyun plugins upgrade <name|--all> [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]",
			Options: []pluginOptionSummary{
				{Name: "--all", Key: "plugin.option.all.update"},
				{Name: "--source <auto|github|gitee>", Key: "plugin.option.source", Default: "auto"},
				{Name: "--channel <stable|beta|alpha>", Key: "plugin.option.channel", Default: "stable"},
			},
		},
	}
}

// printCoreHelp prints help for a built-in command when args match one.
func printCoreHelp(stdout io.Writer, args []string, language string) (bool, error) {
	writer := newOutputWriter(stdout)
	switch args[0] {
	case "config":
		handled, err := printConfigHelp(stdout, args, language)
		if !handled || err != nil {
			return handled, err
		}
	case "completion":
		writer.Line(helpPageText("core.completion", language))
		writer.Format("\n%s:\n  ctyun completion <bash|zsh|fish|powershell>\n", helpText("usage.heading", language))
	case "doctor":
		handled, err := printDoctorHelp(stdout, args, language)
		if !handled || err != nil {
			return handled, err
		}
	case "help":
		writer.Line(helpPageText("core.help", language))
		writer.Format("\n%s:\n  ctyun help [command]\n", helpText("usage.heading", language))
	case "plugin", "plugins":
		handled, err := printPluginHelp(stdout, args, language)
		if !handled || err != nil {
			return handled, err
		}
	case "upgrade", "update":
		writer.Lines(
			helpPageText("core.upgrade", language),
			helpPageText("core.upgrade.plugins", language),
		)
		writer.Format("\n%s:\n", helpText("usage.heading", language))
		writer.Lines(
			"  ctyun update [--check] [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]",
			"  ctyun upgrade [--check] [--source <auto|github|gitee>] [--channel <stable|beta|alpha>]",
		)
		writer.Format("\n%s:\n", helpText("command.heading", language))
		writeAlignedHelpRows(writer, []helpRow{
			{Name: "--check", Description: helpText("core.upgrade.option.check", language)},
			{Name: "--source <auto|github|gitee>", Description: optionHelpText("core.upgrade.option.source", "auto", language)},
			{Name: "--channel <stable|beta|alpha>", Description: optionHelpText("core.upgrade.option.channel", upgradeChannel(""), language)},
		}, "  ")
	case "version":
		writer.Line(helpPageText("core.version", language))
		writer.Format("\n%s:\n  ctyun version\n", helpText("usage.heading", language))
	default:
		return false, nil
	}
	printGlobalOptionsTo(writer, language)
	return true, writer.Err()
}

// printDoctorHelp prints help for doctor and doctor subcommands.
func printDoctorHelp(stdout io.Writer, args []string, language string) (bool, error) {
	writer := newOutputWriter(stdout)
	if len(args) == 1 {
		writer.Line(helpPageText("doctor.description", language))
		writer.Format("\n%s:\n", helpText("usage.heading", language))
		writer.Lines(
			"  ctyun doctor <subcommand>",
			"  ctyun help doctor <subcommand>",
		)
		writer.Format("\n%s:\n", helpText("subcommands.heading", language))
		writer.Format("  %-8s  %s\n", "network", helpText("doctor.network.description", language))
		return true, writer.Err()
	}
	if len(args) == 2 && args[1] == "network" {
		writer.Line(helpPageText("doctor.network.description", language))
		writer.Format("\n%s:\n  ctyun doctor network\n", helpText("usage.heading", language))
		return true, writer.Err()
	}
	return false, nil
}

// printPluginCommandHints prints discovery hints for product commands.
func printPluginCommandHints(stdout io.Writer, language string) error {
	writer := newOutputWriter(stdout)
	printPluginCommandHintsTo(writer, language)
	return writer.Err()
}

// printPluginCommandHintsTo writes product-command discovery hints.
func printPluginCommandHintsTo(writer *outputWriter, language string) {
	writer.Format("\n%s:\n", helpText("plugins.heading", language))
	writer.Format("  %-20s %s\n", "ctyun plugin list", helpText("plugin.hint.list", language))
	writer.Format("  %-20s %s\n", "ctyun plugins list", helpText("plugin.hint.list", language))
	writer.Format("  %-20s %s\n", "ctyun help <plugin>", helpText("plugin.hint.help", language))
}

// printPluginCommandIndex prints unified subcommand help for one plugin group.
func printPluginCommandIndex(stdout io.Writer, bundle plugin.Bundle, prefix []string, commands []plugin.Command, language string) error {
	writer := newOutputWriter(stdout)
	if productName := localizedPluginText(bundle, language, "name", ""); productName != "" {
		writer.Format("%s\n\n", productName)
	}
	prefixText := strings.Join(prefix, " ")
	writer.Format("%s:\n", helpText("usage.heading", language))
	writer.Format("  ctyun %s <subcommand> [%s]\n", prefixText, helpText("usage.command_opts", language))
	writer.Format("  ctyun help %s <subcommand>\n", prefixText)
	writer.Format("\n%s:\n", helpText("subcommands.heading", language))
	writeAlignedHelpRows(writer, pluginCommandGroupHelpRows(bundle, prefix, commands, language), "  ")
	printGlobalOptionsTo(writer, language)
	return writer.Err()
}

// pluginCommandGroupHelpRows returns one help row per immediate subcommand.
func pluginCommandGroupHelpRows(bundle plugin.Bundle, prefix []string, commands []plugin.Command, language string) []helpRow {
	rows := make([]helpRow, 0, len(commands))
	seen := make(map[string]bool)
	for _, command := range commands {
		if len(command.Path) <= len(prefix) {
			continue
		}
		name := command.Path[len(prefix)]
		if seen[name] {
			continue
		}
		seen[name] = true
		rows = append(rows, helpRow{
			Name:        name,
			Description: localizedPluginText(bundle, language, "command."+command.ID+".description", command.ID),
		})
	}
	return rows
}

// printPluginHelp prints plugin-manager overview or subcommand help.
func printPluginHelp(stdout io.Writer, args []string, language string) (bool, error) {
	writer := newOutputWriter(stdout)
	if len(args) == 1 {
		writer.Line(helpPageText("plugin.description", language))
		writer.Format("\n%s:\n", helpText("usage.heading", language))
		writeUsageLines(writer, pluginOverviewUsageLines())
		writer.Format("\n%s:\n", helpText("subcommands.heading", language))
		writeAlignedHelpRows(writer, pluginSubcommandHelpRows(pluginSubcommandSummaries(), language), "  ")
		return true, writer.Err()
	}
	if len(args) != 2 {
		return false, nil
	}
	for _, command := range pluginSubcommandSummaries() {
		if pluginSubcommandMatches(command, args[1]) {
			writer.Line(helpPageText(command.DescriptionKey, language))
			writer.Format("\n%s:\n  %s\n", helpText("usage.heading", language), command.Usage)
			if len(command.Options) > 0 {
				writer.Format("\n%s:\n", helpText("command.heading", language))
				writeAlignedHelpRows(writer, pluginOptionHelpRows(command.Options, language), "  ")
			}
			return true, writer.Err()
		}
	}
	return false, nil
}

// pluginOverviewUsageLines returns concrete plugin-manager usage forms for the
// plugin help overview.
func pluginOverviewUsageLines() []string {
	lines := make([]string, 0, len(pluginSubcommandSummaries())*2+2)
	for _, command := range pluginSubcommandSummaries() {
		for _, line := range strings.Split(command.Usage, "\n") {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return append(lines,
		"ctyun help plugin <subcommand>",
		"ctyun help plugins <subcommand>",
	)
}

// writeUsageLines writes already-formatted usage lines with standard
// indentation.
func writeUsageLines(writer *outputWriter, lines []string) {
	for _, line := range lines {
		writer.Format("  %s\n", line)
	}
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
func printGlobalOptions(stdout io.Writer, language string) error {
	writer := newOutputWriter(stdout)
	printGlobalOptionsTo(writer, language)
	return writer.Err()
}

// printGlobalOptionsTo writes localized global option help.
func printGlobalOptionsTo(writer *outputWriter, language string) {
	writer.Format("\n%s:\n", helpText("global.heading", language))
	writeAlignedHelpRows(writer, globalOptionHelpRows(language), "  ")
}

// writeAlignedHelpRows writes two-column help rows using the widest name.
func writeAlignedHelpRows(writer *outputWriter, rows []helpRow, separator string) {
	maxNameWidth := 0
	for _, row := range rows {
		if width := len(row.Name); width > maxNameWidth {
			maxNameWidth = width
		}
	}
	for _, row := range rows {
		writer.Format("  %-*s%s%s\n", maxNameWidth, row.Name, separator, row.Description)
	}
}

// commandSummaryHelpRows converts core command summaries to aligned help rows.
func commandSummaryHelpRows(commands []commandSummary) []helpRow {
	rows := make([]helpRow, 0, len(commands))
	for _, command := range commands {
		rows = append(rows, helpRow{Name: command.Name, Description: command.Description})
	}
	return rows
}

// pluginSubcommandHelpRows converts plugin subcommands to aligned help rows.
func pluginSubcommandHelpRows(commands []pluginSubcommandHelp, language string) []helpRow {
	rows := make([]helpRow, 0, len(commands))
	for _, command := range commands {
		rows = append(rows, helpRow{
			Name:        pluginSubcommandNames(command),
			Description: helpText(command.DescriptionKey, language),
		})
	}
	return rows
}

// pluginOptionHelpRows converts command options to aligned help rows.
func pluginOptionHelpRows(options []pluginOptionSummary, language string) []helpRow {
	rows := make([]helpRow, 0, len(options))
	for _, option := range options {
		rows = append(rows, helpRow{
			Name:        option.Name,
			Description: optionHelpText(option.Key, option.Default, language),
		})
	}
	return rows
}

// globalOptionHelpRows converts global options to aligned help rows.
func globalOptionHelpRows(language string) []helpRow {
	rows := make([]helpRow, 0, len(globalOptionsHelp))
	for _, option := range globalOptionsHelp {
		rows = append(rows, helpRow{
			Name:        formatGlobalOptionNames(option),
			Description: optionHelpText(option.Key, option.Default, language),
		})
	}
	return rows
}

// optionHelpText appends a localized default-value hint when an option has a
// fixed runtime default.
func optionHelpText(key, defaultValue, language string) string {
	text := helpText(key, language)
	if defaultValue == "" {
		return text
	}
	return text + helpf("option.default_hint", language, defaultValue)
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

// localizedExampleSelectors renders table-control examples with visible column
// labels while leaving command syntax and non-selector values unchanged.
func localizedExampleSelectors(example string, columns []output.Column) string {
	if len(columns) == 0 {
		return example
	}
	labels := make(map[string]string, len(columns))
	for _, column := range columns {
		labels[column.Key] = column.Label
	}
	fields := strings.Fields(example)
	for i := 0; i < len(fields); i++ {
		switch fields[i] {
		case "--cols", "-c":
			if i+1 < len(fields) {
				fields[i+1] = quoteExampleValue(localizedColumnList(fields[i+1], labels))
				i++
			}
		case "--filter", "-f":
			if i+1 < len(fields) {
				fields[i+1] = quoteExampleValue(localizedFilterSelector(fields[i+1], labels))
				i++
			}
		case "--sort", "-s":
			if i+1 < len(fields) {
				fields[i+1] = quoteExampleValue(localizedSortSelector(fields[i+1], labels))
				i++
			}
		default:
			if value, ok := strings.CutPrefix(fields[i], "--cols="); ok {
				fields[i] = "--cols=" + quoteExampleValue(localizedColumnList(value, labels))
			} else if value, ok := strings.CutPrefix(fields[i], "--filter="); ok {
				fields[i] = "--filter=" + quoteExampleValue(localizedFilterSelector(value, labels))
			} else if value, ok := strings.CutPrefix(fields[i], "--sort="); ok {
				fields[i] = "--sort=" + quoteExampleValue(localizedSortSelector(value, labels))
			}
		}
	}
	return strings.Join(fields, " ")
}

// localizedColumnList rewrites comma-separated stable column keys to labels.
func localizedColumnList(value string, labels map[string]string) string {
	parts := strings.Split(value, ",")
	for i, part := range parts {
		if label := labels[part]; label != "" {
			parts[i] = label
		}
	}
	return strings.Join(parts, ",")
}

// localizedFilterSelector rewrites the filter key before the first '='.
func localizedFilterSelector(value string, labels map[string]string) string {
	key, rest, ok := strings.Cut(value, "=")
	if !ok {
		return value
	}
	if label := labels[key]; label != "" {
		key = label
	}
	return key + "=" + rest
}

// localizedSortSelector rewrites a stable sort key, preserving descending
// order markers.
func localizedSortSelector(value string, labels map[string]string) string {
	descending := strings.HasPrefix(value, "-")
	key := strings.TrimPrefix(value, "-")
	if label := labels[key]; label != "" {
		key = label
	}
	if descending {
		return "-" + key
	}
	return key
}

// quoteExampleValue quotes rendered selector values only when the shell would
// otherwise split them.
func quoteExampleValue(value string) string {
	if !strings.ContainsAny(value, " \t\n") {
		return value
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
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
	if parameter.Pattern != "" {
		parts = append(parts, helpf("validation.pattern", language, parameter.Pattern))
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, "; ") + "]"
}
