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
	"usage.missing_command":                      {"en-US": "Usage: ctyun <command>", "en-GB": "Usage: ctyun <command>", "zh-CN": "用法：ctyun <命令>"},
	"error.prefix":                               {"en-US": "Error: %s", "en-GB": "Error: %s", "zh-CN": "错误：%s"},
	"error.missing_command":                      {"en-US": "missing command", "en-GB": "missing command", "zh-CN": "缺少命令"},
	"error.requires_value":                       {"en-US": "%s requires a value", "en-GB": "%s requires a value", "zh-CN": "%s 需要一个值"},
	"error.requires_positive_seconds":            {"en-US": "%s requires a positive integer number of seconds", "en-GB": "%s requires a positive integer number of seconds", "zh-CN": "%s 需要正整数秒数"},
	"error.option_requires_value":                {"en-US": "%s requires a value", "en-GB": "%s requires a value", "zh-CN": "%s 需要一个值"},
	"error.missing_required_argument":            {"en-US": "missing required argument {%s}", "en-GB": "missing required argument {%s}", "zh-CN": "缺少必填参数 {%s}"},
	"error.unknown_command":                      {"en-US": "unknown command %q", "en-GB": "unknown command %q", "zh-CN": "未知命令 %q"},
	"error.missing_path_argument":                {"en-US": "missing required argument <%s>", "en-GB": "missing required argument <%s>", "zh-CN": "缺少必填参数 <%s>"},
	"error.invalid_pattern":                      {"en-US": "--%s has invalid validation pattern: %w", "en-GB": "--%s has invalid validation pattern: %w", "zh-CN": "--%s 的校验表达式无效：%w"},
	"error.pattern_mismatch":                     {"en-US": "--%s does not match %s", "en-GB": "--%s does not match %s", "zh-CN": "--%s 不匹配 %s"},
	"error.plugin_version":                       {"en-US": "plugin %s requires ctyun %s, current version is %s", "en-GB": "plugin %s requires ctyun %s, current version is %s", "zh-CN": "插件 %s 需要 ctyun %s，当前版本是 %s"},
	"error.plugin_name":                          {"en-US": "invalid plugin name %q", "en-GB": "invalid plugin name %q", "zh-CN": "无效插件名称 %q"},
	"error.plugin_manifest_missing_name":         {"en-US": "plugin manifest is missing name", "en-GB": "plugin manifest is missing name", "zh-CN": "插件清单缺少 name"},
	"error.plugin_manifest_missing_version":      {"en-US": "plugin %s manifest is missing version", "en-GB": "plugin %s manifest is missing version", "zh-CN": "插件 %s 清单缺少 version"},
	"error.plugin_invalid_version":               {"en-US": "plugin %s has invalid version %q", "en-GB": "plugin %s has invalid version %q", "zh-CN": "插件 %s 的版本 %q 无效"},
	"error.plugin_unsupported_channel":           {"en-US": "plugin %s has unsupported channel %q", "en-GB": "plugin %s has unsupported channel %q", "zh-CN": "插件 %s 不支持渠道 %q"},
	"error.plugin_unsupported_quality":           {"en-US": "plugin %s has unsupported quality %q", "en-GB": "plugin %s has unsupported quality %q", "zh-CN": "插件 %s 不支持质量 %q"},
	"error.plugin_missing_requires_ctyun":        {"en-US": "plugin %s manifest is missing requires.ctyun", "en-GB": "plugin %s manifest is missing requires.ctyun", "zh-CN": "插件 %s 清单缺少 requires.ctyun"},
	"error.plugin_missing_api_product":           {"en-US": "plugin %s manifest is missing api.product", "en-GB": "plugin %s manifest is missing api.product", "zh-CN": "插件 %s 清单缺少 api.product"},
	"error.plugin_missing_api_ctyun_product_id":  {"en-US": "plugin %s manifest is missing api.ctyun_product_id", "en-GB": "plugin %s manifest is missing api.ctyun_product_id", "zh-CN": "插件 %s 清单缺少 api.ctyun_product_id"},
	"error.plugin_invalid_api_endpoint_url":      {"en-US": "plugin %s has invalid api.endpoint_url %q", "en-GB": "plugin %s has invalid api.endpoint_url %q", "zh-CN": "插件 %s 的 api.endpoint_url %q 无效"},
	"error.deprecation_status":                   {"en-US": "unsupported deprecation status %q", "en-GB": "unsupported deprecation status %q", "zh-CN": "不支持的弃用状态 %q"},
	"error.deprecation_replacement_kind":         {"en-US": "unsupported deprecation replacement kind %q", "en-GB": "unsupported deprecation replacement kind %q", "zh-CN": "不支持的弃用替代类型 %q"},
	"error.plugin_install_name":                  {"en-US": "plugin install requires a plugin name", "en-GB": "plugin install requires a plugin name", "zh-CN": "plugin install 需要插件名称"},
	"error.plugin_remove_name":                   {"en-US": "plugin remove requires a plugin name", "en-GB": "plugin remove requires a plugin name", "zh-CN": "plugin remove 需要插件名称"},
	"error.plugin_lint_dev_only":                 {"en-US": "plugin lint is only available in development builds", "en-GB": "plugin lint is only available in development builds", "zh-CN": "plugin lint 仅可用于开发构建"},
	"error.plugin_bundled_update_target":         {"en-US": "plugin update/upgrade --bundled requires a plugin name or --all", "en-GB": "plugin update/upgrade --bundled requires a plugin name or --all", "zh-CN": "plugin update/upgrade --bundled 需要插件名称或 --all"},
	"error.plugin_update_target":                 {"en-US": "plugin update/upgrade requires a plugin name or --all", "en-GB": "plugin update/upgrade requires a plugin name or --all", "zh-CN": "plugin update/upgrade 需要插件名称或 --all"},
	"error.hosted_plugin_dev":                    {"en-US": "hosted plugin metadata is unavailable for development builds; use --bundled", "en-GB": "hosted plugin metadata is unavailable for development builds; use --bundled", "zh-CN": "开发构建不可使用托管插件元数据；请使用 --bundled"},
	"error.plugin_install_one_name":              {"en-US": "plugin install accepts one plugin name", "en-GB": "plugin install accepts one plugin name", "zh-CN": "plugin install 只接受一个插件名称"},
	"error.plugin_install_all_or_names":          {"en-US": "plugin install accepts either --all or plugin names", "en-GB": "plugin install accepts either --all or plugin names", "zh-CN": "plugin install 只能使用 --all 或插件名称"},
	"error.plugin_install_source_choice":         {"en-US": "plugin install accepts either --bundled or --source", "en-GB": "plugin install accepts either --bundled or --source", "zh-CN": "plugin install 只能使用 --bundled 或 --source 其中之一"},
	"error.plugin_reinstall_target":              {"en-US": "plugin reinstall requires plugin names or --all", "en-GB": "plugin reinstall requires plugin names or --all", "zh-CN": "plugin reinstall 需要插件名称或 --all"},
	"error.plugin_reinstall_all_or_names":        {"en-US": "plugin reinstall accepts either --all or plugin names", "en-GB": "plugin reinstall accepts either --all or plugin names", "zh-CN": "plugin reinstall 只能使用 --all 或插件名称"},
	"error.plugin_reinstall_source_choice":       {"en-US": "plugin reinstall accepts either --bundled or --source", "en-GB": "plugin reinstall accepts either --bundled or --source", "zh-CN": "plugin reinstall 只能使用 --bundled 或 --source 其中之一"},
	"error.plugin_not_installed":                 {"en-US": "plugin %s is not installed", "en-GB": "plugin %s is not installed", "zh-CN": "插件 %s 未安装"},
	"error.operation_batch_failed":               {"en-US": "%d of %d operations failed", "en-GB": "%d of %d operations failed", "zh-CN": "%d/%d 个操作失败"},
	"error.plugin_search_one_query":              {"en-US": "plugin search accepts one query", "en-GB": "plugin search accepts one query", "zh-CN": "plugin search 只接受一个查询词"},
	"error.plugin_search_source_choice":          {"en-US": "plugin search accepts either --bundled or --source", "en-GB": "plugin search accepts either --bundled or --source", "zh-CN": "plugin search 只能使用 --bundled 或 --source 其中之一"},
	"error.plugin_list_available_updates":        {"en-US": "plugin list accepts either --available or --updates", "en-GB": "plugin list accepts either --available or --updates", "zh-CN": "plugin list 只能使用 --available 或 --updates 其中之一"},
	"error.plugin_list_source_choice":            {"en-US": "plugin list accepts either --bundled or --source", "en-GB": "plugin list accepts either --bundled or --source", "zh-CN": "plugin list 只能使用 --bundled 或 --source 其中之一"},
	"error.plugin_remove_all_or_names":           {"en-US": "plugin remove accepts either --all or plugin names", "en-GB": "plugin remove accepts either --all or plugin names", "zh-CN": "plugin remove 只能使用 --all 或插件名称"},
	"error.plugin_remove_confirm":                {"en-US": "plugin remove confirmation required", "en-GB": "plugin remove confirmation required", "zh-CN": "plugin remove 需要确认"},
	"error.confirmation_cancelled":               {"en-US": "operation cancelled", "en-GB": "operation cancelled", "zh-CN": "操作已取消"},
	"error.plugin_update_one_name":               {"en-US": "plugin update accepts one plugin name", "en-GB": "plugin update accepts one plugin name", "zh-CN": "plugin update 只接受一个插件名称"},
	"error.plugin_update_all_or_one":             {"en-US": "plugin update accepts either --all or one plugin name", "en-GB": "plugin update accepts either --all or one plugin name", "zh-CN": "plugin update 只能使用 --all 或一个插件名称"},
	"error.plugin_update_source_choice":          {"en-US": "plugin update accepts either --bundled or --source", "en-GB": "plugin update accepts either --bundled or --source", "zh-CN": "plugin update 只能使用 --bundled 或 --source 其中之一"},
	"error.plugin_source_empty":                  {"en-US": "plugin source is empty", "en-GB": "plugin source is empty", "zh-CN": "插件源为空"},
	"error.unsupported_plugin_source":            {"en-US": "unsupported plugin source %T", "en-GB": "unsupported plugin source %T", "zh-CN": "不支持插件源类型 %T"},
	"error.source_requires_value":                {"en-US": "--source requires a value", "en-GB": "--source requires a value", "zh-CN": "--source 需要一个值"},
	"error.unsupported_source":                   {"en-US": "unsupported %s source %q; use auto, github, or gitee", "en-GB": "unsupported %s source %q; use auto, github, or gitee", "zh-CN": "不支持 %s 源 %q；请使用 auto、github 或 gitee"},
	"error.channel_requires_value":               {"en-US": "--channel requires a value", "en-GB": "--channel requires a value", "zh-CN": "--channel 需要一个值"},
	"error.unsupported_channel":                  {"en-US": "unsupported channel %q; use stable, beta, or alpha", "en-GB": "unsupported channel %q; use stable, beta, or alpha", "zh-CN": "不支持通道 %q；请使用 stable、beta 或 alpha"},
	"error.unsupported_plugin_discovery_channel": {"en-US": "unsupported channel %q; use stable, beta, alpha, or all", "en-GB": "unsupported channel %q; use stable, beta, alpha, or all", "zh-CN": "不支持通道 %q；请使用 stable、beta、alpha 或 all"},
	"error.plugin_list_all_channel_available":    {"en-US": "plugin list --channel all requires --available", "en-GB": "plugin list --channel all requires --available", "zh-CN": "plugin list --channel all 需要 --available"},
	"error.bundled_dev_only":                     {"en-US": "--bundled is only available in development builds", "en-GB": "--bundled is only available in development builds", "zh-CN": "--bundled 仅可用于开发构建"},
	"error.unsupported_output":                   {"en-US": "unsupported output %q", "en-GB": "unsupported output %q", "zh-CN": "不支持输出格式 %q"},
	"error.unknown_filter_key":                   {"en-US": "unknown filter key %q; use a visible column or field label or stable key", "en-GB": "unknown filter key %q; use a visible column or field label or stable key", "zh-CN": "未知筛选键 %q；请使用可见列名/字段名或稳定键"},
	"error.unknown_sort_key":                     {"en-US": "unknown sort key %q; use a visible column or field label or stable key", "en-GB": "unknown sort key %q; use a visible column or field label or stable key", "zh-CN": "未知排序键 %q；请使用可见列名/字段名或稳定键"},
	"error.unknown_waiter":                       {"en-US": "unknown waiter %q", "en-GB": "unknown waiter %q", "zh-CN": "未知等待器 %q"},
	"error.plugin_root_not_directory":            {"en-US": "plugin root %s is not a directory", "en-GB": "plugin root %s is not a directory", "zh-CN": "插件根路径 %s 不是目录"},
	"error.plugin_not_found_registry":            {"en-US": "plugin %s not found in registry", "en-GB": "plugin %s not found in registry", "zh-CN": "注册表中未找到插件 %s"},
	"error.unsupported_shell":                    {"en-US": "unsupported shell %q", "en-GB": "unsupported shell %q", "zh-CN": "不支持 shell %q"},
	"error.config_path_unavailable":              {"en-US": "config path is unavailable", "en-GB": "config path is unavailable", "zh-CN": "配置路径不可用"},
	"error.profile_not_found":                    {"en-US": "profile %q not found", "en-GB": "profile %q not found", "zh-CN": "未找到配置档案 %q"},
	"error.unsupported_secret_key":               {"en-US": "unsupported secret key %q", "en-GB": "unsupported secret key %q", "zh-CN": "不支持密钥 %q"},
	"error.config_profile_reset_confirm":         {"en-US": "config profile reset confirmation required", "en-GB": "config profile reset confirmation required", "zh-CN": "config profile reset 需要确认"},
	"error.config_reset_confirm":                 {"en-US": "config reset confirmation required", "en-GB": "config reset confirmation required", "zh-CN": "config reset 需要确认"},
	"error.active_profile_missing":               {"en-US": "active_profile %q does not exist", "en-GB": "active_profile %q does not exist", "zh-CN": "active_profile %q 不存在"},
	"error.profile_name_empty":                   {"en-US": "profile name cannot be empty", "en-GB": "profile name cannot be empty", "zh-CN": "配置档案名称不能为空"},
	"error.profile_timeout_negative":             {"en-US": "profile %q timeout_seconds cannot be negative", "en-GB": "profile %q timeout_seconds cannot be negative", "zh-CN": "配置档案 %q 的 timeout_seconds 不能为负数"},
	"error.profile_language_unsupported":         {"en-US": "profile %q language %q is not supported", "en-GB": "profile %q language %q is not supported", "zh-CN": "配置档案 %q 不支持语言 %q"},
	"error.unsupported_global_config_key":        {"en-US": "unsupported global config key %q", "en-GB": "unsupported global config key %q", "zh-CN": "不支持全局配置键 %q"},
	"error.unsupported_profile_config_key":       {"en-US": "unsupported profile config key %q", "en-GB": "unsupported profile config key %q", "zh-CN": "不支持配置档案键 %q"},
	"error.missing_credentials":                  {"en-US": "missing CTyun credentials: set CTYUN_AK and CTYUN_SK or config ak/sk", "en-GB": "missing CTyun credentials: set CTYUN_AK and CTYUN_SK or config ak/sk", "zh-CN": "缺少天翼云凭据：请设置 CTYUN_AK 和 CTYUN_SK，或配置 ak/sk"},
	"error.config_unsupported_secret":            {"en-US": "config contains unsupported secret material; use only ak/sk credential fields", "en-GB": "config contains unsupported secret material; use only ak/sk credential fields", "zh-CN": "配置包含不支持的密钥材料；请仅使用 ak/sk 凭据字段"},
	"error.config_multiple_profiles":             {"en-US": "config with multiple profiles requires active_profile or --profile", "en-GB": "config with multiple profiles requires active_profile or --profile", "zh-CN": "配置包含多个配置档案时需要 active_profile 或 --profile"},
	"error.parse_config":                         {"en-US": "parse config: %s", "en-GB": "parse config: %s", "zh-CN": "解析配置失败：%s"},
	"error.parse_timeout_seconds":                {"en-US": "parse timeout_seconds: %s", "en-GB": "parse timeout_seconds: %s", "zh-CN": "解析 timeout_seconds 失败：%s"},
	"error.parse_warn_config_credentials":        {"en-US": "parse warn_config_credentials: %s", "en-GB": "parse warn_config_credentials: %s", "zh-CN": "解析 warn_config_credentials 失败：%s"},
	"error.parse_warn_deprecated":                {"en-US": "parse warn_deprecated: %s", "en-GB": "parse warn_deprecated: %s", "zh-CN": "解析 warn_deprecated 失败：%s"},
	"error.validate_config":                      {"en-US": "validate config: %s", "en-GB": "validate config: %s", "zh-CN": "校验配置失败：%s"},
	"error.parse_response_json":                  {"en-US": "parse response JSON: %s", "en-GB": "parse response JSON: %s", "zh-CN": "解析响应 JSON 失败：%s"},
	"error.api_http":                             {"en-US": "ctyun API returned HTTP %s: %s", "en-GB": "ctyun API returned HTTP %s: %s", "zh-CN": "ctyun API 返回 HTTP %s：%s"},
	"error.api_status":                           {"en-US": "ctyun API returned statusCode %s: %s", "en-GB": "ctyun API returned statusCode %s: %s", "zh-CN": "ctyun API 返回 statusCode %s：%s"},
	"error.api_request_failed":                   {"en-US": "ctyun API request failed", "en-GB": "ctyun API request failed", "zh-CN": "ctyun API 请求失败"},
	"error.api_hint":                             {"en-US": "If this looks like a ctyun CLI issue, open https://github.com/ArvinZJC/ctyun-cli/issues or https://gitee.com/ArvinZJC/ctyun-cli/issues. If this looks like a CTyun service issue, submit a CTyun work order.", "en-GB": "If this looks like a ctyun CLI issue, open https://github.com/ArvinZJC/ctyun-cli/issues or https://gitee.com/ArvinZJC/ctyun-cli/issues. If this looks like a CTyun service issue, submit a CTyun work order.", "zh-CN": "如果你认为这是 ctyun CLI 问题，请在 https://github.com/ArvinZJC/ctyun-cli/issues 提交 GitHub Issue，或在 https://gitee.com/ArvinZJC/ctyun-cli/issues 提交 Gitee Issue；如果你认为这是天翼云服务问题，请提交天翼云工单。"},
	"error.parse_registry_index":                 {"en-US": "parse registry index: %s", "en-GB": "parse registry index: %s", "zh-CN": "解析插件源索引失败：%s"},
	"error.registry_missing_name":                {"en-US": "%s is missing name", "en-GB": "%s is missing name", "zh-CN": "%s 缺少名称"},
	"error.registry_invalid_plugin_name":         {"en-US": "%s has invalid plugin name %q", "en-GB": "%s has invalid plugin name %q", "zh-CN": "%s 的插件名称 %q 无效"},
	"error.registry_missing_version":             {"en-US": "%s is missing version", "en-GB": "%s is missing version", "zh-CN": "%s 缺少版本"},
	"error.registry_invalid_version":             {"en-US": "%s has invalid version %q", "en-GB": "%s has invalid version %q", "zh-CN": "%s 版本 %q 无效"},
	"error.registry_unsupported_channel":         {"en-US": "%s has unsupported channel %q", "en-GB": "%s has unsupported channel %q", "zh-CN": "%s 不支持渠道 %q"},
	"error.registry_unsupported_quality":         {"en-US": "%s has unsupported quality %q", "en-GB": "%s has unsupported quality %q", "zh-CN": "%s 不支持质量 %q"},
	"error.registry_missing_url":                 {"en-US": "%s is missing url", "en-GB": "%s is missing url", "zh-CN": "%s 缺少 url"},
	"error.registry_invalid_artifact_url":        {"en-US": "%s has invalid artifact url %q", "en-GB": "%s has invalid artifact url %q", "zh-CN": "%s 产物 URL %q 无效"},
	"error.release_not_found":                    {"en-US": "no ctyun release found for %s/%s on channel %s", "en-GB": "no ctyun release found for %s/%s on channel %s", "zh-CN": "未找到适用于 %s/%s 的 %s 渠道 ctyun 版本"},
	"error.parse_release_index":                  {"en-US": "parse release index: %s", "en-GB": "parse release index: %s", "zh-CN": "解析发布索引失败：%s"},
	"error.unsupported_release_schema":           {"en-US": "unsupported release index schema %v", "en-GB": "unsupported release index schema %v", "zh-CN": "不支持发布索引 schema %v"},
	"error.release_missing_version":              {"en-US": "%s is missing version", "en-GB": "%s is missing version", "zh-CN": "%s 缺少版本"},
	"error.release_invalid_version":              {"en-US": "%s has invalid version %q", "en-GB": "%s has invalid version %q", "zh-CN": "%s 版本 %q 无效"},
	"error.release_unsupported_channel":          {"en-US": "%s has unsupported channel %q", "en-GB": "%s has unsupported channel %q", "zh-CN": "%s 不支持渠道 %q"},
	"error.release_no_artifacts":                 {"en-US": "%s has no artifacts", "en-GB": "%s has no artifacts", "zh-CN": "%s 没有产物"},
	"error.release_missing_os":                   {"en-US": "%s is missing os", "en-GB": "%s is missing os", "zh-CN": "%s 缺少 os"},
	"error.release_missing_arch":                 {"en-US": "%s is missing arch", "en-GB": "%s is missing arch", "zh-CN": "%s 缺少 arch"},
	"error.release_missing_url":                  {"en-US": "%s is missing url", "en-GB": "%s is missing url", "zh-CN": "%s 缺少 url"},
	"error.release_invalid_artifact_url":         {"en-US": "%s has invalid artifact url %q", "en-GB": "%s has invalid artifact url %q", "zh-CN": "%s 产物 URL %q 无效"},
	"error.release_invalid_sha256":               {"en-US": "%s has invalid sha256", "en-GB": "%s has invalid sha256", "zh-CN": "%s 的 sha256 无效"},
	"error.confirmation_required":                {"en-US": "%s confirmation required", "en-GB": "%s confirmation required", "zh-CN": "%s 需要确认"},
	"error.read_index":                           {"en-US": "read %s index: %s", "en-GB": "read %s index: %s", "zh-CN": "读取 %s 索引失败：%s"},
	"error.read_index_signature":                 {"en-US": "read %s index signature: %s", "en-GB": "read %s index signature: %s", "zh-CN": "读取 %s 索引签名失败：%s"},
	"error.index_signature":                      {"en-US": "%s index signature: %s", "en-GB": "%s index signature: %s", "zh-CN": "%s 索引签名无效：%s"},
	"error.index_public_key_required":            {"en-US": "%s index requires a trusted public key", "en-GB": "%s index requires a trusted public key", "zh-CN": "%s 索引需要受信任的公钥"},
	"error.decode_public_key":                    {"en-US": "decode %s public key: %s", "en-GB": "decode %s public key: %s", "zh-CN": "解析 %s 公钥失败：%s"},
	"error.public_key_length":                    {"en-US": "%s public key has length %v, want %v", "en-GB": "%s public key has length %v, want %v", "zh-CN": "%s 公钥长度为 %v，需要 %v"},
	"error.decode_signature":                     {"en-US": "decode %s signature: %s", "en-GB": "decode %s signature: %s", "zh-CN": "解析 %s 签名失败：%s"},
	"error.signature_failed":                     {"en-US": "%s index signature verification failed", "en-GB": "%s index signature verification failed", "zh-CN": "%s 索引签名验证失败"},
	"error.http_get_status":                      {"en-US": "GET %s returned %s", "en-GB": "GET %s returned %s", "zh-CN": "GET %s 返回 %s"},
	"error.artifact_requires_sha256":             {"en-US": "%s requires sha256", "en-GB": "%s requires sha256", "zh-CN": "%s 需要 sha256"},
	"error.sha256_mismatch":                      {"en-US": "sha256 mismatch for %s: got %s, want %s", "en-GB": "sha256 mismatch for %s: got %s, want %s", "zh-CN": "%s 的 sha256 不匹配：实际 %s，需要 %s"},
	"error.unsupported_table_style":              {"en-US": "unsupported table style %q", "en-GB": "unsupported table style %q", "zh-CN": "不支持表格样式 %q"},
	"error.invalid_filter_syntax":                {"en-US": "invalid filter %q: use key=value or key!=value", "en-GB": "invalid filter %q: use key=value or key!=value", "zh-CN": "筛选表达式 %q 无效：请使用 key=value 或 key!=value"},
	"error.invalid_filter_missing_key":           {"en-US": "invalid filter %q: missing key", "en-GB": "invalid filter %q: missing key", "zh-CN": "筛选表达式 %q 无效：缺少键"},
	"error.invalid_sort_missing_key":             {"en-US": "invalid sort %q: missing key", "en-GB": "invalid sort %q: missing key", "zh-CN": "排序表达式 %q 无效：缺少键"},
	"error.unknown_column":                       {"en-US": "unknown output selector %q; use a visible column or field label or stable key", "en-GB": "unknown output selector %q; use a visible column or field label or stable key", "zh-CN": "未知输出选择器 %q；请使用可见列名/字段名或稳定键"},
	"error.path_cannot_read":                     {"en-US": "path %q cannot read %q", "en-GB": "path %q cannot read %q", "zh-CN": "路径 %q 无法读取 %q"},
	"error.path_missing":                         {"en-US": "path %q is missing %q", "en-GB": "path %q is missing %q", "zh-CN": "路径 %q 缺少 %q"},
	"error.archive_path_escapes_destination":     {"en-US": "archive path escapes destination: %s", "en-GB": "archive path escapes destination: %s", "zh-CN": "归档路径会逃逸目标目录：%s"},
	"error.unsupported_archive_entry":            {"en-US": "unsupported archive entry %s", "en-GB": "unsupported archive entry %s", "zh-CN": "不支持的归档条目 %s"},
	"error.archive_missing_binary":               {"en-US": "archive does not contain %s", "en-GB": "archive does not contain %s", "zh-CN": "归档中不包含 %s"},
	"error.current_executable_required":          {"en-US": "current executable path is required", "en-GB": "current executable path is required", "zh-CN": "需要当前可执行文件路径"},
	"error.archive_path_required":                {"en-US": "archive path is required", "en-GB": "archive path is required", "zh-CN": "需要归档路径"},
	"error.archive_multiple_plugin_roots":        {"en-US": "archive contains multiple plugin roots", "en-GB": "archive contains multiple plugin roots", "zh-CN": "归档包含多个插件根目录"},
	"error.archive_missing_plugin_json":          {"en-US": "archive does not contain plugin.json", "en-GB": "archive does not contain plugin.json", "zh-CN": "归档不包含 plugin.json"},
	"error.unsupported_bundle_entry":             {"en-US": "unsupported bundle entry %s", "en-GB": "unsupported bundle entry %s", "zh-CN": "不支持的插件包条目 %s"},
	"error.duplicate_command_id":                 {"en-US": "duplicate command id %s", "en-GB": "duplicate command id %s", "zh-CN": "重复命令 ID %s"},
	"error.duplicate_command_path":               {"en-US": "duplicate command path %s", "en-GB": "duplicate command path %s", "zh-CN": "重复命令路径 %s"},
	"error.command_missing_table_ref":            {"en-US": "command %s references missing table %s", "en-GB": "command %s references missing table %s", "zh-CN": "命令 %s 引用了缺失表格 %s"},
	"error.command_missing_operation_ref":        {"en-US": "plugin command references a missing operation", "en-GB": "plugin command references a missing operation", "zh-CN": "插件命令引用了缺失操作"},
	"error.command_missing_id":                   {"en-US": "command is missing id", "en-GB": "command is missing id", "zh-CN": "命令缺少 id"},
	"error.command_missing_path":                 {"en-US": "command %s is missing path", "en-GB": "command %s is missing path", "zh-CN": "命令 %s 缺少 path"},
	"error.command_invalid_path_segment":         {"en-US": "command %s has invalid path segment %q", "en-GB": "command %s has invalid path segment %q", "zh-CN": "命令 %s 的路径片段 %q 无效"},
	"error.command_missing_table":                {"en-US": "command %s is missing table", "en-GB": "command %s is missing table", "zh-CN": "命令 %s 缺少 table"},
	"error.command_invalid_fixture_response":     {"en-US": "command %s has invalid fixture_response %q", "en-GB": "command %s has invalid fixture_response %q", "zh-CN": "命令 %s 的 fixture_response %q 无效"},
	"error.command_parameter_missing_name":       {"en-US": "command %s has parameter missing name", "en-GB": "command %s has parameter missing name", "zh-CN": "命令 %s 有参数缺少 name"},
	"error.command_parameter_missing_flag":       {"en-US": "command %s parameter %s is missing flag", "en-GB": "command %s parameter %s is missing flag", "zh-CN": "命令 %s 的参数 %s 缺少 flag"},
	"error.command_parameter_missing_target":     {"en-US": "command %s parameter %s is missing target", "en-GB": "command %s parameter %s is missing target", "zh-CN": "命令 %s 的参数 %s 缺少 target"},
	"error.command_duplicate_parameter_flag":     {"en-US": "command %s has duplicate parameter flag %s", "en-GB": "command %s has duplicate parameter flag %s", "zh-CN": "命令 %s 有重复参数 flag %s"},
	"error.command_parameter_invalid_pattern":    {"en-US": "command %s parameter %s has invalid pattern: %s", "en-GB": "command %s parameter %s has invalid pattern: %s", "zh-CN": "命令 %s 的参数 %s 有无效 pattern：%s"},
	"error.operation_missing_id":                 {"en-US": "operation is missing id", "en-GB": "operation is missing id", "zh-CN": "操作缺少 id"},
	"error.operation_missing_method":             {"en-US": "operation %s is missing method", "en-GB": "operation %s is missing method", "zh-CN": "操作 %s 缺少 method"},
	"error.operation_unsupported_method":         {"en-US": "operation %s has unsupported method %q", "en-GB": "operation %s has unsupported method %q", "zh-CN": "操作 %s 不支持 method %q"},
	"error.operation_missing_path":               {"en-US": "operation %s is missing path", "en-GB": "operation %s is missing path", "zh-CN": "操作 %s 缺少 path"},
	"error.operation_path_must_start_with_slash": {"en-US": "operation %s path must start with /", "en-GB": "operation %s path must start with /", "zh-CN": "操作 %s 的 path 必须以 / 开头"},
	"error.operation_invalid_path":               {"en-US": "operation %s has invalid path %q", "en-GB": "operation %s has invalid path %q", "zh-CN": "操作 %s 的 path %q 无效"},
	"error.table_missing_id":                     {"en-US": "table is missing id", "en-GB": "table is missing id", "zh-CN": "表格缺少 id"},
	"error.table_missing_row_path":               {"en-US": "table %s is missing row_path", "en-GB": "table %s is missing row_path", "zh-CN": "表格 %s 缺少 row_path"},
	"error.table_missing_columns":                {"en-US": "table %s is missing columns", "en-GB": "table %s is missing columns", "zh-CN": "表格 %s 缺少 columns"},
	"error.table_invalid_layout":                 {"en-US": "table %s has invalid layout %q", "en-GB": "table %s has invalid layout %q", "zh-CN": "表格 %s 的 layout %q 无效"},
	"error.table_column_missing_key":             {"en-US": "table %s has column missing key", "en-GB": "table %s has column missing key", "zh-CN": "表格 %s 有列缺少 key"},
	"error.table_column_missing_path":            {"en-US": "table %s column %s is missing path", "en-GB": "table %s column %s is missing path", "zh-CN": "表格 %s 的列 %s 缺少 path"},
	"error.table_duplicate_column_key":           {"en-US": "table %s has duplicate column key %s", "en-GB": "table %s has duplicate column key %s", "zh-CN": "表格 %s 有重复列 key %s"},
	"error.table_column_missing_label":           {"en-US": "table %s column %s is missing %s label", "en-GB": "table %s column %s is missing %s label", "zh-CN": "表格 %s 的列 %s 缺少 %s 标签"},
	"error.waiter_unsupported_timeout_seconds":   {"en-US": "waiter %s uses unsupported timeout_seconds; use max_attempts and interval_seconds for polling, and profile timeout_seconds or --timeout for HTTP request timeouts", "en-GB": "waiter %s uses unsupported timeout_seconds; use max_attempts and interval_seconds for polling, and profile timeout_seconds or --timeout for HTTP request timeouts", "zh-CN": "等待器 %s 使用了不支持的 timeout_seconds；轮询请使用 max_attempts 和 interval_seconds，HTTP 请求超时请使用配置档案 timeout_seconds 或 --timeout"},
	"error.waiter_negative_max_attempts":         {"en-US": "waiter %s has negative max_attempts", "en-GB": "waiter %s has negative max_attempts", "zh-CN": "等待器 %s 的 max_attempts 为负数"},
	"error.waiter_negative_interval_seconds":     {"en-US": "waiter %s has negative interval_seconds", "en-GB": "waiter %s has negative interval_seconds", "zh-CN": "等待器 %s 的 interval_seconds 为负数"},
	"error.http_response_too_large":              {"en-US": "HTTP response from %s exceeds %d bytes", "en-GB": "HTTP response from %s exceeds %d bytes", "zh-CN": "%s 的 HTTP 响应超过 %d 字节"},
	"error.doctor_plugin_metadata":               {"en-US": "available plugin metadata could not be inspected safely", "en-GB": "available plugin metadata could not be inspected safely", "zh-CN": "无法安全检查可用插件元数据"},
	"error.doctor_invalid_endpoint":              {"en-US": "invalid HTTPS diagnostic endpoint %s", "en-GB": "invalid HTTPS diagnostic endpoint %s", "zh-CN": "无效的 HTTPS 诊断端点 %s"},
	"error.doctor_proxy":                         {"en-US": "resolve diagnostic proxy: %s", "en-GB": "resolve diagnostic proxy: %s", "zh-CN": "解析诊断代理失败：%s"},
	"error.doctor_dependency_cycle":              {"en-US": "network diagnostic check dependencies contain a cycle", "en-GB": "network diagnostic check dependencies contain a cycle", "zh-CN": "网络诊断检查依赖存在循环"},
	"error.parse_json_file":                      {"en-US": "parse %s: %s", "en-GB": "parse %s: %s", "zh-CN": "解析 %s 失败：%s"},
	"error.command_missing_fixture_response":     {"en-US": "plugin command has no fixture response for offline mode", "en-GB": "plugin command has no fixture response for offline mode", "zh-CN": "插件命令在离线模式下没有 fixture 响应"},
	"error.parse_fixture_response":               {"en-US": "parse fixture response: %s", "en-GB": "parse fixture response: %s", "zh-CN": "解析 fixture 响应失败：%s"},
	"error.command_missing_live_endpoint":        {"en-US": "plugin command requires plugin api.endpoint_url or profile endpoint_url for live API execution", "en-GB": "plugin command requires plugin api.endpoint_url or profile endpoint_url for live API execution", "zh-CN": "插件命令执行实时 API 需要插件 api.endpoint_url 或配置档案 endpoint_url"},
	"error.missing_profile_region":               {"en-US": "plugin command requires region from command input or profile", "en-GB": "plugin command requires region from command input or profile", "zh-CN": "插件命令需要通过命令输入或配置档案提供 region"},
	"error.row_path_not_array":                   {"en-US": "row path %q is not an array", "en-GB": "row path %q is not an array", "zh-CN": "行路径 %q 不是数组"},
	"error.row_path_non_object":                  {"en-US": "row path %q contains a non-object row", "en-GB": "row path %q contains a non-object row", "zh-CN": "行路径 %q 包含非对象行"},
	"error.unexpected_argument":                  {"en-US": "unexpected argument %q", "en-GB": "unexpected argument %q", "zh-CN": "不支持参数 %q"},
	"error.flag_requires_value":                  {"en-US": "--%s requires a value", "en-GB": "--%s requires a value", "zh-CN": "--%s 需要一个值"},
	"error.unknown_option":                       {"en-US": "unknown option %q", "en-GB": "unknown option %q", "zh-CN": "未知选项 %q"},
	"error.missing_required_flag":                {"en-US": "missing required option --%s", "en-GB": "missing required option --%s", "zh-CN": "缺少必填选项 --%s"},
	"error.missing_conditional_flag":             {"en-US": "requires --%s when --%s is %s", "en-GB": "requires --%s when --%s is %s", "zh-CN": "当 --%[2]s 为 %[3]s 时需要 --%[1]s"},
	"error.missing_conditional_any":              {"en-US": "requires one of %s when --%s is %s", "en-GB": "requires one of %s when --%s is %s", "zh-CN": "当 --%[2]s 为 %[3]s 时需要以下选项之一：%[1]s"},
	"error.allowed_values":                       {"en-US": "--%s must be one of %s", "en-GB": "--%s must be one of %s", "zh-CN": "--%s 必须是以下值之一：%s"},
	"doctor.column.check":                        {"en-US": "Check", "en-GB": "Check", "zh-CN": "检查"},
	"doctor.column.target":                       {"en-US": "Target", "en-GB": "Target", "zh-CN": "目标"},
	"doctor.column.status":                       {"en-US": "Status", "en-GB": "Status", "zh-CN": "状态"},
	"doctor.column.duration":                     {"en-US": "Duration", "en-GB": "Duration", "zh-CN": "耗时"},
	"doctor.column.detail":                       {"en-US": "Detail", "en-GB": "Detail", "zh-CN": "详情"},
	"doctor.status.passed":                       {"en-US": "Passed", "en-GB": "Passed", "zh-CN": "通过"},
	"doctor.status.warning":                      {"en-US": "Warning", "en-GB": "Warning", "zh-CN": "警告"},
	"doctor.status.failed":                       {"en-US": "Failed", "en-GB": "Failed", "zh-CN": "失败"},
	"doctor.status.skipped":                      {"en-US": "Skipped", "en-GB": "Skipped", "zh-CN": "已跳过"},
	"doctor.subject.core":                        {"en-US": "Core", "en-GB": "Core", "zh-CN": "核心"},
	"doctor.subject.plugin":                      {"en-US": "Plugin", "en-GB": "Plugin", "zh-CN": "插件"},
	"doctor.subject.ctyun":                       {"en-US": "CTyun", "en-GB": "CTyun", "zh-CN": "天翼云"},
	"doctor.check.configuration":                 {"en-US": "%s configuration", "en-GB": "%s configuration", "zh-CN": "%s配置"},
	"doctor.check.route":                         {"en-US": "%s route", "en-GB": "%s route", "zh-CN": "%s路由"},
	"doctor.check.https":                         {"en-US": "%s HTTPS", "en-GB": "%s HTTPS", "zh-CN": "%s HTTPS"},
	"doctor.check.index":                         {"en-US": "%s index", "en-GB": "%s index", "zh-CN": "%s索引"},
	"doctor.check.signature":                     {"en-US": "%s signature", "en-GB": "%s signature", "zh-CN": "%s签名"},
	"doctor.check.verification":                  {"en-US": "%s verification", "en-GB": "%s verification", "zh-CN": "%s验证"},
	"doctor.check.capability":                    {"en-US": "%s capability", "en-GB": "%s capability", "zh-CN": "%s能力"},
	"doctor.check.core_source":                   {"en-US": "Core update source", "en-GB": "Core update source", "zh-CN": "核心更新源"},
	"doctor.check.plugin_source":                 {"en-US": "Plugin registry", "en-GB": "Plugin registry", "zh-CN": "插件注册表"},
	"doctor.check.endpoint":                      {"en-US": "CTyun API endpoint", "en-GB": "CTyun API endpoint", "zh-CN": "天翼云 API 端点"},
	"doctor.detail.passed":                       {"en-US": "Available", "en-GB": "Available", "zh-CN": "可用"},
	"doctor.detail.configuration":                {"en-US": "Configuration is invalid", "en-GB": "Configuration is invalid", "zh-CN": "配置无效"},
	"doctor.detail.dns":                          {"en-US": "DNS resolution failed", "en-GB": "DNS resolution failed", "zh-CN": "DNS 解析失败"},
	"doctor.detail.proxy":                        {"en-US": "Proxy route failed", "en-GB": "Proxy route failed", "zh-CN": "代理路由失败"},
	"doctor.detail.connection":                   {"en-US": "Connection failed", "en-GB": "Connection failed", "zh-CN": "连接失败"},
	"doctor.detail.timeout":                      {"en-US": "Timed out", "en-GB": "Timed out", "zh-CN": "超时"},
	"doctor.detail.tls":                          {"en-US": "TLS negotiation failed", "en-GB": "TLS negotiation failed", "zh-CN": "TLS 协商失败"},
	"doctor.detail.http":                         {"en-US": "HTTP request failed", "en-GB": "HTTP request failed", "zh-CN": "HTTP 请求失败"},
	"doctor.detail.redirect":                     {"en-US": "Redirect is unsafe", "en-GB": "Redirect is unsafe", "zh-CN": "重定向不安全"},
	"doctor.detail.index":                        {"en-US": "Index retrieval failed", "en-GB": "Index retrieval failed", "zh-CN": "索引获取失败"},
	"doctor.detail.signature":                    {"en-US": "Signature verification failed", "en-GB": "Signature verification failed", "zh-CN": "签名验证失败"},
	"doctor.detail.public_key":                   {"en-US": "Trusted public key is unavailable or invalid", "en-GB": "Trusted public key is unavailable or invalid", "zh-CN": "可信公钥不可用或无效"},
	"doctor.detail.cancelled":                    {"en-US": "Cancelled", "en-GB": "Cancelled", "zh-CN": "已取消"},
	"doctor.detail.dependency":                   {"en-US": "Skipped because a prerequisite failed", "en-GB": "Skipped because a prerequisite failed", "zh-CN": "因前置检查失败而跳过"},
	"doctor.detail.capability_failed":            {"en-US": "No usable alternative", "en-GB": "No usable alternative", "zh-CN": "没有可用替代项"},
	"doctor.detail.capability_degraded":          {"en-US": "Available through an alternative", "en-GB": "Available through an alternative", "zh-CN": "可通过替代项使用"},
	"doctor.detail.source_passed":                {"en-US": "Signed index verified", "en-GB": "Signed index verified", "zh-CN": "签名索引验证通过"},
	"doctor.detail.endpoint_passed":              {"en-US": "HTTPS reachable", "en-GB": "HTTPS reachable", "zh-CN": "HTTPS 可访问"},
	"doctor.network.progress":                    {"en-US": "Checking %s", "en-GB": "Checking %s", "zh-CN": "正在检查%s"},
	"doctor.network.progress.prepare":            {"en-US": "Preparing source verification", "en-GB": "Preparing source verification", "zh-CN": "正在准备源验证"},
	"doctor.network.progress.finalise":           {"en-US": "Finalising network diagnostics", "en-GB": "Finalising network diagnostics", "zh-CN": "正在完成网络诊断"},
	"doctor.network.summary":                     {"en-US": "Network diagnostics: passed %d; warnings %d; failed %d; skipped %d.", "en-GB": "Network diagnostics: passed %d; warnings %d; failed %d; skipped %d.", "zh-CN": "网络诊断完成：通过 %d 项；警告 %d 项；失败 %d 项；跳过 %d 项。"},
	"error.upgrade_dev_apply":                    {"en-US": "development builds cannot self-upgrade; run ctyun update --check to check hosted release metadata", "en-GB": "development builds cannot self-upgrade; run ctyun update --check to check hosted release metadata", "zh-CN": "开发构建不可执行自升级；请运行 ctyun update --check 检查托管发布元数据"},
	"upgrade.current":                            {"en-US": "ctyun %s is already up to date on channel %s.", "en-GB": "ctyun %s is already up to date on channel %s.", "zh-CN": "ctyun %s 在 %s 渠道已是最新版本。"},
	"upgrade.installed":                          {"en-US": "Upgraded %s: %s -> %s.", "en-GB": "Upgraded %s: %s -> %s.", "zh-CN": "已升级 %s：%s -> %s。"},
	"upgrade.available":                          {"en-US": "ctyun %s is available from %s for %s/%s (%s).", "en-GB": "ctyun %s is available from %s for %s/%s (%s).", "zh-CN": "ctyun %s 可从 %s 获取，适用于 %s/%s（%s）。"},
	"upgrade.failed_summary":                     {"en-US": "Core update failed: upgraded 0; failed %d.", "en-GB": "Core update failed: upgraded 0; failed %d.", "zh-CN": "核心更新失败：已升级 0 个；失败 %d 个。"},
	"operation.target_failed":                    {"en-US": "%s: %s.", "en-GB": "%s: %s.", "zh-CN": "%s：%s。"},
	"operation.progress.install":                 {"en-US": "Installing %s", "en-GB": "Installing %s", "zh-CN": "正在安装 %s"},
	"operation.progress.reinstall":               {"en-US": "Reinstalling %s", "en-GB": "Reinstalling %s", "zh-CN": "正在重新安装 %s"},
	"operation.progress.update":                  {"en-US": "Updating %s", "en-GB": "Updating %s", "zh-CN": "正在更新 %s"},
	"operation.progress.remove":                  {"en-US": "Removing %s", "en-GB": "Removing %s", "zh-CN": "正在删除 %s"},
	"operation.progress.download":                {"en-US": "Downloading %s", "en-GB": "Downloading %s", "zh-CN": "正在下载 %s"},
	"operation.progress.verify":                  {"en-US": "Verifying %s", "en-GB": "Verifying %s", "zh-CN": "正在校验 %s"},
	"operation.progress.core_install":            {"en-US": "Installing %s", "en-GB": "Installing %s", "zh-CN": "正在安装 %s"},
	"plugin.install.summary":                     {"en-US": "Plugin install complete: installed %d; already installed %d; failed %d.", "en-GB": "Plugin install complete: installed %d; already installed %d; failed %d.", "zh-CN": "插件安装完成：已安装 %d 个；已存在 %d 个；失败 %d 个。"},
	"plugin.reinstall.summary":                   {"en-US": "Plugin reinstall complete: reinstalled %d; failed %d.", "en-GB": "Plugin reinstall complete: reinstalled %d; failed %d.", "zh-CN": "插件重新安装完成：已重新安装 %d 个；失败 %d 个。"},
	"plugin.update.summary":                      {"en-US": "Plugin update complete: updated %d; already current %d; failed %d.", "en-GB": "Plugin update complete: updated %d; already current %d; failed %d.", "zh-CN": "插件更新完成：已更新 %d 个；已是最新 %d 个；失败 %d 个。"},
	"plugin.remove.summary":                      {"en-US": "Plugin removal complete: removed %d; failed %d.", "en-GB": "Plugin removal complete: removed %d; failed %d.", "zh-CN": "插件删除完成：已删除 %d 个；失败 %d 个。"},
	"plugin.valid":                               {"en-US": "Valid plugin %s %s.", "en-GB": "Valid plugin %s %s.", "zh-CN": "有效插件 %s %s。"},
	"plugin.update_available":                    {"en-US": "Update available for %s: %s -> %s.", "en-GB": "Update available for %s: %s -> %s.", "zh-CN": "%s 可更新：%s -> %s。"},
	"waiter.status":                              {"en-US": "waiter %s: %s", "en-GB": "waiter %s: %s", "zh-CN": "等待器 %s：%s"},
	"waiter.state.success":                       {"en-US": "success", "en-GB": "success", "zh-CN": "成功"},
	"waiter.state.failure":                       {"en-US": "failure", "en-GB": "failure", "zh-CN": "失败"},
	"waiter.state.pending":                       {"en-US": "pending", "en-GB": "pending", "zh-CN": "等待中"},
	"waiter.state.timeout":                       {"en-US": "timeout", "en-GB": "timeout", "zh-CN": "超时"},
	"plugin.quality.generated":                   {"en-US": "generated", "en-GB": "generated", "zh-CN": "工具生成"},
	"plugin.quality.reviewed":                    {"en-US": "reviewed", "en-GB": "reviewed", "zh-CN": "已复核"},
	"plugin.quality.curated":                     {"en-US": "curated", "en-GB": "curated", "zh-CN": "持续维护"},
	"plugin.status.available":                    {"en-US": "available", "en-GB": "available", "zh-CN": "可安装"},
	"plugin.status.outdated":                     {"en-US": "outdated", "en-GB": "outdated", "zh-CN": "可更新"},
	"plugin.status.installed":                    {"en-US": "installed", "en-GB": "installed", "zh-CN": "已安装"},
	"config.updated":                             {"en-US": "Updated %s.", "en-GB": "Updated %s.", "zh-CN": "已更新 %s。"},
	"config.unset":                               {"en-US": "Unset %s.", "en-GB": "Unset %s.", "zh-CN": "已取消 %s。"},
	"config.active_profile":                      {"en-US": "Active profile: %s.", "en-GB": "Active profile: %s.", "zh-CN": "当前配置档案：%s。"},
	"config.profile_updated":                     {"en-US": "Updated profile %s %s.", "en-GB": "Updated profile %s %s.", "zh-CN": "已更新配置档案 %s 的 %s。"},
	"config.profile_unset":                       {"en-US": "Unset profile %s %s.", "en-GB": "Unset profile %s %s.", "zh-CN": "已取消配置档案 %s 的 %s。"},
	"config.profile_reset":                       {"en-US": "Reset profile %s.", "en-GB": "Reset profile %s.", "zh-CN": "已重置配置档案 %s。"},
	"config.already_reset":                       {"en-US": "Config is already reset.", "en-GB": "Config is already reset.", "zh-CN": "配置已重置。"},
	"config.reset":                               {"en-US": "Reset config; backup: %s.", "en-GB": "Reset config; backup: %s.", "zh-CN": "已重置配置；备份：%s。"},
	"confirmation.prompt":                        {"en-US": "%s requires confirmation. Continue? [y/N]: ", "en-GB": "%s requires confirmation. Continue? [y/N]: ", "zh-CN": "%s 需要确认。是否继续？[y/N]："},
	"warning.config_credentials": {
		"en-US": "Warning: using CTyun AK/SK from config. Disable this warning by setting the CTYUN_WARN_CONFIG_CREDENTIALS=0 environment variable or running ctyun config set warn_config_credentials false.",
		"en-GB": "Warning: using CTyun AK/SK from config. Disable this warning by setting the CTYUN_WARN_CONFIG_CREDENTIALS=0 environment variable or running ctyun config set warn_config_credentials false.",
		"zh-CN": "警告：正在使用配置中的天翼云 AK/SK。可设置环境变量 CTYUN_WARN_CONFIG_CREDENTIALS=0，或运行 ctyun config set warn_config_credentials false 关闭此提醒。",
	},
	"warning.deprecated_command": {
		"en-US": "Warning: this plugin command is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"en-GB": "Warning: this plugin command is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"zh-CN": "警告：此插件命令已弃用。%s可设置环境变量 CTYUN_WARN_DEPRECATED=0，或运行 ctyun config set warn_deprecated false 关闭此提醒。",
	},
	"warning.deprecated_api": {
		"en-US": "Warning: the CTyun API used by this command is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"en-GB": "Warning: the CTyun API used by this command is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"zh-CN": "警告：此命令使用的天翼云 API 已弃用。%s可设置环境变量 CTYUN_WARN_DEPRECATED=0，或运行 ctyun config set warn_deprecated false 关闭此提醒。",
	},
	"warning.deprecated_option": {
		"en-US": "Warning: option --%s is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"en-GB": "Warning: option --%s is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"zh-CN": "警告：选项 --%s 已弃用。%s可设置环境变量 CTYUN_WARN_DEPRECATED=0，或运行 ctyun config set warn_deprecated false 关闭此提醒。",
	},
	"warning.deprecated_field": {
		"en-US": "Warning: output field %s is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"en-GB": "Warning: output field %s is deprecated.%s Disable this warning by setting the CTYUN_WARN_DEPRECATED=0 environment variable or running ctyun config set warn_deprecated false.",
		"zh-CN": "警告：输出字段 %s 已弃用。%s可设置环境变量 CTYUN_WARN_DEPRECATED=0，或运行 ctyun config set warn_deprecated false 关闭此提醒。",
	},
	"warning.deprecated_replacement": {
		"en-US": " Recommended CLI replacement: %s.",
		"en-GB": " Recommended CLI replacement: %s.",
		"zh-CN": " 建议使用 CLI 替代项：%s。",
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

// pluginValidMessage returns the localized plugin lint success status.
func pluginValidMessage(language, name, pluginVersion string) string {
	return messagef("plugin.valid", language, name, pluginVersion)
}

// pluginUpdateAvailableMessage returns the localized plugin update listing
// status.
func pluginUpdateAvailableMessage(language, name, currentVersion, nextVersion string) string {
	return messagef("plugin.update_available", language, name, currentVersion, nextVersion)
}

// waiterStatusMessage returns the localized waiter status line.
func waiterStatusMessage(language, waiterID, state string) string {
	return messagef("waiter.status", language, waiterID, messageText("waiter.state."+state, language))
}

// pluginQualityText returns the localized plugin quality enum for display.
func pluginQualityText(language, quality string) string {
	return messageText("plugin.quality."+quality, language)
}

// pluginStatusText returns the localized plugin installation status enum for
// display.
func pluginStatusText(language, status string) string {
	return messageText("plugin.status."+status, language)
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
