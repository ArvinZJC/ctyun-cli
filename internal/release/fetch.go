/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	stdpath "path"
	"path/filepath"
	"strings"
)

const (
	indexName     = "core-index.json"
	signatureName = "core-index.sig"
)

// ReadSignedIndex reads a local or HTTP release index and verifies its detached
// Ed25519 signature before returning index bytes.
func ReadSignedIndex(source string, publicKey string, transport http.RoundTripper) ([]byte, error) {
	index, signature, err := readIndexAndSignature(source, transport)
	if err != nil {
		return nil, err
	}
	if err := VerifyIndexSignature(index, signature, publicKey); err != nil {
		return nil, fmt.Errorf("release index signature: %w", err)
	}
	return index, nil
}

// VerifyIndexSignature verifies a release index with a trusted base64 Ed25519
// public key and a detached base64 signature.
func VerifyIndexSignature(index, signature []byte, publicKey string) error {
	if publicKey == "" {
		return fmt.Errorf("release index requires a trusted public key")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return fmt.Errorf("decode release public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("release public key has length %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil {
		return fmt.Errorf("decode release signature: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(keyBytes), index, signatureBytes) {
		return fmt.Errorf("release index signature verification failed")
	}
	return nil
}

// VerifySHA256 checks that path matches the expected lowercase hex digest.
func VerifySHA256(path string, want string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != want {
		return fmt.Errorf("sha256 mismatch for %s: got %s, want %s", path, got, want)
	}
	return nil
}

// PrepareArtifact resolves a release artifact to a local path and returns a
// cleanup callback for downloaded temporary files.
func PrepareArtifact(source string, artifact Artifact, transport http.RoundTripper) (string, func(), error) {
	if isHTTPURL(artifact.URL) {
		return downloadArtifact(artifact.URL, transport)
	}
	if !isHTTPURL(source) {
		return filepath.Join(source, artifact.URL), func() {}, nil
	}
	return downloadArtifact(joinSourceURL(source, artifact.URL), transport)
}

// downloadArtifact downloads an artifact URL to a temporary file.
func downloadArtifact(artifactURL string, transport http.RoundTripper) (string, func(), error) {
	data, err := httpGetBytes(artifactURL, transport)
	if err != nil {
		return "", func() {}, err
	}
	tmp, err := os.CreateTemp("", "ctyun-core-*.tar.gz")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmp.Name(), cleanup, nil
}

// readIndexAndSignature loads raw index and signature bytes from source.
func readIndexAndSignature(source string, transport http.RoundTripper) ([]byte, []byte, error) {
	if isHTTPURL(source) {
		index, err := httpGetBytes(joinSourceURL(source, indexName), transport)
		if err != nil {
			return nil, nil, err
		}
		signature, err := httpGetBytes(joinSourceURL(source, signatureName), transport)
		if err != nil {
			return nil, nil, fmt.Errorf("read release index signature: %w", err)
		}
		return index, signature, nil
	}

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
		return nil, nil, fmt.Errorf("read release index signature: %w", err)
	}
	return index, signature, nil
}

// httpGetBytes fetches an HTTP URL and returns successful response bytes.
func httpGetBytes(rawURL string, transport http.RoundTripper) ([]byte, error) {
	if transport == nil {
		transport = http.DefaultTransport
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s returned %s", rawURL, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// joinSourceURL appends name to the path portion of a release source URL.
func joinSourceURL(root, name string) string {
	parsed, err := url.Parse(root)
	if err != nil {
		return root + "/" + name
	}
	parsed.Path = stdpath.Join(parsed.Path, name)
	return parsed.String()
}

// isHTTPURL reports whether raw is an HTTP or HTTPS URL.
func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
