/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import "testing"

// TestNormalizeDisplayLabelCanonicalizesTechnicalTokens verifies that source
// wording is preserved while recognized technical tokens use canonical form.
func TestNormalizeDisplayLabelCanonicalizesTechnicalTokens(t *testing.T) {
	tests := []struct {
		language string
		label    string
		want     string
	}{
		{"en-US", "Cmk UUID", "CMK UUID"},
		{"en-US", "E Tag", "ETag"},
		{"en-US", "Etag", "ETag"},
		{"en-US", "Acl Enable", "ACL Enable"},
		{"en-US", "Ssekms Key ID", "SSE-KMS Key ID"},
		{"zh-CN", "HPFS文件系统共享路径", "HPFS 文件系统共享路径"},
		{"zh-CN", "加入AD域的操作号", "加入 AD 域的操作号"},
		{"zh-CN", "该分段数据对应Etag", "该分段数据对应 ETag"},
		{"zh-CN", "Windows共享路径", "Windows 共享路径"},
	}
	for _, test := range tests {
		if got := NormalizeDisplayLabel(test.language, test.label); got != test.want {
			t.Errorf("NormalizeDisplayLabel(%q, %q) = %q, want %q", test.language, test.label, got, test.want)
		}
	}
}

// TestChineseColumnLabelPreservesNonEmptySource verifies that generation does
// not replace concise localized evidence with an English fallback.
func TestChineseColumnLabelPreservesNonEmptySource(t *testing.T) {
	column := Column{Key: "share_path", LabelZH: "HPFS文件系统共享路径"}
	if got := chineseColumnLabel(column, "HPFS Share Path"); got != "HPFS 文件系统共享路径" {
		t.Fatalf("chineseColumnLabel = %q, want %q", got, "HPFS 文件系统共享路径")
	}
}

// TestEnglishColumnLabelCanonicalizesExplicitSource verifies that explicit
// English evidence uses the shared technical vocabulary.
func TestEnglishColumnLabelCanonicalizesExplicitSource(t *testing.T) {
	if got := englishColumnLabel(Column{LabelEN: "Cmk UUID"}); got != "CMK UUID" {
		t.Fatalf("englishColumnLabel = %q, want CMK UUID", got)
	}
}

// TestInvalidChineseSourceIsPreservedForReview verifies that generation does
// not hide invalid source evidence behind a translated fallback.
func TestInvalidChineseSourceIsPreservedForReview(t *testing.T) {
	tests := []struct {
		column Column
		want   string
	}{
		{Column{Key: "key_pair_name", LabelZH: "密钥 Pair 名称"}, "密钥 Pair 名称"},
		{Column{Key: "created_time", LabelZH: "Created Time"}, "Created Time"},
	}
	for _, test := range tests {
		if got := chineseColumnLabel(test.column, "fallback"); got != test.want {
			t.Errorf("chineseColumnLabel(%q) = %q, want %q", test.column.LabelZH, got, test.want)
		}
		if finding := DisplayLabelQualityFinding("zh-CN", test.want); finding == "" {
			t.Errorf("DisplayLabelQualityFinding(%q) returned no finding", test.want)
		}
	}
}

// TestDisplayLabelQualityAcceptsCloudTechnicalTokens verifies common products,
// formats, hashes, and protocol names remain usable inside Chinese labels.
func TestDisplayLabelQualityAcceptsCloudTechnicalTokens(t *testing.T) {
	for _, label := range []string{
		"HPC 型挂载密钥",
		"JSON 桶策略",
		"Keytab MD5",
		"S3 对象元数据",
		"SSE-KMS 密钥 ID",
		"HTTPS 证书信息",
		"CNAME 是否有效",
		"OpenAPI 可用",
		"ZOS 挂载配置",
		"RPO 过高数量",
		"NAS 配置",
		"网络收发包能力（万 PPS）",
		"基准带宽（Gbps）",
		"内存大小 MB",
		"SaaS",
		"CDA",
		"CPU 频率（GHz）",
		"内存大小（GB）",
		"内存频率（MHz）",
		"NVMe 硬盘数量",
		"RoCE 网卡速率（GE）",
		"RAID UUID",
		"IB 网卡数量",
		"XSSD 系列云硬盘",
		"MAC 地址",
		"OID",
		"DNAT 规则",
		"SNAT 规则",
		"VIP 地址",
	} {
		if finding := DisplayLabelQualityFinding("zh-CN", label); finding != "" {
			t.Errorf("DisplayLabelQualityFinding(%q) = %q", label, finding)
		}
	}
}

