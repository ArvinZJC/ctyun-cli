/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// ReviewReport summarizes draft metadata readiness for promotion.
type ReviewReport struct {
	Product  string   `json:"product"`
	Quality  string   `json:"quality"`
	Ready    bool     `json:"ready"`
	Findings []string `json:"findings"`
	Notes    []string `json:"notes,omitempty"`
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
	draftI18N := make(map[string]map[string]string, 3)
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		entries, err := readDraftJSON[map[string]string](filepath.Join(draftDir, "i18n", language+".json"))
		if err != nil {
			return ReviewReport{}, err
		}
		draftI18N[language] = entries
	}
	catalogs, err := workspace.readTrackedCatalogs()
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
	if !apiScopeEqual(manifest.API.Scope, source.Product.APIScope) {
		addReviewFinding(&report, "plugin API scope does not match source catalog")
	}
	commandsByOperation := make(map[string]plugin.Command, len(commands.Commands))
	for _, command := range commands.Commands {
		commandsByOperation[command.Operation] = command
	}
	for _, operation := range source.Operations {
		command, ok := commandsByOperation[operation.ID]
		workspace.reviewOperationRecommendation(&report, catalogs, source, operation, command, draftI18N)
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
		for _, language := range []string{"en-US", "en-GB"} {
			if finding := descriptionQualityFinding(source, operation, command, operation.Description[language]); finding != "" {
				addReviewFinding(&report, fmt.Sprintf("operation %s description %s %s", operation.ID, language, finding))
			}
		}
		if len(command.Examples) == 0 && (commandNeedsExample(command) || operationHasSourceCommandExample(operation)) {
			addReviewFinding(&report, fmt.Sprintf("command %s has no example", command.ID))
		}
		for _, example := range command.Examples {
			if err := plugin.ValidateCommandExample(command, example); err != nil {
				addReviewFinding(&report, fmt.Sprintf("command %s example is invalid: %v", command.ID, err))
			}
		}
		for _, name := range operationMissingExampleEvidence(operation, command) {
			addReviewFinding(&report, fmt.Sprintf("operation %s required input %s lacks example evidence", operation.ID, name))
		}
		for _, parameter := range operation.Parameters {
			if parameter.Argument != "" && !commandPathHasArgument(command.Path, parameter.Argument) {
				addReviewFinding(&report, fmt.Sprintf("operation %s argument %s is not exposed by command %s", operation.ID, parameter.Argument, command.ID))
			}
		}
	}
	graphReview := analyzeRecommendationGraph(catalogs)
	for _, ambiguity := range graphReview.Ambiguities {
		addReviewFinding(&report, fmt.Sprintf("recommendation source %s has ambiguous targets: %s", ambiguity.Source, strings.Join(ambiguity.Targets, ", ")))
	}
	for _, cycle := range graphReview.Cycles {
		addReviewFinding(&report, fmt.Sprintf("recommendation cycle detected: %s", strings.Join(cycle, " -> ")))
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
	body := fmt.Sprintf("# OpenAPI Review: %s\n\nStatus: %s\n\n## Findings\n", report.Product, status)
	if len(report.Findings) == 0 {
		body += "\nNone.\n"
	} else {
		for _, finding := range report.Findings {
			body += "\n- " + finding
		}
		body += "\n"
	}
	body += "\n## Notes\n"
	if len(report.Notes) == 0 {
		body += "\nNone.\n"
	} else {
		for _, note := range report.Notes {
			body += "\n- " + note
		}
		body += "\n"
	}
	return body
}

// addReviewFinding marks the report blocked and appends one review finding.
func addReviewFinding(report *ReviewReport, finding string) {
	report.Ready = false
	report.Findings = append(report.Findings, finding)
}

// addReviewNote appends non-blocking context without changing review readiness.
func addReviewNote(report *ReviewReport, note string) {
	report.Notes = append(report.Notes, note)
}

// commandPathHasArgument reports whether a command path exposes an argument.
func commandPathHasArgument(path []string, argument string) bool {
	want := "{" + argument + "}"
	for _, segment := range path {
		if segment == want {
			return true
		}
	}
	return false
}

// apiScopeEqual reports whether two plugin API scopes declare the same
// operation selection boundary.
func apiScopeEqual(left, right plugin.APIScope) bool {
	if left.Notes != right.Notes {
		return false
	}
	if len(left.IncludeURIPrefixes) != len(right.IncludeURIPrefixes) || len(left.ExcludeURIPrefixes) != len(right.ExcludeURIPrefixes) {
		return false
	}
	for i, prefix := range left.IncludeURIPrefixes {
		if prefix != right.IncludeURIPrefixes[i] {
			return false
		}
	}
	for i, prefix := range left.ExcludeURIPrefixes {
		if prefix != right.ExcludeURIPrefixes[i] {
			return false
		}
	}
	return true
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
