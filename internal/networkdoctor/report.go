/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import "time"

// visibleResults projects the internal dependency graph into user-level source
// mirror and CTyun endpoint outcomes.
func visibleResults(plan Plan, internalResults, evaluatedResults []Result) []Result {
	rawByID := resultsByID(internalResults)
	evaluatedByID := resultsByID(evaluatedResults)
	checksByID := make(map[string]Check, len(plan.Checks))
	for _, check := range plan.Checks {
		checksByID[check.ID] = check
	}
	results := make([]Result, 0)
	for _, check := range plan.Checks {
		if check.Kind != CheckVerification || check.SourceName == "" {
			continue
		}
		results = append(results, visibleSourceResult(check, rawByID, evaluatedByID, checksByID))
	}
	for _, check := range plan.Checks {
		if check.Kind != CheckHTTPS || check.Subject != "ctyun" {
			continue
		}
		results = append(results, visibleEndpointResult(check, rawByID, checksByID))
	}
	return results
}

// visibleSourceResult collapses one signed-index dependency chain into one row.
func visibleSourceResult(check Check, rawByID, evaluatedByID map[string]Result, checksByID map[string]Check) Result {
	terminal := rawByID[check.ID]
	visibleCheck := Check{
		ID:         "visible-" + check.ID,
		Capability: check.Capability,
		Subject:    check.Subject,
		SourceName: check.SourceName,
		Role:       check.Role,
		Sequence:   check.Sequence,
		Kind:       CheckSource,
		Target:     check.Target,
	}
	result := Result{Check: visibleCheck, Status: StatusPassed, Duration: dependencyDuration(check.ID, rawByID, checksByID), DetailKey: "doctor.detail.source_passed"}
	if terminal.Status == StatusPassed {
		return result
	}
	result.Status = StatusFailed
	capabilityID := "capability-" + stableID(check.Capability)
	if evaluatedByID[capabilityID].Status != StatusFailed {
		result.Status = StatusWarning
	}
	return applyRootFailure(result, check.ID, rawByID, checksByID)
}

// visibleEndpointResult collapses one CTyun route and HTTPS chain into one row.
func visibleEndpointResult(check Check, rawByID map[string]Result, checksByID map[string]Check) Result {
	terminal := rawByID[check.ID]
	result := Result{
		Check: Check{
			ID:       "visible-" + check.ID,
			Subject:  check.Subject,
			Sequence: check.Sequence,
			Kind:     CheckEndpoint,
			Target:   check.Target,
		},
		Status:    StatusPassed,
		Duration:  dependencyDuration(check.ID, rawByID, checksByID),
		DetailKey: "doctor.detail.endpoint_passed",
	}
	if terminal.Status == StatusPassed {
		return result
	}
	result.Status = StatusFailed
	return applyRootFailure(result, check.ID, rawByID, checksByID)
}

// applyRootFailure copies the actionable failed prerequisite onto a visible row.
func applyRootFailure(result Result, checkID string, rawByID map[string]Result, checksByID map[string]Check) Result {
	failure := rootFailure(checkID, rawByID, checksByID)
	result.Category = failure.Category
	result.DetailKey = failure.DetailKey
	result.DetailArgs = failure.DetailArgs
	result.Err = failure.Err
	return result
}

// rootFailure finds the first deterministic failed prerequisite in one chain.
func rootFailure(checkID string, rawByID map[string]Result, checksByID map[string]Check) Result {
	result := rawByID[checkID]
	if result.Status == StatusFailed {
		return result
	}
	check := checksByID[checkID]
	for _, dependency := range check.DependsOn {
		failure := rootFailure(dependency, rawByID, checksByID)
		if failure.Status == StatusFailed {
			return failure
		}
	}
	return result
}

// dependencyDuration returns the longest completed dependency path duration.
func dependencyDuration(checkID string, rawByID map[string]Result, checksByID map[string]Check) time.Duration {
	check := checksByID[checkID]
	longest := rawByID[checkID].Duration
	for _, dependency := range check.DependsOn {
		duration := rawByID[checkID].Duration + dependencyDuration(dependency, rawByID, checksByID)
		if duration > longest {
			longest = duration
		}
	}
	return longest
}

// resultsByID indexes result values by their internal check ID.
func resultsByID(results []Result) map[string]Result {
	indexed := make(map[string]Result, len(results))
	for _, result := range results {
		indexed[result.Check.ID] = result
	}
	return indexed
}
