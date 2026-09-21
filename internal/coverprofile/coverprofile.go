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

// Exclusion describes a source range whose contained coverage blocks should be ignored.
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
		{File: "tools/openapi/main.go", WholeFile: true},
		{File: "tools/release/main.go", WholeFile: true},
		{File: "internal/testarchive/archive.go", WholeFile: true},
		{File: "internal/distribution/fetch.go", StartLine: 109, EndLine: 110},
		{File: "internal/distribution/fetch.go", StartLine: 113, EndLine: 116},
		{File: "internal/distribution/fetch.go", StartLine: 118, EndLine: 120},
		{File: "internal/release/install.go", StartLine: 57, EndLine: 59},
		{File: "internal/release/install.go", StartLine: 66, EndLine: 68},
		{File: "internal/release/install.go", StartLine: 70, EndLine: 72},
		{File: "internal/release/install.go", StartLine: 75, EndLine: 77},
		{File: "internal/release/install.go", StartLine: 84, EndLine: 86},
		{File: "internal/release/install.go", StartLine: 118, EndLine: 120},
		{File: "internal/release/install.go", StartLine: 129, EndLine: 131},
		{File: "internal/plugin/install.go", StartLine: 145, EndLine: 147},
		{File: "internal/plugin/install.go", StartLine: 222, EndLine: 224},
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
	for line := range strings.SplitSeq(report, "\n") {
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// shouldExclude reports whether a coverage block is wholly contained in an
// exclusion, allowing compiler versions to split blocks within the same range.
func shouldExclude(line string, exclusions []Exclusion) bool {
	file, start, end, ok := parseBlock(line)
	if !ok {
		return false
	}
	file = strings.ReplaceAll(file, "\\", "/")
	for _, exclusion := range exclusions {
		if !strings.HasSuffix(file, exclusion.File) {
			continue
		}
		if exclusion.WholeFile || (start >= exclusion.StartLine && end <= exclusion.EndLine) {
			return true
		}
	}
	return false
}

// parseBlock extracts the file and line range from one coverage profile block.
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
