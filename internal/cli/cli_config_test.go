/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
)

func TestConfigPathPrintsResolvedPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"config", "path"},
		Stdout:     &stdout,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("config path returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != configPath {
		t.Fatalf("config path = %q, want %q", got, configPath)
	}
}

func TestConfigShowRedactsCredentialValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{
  "ak": "global-ak",
  "sk": "global-sk",
  "profiles": {
    "dev": {
      "region": "cn-huadong2"
    },
    "prod": {
      "region": "cn-huadong1",
      "ak": "profile-ak",
      "sk": "profile-sk"
    }
  }
}`)

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"config", "show"},
		Stdout:     &stdout,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("config show returned error: %v", err)
	}
	output := stdout.String()
	for _, secret := range []string{"global-ak", "global-sk", "profile-ak", "profile-sk"} {
		if strings.Contains(output, secret) {
			t.Fatalf("config show leaked %q in %s", secret, output)
		}
	}
	for _, masked := range []string{`"ak": "gl*****ak"`, `"sk": "gl*****sk"`, `"ak": "pr*****ak"`, `"sk": "pr*****sk"`} {
		if !strings.Contains(output, masked) {
			t.Fatalf("config show missing masked credential %q in %s", masked, output)
		}
	}
	if !strings.Contains(output, `"dev":`) || !strings.Contains(output, `"ak": ""`) || !strings.Contains(output, `"sk": ""`) {
		t.Fatalf("config show did not keep empty credential fields empty: %s", output)
	}
}

func TestConfigSetUnsetAndProfileUsePersistValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	runConfigForTest(t, configPath, nil, "config", "set", "region", "cn-huadong1", "--profile", "prod")
	runConfigForTest(t, configPath, nil, "config", "set", "language", "en-GB", "--profile", "prod")
	runConfigForTest(t, configPath, nil, "config", "profile", "use", "prod")
	runConfigForTest(t, configPath, nil, "config", "unset", "language", "--profile", "prod")

	cfg := readConfigForTest(t, configPath)
	if cfg.ActiveProfileName != "prod" {
		t.Fatalf("active profile = %q, want prod", cfg.ActiveProfileName)
	}
	profile := cfg.Profiles["prod"]
	if profile.Region != "cn-huadong1" {
		t.Fatalf("profile region = %q, want cn-huadong1", profile.Region)
	}
	if profile.Language != "" {
		t.Fatalf("profile language = %q, want empty after unset", profile.Language)
	}
}

func TestConfigMutationStatusMessagesUseLanguage(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var stdout bytes.Buffer
	steps := []struct {
		args []string
		want string
	}{
		{args: []string{"--lang", "zh-CN", "config", "set", "region", "cn-huadong1", "--profile", "prod"}, want: "已更新 region。"},
		{args: []string{"--lang", "zh-CN", "config", "profile", "use", "prod"}, want: "当前配置档案：prod。"},
		{args: []string{"--lang", "zh-CN", "config", "profile", "set", "prod", "language=zh-CN"}, want: "已更新配置档案 prod 的 language。"},
		{args: []string{"--lang", "zh-CN", "config", "unset", "language", "--profile", "prod"}, want: "已取消 language。"},
	}
	for _, step := range steps {
		stdout.Reset()
		err := Run(Config{
			Args:       step.args,
			Stdout:     &stdout,
			ConfigPath: configPath,
		})
		if err != nil {
			t.Fatalf("%v returned error: %v", step.args, err)
		}
		if got := strings.TrimSpace(stdout.String()); got != step.want {
			t.Fatalf("%v output = %q, want %q", step.args, got, step.want)
		}
	}
}

func TestConfigProfileListShowsActiveProfile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{
  "active_profile": "prod",
  "profiles": {
    "dev": {"region": "cn-dev"},
    "prod": {"region": "cn-prod"}
  }
}`)

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"config", "profile", "list"},
		Stdout:     &stdout,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("config profile list returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "* prod") || !strings.Contains(output, "  dev") {
		t.Fatalf("profile list output = %q", output)
	}
}

