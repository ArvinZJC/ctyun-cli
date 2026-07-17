/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// operationSummaryFunc formats one localized final summary for a batch.
type operationSummaryFunc func(language string, results []operationResult, counts operationCounts) string

// executeAndReportOperationTasks runs tasks with the configured progress
// display and reports their structured results.
func executeAndReportOperationTasks(stdout, stderr io.Writer, tasks []operationTask, language string, summary operationSummaryFunc) error {
	results, err := runOperationTasks(operationProgressFactory(stderr), tasks)
	if err != nil {
		return err
	}
	return reportOperationResults(stdout, stderr, results, language, summary)
}

// reportOperationResults writes target failures and exactly one final summary,
// then returns an aggregate diagnostic when any target failed.
func reportOperationResults(stdout, stderr io.Writer, results []operationResult, language string, summary operationSummaryFunc) error {
	counts := countOperationResults(results)
	for _, result := range results {
		if result.Outcome != operationFailed {
			continue
		}
		if err := writeLine(stderr, messagef("operation.target_failed", language, result.Target, localizedError(result.Err, language))); err != nil {
			return err
		}
	}
	if err := writeLine(stdout, summary(language, results, counts)); err != nil {
		return err
	}
	if counts.Failed > 0 {
		return diagnostic.New("error.operation_batch_failed", counts.Failed, len(results))
	}
	return nil
}

// pluginInstallSummary returns the localized plugin installation summary.
func pluginInstallSummary(language string, _ []operationResult, counts operationCounts) string {
	return messagef("plugin.install.summary", language, counts.Changed, counts.Skipped, counts.Failed)
}

// pluginReinstallSummary returns the localized plugin reinstallation summary.
func pluginReinstallSummary(language string, _ []operationResult, counts operationCounts) string {
	return messagef("plugin.reinstall.summary", language, counts.Changed, counts.Failed)
}

// pluginUpdateSummary returns the localized plugin update summary.
func pluginUpdateSummary(language string, _ []operationResult, counts operationCounts) string {
	return messagef("plugin.update.summary", language, counts.Changed, counts.Unchanged, counts.Failed)
}

// pluginRemoveSummary returns the localized plugin removal summary.
func pluginRemoveSummary(language string, _ []operationResult, counts operationCounts) string {
	return messagef("plugin.remove.summary", language, counts.Changed, counts.Failed)
}

// coreUpgradeSummary returns the existing version transition on success and a
// localized failure summary otherwise.
func coreUpgradeSummary(language string, results []operationResult, counts operationCounts) string {
	for _, result := range results {
		if result.Outcome == operationChanged {
			return upgradeInstalledMessage(language, result.Target, result.OldVersion, result.NewVersion)
		}
	}
	return messagef("upgrade.failed_summary", language, counts.Failed)
}

// operationProgressLabel returns a localized active-operation label.
func operationProgressLabel(language, action, target string) string {
	return messagef("operation.progress."+action, language, target)
}
