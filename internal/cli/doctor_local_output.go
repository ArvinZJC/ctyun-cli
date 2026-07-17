/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
	"github.com/ArvinZJC/ctyun-cli/internal/localdoctor"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
)

// doctorLocalJSONResult is one stable, localized local diagnostic row.
type doctorLocalJSONResult struct {
	Key      string `json:"key"`
	Check    string `json:"check"`
	Target   string `json:"target"`
	Status   string `json:"status"`
	Category string `json:"category,omitempty"`
	Detail   string `json:"detail"`
}

// doctorLocalJSONReport contains displayed results and complete summary counts.
type doctorLocalJSONReport struct {
	Results []doctorLocalJSONResult `json:"results"`
	Summary doctor.Counts           `json:"summary"`
}

// renderDoctorLocalReport renders a deterministic localized local diagnostic report.
func renderDoctorLocalReport(stdout io.Writer, report localdoctor.Report, opts globalOptions) error {
	columns := doctorLocalColumns(opts.Language)
	rows := make([]map[string]string, 0, len(report.Findings))
	jsonResults := make([]doctorLocalJSONResult, 0, len(report.Findings))
	for index, finding := range report.Findings {
		check := messageText("doctor.local.check."+string(finding.Key), opts.Language)
		status := doctorStatusText(finding.Status, opts.Language)
		detail := doctorLocalDetail(finding, opts.Language)
		rows = append(rows, map[string]string{
			"check": check, "target": abbreviateHomePath(finding.Target),
			"status": status, "detail": detail, "_index": strconv.Itoa(index),
		})
		jsonResults = append(jsonResults, doctorLocalJSONResult{
			Key: string(finding.Key), Check: check, Target: finding.Target,
			Status: finding.Status.String(), Category: string(finding.Category), Detail: detail,
		})
	}
	rows, filteredResults, err := selectReportRows(rows, jsonResults, columns, opts)
	if err != nil {
		return err
	}
	jsonReport := doctorLocalJSONReport{Results: filteredResults, Summary: report.Counts}
	return renderReportOutput(stdout, rows, columns, jsonReport, doctorLocalSummary(opts.Language, report.Counts), opts)
}

// doctorLocalDetail localizes human-facing detail arguments while preserving
// stable machine tokens in the finding model.
func doctorLocalDetail(finding localdoctor.Finding, language string) string {
	args := finding.DetailArgs
	if finding.DetailKey == "doctor.local.detail.credentials_stored" || finding.DetailKey == "doctor.local.detail.credentials_mixed" {
		args = make([]any, len(finding.DetailArgs))
		for index, argument := range finding.DetailArgs {
			source, _ := argument.(string)
			args[index] = messageText("config.explain.source."+source, language)
		}
	}
	return messagef(finding.DetailKey, language, args...)
}

// doctorLocalColumns returns stable local report keys with localized labels.
func doctorLocalColumns(language string) []output.Column {
	return []output.Column{
		{Key: "check", Label: messageText("doctor.column.check", language)},
		{Key: "target", Label: messageText("doctor.column.target", language)},
		{Key: "status", Label: messageText("doctor.column.status", language)},
		{Key: "detail", Label: messageText("doctor.column.detail", language)},
	}
}

// doctorLocalSummary returns the one-line complete local diagnostic summary.
func doctorLocalSummary(language string, counts doctor.Counts) string {
	return messagef("doctor.local.summary", language, counts.Passed, counts.Warning, counts.Failed, counts.Skipped)
}

// abbreviateHomePath shortens human table paths below the current home directory.
func abbreviateHomePath(path string) string {
	if path == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(home)
	if cleanPath == cleanHome {
		return "~"
	}
	prefix := cleanHome + string(filepath.Separator)
	if relativePath, ok := strings.CutPrefix(cleanPath, prefix); ok {
		return filepath.Join("~", relativePath)
	}
	return path
}
