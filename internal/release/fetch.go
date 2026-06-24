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
		if err := distribution.VerifyIndexSignature(index, signature, publicKey, "release"); err != nil {
			return nil, diagnostic.Wrap("error.index_signature", err, "release")
		}
		return index, nil
	}
	return distribution.ReadSignedIndex(source, indexName, signatureName, publicKey, "release", transport)
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
		return distribution.DownloadArtifact(artifactURL, transport)
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