func TestConfigProfileSetSecretReadsFromStdin(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var stdout bytes.Buffer
	err := Run(Config{
		Args:       []string{"config", "profile", "set-secret", "prod", "ak", "--from-stdin"},
		Stdout:     &stdout,
		Stdin:      strings.NewReader("stdin-ak\n"),
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("config profile set-secret returned error: %v", err)
	}
	if strings.Contains(stdout.String(), "stdin-ak") {
		t.Fatalf("set-secret leaked secret in stdout: %q", stdout.String())
	}
	cfg := readConfigForTest(t, configPath)
	if got := cfg.Profiles["prod"].AccessKey; got != "stdin-ak" {
		t.Fatalf("profile ak = %q, want stdin-ak", got)
	}
}

func TestConfigResetPromptsAndCreatesBackup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	mustWrite(t, configPath, `{"active_profile":"prod","profiles":{"prod":{"region":"cn"}}}`)

	var stderr bytes.Buffer
	if err := Run(Config{
		Args:       []string{"config", "reset"},
		Stderr:     &stderr,
		Stdin:      strings.NewReader("n\n"),
		ConfigPath: configPath,
	}); err == nil {
		t.Fatal("declined config reset returned nil error")
	}
	if !strings.Contains(stderr.String(), "Continue? [y/N]:") {
		t.Fatalf("confirmation prompt missing from stderr:\n%s", stderr.String())
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config reset removed config after declined confirmation: %v", err)
	}

	runConfigForTest(t, configPath, strings.NewReader("y\n"), "config", "reset")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config file exists after reset, stat err = %v", err)
	}
	if _, err := os.Stat(configPath + ".bak"); err != nil {
		t.Fatalf("config backup missing: %v", err)
	}
}

