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
