/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import (
	"context"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// Progress describes one serialized diagnostic completion event.
type Progress struct {
	Completed int
	Total     int
	Check     Check
}

// Observer receives serialized progress events from the runner coordinator.
type Observer func(Progress) error

// Run executes a diagnostic plan with concurrent dependency waves.
func Run(ctx context.Context, plan Plan, timeout time.Duration, dependencies Dependencies, observer Observer) (Report, error) {
	dependencies = defaultDependencies(dependencies)
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	completed := make(map[string]Result, len(plan.Checks))
	data := make(map[string][]byte)
	remaining := len(plan.Checks)
	for remaining > 0 {
		ready, skipped := readyChecks(plan.Checks, completed)
		for _, result := range skipped {
			completed[result.Check.ID] = result
			remaining--
			if err := observeProgress(observer, completed, plan.Checks, result.Check); err != nil {
				return buildReport(plan, completed), err
			}
		}
		if len(ready) == 0 && len(skipped) > 0 {
			continue
		}
		if len(ready) == 0 {
			return buildReport(plan, completed), diagnostic.New("error.doctor_dependency_cycle")
		}
		outputs := runWave(ctx, ready, plan, timeout, dependencies, data)
		for _, output := range outputs {
			completed[output.result.Check.ID] = output.result
			if output.data != nil {
				data[output.result.Check.ID] = output.data
			}
			remaining--
			if err := observeProgress(observer, completed, plan.Checks, output.result.Check); err != nil {
				return buildReport(plan, completed), err
			}
		}
	}
	return buildReport(plan, completed), nil
}

// defaultDependencies fills omitted production resolver, transport, proxy, and clock values.
func defaultDependencies(dependencies Dependencies) Dependencies {
	if dependencies.Resolver == nil {
		dependencies.Resolver = net.DefaultResolver
	}
	if dependencies.Transport == nil {
		dependencies.Transport = http.DefaultTransport
	}
	if dependencies.Proxy == nil {
		dependencies.Proxy = http.ProxyFromEnvironment
	}
	if dependencies.Now == nil {
		dependencies.Now = time.Now
	}
	return dependencies
}

// readyChecks selects the next dependency wave and materializes skipped checks.
func readyChecks(checks []Check, completed map[string]Result) ([]Check, []Result) {
	var ready []Check
	var skipped []Result
	for _, check := range checks {
		if _, ok := completed[check.ID]; ok {
			continue
		}
		allComplete := true
		prerequisiteFailed := false
		for _, dependency := range check.DependsOn {
			result, ok := completed[dependency]
			if !ok {
				allComplete = false
				break
			}
			if result.Status == StatusFailed || result.Status == StatusSkipped {
				prerequisiteFailed = true
			}
		}
		if !allComplete {
			continue
		}
		if prerequisiteFailed && check.Kind != CheckCapability {
			skipped = append(skipped, Result{Check: check, Status: StatusSkipped, Category: FailureConfiguration, DetailKey: "doctor.detail.dependency"})
			continue
		}
		ready = append(ready, check)
	}
	return ready, skipped
}

// runWave executes independent ready checks concurrently and returns planned order.
func runWave(ctx context.Context, checks []Check, plan Plan, timeout time.Duration, dependencies Dependencies, data map[string][]byte) []probeOutput {
	results := make(chan probeOutput, len(checks))
	for _, check := range checks {
		check := check
		go func() {
			if check.Kind == CheckCapability {
				results <- probeOutput{result: evaluateCapabilityCheck(check, data, nil)}
				return
			}
			checkContext, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			results <- executeCheck(checkContext, check, plan, dependencies, data)
		}()
	}
	outputs := make([]probeOutput, 0, len(checks))
	for range checks {
		outputs = append(outputs, <-results)
	}
	slices.SortFunc(outputs, func(left, right probeOutput) int {
		return left.result.Check.Sequence - right.result.Check.Sequence
	})
	return outputs
}

// evaluateCapabilityCheck reports full, degraded, or unavailable alternatives.
func evaluateCapabilityCheck(check Check, _ map[string][]byte, completed map[string]Result) Result {
	result := Result{Check: check, Status: StatusPassed, DetailKey: "doctor.detail.passed"}
	if completed == nil {
		return result
	}
	passed := 0
	for _, dependency := range check.DependsOn {
		if completed[dependency].Status == StatusPassed {
			passed++
		}
	}
	if passed == 0 {
		result.Status = StatusFailed
		result.DetailKey = "doctor.detail.capability_failed"
	} else if passed < len(check.DependsOn) {
		result.Status = StatusWarning
		result.DetailKey = "doctor.detail.capability_degraded"
	}
	return result
}

// observeProgress emits one coordinator-owned progress event when configured.
func observeProgress(observer Observer, completed map[string]Result, checks []Check, check Check) error {
	if observer == nil {
		return nil
	}
	return observer(Progress{Completed: len(completed), Total: len(checks), Check: check})
}

// buildReport orders completed results and derives final capability state.
func buildReport(plan Plan, completed map[string]Result) Report {
	internalResults := make([]Result, 0, len(completed))
	for _, check := range plan.Checks {
		if result, ok := completed[check.ID]; ok {
			internalResults = append(internalResults, result)
		}
	}
	evaluatedResults := evaluateCapabilityResults(plan, slices.Clone(internalResults))
	failedCapabilities := make([]string, 0)
	for _, result := range evaluatedResults {
		if result.Check.Kind == CheckCapability && result.Status == StatusFailed {
			failedCapabilities = append(failedCapabilities, result.Check.Capability)
		}
	}
	results := visibleResults(plan, internalResults, evaluatedResults)
	return Report{Results: results, Counts: Count(results), FailedCapabilities: failedCapabilities}
}

// evaluateCapabilityResults computes capability rows and alternative warnings.
func evaluateCapabilityResults(plan Plan, results []Result) []Result {
	byID := make(map[string]Result, len(results))
	for _, result := range results {
		byID[result.Check.ID] = result
	}
	for index := range results {
		if results[index].Check.Kind != CheckCapability {
			continue
		}
		results[index] = evaluateCapabilityCheck(results[index].Check, nil, byID)
		byID[results[index].Check.ID] = results[index]
	}
	for index := range results {
		if results[index].Status != StatusFailed || results[index].Check.Kind == CheckCapability {
			continue
		}
		capabilities := dependentCapabilities(plan, results[index].Check.ID)
		if len(capabilities) == 0 {
			continue
		}
		usable := true
		for _, capabilityID := range capabilities {
			if byID["capability-"+stableID(capabilityID)].Status == StatusFailed {
				usable = false
				break
			}
		}
		if usable {
			results[index].Status = StatusWarning
		}
	}
	return results
}

// dependentCapabilities returns capabilities transitively affected by a check.
func dependentCapabilities(plan Plan, checkID string) []string {
	var capabilities []string
	for _, capability := range plan.Capabilities {
		for _, alternative := range capability.Alternatives {
			if checkDependsOn(plan.Checks, alternative, checkID, make(map[string]bool)) {
				capabilities = append(capabilities, capability.ID)
				break
			}
		}
	}
	return capabilities
}

// checkDependsOn reports whether one check transitively requires another.
func checkDependsOn(checks []Check, checkID, dependencyID string, visited map[string]bool) bool {
	if checkID == dependencyID {
		return true
	}
	if visited[checkID] {
		return false
	}
	visited[checkID] = true
	for _, check := range checks {
		if check.ID != checkID {
			continue
		}
		for _, dependency := range check.DependsOn {
			if checkDependsOn(checks, dependency, dependencyID, visited) {
				return true
			}
		}
	}
	return false
}
