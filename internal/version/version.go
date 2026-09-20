/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package version exposes build-time CLI identity values.
package version

import (
	"cmp"
	"regexp"
	"strings"
)

const (
	// Name is the shipped command name.
	Name = "ctyun"
)

var (
	// semanticVersionPattern accepts SemVer identifiers without imposing integer limits.
	semanticVersionPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	// Version is the next release version shown by unpackaged builds unless
	// release packaging overrides it.
	Version = "0.5.0"
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
		if result := compareNumericIdentifiers(leftVersion.Core[i], rightVersion.Core[i]); result != 0 {
			return result
		}
	}
	return comparePrerelease(leftVersion.Prerelease, rightVersion.Prerelease)
}

// IsDevelopmentBuild reports whether the current binary was built from the
// development channel rather than release-stamped packaging.
func IsDevelopmentBuild() bool {
	return Channel == "dev"
}

// semanticVersion retains numeric components as decimal strings to avoid overflow.
type semanticVersion struct {
	Core       [3]string
	Prerelease []string
}

// parseSemanticVersion splits valid versions and maps invalid inputs to zero.
func parseSemanticVersion(value string) semanticVersion {
	matches := semanticVersionPattern.FindStringSubmatch(value)
	if matches == nil {
		return semanticVersion{Core: [3]string{"0", "0", "0"}}
	}
	var parsed semanticVersion
	for i := range parsed.Core {
		parsed.Core[i] = matches[i+1]
	}
	if matches[4] != "" {
		parsed.Prerelease = strings.Split(matches[4], ".")
	}
	return parsed
}

// comparePrerelease applies SemVer prerelease identifier and sequence precedence.
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

// comparePrereleaseIdentifier compares numeric identifiers before lexical identifiers.
func comparePrereleaseIdentifier(left, right string) int {
	leftNumeric := numericIdentifier(left)
	rightNumeric := numericIdentifier(right)
	if leftNumeric && rightNumeric {
		return compareNumericIdentifiers(left, right)
	}
	if leftNumeric {
		return -1
	}
	if rightNumeric {
		return 1
	}
	return strings.Compare(left, right)
}

// compareNumericIdentifiers compares canonical nonnegative decimal integers of any size.
func compareNumericIdentifiers(left, right string) int {
	if result := cmp.Compare(len(left), len(right)); result != 0 {
		return result
	}
	return strings.Compare(left, right)
}

// numericIdentifier reports whether every character in a nonempty identifier is a digit.
func numericIdentifier(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return value != ""
}
