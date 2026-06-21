/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

const (
	// GitHubReleaseSource is the canonical GitHub-hosted core release metadata root.
	GitHubReleaseSource = "https://github.com/ArvinZJC/ctyun-cli/releases/download/core"
	// GiteeReleaseSource is the official mirrored core release metadata root.
	GiteeReleaseSource = "https://gitee.com/ArvinZJC/ctyun-cli/releases/download/core"
)

// SourceKind describes whether a resolved source is ready to fetch.
type SourceKind = distribution.SourceKind

const (
	// SourceReady means URL contains a concrete source to read.
	SourceReady = distribution.SourceReady
	// SourceDevelopmentUnavailable means auto upgrade is intentionally disabled
	// for a development build that has no explicit release source.
	SourceDevelopmentUnavailable = distribution.SourceDevelopmentUnavailable
)

// SourceOptions captures release source selection inputs.
type SourceOptions struct {
	Requested      string
	CurrentVersion string
	Getenv         func(string) string
}

// Source is the resolved release metadata source.
type Source = distribution.Source

// ResolveSource applies source precedence for core upgrade metadata.
func ResolveSource(opts SourceOptions) (Source, error) {
	return distribution.ResolveSource(distribution.SourceOptions{
		Label:          "core upgrade",
		Requested:      opts.Requested,
		EnvName:        "CTYUN_UPGRADE_SOURCE",
		CurrentVersion: opts.CurrentVersion,
		GitHubURL:      GitHubReleaseSource,
		GiteeURL:       GiteeReleaseSource,
		DisableDevAuto: true,
		Getenv:         opts.Getenv,
	})
}
