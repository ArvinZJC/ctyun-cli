/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestOperationProgressRendersAndClearsInteractiveLine(t *testing.T) {
	var stderr bytes.Buffer
	display := newOperationProgressWithCapabilities(&stderr, terminalCapabilities{
		Interactive: func(io.Writer) bool { return true },
		Width:       func(io.Writer) (int, error) { return 40, nil },
	})

	if err := display.Start(5); err != nil {
		t.Fatal(err)
	}
	if err := display.Update(2, "Installing region"); err != nil {
		t.Fatal(err)
	}
	if err := display.Clear(); err != nil {
		t.Fatal(err)
	}

	got := stderr.String()
	for _, want := range []string{"\r[", "] 2/5  Installing region", "\r\x1b[2K"} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output %q missing %q", got, want)
		}
	}
}

func TestOperationProgressIsSilentForNonTerminalWriter(t *testing.T) {
	var stderr bytes.Buffer
	display := newOperationProgressWithCapabilities(&stderr, terminalCapabilities{
		Interactive: func(io.Writer) bool { return false },
		Width:       func(io.Writer) (int, error) { return 80, nil },
	})
	if err := display.Start(1); err != nil {
		t.Fatal(err)
	}
	if err := display.Update(1, "Installing ecs"); err != nil {
		t.Fatal(err)
	}
	if err := display.Clear(); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("non-terminal progress output = %q", stderr.String())
	}
}

func TestOperationProgressFitsNarrowTerminalAndWideLabel(t *testing.T) {
	for _, width := range []int{8, 24} {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			var stderr bytes.Buffer
			display := newOperationProgressWithCapabilities(&stderr, terminalCapabilities{
				Interactive: func(io.Writer) bool { return true },
				Width:       func(io.Writer) (int, error) { return width, nil },
			})
			if err := display.Start(10); err != nil {
				t.Fatal(err)
			}
			if err := display.Update(3, "正在安装 region"); err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(stderr.String(), "\r") {
				if runewidth.StringWidth(strings.TrimSuffix(line, "\x1b[2K")) > width {
					t.Fatalf("progress line %q exceeds terminal width %d", line, width)
				}
			}
		})
	}
}

func TestOperationProgressUsesFallbackWidth(t *testing.T) {
	var stderr bytes.Buffer
	display := newOperationProgressWithCapabilities(&stderr, terminalCapabilities{
		Interactive: func(io.Writer) bool { return true },
		Width:       func(io.Writer) (int, error) { return 0, errors.New("unavailable") },
	})
	if err := display.Start(1); err != nil {
		t.Fatal(err)
	}
	if err := display.Update(1, strings.Repeat("x", 100)); err != nil {
		t.Fatal(err)
	}
	if got := runewidth.StringWidth(strings.TrimPrefix(stderr.String(), "\r")); got > 80 {
		t.Fatalf("fallback-width line uses %d cells, want at most 80", got)
	}
}

func TestDefaultTerminalCapabilitiesHandleFilesAndOtherWriters(t *testing.T) {
	if defaultTerminalCapabilities.Interactive(&bytes.Buffer{}) {
		t.Fatal("buffer reported as interactive")
	}
	if _, err := defaultTerminalCapabilities.Width(&bytes.Buffer{}); err == nil {
		t.Fatal("buffer width returned nil error")
	}
	_ = defaultTerminalCapabilities.Interactive(os.Stderr)
	_, _ = defaultTerminalCapabilities.Width(os.Stderr)
}

func TestOperationProgressReturnsWriterError(t *testing.T) {
	display := newOperationProgressWithCapabilities(failingWriter{}, terminalCapabilities{
		Interactive: func(io.Writer) bool { return true },
		Width:       func(io.Writer) (int, error) { return 80, nil },
	})
	if err := display.Start(1); err != nil {
		t.Fatal(err)
	}
	if err := display.Update(0, "Installing ecs"); err == nil {
		t.Fatal("progress writer error was ignored")
	}
}
