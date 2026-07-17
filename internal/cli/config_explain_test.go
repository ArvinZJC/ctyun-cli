/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
)

func TestConfigExplainReportsSourcesWithoutCredentials(t *testing.T) {
	raw := []byte(`{"active_profile":"prod","profiles":{"prod":{"region":"region-1","ak":"profile-ak","sk":"profile-sk","registry_public_key":"registry-key"}}}`)
	var stdout bytes.Buffer
	err := Run(Config{
		Args: []string{"config", "explain", "--output", "json"}, Stdout: &stdout, Config: raw,
		Env: func(key string) string {
			if key == "CTYUN_AK" {
				return "environment-ak"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rendered := stdout.String()
	for _, secret := range []string{"environment-ak", "profile-ak", "profile-sk", "registry-key", "*****"} {
		if strings.Contains(rendered, secret) {
			t.Fatalf("output leaked %q: %s", secret, rendered)
		}
	}
	for _, want := range []string{`"key": "region"`, `"source": "profile"`, `"configured": true`, `"sensitive": true`, `"effective": false`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output missing %q: %s", want, rendered)
		}
	}
	var report configExplainJSONReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || len(report.Settings) != len(configExplainSettingKeys()) {
		t.Fatalf("JSON report = %#v, err = %v", report, err)
	}
	for _, setting := range report.Settings {
		if setting.Sensitive && setting.Value != nil {
			t.Fatalf("sensitive setting retained value: %#v", setting)
		}
	}
}

func TestConfigExplainPreservesActualLanguageProvenance(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args: []string{"config", "explain", "language", "--output", "json"}, Stdout: &stdout,
		Config: []byte(`{}`),
		Env: func(key string) string {
			if key == "LANG" {
				return "en_GB.UTF-8"
			}
			return ""
		},
	}); err != nil {
		t.Fatal(err)
	}
	var report configExplainJSONReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || len(report.Settings) != 1 {
		t.Fatalf("report = %#v, err = %v", report, err)
	}
	if setting := report.Settings[0]; setting.Value == nil || *setting.Value != "en-GB" || setting.Source != "os" {
		t.Fatalf("OS-derived language = %#v", setting)
	}

	stdout.Reset()
	if err := Run(Config{
		Args: []string{"config", "explain", "language", "--lang", "zh-CN", "--output", "json"}, Stdout: &stdout,
		Config: []byte(`{}`), Env: func(string) string { return "" },
	}); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Settings[0].Source != "option" {
		t.Fatalf("option-derived language = %#v, err = %v", report.Settings, err)
	}
}

func TestConfigExplainLocalizesStaticSourceNames(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{
		Args: []string{"config", "explain", "config_path", "--lang", "zh-CN"}, Stdout: &stdout,
		Config: []byte(`{}`), Env: func(string) string { return "" },
	}); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "默认值（默认配置路径）") || strings.Contains(got, "default config path") {
		t.Fatalf("localized source output = %q", got)
	}

	stdout.Reset()
	if err := Run(Config{
		Args: []string{"config", "explain", "config_path", "--lang", "zh-CN"}, Stdout: &stdout,
		Config: []byte(`{}`), ConfigPath: "/tmp/config.json", Env: func(string) string { return "" },
	}); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "进程（嵌入的配置路径）") || strings.Contains(got, "embedded config path") {
		t.Fatalf("localized process source output = %q", got)
	}
}

func TestConfigExplainSelectsOneSettingAndRejectsInvalidArguments(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"config", "explain", "region", "--output", "json"}, Stdout: &stdout, Config: []byte(`{"profiles":{"dev":{"region":"region-1"}}}`)}); err != nil {
		t.Fatal(err)
	}
	if strings.Count(stdout.String(), `"key":`) != 1 || !strings.Contains(stdout.String(), `"key": "region"`) {
		t.Fatalf("one-setting output = %s", stdout.String())
	}
	if err := Run(Config{Args: []string{"config", "explain", "unknown"}}); err == nil || err.Error() != "error.unsupported_config_setting" {
		t.Fatalf("unknown setting error = %v", err)
	}
	if err := Run(Config{Args: []string{"config", "explain", "region", "extra"}}); err == nil || err.Error() != "error.unexpected_argument" {
		t.Fatalf("extra argument error = %v", err)
	}
}

func TestConfigExplainRendersLocalizedTableAndOutputControls(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"config", "explain", "--lang", "en-US", "--cols", "Setting,Source", "--filter", "Setting=region", "--no-header"},
		Stdout: &stdout,
		Config: []byte(`{"profiles":{"dev":{"region":"region-1"}}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if !strings.Contains(got, "region") || !strings.Contains(got, "profile") || strings.Contains(got, "Setting") || strings.Contains(got, "language") {
		t.Fatalf("filtered table = %q", got)
	}
}

func TestConfigExplainHelpCompletionAndOptionScope(t *testing.T) {
	assertEqualCompletions(t, commandCompletions([]string{"config", "explain"}, completionContext{}), configExplainSettingKeys())
	completions := commandCompletions([]string{"config"}, completionContext{})
	if !strings.Contains(strings.Join(completions, " "), "explain") {
		t.Fatalf("config completions = %v", completions)
	}
	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"help", "config", "explain"}, Stdout: &stdout, Env: func(string) string { return "" }}); err != nil {
		t.Fatal(err)
	}
	help := stdout.String()
	for _, want := range []string{"ctyun [global options] config explain [{key}]", "Arguments:", "Global Options:"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help missing %q:\n%s", want, help)
		}
	}
	if count := strings.Count(help, "Global Options:"); count != 1 {
		t.Fatalf("config explain help has %d Global Options sections:\n%s", count, help)
	}
	for _, option := range []string{"--timeout", "--debug"} {
		if err := Run(Config{Args: []string{"config", "explain", option, "1"}}); err == nil || err.Error() != "error.unknown_option" {
			t.Fatalf("%s error = %v", option, err)
		}
	}
}

func TestConfigExplainCoversCommandAndRendererErrors(t *testing.T) {
	input := configCommandInput{Raw: []byte(`{}`), Getenv: func(string) string { return "" }}
	if err := runConfigExplain(io.Discard, []string{"--offline"}, globalOptions{Output: "table", Language: "en-US"}, input); err == nil {
		t.Fatal("runConfigExplain accepted an undeclared option")
	}
	if err := runConfigExplain(io.Discard, nil, globalOptions{Output: "table", Language: "en-US"}, configCommandInput{Raw: []byte(`{"profiles":`)}); err == nil {
		t.Fatal("runConfigExplain accepted malformed config")
	}

	settings := []coreconfig.Setting{{Key: "region", Value: "region-1", Configured: true, Effective: true, Source: coreconfig.Source{Kind: coreconfig.SourceProfile, Name: "dev"}}}
	assertReportRendererErrors(t, func(writer io.Writer, opts globalOptions) error {
		return renderConfigExplanation(writer, settings, opts)
	})
}

// configExplainSettingKeys returns the expected sorted completion catalogue.
func configExplainSettingKeys() []string {
	return []string{"ak", "config_path", "endpoint_url", "language", "profile", "region", "registry_public_key", "registry_url", "sk", "timeout_seconds", "warn_config_credentials", "warn_deprecated"}
}
