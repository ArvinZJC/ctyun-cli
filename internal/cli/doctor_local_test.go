/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
	"github.com/ArvinZJC/ctyun-cli/internal/localdoctor"
	"github.com/mattn/go-runewidth"
)

func TestDoctorLocalReportsMalformedConfigBeforeProfileDispatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"doctor", "local", "--output", "json"},
		Stdout: &stdout, Stderr: &stderr, Config: []byte(`{"profiles":`),
		PluginRoot: filepath.Join(t.TempDir(), "missing"),
		Env:        func(string) string { return "" },
	})
	if code != 1 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"key": "config_file"`) || !strings.Contains(stdout.String(), `"status": "failed"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	var report doctorLocalJSONReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Summary.Failed == 0 || len(report.Results) < 6 {
		t.Fatalf("report = %#v, err = %v", report, err)
	}
}

func TestDoctorLocalWarningsAndSkippedFindingsExitZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"doctor", "local", "--lang", "en-US"},
		Stdout: &stdout, Stderr: &stderr,
		ConfigPath: filepath.Join(t.TempDir(), "missing.json"),
		PluginRoot: filepath.Join(t.TempDir(), "missing-plugins"),
		Env: func(key string) string {
			if key == "CTYUN_AK" || key == "CTYUN_SK" {
				return "configured"
			}
			return ""
		},
	})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Local diagnostics:") || !strings.Contains(stdout.String(), "warnings 1") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestDoctorLocalTableFitsInteractiveTerminalWidth(t *testing.T) {
	original := defaultTerminalCapabilities
	defaultTerminalCapabilities = terminalCapabilities{
		Interactive: func(io.Writer) bool { return true },
		Width:       func(io.Writer) (int, error) { return 60, nil },
	}
	t.Cleanup(func() { defaultTerminalCapabilities = original })
	var stdout bytes.Buffer
	if err := Run(Config{
		Args: []string{"doctor", "local", "--lang", "zh-CN"}, Stdout: &stdout,
		Config:     []byte(`{"ak":"stored-ak","sk":"stored-sk"}`),
		PluginRoot: filepath.Join(t.TempDir(), "missing"), Env: func(string) string { return "" },
	}); err != nil {
		t.Fatal(err)
	}
	for line := range strings.SplitSeq(strings.TrimSuffix(stdout.String(), "\n"), "\n") {
		if width := runewidth.StringWidth(line); width > 60 {
			t.Fatalf("output line width = %d, want <= 60:\n%s", width, stdout.String())
		}
	}
}

func TestDoctorLocalCredentialDetailsAreActionableAndLocalized(t *testing.T) {
	tests := []struct {
		name     string
		language string
		raw      string
		env      func(string) string
		want     string
	}{
		{
			name: "stored English", language: "en-US", raw: `{"ak":"stored-ak","sk":"stored-sk"}`,
			env:  func(string) string { return "" },
			want: "AK and SK resolve from config; environment variables avoid keeping credentials on disk",
		},
		{
			name: "stored Chinese", language: "zh-CN", raw: `{"ak":"stored-ak","sk":"stored-sk"}`,
			env:  func(string) string { return "" },
			want: "AK 和 SK 均来自配置；使用环境变量可避免将凭据保存在磁盘上",
		},
		{
			name: "mixed Chinese", language: "zh-CN", raw: `{"sk":"stored-sk"}`,
			env: func(key string) string {
				if key == "CTYUN_AK" {
					return "environment-ak"
				}
				return ""
			},
			want: "AK 和 SK 来自不同来源（环境变量、配置）；至少一个凭据仍保存在配置文件中",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			if err := Run(Config{
				Args: []string{"doctor", "local", "--lang", tc.language, "--output", "json"}, Stdout: &stdout,
				Config: []byte(tc.raw), PluginRoot: filepath.Join(t.TempDir(), "missing"), Env: tc.env,
			}); err != nil {
				t.Fatal(err)
			}
			var report doctorLocalJSONReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			for _, result := range report.Results {
				if result.Key == string(localdoctor.CheckCredentials) {
					if result.Detail != tc.want {
						t.Fatalf("credential detail = %q, want %q", result.Detail, tc.want)
					}
					return
				}
			}
			t.Fatal("credential finding missing")
		})
	}
}

