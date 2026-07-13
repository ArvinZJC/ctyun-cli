/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package networkdoctor plans and runs safe network capability diagnostics.
package networkdoctor

import (
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
)

// Status classifies the user-visible outcome of one diagnostic check.
type Status = doctor.Status

const (
	// StatusPassed indicates that a check proved its intended capability.
	StatusPassed = doctor.StatusPassed
	// StatusWarning indicates a failed alternative whose capability remains usable.
	StatusWarning = doctor.StatusWarning
	// StatusFailed indicates that a required capability is unavailable.
	StatusFailed = doctor.StatusFailed
	// StatusSkipped indicates that a prerequisite prevented the check from running.
	StatusSkipped = doctor.StatusSkipped
)

// FailureCategory identifies the network layer responsible for a failed check.
type FailureCategory string

const (
	// FailureConfiguration identifies invalid or missing local diagnostic inputs.
	FailureConfiguration FailureCategory = "configuration"
	// FailureDNS identifies name-resolution failures.
	FailureDNS FailureCategory = "dns"
	// FailureProxy identifies proxy selection or proxy-route failures.
	FailureProxy FailureCategory = "proxy"
	// FailureConnection identifies transport connection failures.
	FailureConnection FailureCategory = "connection"
	// FailureTimeout identifies expired check deadlines.
	FailureTimeout FailureCategory = "timeout"
	// FailureTLS identifies TLS negotiation or certificate failures.
	FailureTLS FailureCategory = "tls"
	// FailureRedirect identifies unsafe redirect behavior.
	FailureRedirect FailureCategory = "redirect"
	// FailureIndex identifies index retrieval failures.
	FailureIndex FailureCategory = "index"
	// FailureSignature identifies signature retrieval or verification failures.
	FailureSignature FailureCategory = "signature"
	// FailurePublicKey identifies missing or invalid trusted keys.
	FailurePublicKey FailureCategory = "public_key"
	// FailureCancelled identifies caller cancellation.
	FailureCancelled FailureCategory = "cancelled"
)

// CheckKind identifies the work performed by one planned check.
type CheckKind string

const (
	// CheckConfiguration validates local inputs.
	CheckConfiguration CheckKind = "configuration"
	// CheckRoute validates a direct or proxy route host.
	CheckRoute CheckKind = "route"
	// CheckHTTPS validates unauthenticated HTTPS reachability.
	CheckHTTPS CheckKind = "https"
	// CheckIndex retrieves a bounded signed-index document.
	CheckIndex CheckKind = "index"
	// CheckSignature retrieves a bounded detached signature.
	CheckSignature CheckKind = "signature"
	// CheckVerification verifies an index and signature with the trusted key.
	CheckVerification CheckKind = "verification"
	// CheckCapability aggregates alternative checks into one required capability.
	CheckCapability CheckKind = "capability"
	// CheckSource is one user-visible signed source mirror result.
	CheckSource CheckKind = "source"
	// CheckEndpoint is one user-visible CTyun API endpoint result.
	CheckEndpoint CheckKind = "endpoint"
)

// Check describes one deterministic unit in a diagnostic plan.
type Check struct {
	ID          string
	Capability  string
	Subject     string
	SourceName  string
	Role        string
	Sequence    int
	Kind        CheckKind
	Target      string
	RequestURL  string
	DependsOn   []string
	Alternative bool
}

// Capability describes alternative terminal checks that can satisfy one requirement.
type Capability struct {
	ID           string
	Alternatives []string
	Required     bool
}

// Plan contains ordered checks and their required capability relationships.
type Plan struct {
	Checks       []Check
	Capabilities []Capability
	PublicKey    string
}

// Result records one completed, failed, or skipped check.
type Result struct {
	Check      Check
	Status     Status
	Duration   time.Duration
	Category   FailureCategory
	DetailKey  string
	DetailArgs []any
	Err        error
}

// Counts contains aggregate diagnostic status counts.
type Counts = doctor.Counts

// Report contains deterministic results and capability-level failure state.
type Report struct {
	Results            []Result
	Counts             Counts
	FailedCapabilities []string
}

// countResults aggregates diagnostic result statuses.
func countResults(results []Result) Counts {
	statuses := make([]doctor.Status, 0, len(results))
	for _, result := range results {
		statuses = append(statuses, result.Status)
	}
	return doctor.Count(statuses)
}
