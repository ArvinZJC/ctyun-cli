/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package release validates signed core release indexes and selects platform
// artifacts for the ctyun self-upgrade flow.
package release

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"

	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

// Index is the core release index document listing available ctyun binaries.
type Index struct {
	Schema   int       `json:"schema"`
	Releases []Release `json:"releases"`
}

// Release describes one ctyun core release in the release index.
type Release struct {
	Version     string     `json:"version"`
	Channel     string     `json:"channel"`
	PublishedAt string     `json:"published_at"`
	NotesURL    string     `json:"notes_url"`
	Artifacts   []Artifact `json:"artifacts"`
}

// Artifact describes one platform-specific downloadable ctyun binary archive.
type Artifact struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// LoadIndex decodes and validates a core release index document.
func LoadIndex(raw []byte) (Index, error) {
	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return Index{}, fmt.Errorf("parse release index: %w", err)
	}
	if err := validateIndex(idx); err != nil {
		return Index{}, err
	}
	return idx, nil
}

// VersionNewer reports whether available is newer than current.
func VersionNewer(available, current string) bool {
	return compareVersion(available, current) > 0
}

// FindLatest returns the newest artifact matching channel and Go platform.
func (i Index) FindLatest(channel, goos, goarch string) (Release, Artifact, bool) {
	if channel == "" {
		channel = "stable"
	}

	type candidate struct {
		release  Release
		artifact Artifact
	}
	candidates := make([]candidate, 0)
	for _, rel := range i.Releases {
		if rel.Channel != channel {
			continue
		}
		for _, artifact := range rel.Artifacts {
			if artifact.OS == goos && artifact.Arch == goarch {
				candidates = append(candidates, candidate{release: rel, artifact: artifact})
			}
		}
	}
	if len(candidates) == 0 {
		return Release{}, Artifact{}, false
	}

	slices.SortFunc(candidates, func(left, right candidate) int {
		return compareVersion(right.release.Version, left.release.Version)
	})
	return candidates[0].release, candidates[0].artifact, true
}

// validateIndex checks release metadata before it is trusted.
func validateIndex(idx Index) error {
	if idx.Schema != 1 {
		return fmt.Errorf("unsupported release index schema %d", idx.Schema)
	}
	for i, rel := range idx.Releases {
		prefix := fmt.Sprintf("release %d", i)
		if rel.Version == "" {
			return fmt.Errorf("%s is missing version", prefix)
		}
		if !coreversion.IsSemanticVersion(rel.Version) {
			return fmt.Errorf("%s has invalid version %q", prefix, rel.Version)
		}
		if !oneOf(rel.Channel, "stable", "beta", "alpha") {
			return fmt.Errorf("%s %s has unsupported channel %q", prefix, rel.Version, rel.Channel)
		}
		if len(rel.Artifacts) == 0 {
			return fmt.Errorf("%s %s has no artifacts", prefix, rel.Version)
		}
		for j, artifact := range rel.Artifacts {
			artifactPrefix := fmt.Sprintf("%s %s artifact %d", prefix, rel.Version, j)
			if artifact.OS == "" {
				return fmt.Errorf("%s is missing os", artifactPrefix)
			}
			if artifact.Arch == "" {
				return fmt.Errorf("%s is missing arch", artifactPrefix)
			}
			if artifact.URL == "" {
				return fmt.Errorf("%s is missing url", artifactPrefix)
			}
			if !validArtifactURL(artifact.URL) {
				return fmt.Errorf("%s has invalid artifact url %q", artifactPrefix, artifact.URL)
			}
			if !validSHA256(artifact.SHA256) {
				return fmt.Errorf("%s has invalid sha256", artifactPrefix)
			}
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
	clean := path.Clean(raw)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	return true
}

// validSHA256 reports whether value is a lowercase hex SHA-256 digest.
func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
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
