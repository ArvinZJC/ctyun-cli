/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

func TestReportOperationResultsWritesOneSummaryAndFailureDetails(t *testing.T) {
	results := []operationResult{
		{Target: "ecs", Outcome: operationChanged},
		{Target: "region", Outcome: operationSkipped},
		{Target: "vpc", Outcome: operationFailed, Err: diagnostic.New("error.plugin_not_found_registry", "vpc")},
	}
	var stdout, stderr bytes.Buffer
	err := reportOperationResults(&stdout, &stderr, results, "en-US", pluginInstallSummary)
	if err == nil {
		t.Fatal("reportOperationResults returned nil aggregate error")
	}
	if got := strings.Count(strings.TrimSpace(stdout.String()), "\n"); got != 0 {
		t.Fatalf("stdout contains more than one summary line: %q", stdout.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin install complete: installed 1; already installed 1; failed 1." {
		t.Fatalf("stdout = %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "vpc: plugin vpc not found in registry.") {
		t.Fatalf("stderr = %q, want target failure", got)
	}
	requireDiagnosticKey(t, err, "error.operation_batch_failed")
}

func TestOperationSummariesUseLanguage(t *testing.T) {
	counts := operationCounts{Changed: 2, Skipped: 3, Failed: 1}
	if got := pluginInstallSummary("en-GB", nil, counts); got != "Plugin install complete: installed 2; already installed 3; failed 1." {
		t.Fatalf("en-GB summary = %q", got)
	}
	if got := pluginInstallSummary("zh-CN", nil, counts); got != "插件安装完成：已安装 2 个；已存在 3 个；失败 1 个。" {
		t.Fatalf("zh-CN summary = %q", got)
	}
}

func TestReportOperationResultsReportsEmptyBatch(t *testing.T) {
	var stdout bytes.Buffer
	if err := reportOperationResults(&stdout, &bytes.Buffer{}, nil, "en-US", pluginReinstallSummary); err != nil {
		t.Fatalf("empty report returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "Plugin reinstall complete: reinstalled 0; failed 0." {
		t.Fatalf("empty summary = %q", got)
	}
}

func TestReportOperationResultsReturnsFailureDetailWriteError(t *testing.T) {
	results := []operationResult{{Target: "ecs", Outcome: operationFailed, Err: errors.New("broken")}}
	if err := reportOperationResults(io.Discard, failingWriter{}, results, "en-US", pluginInstallSummary); err == nil {
		t.Fatal("failure detail writer error was ignored")
	}
}

func TestExecuteAndReportOperationTasksReturnsDisplayError(t *testing.T) {
	originalFactory := operationProgressFactory
	t.Cleanup(func() { operationProgressFactory = originalFactory })
	operationProgressFactory = func(io.Writer) operationDisplay {
		return &recordingOperationDisplay{err: errors.New("display failed")}
	}
	if err := executeAndReportOperationTasks(io.Discard, io.Discard, nil, "en-US", pluginInstallSummary); err == nil {
		t.Fatal("display error was ignored")
	}
}
