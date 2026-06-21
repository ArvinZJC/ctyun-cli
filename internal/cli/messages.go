/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
)

// messageCatalog contains localized runtime warnings, statuses, diagnostics,
// and error templates.
var messageCatalog = map[string]map[string]string{
	"usage.missing_command":              {"en-US": "Usage: ctyun <command>", "en-GB": "Usage: ctyun <command>", "zh-CN": "用法：ctyun <命令>"},
	"error.prefix":                       {"en-US": "Error: %s", "en-GB": "Error: %s", "zh-CN": "错误：%s"},
	"error.missing_command":              {"en-US": "missing command", "en-GB": "missing command", "zh-CN": "缺少命令"},
	"error.doctor_supports":              {"en-US": "doctor supports: network", "en-GB": "doctor supports: network", "zh-CN": "doctor 支持：network"},
	"error.requires_value":               {"en-US": "%s requires a value", "en-GB": "%s requires a value", "zh-CN": "%s 需要一个值"},
	"error.unknown_command":              {"en-US": "unknown command %q", "en-GB": "unknown command %q", "zh-CN": "未知命令 %q"},
	"error.invalid_pattern":              {"en-US": "%s --%s has invalid validation pattern: %w", "en-GB": "%s --%s has invalid validation pattern: %w", "zh-CN": "%s --%s 的校验表达式无效: %w"},
	"error.pattern_mismatch":             {"en-US": "%s --%s does not match %s", "en-GB": "%s --%s does not match %s", "zh-CN": "%s --%s 不匹配 %s"},
	"error.plugin_version":               {"en-US": "plugin %s requires ctyun %s, current version is %s", "en-GB": "plugin %s requires ctyun %s, current version is %s", "zh-CN": "插件 %s 需要 ctyun %s，当前版本是 %s"},
	"error.plugin_subcommand":            {"en-US": "plugin requires a subcommand", "en-GB": "plugin requires a subcommand", "zh-CN": "plugin 需要子命令"},
	"error.plugin_name":                  {"en-US": "invalid plugin name %q", "en-GB": "invalid plugin name %q", "zh-CN": "无效插件名称 %q"},
	"error.upgrade_option":               {"en-US": "unknown upgrade option %q", "en-GB": "unknown upgrade option %q", "zh-CN": "未知 upgrade 选项 %q"},
	"error.unknown_plugin_subcommand":    {"en-US": "unknown plugin subcommand %q", "en-GB": "unknown plugin subcommand %q", "zh-CN": "未知 plugin 子命令 %q"},
	"error.plugin_install_name":          {"en-US": "plugin install requires a plugin name", "en-GB": "plugin install requires a plugin name", "zh-CN": "plugin install 需要插件名称"},
	"error.plugin_remove_name":           {"en-US": "plugin remove requires a plugin name", "en-GB": "plugin remove requires a plugin name", "zh-CN": "plugin remove 需要插件名称"},
	"error.plugin_lint_path":             {"en-US": "plugin lint requires a bundle path", "en-GB": "plugin lint requires a bundle path", "zh-CN": "plugin lint 需要插件包路径"},
	"error.plugin_bundled_update_target": {"en-US": "plugin update/upgrade --bundled requires a plugin name or --all", "en-GB": "plugin update/upgrade --bundled requires a plugin name or --all", "zh-CN": "plugin update/upgrade --bundled 需要插件名称或 --all"},
	"error.plugin_update_target":         {"en-US": "plugin update/upgrade requires a plugin name or --all", "en-GB": "plugin update/upgrade requires a plugin name or --all", "zh-CN": "plugin update/upgrade 需要插件名称或 --all"},
	"error.hosted_plugin_dev":            {"en-US": "hosted plugin updates are unavailable for development builds; use --bundled", "en-GB": "hosted plugin updates are unavailable for development builds; use --bundled", "zh-CN": "开发构建不可使用托管插件更新；请使用 --bundled"},
	"error.plugin_install_one_name":      {"en-US": "plugin install accepts one plugin name", "en-GB": "plugin install accepts one plugin name", "zh-CN": "plugin install 只接受一个插件名称"},
	"error.plugin_install_source_choice": {"en-US": "plugin install accepts either --bundled or --source", "en-GB": "plugin install accepts either --bundled or --source", "zh-CN": "plugin install 只能使用 --bundled 或 --source 其中之一"},
	"error.plugin_search_one_query":      {"en-US": "plugin search accepts one query", "en-GB": "plugin search accepts one query", "zh-CN": "plugin search 只接受一个查询词"},
	"error.plugin_update_one_name":       {"en-US": "plugin update accepts one plugin name", "en-GB": "plugin update accepts one plugin name", "zh-CN": "plugin update 只接受一个插件名称"},
	"error.plugin_update_all_or_one":     {"en-US": "plugin update accepts either --all or one plugin name", "en-GB": "plugin update accepts either --all or one plugin name", "zh-CN": "plugin update 只能使用 --all 或一个插件名称"},
	"error.plugin_update_source_choice":  {"en-US": "plugin update accepts either --bundled or --source", "en-GB": "plugin update accepts either --bundled or --source", "zh-CN": "plugin update 只能使用 --bundled 或 --source 其中之一"},
	"error.plugin_source_empty":          {"en-US": "plugin source is empty", "en-GB": "plugin source is empty", "zh-CN": "插件源为空"},
	"error.source_requires_value":        {"en-US": "--source requires a value", "en-GB": "--source requires a value", "zh-CN": "--source 需要一个值"},
	"error.channel_requires_value":       {"en-US": "--channel requires a value", "en-GB": "--channel requires a value", "zh-CN": "--channel 需要一个值"},
	"error.bundled_dev_only":             {"en-US": "--bundled is only available in development builds", "en-GB": "--bundled is only available in development builds", "zh-CN": "--bundled 仅可用于开发构建"},
	"error.confirmation_required":        {"en-US": "confirmation required for %s: rerun with --yes", "en-GB": "confirmation required for %s: rerun with --yes", "zh-CN": "%s 需要确认：请使用 --yes 重新执行"},
	"error.unexpected_argument":          {"en-US": "unexpected argument %q for %s", "en-GB": "unexpected argument %q for %s", "zh-CN": "%[2]s 不支持参数 %[1]q"},
	"error.flag_requires_value":          {"en-US": "--%s requires a value", "en-GB": "--%s requires a value", "zh-CN": "--%s 需要一个值"},
	"error.unknown_option":               {"en-US": "unknown option --%s for %s", "en-GB": "unknown option --%s for %s", "zh-CN": "%[2]s 不支持选项 --%[1]s"},
	"error.missing_required_flag":        {"en-US": "%s requires --%s", "en-GB": "%s requires --%s", "zh-CN": "%s 需要 --%s"},
	"error.allowed_values":               {"en-US": "%s --%s must be one of %s", "en-GB": "%s --%s must be one of %s", "zh-CN": "%s --%s 必须是以下值之一 %s"},
	"doctor.network.source":              {"en-US": "Plugin source: use --source or CTYUN_PLUGIN_SOURCE.", "en-GB": "Plugin source: use --source or CTYUN_PLUGIN_SOURCE.", "zh-CN": "插件源：使用 --source 或 CTYUN_PLUGIN_SOURCE。"},
	"doctor.network.mirrors":             {"en-US": "Mirrors: auto uses GitHub release assets first, then falls back to Gitee.", "en-GB": "Mirrors: auto uses GitHub release assets first, then falls back to Gitee.", "zh-CN": "镜像：auto 先使用 GitHub 发布资产，失败后回退到 Gitee。"},
	"doctor.network.live_api":            {"en-US": "Live API: retrieval commands prefer CTYUN_AK and CTYUN_SK, then profile or global config ak/sk.", "en-GB": "Live API: retrieval commands prefer CTYUN_AK and CTYUN_SK, then profile or global config ak/sk.", "zh-CN": "实时 API：查询命令优先使用 CTYUN_AK 和 CTYUN_SK，然后使用配置档案或全局配置中的 ak/sk。"},
	"upgrade.dev_unavailable":            {"en-US": "Self-upgrade is unavailable for development builds without an explicit release source.", "en-GB": "Self-upgrade is unavailable for development builds without an explicit release source.", "zh-CN": "开发构建未指定发布源时不可执行自升级。"},
	"upgrade.dev_guidance":               {"en-US": "Use a released ctyun build or test hosted metadata with --source auto|github|gitee.", "en-GB": "Use a released ctyun build or test hosted metadata with --source auto|github|gitee.", "zh-CN": "请使用已发布的 ctyun 构建，或通过 --source auto|github|gitee 测试托管元数据。"},
	"upgrade.current":                    {"en-US": "ctyun %s is already up to date on channel %s.", "en-GB": "ctyun %s is already up to date on channel %s.", "zh-CN": "ctyun %s 在 %s 渠道已是最新版本。"},
	"upgrade.installed":                  {"en-US": "Upgraded %s: %s -> %s.", "en-GB": "Upgraded %s: %s -> %s.", "zh-CN": "已升级 %s：%s -> %s。"},
	"upgrade.available":                  {"en-US": "ctyun %s is available from %s for %s/%s (%s).", "en-GB": "ctyun %s is available from %s for %s/%s (%s).", "zh-CN": "ctyun %s 可从 %s 获取，适用于 %s/%s（%s）。"},
	"plugin.installed":                   {"en-US": "Installed %s.", "en-GB": "Installed %s.", "zh-CN": "已安装 %s。"},
	"plugin.removed":                     {"en-US": "Removed %s.", "en-GB": "Removed %s.", "zh-CN": "已删除 %s。"},
	"plugin.valid":                       {"en-US": "Valid plugin %s %s.", "en-GB": "Valid plugin %s %s.", "zh-CN": "有效插件 %s %s。"},
	"plugin.updated":                     {"en-US": "Updated %s: %s -> %s.", "en-GB": "Updated %s: %s -> %s.", "zh-CN": "已更新 %s：%s -> %s。"},
	"plugin.current":                     {"en-US": "%s is already up to date.", "en-GB": "%s is already up to date.", "zh-CN": "%s 已是最新版本。"},
	"plugin.update_available":            {"en-US": "Update available for %s: %s -> %s.", "en-GB": "Update available for %s: %s -> %s.", "zh-CN": "%s 可更新：%s -> %s。"},
	"config.updated":                     {"en-US": "Updated %s.", "en-GB": "Updated %s.", "zh-CN": "已更新 %s。"},
	"config.unset":                       {"en-US": "Unset %s.", "en-GB": "Unset %s.", "zh-CN": "已取消 %s。"},
	"config.active_profile":              {"en-US": "Active profile: %s.", "en-GB": "Active profile: %s.", "zh-CN": "当前配置档案：%s。"},
	"config.profile_updated":             {"en-US": "Updated profile %s %s.", "en-GB": "Updated profile %s %s.", "zh-CN": "已更新配置档案 %s 的 %s。"},
	"config.profile_unset":               {"en-US": "Unset profile %s %s.", "en-GB": "Unset profile %s %s.", "zh-CN": "已取消配置档案 %s 的 %s。"},
	"config.profile_reset":               {"en-US": "Reset profile %s.", "en-GB": "Reset profile %s.", "zh-CN": "已重置配置档案 %s。"},
	"config.already_reset":               {"en-US": "Config is already reset.", "en-GB": "Config is already reset.", "zh-CN": "配置已重置。"},
	"config.reset":                       {"en-US": "Reset config; backup: %s.", "en-GB": "Reset config; backup: %s.", "zh-CN": "已重置配置；备份：%s。"},
	"warning.config_credentials": {
		"en-US": "Warning: using CTyun AK/SK from config. Disable this warning by setting the CTYUN_WARN_CONFIG_CREDENTIALS=0 environment variable or running ctyun config set warn_config_credentials false.",
		"en-GB": "Warning: using CTyun AK/SK from config. Disable this warning by setting the CTYUN_WARN_CONFIG_CREDENTIALS=0 environment variable or running ctyun config set warn_config_credentials false.",
		"zh-CN": "警告：正在使用配置中的天翼云 AK/SK。可设置环境变量 CTYUN_WARN_CONFIG_CREDENTIALS=0，或运行 ctyun config set warn_config_credentials false 关闭此提醒。",
	},
}

