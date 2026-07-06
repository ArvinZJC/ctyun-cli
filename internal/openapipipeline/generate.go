/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// GenerateDraft writes candidate plugin metadata under
// openapi-catalogs/<product>/draft without modifying promoted plugins.
func (workspace Workspace) GenerateDraft(product string) error {
	catalog, err := workspace.ReadSource(product)
	if err != nil {
		return err
	}
	draftDir := workspace.ProductPath(product, "draft")
	if err := prepareDraftDir(draftDir); err != nil {
		return err
	}
	return writeDraft(draftDir, catalog)
}

// writeDraft writes generated plugin metadata, i18n catalogs, and fixtures.
func writeDraft(draftDir string, catalog Catalog) error {
	files := map[string]any{
		"plugin.json":   buildManifest(catalog),
		"apis.json":     buildAPIs(catalog),
		"commands.json": buildCommands(catalog),
		"tables.json":   buildTables(catalog),
		"waiters.json":  buildWaiters(catalog),
	}
	for name, value := range files {
		if err := writeJSON(filepath.Join(draftDir, name), value); err != nil {
			return err
		}
	}
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		if err := writeJSON(filepath.Join(draftDir, "i18n", language+".json"), buildI18N(catalog, language)); err != nil {
			return err
		}
	}
	return writeFixtures(draftDir, catalog)
}

// prepareDraftDir removes stale generated draft contents before regeneration.
func prepareDraftDir(draftDir string) error {
	info, err := os.Stat(draftDir)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("draft path %s is not a directory", draftDir)
		}
		return os.RemoveAll(draftDir)
	case os.IsNotExist(err):
		return nil
	default:
		return err
	}
}

// buildManifest converts catalog product metadata into plugin.json.
func buildManifest(catalog Catalog) plugin.Manifest {
	return plugin.Manifest{
		Name:    catalog.Product.PluginName,
		Version: "0.1.0-alpha.1",
		Channel: "alpha",
		Quality: "generated",
		Requires: plugin.Requirements{
			Ctyun: ">=0.1.0-alpha.1 <1.0.0",
		},
		API: plugin.APIInfo{
			Product:           catalog.Product.APIProduct,
			CtyunProductID:    catalog.Product.CtyunProductID,
			SourceRevision:    catalog.Product.SourceRevision,
			SourceFingerprint: catalogFingerprint(catalog),
			EndpointURL:       catalog.Product.EndpointURL,
		},
	}
}

// buildAPIs converts catalog operations into apis.json.
func buildAPIs(catalog Catalog) plugin.APIs {
	operations := make(map[string]plugin.Operation, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		next := plugin.Operation{
			Method:           operation.Method,
			Path:             operation.Path,
			ContentType:      operation.ContentType,
			Retryable:        operation.Retryable,
			AcceptedStatuses: operation.Response.AcceptedStatuses,
			Query:            map[string]string{},
			Headers:          map[string]string{},
			Body:             map[string]string{},
		}
		for _, parameter := range operation.Parameters {
			binding := parameterBinding(parameter)
			if binding == "" {
				continue
			}
			switch parameter.Location {
			case "query":
				next.Query[parameter.Name] = binding
			case "header":
				next.Headers[parameter.Name] = binding
			case "body":
				next.Body[parameter.Name] = binding
			}
		}
		operations[operation.ID] = next
	}
	return plugin.APIs{Operations: operations}
}

// buildCommands converts catalog operations into commands.json.
func buildCommands(catalog Catalog) plugin.Commands {
	commands := make([]plugin.Command, 0, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		action := commandAction(operation)
		examples := append([]string(nil), operation.Examples...)
		if len(examples) == 0 {
			examples = []string{"ctyun " + strings.Join(commandPath(catalog, operation), " ")}
		}
		command := plugin.Command{
			ID:                      commandID(operation),
			Path:                    commandPath(catalog, operation),
			Operation:               operation.ID,
			Table:                   tableID(catalog, operation),
			ConditionalRequirements: operation.ConditionalRequirements,
			FixtureResponse:         fixturePath(operation),
			DocsURL:                 operation.DocsURL,
			Examples:                examples,
			Dangerous:               plugin.Dangerous{},
		}
		if operation.Dangerous {
			command.Dangerous = plugin.Dangerous{Confirm: "yes", Message: action}
		}
		for _, parameter := range operation.Parameters {
			if parameter.CLIName == "" {
				continue
			}
			flag := parameter.CLIFlag
			if flag == "" {
				flag = parameter.CLIName
			}
			command.Parameters = append(command.Parameters, plugin.Parameter{
				Name:          parameter.CLIName,
				Flag:          flag,
				Target:        parameter.TableTarget,
				Required:      parameter.Required,
				AllowedValues: parameter.Enum,
				Pattern:       parameter.Pattern,
				Description:   parameterEnglishDescription(parameter),
			})
		}
		commands = append(commands, command)
	}
	return plugin.Commands{Commands: commands}
}

