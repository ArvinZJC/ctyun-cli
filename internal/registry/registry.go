/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package registry validates plugin registry indexes and verifies downloaded
// artifacts.
package registry

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

const (
	// GitHubPluginSource is the canonical GitHub-hosted plugin registry root.
	GitHubPluginSource = "https://github.com/ArvinZJC/ctyun-cli/releases/download/ctyun-plugins"
	// GiteePluginSource is the official mirrored plugin registry root.
	GiteePluginSource = "https://gitee.com/ArvinZJC/ctyun-cli/releases/download/ctyun-plugins"
)

// Index is the registry index document listing available plugin artifacts.
type Index struct {
	Plugins []Artifact `json:"plugins"`
}

// Artifact describes one downloadable plugin bundle in a registry index.
type Artifact struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Channel string `json:"channel"`
	Quality string `json:"quality"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// LoadIndex decodes and validates a registry index document.
func LoadIndex(raw []byte) (Index, error) {
	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return Index{}, fmt.Errorf("parse registry index: %w", err)
	}
	if err := validateIndex(idx); err != nil {
		return Index{}, err
	}
	return idx, nil
}

// validateIndex checks registry artifact metadata before it is trusted.
func validateIndex(idx Index) error {
	for i, artifact := range idx.Plugins {
		prefix := fmt.Sprintf("registry plugin %d", i)
		if artifact.Name == "" {
			return fmt.Errorf("%s is missing name", prefix)
		}
		if !plugin.ValidName(artifact.Name) {
			return fmt.Errorf("%s has invalid plugin name %q", prefix, artifact.Name)
		}
		if artifact.Version == "" {
			return fmt.Errorf("%s %s is missing version", prefix, artifact.Name)
		}
		if !coreversion.IsSemanticVersion(artifact.Version) {
			return fmt.Errorf("%s %s has invalid version %q", prefix, artifact.Name, artifact.Version)
		}
		if !oneOf(artifact.Channel, "stable", "beta", "edge") {
			return fmt.Errorf("%s %s has unsupported channel %q", prefix, artifact.Name, artifact.Channel)
		}
		if !oneOf(artifact.Quality, "generated", "reviewed", "curated") {
			return fmt.Errorf("%s %s has unsupported quality %q", prefix, artifact.Name, artifact.Quality)
		}
		if artifact.URL == "" {
			return fmt.Errorf("%s %s is missing url", prefix, artifact.Name)
		}
		if !validArtifactURL(artifact.URL) {
			return fmt.Errorf("%s %s has invalid artifact url %q", prefix, artifact.Name, artifact.URL)
		}
	}
	return nil
}

// validArtifactURL accepts HTTP(S) URLs and safe relative artifact paths.
func validArtifactURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Scheme != "" {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "\\") {
		return false
	}
	if strings.Contains(raw, "\\") {
		return false
	}
	return distribution.SafeRelativePath(raw)
}

// Find returns the newest acceptable artifact for name and channel.
func (i Index) Find(name, channel string) (Artifact, bool) {
	if channel == "" {
		channel = "stable"
	}

	candidates := make([]Artifact, 0)
	for _, artifact := range i.Plugins {
		if artifact.Name != name || artifact.Channel != channel {
			continue
		}
		if channel == "stable" && artifact.Quality != "reviewed" && artifact.Quality != "curated" {
			continue
		}
		candidates = append(candidates, artifact)
	}
	if len(candidates) == 0 {
		return Artifact{}, false
	}

	slices.SortFunc(candidates, func(left, right Artifact) int {
		return compareVersion(right.Version, left.Version)
	})
	return candidates[0], true
}

// Search returns the newest acceptable artifact per plugin name, optionally
// filtered by a case-insensitive name query.
func (i Index) Search(query, channel string) []Artifact {
	if channel == "" {
		channel = "stable"
	}
	query = strings.ToLower(query)

	latestByName := make(map[string]Artifact)
	for _, artifact := range i.Plugins {
		if artifact.Channel != channel {
			continue
		}
		if channel == "stable" && artifact.Quality != "reviewed" && artifact.Quality != "curated" {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(artifact.Name), query) {
			continue
		}
		current, ok := latestByName[artifact.Name]
		if !ok || compareVersion(artifact.Version, current.Version) > 0 {
			latestByName[artifact.Name] = artifact
		}
	}

	results := make([]Artifact, 0, len(latestByName))
	for _, artifact := range latestByName {
		results = append(results, artifact)
	}
	slices.SortFunc(results, func(left, right Artifact) int {
		return strings.Compare(left.Name, right.Name)
	})
	return results
}

// VerifySHA256 checks that the file at path matches the expected hex digest.
func VerifySHA256(path, want string) error {
	return distribution.VerifySHA256(path, want)
}

// VerifyIndexSignature verifies an HTTP registry index with a trusted base64
// Ed25519 public key.
func VerifyIndexSignature(index, signature []byte, publicKey string) error {
	return distribution.VerifyIndexSignature(index, signature, publicKey, "registry")
}

// compareVersion compares dotted numeric versions from oldest to newest.
func compareVersion(left, right string) int {
	leftParts := parseVersion(left)
	rightParts := parseVersion(right)
	for i := 0; i < len(leftParts); i++ {
		if leftParts[i] < rightParts[i] {
			return -1
		}
		if leftParts[i] > rightParts[i] {
			return 1
		}
	}
	return 0
}

// parseVersion converts a dotted version string into a three-part numeric key.
func parseVersion(version string) [3]int {
	var parsed [3]int
	parts := strings.Split(version, ".")
	for i := 0; i < len(parsed) && i < len(parts); i++ {
		value, _ := strconv.Atoi(parts[i])
		parsed[i] = value
	}
	return parsed
}

// oneOf reports whether value is present in allowed.
func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