func TestConfigProfileResetPrompts(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{
  "active_profile": "prod",
  "profiles": {
    "dev": {"region": "cn-dev"},
    "prod": {"region": "cn-prod"}
  }
}`)

	var stderr bytes.Buffer
	if err := Run(Config{
		Args:       []string{"config", "profile", "reset", "prod"},
		Stderr:     &stderr,
		Stdin:      strings.NewReader("n\n"),
		ConfigPath: configPath,
	}); err == nil {
		t.Fatal("declined config profile reset returned nil error")
	}
	if !strings.Contains(stderr.String(), "Continue? [y/N]:") {
		t.Fatalf("confirmation prompt missing from stderr:\n%s", stderr.String())
	}
	cfg := readConfigForTest(t, configPath)
	if _, ok := cfg.Profiles["prod"]; !ok {
		t.Fatal("prod profile was removed after declined confirmation")
	}

	runConfigForTest(t, configPath, strings.NewReader("yes\n"), "config", "profile", "reset", "prod")
	cfg = readConfigForTest(t, configPath)
	if _, ok := cfg.Profiles["prod"]; ok {
		t.Fatal("prod profile still exists after reset")
	}
	if cfg.ActiveProfileName != "" {
		t.Fatalf("active profile = %q, want empty after deleting active profile", cfg.ActiveProfileName)
	}
	if _, ok := cfg.Profiles["dev"]; !ok {
		t.Fatal("dev profile missing after resetting prod")
	}
}

func TestConfigCommandCoversHelpAliasesAndErrors(t *testing.T) {
	var stdout bytes.Buffer
	if err := runConfigCommand(&stdout, ioDiscardForConfigTest{}, strings.NewReader(""), nil, globalOptions{Language: "en-US"}, nil, ""); err != nil {
		t.Fatalf("config help returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "ctyun config <subcommand> [options]") {
		t.Fatalf("config help output = %q", stdout.String())
	}
	if err := runConfigCommand(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), []string{"missing"}, globalOptions{}, nil, ""); err == nil {
		t.Fatal("unknown config subcommand returned nil error")
	}
	if err := runConfigCommand(failingWriter{}, ioDiscardForConfigTest{}, strings.NewReader(""), nil, globalOptions{Language: "en-US"}, nil, ""); err == nil {
		t.Fatal("config help returned nil error for writer failure")
	}
	if err := runConfigPath(ioDiscardForConfigTest{}, ""); err == nil {
		t.Fatal("empty config path returned nil error")
	}
	stdout.Reset()
	handled, err := printCoreHelp(&stdout, []string{"config"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp config returned error: %v", err)
	}
	if !handled {
		t.Fatal("printCoreHelp did not handle config")
	}
	output := stdout.String()
	for _, want := range []string{"Usage:", "ctyun config <subcommand> [options]", "Subcommands:", "profile|profiles"} {
		if !strings.Contains(output, want) {
			t.Fatalf("config help missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "without interactive prompts") {
		t.Fatalf("config help still mentions interactive prompts:\n%s", output)
	}
	stdout.Reset()
	handled, err = printCoreHelp(&stdout, []string{"config", "show"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp config show returned error: %v", err)
	}
	if !handled {
		t.Fatal("printCoreHelp did not handle config show")
	}
	output = stdout.String()
	if !strings.Contains(output, "ctyun config show") || strings.Contains(output, "--json") {
		t.Fatalf("config show help output = %q", output)
	}
	stdout.Reset()
	handled, err = printCoreHelp(&stdout, []string{"config", "profile"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp config profile returned error: %v", err)
	}
	if !handled {
		t.Fatal("printCoreHelp did not handle config profile")
	}
	output = stdout.String()
	if !strings.Contains(output, "ctyun config profile <subcommand> [options]") || !strings.Contains(output, "set-secret") {
		t.Fatalf("config profile help output = %q", output)
	}
	stdout.Reset()
	handled, err = printCoreHelp(&stdout, []string{"config", "profile", "set-secret"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp config profile set-secret returned error: %v", err)
	}
	if !handled {
		t.Fatal("printCoreHelp did not handle config profile set-secret")
	}
	output = stdout.String()
	if !strings.Contains(output, "Command Options") || !strings.Contains(output, "--from-stdin") {
		t.Fatalf("config profile set-secret help output = %q", output)
	}
	handled, err = printCoreHelp(ioDiscardForConfigTest{}, []string{"config", "extra"}, "en-US")
	if err != nil {
		t.Fatalf("printCoreHelp unknown config subcommand returned error: %v", err)
	}
	if handled {
		t.Fatal("printCoreHelp handled unknown config subcommand")
	}
	handled, err = printConfigHelp(ioDiscardForConfigTest{}, nil, "en-US")
	if err != nil {
		t.Fatalf("printConfigHelp empty args returned error: %v", err)
	}
	if handled {
		t.Fatal("printConfigHelp handled empty args")
	}
	handled, err = printConfigHelp(ioDiscardForConfigTest{}, []string{"config", "show", "extra"}, "en-US")
	if err != nil {
		t.Fatalf("printConfigHelp extra args returned error: %v", err)
	}
	if handled {
		t.Fatal("printConfigHelp handled config subcommand with too many args")
	}
	handled, err = printConfigProfileHelp(ioDiscardForConfigTest{}, []string{"config", "profile", "set", "extra"}, "en-US")
	if err != nil {
		t.Fatalf("printConfigProfileHelp extra args returned error: %v", err)
	}
	if handled {
		t.Fatal("printConfigProfileHelp handled profile subcommand with too many args")
	}
	handled, err = printConfigProfileHelp(ioDiscardForConfigTest{}, []string{"config", "profile", "missing"}, "en-US")
	if err != nil {
		t.Fatalf("printConfigProfileHelp missing subcommand returned error: %v", err)
	}
	if handled {
		t.Fatal("printConfigProfileHelp handled unknown profile subcommand")
	}
	if !configSubcommandMatches(configSubcommandSummaries()[4], "profiles") {
		t.Fatal("configSubcommandMatches did not match profiles alias")
	}
	assertEqualCompletions(t, commandCompletions([]string{"config"}, completionContext{}), []string{"path", "profile", "profiles", "reset", "set", "show", "unset"})
	assertEqualCompletions(t, configCommandCompletions([]string{"config", "profile"}), []string{"list", "reset", "set", "set-secret", "unset", "use"})
	assertEqualCompletions(t, configCommandCompletions([]string{"config", "profile", "set-secret", "prod"}), []string{"ak", "sk"})
	assertEqualCompletions(t, configCommandCompletions([]string{"config", "profile", "set-secret", "prod", "ak"}), []string{"--from-stdin"})
	assertEqualCompletions(t, configCommandCompletions([]string{"config", "show"}), nil)
	assertEqualCompletions(t, configCommandCompletions([]string{"config", "path", "extra"}), nil)
	assertEqualCompletions(t, configProfileCompletionOptionNames("missing"), nil)
	assertHasCompletions(t, allCompletionWords(t.TempDir()), "ak", "sk", "--from-stdin", "set-secret")
	if got := globalCompletionOptionValues("--wait")(completionContext{}); got != nil {
		t.Fatalf("wait completions without command = %v, want nil", got)
	}
}

func TestConfigShowBranchesAndWriterErrors(t *testing.T) {
	raw := []byte(`{"profiles":{"prod":{"ak":"profile-ak","sk":"profile-sk"}}}`)
	var stdout bytes.Buffer
	if err := runConfigShow(&stdout, nil, globalOptions{Profile: "prod"}, raw); err != nil {
		t.Fatalf("profile config show returned error: %v", err)
	}
	if strings.Contains(stdout.String(), "profile-ak") || !strings.Contains(stdout.String(), "pr*****ak") {
		t.Fatalf("profile config show output = %q", stdout.String())
	}
	if err := runConfigShow(ioDiscardForConfigTest{}, []string{"--bad"}, globalOptions{}, raw); err == nil {
		t.Fatal("unknown show option returned nil error")
	}
	if err := runConfigShow(ioDiscardForConfigTest{}, nil, globalOptions{}, []byte("{")); err == nil {
		t.Fatal("invalid config show returned nil error")
	}
	if err := runConfigShow(ioDiscardForConfigTest{}, nil, globalOptions{Profile: "missing"}, raw); err == nil {
		t.Fatal("missing profile show returned nil error")
	}
	if err := runConfigShow(failingWriter{}, nil, globalOptions{}, raw); err == nil {
		t.Fatal("show with failing writer returned nil error")
	}
	if err := writeJSON(ioDiscardForConfigTest{}, make(chan int), true); err == nil {
		t.Fatal("indented writeJSON with unsupported value returned nil error")
	}
	if err := writeJSON(ioDiscardForConfigTest{}, make(chan int), false); err == nil {
		t.Fatal("compact writeJSON with unsupported value returned nil error")
	}
	if got := maskCredentialValue(""); got != "" {
		t.Fatalf("empty maskCredentialValue = %q, want empty", got)
	}
	if got := maskCredentialValue("abcd"); got != "*****" {
		t.Fatalf("short maskCredentialValue = %q, want *****", got)
	}
}

func TestConfigSetUnsetGlobalBranches(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{"profiles":{"prod":{"region":"cn"}}}`)
	runConfigForTest(t, configPath, nil, "config", "set", "active_profile", "prod")
	runConfigForTest(t, configPath, nil, "config", "set", "ak", "global-ak")
	runConfigForTest(t, configPath, nil, "config", "set", "sk", "global-sk")
	runConfigForTest(t, configPath, nil, "config", "set", "warn_config_credentials", "false")
	runConfigForTest(t, configPath, nil, "config", "unset", "active_profile")
	runConfigForTest(t, configPath, nil, "config", "unset", "ak")
	runConfigForTest(t, configPath, nil, "config", "unset", "sk")
	runConfigForTest(t, configPath, nil, "config", "unset", "warn_config_credentials")

	cfg := readConfigForTest(t, configPath)
	if cfg.ActiveProfileName != "" || cfg.AccessKey != "" || cfg.SecretKey != "" || cfg.WarnConfigCredentials != nil {
		t.Fatalf("global config fields after unset = %+v", cfg)
	}

	for _, args := range [][]string{
		{"only-one"},
		{"active_profile", "missing"},
		{"warn_config_credentials", "maybe"},
		{"unsupported", "value"},
	} {
		if err := runConfigSet(ioDiscardForConfigTest{}, nil, configPath, "", args, "en-US"); err == nil {
			t.Fatalf("config set %v returned nil error", args)
		}
	}
	for _, args := range [][]string{{}, {"unsupported"}} {
		if err := runConfigUnset(ioDiscardForConfigTest{}, nil, configPath, "", args, "en-US"); err == nil {
			t.Fatalf("config unset %v returned nil error", args)
		}
	}
	if err := runConfigSet(ioDiscardForConfigTest{}, []byte("{"), configPath, "", []string{"ak", "value"}, "en-US"); err == nil {
		t.Fatal("config set with invalid raw returned nil error")
	}
	if err := runConfigUnset(ioDiscardForConfigTest{}, []byte("{"), configPath, "", []string{"ak"}, "en-US"); err == nil {
		t.Fatal("config unset with invalid raw returned nil error")
	}
	if err := runConfigSet(ioDiscardForConfigTest{}, nil, "", "", []string{"ak", "value"}, "en-US"); err == nil {
		t.Fatal("config set with empty path returned nil error")
	}
	if err := runConfigSet(ioDiscardForConfigTest{}, nil, configPath, "prod", []string{"unknown", "value"}, "en-US"); err == nil {
		t.Fatal("profile-scoped config set with unknown key returned nil error")
	}
	if err := runConfigUnset(ioDiscardForConfigTest{}, nil, "", "", []string{"ak"}, "en-US"); err == nil {
		t.Fatal("config unset with empty path returned nil error")
	}
	if err := runConfigUnset(ioDiscardForConfigTest{}, []byte(`{"profiles":{"prod":{}}}`), configPath, "prod", []string{"unknown"}, "en-US"); err == nil {
		t.Fatal("profile-scoped config unset with unknown key returned nil error")
	}
	if err := runConfigSet(failingWriter{}, nil, configPath, "", []string{"ak", "value"}, "en-US"); err == nil {
		t.Fatal("config set with failing writer returned nil error")
	}
	if err := runConfigUnset(failingWriter{}, nil, configPath, "", []string{"ak"}, "en-US"); err == nil {
		t.Fatal("config unset with failing writer returned nil error")
	}
}

