/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/output"
)

func TestMainHelpShowsDescriptionCommandsAndGlobalOptions(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"ctyun is an unofficial, plugin-based CLI for CTyun.",
		"It prioritizes terminal-friendly cloud workflows because CTyun has no official CLI.",
		"Core Commands:",
		"Plugin Commands:",
		"ctyun plugin list",
		"ctyun plugins list",
		"ctyun help <plugin>",
		"plugin|plugins",
		"update|upgrade",
		"For plugin updates, run ctyun plugin|plugins update|upgrade",
		"Global Options:",
		"-o, --output <table|json>",
		"-l, --lang, --language <locale>",
		"-t, --table <bordered|compact|plain>",
		"-C, --config <path>",
		"-P, --profile <name>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
	coreDescriptionColumn := -1
	inCoreCommands := false
	for _, line := range strings.Split(got, "\n") {
		if line == "Core Commands:" {
			inCoreCommands = true
			continue
		}
		if inCoreCommands && line == "" {
			break
		}
		if !inCoreCommands || !strings.HasPrefix(line, "  ") {
			continue
		}
		column := helpDescriptionColumn(line)
		if column < 0 {
			t.Fatalf("core help line has no description column: %q", line)
		}
		if coreDescriptionColumn < 0 {
			coreDescriptionColumn = column
			continue
		}
		if column != coreDescriptionColumn {
			t.Fatalf("core help descriptions are not aligned:\n%s", got)
		}
	}
	if first := firstNonEmptyLine(got); first != "ctyun is an unofficial, plugin-based CLI for CTyun." {
		t.Fatalf("main help first line = %q", first)
	}
	for _, unwanted := range []string{"ctyun - plugin-based CTyun CLI", "Description:", "Product Commands:", "Available Plugins:", "Elastic Cloud Server", "region  Region", "ecs instance list", "O&M", "live retrieval", "--offline", "--fixture", "plugin and plugins are equivalent", "plugin, plugins", "upgrade, update"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestMainHelpUsesI18N(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "zh-CN", "help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"目前天翼云没有官方 CLI", "核心命令:", "插件命令:", "全局选项:", "选择帮助和输出语言"} {
		if !strings.Contains(got, want) {
			t.Fatalf("localized help output missing %q:\n%s", want, got)
		}
	}
	if first := firstNonEmptyLine(got); first != "ctyun 是非官方插件化天翼云 CLI。" {
		t.Fatalf("localized main help first line = %q", first)
	}
	if strings.Contains(got, "Global Options") || strings.Contains(got, "Description:") || strings.Contains(got, "产品命令:") || strings.Contains(got, "可用插件:") || strings.Contains(got, "描述:") {
		t.Fatalf("localized help output still contains English section text:\n%s", got)
	}
}

func TestHelpFlagShowsCommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "ecs", "instance", "list", "--help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("command help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"List cloud servers", "Product: Elastic Cloud Server", "Command Options:", "--name <value>", "Global Options:", "ctyun ecs instance list --cols \"Instance ID,Name,Status\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("command help output missing %q:\n%s", want, got)
		}
	}
	if first := firstNonEmptyLine(got); first != "List cloud servers." {
		t.Fatalf("command help first line = %q", first)
	}
	for _, unwanted := range []string{"ecs.instance.list", "Description:", "--offline", "--fixture", "--cols instance_id,name,status"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("command help output contains %q:\n%s", unwanted, got)
		}
	}
}

func TestCommandHelpExamplesUseLocalizedColumnLabels(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		want     string
		unwanted string
	}{
		{
			name:     "english",
			args:     []string{"--lang", "en-US", "help", "region", "list"},
			want:     `ctyun region list --name 华东1 --cols "Region ID,Region Name,Region Code"`,
			unwanted: "--cols region_id,region_name,region_code",
		},
		{
			name:     "chinese",
			args:     []string{"--lang", "zh-CN", "help", "region", "list"},
			want:     "ctyun region list --name 华东1 --cols 资源池ID,资源池名称,地域编号",
			unwanted: "--cols region_id,region_name,region_code",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := Run(Config{Args: tc.args, Stdout: &stdout}); err != nil {
				t.Fatalf("command help returned error: %v", err)
			}
			got := stdout.String()
			if !strings.Contains(got, tc.want) {
				t.Fatalf("command help output missing localized example %q:\n%s", tc.want, got)
			}
			if strings.Contains(got, tc.unwanted) {
				t.Fatalf("command help output contains raw-key example %q:\n%s", tc.unwanted, got)
			}
		})
	}
}

