/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package coverprofile filters generated Go coverage profiles for the local
// repository coverage gate.
package coverprofile

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Exclusion describes a coverage profile block that should be ignored.
type Exclusion struct {
	File      string
	StartLine int
	EndLine   int
	WholeFile bool
}

// DefaultExclusions returns the repository's intentionally unreachable or
// platform-specific coverage blocks.
func DefaultExclusions() []Exclusion {
	return []Exclusion{
		{File: "cmd/ctyun/main.go", WholeFile: true},
		{File: "tools/coverage/main.go", WholeFile: true},
		{File: "internal/cli/locale_windows.go", WholeFile: true},
		{File: "internal/plugin/install.go", StartLine: 124, EndLine: 126},
		{File: "internal/plugin/install.go", StartLine: 194, EndLine: 196},
		{File: "internal/cli/cli.go", StartLine: 534, EndLine: 536},
		{File: "internal/cli/cli.go", StartLine: 1260, EndLine: 1263},
		{File: "internal/cli/cli.go", StartLine: 1265, EndLine: 1267},
		{File: "internal/cli/cli.go", StartLine: 1869, EndLine: 1871},
		{File: "internal/cli/cli.go", StartLine: 1882, EndLine: 1884},
	}
}

// Filter copies a coverage profile from r to w while dropping matching
// exclusions.
func Filter(r io.Reader, w io.Writer, exclusions []Exclusion) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if shouldExclude(line, exclusions) {
			continue
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// TotalPercent extracts the total coverage percentage from "go tool cover"
// output.
func TotalPercent(report string) string {
	for _, line := range strings.Split(report, "\n") {
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

func shouldExclude(line string, exclusions []Exclusion) bool {
	file, start, end, ok := parseBlock(line)
	if !ok {
		return false
	}
	for _, exclusion := range exclusions {
		if !strings.HasSuffix(file, exclusion.File) {
			continue
		}
		if exclusion.WholeFile || (start == exclusion.StartLine && end == exclusion.EndLine) {
			return true
		}
	}
	return false
}

func parseBlock(line string) (string, int, int, bool) {
	file, rest, ok := strings.Cut(line, ":")
	if !ok {
		return "", 0, 0, false
	}
	startText, rest, ok := strings.Cut(rest, ".")
	if !ok {
		return "", 0, 0, false
	}
	_, rest, ok = strings.Cut(rest, ",")
	if !ok {
		return "", 0, 0, false
	}
	endText, _, ok := strings.Cut(rest, ".")
	if !ok {
		return "", 0, 0, false
	}
	start, err := strconv.Atoi(startText)
	if err != nil {
		return "", 0, 0, false
	}
	end, err := strconv.Atoi(endText)
	if err != nil {
		return "", 0, 0, false
	}
	return file, start, end, true
}
