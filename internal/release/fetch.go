/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

const (
	indexName     = "core-index.json"
	signatureName = "core-index.sig"
)

// ReadSignedIndex reads a hosted release index and verifies its detached
// Ed25519 signature before returning index bytes.
func ReadSignedIndex(source string, publicKey string, transport http.RoundTripper) ([]byte, error) {
	if !distribution.IsHTTPURL(source) {
		index, signature, err := readLocalIndexAndSignature(source)
		if err != nil {
			return nil, err
		}
		if err := VerifyIndexSignature(index, signature, publicKey); err != nil {
			return nil, diagnostic.Wrap("error.index_signature", err, "release")
		}
		return index, nil
	}
	return distribution.ReadSignedIndex(source, indexName, signatureName, publicKey, "release", transport)
}

// VerifyIndexSignature verifies a release index with a trusted base64 Ed25519
// public key and a detached base64 signature.
func VerifyIndexSignature(index, signature []byte, publicKey string) error {
	return distribution.VerifyIndexSignature(index, signature, publicKey, "release")
}

// VerifySHA256 checks that path matches the expected lowercase hex digest.
func VerifySHA256(path string, want string) error {
	return distribution.VerifySHA256(path, want)
}

// PrepareArtifact downloads a release artifact to a local path and returns a
// cleanup callback for downloaded temporary files.
func PrepareArtifact(source string, artifact Artifact, transport http.RoundTripper) (string, func(), error) {
	if !distribution.IsHTTPURL(source) && !distribution.IsHTTPURL(artifact.URL) {
		return filepath.Join(source, artifact.URL), func() {}, nil
	}
	if artifact.SHA256 == "" {
		artifactURL := artifact.URL
		if !distribution.IsHTTPURL(artifactURL) {
			artifactURL = distribution.JoinURL(source, artifactURL)
		}
		return downloadArtifact(artifactURL, transport)
	}
	return distribution.PrepareArtifact(source, distribution.Artifact{Name: distribution.ArtifactBase(artifact.URL), URL: artifact.URL, SHA256: artifact.SHA256}, transport)
}

// readLocalIndexAndSignature loads test fixture index and signature files.
func readLocalIndexAndSignature(source string) ([]byte, []byte, error) {
	indexPath := source
	signaturePath := source + ".sig"
	if filepath.Base(source) == indexName {
		signaturePath = filepath.Join(filepath.Dir(source), signatureName)
	} else {
		indexPath = filepath.Join(source, indexName)
		signaturePath = filepath.Join(source, signatureName)
	}
	index, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, nil, err
	}
	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		return nil, nil, diagnostic.Wrap("error.read_index_signature", err, "release")
	}
	return index, signature, nil
}

// downloadArtifact downloads an artifact URL to a temporary file.
func downloadArtifact(artifactURL string, transport http.RoundTripper) (string, func(), error) {
	return distribution.DownloadArtifact(artifactURL, transport)
}

// httpGetBytes fetches an HTTP URL and returns successful response bytes.
func httpGetBytes(rawURL string, transport http.RoundTripper) ([]byte, error) {
	return distribution.HTTPGetBytes(rawURL, transport)
}

// joinSourceURL appends name to the path portion of a release source URL.
func joinSourceURL(root, name string) string {
	return distribution.JoinURL(root, name)
}

// isHTTPURL reports whether raw is an HTTP or HTTPS URL.
func isHTTPURL(raw string) bool {
	return distribution.IsHTTPURL(raw)
}
