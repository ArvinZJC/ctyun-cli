/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import "testing"

// TestResourceOfflineTextDoesNotDeprecateFields distinguishes operational
// disconnection and state values from retirement of an API field itself.
func TestResourceOfflineTextDoesNotDeprecateFields(t *testing.T) {
	for _, text := range []string{
		"本参数表示设备状态 取值范围: online:上线 offline:下线",
		"本参数表示状态。取值范围： 0：上线。 1：下线。 根据以上范围取值",
		"调度器状态：down：节点宕机 drain：节点已下线 fail：节点失效",
		"之前的登录将被强制下线。默认值为true。",
		"是否开启无损上下线 0不开启 1开启",
	} {
		if got := deprecationFromTexts("field", text, nil); got != nil {
			t.Errorf("resource state marked deprecated: %q: %#v", text, got)
		}
		if got := deprecationFromTexts("field", text+" 该字段未来将会下线。", nil); got == nil {
			t.Errorf("explicit retirement lost for %q", text)
		}
	}
}

// TestEnumMemberRetirementLeavesFieldUsable distinguishes a resource state or
// a retiring enum action from a notice that the whole field is being retired.
func TestEnumMemberRetirementLeavesFieldUsable(t *testing.T) {
	for _, tc := range []struct{ kind, text string }{
		{"field", "规格状态normal（正常使用）、abandon（下线）、sellout（售罄）"},
		{"parameter", "域名操作动作，取值范围：1-创建（固定值），2-修改（该功能即将下线，若需修改域名请使用增量修改域名配置接口）"},
	} {
		if got := deprecationFromTexts(tc.kind, tc.text, nil); got != nil {
			t.Errorf("enum member retired its containing %s: %#v", tc.kind, got)
		}
		if got := deprecationFromTexts(tc.kind, tc.text+"。该字段即将下线。", nil); got == nil {
			t.Errorf("explicit %s retirement was lost", tc.kind)
		}
	}
}
