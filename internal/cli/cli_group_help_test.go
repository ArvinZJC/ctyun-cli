/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestPluginCommandGroupHelpUsesGroupDescriptions(t *testing.T) {
	bundle := plugin.Bundle{
		Manifest: plugin.Manifest{Name: "region"},
		Commands: plugin.Commands{Commands: []plugin.Command{
			{ID: "region.list", Path: []string{"region", "list"}},
			{ID: "region.show", Path: []string{"region", "show", "{region_id}"}},
			{ID: "region.zone.list", Path: []string{"region", "zone", "list", "{region_id}"}},
		}},
		I18N: map[string]map[string]string{
			"en-US": {
				"name":                                 "Region",
				"command.region.list.description":      "List visible resource pools",
				"command.region.show.description":      "Show resource pool summary",
				"command.region.zone.list.description": "List resource pool zones",
			},
			"zh-CN": {
				"name":                                 "资源池",
				"command.region.list.description":      "列出可见资源池",
				"command.region.show.description":      "显示资源池概要",
				"command.region.zone.list.description": "列出资源池可用区",
			},
		},
	}

	var root bytes.Buffer
	if err := printPluginCommandIndex(&root, bundle, []string{"region"}, bundle.Commands.Commands, "en-US"); err != nil {
		t.Fatalf("printPluginCommandIndex root returned error: %v", err)
	}
	rootHelp := root.String()
	if first := firstNonEmptyLine(rootHelp); first != "Manage Region commands." {
		t.Fatalf("root group first line = %q\n%s", first, rootHelp)
	}
	for _, want := range []string{
		"list  List visible resource pools",
		"show  Show resource pool summary",
		"zone  Show Region zone subcommands",
	} {
		if !strings.Contains(rootHelp, want) {
			t.Fatalf("root group help missing %q:\n%s", want, rootHelp)
		}
	}

	var zone bytes.Buffer
	if err := printPluginCommandIndex(&zone, bundle, []string{"region", "zone"}, []plugin.Command{bundle.Commands.Commands[2]}, "en-US"); err != nil {
		t.Fatalf("printPluginCommandIndex zone returned error: %v", err)
	}
	zoneHelp := zone.String()
	if first := firstNonEmptyLine(zoneHelp); first != "Manage Region zone commands." {
		t.Fatalf("zone group first line = %q\n%s", first, zoneHelp)
	}
	if !strings.Contains(zoneHelp, "list  List resource pool zones") {
		t.Fatalf("zone group help missing leaf description:\n%s", zoneHelp)
	}

	var rootZH bytes.Buffer
	if err := printPluginCommandIndex(&rootZH, bundle, []string{"region"}, bundle.Commands.Commands, "zh-CN"); err != nil {
		t.Fatalf("printPluginCommandIndex zh-CN root returned error: %v", err)
	}
	rootZHHelp := rootZH.String()
	if first := firstNonEmptyLine(rootZHHelp); first != "管理资源池命令。" {
		t.Fatalf("zh-CN root group first line = %q\n%s", first, rootZHHelp)
	}
	if !strings.Contains(rootZHHelp, "zone  查看资源池 zone 子命令") {
		t.Fatalf("zh-CN root group help has awkward spacing:\n%s", rootZHHelp)
	}
}
