/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package version exposes build-time CLI identity values.
package version

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	// Name is the shipped command name.
	Name = "ctyun"
)

var (
	semanticVersionPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	// Version is the next release version shown by unpackaged builds unless
	// release packaging overrides it.
	Version = "0.4.0"
	// Channel is the build channel; source builds stay dev until release
	// packaging stamps stable, beta, or alpha.
	Channel = "dev"
	// ReleasePublicKey is the trusted base64 Ed25519 key for core release indexes.
	ReleasePublicKey = ""
)

// IsSemanticVersion reports whether value follows Semantic Versioning 2.0.0.
func IsSemanticVersion(value string) bool {
	return semanticVersionPattern.MatchString(value)
}

// CompareSemanticVersions compares two SemVer 2.0.0 versions by precedence.
// Build metadata is ignored. Valid SemVer inputs are expected.
func CompareSemanticVersions(left, right string) int {
	leftVersion := parseSemanticVersion(left)
	rightVersion := parseSemanticVersion(right)
	for i := range len(leftVersion.Core) {
		if leftVersion.Core[i] < rightVersion.Core[i] {
			return -1
		}
		if leftVersion.Core[i] > rightVersion.Core[i] {
			return 1
		}
	}
	return comparePrerelease(leftVersion.Prerelease, rightVersion.Prerelease)
}

// IsDevelopmentBuild reports whether the current binary was built from the
// development channel rather than release-stamped packaging.
func IsDevelopmentBuild() bool {
	return Channel == "dev"
}

type semanticVersion struct {
	Core       [3]int
	Prerelease []string
}

func parseSemanticVersion(value string) semanticVersion {
	matches := semanticVersionPattern.FindStringSubmatch(value)
	if matches == nil {
		return semanticVersion{}
	}
	var parsed semanticVersion
	for i := range parsed.Core {
		parsed.Core[i], _ = strconv.Atoi(matches[i+1])
	}
	if matches[4] != "" {
		parsed.Prerelease = strings.Split(matches[4], ".")
	}
	return parsed
}

func comparePrerelease(left, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}
	for i := 0; i < len(left) && i < len(right); i++ {
		if result := comparePrereleaseIdentifier(left[i], right[i]); result != 0 {
			return result
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

func comparePrereleaseIdentifier(left, right string) int {
	leftNumber, leftNumeric := numericIdentifier(left)
	rightNumber, rightNumeric := numericIdentifier(right)
	if leftNumeric && rightNumeric {
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
		return 0
	}
	if leftNumeric {
		return -1
	}
	if rightNumeric {
		return 1
	}
	return strings.Compare(left, right)
}

func numericIdentifier(value string) (int, bool) {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(value)
	return n, err == nil
}
