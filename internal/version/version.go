/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package version exposes build-time CLI identity values.
package version

const (
	// Name is the shipped command name.
	Name = "ctyun"
)

var (
	// Version is the default development version when no release build overrides it.
	Version = "0.1.0-dev"
	// Commit is the source revision stamped into release builds.
	Commit = "unknown"
	// Date is the build timestamp stamped into release builds.
	Date = "unknown"
	// Channel is the release channel stamped into release builds.
	Channel = "dev"
	// ReleasePublicKey is the trusted base64 Ed25519 key for core release indexes.
	ReleasePublicKey = ""
)