func TestConfigProfileSubcommandsBranches(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{"profiles":{"prod":{"region":"cn"}}}`)

	runConfigForTest(t, configPath, nil, "config", "profiles", "list")
	runConfigForTest(t, configPath, nil, "config", "profile", "set", "prod", "language=en-GB")
	runConfigForTest(t, configPath, nil, "config", "profile", "set", "prod", "endpoint_url", "https://example.com")
	runConfigForTest(t, configPath, nil, "config", "profile", "unset", "prod", "endpoint_url")
	runConfigForTest(t, configPath, strings.NewReader("stdin-sk\n"), "config", "profile", "set-secret", "prod", "sk", "--from-stdin")

	cfg := readConfigForTest(t, configPath)
	profile := cfg.Profiles["prod"]
	if profile.Language != "en-GB" || profile.EndpointURL != "" || profile.SecretKey != "stdin-sk" {
		t.Fatalf("profile after subcommands = %+v", profile)
	}

	if err := runConfigProfile(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), nil, configPath, globalOptions{}, nil); err == nil {
		t.Fatal("empty profile subcommand returned nil error")
	}
	if err := runConfigProfile(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), nil, configPath, globalOptions{}, []string{"missing"}); err == nil {
		t.Fatal("unknown profile subcommand returned nil error")
	}
	if err := runConfigProfileList(failingWriter{}, []byte(`{"profiles":{"prod":{}}}`)); err == nil {
		t.Fatal("profile list with failing writer returned nil error")
	}
	if err := runConfigProfileList(ioDiscardForConfigTest{}, []byte("{")); err == nil {
		t.Fatal("profile list with invalid raw returned nil error")
	}
	if err := runConfigProfileUse(ioDiscardForConfigTest{}, nil, configPath, nil, "en-US"); err == nil {
		t.Fatal("profile use without name returned nil error")
	}
	if err := runConfigProfileUse(ioDiscardForConfigTest{}, []byte("{"), configPath, []string{"prod"}, "en-US"); err == nil {
		t.Fatal("profile use with invalid raw returned nil error")
	}
	if err := runConfigProfileUse(ioDiscardForConfigTest{}, nil, configPath, []string{"missing"}, "en-US"); err == nil {
		t.Fatal("profile use missing profile returned nil error")
	}
	if err := runConfigProfileUse(ioDiscardForConfigTest{}, []byte(`{"profiles":{"prod":{}}}`), "", []string{"prod"}, "en-US"); err == nil {
		t.Fatal("profile use with empty path returned nil error")
	}
	if err := runConfigProfileUse(failingWriter{}, []byte(`{"profiles":{"prod":{}}}`), configPath, []string{"prod"}, "en-US"); err == nil {
		t.Fatal("profile use with failing writer returned nil error")
	}
}

func TestConfigProfileSetUnsetBranches(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{"profiles":{"prod":{"region":"cn"}}}`)

	if err := runConfigProfileSet(ioDiscardForConfigTest{}, nil, configPath, []string{"prod"}, "en-US"); err == nil {
		t.Fatal("profile set usage error returned nil")
	}
	if err := runConfigProfileSet(ioDiscardForConfigTest{}, nil, configPath, []string{"prod", "=bad"}, "en-US"); err == nil {
		t.Fatal("profile set bad key=value returned nil")
	}
	if err := runConfigProfileSet(ioDiscardForConfigTest{}, []byte("{"), configPath, []string{"prod", "region=cn"}, "en-US"); err == nil {
		t.Fatal("profile set invalid raw returned nil")
	}
	if err := runConfigProfileSet(ioDiscardForConfigTest{}, nil, configPath, []string{"prod", "unknown=value"}, "en-US"); err == nil {
		t.Fatal("profile set unknown key returned nil")
	}
	if err := runConfigProfileSet(ioDiscardForConfigTest{}, nil, "", []string{"prod", "region=cn"}, "en-US"); err == nil {
		t.Fatal("profile set empty path returned nil")
	}
	if err := runConfigProfileSet(failingWriter{}, nil, configPath, []string{"prod", "region=cn"}, "en-US"); err == nil {
		t.Fatal("profile set failing writer returned nil")
	}
	if err := runConfigProfileUnset(ioDiscardForConfigTest{}, nil, configPath, []string{"prod"}, "en-US"); err == nil {
		t.Fatal("profile unset usage error returned nil")
	}
	if err := runConfigProfileUnset(ioDiscardForConfigTest{}, []byte("{"), configPath, []string{"prod", "region"}, "en-US"); err == nil {
		t.Fatal("profile unset invalid raw returned nil")
	}
	if err := runConfigProfileUnset(ioDiscardForConfigTest{}, nil, configPath, []string{"missing", "region"}, "en-US"); err == nil {
		t.Fatal("profile unset missing profile returned nil")
	}
	if err := runConfigProfileUnset(ioDiscardForConfigTest{}, nil, configPath, []string{"prod", "unknown"}, "en-US"); err == nil {
		t.Fatal("profile unset unknown key returned nil")
	}
	if err := runConfigProfileUnset(ioDiscardForConfigTest{}, []byte(`{"profiles":{"prod":{"region":"cn"}}}`), "", []string{"prod", "region"}, "en-US"); err == nil {
		t.Fatal("profile unset empty path returned nil")
	}
	if err := runConfigProfileUnset(failingWriter{}, nil, configPath, []string{"prod", "region"}, "en-US"); err == nil {
		t.Fatal("profile unset failing writer returned nil")
	}
}