func TestDoctorLocalFilteringKeepsCompleteSummaryAndExit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"doctor", "local", "--output", "json", "--filter", "status=passed"},
		Stdout: &stdout, Stderr: &stderr,
		Config:     []byte(`{"profiles":{"dev":{}}}`),
		PluginRoot: filepath.Join(t.TempDir(), "missing"),
		Env:        func(string) string { return "" },
	})
	if code != 1 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var report doctorLocalJSONReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Summary.Failed == 0 {
		t.Fatalf("summary = %#v", report.Summary)
	}
	for _, result := range report.Results {
		if result.Status != "passed" {
			t.Fatalf("filtered result = %#v", result)
		}
	}
}

func TestDoctorLocalHelpAndOptionScopeIgnoreMalformedConfig(t *testing.T) {
	for _, args := range [][]string{{"doctor", "local", "--help"}, {"help", "doctor", "local"}} {
		var stdout bytes.Buffer
		if err := Run(Config{Args: args, Stdout: &stdout, Config: []byte(`{"profiles":`), Env: func(string) string { return "" }}); err != nil {
			t.Fatalf("Run(%v) = %v", args, err)
		}
		for _, want := range []string{"ctyun [global options] doctor local", "Global Options:"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("help missing %q:\n%s", want, stdout.String())
			}
		}
	}
	for _, args := range [][]string{{"doctor", "local", "extra"}, {"doctor", "local", "--timeout", "1"}, {"doctor", "local", "--offline"}} {
		if err := Run(Config{Args: args, Stdout: io.Discard, Env: func(string) string { return "" }}); err == nil {
			t.Fatalf("Run(%v) returned nil error", args)
		}
	}
}

func TestDoctorLocalCompletionListsLocalBeforeNetwork(t *testing.T) {
	got := commandCompletions([]string{"doctor"}, completionContext{})
	if strings.Join(got, " ") != "local network" {
		t.Fatalf("doctor completions = %v", got)
	}
}

func TestDoctorLocalCoversHelpAndRendererErrors(t *testing.T) {
	var stdout bytes.Buffer
	if err := Run(Config{Args: []string{"help", "doctor", "local"}, Stdout: &stdout, ConfigPath: t.TempDir(), Env: func(string) string { return "" }}); err != nil {
		t.Fatalf("help with unreadable config = %v", err)
	}
	if err := Run(Config{Args: []string{"help", "doctor", "local", "extra"}, Stdout: io.Discard, Env: func(string) string { return "" }}); err == nil {
		t.Fatal("doctor local help accepted an extra argument")
	}

	report := localdoctor.Report{
		Findings: []localdoctor.Finding{{Key: localdoctor.CheckConfigFile, Target: "/config", Status: doctor.StatusPassed, DetailKey: "doctor.local.detail.config_passed"}},
		Counts:   doctor.Counts{Passed: 1}, Language: "en-US",
	}
	assertReportRendererErrors(t, func(writer io.Writer, opts globalOptions) error {
		return renderDoctorLocalReport(writer, report, opts)
	})
	if err := runDoctorLocal(io.Discard, localdoctor.Input{UseInjectedConfig: true, Getenv: func(string) string { return "" }}, localdoctor.Dependencies{}, globalOptions{Output: "table", Filter: "missing=value"}); err == nil {
		t.Fatal("runDoctorLocal ignored a render error")
	}

	silentExitError{}.silentExit()
}

func TestAbbreviateHomePathCoversHomeStates(t *testing.T) {
	t.Setenv("HOME", "")
	if got := abbreviateHomePath("/path"); got != "/path" {
		t.Fatalf("path without home = %q", got)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := abbreviateHomePath(home); got != "~" {
		t.Fatalf("home path = %q", got)
	}
	child := filepath.Join(home, "config.json")
	if got := abbreviateHomePath(child); got != filepath.Join("~", "config.json") {
		t.Fatalf("child path = %q", got)
	}
	if got := abbreviateHomePath(""); got != "" {
		t.Fatalf("empty path = %q", got)
	}
}
