/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/networkdoctor"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
)

// doctorNetworkJSONResult is one localized structured diagnostic row.
type doctorNetworkJSONResult struct {
	Check    string `json:"check"`
	Target   string `json:"target"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Category string `json:"category,omitempty"`
	Detail   string `json:"detail"`
}

// doctorNetworkJSONReport contains displayed results and complete summary counts.
type doctorNetworkJSONReport struct {
	Results []doctorNetworkJSONResult `json:"results"`
	Summary networkdoctor.Counts      `json:"summary"`
}

// renderDoctorNetworkReport renders a deterministic localized diagnostic report.
func renderDoctorNetworkReport(stdout io.Writer, report networkdoctor.Report, opts globalOptions) error {
	columns := doctorNetworkColumns(opts.Language)
	rows := make([]map[string]string, 0, len(report.Results))
	jsonResults := make([]doctorNetworkJSONResult, 0, len(report.Results))
	for index, result := range report.Results {
		check := doctorCheckText(result.Check, opts.Language)
		status := doctorStatusText(result.Status, opts.Language)
		duration := formatDoctorDuration(result.Duration)
		detail := doctorDetailText(result, opts.Language)
		rows = append(rows, map[string]string{"check": check, "target": result.Check.Target, "status": status, "duration": duration, "detail": detail, "_index": strconv.Itoa(index)})
		jsonResults = append(jsonResults, doctorNetworkJSONResult{Check: check, Target: result.Check.Target, Status: status, Duration: duration, Category: string(result.Category), Detail: detail})
	}
	filter, err := output.ResolveFilterExpression(columns, opts.Filter)
	if err != nil {
		return err
	}
	sortExpression, err := output.ResolveSortExpression(columns, opts.Sort)
	if err != nil {
		return err
	}
	rows, _ = output.FilterRows(rows, filter)
	rows, _ = output.SortRows(rows, sortExpression)
	switch opts.Output {
	case "json":
		filteredResults := make([]doctorNetworkJSONResult, 0, len(rows))
		for _, row := range rows {
			index, _ := strconv.Atoi(row["_index"])
			filteredResults = append(filteredResults, jsonResults[index])
		}
		rendered, err := renderOutputJSON(doctorNetworkJSONReport{Results: filteredResults, Summary: report.Counts})
		if err != nil {
			return err
		}
		return writeString(stdout, rendered)
	case "table":
		rendered, err := renderTableOutput(stdout, rows, columns, output.TableOptions{Columns: opts.Columns, NoHeader: opts.NoHeader, Style: opts.Table})
		if err != nil {
			return err
		}
		if err := writeString(stdout, rendered); err != nil {
			return err
		}
		return writeLine(stdout, doctorNetworkSummary(opts.Language, report.Counts))
	default:
		return diagnostic.New("error.unsupported_output", opts.Output)
	}
}

// doctorNetworkColumns returns stable report keys with localized labels.
func doctorNetworkColumns(language string) []output.Column {
	return []output.Column{
		{Key: "check", Label: messageText("doctor.column.check", language)},
		{Key: "target", Label: messageText("doctor.column.target", language)},
		{Key: "status", Label: messageText("doctor.column.status", language)},
		{Key: "duration", Label: messageText("doctor.column.duration", language)},
		{Key: "detail", Label: messageText("doctor.column.detail", language)},
	}
}

// doctorNetworkSummary returns the one-line complete diagnostic summary.
func doctorNetworkSummary(language string, counts networkdoctor.Counts) string {
	return messagef("doctor.network.summary", language, counts.Passed, counts.Warning, counts.Failed, counts.Skipped)
}

// doctorCheckText renders a visible check subject without exposing internal IDs.
func doctorCheckText(check networkdoctor.Check, language string) string {
	switch check.Kind {
	case networkdoctor.CheckSource:
		if check.Subject == "core" {
			return messageText("doctor.check.core_source", language)
		}
		if check.Subject == "plugin" {
			return messageText("doctor.check.plugin_source", language)
		}
	case networkdoctor.CheckEndpoint:
		return messageText("doctor.check.endpoint", language)
	}
	subject := doctorSubjectText(check.Subject, language)
	if check.SourceName != "" {
		subject = sourceDisplayName(check.SourceName) + " " + subject
	}
	return messagef("doctor.check."+string(check.Kind), language, subject)
}

// doctorProgressText hides internal dependency and capability terminology from
// transient progress while retaining useful stage information.
func doctorProgressText(check networkdoctor.Check, language string) string {
	switch check.Kind {
	case networkdoctor.CheckConfiguration:
		return messageText("doctor.network.progress.prepare", language)
	case networkdoctor.CheckCapability:
		return messageText("doctor.network.progress.finalise", language)
	default:
		return messagef("doctor.network.progress", language, doctorCheckText(check, language))
	}
}

// doctorSubjectText localizes known core diagnostic subjects.
func doctorSubjectText(subject, language string) string {
	switch subject {
	case "core", "core-source":
		return messageText("doctor.subject.core", language)
	case "plugin", "plugin-source":
		return messageText("doctor.subject.plugin", language)
	case "ctyun":
		return messageText("doctor.subject.ctyun", language)
	default:
		return subject
	}
}

// sourceDisplayName applies canonical capitalization to hosted source names.
func sourceDisplayName(source string) string {
	switch source {
	case "github":
		return "GitHub"
	case "gitee":
		return "Gitee"
	default:
		return source
	}
}

// doctorStatusText localizes one compact diagnostic status token.
func doctorStatusText(status networkdoctor.Status, language string) string {
	key := "doctor.status.skipped"
	switch status {
	case networkdoctor.StatusPassed:
		key = "doctor.status.passed"
	case networkdoctor.StatusWarning:
		key = "doctor.status.warning"
	case networkdoctor.StatusFailed:
		key = "doctor.status.failed"
	case networkdoctor.StatusSkipped:
		key = "doctor.status.skipped"
	}
	return messageText(key, language)
}

// doctorDetailText formats a structured result detail through runtime catalogs.
func doctorDetailText(result networkdoctor.Result, language string) string {
	key := result.DetailKey
	if key == "" {
		key = "doctor.detail.passed"
	}
	return messagef(key, language, result.DetailArgs...)
}

// formatDoctorDuration returns a compact machine-oriented elapsed-time token.
func formatDoctorDuration(duration time.Duration) string {
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", duration.Seconds())
}