func TestConfigSetSecretAndResetBranches(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	mustWrite(t, configPath, `{"active_profile":"prod","profiles":{"prod":{"region":"cn"}}}`)

	if err := runConfigProfileSetSecret(ioDiscardForConfigTest{}, strings.NewReader(""), nil, configPath, []string{"prod", "ak"}, "en-US"); err == nil {
		t.Fatal("set-secret usage error returned nil")
	}
	if err := runConfigProfileSetSecret(ioDiscardForConfigTest{}, strings.NewReader(""), nil, configPath, []string{"prod", "region", "--from-stdin"}, "en-US"); err == nil {
		t.Fatal("set-secret unsupported key returned nil")
	}
	if err := runConfigProfileSetSecret(ioDiscardForConfigTest{}, configFailingReader{}, nil, configPath, []string{"prod", "ak", "--from-stdin"}, "en-US"); err == nil {
		t.Fatal("set-secret failing reader returned nil")
	}
	if err := runConfigProfileSetSecret(ioDiscardForConfigTest{}, strings.NewReader("ak"), []byte("{"), configPath, []string{"prod", "ak", "--from-stdin"}, "en-US"); err == nil {
		t.Fatal("set-secret invalid raw returned nil")
	}
	if err := runConfigProfileSetSecret(ioDiscardForConfigTest{}, strings.NewReader("ak"), nil, "", []string{"prod", "ak", "--from-stdin"}, "en-US"); err == nil {
		t.Fatal("set-secret empty path returned nil")
	}
	if err := runConfigProfileSetSecret(failingWriter{}, strings.NewReader("ak"), nil, configPath, []string{"prod", "ak", "--from-stdin"}, "en-US"); err == nil {
		t.Fatal("set-secret failing writer returned nil")
	}
	if err := runConfigProfileReset(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), nil, configPath, globalOptions{Yes: true}, nil); err == nil {
		t.Fatal("profile reset usage error returned nil")
	}
	if err := runConfigProfileReset(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), []byte("{"), configPath, globalOptions{Yes: true}, []string{"prod"}); err == nil {
		t.Fatal("profile reset invalid raw returned nil")
	}
	if err := runConfigProfileReset(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), nil, "", globalOptions{Yes: true}, []string{"prod"}); err == nil {
		t.Fatal("profile reset empty path returned nil")
	}
	if err := runConfigProfileReset(failingWriter{}, ioDiscardForConfigTest{}, strings.NewReader(""), nil, configPath, globalOptions{Yes: true}, []string{"prod"}); err == nil {
		t.Fatal("profile reset failing writer returned nil")
	}
	if err := runConfigReset(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), "", globalOptions{Yes: true}); err == nil {
		t.Fatal("config reset empty path returned nil")
	}
	if err := runConfigReset(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), string([]byte{0}), globalOptions{Yes: true}); err == nil {
		t.Fatal("config reset invalid path returned nil")
	}
	if err := runConfigReset(failingWriter{}, ioDiscardForConfigTest{}, strings.NewReader(""), filepath.Join(t.TempDir(), "missing.json"), globalOptions{Yes: true}); err == nil {
		t.Fatal("config reset missing file with failing writer returned nil")
	}
	dirPath := t.TempDir()
	if err := runConfigReset(ioDiscardForConfigTest{}, ioDiscardForConfigTest{}, strings.NewReader(""), dirPath, globalOptions{Yes: true}); err == nil {
		t.Fatal("config reset directory path returned nil")
	}
}

