/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"errors"
	"io"
	"testing"
)

func TestTerminalDisplayWidthHandlesNonInteractiveAndUnavailableSizes(t *testing.T) {
	if got := terminalDisplayWidth(io.Discard, terminalCapabilities{}); got != 0 {
		t.Fatalf("non-interactive width = %d", got)
	}
	for _, width := range []int{0, -1} {
		capabilities := terminalCapabilities{
			Interactive: func(io.Writer) bool { return true },
			Width:       func(io.Writer) (int, error) { return width, nil },
		}
		if got := terminalDisplayWidth(io.Discard, capabilities); got != defaultTerminalWidth {
			t.Fatalf("unavailable width %d resolved to %d", width, got)
		}
	}
	capabilities := terminalCapabilities{
		Interactive: func(io.Writer) bool { return true },
		Width:       func(io.Writer) (int, error) { return 0, errors.New("size") },
	}
	if got := terminalDisplayWidth(io.Discard, capabilities); got != defaultTerminalWidth {
		t.Fatalf("failed terminal sizing resolved to %d", got)
	}
}