func TestLocalizedExampleSelectorsCoversTableControlForms(t *testing.T) {
	columns := []output.Column{
		{Key: "instance_id", Label: "Instance ID"},
		{Key: "status", Label: "Status"},
	}
	for _, tc := range []struct {
		name    string
		example string
		columns []output.Column
		want    string
	}{
		{name: "no columns", example: "ctyun ecs instance list --cols instance_id", want: "ctyun ecs instance list --cols instance_id"},
		{name: "filter", example: "ctyun ecs instance list --filter status=running", columns: columns, want: "ctyun ecs instance list --filter Status=running"},
		{name: "sort descending", example: "ctyun ecs instance list --sort -instance_id", columns: columns, want: `ctyun ecs instance list --sort "-Instance ID"`},
		{name: "cols equals", example: "ctyun ecs instance list --cols=instance_id,status", columns: columns, want: `ctyun ecs instance list --cols="Instance ID,Status"`},
		{name: "filter equals", example: "ctyun ecs instance list --filter=instance_id=ins-demo-1", columns: columns, want: `ctyun ecs instance list --filter="Instance ID=ins-demo-1"`},
		{name: "sort equals", example: "ctyun ecs instance list --sort=status", columns: columns, want: "ctyun ecs instance list --sort=Status"},
		{name: "filter without operator", example: "ctyun ecs instance list --filter status", columns: columns, want: "ctyun ecs instance list --filter status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := localizedExampleSelectors(tc.example, tc.columns); got != tc.want {
				t.Fatalf("localizedExampleSelectors() = %q, want %q", got, tc.want)
			}
		})
	}
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
	return ""
}

func TestHelpPageLeadDescriptionsEndWithPunctuation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		language string
	}{
		{name: "completion", args: []string{"help", "completion"}, language: "en-US"},
		{name: "help", args: []string{"help", "help"}, language: "en-US"},
		{name: "version", args: []string{"help", "version"}, language: "en-US"},
		{name: "update", args: []string{"help", "update"}, language: "en-US"},
		{name: "config", args: []string{"help", "config"}, language: "en-US"},
		{name: "config-show", args: []string{"help", "config", "show"}, language: "en-US"},
		{name: "config-profile", args: []string{"help", "config", "profile"}, language: "en-US"},
		{name: "config-profile-set-secret", args: []string{"help", "config", "profile", "set-secret"}, language: "en-US"},
		{name: "doctor", args: []string{"help", "doctor"}, language: "en-US"},
		{name: "doctor-network", args: []string{"help", "doctor", "network"}, language: "en-US"},
		{name: "plugin", args: []string{"help", "plugin"}, language: "en-US"},
		{name: "plugin-install", args: []string{"help", "plugin", "install"}, language: "en-US"},
		{name: "plugin-reinstall", args: []string{"help", "plugin", "reinstall"}, language: "en-US"},
		{name: "product-command", args: []string{"ecs", "instance", "list", "--help"}, language: "en-US"},
		{name: "config-zh", args: []string{"help", "config"}, language: "zh-CN"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			args := append([]string{"--lang", tc.language}, tc.args...)
			if err := Run(Config{Args: args, Stdout: &stdout}); err != nil {
				t.Fatalf("help returned error: %v", err)
			}
			first := firstNonEmptyLine(stdout.String())
			if !strings.ContainsRune(".。!?！？", lastHelpRune(first)) {
				t.Fatalf("help first line has no sentence punctuation: %q\n%s", first, stdout.String())
			}
		})
	}
}

func TestHelpPageDescriptionFormatsSentencePunctuation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		text     string
		language string
		want     string
	}{
		{name: "empty", text: "  ", language: "en-US", want: ""},
		{name: "english", text: "Show command help", language: "en-US", want: "Show command help."},
		{name: "english-existing", text: "Show command help.", language: "en-US", want: "Show command help."},
		{name: "chinese", text: "显示命令帮助", language: "zh-CN", want: "显示命令帮助。"},
		{name: "chinese-existing", text: "显示命令帮助。", language: "zh-CN", want: "显示命令帮助。"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := helpPageDescription(tc.text, tc.language); got != tc.want {
				t.Fatalf("helpPageDescription() = %q, want %q", got, tc.want)
			}
		})
	}
}

