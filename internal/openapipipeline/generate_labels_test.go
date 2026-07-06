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