// buildTables converts catalog response columns into tables.json.
func buildTables(catalog Catalog) plugin.Tables {
	tables := make(map[string]plugin.Table, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		columns := make([]plugin.TableColumn, 0, len(operation.Response.Columns))
		for _, column := range operation.Response.Columns {
			labelEN := englishColumnLabel(column)
			columns = append(columns, plugin.TableColumn{
				Key:  column.Key,
				Path: column.Path,
				Labels: map[string]string{
					"en-US": labelEN,
					"en-GB": labelEN,
					"zh-CN": chineseColumnLabel(column, labelEN),
				},
			})
		}
		tables[tableID(catalog, operation)] = plugin.Table{
			RowPath:        operation.Response.RowPath,
			Layout:         operation.Response.Layout,
			DefaultColumns: operation.Response.DefaultColumns,
			Columns:        columns,
		}
	}
	return plugin.Tables{Tables: tables}
}

// buildWaiters derives conservative waiters from reviewed response evidence.
func buildWaiters(catalog Catalog) plugin.Waiters {
	waiters := map[string]plugin.Waiter{}
	for _, operation := range catalog.Operations {
		if commandID(operation) != catalog.Product.PluginName+".instance.show" {
			continue
		}
		if operation.Response.RowPath != "returnObj" || !hasResponseColumnPath(operation.Response, "instanceStatus") {
			continue
		}
		waiters[catalog.Product.PluginName+".instance.running"] = plugin.Waiter{
			Path:            "returnObj.instanceStatus",
			Success:         "running",
			Failure:         "error",
			MaxAttempts:     20,
			IntervalSeconds: 3,
		}
		waiters[catalog.Product.PluginName+".instance.stopped"] = plugin.Waiter{
			Path:            "returnObj.instanceStatus",
			Success:         "stopped",
			Failure:         "error",
			MaxAttempts:     20,
			IntervalSeconds: 3,
		}
	}
	return plugin.Waiters{Waiters: waiters}
}

// buildI18N converts catalog display names into plugin i18n entries.
func buildI18N(catalog Catalog, language string) map[string]string {
	entries := map[string]string{}
	if name := catalog.Product.DisplayName[language]; name != "" {
		entries["name"] = name
	}
	for _, operation := range catalog.Operations {
		id := commandID(operation)
		if description := operation.Description[language]; description != "" {
			entries["command."+id+".description"] = description
		}
		for _, parameter := range operation.Parameters {
			if parameter.CLIName == "" {
				continue
			}
			description := parameterLocalizedDescription(parameter, language)
			if description != "" {
				entries["parameter."+id+"."+parameter.CLIName+".description"] = description
			}
		}
	}
	return entries
}

// hasResponseColumnPath reports whether response exposes a table column path.
func hasResponseColumnPath(response Response, path string) bool {
	for _, column := range response.Columns {
		if column.Path == path {
			return true
		}
	}
	return false
}

// parameterLocalizedDescription returns safe localized help text for a CLI
// parameter without leaking Chinese-only upstream prose into English catalogs.
func parameterLocalizedDescription(parameter Parameter, language string) string {
	if description := strings.TrimSpace(parameter.Descriptions[language]); description != "" {
		return description
	}
	if language == "zh-CN" {
		if description := strings.TrimSpace(parameter.Description); description != "" {
			return description
		}
		return chineseNameForIdentifier(parameterIdentifier(parameter))
	}
	return parameterEnglishDescription(parameter)
}