func helpDescriptionColumn(line string) int {
	spaceRun := 0
	for i := 2; i < len(line); i++ {
		if line[i] == ' ' {
			spaceRun++
			continue
		}
		if spaceRun >= 2 {
			return i
		}
		spaceRun = 0
	}
	return -1
}

func TestHelpFlagShowsCoreSubcommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "plugin", "install", "--help"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("core subcommand help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"plugin install", "Install a plugin from a hosted source", "ctyun plugin install <name...|--all>", "Command Options:", "--all", "--source name", "--channel name", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("core subcommand help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "--bundled") {
		t.Fatalf("core subcommand help output exposed dev-only --bundled:\n%s", got)
	}
}

func TestHelpShowsPluginManagementSubcommands(t *testing.T) {
	for _, tc := range []struct {
		args     []string
		want     []string
		unwanted []string
	}{
		{args: []string{"help", "plugin"}, want: []string{"Manage plugin bundles and discover metadata-defined product commands.", "Usage:", "ctyun plugin <subcommand> [options]", "ctyun plugins <subcommand> [options]", "Subcommands:", "install", "Install a plugin from a hosted source", "reinstall", "Reinstall one or all installed plugins", "update|upgrade", "Update or upgrade one or all installed plugins", "Global Options:"}, unwanted: []string{"Description:", "Plugin Commands:", "plugin and plugins are equivalent", "update, upgrade", "--bundled", "lint"}},
		{args: []string{"help", "plugins"}, want: []string{"Manage plugin bundles and discover metadata-defined product commands.", "Usage:", "ctyun plugin <subcommand> [options]", "ctyun plugins <subcommand> [options]", "Subcommands:", "reinstall", "Reinstall one or all installed plugins", "update|upgrade", "Update or upgrade one or all installed plugins", "Global Options:"}, unwanted: []string{"Description:", "Plugin Commands:", "plugin and plugins are equivalent", "update, upgrade", "lint"}},
		{args: []string{"help", "plugin", "list"}, want: []string{"List installed or available plugins.", "ctyun plugin list [--available|--updates] [--source auto|github|gitee]", "--available", "--updates", "default: auto", "default: stable"}, unwanted: []string{"plugin list\n", "Description:"}},
		{args: []string{"help", "plugin", "remove"}, want: []string{"Remove installed plugins.", "ctyun plugin remove <name...|--all>", "--all"}, unwanted: []string{"plugin remove\n", "Description:"}},
		{args: []string{"help", "plugin", "search"}, want: []string{"Search hosted plugin metadata.", "ctyun plugin search <query>", "--channel name", "default: auto", "default: stable"}, unwanted: []string{"plugin search\n", "Description:"}},
		{args: []string{"help", "plugin", "reinstall"}, want: []string{"Reinstall one or all installed plugins.", "ctyun plugin reinstall <name...|--all>", "Command Options:", "--all", "--source name", "--channel name", "default: auto", "default: stable"}, unwanted: []string{"plugin reinstall\n", "Description:", "--bundled"}},
		{args: []string{"help", "plugin", "update"}, want: []string{"Update or upgrade one or all installed plugins.", "ctyun plugin update <name|--all>", "ctyun plugin upgrade <name|--all>", "ctyun plugins update <name|--all>", "ctyun plugins upgrade <name|--all>", "Command Options:", "--all"}, unwanted: []string{"plugin update|upgrade\n", "Description:", "Plugin Options:", "update|upgrade <name|--all>", "update, upgrade"}},
		{args: []string{"help", "plugins", "upgrade"}, want: []string{"Update or upgrade one or all installed plugins.", "ctyun plugin update <name|--all>", "ctyun plugin upgrade <name|--all>", "ctyun plugins update <name|--all>", "ctyun plugins upgrade <name|--all>", "Command Options:", "--all"}, unwanted: []string{"plugin update|upgrade\n", "Description:", "Plugin Options:", "update|upgrade <name|--all>", "update, upgrade"}},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			if err := Run(Config{
				Args:   append([]string{"--lang", "en-US"}, tc.args...),
				Stdout: &stdout,
			}); err != nil {
				t.Fatalf("help returned error: %v", err)
			}
			got := stdout.String()
			if len(tc.want) > 0 {
				if first := firstNonEmptyLine(got); first != tc.want[0] {
					t.Fatalf("plugin help first line = %q, want %q\n%s", first, tc.want[0], got)
				}
			}
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("plugin help output missing %q:\n%s", want, got)
				}
			}
			for _, unwanted := range tc.unwanted {
				if strings.Contains(got, unwanted) {
					t.Fatalf("plugin help output contains %q:\n%s", unwanted, got)
				}
			}
		})
	}
}

