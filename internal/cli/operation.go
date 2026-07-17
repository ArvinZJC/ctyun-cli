/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

// operationOutcome classifies the result of one named operation target.
type operationOutcome uint8

const (
	// operationChanged indicates that the target was mutated successfully.
	operationChanged operationOutcome = iota
	// operationSkipped indicates that policy intentionally skipped the target.
	operationSkipped
	// operationUnchanged indicates that the target required no mutation.
	operationUnchanged
	// operationFailed indicates that the target operation failed.
	operationFailed
)

// operationTask describes one named unit of work and its progress label.
type operationTask struct {
	Target string
	Label  string
	Run    func() operationResult
}

// operationResult records the outcome and optional version transition for one
// target.
type operationResult struct {
	Target     string
	Outcome    operationOutcome
	OldVersion string
	NewVersion string
	Err        error
}

// operationCounts contains aggregate outcome counts for an operation batch.
type operationCounts struct {
	Changed   int
	Skipped   int
	Unchanged int
	Failed    int
}

// operationDisplay receives lifecycle updates while a batch is running.
type operationDisplay interface {
	Start(total int) error
	Update(completed int, label string) error
	Clear() error
}

// runOperationTasks executes tasks sequentially while continuing after
// independent target failures.
func runOperationTasks(display operationDisplay, tasks []operationTask) ([]operationResult, error) {
	if err := display.Start(len(tasks)); err != nil {
		return nil, err
	}
	results := make([]operationResult, 0, len(tasks))
	for index, task := range tasks {
		if err := display.Update(index, task.Label); err != nil {
			_ = display.Clear()
			return results, err
		}
		result := task.Run()
		if result.Target == "" {
			result.Target = task.Target
		}
		if result.Err != nil {
			result.Outcome = operationFailed
		}
		results = append(results, result)
		if err := display.Update(index+1, task.Label); err != nil {
			_ = display.Clear()
			return results, err
		}
	}
	if err := display.Clear(); err != nil {
		return results, err
	}
	return results, nil
}

// countOperationResults aggregates structured target outcomes.
func countOperationResults(results []operationResult) operationCounts {
	var counts operationCounts
	for _, result := range results {
		switch result.Outcome {
		case operationChanged:
			counts.Changed++
		case operationSkipped:
			counts.Skipped++
		case operationUnchanged:
			counts.Unchanged++
		case operationFailed:
			counts.Failed++
		}
	}
	return counts
}
