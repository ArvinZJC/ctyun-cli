/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestBuildCommandsResolvesParameterReplacementToVisibleOption(t *testing.T) {
	notice := "是否是云上资源。该参数后续即将下线，推荐使用sourceType参数"
	catalog := Catalog{
		Product: Product{PluginName: "cbr"},
		Operations: []Operation{{
			ID: "v4.cbr.repository.create",
			Parameters: []Parameter{
				{Name: "isOnDemand", CLIName: "is_on_demand", CLIFlag: "is-on-demand", Description: notice},
				{Name: "sourceType", CLIName: "source_type", CLIFlag: "source-type"},
			},
		}},
	}

	deprecation := buildCommands(catalog).Commands[0].Parameters[0].Deprecation
	if deprecation == nil || deprecation.Notice != notice || deprecation.Replacement == nil {
		t.Fatalf("parameter deprecation = %#v, want preserved notice and replacement", deprecation)
	}
	if got := *deprecation.Replacement; got.Kind != "option" || got.Label != "--source-type" {
		t.Fatalf("parameter replacement = %#v, want visible --source-type option", got)
	}
}

func TestBuildTablesKeepsResolvedSiblingFieldReplacement(t *testing.T) {
	notice := "主机ip。推荐使用instanceIps参数，该参数后续即将下线"
	table := generatedDeprecationTable(notice)

	deprecation := table.Columns[0].Deprecation
	if deprecation == nil || deprecation.Notice != notice || deprecation.Replacement == nil {
		t.Fatalf("column deprecation = %#v, want preserved notice and replacement", deprecation)
	}
	if got := *deprecation.Replacement; got.Kind != "field" || got.Label != "instanceIps" {
		t.Fatalf("column replacement = %#v, want resolved instanceIps field", got)
	}
}

func TestBuildTablesKeepsResolvedSiblingColumnKeyReplacement(t *testing.T) {
	notice := "主机ip。推荐使用instance_ips字段，该参数后续即将下线"
	table := generatedDeprecationTable(notice)

	deprecation := table.Columns[0].Deprecation
	if deprecation == nil || deprecation.Replacement == nil {
		t.Fatalf("column deprecation = %#v, want replacement", deprecation)
	}
	if got := deprecation.Replacement.Label; got != "instance_ips" {
		t.Fatalf("column replacement label = %q, want resolved instance_ips key", got)
	}
}

func TestBuildTablesOmitsUnresolvedFieldReplacement(t *testing.T) {
	notice := "主机ip。推荐使用backupStorageCreateTime参数，该参数后续即将下线"
	table := generatedDeprecationTable(notice)

	deprecation := table.Columns[0].Deprecation
	if deprecation == nil || deprecation.Notice != notice {
		t.Fatalf("column deprecation = %#v, want preserved notice", deprecation)
	}
	if deprecation.Replacement != nil {
		t.Fatalf("column replacement = %#v, want unresolved replacement omitted", deprecation.Replacement)
	}
}

// TestBuildAPIsOmitsGenericNewVersionReplacement verifies category-wide
// lifecycle guidance stays as a notice without inventing a structured API
// target.
func TestBuildAPIsOmitsGenericNewVersionReplacement(t *testing.T) {
	const notice = "当前页面接口为旧版 API，未来根据实际使用情况可能退役，推荐使用新版本接口，新版本接口更加规范，覆盖场景更全。"
	catalog := Catalog{
		Product: Product{PluginName: "vbs"},
		Operations: []Operation{{
			ID:          "v4.vbs.backup.legacy-list",
			Description: map[string]string{"zh-CN": notice},
		}},
	}

	deprecation := buildAPIs(catalog).Operations["v4.vbs.backup.legacy-list"].Deprecation
	if deprecation == nil || deprecation.Notice != notice {
		t.Fatalf("API deprecation = %#v, want preserved lifecycle notice", deprecation)
	}
	if deprecation.Replacement != nil {
		t.Fatalf("API replacement = %#v, want generic guidance omitted", deprecation.Replacement)
	}
}

func generatedDeprecationTable(notice string) plugin.Table {
	catalog := Catalog{
		Product: Product{PluginName: "cbr"},
		Operations: []Operation{{
			ID:       "v4.cbr.repository.list",
			Category: "repository",
			Response: Response{Columns: []Column{
				{Key: "host_ip", Path: "hostIp", Description: notice},
				{Key: "instance_ips", Path: "instanceIps"},
			}},
		}},
	}
	return buildTables(catalog).Tables["cbr.repository.list"]
}
