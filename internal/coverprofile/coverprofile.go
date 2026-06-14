package coverprofile

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Exclusion struct {
	File      string
	StartLine int
	EndLine   int
	WholeFile bool
}

func DefaultExclusions() []Exclusion {
	return []Exclusion{
		{File: "cmd/ctyun/main.go", WholeFile: true},
		{File: "tools/coverage/main.go", WholeFile: true},
		{File: "internal/cli/locale_windows.go", WholeFile: true},
		{File: "internal/plugin/install.go", StartLine: 111, EndLine: 113},
		{File: "internal/plugin/install.go", StartLine: 179, EndLine: 181},
		{File: "internal/cli/cli.go", StartLine: 523, EndLine: 525},
		{File: "internal/cli/cli.go", StartLine: 1248, EndLine: 1251},
		{File: "internal/cli/cli.go", StartLine: 1253, EndLine: 1255},
		{File: "internal/cli/cli.go", StartLine: 1844, EndLine: 1846},
		{File: "internal/cli/cli.go", StartLine: 1854, EndLine: 1856},
	}
}

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