func TestPluginAllOptionHelpDescribesCommandScope(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "install", args: []string{"help", "plugin", "install"}, want: "Install every available plugin"},
		{name: "remove", args: []string{"help", "plugin", "remove"}, want: "Remove every installed plugin"},
		{name: "reinstall", args: []string{"help", "plugin", "reinstall"}, want: "Reinstall every installed plugin"},
		{name: "update", args: []string{"help", "plugin", "update"}, want: "Update every installed plugin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := Run(Config{
				Args:   append([]string{"--lang", "en-US"}, tc.args...),
				Stdout: &stdout,
			}); err != nil {
				t.Fatalf("help returned error: %v", err)
			}
			if got := stdout.String(); !strings.Contains(got, tc.want) {
				t.Fatalf("plugin --all help missing %q:\n%s", tc.want, got)
			}
		})
	}

	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "zh-CN", "help", "plugin", "install"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("Chinese help returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "安装全部可用插件") {
		t.Fatalf("Chinese plugin install --all help missing localized scope:\n%s", got)
	}
}

func TestHelpShowsCoreUpdateGuidance(t *testing.T) {
	for _, command := range []string{"update", "upgrade"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := Run(Config{
				Args:   []string{"--lang", "en-US", "help", command},
				Stdout: &stdout,
			}); err != nil {
				t.Fatalf("help %s returned error: %v", command, err)
			}
			got := stdout.String()
			for _, want := range []string{"Update or upgrade the core ctyun binary.", "For plugin updates, run ctyun plugin|plugins update|upgrade.", "ctyun update [--check]", "ctyun upgrade [--check]", "--source auto|github|gitee", "Command Options:", "--check", "--source value", "--channel name", "default: auto", "default: stable"} {
				if !strings.Contains(got, want) {
					t.Fatalf("core update help output missing %q:\n%s", want, got)
				}
			}
			if first := firstNonEmptyLine(got); first != "Update or upgrade the core ctyun binary." {
				t.Fatalf("core update help first line = %q\n%s", first, got)
			}
			for _, unwanted := range []string{"Description:", "URL|path", "path-or-url", "ctyun plugin update|upgrade <name|--all>", "ctyun plugins update|upgrade <name|--all>", "upgrade, update"} {
				if strings.Contains(got, unwanted) {
					t.Fatalf("core update help output contains %q:\n%s", unwanted, got)
				}
			}
		})
	}
}

func TestHelpShowsDoctorSubcommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "doctor"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("doctor help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Inspect local environment details that affect ctyun connectivity.", "Usage:", "ctyun doctor <subcommand>", "Subcommands:", "network", "Inspect local network and registry configuration", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor help output missing %q:\n%s", want, got)
		}
	}
	if first := firstNonEmptyLine(got); first != "Inspect local environment details that affect ctyun connectivity." {
		t.Fatalf("doctor help first line = %q", first)
	}
	if strings.Contains(got, "doctor\n") || strings.Contains(got, "Description:") || strings.Contains(got, "Core Commands:") {
		t.Fatalf("doctor help output contains redundant title or description heading:\n%s", got)
	}
}

func TestHelpShowsDoctorNetworkDetailAndRejectsUnknownSubcommands(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:   []string{"--lang", "en-US", "help", "doctor", "network"},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("doctor network help returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{"Inspect local network and registry configuration.", "Usage:", "ctyun doctor network", "Global Options:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor network help output missing %q:\n%s", want, got)
		}
	}
	if first := firstNonEmptyLine(got); first != "Inspect local network and registry configuration." {
		t.Fatalf("doctor network help first line = %q", first)
	}
	if strings.HasPrefix(got, "doctor network\n") || strings.Contains(got, "Description:") {
		t.Fatalf("doctor network help output contains redundant title or description heading:\n%s", got)
	}
	handled, err := printCoreHelp(io.Discard, []string{"doctor", "unknown"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp unknown doctor returned error: %v", err)
	}
	if handled {
		t.Fatal("printCoreHelp returned true for unknown doctor subcommand")
	}
	handled, err = printCoreHelp(io.Discard, []string{"doctor", "network", "extra"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp extra doctor returned error: %v", err)
	}
	if handled {
		t.Fatal("printCoreHelp returned true for extra doctor help argument")
	}
}