// parameterEnglishDescription returns English command metadata text for a CLI
// parameter.
func parameterEnglishDescription(parameter Parameter) string {
	for _, language := range []string{"en-US", "en-GB"} {
		if description := strings.TrimSpace(parameter.Descriptions[language]); description != "" {
			return description
		}
	}
	if description := strings.TrimSpace(parameter.Description); description != "" && !containsCJK(description) {
		return description
	}
	return englishNameForIdentifier(parameterIdentifier(parameter))
}

// parameterIdentifier picks the most useful stable name for generated labels.
func parameterIdentifier(parameter Parameter) string {
	for _, value := range []string{parameter.CLIName, parameter.TableTarget, parameter.Name} {
		if value != "" {
			return value
		}
	}
	return "value"
}

// englishColumnLabel returns normalized English table label text.
func englishColumnLabel(column Column) string {
	if label := strings.TrimSpace(column.LabelEN); label != "" && !containsCJK(label) {
		return label
	}
	return englishNameForIdentifier(columnIdentifier(column))
}

// chineseColumnLabel returns Chinese table label text when source evidence or a
// conservative generated phrase is available.
func chineseColumnLabel(column Column, englishLabel string) string {
	if label := strings.TrimSpace(column.LabelZH); label != "" {
		if containsCJK(label) || isCompactTechnicalLabel(label) {
			return label
		}
	}
	if label := chineseNameForIdentifier(columnIdentifier(column)); label != "" {
		englishLabel = strings.TrimSpace(englishLabel)
		if !containsCJK(label) && englishLabel != "" && !containsCJK(englishLabel) {
			return englishLabel
		}
		return label
	}
	return englishLabel
}

// isCompactTechnicalLabel reports whether a non-Chinese source label is likely
// a technical acronym or product token that should be preserved verbatim.
func isCompactTechnicalLabel(label string) bool {
	return !strings.ContainsAny(label, " \t\r\n") && label != strings.ToLower(label)
}

// columnIdentifier picks the most useful stable name for generated labels.
func columnIdentifier(column Column) string {
	for _, value := range []string{column.Key, column.Path, column.LabelEN} {
		if value != "" {
			return value
		}
	}
	return "value"
}

// normalizeEnglishLabel tidies generated English labels and common OpenAPI
// acronyms.
func normalizeEnglishLabel(value string) string {
	words := identifierWords(value)
	for index, word := range words {
		words[index] = englishWord(word)
	}
	return strings.Join(words, " ")
}

// englishNameForIdentifier turns a field or flag identifier into English text.
func englishNameForIdentifier(identifier string) string {
	return normalizeEnglishLabel(identifier)
}

// chineseNameForIdentifier turns common generated field identifiers into
// Chinese text while leaving unknown technical tokens in English.
func chineseNameForIdentifier(identifier string) string {
	key := snakeIdentifier(identifier)
	if label, ok := chinesePhraseLabels[key]; ok {
		return label
	}
	words := identifierWords(identifier)
	translated := make([]string, 0, len(words))
	for _, word := range words {
		normalized := strings.ToLower(word)
		if label, ok := chineseTokenLabels[normalized]; ok {
			translated = append(translated, label)
			continue
		}
		translated = append(translated, englishWord(word))
	}
	return joinChineseLabelParts(translated)
}

// joinChineseLabelParts joins generated Chinese label fragments without adding
// spaces between adjacent Chinese phrases.
func joinChineseLabelParts(parts []string) string {
	var builder strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if builder.Len() > 0 && needsChineseLabelSpace(lastRuneString(builder.String()), []rune(part)[0]) {
			builder.WriteByte(' ')
		}
		builder.WriteString(part)
	}
	return builder.String()
}

// needsChineseLabelSpace reports whether adjacent generated label fragments
// need an ASCII separator for readability.
func needsChineseLabelSpace(previous, current rune) bool {
	if isCJKRune(previous) && isCJKRune(current) {
		return false
	}
	return true
}

// lastRuneString returns the final rune in value.
func lastRuneString(value string) rune {
	var last rune
	for _, char := range value {
		last = char
	}
	return last
}

// snakeIdentifier normalizes identifiers for phrase lookup.
func snakeIdentifier(value string) string {
	words := identifierWords(value)
	for index, word := range words {
		words[index] = strings.ToLower(word)
	}
	return strings.Join(words, "_")
}

