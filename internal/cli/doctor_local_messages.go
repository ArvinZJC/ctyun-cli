/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

// doctorLocalMessageCatalog contains localized local diagnostic rows and summaries.
var doctorLocalMessageCatalog = map[string]map[string]string{
	"doctor.local.summary":                               {"en-US": "Local diagnostics: passed %d; warnings %d; failed %d; skipped %d.", "en-GB": "Local diagnostics: passed %d; warnings %d; failed %d; skipped %d.", "zh-CN": "本地诊断：通过 %d；警告 %d；失败 %d；跳过 %d。"},
	"doctor.local.check.config_file":                     {"en-US": "Config file", "en-GB": "Config file", "zh-CN": "配置文件"},
	"doctor.local.check.profile":                         {"en-US": "Profile", "en-GB": "Profile", "zh-CN": "配置档案"},
	"doctor.local.check.credentials":                     {"en-US": "Credentials", "en-GB": "Credentials", "zh-CN": "凭据"},
	"doctor.local.check.region":                          {"en-US": "Region", "en-GB": "Region", "zh-CN": "资源池"},
	"doctor.local.check.endpoint":                        {"en-US": "Endpoint override", "en-GB": "Endpoint override", "zh-CN": "终端节点覆盖"},
	"doctor.local.check.plugin_directory":                {"en-US": "Plugin directory", "en-GB": "Plugin directory", "zh-CN": "插件目录"},
	"doctor.local.check.plugin":                          {"en-US": "Installed plugin", "en-GB": "Installed plugin", "zh-CN": "已安装插件"},
	"doctor.local.detail.config_injected":                {"en-US": "Injected config is valid", "en-GB": "Injected config is valid", "zh-CN": "注入配置有效"},
	"doctor.local.detail.config_path_unavailable":        {"en-US": "Config path is unavailable", "en-GB": "Config path is unavailable", "zh-CN": "配置路径不可用"},
	"doctor.local.detail.config_missing":                 {"en-US": "Config file is not present", "en-GB": "Config file is not present", "zh-CN": "配置文件不存在"},
	"doctor.local.detail.config_unreadable":              {"en-US": "Config file cannot be read", "en-GB": "Config file cannot be read", "zh-CN": "无法读取配置文件"},
	"doctor.local.detail.config_not_regular":             {"en-US": "Config path is not a regular file", "en-GB": "Config path is not a regular file", "zh-CN": "配置路径不是常规文件"},
	"doctor.local.detail.config_invalid":                 {"en-US": "Config content is invalid", "en-GB": "Config content is invalid", "zh-CN": "配置内容无效"},
	"doctor.local.detail.config_passed":                  {"en-US": "Config file is readable and valid", "en-GB": "Config file is readable and valid", "zh-CN": "配置文件可读且有效"},
	"doctor.local.detail.config_permissions":             {"en-US": "Stored credentials are exposed by group or other permission bits", "en-GB": "Stored credentials are exposed by group or other permission bits", "zh-CN": "组或其他用户权限暴露了已存储凭据"},
	"doctor.local.detail.dependency":                     {"en-US": "A prerequisite could not be evaluated", "en-GB": "A prerequisite could not be evaluated", "zh-CN": "无法评估前置条件"},
	"doctor.local.detail.profile_invalid":                {"en-US": "Profile selection is invalid", "en-GB": "Profile selection is invalid", "zh-CN": "配置档案选择无效"},
	"doctor.local.detail.profile_missing":                {"en-US": "No profile is configured", "en-GB": "No profile is configured", "zh-CN": "未配置档案"},
	"doctor.local.detail.profile_passed":                 {"en-US": "Profile selection is valid", "en-GB": "Profile selection is valid", "zh-CN": "配置档案选择有效"},
	"doctor.local.detail.credentials_environment":        {"en-US": "AK and SK are available from the environment", "en-GB": "AK and SK are available from the environment", "zh-CN": "AK 和 SK 可从环境变量获取"},
	"doctor.local.detail.credentials_missing":            {"en-US": "AK or SK is not configured", "en-GB": "AK or SK is not configured", "zh-CN": "AK 或 SK 未配置"},
	"doctor.local.detail.credentials_stored":             {"en-US": "AK and SK resolve from %s; environment variables avoid keeping credentials on disk", "en-GB": "AK and SK resolve from %s; environment variables avoid keeping credentials on disk", "zh-CN": "AK 和 SK 均来自%s；使用环境变量可避免将凭据保存在磁盘上"},
	"doctor.local.detail.credentials_mixed":              {"en-US": "AK and SK resolve from different sources (%s and %s); at least one credential remains stored in the config file", "en-GB": "AK and SK resolve from different sources (%s and %s); at least one credential remains stored in the config file", "zh-CN": "AK 和 SK 来自不同来源（%s、%s）；至少一个凭据仍保存在配置文件中"},
	"doctor.local.detail.region_missing":                 {"en-US": "No profile region is configured", "en-GB": "No profile region is configured", "zh-CN": "未配置档案资源池"},
	"doctor.local.detail.region_passed":                  {"en-US": "Profile region is configured", "en-GB": "Profile region is configured", "zh-CN": "已配置档案资源池"},
	"doctor.local.detail.endpoint_missing":               {"en-US": "No endpoint override is configured", "en-GB": "No endpoint override is configured", "zh-CN": "未配置终端节点覆盖"},
	"doctor.local.detail.endpoint_invalid":               {"en-US": "Endpoint override must be an absolute HTTPS URL", "en-GB": "Endpoint override must be an absolute HTTPS URL", "zh-CN": "终端节点覆盖必须是绝对 HTTPS URL"},
	"doctor.local.detail.endpoint_passed":                {"en-US": "Endpoint override syntax is valid", "en-GB": "Endpoint override syntax is valid", "zh-CN": "终端节点覆盖语法有效"},
	"doctor.local.detail.plugin_directory_missing":       {"en-US": "Installed-plugin directory is not present", "en-GB": "Installed-plugin directory is not present", "zh-CN": "已安装插件目录不存在"},
	"doctor.local.detail.plugin_directory_unreadable":    {"en-US": "Installed-plugin directory cannot be inspected", "en-GB": "Installed-plugin directory cannot be inspected", "zh-CN": "无法检查已安装插件目录"},
	"doctor.local.detail.plugin_directory_not_directory": {"en-US": "Installed-plugin path is not a directory", "en-GB": "Installed-plugin path is not a directory", "zh-CN": "已安装插件路径不是目录"},
	"doctor.local.detail.plugin_directory_passed":        {"en-US": "Installed-plugin directory is readable", "en-GB": "Installed-plugin directory is readable", "zh-CN": "已安装插件目录可读"},
	"doctor.local.detail.plugin_invalid":                 {"en-US": "Plugin metadata is invalid or incompatible", "en-GB": "Plugin metadata is invalid or incompatible", "zh-CN": "插件元数据无效或不兼容"},
	"doctor.local.detail.plugin_passed":                  {"en-US": "Plugin metadata is valid; version %s", "en-GB": "Plugin metadata is valid; version %s", "zh-CN": "插件元数据有效；版本 %s"},
}
