/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"errors"
	"slices"
	"testing"
)

type recordingOperationDisplay struct {
	updates []string
	cleared bool
	err     error
}

func (display *recordingOperationDisplay) Start(int) error {
	return display.err
}

func (display *recordingOperationDisplay) Update(_ int, label string) error {
	display.updates = append(display.updates, label)
	return nil
}

func (display *recordingOperationDisplay) Clear() error {
	display.cleared = true
	return nil
}

func TestRunOperationTasksPreservesOrderAndContinuesAfterFailure(t *testing.T) {
	var ran []string
	display := &recordingOperationDisplay{}
	tasks := []operationTask{
		{Target: "ecs", Label: "Installing ecs", Run: func() operationResult {
			ran = append(ran, "ecs")
			return operationResult{Outcome: operationChanged}
		}},
		{Target: "region", Label: "Installing region", Run: func() operationResult {
			ran = append(ran, "region")
			return operationResult{Target: "region", Err: errors.New("broken")}
		}},
		{Target: "vpc", Label: "Installing vpc", Run: func() operationResult {
			ran = append(ran, "vpc")
			return operationResult{Target: "vpc", Outcome: operationSkipped}
		}},
	}

	results, err := runOperationTasks(display, tasks)
	if err != nil {
		t.Fatalf("runOperationTasks returned display error: %v", err)
	}
	if !slices.Equal(ran, []string{"ecs", "region", "vpc"}) {
		t.Fatalf("run order = %v", ran)
	}
	if got, want := countOperationResults(results), (operationCounts{Changed: 1, Skipped: 1, Failed: 1}); got != want {
		t.Fatalf("counts = %+v, want %+v", got, want)
	}
	if !display.cleared {
		t.Fatal("operation display was not cleared")
	}
	if results[0].Target != "ecs" {
		t.Fatalf("defaulted result target = %q", results[0].Target)
	}
}

func TestRunOperationTasksReturnsDisplayStartError(t *testing.T) {
	want := errors.New("display failed")
	display := &recordingOperationDisplay{err: want}
	ran := make(chan struct{}, 1)

	results, err := runOperationTasks(display, []operationTask{{
		Target: "ecs",
		Run: func() operationResult {
			ran <- struct{}{}
			return operationResult{Target: "ecs", Outcome: operationChanged}
		},
	}})
	if !errors.Is(err, want) {
		t.Fatalf("runOperationTasks error = %v, want %v", err, want)
	}
	if len(results) != 0 {
		t.Fatalf("start failure returned results %v", results)
	}
	select {
	case <-ran:
		t.Fatal("task ran after the display failed to start")
	default:
	}
}

type failingUpdateOperationDisplay struct {
	failCall int
	calls    int
	clearErr error
	cleared  bool
}

func (*failingUpdateOperationDisplay) Start(int) error { return nil }

func (display *failingUpdateOperationDisplay) Update(int, string) error {
	display.calls++
	if display.calls == display.failCall {
		return errors.New("update failed")
	}
	return nil
}

func (display *failingUpdateOperationDisplay) Clear() error {
	display.cleared = true
	return display.clearErr
}

func TestRunOperationTasksReturnsDisplayUpdateErrors(t *testing.T) {
	for _, failCall := range []int{1, 2} {
		t.Run(string(rune('0'+failCall)), func(t *testing.T) {
			display := &failingUpdateOperationDisplay{failCall: failCall}
			_, err := runOperationTasks(display, []operationTask{{
				Target: "ecs",
				Run: func() operationResult {
					return operationResult{Outcome: operationChanged}
				},
			}})
			if err == nil || err.Error() != "update failed" {
				t.Fatalf("update failure error = %v", err)
			}
			if !display.cleared {
				t.Fatal("update failure did not clear display")
			}
		})
	}
}

func TestRunOperationTasksReturnsDisplayClearError(t *testing.T) {
	display := &failingUpdateOperationDisplay{clearErr: errors.New("clear failed")}
	_, err := runOperationTasks(display, nil)
	if err == nil || err.Error() != "clear failed" {
		t.Fatalf("clear failure error = %v", err)
	}
}