func TestConfigValueHelpersCoverSupportedKeys(t *testing.T) {
	var cfg coreconfig.Config
	cfg.Profiles = map[string]coreconfig.Profile{"prod": {}}
	for _, item := range []struct {
		key   string
		value string
	}{
		{"region", "cn"},
		{"language", "en-GB"},
		{"registry_url", "https://registry.example.com"},
		{"registry_public_key", "pub"},
		{"registry.url", "https://registry.example.com/nested"},
		{"registry.public_key", "nested-pub"},
		{"endpoint_url", "https://endpoint.example.com"},
		{"timeout_seconds", "30"},
		{"ak", "ak"},
		{"sk", "sk"},
		{"warn_config_credentials", "true"},
	} {
		if err := setProfileValue(&cfg, "prod", item.key, item.value); err != nil {
			t.Fatalf("setProfileValue %s returned error: %v", item.key, err)
		}
		if err := unsetProfileValue(&cfg, "prod", item.key); err != nil {
			t.Fatalf("unsetProfileValue %s returned error: %v", item.key, err)
		}
	}
	if err := setProfileValue(&coreconfig.Config{}, "new", "region", "cn"); err != nil {
		t.Fatalf("setProfileValue with nil profiles returned error: %v", err)
	}
	for _, item := range []struct {
		key   string
		value string
	}{
		{"timeout_seconds", "bad"},
		{"warn_config_credentials", "bad"},
		{"missing", "value"},
	} {
		if err := setProfileValue(&cfg, "prod", item.key, item.value); err == nil {
			t.Fatalf("setProfileValue %s returned nil error", item.key)
		}
	}
	if err := clearProfileValue(&coreconfig.Profile{}, "missing"); err == nil {
		t.Fatal("clearProfileValue missing key returned nil error")
	}
	if key, value, err := profileSetPair([]string{"region=cn"}); err != nil || key != "region" || value != "cn" {
		t.Fatalf("profileSetPair key=value = %q %q %v", key, value, err)
	}
	if key, value, err := profileSetPair([]string{"region", "cn"}); err != nil || key != "region" || value != "cn" {
		t.Fatalf("profileSetPair key value = %q %q %v", key, value, err)
	}
}