// identifierWords splits snake, kebab, camel, and acronym-heavy identifiers.
func identifierWords(value string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
	}
	value = normalizeIdentifierAcronyms(value)
	runes := []rune(strings.TrimSpace(value))
	for index, char := range runes {
		if char == '_' || char == '-' || char == ' ' || char == '.' {
			flush()
			continue
		}
		if len(current) > 0 && shouldSplitIdentifierWord(current[len(current)-1], char, nextRune(runes, index)) {
			flush()
		}
		current = append(current, char)
	}
	flush()
	return words
}

// normalizeIdentifierAcronyms separates common acronym runs before generic
// identifier splitting.
func normalizeIdentifierAcronyms(value string) string {
	replacer := strings.NewReplacer(
		"IDList", " idlist ",
		"IPv6", " ipv6 ",
		"IPv4", " ipv4 ",
		"UUID", " uuid ",
		"IDs", " ids ",
		"ID", " id ",
		"CPU", " cpu ",
		"GPU", " gpu ",
		"VPC", " vpc ",
		"EIP", " eip ",
		"ECS", " ecs ",
		"DNS", " dns ",
		"VNC", " vnc ",
		"URL", " url ",
		"ACL", " acl ",
		"KMS", " kms ",
		"QoS", " qos ",
		"AZ", " az ",
		"OS", " os ",
		"IP", " ip ",
	)
	return replacer.Replace(value)
}

// shouldSplitIdentifierWord reports whether a new identifier word starts.
func shouldSplitIdentifierWord(previous, current, next rune) bool {
	if isDigit(previous) && isLetter(current) {
		return true
	}
	if isLower(previous) && isUpper(current) {
		return true
	}
	return isUpper(previous) && isUpper(current) && next != 0 && isLower(next)
}

// nextRune returns the next rune after index.
func nextRune(runes []rune, index int) rune {
	if index+1 >= len(runes) {
		return 0
	}
	return runes[index+1]
}

// isLetter reports whether char is an ASCII letter.
func isLetter(char rune) bool {
	return isLower(char) || isUpper(char)
}

// isLower reports whether char is an ASCII lowercase letter.
func isLower(char rune) bool {
	return char >= 'a' && char <= 'z'
}

// isUpper reports whether char is an ASCII uppercase letter.
func isUpper(char rune) bool {
	return char >= 'A' && char <= 'Z'
}

// isDigit reports whether char is an ASCII digit.
func isDigit(char rune) bool {
	return char >= '0' && char <= '9'
}

// containsCJK reports whether text contains common CJK ideographs.
func containsCJK(value string) bool {
	for _, char := range value {
		if isCJKRune(char) {
			return true
		}
	}
	return false
}

// isCJKRune reports whether char is a common CJK ideograph.
func isCJKRune(char rune) bool {
	return (char >= '\u3400' && char <= '\u9fff') || (char >= '\uf900' && char <= '\ufaff')
}

// englishWord normalizes common OpenAPI acronyms and generated token casing.
func englishWord(word string) string {
	lower := strings.ToLower(word)
	if replacement, ok := englishAcronyms[lower]; ok {
		return replacement
	}
	if lower == "" {
		return ""
	}
	return strings.ToUpper(lower[:1]) + lower[1:]
}

// englishAcronyms preserves common cloud and OpenAPI acronyms in generated
// English labels.
var englishAcronyms = map[string]string{
	"acl":    "ACL",
	"az":     "AZ",
	"cpu":    "CPU",
	"dns":    "DNS",
	"ebs":    "EBS",
	"ecs":    "ECS",
	"eip":    "EIP",
	"gpu":    "GPU",
	"id":     "ID",
	"idlist": "ID List",
	"ids":    "IDs",
	"ip":     "IP",
	"ipv4":   "IPv4",
	"ipv6":   "IPv6",
	"kms":    "KMS",
	"no":     "No",
	"os":     "OS",
	"qos":    "QoS",
	"sg":     "Security Group",
	"url":    "URL",
	"uuid":   "UUID",
	"vnc":    "VNC",
	"vpc":    "VPC",
}

