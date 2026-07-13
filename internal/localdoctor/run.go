/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package localdoctor

import "github.com/ArvinZJC/ctyun-cli/internal/doctor"

// Run executes deterministic offline local config checks.
func Run(input Input, dependencies Dependencies) Report {
	defaults := DefaultDependencies()
	if dependencies.Stat == nil {
		dependencies.Stat = defaults.Stat
	}
	if dependencies.ReadFile == nil {
		dependencies.ReadFile = defaults.ReadFile
	}
	if dependencies.GOOS == "" {
		dependencies.GOOS = defaults.GOOS
	}
	if dependencies.ReadDir == nil {
		dependencies.ReadDir = defaults.ReadDir
	}
	if dependencies.LoadBundle == nil {
		dependencies.LoadBundle = defaults.LoadBundle
	}
	evaluation := evaluateConfigFile(input, dependencies)
	findings := configFindings(input, evaluation)
	if input.PluginRoot != "" {
		findings = append(findings, evaluatePlugins(input, dependencies)...)
	}
	language, _ := evaluation.inspection.Resolution.Setting("language")
	return Report{Findings: findings, Counts: doctor.Count(findingStatusValues(findings)), Language: language.Value}
}

// findingStatusValues projects findings onto the shared count model.
func findingStatusValues(findings []Finding) []doctor.Status {
	statuses := make([]doctor.Status, 0, len(findings))
	for _, finding := range findings {
		statuses = append(statuses, finding.Status)
	}
	return statuses
}
