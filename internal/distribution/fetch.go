/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package distribution

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	stdpath "path"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// ErrUnsafeRedirect reports an unsafe or excessive HTTP redirect chain.
var ErrUnsafeRedirect = errors.New("unsafe or excessive redirect")

// Artifact describes the shared fields required to fetch and verify an update
// artifact.
type Artifact struct {
	Name   string
	URL    string
	SHA256 string
}

// ReadSignedIndex fetches an index and detached signature from a hosted source.
func ReadSignedIndex(source, indexName, signatureName, publicKey, subject string, transport http.RoundTripper) ([]byte, error) {
	index, err := HTTPGetBytes(JoinURL(source, indexName), transport)
	if err != nil {
		return nil, diagnostic.Wrap("error.read_index", err, subject)
	}
	signature, err := HTTPGetBytes(JoinURL(source, signatureName), transport)
	if err != nil {
		return nil, diagnostic.Wrap("error.read_index_signature", err, subject)
	}
	if err := VerifyIndexSignature(index, signature, publicKey, subject); err != nil {
		return nil, diagnostic.Wrap("error.index_signature", err, subject)
	}
	return index, nil
}

// ReadSignedIndexWithFallbacks reads the first source with a valid signed index.
func ReadSignedIndexWithFallbacks(source Source, indexName, signatureName, publicKey, subject string, transport http.RoundTripper) (Source, []byte, error) {
	sources := append([]Source{source}, source.Fallbacks...)
	var lastErr error
	for _, candidate := range sources {
		indexBytes, err := ReadSignedIndex(candidate.URL, indexName, signatureName, publicKey, subject, transport)
		if err == nil {
			return candidate, indexBytes, nil
		}
		lastErr = err
	}
	return Source{}, nil, lastErr
}

// VerifyIndexSignature verifies an index with a trusted base64 Ed25519 public
// key and detached base64 signature.
func VerifyIndexSignature(index, signature []byte, publicKey, subject string) error {
	if publicKey == "" {
		return diagnostic.New("error.index_public_key_required", subject)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return diagnostic.Wrap("error.decode_public_key", err, subject)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return diagnostic.New("error.public_key_length", subject, len(keyBytes), ed25519.PublicKeySize)
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil {
		return diagnostic.Wrap("error.decode_signature", err, subject)
	}
	if !ed25519.Verify(keyBytes, index, signatureBytes) {
		return diagnostic.New("error.signature_failed", subject)
	}
	return nil
}

// PrepareArtifact downloads a hosted artifact and returns a cleanup callback.
func PrepareArtifact(source string, artifact Artifact, transport http.RoundTripper) (string, func(), error) {
	if artifact.SHA256 == "" {
		return "", func() {}, diagnostic.New("error.artifact_requires_sha256", artifact.Name)
	}
	artifactURL := artifact.URL
	if !IsHTTPURL(artifactURL) {
		artifactURL = JoinURL(source, artifactURL)
	}
	return DownloadArtifact(artifactURL, transport)
}

// DownloadArtifact downloads a hosted artifact into a temporary file.
func DownloadArtifact(rawURL string, transport http.RoundTripper) (string, func(), error) {
	data, err := HTTPGetBytes(rawURL, transport)
	if err != nil {
		return "", func() {}, err
	}
	file, err := os.CreateTemp("", "ctyun-artifact-*.tar.gz")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.Remove(file.Name()) }
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return file.Name(), cleanup, nil
}

// VerifySHA256 checks that path matches the expected lowercase hex digest.
func VerifySHA256(path, want string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		err = closeWithError(err, file.Close)
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != want {
		return diagnostic.New("error.sha256_mismatch", path, got, want)
	}
	return nil
}

// HTTPGetBytesContext reads one HTTP(S) URL with a context and optional byte
// limit and requires a 2xx response.
func HTTPGetBytesContext(ctx context.Context, rawURL string, transport http.RoundTripper, maxBytes int64) (data []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: transport}
	if transport == nil {
		client.Transport = http.DefaultTransport
	}
	client.CheckRedirect = func(next *http.Request, previous []*http.Request) error {
		if len(previous) > 0 && previous[0].URL.Scheme == "https" && next.URL.Scheme != "https" {
			return ErrUnsafeRedirect
		}
		if len(previous) >= 10 {
			return ErrUnsafeRedirect
		}
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = closeWithError(err, resp.Body.Close)
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, diagnostic.New("error.http_get_status", rawURL, resp.Status)
	}
	if maxBytes <= 0 {
		return io.ReadAll(resp.Body)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err == nil && int64(len(data)) > maxBytes {
		return nil, diagnostic.New("error.http_response_too_large", rawURL, maxBytes)
	}
	return data, err
}

// HTTPGetBytes reads one HTTP(S) URL and requires a 2xx response.
func HTTPGetBytes(rawURL string, transport http.RoundTripper) ([]byte, error) {
	return HTTPGetBytesContext(context.Background(), rawURL, transport, 0)
}

// JoinURL appends name to the path portion of a hosted source URL.
func JoinURL(root string, name string) string {
	parsed, err := url.Parse(root)
	if err != nil || parsed.Scheme == "" {
		return strings.TrimRight(root, "/") + "/" + name
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/" + strings.TrimLeft(name, "/")
	} else {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(name, "/")
	}
	return parsed.String()
}

// IsHTTPURL reports whether raw is an HTTP or HTTPS URL.
func IsHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// SafeRelativePath reports whether raw is a safe hosted artifact path.
func SafeRelativePath(raw string) bool {
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "\\") {
		return false
	}
	if strings.Contains(raw, "\\") {
		return false
	}
	clean := stdpath.Clean(raw)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

// ValidArtifactURL reports whether raw is an HTTP(S) URL or safe relative path.
func ValidArtifactURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Scheme != "" {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}
	return SafeRelativePath(raw)
}

// ArtifactBase returns a useful fallback name for checksum messages.
func ArtifactBase(raw string) string {
	base := filepath.Base(raw)
	if base == "." || base == string(filepath.Separator) {
		return raw
	}
	return base
}

// closeWithError preserves the primary error unless closing also fails.
func closeWithError(err error, close func() error) error {
	closeErr := close()
	if closeErr == nil {
		return err
	}
	if err == nil {
		return closeErr
	}
	return errors.Join(err, closeErr)
}