// chinesePhraseLabels gives preferred Chinese labels for common generated
// response and parameter identifiers.
var chinesePhraseLabels = map[string]string{
	"affinity_group_id":      "云主机组 ID",
	"affinity_group_name":    "云主机组名称",
	"alarm_id":               "告警 ID",
	"attachment_id":          "挂载 ID",
	"auto_deploy":            "自动部署",
	"az_id":                  "可用区 ID",
	"az_name":                "可用区名称",
	"cpu_arch":               "CPU 架构",
	"cpu_info":               "CPU 信息",
	"created_time":           "创建时间",
	"dedicated_host_id":      "专属宿主机 ID",
	"deletion_protection":    "删除保护",
	"display_name":           "显示名称",
	"disk_id":                "云硬盘 ID",
	"disk_name":              "云硬盘名称",
	"disk_request_id":        "云硬盘请求 ID",
	"finger_print":           "指纹",
	"f_uid":                  "UID",
	"gpu_driver_list":        "GPU 驱动列表",
	"instance_backup_id":     "云主机备份 ID",
	"instance_backup_idlist": "云主机备份 ID 列表",
	"instance_id":            "云主机 ID",
	"instance_idlist":        "云主机 ID 列表",
	"instance_name":          "云主机名称",
	"instance_status":        "云主机状态",
	"job_id":                 "任务 ID",
	"job_status":             "任务状态",
	"market_price":           "市场价格",
	"master_order_id":        "主订单 ID",
	"master_order_no":        "主订单号",
	"master_resource_id":     "主资源 ID",
	"master_resource_status": "主资源状态",
	"network_interface_id":   "网卡 ID",
	"need_migrate":           "需要迁移",
	"option":                 "选项",
	"policy_id":              "策略 ID",
	"policy_name":            "策略名称",
	"policy_type_name":       "策略类型名称",
	"project_id":             "企业项目 ID",
	"repository_id":          "存储库 ID",
	"retention_day":          "保留天数",
	"security_group_id":      "安全组 ID",
	"security_group_rule_id": "安全组规则 ID",
	"secondary_private_ips":  "辅助私网 IP",
	"sg_rule_ids":            "安全组规则 ID",
	"snapshot_id":            "快照 ID",
	"snapshot_name":          "快照名称",
	"snapshot_status":        "快照状态",
	"stage":                  "阶段",
	"task_id":                "任务 ID",
	"template_description":   "模板描述",
	"template_id":            "模板 ID",
	"template_name":          "模板名称",
	"total_disk_size":        "云硬盘总容量",
	"updated_time":           "更新时间",
	"usage":                  "使用量",
	"vpc_id":                 "虚拟私有云 ID",
	"vpc_name":               "VPC 名称",
}

// chineseTokenLabels translates common identifier tokens when no full phrase
// match is available.
var chineseTokenLabels = map[string]string{
	"action":      "动作",
	"available":   "可用",
	"backup":      "备份",
	"bandwidth":   "带宽",
	"client":      "客户端",
	"count":       "数量",
	"created":     "创建",
	"description": "描述",
	"device":      "设备",
	"direction":   "方向",
	"disk":        "云硬盘",
	"ecs":         "云主机",
	"eip":         "弹性公网 IP",
	"flavor":      "规格",
	"force":       "强制",
	"group":       "组",
	"id":          "ID",
	"idlist":      "ID 列表",
	"ids":         "ID 列表",
	"image":       "镜像",
	"instance":    "云主机",
	"ip":          "IP 地址",
	"job":         "任务",
	"key":         "密钥",
	"memory":      "内存",
	"metadata":    "元数据",
	"name":        "名称",
	"no":          "号",
	"order":       "订单",
	"origin":      "来源",
	"page":        "页",
	"policy":      "策略",
	"port":        "网卡",
	"project":     "企业项目",
	"region":      "资源池",
	"repo":        "存储库",
	"repository":  "存储库",
	"request":     "请求",
	"resource":    "资源",
	"result":      "结果",
	"rule":        "规则",
	"security":    "安全组",
	"size":        "大小",
	"snapshot":    "快照",
	"status":      "状态",
	"task":        "任务",
	"template":    "模板",
	"time":        "时间",
	"token":       "令牌",
	"total":       "总数",
	"type":        "类型",
	"updated":     "更新",
	"userdata":    "用户数据",
	"value":       "值",
	"volume":      "云硬盘",
	"vpc":         "VPC",
}

