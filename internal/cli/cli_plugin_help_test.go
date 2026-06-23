/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpCommandUsesPluginMetadata(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"help", "ecs", "instance", "list"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"List cloud servers", "Product: Elastic Cloud Server", "ctyun ecs instance list", "https://"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
	if first := firstNonEmptyLine(got); first != "List cloud servers." {
		t.Fatalf("plugin command help first line = %q", first)
	}
	for _, unwanted := range []string{"ecs.instance.list", "Description:", "ctyun ecs server ls"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestHelpPluginPrefixListsPluginCommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "help", "ecs"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Elastic Cloud Server",
		"Commands:",
		"ecs instance list                List cloud servers",
		"ecs instance show {instance_id}  Show cloud server details",
		"ctyun help ecs instance list",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plugin help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "region list") {
		t.Fatalf("plugin help output exposed unrelated commands:\n%s", got)
	}
	if strings.Contains(got, "Available Commands:") {
		t.Fatalf("plugin help output contains old command heading:\n%s", got)
	}
	if strings.Contains(got, "ecs server ls") {
		t.Fatalf("plugin help output exposed unsupported alias:\n%s", got)
	}
}

func TestHelpNestedPrefixListsMatchingPluginCommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "help", "ecs", "instance"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Commands:",
		"ecs instance list",
		"ecs instance show {instance_id}",
		"ecs instance start {instance_id}",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("nested plugin help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Available Commands:") {
		t.Fatalf("nested plugin help output contains old command heading:\n%s", got)
	}
}

func TestHelpUsesSentenceCaseForEnglishDescriptions(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "help", "ecs", "instance", "list"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"List cloud servers", "Filter by instance name", "Render output as a table or raw JSON", "Show help for the command", "Instance ID,Name,Status,Private IP"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing sentence-case text %q:\n%s", want, got)
		}
	}
}

func TestProductCommandOutputControlsAcceptLocalizedColumnLabels(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "--offline", "--table", "plain", "--cols", "实例ID,名称", "--filter", "名称=demo-web", "--sort", "实例ID", "--no-header", "ecs", "instance", "list"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("product command with localized output controls returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "ins-demo-1") || !strings.Contains(got, "demo-web") {
		t.Fatalf("localized output controls missing expected row:\n%s", got)
	}
	if strings.Contains(got, "ins-demo-2") || strings.Contains(got, "状态") {
		t.Fatalf("localized output controls did not filter/select columns:\n%s", got)
	}
}

func TestHelpCommandUsesPluginI18N(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "help", "ecs", "instance", "list"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"弹性云主机", "列出云主机", "命令选项:", "全局选项:", "--name <value>  按云主机名称过滤", "[匹配 ^[A-Za-z0-9._-]+$]", "实例ID,名称,状态,私有IP"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Filter by instance name") || strings.Contains(got, "matches ^") {
		t.Fatalf("localized help output still contains English option description:\n%s", got)
	}
}