// messageText resolves localized runtime text with fallbacks.
func messageText(key, language string) string {
	return localizedCatalogText(messageCatalog, key, language)
}

// messagef formats localized runtime text with fallbacks.
func messagef(key, language string, args ...any) string {
	return fmt.Sprintf(messageText(key, language), args...)
}

// missingCommandUsageLine returns the localized short usage shown before the
// missing-command error.
func missingCommandUsageLine(language string) string {
	return messageText("usage.missing_command", language)
}

// doctorNetworkMessages returns localized network diagnostic guidance.
func doctorNetworkMessages(language string) []string {
	return []string{
		messageText("doctor.network.source", language),
		messageText("doctor.network.mirrors", language),
		messageText("doctor.network.live_api", language),
	}
}

// upgradeDevelopmentMessages returns localized guidance for unreleased dev
// builds without an explicit source.
func upgradeDevelopmentMessages(language string) []string {
	return []string{
		messageText("upgrade.dev_unavailable", language),
		messageText("upgrade.dev_guidance", language),
	}
}

// upgradeCurrentMessage returns the localized already-current status.
func upgradeCurrentMessage(language, currentVersion, channel string) string {
	return messagef("upgrade.current", language, currentVersion, channel)
}

// upgradeInstalledMessage returns the localized installed-upgrade status.
func upgradeInstalledMessage(language, name, currentVersion, nextVersion string) string {
	return messagef("upgrade.installed", language, name, currentVersion, nextVersion)
}

