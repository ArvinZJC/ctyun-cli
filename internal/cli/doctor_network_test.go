/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/networkdoctor"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestParseDoctorNetworkOptions(t *testing.T) {
	for _, source := range []string{"auto", "github", "gitee"} {
		opts, err := parseDoctorNetworkOptions([]string{"network", "--source", source})
		if err != nil || opts.Source != source {
			t.Fatalf("source %q: opts = %#v, err = %v", source, opts, err)
		}
	}
	for _, args := range [][]string{{"network", "--source"}, {"network", "--source", "local"}, {"network", "--bad"}, {"unknown"}} {
		if _, err := parseDoctorNetworkOptions(args); err == nil {
			t.Fatalf("parseDoctorNetworkOptions(%v) returned nil error", args)
		}
	}
}

func TestDoctorEndpointURLsUseDocumentedPrecedence(t *testing.T) {
	bundles := []plugin.Bundle{
		{Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://second.example.test"}}},
		{Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://first.example.test"}}},
		{Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://second.example.test/"}}},
	}
	if got := doctorEndpointURLs(coreconfig.Profile{EndpointURL: "https://profile.example.test"}, bundles); !slices.Equal(got, []string{"https://profile.example.test"}) {
		t.Fatalf("profile endpoints = %v", got)
	}
	if got := doctorEndpointURLs(coreconfig.Profile{}, bundles); !slices.Equal(got, []string{"https://first.example.test", "https://second.example.test"}) {
		t.Fatalf("bundle endpoints = %v", got)
	}
	if got := doctorEndpointURLs(coreconfig.Profile{}, nil); len(got) != 0 {
		t.Fatalf("empty plugin endpoints = %v", got)
	}
}

func TestDoctorNetworkReportUsesOnlyUserLevelChecks(t *testing.T) {
	results := []networkdoctor.Result{
		{Check: networkdoctor.Check{Kind: networkdoctor.CheckSource, Subject: "core", SourceName: "github", Target: "https://github.example.test"}, Status: networkdoctor.StatusPassed, DetailKey: "doctor.detail.source_passed"},
		{Check: networkdoctor.Check{Kind: networkdoctor.CheckEndpoint, Subject: "ctyun", Target: "https://ctapi.example.test"}, Status: networkdoctor.StatusPassed, DetailKey: "doctor.detail.endpoint_passed"},
	}
	report := networkdoctor.Report{Results: results, Counts: networkdoctor.Count(results)}
	var stdout bytes.Buffer
	if err := renderDoctorNetworkReport(&stdout, report, globalOptions{Output: "table", Language: "en-US", Table: "compact"}); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	for _, want := range []string{"Core update source", "Signed index verified", "CTyun API endpoint", "HTTPS reachable", "https://ctapi.example.test"} {
		if !strings.Contains(got, want) {
			t.Fatalf("report missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"public key configuration", "capability", "ctyun-https---"} {
		if strings.Contains(strings.ToLower(got), unwanted) {
			t.Fatalf("report contains internal value %q:\n%s", unwanted, got)
		}
	}
}

func TestDoctorNetworkProgressHidesInternalCheckNames(t *testing.T) {
	for _, check := range []networkdoctor.Check{{Kind: networkdoctor.CheckConfiguration, Subject: "public-key"}, {Kind: networkdoctor.CheckCapability, Subject: "core-source"}} {
		got := doctorProgressText(check, "en-US")
		if strings.Contains(strings.ToLower(got), "public key") || strings.Contains(strings.ToLower(got), "capability") {
			t.Fatalf("progress = %q for %#v", got, check)
		}
	}
}

func TestDoctorNetworkBuildsResolvedPlanAndTimeout(t *testing.T) {
	originalRunner := runNetworkDoctor
	originalDependencies := doctorNetworkDependenciesFactory
	originalWriter := writeDoctorNetworkReport
	t.Cleanup(func() {
		runNetworkDoctor = originalRunner
		doctorNetworkDependenciesFactory = originalDependencies
		writeDoctorNetworkReport = originalWriter
	})
	var captured networkdoctor.Plan
	var timeout time.Duration
	runNetworkDoctor = func(_ context.Context, plan networkdoctor.Plan, gotTimeout time.Duration, _ networkdoctor.Dependencies, _ networkdoctor.Observer) (networkdoctor.Report, error) {
		captured = plan
		timeout = gotTimeout
		return networkdoctor.Report{}, nil
	}
	doctorNetworkDependenciesFactory = func(http.RoundTripper) networkdoctor.Dependencies { return networkdoctor.Dependencies{} }
	writeDoctorNetworkReport = func(io.Writer, networkdoctor.Report, globalOptions) error { return nil }
	getenv := func(key string) string {
		switch key {
		case "CTYUN_UPGRADE_SOURCE":
			return "github"
		case "CTYUN_PLUGIN_SOURCE":
			return "gitee"
		case "CTYUN_RELEASE_PUBLIC_KEY":
			return "public-key"
		default:
			return ""
		}
	}
	err := Run(Config{
		Args:   []string{"--timeout", "7", "doctor", "network"},
		Stdout: io.Discard,
		Stderr: io.Discard,
		Env:    getenv,
		Config: []byte(`{"profiles":{"prod":{"endpoint_url":"https://profile.example.test"}},"active_profile":"prod"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if timeout != 7*time.Second || len(captured.Capabilities) != 3 {
		t.Fatalf("timeout = %s, plan = %#v", timeout, captured)
	}
	if sourceNames(captured, "core") != "github" || sourceNames(captured, "plugin") != "gitee" {
		t.Fatalf("plan sources = %#v", captured.Checks)
	}
}

func TestDoctorNetworkSourceFlagOverridesBothSources(t *testing.T) {
	originalRunner := runNetworkDoctor
	originalWriter := writeDoctorNetworkReport
	t.Cleanup(func() { runNetworkDoctor = originalRunner; writeDoctorNetworkReport = originalWriter })
	var captured networkdoctor.Plan
	runNetworkDoctor = func(_ context.Context, plan networkdoctor.Plan, _ time.Duration, _ networkdoctor.Dependencies, _ networkdoctor.Observer) (networkdoctor.Report, error) {
		captured = plan
		return networkdoctor.Report{}, nil
	}
	writeDoctorNetworkReport = func(io.Writer, networkdoctor.Report, globalOptions) error { return nil }
	err := Run(Config{
		Args:   []string{"doctor", "network", "--source", "gitee"},
		Stdout: io.Discard,
		Stderr: io.Discard,
		Config: []byte(`{"profiles":{"prod":{"endpoint_url":"https://profile.example.test"}},"active_profile":"prod"}`),
		Env: func(key string) string {
			if key == "CTYUN_UPGRADE_SOURCE" || key == "CTYUN_PLUGIN_SOURCE" {
				return "github"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sourceNames(captured, "core") != "gitee" || sourceNames(captured, "plugin") != "gitee" {
		t.Fatalf("plan sources = %#v", captured.Checks)
	}
}

func TestDoctorNetworkRejectsOfflineAliasesBeforeRunning(t *testing.T) {
	originalRunner := runNetworkDoctor
	t.Cleanup(func() { runNetworkDoctor = originalRunner })
	for _, flag := range []string{"--offline", "--fixture", "-O"} {
		t.Run(flag, func(t *testing.T) {
			called := false
			runNetworkDoctor = func(context.Context, networkdoctor.Plan, time.Duration, networkdoctor.Dependencies, networkdoctor.Observer) (networkdoctor.Report, error) {
				called = true
				return networkdoctor.Report{}, nil
			}
			err := Run(Config{Args: []string{"doctor", "network", flag}, Stdout: io.Discard, Stderr: io.Discard})
			if err == nil || called {
				t.Fatalf("error = %v, called = %t", err, called)
			}
			var diagnosticErr interface{ MessageKey() string }
			if !errors.As(err, &diagnosticErr) || diagnosticErr.MessageKey() != "error.unknown_option" {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestDoctorNetworkFailureDoesNotReadCredentials(t *testing.T) {
	originalRunner := runNetworkDoctor
	t.Cleanup(func() { runNetworkDoctor = originalRunner })
	runNetworkDoctor = func(context.Context, networkdoctor.Plan, time.Duration, networkdoctor.Dependencies, networkdoctor.Observer) (networkdoctor.Report, error) {
		return networkdoctor.Report{}, errors.New("runner failed")
	}
	var credentialReads []string
	var stderr bytes.Buffer
	code := Execute(Config{
		Args:   []string{"doctor", "network", "--source", "github"},
		Stdout: io.Discard,
		Stderr: &stderr,
		Config: []byte(`{"profiles":{"prod":{"endpoint_url":"https://profile.example.test","ak":"profile-ak","sk":"profile-sk"}},"active_profile":"prod"}`),
		Env: func(key string) string {
			if key == "CTYUN_AK" || key == "CTYUN_SK" {
				credentialReads = append(credentialReads, key)
			}
			return ""
		},
	})
	if code != 1 || len(credentialReads) != 0 {
		t.Fatalf("code = %d, credential reads = %v, stderr = %q", code, credentialReads, stderr.String())
	}
}

func TestDoctorNetworkDiagnosticFailureExitsSilently(t *testing.T) {
	originalRunner := runNetworkDoctor
	t.Cleanup(func() { runNetworkDoctor = originalRunner })
	runNetworkDoctor = func(context.Context, networkdoctor.Plan, time.Duration, networkdoctor.Dependencies, networkdoctor.Observer) (networkdoctor.Report, error) {
		return doctorReportFixture(), nil
	}
	for _, format := range []string{"table", "json"} {
		t.Run(format, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Execute(Config{
				Args:   []string{"--output", format, "doctor", "network", "--source", "github"},
				Stdout: &stdout,
				Stderr: &stderr,
				Config: []byte(`{"profiles":{"prod":{"endpoint_url":"https://profile.example.test"}},"active_profile":"prod"}`),
			})
			if code != 1 || stderr.Len() != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if format == "json" {
				var payload doctorNetworkJSONReport
				if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil || len(payload.Results) == 0 {
					t.Fatalf("JSON report = %q, %v", stdout.String(), err)
				}
			} else if !strings.Contains(stdout.String(), "Network diagnostics:") {
				t.Fatalf("table report = %q", stdout.String())
			}
		})
	}
}

func TestDoctorNetworkOrdinaryErrorStillUsesStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute(Config{Args: []string{"doctor", "network", "--bad"}, Stdout: &stdout, Stderr: &stderr})
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "Error: unknown option \"--bad\"") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestDoctorNetworkReportRendersLocalizedTableAndCompleteSummary(t *testing.T) {
	report := doctorReportFixture()
	var stdout bytes.Buffer
	err := renderDoctorNetworkReport(&stdout, report, globalOptions{Output: "table", Language: "en-US", Table: "compact", Filter: "Status=Passed"})
	if err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	for _, want := range []string{"Check", "Target", "Status", "Duration", "Detail", "Passed", "Network diagnostics: passed 1; warnings 1; failed 1; skipped 1."} {
		if !strings.Contains(got, want) {
			t.Fatalf("table output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "https://gitee.example.test") {
		t.Fatalf("filtered table contains warning row:\n%s", got)
	}

	stdout.Reset()
	err = renderDoctorNetworkReport(&stdout, report, globalOptions{Output: "table", Language: "zh-CN", Table: "compact", Columns: []string{"检查", "状态", "详情"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"检查", "状态", "详情", "通过", "警告", "失败", "已跳过", "网络诊断完成：通过 1 项；警告 1 项；失败 1 项；跳过 1 项。"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("Chinese table missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestDoctorNetworkReportRendersOneJSONDocument(t *testing.T) {
	var stdout bytes.Buffer
	err := renderDoctorNetworkReport(&stdout, doctorReportFixture(), globalOptions{Output: "json", Language: "en-GB", Filter: "Status=Passed"})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Results []map[string]string  `json:"results"`
		Summary networkdoctor.Counts `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("JSON output is not one document: %v\n%s", err, stdout.String())
	}
	if len(payload.Results) != 1 || payload.Summary != (networkdoctor.Counts{Passed: 1, Warning: 1, Failed: 1, Skipped: 1}) {
		t.Fatalf("payload = %#v", payload)
	}
	if strings.Contains(stdout.String(), "\x1b") || strings.Contains(stdout.String(), "Network diagnostics:") {
		t.Fatalf("JSON output contains terminal or text summary data:\n%s", stdout.String())
	}
}

func TestDoctorNetworkUsesProgressAndReportsBeforeFailure(t *testing.T) {
	originalRunner := runNetworkDoctor
	originalFactory := operationProgressFactory
	originalWriter := writeDoctorNetworkReport
	t.Cleanup(func() {
		runNetworkDoctor = originalRunner
		operationProgressFactory = originalFactory
		writeDoctorNetworkReport = originalWriter
	})
	display := &recordingOperationDisplay{}
	operationProgressFactory = func(io.Writer) operationDisplay { return display }
	written := false
	writeDoctorNetworkReport = func(io.Writer, networkdoctor.Report, globalOptions) error {
		written = true
		return nil
	}
	runNetworkDoctor = func(_ context.Context, plan networkdoctor.Plan, _ time.Duration, _ networkdoctor.Dependencies, observer networkdoctor.Observer) (networkdoctor.Report, error) {
		for index, check := range plan.Checks {
			if err := observer(networkdoctor.Progress{Completed: index + 1, Total: len(plan.Checks), Check: check}); err != nil {
				return networkdoctor.Report{}, err
			}
		}
		return networkdoctor.Report{FailedCapabilities: []string{"core-source"}}, nil
	}
	err := Run(Config{
		Args:   []string{"doctor", "network", "--source", "github"},
		Stdout: io.Discard,
		Stderr: io.Discard,
		Config: []byte(`{"profiles":{"prod":{"endpoint_url":"https://profile.example.test"}},"active_profile":"prod"}`),
	})
	if err == nil || !written || !display.cleared || len(display.updates) == 0 {
		t.Fatalf("error = %v, written = %t, display = %#v", err, written, display)
	}
	var silent interface{ silentExit() }
	if !errors.As(err, &silent) {
		t.Fatalf("result error = %v, want silent exit", err)
	}
	if err.Error() != "command result requires a non-zero exit status" {
		t.Fatalf("result error fallback = %q", err.Error())
	}
}

func doctorReportFixture() networkdoctor.Report {
	results := []networkdoctor.Result{
		{Check: networkdoctor.Check{Kind: networkdoctor.CheckSource, Subject: "core", SourceName: "github", Target: "https://github.example.test"}, Status: networkdoctor.StatusPassed, Duration: 12 * time.Millisecond, DetailKey: "doctor.detail.source_passed"},
		{Check: networkdoctor.Check{Kind: networkdoctor.CheckSource, Subject: "plugin", SourceName: "gitee", Target: "https://gitee.example.test"}, Status: networkdoctor.StatusWarning, Duration: 25 * time.Millisecond, Category: networkdoctor.FailureConnection, DetailKey: "doctor.detail.connection"},
		{Check: networkdoctor.Check{Kind: networkdoctor.CheckEndpoint, Subject: "ctyun", Target: "https://ctapi.example.test"}, Status: networkdoctor.StatusFailed, Duration: time.Second, Category: networkdoctor.FailureTLS, DetailKey: "doctor.detail.tls"},
		{Check: networkdoctor.Check{Kind: networkdoctor.CheckSource, Subject: "core", SourceName: "custom", Target: "https://custom.example.test"}, Status: networkdoctor.StatusSkipped, DetailKey: "doctor.detail.dependency"},
	}
	return networkdoctor.Report{Results: results, Counts: networkdoctor.Count(results), FailedCapabilities: []string{"ctyun"}}
}

func TestRunDoctorCoversSourceEndpointAndDisplayErrors(t *testing.T) {
	baseGlobal := globalOptions{Language: "en-US", Output: "table"}
	if err := runDoctor(io.Discard, io.Discard, []string{"network"}, t.TempDir(), coreconfig.Profile{EndpointURL: "https://profile.example.test"}, func(key string) string {
		if key == "CTYUN_UPGRADE_SOURCE" {
			return "invalid"
		}
		return ""
	}, nil, baseGlobal); err == nil {
		t.Fatal("runDoctor accepted an invalid core source")
	}
	if err := runDoctor(io.Discard, io.Discard, []string{"network"}, t.TempDir(), coreconfig.Profile{EndpointURL: "https://profile.example.test"}, func(key string) string {
		if key == "CTYUN_UPGRADE_SOURCE" {
			return "github"
		}
		if key == "CTYUN_PLUGIN_SOURCE" {
			return "invalid"
		}
		return ""
	}, nil, baseGlobal); err == nil {
		t.Fatal("runDoctor accepted an invalid plugin source")
	}
	if err := runDoctor(io.Discard, io.Discard, []string{"network", "--source", "github"}, t.TempDir(), coreconfig.Profile{EndpointURL: "http://profile.example.test"}, func(string) string { return "" }, nil, baseGlobal); err == nil {
		t.Fatal("runDoctor accepted an unsafe profile endpoint")
	}

	root := t.TempDir()
	badBundle := filepath.Join(root, "bad")
	if err := os.MkdirAll(badBundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badBundle, "plugin.json"), []byte(`{"name":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveDoctorEndpointURLs(root, coreconfig.Profile{}); err == nil {
		t.Fatal("resolveDoctorEndpointURLs ignored malformed plugin metadata")
	}
	if err := runDoctor(io.Discard, io.Discard, []string{"network", "--source", "github"}, root, coreconfig.Profile{}, func(string) string { return "" }, nil, baseGlobal); err == nil {
		t.Fatal("runDoctor ignored malformed plugin metadata")
	}
	if endpoints, err := resolveDoctorEndpointURLs(t.TempDir(), coreconfig.Profile{}); err != nil || len(endpoints) == 0 {
		t.Fatalf("bundled endpoint resolution = %v, %v", endpoints, err)
	}

	originalFactory := operationProgressFactory
	originalRunner := runNetworkDoctor
	originalWriter := writeDoctorNetworkReport
	t.Cleanup(func() {
		operationProgressFactory = originalFactory
		runNetworkDoctor = originalRunner
		writeDoctorNetworkReport = originalWriter
	})
	startErr := errors.New("start")
	operationProgressFactory = func(io.Writer) operationDisplay { return &doctorErrorDisplay{startErr: startErr} }
	if err := runDoctor(io.Discard, io.Discard, []string{"network", "--source", "github"}, t.TempDir(), coreconfig.Profile{EndpointURL: "https://profile.example.test"}, func(string) string { return "" }, nil, baseGlobal); !errors.Is(err, startErr) {
		t.Fatalf("start error = %v", err)
	}

	clearErr := errors.New("clear")
	operationProgressFactory = func(io.Writer) operationDisplay { return &doctorErrorDisplay{clearErr: clearErr} }
	runNetworkDoctor = func(context.Context, networkdoctor.Plan, time.Duration, networkdoctor.Dependencies, networkdoctor.Observer) (networkdoctor.Report, error) {
		return networkdoctor.Report{}, nil
	}
	if err := runDoctor(io.Discard, io.Discard, []string{"network", "--source", "github"}, t.TempDir(), coreconfig.Profile{EndpointURL: "https://profile.example.test"}, func(string) string { return "" }, nil, baseGlobal); !errors.Is(err, clearErr) {
		t.Fatalf("clear error = %v", err)
	}

	operationProgressFactory = func(io.Writer) operationDisplay { return &doctorErrorDisplay{} }
	writeErr := errors.New("report")
	writeDoctorNetworkReport = func(io.Writer, networkdoctor.Report, globalOptions) error { return writeErr }
	if err := runDoctor(io.Discard, io.Discard, []string{"network", "--source", "github"}, t.TempDir(), coreconfig.Profile{EndpointURL: "https://profile.example.test"}, func(string) string { return "" }, nil, baseGlobal); !errors.Is(err, writeErr) {
		t.Fatalf("report error = %v", err)
	}
}

func TestDoctorNetworkReportCoversOutputErrorsAndFallbackLabels(t *testing.T) {
	report := doctorReportFixture()
	if err := renderDoctorNetworkReport(io.Discard, report, globalOptions{Output: "table", Language: "en-US", Filter: "missing=value"}); err == nil {
		t.Fatal("renderer accepted an unknown filter")
	}
	if err := renderDoctorNetworkReport(io.Discard, report, globalOptions{Output: "table", Language: "en-US", Sort: "missing"}); err == nil {
		t.Fatal("renderer accepted an unknown sort")
	}
	if err := renderDoctorNetworkReport(io.Discard, report, globalOptions{Output: "xml", Language: "en-US"}); err == nil {
		t.Fatal("renderer accepted an unsupported output")
	}
	originalJSON := renderOutputJSON
	originalTable := renderOutputTable
	t.Cleanup(func() { renderOutputJSON = originalJSON; renderOutputTable = originalTable })
	want := errors.New("render")
	renderOutputJSON = func(any) (string, error) { return "", want }
	if err := renderDoctorNetworkReport(io.Discard, report, globalOptions{Output: "json", Language: "en-US"}); !errors.Is(err, want) {
		t.Fatalf("JSON render error = %v", err)
	}
	renderOutputTable = func([]map[string]string, []output.Column, output.TableOptions) (string, error) { return "", want }
	if err := renderDoctorNetworkReport(io.Discard, report, globalOptions{Output: "table", Language: "en-US"}); !errors.Is(err, want) {
		t.Fatalf("table render error = %v", err)
	}
	renderOutputTable = originalTable
	if err := renderDoctorNetworkReport(failingWriter{}, report, globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("renderer ignored stdout failure")
	}
	if got := doctorSubjectText("custom", "en-US"); got != "custom" {
		t.Fatalf("custom subject = %q", got)
	}
	if got := sourceDisplayName("custom"); got != "custom" {
		t.Fatalf("custom source = %q", got)
	}
	if got := sourceDisplayName("gitee"); got != "Gitee" {
		t.Fatalf("Gitee source = %q", got)
	}
	if got := doctorDetailText(networkdoctor.Result{}, "en-US"); got != "Available" {
		t.Fatalf("empty detail = %q", got)
	}
	if got := commandCompletions([]string{"doctor", "network", "extra"}, completionContext{}); got != nil {
		t.Fatalf("extra doctor completions = %v", got)
	}
}

type doctorErrorDisplay struct {
	startErr error
	clearErr error
}

func (display *doctorErrorDisplay) Start(int) error  { return display.startErr }
func (*doctorErrorDisplay) Update(int, string) error { return nil }
func (display *doctorErrorDisplay) Clear() error     { return display.clearErr }

func sourceNames(plan networkdoctor.Plan, subject string) string {
	for _, check := range plan.Checks {
		if check.Kind == networkdoctor.CheckVerification && check.Subject == subject {
			return check.SourceName
		}
	}
	return ""
}
