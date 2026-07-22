/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
)

const defaultGiteePluginDownloadRoot = "https://gitee.com/ArvinZJC/ctyun-cli/releases/download"

// encodeGiteeJSON is the JSON encoding seam used to verify publication error
// propagation without changing the release document types.
var encodeGiteeJSON = indentedJSON

// giteeReleaseManifest describes immutable Gitee Releases that receive the
// plugin archives referenced by the separately signed Gitee registry index.
type giteeReleaseManifest struct {
	Schema   int                  `json:"schema"`
	Releases []giteePluginRelease `json:"releases"`
}

// giteePluginRelease maps one plugin artifact to its immutable version tag.
type giteePluginRelease struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Channel string `json:"channel"`
	Tag     string `json:"tag"`
	Archive string `json:"archive"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// writeGiteePluginRegistry derives absolute immutable-release URLs from the
// canonical merged plugin index and writes separately signed Gitee metadata.
func writeGiteePluginRegistry(outDir, downloadRoot string, privateKey ed25519.PrivateKey) error {
	if downloadRoot == "" {
		downloadRoot = defaultGiteePluginDownloadRoot
	}
	canonicalBytes, err := os.ReadFile(filepath.Join(outDir, "index.json"))
	if err != nil {
		return err
	}
	index, err := registry.LoadIndex(canonicalBytes)
	if err != nil {
		return err
	}
	manifest := giteeReleaseManifest{Schema: 1, Releases: make([]giteePluginRelease, 0, len(index.Plugins))}
	for i := range index.Plugins {
		artifact := &index.Plugins[i]
		archive := distribution.ArtifactBase(artifact.URL)
		tag := "releases/plugins/" + artifact.Name + "/" + artifact.Version
		artifact.URL = giteePluginDownloadURL(downloadRoot, tag, archive)
		manifest.Releases = append(manifest.Releases, giteePluginRelease{
			Name: artifact.Name, Version: artifact.Version, Channel: artifact.Channel,
			Tag: tag, Archive: archive, URL: artifact.URL, SHA256: artifact.SHA256,
		})
	}
	slices.SortFunc(manifest.Releases, func(left, right giteePluginRelease) int {
		if left.Tag != right.Tag {
			return strings.Compare(left.Tag, right.Tag)
		}
		return strings.Compare(left.Channel, right.Channel)
	})

	giteeDir := filepath.Join(outDir, "gitee")
	if err := os.MkdirAll(giteeDir, 0o755); err != nil {
		return err
	}
	indexBytes, err := encodeGiteeJSON(index)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(giteeDir, "index.json"), indexBytes, 0o644); err != nil {
		return err
	}
	signature := ed25519.Sign(privateKey, indexBytes)
	if err := os.WriteFile(filepath.Join(giteeDir, "index.sig"), []byte(base64.StdEncoding.EncodeToString(signature)), 0o644); err != nil {
		return err
	}
	manifestBytes, err := encodeGiteeJSON(manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(giteeDir, "releases.json"), manifestBytes, 0o644)
}

// giteePluginDownloadURL constructs one attachment URL while preserving the
// slash-containing release tag as a single escaped URL path segment.
func giteePluginDownloadURL(downloadRoot, tag, archive string) string {
	return strings.TrimRight(downloadRoot, "/") + "/" + url.PathEscape(tag) + "/" + url.PathEscape(archive)
}

// validateGiteePluginDownloadRoot requires a credential-free HTTPS root.
func validateGiteePluginDownloadRoot(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("--gitee-plugin-download-root must be an HTTPS URL without credentials, query, or fragment")
	}
	return nil
}

// indentedJSON encodes one release document with stable formatting and a
// trailing newline suitable for signing and publication.
func indentedJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
