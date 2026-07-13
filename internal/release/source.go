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

// SourceOptions captures release source selection inputs.
type SourceOptions struct {
	Requested string
	Getenv    func(string) string
}

// Source is the resolved release metadata source.
type Source = distribution.Source

// ResolveSource applies source precedence for core upgrade metadata.
func ResolveSource(opts SourceOptions) (Source, error) {
	return distribution.ResolveSource(distribution.SourceOptions{
		Label:     "core upgrade",
		Requested: opts.Requested,
		EnvName:   "CTYUN_UPGRADE_SOURCE",
		GitHubURL: GitHubReleaseSource,
		GiteeURL:  GiteeReleaseSource,
		Getenv:    opts.Getenv,
	})
}
