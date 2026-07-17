/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package doctor defines status and aggregate models shared by diagnostics.
package doctor

// Status classifies the user-visible outcome of one diagnostic check.
type Status uint8

const (
	// StatusPassed indicates that a check proved its intended condition.
	StatusPassed Status = iota
	// StatusWarning indicates a non-fatal condition that deserves attention.
	StatusWarning
	// StatusFailed indicates that a required condition is not satisfied.
	StatusFailed
	// StatusSkipped indicates that a check was not applicable or could not run.
	StatusSkipped
)

// Counts contains aggregate diagnostic status counts.
type Counts struct {
	Passed  int `json:"passed"`
	Warning int `json:"warning"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// String returns the stable machine-oriented token for a status.
func (status Status) String() string {
	switch status {
	case StatusPassed:
		return "passed"
	case StatusWarning:
		return "warning"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// Count aggregates diagnostic statuses.
func Count(statuses []Status) Counts {
	var counts Counts
	for _, status := range statuses {
		switch status {
		case StatusPassed:
			counts.Passed++
		case StatusWarning:
			counts.Warning++
		case StatusFailed:
			counts.Failed++
		case StatusSkipped:
			counts.Skipped++
		}
	}
	return counts
}

// HasFailures reports whether at least one diagnostic check failed.
func (counts Counts) HasFailures() bool {
	return counts.Failed > 0
}