func TestConfigFileHelpersCoverFilesystemErrors(t *testing.T) {
	if err := writeConfigFile("", coreconfig.Config{}); err == nil {
		t.Fatal("writeConfigFile empty path returned nil error")
	}
	parentFile := filepath.Join(t.TempDir(), "parent")
	mustWrite(t, parentFile, "")
	if err := writeConfigFile(filepath.Join(parentFile, "config.json"), coreconfig.Config{}); err == nil {
		t.Fatal("writeConfigFile under file parent returned nil error")
	}
	if _, err := backupConfigFile(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("backupConfigFile missing path returned nil error")
	}
	noWriteDir := t.TempDir()
	noWriteConfig := filepath.Join(noWriteDir, "config.json")
	mustWrite(t, noWriteConfig, "{}")
	if err := os.Chmod(noWriteDir, 0o500); err != nil {
		t.Fatalf("chmod no-write dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(noWriteDir, 0o700)
	})
	if _, err := backupConfigFile(noWriteConfig); err == nil {
		t.Fatal("backupConfigFile in no-write dir returned nil error")
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	mustWrite(t, configPath, "{}")
	mustWrite(t, configPath+".bak", "{}")
	backupPath, err := backupConfigFile(configPath)
	if err != nil {
		t.Fatalf("backupConfigFile with existing backup returned error: %v", err)
	}
	if backupPath != configPath+".bak.2" {
		t.Fatalf("backup path = %q, want %q", backupPath, configPath+".bak.2")
	}
}

func TestConfigWriteValidatesBeforePersisting(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	original := `{"profiles":{"prod":{"region":"cn"}}}`
	mustWrite(t, configPath, original)

	invalid := coreconfig.Config{
		ActiveProfileName: "missing",
		Profiles: map[string]coreconfig.Profile{
			"prod": {Region: "cn"},
		},
	}
	if err := writeConfigFile(configPath, invalid); err == nil {
		t.Fatal("writeConfigFile accepted invalid active_profile")
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after failed write: %v", err)
	}
	if string(raw) != original {
		t.Fatalf("config changed after failed validation: %s", raw)
	}

	nestedPath := filepath.Join(dir, "missing-parent", "config.json")
	if err := writeConfigFile(nestedPath, invalid); err == nil {
		t.Fatal("writeConfigFile accepted invalid config for missing parent")
	}
	if _, err := os.Stat(filepath.Dir(nestedPath)); !os.IsNotExist(err) {
		t.Fatalf("invalid config created parent dir, stat err = %v", err)
	}

	badLanguage := coreconfig.Config{Profiles: map[string]coreconfig.Profile{"prod": {Language: "fr-FR"}}}
	if err := writeConfigFile(configPath, badLanguage); err == nil {
		t.Fatal("writeConfigFile accepted unsupported profile language")
	}

	emptyProfile := coreconfig.Config{Profiles: map[string]coreconfig.Profile{"": {Region: "cn"}}}
	if err := writeConfigFile(configPath, emptyProfile); err == nil {
		t.Fatal("writeConfigFile accepted empty profile name")
	}

	negativeTimeout := coreconfig.Config{Profiles: map[string]coreconfig.Profile{"prod": {TimeoutSeconds: -1}}}
	if err := writeConfigFile(configPath, negativeTimeout); err == nil {
		t.Fatal("writeConfigFile accepted negative timeout_seconds")
	}

	if err := validateConfigBytes([]byte(`{"secret":"value"}`)); err == nil {
		t.Fatal("validateConfigBytes accepted unsupported secret field")
	}
	if _, err := validatedConfigJSONWith(coreconfig.Config{}, func([]byte) error {
		return errors.New("validation failed")
	}); err == nil {
		t.Fatal("validatedConfigJSONWith ignored serialized validation error")
	}
}

func runConfigForTest(t *testing.T, configPath string, stdin *strings.Reader, args ...string) {
	t.Helper()

	var stdout bytes.Buffer
	err := Run(Config{
		Args:       args,
		Stdout:     &stdout,
		Stdin:      stdin,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("%v returned error: %v\nstdout: %s", args, err, stdout.String())
	}
}

func readConfigForTest(t *testing.T, path string) coreconfig.Config {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	cfg, err := coreconfig.Load(raw)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

type configFailingReader struct{}

func (configFailingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

type ioDiscardForConfigTest struct{}

func (ioDiscardForConfigTest) Write(p []byte) (int, error) {
	return len(p), nil
}
