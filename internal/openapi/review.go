/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapi

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// ReviewReport summarizes draft metadata readiness for promotion.
type ReviewReport struct {
	Product  string   `json:"product"`
	Quality  string   `json:"quality"`
	Ready    bool     `json:"ready"`
	Findings []string `json:"findings"`
}

// ReviewDraft checks generated draft metadata against source evidence.
func (workspace Workspace) ReviewDraft(product string) (ReviewReport, error) {
	source, err := workspace.ReadSource(product)
	if err != nil {
		return ReviewReport{}, err
	}
	draftDir := workspace.ProductPath(product, "draft")
	manifest, err := readDraftJSON[plugin.Manifest](filepath.Join(draftDir, "plugin.json"))
	if err != nil {
		return ReviewReport{}, err
	}
	commands, err := readDraftJSON[plugin.Commands](filepath.Join(draftDir, "commands.json"))
	if err != nil {
		return ReviewReport{}, err
	}
	report := ReviewReport{Product: product, Quality: manifest.Quality, Ready: true}
	if manifest.Quality != "generated" && manifest.Quality != "reviewed" && manifest.Quality != "curated" {
		addReviewFinding(&report, fmt.Sprintf("plugin quality %s is unsupported", manifest.Quality))
	}
	if manifest.Quality == "reviewed" || manifest.Quality == "curated" {
		sourceFingerprint := catalogFingerprint(source)
		if manifest.API.SourceFingerprint == "" {
			addReviewFinding(&report, fmt.Sprintf("%s plugin must include source fingerprint %s", manifest.Quality, sourceFingerprint))
		} else if manifest.API.SourceFingerprint != sourceFingerprint {
			addReviewFinding(&report, fmt.Sprintf("%s plugin source fingerprint %s does not match source catalog %s", manifest.Quality, manifest.API.SourceFingerprint, sourceFingerprint))
		}
	}
	commandsByOperation := make(map[string]plugin.Command, len(commands.Commands))
	for _, command := range commands.Commands {
		commandsByOperation[command.Operation] = command
	}
	for _, operation := range source.Operations {
		command, ok := commandsByOperation[operation.ID]
		if !ok {
			addReviewFinding(&report, fmt.Sprintf("operation %s has no command", operation.ID))
			continue
		}
		if operation.Dangerous && command.Dangerous.Confirm == "" {
			addReviewFinding(&report, fmt.Sprintf("operation %s is dangerous but command %s lacks confirmation", operation.ID, command.ID))
		}
		if operation.Response.RowPath != "" && command.Table == "" {
			addReviewFinding(&report, fmt.Sprintf("operation %s has response rows but no table", operation.ID))
		}
	}
	if err := writeText(workspace.ProductPath(product, "review.md"), report.Markdown()); err != nil {
		return ReviewReport{}, err
	}
	return report, nil
}

// Markdown renders review status for human reviewers.
func (report ReviewReport) Markdown() string {
	status := "ready"
	if !report.Ready {
		status = "blocked"
	}
	body := fmt.Sprintf("# OpenAPI Review: %s\n\nStatus: %s\n", report.Product, status)
	for _, finding := range report.Findings {
		body += "\n- " + finding
	}
	if len(report.Findings) > 0 {
		body += "\n"
	}
	return body
}

// addReviewFinding marks the report blocked and appends one review finding.
func addReviewFinding(report *ReviewReport, finding string) {
	report.Ready = false
	report.Findings = append(report.Findings, finding)
}

// readDraftJSON reads generated draft JSON into a typed metadata value.
func readDraftJSON[T any](path string) (T, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	if err := decodeJSON(data, &value); err != nil {
		return value, err
	}
	return value, nil
}
