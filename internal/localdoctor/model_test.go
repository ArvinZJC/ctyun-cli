/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package localdoctor

import (
	"slices"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
)

func TestRunOrdersConfigFindingsAndCountsStatuses(t *testing.T) {
	report := Run(Input{ConfigPath: "/missing", Getenv: func(string) string { return "" }}, DefaultDependencies())
	got := findingKeys(report.Findings)
	want := []CheckKey{CheckConfigFile, CheckProfile, CheckCredentials, CheckRegion, CheckEndpoint}
	if !slices.Equal(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	if report.Counts != doctor.Count(findingStatuses(report.Findings)) {
		t.Fatalf("counts = %#v", report.Counts)
	}
}

// findingKeys projects findings onto their stable keys for ordering assertions.
func findingKeys(findings []Finding) []CheckKey {
	keys := make([]CheckKey, 0, len(findings))
	for _, finding := range findings {
		keys = append(keys, finding.Key)
	}
	return keys
}

// findingStatuses projects findings onto shared statuses for count assertions.
func findingStatuses(findings []Finding) []doctor.Status {
	statuses := make([]doctor.Status, 0, len(findings))
	for _, finding := range findings {
		statuses = append(statuses, finding.Status)
	}
	return statuses
}
