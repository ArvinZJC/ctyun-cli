/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"

	"github.com/ArvinZJC/ctyun-cli/internal/output"
)

const defaultTerminalWidth = 80

// terminalDisplayWidth returns a positive interactive terminal width or zero
// when output is not attached to a terminal.
func terminalDisplayWidth(writer io.Writer, capabilities terminalCapabilities) int {
	if capabilities.Interactive == nil || !capabilities.Interactive(writer) {
		return 0
	}
	width, err := capabilities.Width(writer)
	if err != nil || width <= 0 {
		return defaultTerminalWidth
	}
	return width
}

// renderTableOutput applies the shared terminal-width policy before rendering
// any core, plugin-management, diagnostic, or product-command table.
func renderTableOutput(writer io.Writer, rows []map[string]string, columns []output.Column, options output.TableOptions) (string, error) {
	if options.MaxWidth <= 0 {
		options.MaxWidth = terminalDisplayWidth(writer, defaultTerminalCapabilities)
	}
	return renderOutputTable(rows, columns, options)
}
