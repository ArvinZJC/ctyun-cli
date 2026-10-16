/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package coverprofile validates complete Go statement coverage without exclusions.
package coverprofile

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// CheckFullCoverage requires every executable statement block in an unfiltered
// Go coverage profile to have executed. It checks counters rather than rounded
// percentages and rejects missing, empty, or invalid coverage evidence.
func CheckFullCoverage(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("coverage profile is empty")
	}
	switch scanner.Text() {
	case "mode: set", "mode: count", "mode: atomic":
	default:
		return fmt.Errorf("invalid coverage profile mode %q", scanner.Text())
	}
	hasStatements := false
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			return fmt.Errorf("invalid coverage block %q", scanner.Text())
		}
		statements, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid statement count in %q: %w", scanner.Text(), err)
		}
		count, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid execution count in %q: %w", scanner.Text(), err)
		}
		if statements > 0 {
			hasStatements = true
			if count == 0 {
				return fmt.Errorf("coverage gate failed: %s has %d uncovered statements; require 100%% unfiltered statement coverage", fields[0], statements)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if !hasStatements {
		return fmt.Errorf("coverage profile contains no executable statements")
	}
	return nil
}
