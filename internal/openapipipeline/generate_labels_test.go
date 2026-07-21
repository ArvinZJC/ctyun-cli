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
	if got := chineseColumnLabel(Column{Key: "key_pair_name", LabelZH: "密钥 Pair 名称"}, "Key Pair Name"); got != "密钥 Pair 名称" {
		t.Fatalf("source Chinese column label with invalid English text = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "created_time", LabelZH: "Created Time"}, "Created Time"); got != "Created Time" {
		t.Fatalf("source Chinese label with untranslated English text = %q", got)
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

// TestChineseColumnLabelPreservesFILESETCasing verifies that a documented
// storage token remains an acronym inside localized table labels.
func TestChineseColumnLabelPreservesFILESETCasing(t *testing.T) {
	column := Column{Key: "fileset_status", LabelZH: "FILESET状态"}
	if got := englishColumnLabel(Column{Key: "fileset_status"}); got != "FILESET Status" {
		t.Fatalf("englishColumnLabel = %q, want %q", got, "FILESET Status")
	}
	if got := chineseColumnLabel(column, "FILESET Status"); got != "FILESET 状态" {
		t.Fatalf("chineseColumnLabel = %q, want %q", got, "FILESET 状态")
	}
}

func TestChineseColumnLabelPreservesLocalizedTechnicalTokens(t *testing.T) {
	tests := []struct {
		name         string
		column       Column
		englishLabel string
		want         string
	}{
		{
			name:         "leading uppercase acronym",
			column:       Column{Key: "cors_rules", LabelZH: "CORS 规则"},
			englishLabel: "Cors Rules",
			want:         "CORS 规则",
		},
		{
			name:         "trailing title case token",
			column:       Column{Key: "explicit_placement", LabelZH: "显式 Placement"},
			englishLabel: "Explicit Placement",
			want:         "显式 Placement",
		},
		{
			name:         "leading title case token",
			column:       Column{Key: "zonegroup", LabelZH: "Zone 组"},
			englishLabel: "Zonegroup",
			want:         "Zone 组",
		},
		{
			name:         "embedded title case intranet token",
			column:       Column{Key: "intranet_endpoint", LabelZH: "内网 Endpoint 列表"},
			englishLabel: "Intranet Endpoint",
			want:         "内网 Endpoint 列表",
		},
		{
			name:         "embedded title case internet token",
			column:       Column{Key: "internet_endpoint", LabelZH: "外网 Endpoint 列表"},
			englishLabel: "Internet Endpoint",
			want:         "外网 Endpoint 列表",
		},
		{
			name:         "trailing uppercase acronym",
			column:       Column{Key: "arn", LabelZH: "角色 ARN"},
			englishLabel: "Arn",
			want:         "角色 ARN",
		},
		{
			name:         "trailing uppercase acronym with compound key",
			column:       Column{Key: "role_arn", LabelZH: "角色 ARN"},
			englishLabel: "Role Arn",
			want:         "角色 ARN",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := chineseColumnLabel(test.column, test.englishLabel); got != test.want {
				t.Fatalf("chineseColumnLabel = %q, want %q", got, test.want)
			}
		})
	}
}

// TestEnglishColumnLabelPreservesObjectStorageAcronyms verifies that stable
// object-storage identifiers retain their published casing in generated
// English table labels.
func TestEnglishColumnLabelPreservesObjectStorageAcronyms(t *testing.T) {
	tests := map[string]string{
		"cors_rules": "CORS Rules",
		"cmk_uuid":   "CMK UUID",
		"arn":        "ARN",
		"role_arn":   "Role ARN",
		"acl_conf":   "ACL Conf",
	}
	for key, want := range tests {
		if got := englishColumnLabel(Column{Key: key}); got != want {
			t.Errorf("englishColumnLabel(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestDisplayLabelQualityRejectsUntranslatedEnglish(t *testing.T) {
	for _, label := range []string{
		"CORS Rules",
		"Explicit Placement",
		"Zone Group",
		"Intranet Endpoint List",
		"Role ARN",
	} {
		t.Run(label, func(t *testing.T) {
			if finding := DisplayLabelQualityFinding("zh-CN", label); finding == "" {
				t.Fatalf("DisplayLabelQualityFinding(%q) returned no finding", label)
			}
		})
	}
}

func TestGeneratedChineseDescriptionPredicatesCoverFallbacks(t *testing.T) {
	if got := parameterLocalizedDescription(Parameter{Name: "displayName", Descriptions: map[string]string{"zh-CN": "显示名称"}}, "zh-CN"); got != "显示名称" {
		t.Fatalf("source zh-CN parameter description = %q", got)
	}
	for _, description := range []string{
		"目标 NAT 网关规格",
		"目标备份存储库大小（GiB）",
		"目标共享带宽大小（Mbit/s）",
	} {
		parameter := Parameter{Name: "technicalValue", Descriptions: map[string]string{"zh-CN": description}}
		if got := parameterLocalizedDescription(parameter, "zh-CN"); got != description {
			t.Fatalf("technical zh-CN parameter description = %q, want %q", got, description)
		}
	}
	if got := parameterLocalizedDescription(Parameter{Name: "displayName", Description: "显示名称"}, "zh-CN"); got != "显示名称" {
		t.Fatalf("fallback zh-CN parameter description = %q", got)
	}
	for _, description := range []string{
		"参考 https://example.com/doc",
		"命令类型：Shell",
		"命令类型：脚本",
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
		if finding := DisplayLabelQualityFinding("zh-CN", label); finding == "" {
			t.Fatalf("DisplayLabelQualityFinding(%q) returned no finding", label)
		}
	}
	if finding := DisplayLabelQualityFinding("zh-CN", "实例ID"); finding != "" {
		t.Fatalf("DisplayLabelQualityFinding(\"实例ID\") = %q", finding)
	}
	if got := chineseColumnLabel(Column{Key: "total_price", LabelZH: "总价格（CNY）"}, "Total Price (CNY)"); got != "总价格（CNY）" {
		t.Fatalf("technical zh-CN column label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "max_iops", LabelZH: "最大 IOPS"}, "Max IOPS"); got != "最大 IOPS" {
		t.Fatalf("IOPS zh-CN column label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "paas", LabelZH: "是否支持PAAS"}, "PaaS"); got != "是否支持 PaaS" {
		t.Fatalf("PaaS zh-CN column label = %q", got)
	}
	if got := chineseColumnLabel(Column{Key: "pass", LabelZH: "是否支持PASS"}, "Pass"); got != "是否支持 PASS" {
		t.Fatalf("PASS zh-CN column label = %q", got)
	}
	if got := normalizeChineseTechnicalLabel("ID名称"); got != "ID 名称" {
		t.Fatalf("leading technical zh-CN label = %q", got)
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