// commandPath derives the canonical plugin command path for an operation.
func commandPath(catalog Catalog, operation Operation) []string {
	path := []string{catalog.Product.PluginName}
	if operation.Category != "" {
		path = append(path, operation.Category)
	}
	path = append(path, commandAction(operation))
	if argument := firstArgument(operation); argument != "" {
		path = append(path, "{"+argument+"}")
	}
	return path
}

// commandAction derives the leaf command action from operation evidence.
func commandAction(operation Operation) string {
	parts := strings.Split(operation.ID, ".")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		return parts[len(parts)-1]
	}
	lowerPath := strings.ToLower(operation.Path)
	if strings.Contains(lowerPath, "list-") || strings.Contains(lowerPath, "-list") {
		return "list"
	}
	if strings.Contains(lowerPath, "detail") || strings.Contains(lowerPath, "details") {
		return "show"
	}
	return "call"
}

// parameterBinding returns the metadata binding expression for a parameter.
func parameterBinding(parameter Parameter) string {
	switch {
	case parameter.Profile != "":
		return "$profile." + parameter.Profile
	case parameter.Argument != "":
		return "$arg." + parameter.Argument
	case parameter.CLIName != "":
		return "$param." + parameter.CLIName
	default:
		return ""
	}
}

// tableID returns the generated table identifier for an operation.
func tableID(catalog Catalog, operation Operation) string {
	parts := []string{catalog.Product.PluginName}
	if operation.Category != "" {
		parts = append(parts, operation.Category)
	}
	parts = append(parts, commandAction(operation))
	return strings.Join(parts, ".")
}

// commandID returns the generated command identifier for an operation.
func commandID(operation Operation) string {
	return strings.TrimPrefix(operation.ID, "v4.")
}

// firstArgument returns the first path argument bound by an operation.
func firstArgument(operation Operation) string {
	for _, parameter := range operation.Parameters {
		if parameter.Argument != "" {
			return parameter.Argument
		}
	}
	return ""
}

// fixturePath returns the generated fixture path for an operation.
func fixturePath(operation Operation) string {
	return "fixtures/" + strings.ReplaceAll(commandID(operation), ".", "-") + ".json"
}

// writeFixtures writes one generated fixture file per catalog operation.
func writeFixtures(draftDir string, catalog Catalog) error {
	for _, operation := range catalog.Operations {
		raw := compactRawMessage(operation.ExampleResponse)
		if len(raw) == 0 {
			raw = json.RawMessage(`{}`)
		}
		if err := writeRawJSON(filepath.Join(draftDir, fixturePath(operation)), raw); err != nil {
			return err
		}
	}
	return nil
}

// writeRawJSON writes an already validated JSON document with indentation.
func writeRawJSON(path string, raw json.RawMessage) error {
	var buffer bytes.Buffer
	if err := json.Indent(&buffer, raw, "", "  "); err != nil {
		return err
	}
	data := collapseIndentedScalarArrays(buffer.String())
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// collapseIndentedScalarArrays keeps short fixture scalar lists on one line.
func collapseIndentedScalarArrays(value string) []byte {
	lines := strings.Split(value, "\n")
	collapsed := make([]string, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		if !strings.HasSuffix(trimmed, "[") {
			collapsed = append(collapsed, line)
			continue
		}
		items, endIndex, ok := scalarArrayItems(lines, index+1)
		if !ok {
			collapsed = append(collapsed, line)
			continue
		}
		endTrimmed := strings.TrimSpace(lines[endIndex])
		suffix := ""
		if strings.HasSuffix(endTrimmed, ",") {
			suffix = ","
		}
		prefix := strings.TrimSuffix(line, "[")
		collapsed = append(collapsed, prefix+"["+strings.Join(items, ", ")+"]"+suffix)
		index = endIndex
	}
	return []byte(strings.Join(collapsed, "\n"))
}

// scalarArrayItems returns compact item text for an indented scalar array.
func scalarArrayItems(lines []string, start int) ([]string, int, bool) {
	var items []string
	for index := start; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "]" || trimmed == "]," {
			return items, index, true
		}
		item := strings.TrimSuffix(trimmed, ",")
		if item == "" || strings.HasPrefix(item, "{") || strings.HasPrefix(item, "[") {
			return nil, 0, false
		}
		items = append(items, item)
	}
	return nil, 0, false
}
