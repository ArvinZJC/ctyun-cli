/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import "testing"

func TestGeneratedLabelHelpersCoverFallbacks(t *testing.T) {
	if got := parameterEnglishDescription(Parameter{Descriptions: map[string]string{"en-GB": "British text"}}); got != "British text" {
		t.Fatalf("en-GB parameter description = %q", got)
	}
	if got := parameterEnglishDescription(Parameter{Description: "plain English"}); got != "plain English" {
		t.Fatalf("plain parameter description = %q", got)
	}
	if got := parameterLocalizedDescription(Parameter{Name: "fallbackName"}, "zh-CN"); got != "Fallback 名称" {
		t.Fatalf("fallback zh-CN parameter description = %q", got)
	}
	if got := parameterIdentifier(Parameter{}); got != "value" {
		t.Fatalf("empty parameter identifier = %q", got)
	}
	if got := englishColumnLabel(Column{Path: "instanceID"}); got != "Instance ID" {
		t.Fatalf("path-derived English column label = %q", got)
	}
	if got := englishColumnLabel(Column{LabelEN: "OpenAPI Available"}); got != "OpenAPI Available" {
		t.Fatalf("source English column label = %q", got)
	}
	if got := englishColumnLabel(Column{LabelEN: "Pay-As-You-Go PaaS"}); got != "Pay-As-You-Go PaaS" {
		t.Fatalf("hyphenated source English column label = %q", got)
	}
	if got := englishColumnLabel(Column{LabelEN: "中文", Key: "cpu_arch"}); got != "CPU Arch" {
		t.Fatalf("fallback English column label = %q", got)
	}
	if got := chineseColumnLabel(Column{LabelZH: "资源池"}, "Region"); got != "资源池" {
		t.Fatalf("source Chinese column label = %q", got)
	}
	if got := chineseColumnLabel(Column{LabelZH: "PaaS"}, "PaaS"); got != "PaaS" {
		t.Fatalf("source Chinese acronym column label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "command_id", LabelZH: "命令ID"}, "Command ID"); got != "命令 ID" {
		t.Fatalf("generated Chinese column label from raw ID label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "instance_id", LabelZH: "实例ID"}, "Instance ID"); got != "实例 ID" {
		t.Fatalf("normalized Chinese column label from compact ID label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "key_pair_name", LabelZH: "密钥 Pair 名称"}, "Key Pair Name"); got != "密钥对名称" {
		t.Fatalf("generated Chinese column label from mixed English label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "created_time", LabelZH: "Created Time"}, "Created Time"); got != "创建时间" {
		t.Fatalf("generated Chinese label from English source label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "SD-WAN"}, "SD-WAN"); got != "SD-WAN" {
		t.Fatalf("fallback Chinese acronym column label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "created_time"}, "Created Time"); got != "创建时间" {
		t.Fatalf("generated Chinese column label = %q", got)
	}
	if got := chineseNameForIdentifier("instance_backup_status"); got != "云主机备份状态" {
		t.Fatalf("generated Chinese phrase label = %q", got)
	}
	if got := chineseNameForIdentifier("policy_id"); got != "策略 ID" {
		t.Fatalf("generated Chinese technical label = %q", got)
	}
	if got := chineseNameForIdentifier("key_pair_id"); got != "密钥对 ID" {
		t.Fatalf("generated Chinese key pair label = %q", got)
	}
	if got := chineseNameForIdentifier("key_pair_name"); got != "密钥对名称" {
		t.Fatalf("generated Chinese key pair name label = %q", got)
	}
	if got := chineseNameForIdentifier("page_number"); got != "页码" {
		t.Fatalf("generated Chinese page number label = %q", got)
	}
	if got := chineseNameForIdentifier("subnet_id"); got != "子网 ID" {
		t.Fatalf("generated Chinese subnet label = %q", got)
	}
	if got := joinChineseLabelParts([]string{"云主机", "", "备份"}); got != "云主机备份" {
		t.Fatalf("generated Chinese label with empty fragment = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "unknownField"}, "Unknown Field"); got != "Unknown Field" {
		t.Fatalf("unknown Chinese column label = %q", got)
	}
	if got := columnIdentifier(Column{}); got != "value" {
		t.Fatalf("empty column identifier = %q", got)
	}
	if got := normalizeEnglishLabel("IPv6AddressID"); got != "IPv6 Address ID" {
		t.Fatalf("normalized acronym label = %q", got)
	}
	if got := identifierWords("disk2Name"); len(got) != 2 || got[0] != "disk2" || got[1] != "Name" {
		t.Fatalf("digit-to-letter identifier words = %#v", got)
	}
	if !shouldSplitIdentifierWord('J', 'S', 'o') {
		t.Fatalf("expected upper acronym boundary to split")
	}
	if got := englishWord(""); got != "" {
		t.Fatalf("empty English word = %q", got)
	}
}

func TestGeneratedChineseDescriptionPredicatesCoverFallbacks(t *testing.T) {
	if got := parameterLocalizedDescription(Parameter{Name: "displayName", Descriptions: map[string]string{"zh-CN": "显示名称"}}, "zh-CN"); got != "显示名称" {
		t.Fatalf("source zh-CN parameter description = %q", got)
	}
	if got := parameterLocalizedDescription(Parameter{Name: "displayName", Description: "显示名称"}, "zh-CN"); got != "显示名称" {
		t.Fatalf("fallback zh-CN parameter description = %q", got)
	}
	for _, description := range []string{
		"参考 https://example.com/doc",
		"命令类型：Shell",
		"这是一个超过二十四个字符的参数说明文本用于覆盖长描述分支",
	} {
		if !generatedChineseParameterDescription(description) {
			t.Fatalf("generatedChineseParameterDescription(%q) = false, want true", description)
		}
	}
	if generatedChineseParameterDescription("名称") {
		t.Fatal("generatedChineseParameterDescription(\"名称\") = true, want false")
	}
	for _, label := range []string{
		"您可以查看 产品定义",
		"参考 https://example.com/doc",
		"命令类型：Shell",
		"这是一个超过二十四个字符的表格列名用于覆盖长标签分支",
		"密钥 Pair 名称",
	} {
		if !generatedChineseColumnLabel(label) {
			t.Fatalf("generatedChineseColumnLabel(%q) = false, want true", label)
		}
	}
	if generatedChineseColumnLabel("实例ID") {
		t.Fatal("generatedChineseColumnLabel(\"实例ID\") = true, want false")
	}
	if got := canonicalTechnicalASCIIWord("custom"); got != "custom" {
		t.Fatalf("unknown technical word canonicalization = %q", got)
	}
}

func TestBuildWaitersRequiresInstanceShowStatus(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations[1].Response.RowPath = "returnObj.other"
	if got := buildWaiters(catalog).Waiters; len(got) != 0 {
		t.Fatalf("waiters with mismatched row path = %#v", got)
	}
	catalog = loadCatalogFixture(t)
	catalog.Operations[1].Response.Columns = nil
	if got := buildWaiters(catalog).Waiters; len(got) != 0 {
		t.Fatalf("waiters without status column = %#v", got)
	}
}
