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
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

const (
	// GitHubPluginSource is the canonical GitHub-hosted plugin registry root.
	GitHubPluginSource = "https://github.com/ArvinZJC/ctyun-cli/releases/download/plugins"
	// GiteePluginSource is the official mirrored plugin registry root.
	GiteePluginSource = "https://gitee.com/ArvinZJC/ctyun-cli/releases/download/plugins"
)

// Index is the registry index document listing available plugin artifacts.
type Index struct {
	Plugins []Artifact `json:"plugins"`
}

// Artifact describes one downloadable plugin bundle in a registry index.
type Artifact struct {
	Name        string `json:"name"`
	Product     string `json:"product,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Version     string `json:"version"`
	Channel     string `json:"channel"`
	Quality     string `json:"quality"`
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
}

// LoadIndex decodes and validates a registry index document.
func LoadIndex(raw []byte) (Index, error) {
	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return Index{}, diagnostic.Wrap("error.parse_registry_index", err)
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
			return diagnostic.New("error.registry_missing_name", prefix)
		}
		if !plugin.ValidName(artifact.Name) {
			return diagnostic.New("error.registry_invalid_plugin_name", prefix, artifact.Name)
		}
		artifactPrefix := fmt.Sprintf("%s %s", prefix, artifact.Name)
		if artifact.Version == "" {
			return diagnostic.New("error.registry_missing_version", artifactPrefix)
		}
		if !coreversion.IsSemanticVersion(artifact.Version) {
			return diagnostic.New("error.registry_invalid_version", artifactPrefix, artifact.Version)
		}
		if !oneOf(artifact.Channel, "stable", "beta", "alpha") {
			return diagnostic.New("error.registry_unsupported_channel", artifactPrefix, artifact.Channel)
		}
		if !oneOf(artifact.Quality, "generated", "reviewed", "curated") {
			return diagnostic.New("error.registry_unsupported_quality", artifactPrefix, artifact.Quality)
		}
		if artifact.URL == "" {
			return diagnostic.New("error.registry_missing_url", artifactPrefix)
		}
		if !validArtifactURL(artifact.URL) {
			return diagnostic.New("error.registry_invalid_artifact_url", artifactPrefix, artifact.URL)
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
		return coreversion.CompareSemanticVersions(right.Version, left.Version)
	})
	return candidates[0], true
}

// Search returns the newest acceptable artifact per plugin name, optionally
// filtered by a case-insensitive fuzzy query.
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
		if query != "" && !artifactMatchesQuery(artifact, query) {
			continue
		}
		current, ok := latestByName[artifact.Name]
		if !ok || coreversion.CompareSemanticVersions(artifact.Version, current.Version) > 0 {
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

// artifactMatchesQuery reports whether query fuzzily matches searchable
// registry storefront fields.
func artifactMatchesQuery(artifact Artifact, query string) bool {
	for _, value := range []string{artifact.Name, artifact.Product, artifact.DisplayName} {
		value = strings.ToLower(value)
		if strings.Contains(value, query) || isSubsequence(query, value) {
			return true
		}
	}
	return false
}

// isSubsequence reports whether every query rune appears in order in value.
func isSubsequence(query, value string) bool {
	if query == "" {
		return true
	}
	queryRunes := []rune(query)
	next := 0
	for _, r := range value {
		if r == queryRunes[next] {
			next++
			if next == len(queryRunes) {
				return true
			}
		}
	}
	return false
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

// oneOf reports whether value is present in allowed.
func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
