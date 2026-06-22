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
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
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
		return Index{}, diagnostic.Wrap("error.parse_release_index", err)
	}
	if err := validateIndex(idx); err != nil {
		return Index{}, err
	}
	return idx, nil
}

// VersionNewer reports whether available is newer than current.
func VersionNewer(available, current string) bool {
	return coreversion.CompareSemanticVersions(available, current) > 0
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
		return coreversion.CompareSemanticVersions(right.release.Version, left.release.Version)
	})
	return candidates[0].release, candidates[0].artifact, true
}

// validateIndex checks release metadata before it is trusted.
func validateIndex(idx Index) error {
	if idx.Schema != 1 {
		return diagnostic.New("error.unsupported_release_schema", idx.Schema)
	}
	for i, rel := range idx.Releases {
		prefix := fmt.Sprintf("release %d", i)
		if rel.Version == "" {
			return diagnostic.New("error.release_missing_version", prefix)
		}
		if !coreversion.IsSemanticVersion(rel.Version) {
			return diagnostic.New("error.release_invalid_version", prefix, rel.Version)
		}
		releasePrefix := fmt.Sprintf("%s %s", prefix, rel.Version)
		if !oneOf(rel.Channel, "stable", "beta", "alpha") {
			return diagnostic.New("error.release_unsupported_channel", releasePrefix, rel.Channel)
		}
		if len(rel.Artifacts) == 0 {
			return diagnostic.New("error.release_no_artifacts", releasePrefix)
		}
		for j, artifact := range rel.Artifacts {
			artifactPrefix := fmt.Sprintf("%s artifact %d", releasePrefix, j)
			if artifact.OS == "" {
				return diagnostic.New("error.release_missing_os", artifactPrefix)
			}
			if artifact.Arch == "" {
				return diagnostic.New("error.release_missing_arch", artifactPrefix)
			}
			if artifact.URL == "" {
				return diagnostic.New("error.release_missing_url", artifactPrefix)
			}
			if !validArtifactURL(artifact.URL) {
				return diagnostic.New("error.release_invalid_artifact_url", artifactPrefix, artifact.URL)
			}
			if !validSHA256(artifact.SHA256) {
				return diagnostic.New("error.release_invalid_sha256", artifactPrefix)
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

// oneOf reports whether value is present in allowed.
func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
