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
		"Usage:",
		"ctyun ecs <subcommand> [command options]",
		"ctyun help ecs <subcommand>",
		"Subcommands:",
		"instance  List cloud servers",
		"Global Options:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plugin help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "region list") {
		t.Fatalf("plugin help output exposed unrelated commands:\n%s", got)
	}
	for _, unwanted := range []string{"Commands:", "Examples:", "ctyun help ecs instance list", "Available Commands:"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("plugin help output contains %q:\n%s", unwanted, got)
		}
	}
	if strings.Contains(got, "ecs server ls") {
		t.Fatalf("plugin help output exposed unsupported alias:\n%s", got)
	}
}

func TestHelpNestedPrefixListsMatchingPluginSubcommands(t *testing.T) {
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
		"Usage:",
		"ctyun ecs instance <subcommand> [command options]",
		"ctyun help ecs instance <subcommand>",
		"Subcommands:",
		"list   List cloud servers",
		"show   Show cloud server details",
		"start  Start a cloud server",
		"Global Options:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("nested plugin help output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Commands:", "Examples:", "ecs instance show {instance_id}", "Available Commands:"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("nested plugin help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestHelpWithOmittedTrailingPathArgumentUsesCommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "help", "ecs", "instance", "show"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Show cloud server details.",
		"Product: Elastic Cloud Server",
		"Usage:",
		"ctyun [global options] ecs instance show {instance_id}",
		"Arguments:",
		"{instance_id}  Instance ID",
		"Global Options:",
		"Examples:",
		"ctyun ecs instance show ins-demo-1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("placeholder command help output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Commands:", "Subcommands:", "[command options]", "ctyun help ecs instance show {instance_id}"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("placeholder command help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestHelpCommandUsageIncludesOptionsOnlyWhenDeclared(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "en-US", "help", "ecs", "instance", "list"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"ctyun [global options] ecs instance list [--name <name>]",
		"Command Options:",
		"--name <name>  Filter by instance name",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("command option help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Arguments:") {
		t.Fatalf("command without path arguments rendered argument help:\n%s", got)
	}
}

func TestHelpCommandUsageMarksRequiredAndOptionalOptions(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "region", "demand", "check"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"ctyun [global options] region demand check {region_id} --product-type <ecs|eip|ebs> [--zone <zone>]",
		"Arguments:",
		"{region_id}  Region ID",
		"Command Options:",
		"--product-type <ecs|eip|ebs>",
		"Product type to check (required)",
		"--zone <zone>",
		"Availability zone name",
		"--ebs-type <SATA|SAS|SSD|SATA-KUNPENG|SATA-HAIGUANG|SAS-KUNPENG|SAS-HAIGUANG>",
		"EBS disk type",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("required option help output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"[command options]", "[one of ecs,eip,ebs]", "[one of SATA,SAS"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("required option help output contains %q:\n%s", unwanted, got)
		}
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
		"Subcommands:",
		"list",
		"show",
		"start",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("nested plugin help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Commands:") {
		t.Fatalf("plugin help output contains old command heading:\n%s", got)
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
	for _, want := range []string{"弹性云主机", "列出云主机", "命令选项:", "全局选项:", "--name <name>  按云主机名称过滤", "[匹配 ^[A-Za-z0-9._-]+$]", "实例ID,名称,状态,私有IP"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Filter by instance name") || strings.Contains(got, "matches ^") {
		t.Fatalf("localized help output still contains English option description:\n%s", got)
	}
}