// TestDisplayLabelQualityAcceptsEnglishNounPhrases verifies length alone does
// not misclassify descriptive English headings as prose.
func TestDisplayLabelQualityAcceptsEnglishNounPhrases(t *testing.T) {
	for _, label := range []string{
		"Server Side Encryption Configuration",
		"Object Lock Retain Until Date",
		"Permission Group Description",
	} {
		if finding := DisplayLabelQualityFinding("en-US", label); finding != "" {
			t.Errorf("DisplayLabelQualityFinding(%q) = %q", label, finding)
		}
	}
}

// TestDisplayLabelQualityEdgeCases covers empty evidence, whole technical
// labels, non-canonical casing, and source-agreement boundaries.
func TestDisplayLabelQualityEdgeCases(t *testing.T) {
	if got := NormalizeDisplayLabel("zh-CN", "  "); got != "" {
		t.Fatalf("empty normalized label = %q", got)
	}
	if got := NormalizeDisplayLabel("zh-CN", "sse-kms"); got != "SSE-KMS" {
		t.Fatalf("whole technical label = %q, want SSE-KMS", got)
	}
	for _, test := range []struct {
		label string
		want  string
	}{
		{"", "is empty"},
		{"加入 ad 域", "uses non-canonical technical casing"},
		{"uuid", "uses non-canonical technical casing"},
	} {
		if got := DisplayLabelQualityFinding("zh-CN", test.label); got != test.want {
			t.Errorf("DisplayLabelQualityFinding(%q) = %q, want %q", test.label, got, test.want)
		}
	}
	if got := DisplayLabelQualityFinding("zh-CN", "SSE-KMS"); got != "" {
		t.Fatalf("whole technical quality finding = %q", got)
	}
	if got := DisplayLabelAgreementFinding("zh-CN", "", "任意值"); got != "" {
		t.Fatalf("empty source agreement finding = %q", got)
	}
	if isTechnicalOnlyLabel("中文") {
		t.Fatal("Chinese text classified as a technical-only label")
	}
}

// TestChineseNameForIdentifierCoversCatalogRepairPhrases verifies that shared
// response concepts receive stable localized suggestions across products.
func TestChineseNameForIdentifierCoversCatalogRepairPhrases(t *testing.T) {
	tests := map[string]string{
		"execution_mode":                 "执行方式",
		"security_group_id_list":         "安全组 ID 列表",
		"reserved_concurrency":           "预留并发数",
		"max_async_event_age_in_seconds": "异步事件最大存活时间",
		"dedicated_host_resource_id":     "专属宿主机资源 ID",
		"auto_renew_cycle_type":          "自动续订周期类型",
		"cluster_type_name":              "集群类型名称",
		"backup_restore_done_time":       "备份恢复完成时间",
		"status_code":                    "状态码",
		"openapi_available":              "OpenAPI 可用",
		"numa_node_amount":               "单个 CPU NUMA 节点数量",
		"system_volume_interface":        "系统盘接口",
		"enable_shadow_vpc":              "存储高速网络支持",
		"return_obj":                     "返回结果",
		"on_demand":                      "是否按需付费",
		"name_zh":                        "中文名称",
		"layout_type":                    "布局类型",
		"batch_order_placement_results":  "订单结果",
		"concurrent_mode":                "并发模式",
		"snapshot_time_points":           "快照时间点",
		"device_private_ip_address":      "设备私网 IP 地址",
		"port_sg_list":                   "网卡安全组列表",
		"open_sms_switch":                "是否开启短信告警",
		"deduction_start_time":           "核时开始时间",
	}
	for identifier, want := range tests {
		if got := chineseNameForIdentifier(identifier); got != want {
			t.Errorf("chineseNameForIdentifier(%q) = %q, want %q", identifier, got, want)
		}
	}
}
