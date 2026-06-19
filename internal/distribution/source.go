/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package distribution contains shared hosted update-source plumbing for core
// releases and plugin registries.
package distribution

import (
	"fmt"
	"strings"
)

// SourceKind describes whether a resolved source is ready to fetch.
type SourceKind int

const (
	// SourceReady means URL contains a concrete hosted source to read.
	SourceReady SourceKind = iota
	// SourceDevelopmentUnavailable means auto update is intentionally disabled
	// for a development build without an explicit hosted source.
	SourceDevelopmentUnavailable
)

// SourceOptions captures hosted source selection inputs.
type SourceOptions struct {
	Label          string
	Requested      string
	EnvName        string
	CurrentVersion string
	GitHubURL      string
	GiteeURL       string
	DisableDevAuto bool
	Getenv         func(string) string
}

// Source is the resolved hosted metadata source.
type Source struct {
	Name      string
	URL       string
	Kind      SourceKind
	Fallbacks []Source
}

// ResolveSource applies auto/github/gitee source precedence.
func ResolveSource(opts SourceOptions) (Source, error) {
	getenv := opts.Getenv
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	label := opts.Label
	if label == "" {
		label = "update"
	}

	requested := strings.TrimSpace(opts.Requested)
	if requested == "" && opts.EnvName != "" {
		requested = strings.TrimSpace(getenv(opts.EnvName))
	}
	if requested == "" {
		requested = "auto"
	}

	switch requested {
	case "auto":
		if opts.DisableDevAuto && strings.HasSuffix(opts.CurrentVersion, "-dev") {
			return Source{Name: "auto", Kind: SourceDevelopmentUnavailable}, nil
		}
		return Source{Name: "github", URL: opts.GitHubURL, Kind: SourceReady, Fallbacks: []Source{{Name: "gitee", URL: opts.GiteeURL, Kind: SourceReady}}}, nil
	case "github":
		return Source{Name: "github", URL: opts.GitHubURL, Kind: SourceReady}, nil
	case "gitee":
		return Source{Name: "gitee", URL: opts.GiteeURL, Kind: SourceReady}, nil
	default:
		return Source{}, fmt.Errorf("unsupported %s source %q; use auto, github, or gitee", label, requested)
	}
}
