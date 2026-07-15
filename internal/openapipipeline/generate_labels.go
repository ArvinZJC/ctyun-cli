/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import "strings"

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
		if !containsCJK(label) && isCompactTechnicalLabel(label) {
			return label
		}
		if containsCJK(label) && !generatedChineseColumnLabel(label) {
			return normalizeChineseTechnicalLabel(label)
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

// generatedChineseColumnLabel reports source table labels that should be
// replaced with generated labels instead of merely normalized.
func generatedChineseColumnLabel(label string) bool {
	if strings.Contains(label, "您可以查看") ||
		(strings.Contains(label, "获取：") && (strings.Contains(label, " 查 ") || strings.Contains(label, " 创 "))) {
		return true
	}
	//goland:noinspection HttpUrlsUsage
	if strings.Contains(label, "http://") || strings.Contains(label, "https://") {
		return true
	}
	if strings.ContainsAny(label, "，。；;：:") {
		return true
	}
	if len([]rune(label)) > 24 {
		return true
	}
	return containsNonTechnicalASCIIWord(label)
}

// containsNonTechnicalASCIIWord reports whether label includes English words
// that are not compact cloud acronyms such as ID or IP.
func containsNonTechnicalASCIIWord(label string) bool {
	var word []rune
	flush := func() bool {
		if len(word) == 0 {
			return false
		}
		token := string(word)
		word = nil
		return !isTechnicalASCIIWord(token)
	}
	for _, char := range label {
		if isASCIIAlphaNum(char) {
			word = append(word, char)
			continue
		}
		if flush() {
			return true
		}
	}
	return flush()
}

// normalizeChineseTechnicalLabel adds readable spacing around compact technical
// tokens embedded in Chinese labels.
func normalizeChineseTechnicalLabel(label string) string {
	var builder strings.Builder
	var token []rune
	flush := func() {
		if len(token) == 0 {
			return
		}
		text := canonicalTechnicalASCIIWord(string(token))
		token = nil
		if builder.Len() > 0 && needsChineseTechnicalLabelSpace(lastRuneString(builder.String()), []rune(text)[0]) {
			builder.WriteByte(' ')
		}
		builder.WriteString(text)
	}
	for _, char := range label {
		if isASCIIAlphaNum(char) {
			token = append(token, char)
			continue
		}
		flush()
		if builder.Len() > 0 && needsChineseTechnicalLabelSpace(lastRuneString(builder.String()), char) {
			builder.WriteByte(' ')
		}
		builder.WriteRune(char)
	}
	flush()
	return strings.Join(strings.Fields(builder.String()), " ")
}

// needsChineseTechnicalLabelSpace reports whether a compact ASCII token and
// adjacent Chinese text need a separator while punctuation remains attached.
func needsChineseTechnicalLabelSpace(previous, current rune) bool {
	return (isCJKRune(previous) && isASCIIAlphaNum(current)) ||
		(isASCIIAlphaNum(previous) && isCJKRune(current))
}

// isASCIIAlphaNum reports whether char is an ASCII letter or digit.
func isASCIIAlphaNum(char rune) bool {
	return isLetter(char) || isDigit(char)
}

// isTechnicalASCIIWord reports whether word is a compact technical token that
// can appear inside a Chinese label.
func isTechnicalASCIIWord(word string) bool {
	_, ok := technicalASCIIWords[strings.ToLower(word)]
	return ok
}

// canonicalTechnicalASCIIWord returns the preferred casing for a compact
// technical token.
func canonicalTechnicalASCIIWord(word string) string {
	lower := strings.ToLower(word)
	if canonical, ok := technicalASCIIWords[lower]; ok {
		return canonical
	}
	return word
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

// technicalASCIIWords lists compact technical tokens allowed inside Chinese
// labels.
var technicalASCIIWords = map[string]string{
	"acl":   "ACL",
	"az":    "AZ",
	"cbr":   "CBR",
	"cny":   "CNY",
	"cpu":   "CPU",
	"dns":   "DNS",
	"ebs":   "EBS",
	"ecs":   "ECS",
	"eip":   "EIP",
	"gib":   "GiB",
	"gpu":   "GPU",
	"id":    "ID",
	"ip":    "IP",
	"ipv4":  "IPv4",
	"ipv6":  "IPv6",
	"kms":   "KMS",
	"mbit":  "Mbit",
	"nat":   "NAT",
	"os":    "OS",
	"pgelb": "PGELB",
	"qos":   "QoS",
	"s":     "s",
	"uid":   "UID",
	"url":   "URL",
	"uuid":  "UUID",
	"vbs":   "VBS",
	"vm":    "VM",
	"vnc":   "VNC",
	"vpc":   "VPC",
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
	"command_content":        "命令内容",
	"command_id":             "命令 ID",
	"command_name":           "命令名称",
	"command_type":           "命令类型",
	"cpu_arch":               "CPU 架构",
	"cpu_info":               "CPU 信息",
	"created_time":           "创建时间",
	"dedicated_host_id":      "专属宿主机 ID",
	"default_parameter":      "自定义参数默认值",
	"deletion_protection":    "删除保护",
	"display_name":           "显示名称",
	"disk_id":                "云硬盘 ID",
	"disk_name":              "云硬盘名称",
	"disk_request_id":        "云硬盘请求 ID",
	"enabled_parameter":      "启用自定义参数",
	"expect_number":          "期望数量",
	"file_content":           "文件内容",
	"file_group":             "文件用户组",
	"file_mode":              "文件权限",
	"file_name":              "文件名称",
	"file_owner":             "文件所属用户",
	"filters":                "过滤条件",
	"finger_print":           "指纹",
	"f_uid":                  "UID",
	"gpu_driver_list":        "GPU 驱动列表",
	"instance_backup_id":     "云主机备份 ID",
	"instance_backup_idlist": "云主机备份 ID 列表",
	"instance_id":            "云主机 ID",
	"instance_idlist":        "云主机 ID 列表",
	"instance_ids":           "实例 ID 列表",
	"instance_name":          "云主机名称",
	"instance_status":        "云主机状态",
	"invoked_id":             "执行 ID",
	"job_id":                 "任务 ID",
	"job_status":             "任务状态",
	"key_pair_description":   "密钥对描述",
	"key_pair_id":            "密钥对 ID",
	"key_pair_name":          "密钥对名称",
	"market_price":           "市场价格",
	"master_order_id":        "主订单 ID",
	"master_order_no":        "主订单号",
	"master_resource_id":     "主资源 ID",
	"master_resource_status": "主资源状态",
	"network_interface_id":   "网卡 ID",
	"need_migrate":           "需要迁移",
	"option":                 "选项",
	"overwrite":              "覆盖",
	"page_no":                "页码",
	"page_number":            "页码",
	"page_size":              "每页行数",
	"parameter":              "自定义参数",
	"policy_id":              "策略 ID",
	"policy_name":            "策略名称",
	"policy_type_name":       "策略类型名称",
	"private_ip":             "内网 IP",
	"project_id":             "企业项目 ID",
	"region":                 "资源池 ID",
	"repository_id":          "存储库 ID",
	"retention_day":          "保留天数",
	"save_command":           "保存命令",
	"security_group_id":      "安全组 ID",
	"security_group_rule_id": "安全组规则 ID",
	"secondary_private_ips":  "辅助私网 IP",
	"sg_rule_ids":            "安全组规则 ID",
	"snapshot_id":            "快照 ID",
	"snapshot_name":          "快照名称",
	"snapshot_status":        "快照状态",
	"stage":                  "阶段",
	"subnet_id":              "子网 ID",
	"task_id":                "任务 ID",
	"target_directory":       "目标路径",
	"template_description":   "模板描述",
	"template_id":            "模板 ID",
	"template_name":          "模板名称",
	"timeout":                "超时时间",
	"total_disk_size":        "云硬盘总容量",
	"updated_time":           "更新时间",
	"usage":                  "使用量",
	"vpc_id":                 "虚拟私有云 ID",
	"vpc_name":               "VPC 名称",
	"working_directory":      "工作目录",
}

// chineseTokenLabels translates common identifier tokens when no full phrase
// match is available.
var chineseTokenLabels = map[string]string{
	"action":      "动作",
	"available":   "可用",
	"backup":      "备份",
	"bandwidth":   "带宽",
	"client":      "客户端",
	"command":     "命令",
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
	"python":      "Python",
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
