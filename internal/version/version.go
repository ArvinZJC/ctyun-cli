/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package version exposes build-time CLI identity values.
package version

import "regexp"

const (
	// Name is the shipped command name.
	Name = "ctyun"
)

var (
	// Version is the default development version when no release build overrides it.
	Version = "0.1.0-dev"
	// Channel is the release channel stamped into release builds.
	Channel = "dev"
	// ReleasePublicKey is the trusted base64 Ed25519 key for core release indexes.
	ReleasePublicKey = ""
)

// IsSemanticVersion reports whether value follows Semantic Versioning 2.0.0.
func IsSemanticVersion(value string) bool {
	pattern := `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`
	return regexp.MustCompile(pattern).MatchString(value)
}

// IsDevelopmentBuild reports whether the current binary was built from the
// development channel rather than release-stamped packaging.
func IsDevelopmentBuild() bool {
	return Channel == "dev"
}