// upgradeAvailableMessage returns the localized check-only update status.
func upgradeAvailableMessage(language, nextVersion, source, goos, goarch, artifact string) string {
	return messagef("upgrade.available", language, nextVersion, source, goos, goarch, artifact)
}

// pluginInstalledMessage returns the localized plugin installation status.
func pluginInstalledMessage(language, name string) string {
	return messagef("plugin.installed", language, name)
}

// pluginRemovedMessage returns the localized plugin removal status.
func pluginRemovedMessage(language, name string) string {
	return messagef("plugin.removed", language, name)
}

// pluginValidMessage returns the localized plugin lint success status.
func pluginValidMessage(language, name, pluginVersion string) string {
	return messagef("plugin.valid", language, name, pluginVersion)
}

// pluginUpdatedMessage returns the localized plugin update status.
func pluginUpdatedMessage(language, name, currentVersion, nextVersion string) string {
	return messagef("plugin.updated", language, name, currentVersion, nextVersion)
}

// pluginCurrentMessage returns the localized already-current plugin status.
func pluginCurrentMessage(language, name string) string {
	return messagef("plugin.current", language, name)
}

// pluginUpdateAvailableMessage returns the localized plugin update listing
// status.
func pluginUpdateAvailableMessage(language, name, currentVersion, nextVersion string) string {
	return messagef("plugin.update_available", language, name, currentVersion, nextVersion)
}

// configUpdatedMessage returns the localized config key update status.
func configUpdatedMessage(language, key string) string {
	return messagef("config.updated", language, key)
}

// configUnsetMessage returns the localized config key unset status.
func configUnsetMessage(language, key string) string {
	return messagef("config.unset", language, key)
}

// configActiveProfileMessage returns the localized active-profile status.
func configActiveProfileMessage(language, name string) string {
	return messagef("config.active_profile", language, name)
}

// configProfileUpdatedMessage returns the localized profile key update status.
func configProfileUpdatedMessage(language, name, key string) string {
	return messagef("config.profile_updated", language, name, key)
}

// configProfileUnsetMessage returns the localized profile key unset status.
func configProfileUnsetMessage(language, name, key string) string {
	return messagef("config.profile_unset", language, name, key)
}

// configProfileResetMessage returns the localized profile reset status.
func configProfileResetMessage(language, name string) string {
	return messagef("config.profile_reset", language, name)
}

// configAlreadyResetMessage returns the localized no-op config reset status.
func configAlreadyResetMessage(language string) string {
	return messageText("config.already_reset", language)
}

// configResetMessage returns the localized config reset status with backup
// path.
func configResetMessage(language, backupPath string) string {
	return messagef("config.reset", language, backupPath)
}
