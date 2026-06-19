/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"fmt"
	"strings"
)

const (
	// GitHubReleaseSource is the canonical GitHub-hosted core release metadata root.
	GitHubReleaseSource = "https://github.com/ArvinZJC/ctyun-cli/releases/download/ctyun-core"
	// GiteeReleaseSource is the official mirrored core release metadata root.
	GiteeReleaseSource = "https://gitee.com/ArvinZJC/ctyun-cli/releases/download/ctyun-core"
)

// SourceKind describes whether a resolved source is ready to fetch.
type SourceKind int

const (
	// SourceReady means URL contains a concrete source to read.
	SourceReady SourceKind = iota
	// SourceDevelopmentUnavailable means auto upgrade is intentionally disabled
	// for a development build that has no explicit release source.
	SourceDevelopmentUnavailable
)

// SourceOptions captures release source selection inputs.
type SourceOptions struct {
	Requested      string
	CurrentVersion string
	Getenv         func(string) string
}

// Source is the resolved release metadata source.
type Source struct {
	Name string
	URL  string
	Kind SourceKind
}

// ResolveSource applies source precedence for core upgrade metadata.
func ResolveSource(opts SourceOptions) (Source, error) {
	getenv := opts.Getenv
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	requested := strings.TrimSpace(opts.Requested)
	if requested == "" {
		requested = strings.TrimSpace(getenv("CTYUN_UPGRADE_SOURCE"))
	}
	if requested == "" {
		requested = strings.TrimSpace(getenv("CTYUN_UPGRADE_URL"))
	}
	if requested == "" {
		requested = "auto"
	}

	switch requested {
	case "auto":
		if strings.HasSuffix(opts.CurrentVersion, "-dev") {
			return Source{Name: "auto", Kind: SourceDevelopmentUnavailable}, nil
		}
		return Source{Name: "github", URL: GitHubReleaseSource, Kind: SourceReady}, nil
	case "github":
		return Source{Name: "github", URL: GitHubReleaseSource, Kind: SourceReady}, nil
	case "gitee":
		return Source{Name: "gitee", URL: GiteeReleaseSource, Kind: SourceReady}, nil
	default:
		if requested == "" {
			return Source{}, fmt.Errorf("upgrade source is empty")
		}
		return Source{Name: "custom", URL: requested, Kind: SourceReady}, nil
	}
}
