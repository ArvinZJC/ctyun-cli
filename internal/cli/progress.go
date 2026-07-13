/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const (
	// minimumProgressBarWidth keeps narrow-terminal progress recognizable.
	minimumProgressBarWidth = 3
)

// terminalCapabilities provides injectable terminal detection and sizing.
type terminalCapabilities struct {
	Interactive func(io.Writer) bool
	Width       func(io.Writer) (int, error)
}

// defaultTerminalCapabilities detects terminals backed by operating-system
// file descriptors.
var defaultTerminalCapabilities = terminalCapabilities{
	Interactive: func(writer io.Writer) bool {
		file, ok := writer.(*os.File)
		return ok && term.IsTerminal(int(file.Fd()))
	},
	Width: func(writer io.Writer) (int, error) {
		file, ok := writer.(*os.File)
		if !ok {
			return 0, fmt.Errorf("terminal writer has no file descriptor")
		}
		width, _, err := term.GetSize(int(file.Fd()))
		return width, err
	},
}

// operationProgressFactory creates the display used by command operations and
// provides a deterministic test seam.
var operationProgressFactory = func(writer io.Writer) operationDisplay {
	return newOperationProgress(writer)
}

// operationProgress renders one determinate progress line on an interactive
// terminal.
type operationProgress struct {
	writer       io.Writer
	capabilities terminalCapabilities
	interactive  bool
	width        int
	total        int
	drawn        bool
}

// newOperationProgress creates a display using real terminal capabilities.
func newOperationProgress(writer io.Writer) operationDisplay {
	return newOperationProgressWithCapabilities(writer, defaultTerminalCapabilities)
}

// newOperationProgressWithCapabilities creates a display with injected
// terminal behavior for deterministic tests.
func newOperationProgressWithCapabilities(writer io.Writer, capabilities terminalCapabilities) operationDisplay {
	return &operationProgress{writer: writer, capabilities: capabilities}
}

// Start initializes terminal state for a batch of total work units.
func (progress *operationProgress) Start(total int) error {
	progress.total = total
	progress.interactive = total > 0 && progress.capabilities.Interactive(progress.writer)
	if !progress.interactive {
		return nil
	}
	width, err := progress.capabilities.Width(progress.writer)
	if err != nil || width <= 0 {
		width = defaultTerminalWidth
	}
	progress.width = width
	return nil
}

// Update redraws progress for completed work units and the active label.
func (progress *operationProgress) Update(completed int, label string) error {
	if !progress.interactive || progress.total <= 0 {
		return nil
	}
	completed = max(0, min(completed, progress.total))
	line := progressLine(progress.width, completed, progress.total, label)
	if _, err := fmt.Fprintf(progress.writer, "\r%s", line); err != nil {
		return err
	}
	progress.drawn = true
	return nil
}

// Clear removes a previously drawn interactive progress line.
func (progress *operationProgress) Clear() error {
	if !progress.interactive || !progress.drawn {
		return nil
	}
	_, err := io.WriteString(progress.writer, "\r\x1b[2K")
	progress.drawn = false
	return err
}

// progressLine builds one terminal-width-aware progress line.
func progressLine(width, completed, total int, label string) string {
	counter := fmt.Sprintf(" %d/%d  ", completed, total)
	available := max(0, width-runewidth.StringWidth(counter)-2-minimumProgressBarWidth)
	label = runewidth.Truncate(label, available, "")
	barWidth := max(minimumProgressBarWidth, width-runewidth.StringWidth(counter)-runewidth.StringWidth(label)-2)
	filled := barWidth * completed / total
	var bar string
	if filled >= barWidth {
		bar = strings.Repeat("=", barWidth)
	} else if filled > 0 {
		bar = strings.Repeat("=", filled-1) + ">" + strings.Repeat(".", barWidth-filled)
	} else {
		bar = strings.Repeat(".", barWidth)
	}
	return runewidth.Truncate("["+bar+"]"+counter+label, width, "")
}
