/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
)

// outputWriter records the first write failure while allowing callers to keep
// linear rendering code.
type outputWriter struct {
	writer io.Writer
	err    error
}

// newOutputWriter returns an error-recording wrapper around writer.
func newOutputWriter(writer io.Writer) *outputWriter {
	return &outputWriter{writer: writer}
}

// Line writes one newline-terminated line unless a previous write failed.
func (writer *outputWriter) Line(values ...any) {
	if writer.err != nil {
		return
	}
	writer.err = writeLine(writer.writer, values...)
}

// Format writes formatted text unless a previous write failed.
func (writer *outputWriter) Format(format string, values ...any) {
	if writer.err != nil {
		return
	}
	writer.err = writeFormat(writer.writer, format, values...)
}

// String writes raw text unless a previous write failed.
func (writer *outputWriter) String(value string) {
	if writer.err != nil {
		return
	}
	writer.err = writeString(writer.writer, value)
}

// Lines writes newline-terminated lines in order until one write fails.
func (writer *outputWriter) Lines(lines ...string) {
	for _, line := range lines {
		writer.Line(line)
	}
}

// Err returns the first write failure, if any.
func (writer *outputWriter) Err() error {
	return writer.err
}

// writeLine writes one newline-terminated line and returns writer failures.
func writeLine(writer io.Writer, values ...any) error {
	_, err := fmt.Fprintln(writer, values...)
	return err
}

// writeFormat writes formatted text and returns writer failures.
func writeFormat(writer io.Writer, format string, values ...any) error {
	_, err := fmt.Fprintf(writer, format, values...)
	return err
}

// writeString writes raw text and returns writer failures.
func writeString(writer io.Writer, value string) error {
	_, err := io.WriteString(writer, value)
	return err
}

// writeLines writes newline-terminated lines in order and stops at the first
// writer failure.
func writeLines(writer io.Writer, lines ...string) error {
	for _, line := range lines {
		if err := writeLine(writer, line); err != nil {
			return err
		}
	}
	return nil
}
