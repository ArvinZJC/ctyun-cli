/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"path/filepath"
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
	for _, want := range []string{"List ECS instances", "Product: Elastic Cloud Server", "ctyun ecs instance list", "https://"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
	if first := firstNonEmptyLine(got); first != "List ECS instances." {
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
		"ctyun [global options] ecs <subcommand>",
		"ctyun help ecs <subcommand>",
		"Subcommands:",
		"instance",
		"Global Options:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("plugin help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "region list") {
		t.Fatalf("plugin help output exposed unrelated commands:\n%s", got)
	}
	for _, unwanted := range []string{"Commands:", "Examples:", "[options]", "[command options]", "ctyun help ecs instance list", "Available Commands:"} {
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
		"ctyun [global options] ecs instance <subcommand>",
		"ctyun help ecs instance <subcommand>",
		"Subcommands:",
		"describe-instances",
		"list",
		"List ECS instances",
		"show",
		"Show an ECS instance",
		"start",
		"Global Options:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("nested plugin help output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"Commands:", "Examples:", "[options]", "[command options]", "ecs instance show {instance_id}", "Available Commands:"} {
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
		"Show an ECS instance.",
		"Product: Elastic Cloud Server",
		"Usage:",
		"ctyun [global options] ecs instance show {instance_id}",
		"Arguments:",
		"{instance_id}  Instance ID",
		"Global Options:",
		"Examples:",
		"ctyun ecs instance show c5a7966a-88e7-362b-6e11-c2d8fbfc07ca",
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
		"ctyun [global options] ecs instance list",
		"Command Options:",
		"--name <value>",
		"Filter by exact instance name",
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
		"ctyun [global options] region demand check [{region_id}] --product-type <ecs|eip|ebs> [--zone <value>]",
		"Arguments:",
		"{region_id}  Region ID",
		"Command Options:",
		"Region ID",
		"--product-type <ecs|eip|ebs>",
		"Product type to check (required)",
		"--zone <value>",
		"Availability zone name",
		"--ebs-type <SATA|SAS|SSD|SATA-KUNPENG|SATA-HAIGUANG|SAS-KUNPENG|SAS-HAIGUANG>",
		"EBS disk type",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("required option help output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"[command options]", "[one of ecs,eip,ebs]", "[one of SATA,SAS", "--region <region>"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("required option help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestHelpShowsDeprecatedCommandOptionAndColumnMetadata(t *testing.T) {
	root := t.TempDir()
	writeDeprecatedWarningBundle(t, filepath.Join(root, "demo"))

	var stdout bytes.Buffer
	if err := runHelp(&stdout, []string{"demo", "list"}, root, "en-US"); err != nil {
		t.Fatalf("runHelp returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"Deprecated command.",
		"Use ctyun demo new-list.",
		"Deprecated CTyun API.",
		"--page <value>",
		"deprecated",
		"Old Size",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("deprecated help output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{
		"old command",
		"newer demo command",
		"old API",
		"old page field",
		"--page-no",
		"old response field",
		"remainingSize",
		"upstream notice",
		"replacement:",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("deprecated help output exposed upstream metadata %q:\n%s", unwanted, got)
		}
	}
	if strings.Contains(got, "demo.list") || strings.Contains(got, "v4.demo.list") {
		t.Fatalf("deprecated help output exposed internal IDs:\n%s", got)
	}
}

func TestHelpUsesFieldsHeadingForVerticalPluginTables(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "region", "product", "show"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Fields:") {
		t.Fatalf("vertical table help output missing Fields heading:\n%s", got)
	}
	if strings.Contains(got, "Columns:") {
		t.Fatalf("vertical table help output should not use Columns heading:\n%s", got)
	}
	if !strings.Contains(got, "-c, --cols <keys>") || !strings.Contains(got, "columns or fields") {
		t.Fatalf("global column selector help did not mention fields:\n%s", got)
	}
	if strings.Contains(got, "(default)") {
		t.Fatalf("vertical table help rendered default as part of the field name:\n%s", got)
	}
	if !helpLineHasMarker(got, "ECS Flavor Types", "default") {
		t.Fatalf("vertical table help did not mark default fields:\n%s", got)
	}
	if helpLineHasMarker(got, "ECS", "default") {
		t.Fatalf("vertical table help marked non-default object field:\n%s", got)
	}

	stdout.Reset()
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "region", "resource", "show"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("resource help returned error: %v", err)
	}
	got = stdout.String()
	if !strings.Contains(got, "Fields:\n  ACL Summary") || !helpLineHasMarker(got, "ECS Total", "default") {
		t.Fatalf("vertical table help did not render fields one per line:\n%s", got)
	}
	if strings.Contains(got, "ECS Summary,Volume Summary") {
		t.Fatalf("vertical table help rendered fields as a comma list:\n%s", got)
	}
}

func TestHelpUsesOneSelectorPerLineForHorizontalPluginTables(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "region", "list"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Columns:\n  ") {
		t.Fatalf("horizontal table help output missing Columns heading:\n%s", got)
	}
	if !strings.Contains(got, "  Region ID") || !strings.Contains(got, "  Region Name") {
		t.Fatalf("horizontal table help did not render columns one per line:\n%s", got)
	}
	if strings.Contains(got, "(default)") {
		t.Fatalf("horizontal table help rendered default as part of the column name:\n%s", got)
	}
	if !helpLineHasMarker(got, "Region ID", "default") {
		t.Fatalf("horizontal table help did not mark default columns:\n%s", got)
	}
	columnsSection := strings.Split(strings.Split(got, "Columns:\n")[1], "\nExamples:")[0]
	if strings.Contains(columnsSection, ",") {
		t.Fatalf("horizontal table help rendered column section as a comma list:\n%s", got)
	}

	stdout.Reset()
	if err := Run(Config{
		Args:   []string{"region", "list", "-h", "-l", "zh"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("localized help returned error: %v", err)
	}
	got = stdout.String()
	markerColumns := helpMarkerColumns(got, "列:", "默认")
	if len(markerColumns) < 2 {
		t.Fatalf("localized column help did not render enough default markers:\n%s", got)
	}
	for _, column := range markerColumns[1:] {
		if column != markerColumns[0] {
			t.Fatalf("localized default markers are not aligned: %v\n%s", markerColumns, got)
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
	for _, want := range []string{"List ECS instances", "Filter by exact instance name", "Render output as a table or raw JSON", "Show help for the command", "Columns:\n  Instance ID"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing sentence-case text %q:\n%s", want, got)
		}
	}
	if !helpLineHasMarker(got, "Instance ID", "default") || !helpLineHasMarker(got, "Status", "default") {
		t.Fatalf("help output did not mark implicit default columns:\n%s", got)
	}
}

func helpLineHasMarker(text, name, marker string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, name) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, name))
		if rest == marker {
			return true
		}
	}
	return false
}

func helpMarkerColumns(text, heading, marker string) []int {
	inSection := false
	var columns []int
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == heading {
			inSection = true
			continue
		}
		if inSection && strings.TrimSpace(line) == "" {
			break
		}
		if !inSection {
			continue
		}
		index := strings.LastIndex(line, marker)
		if index < 0 {
			continue
		}
		columns = append(columns, helpDisplayWidth(line[:index]))
	}
	return columns
}

func TestProductCommandOutputControlsAcceptLocalizedColumnLabels(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", "zh-CN", "--table", "plain", "--cols", "实例 ID,名称", "--filter", "名称=api-test01", "--sort", "实例 ID", "--no-header", "ecs", "instance", "list", "--offline"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("product command with localized output controls returned error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "c5a7966a-88e7-362b-6e11-c2d8fbfc07ca") || !strings.Contains(got, "api-test01") {
		t.Fatalf("localized output controls missing expected row:\n%s", got)
	}
	if strings.Contains(got, "状态") {
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
	for _, want := range []string{"弹性云主机", "查询云主机列表", "命令选项:", "全局选项:", "--name <value>", "按云主机名称过滤", "列:\n  云主机名称"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if !helpLineHasMarker(got, "云主机名称", "默认") || !helpLineHasMarker(got, "状态", "默认") {
		t.Fatalf("localized help output did not mark implicit default columns:\n%s", got)
	}
	if strings.Contains(got, "Filter by exact instance name") || strings.Contains(got, "matches ^") {
		t.Fatalf("localized help output still contains English option description:\n%s", got)
	}
}
