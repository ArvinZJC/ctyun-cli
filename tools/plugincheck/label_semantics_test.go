/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
)

// TestProductLabelsPreserveResourceMeaning prevents unrelated product terminology from leaking through shared field names.
func TestProductLabelsPreserveResourceMeaning(t *testing.T) {
	for _, tc := range []struct{ product, operation, field, english, chinese string }{
		{"nat", "nat.gateway.show", "name", "Name", "名称"},
		{"private-nat", "private-nat.gateway.list", "name", "Name", "名称"},
		{"eip", "eip.show", "name", "Name", "名称"},
		{"as", "v4.as.rule.scheduled.list", "name", "Name", "名称"},
		{"rds-postgresql", "rds-postgresql.instance.health.show", "name", "Name", "名称"},
		{"redis", "redis.tag.resource.list", "resourceCount", "Bound Resource Count", "绑定资源数"},
		{"redis", "redis.audit.export.list", "format", "File Format", "文件格式"},
		{"redis", "redis.key.eviction-policy.show", "policy", "Eviction Policy", "淘汰策略"},
		{"mongodb", "mongodb.backup.cross-region.policy.show", "history", "Historical Data Sync Enabled", "是否同步历史数据"},
	} {
		t.Run(tc.operation+"/"+tc.field, func(t *testing.T) {
			data, err := os.ReadFile(repoPath(t, "openapi-catalogs/"+tc.product+"/baseline.json"))
			if err != nil {
				t.Fatal(err)
			}
			var catalog openapipipeline.Catalog
			if err = json.Unmarshal(data, &catalog); err != nil {
				t.Fatal(err)
			}
			for _, op := range catalog.Operations {
				if op.ID != tc.operation {
					continue
				}
				for _, c := range op.Response.Columns {
					if c.Path == tc.field {
						if c.LabelEN != tc.english || c.LabelZH != tc.chinese {
							t.Fatalf("got %q / %q, want %q / %q", c.LabelEN, c.LabelZH, tc.english, tc.chinese)
						}
						return
					}
				}
			}
			t.Fatal("missing operation or field")
		})
	}
}
