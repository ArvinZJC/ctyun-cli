/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package distribution

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

type diagnosticError interface {
	MessageKey() string
	MessageArgs() []any
}

func requireDiagnosticKey(t *testing.T, err error, key string) {
	t.Helper()
	var diagnosticErr diagnosticError
	if !errors.As(err, &diagnosticErr) {
		t.Fatalf("error %T is not a diagnostic error: %v", err, err)
	}
	if diagnosticErr.MessageKey() != key {
		t.Fatalf("diagnostic key = %q, want %q", diagnosticErr.MessageKey(), key)
	}
}

func TestFetchErrorsUseDiagnosticKeys(t *testing.T) {
	if err := VerifyIndexSignature([]byte("index"), []byte("sig"), "", "registry"); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error without public key")
	} else {
		requireDiagnosticKey(t, err, "error.index_public_key_required")
	}

	if _, _, err := PrepareArtifact("https://example.test/plugins", Artifact{Name: "ecs"}, nil); err == nil {
		t.Fatal("PrepareArtifact returned nil error without sha256")
	} else {
		requireDiagnosticKey(t, err, "error.artifact_requires_sha256")
	}

	err := diagnostic.New("test.key")
	requireDiagnosticKey(t, err, "test.key")
}

func TestReadSignedIndexUsesDiagnosticKeys(t *testing.T) {
	_, err := ReadSignedIndex("https://registry.example.test", "index.json", "index.sig", "bad-key", "registry", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	}))
	if err == nil {
		t.Fatal("ReadSignedIndex returned nil error for fetch failure")
	}
	requireDiagnosticKey(t, err, "error.read_index")
}

func TestSignedIndexFetchAndFallback(t *testing.T) {
	index := []byte(`{"ok":true}`)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, index))
	source := Source{
		Name:      "github",
		URL:       "https://github.example.test/releases",
		Kind:      SourceReady,
		Fallbacks: []Source{{Name: "gitee", URL: "https://gitee.example.test/releases", Kind: SourceReady}},
	}
	selected, got, err := ReadSignedIndexWithFallbacks(source, "index.json", "index.sig", base64.StdEncoding.EncodeToString(publicKey), "test", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "github.example.test" {
			return stringResponse(http.StatusNotFound, "missing"), nil
		}
		switch filepath.Base(req.URL.Path) {
		case "index.json":
			return bytesResponse(http.StatusOK, index), nil
		case "index.sig":
			return stringResponse(http.StatusOK, signature), nil
		default:
			return stringResponse(http.StatusNotFound, "missing"), nil
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	if selected.Name != "gitee" || string(got) != string(index) {
		t.Fatalf("selected = %#v, index = %q", selected, string(got))
	}

	got, err = ReadSignedIndex(source.Fallbacks[0].URL, "index.json", "index.sig", base64.StdEncoding.EncodeToString(publicKey), "test", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch filepath.Base(req.URL.Path) {
		case "index.json":
			return bytesResponse(http.StatusOK, index), nil
		case "index.sig":
			return stringResponse(http.StatusOK, signature), nil
		default:
			return stringResponse(http.StatusNotFound, "missing"), nil
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(index) {
		t.Fatalf("direct index = %q", string(got))
	}

	if _, err := ReadSignedIndex(source.URL, "index.json", "index.sig", "", "test", nil); err == nil {
		t.Fatal("ReadSignedIndex returned nil error without public key")
	}
	if err := VerifyIndexSignature(index, []byte(signature), "bad-key", "test"); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for malformed key")
	}
	if err := VerifyIndexSignature(index, []byte(signature), base64.StdEncoding.EncodeToString([]byte("short")), "test"); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for short key")
	}
	if err := VerifyIndexSignature(index, []byte("bad-signature"), base64.StdEncoding.EncodeToString(publicKey), "test"); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for malformed signature")
	}
	if err := VerifyIndexSignature(index, []byte(base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))), base64.StdEncoding.EncodeToString(publicKey), "test"); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for wrong signature")
	}
	if err := VerifyIndexSignature(index, []byte(signature), "", "test"); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error without public key")
	}
	if _, err := ReadSignedIndex(source.Fallbacks[0].URL, "index.json", "index.sig", base64.StdEncoding.EncodeToString(publicKey), "test", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if filepath.Base(req.URL.Path) == "index.sig" {
			return nil, errors.New("signature network")
		}
		return bytesResponse(http.StatusOK, index), nil
	})); err == nil {
		t.Fatal("ReadSignedIndex returned nil error for signature fetch failure")
	}
	if _, err := ReadSignedIndex(source.Fallbacks[0].URL, "index.json", "index.sig", base64.StdEncoding.EncodeToString(publicKey), "test", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if filepath.Base(req.URL.Path) == "index.sig" {
			return stringResponse(http.StatusOK, "bad-signature"), nil
		}
		return bytesResponse(http.StatusOK, index), nil
	})); err == nil {
		t.Fatal("ReadSignedIndex returned nil error for signature verification failure")
	}
	if _, _, err := ReadSignedIndexWithFallbacks(Source{Name: "github", URL: source.URL, Kind: SourceReady}, "index.json", "index.sig", base64.StdEncoding.EncodeToString(publicKey), "test", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	})); err == nil {
		t.Fatal("ReadSignedIndexWithFallbacks returned nil error when every source fails")
	}
}

func TestArtifactDownloadAndChecksum(t *testing.T) {
	body := []byte("archive")
	sum := sha256.Sum256(body)
	path, cleanup, err := PrepareArtifact("https://example.test/releases", Artifact{Name: "ecs", URL: "ecs.tar.gz", SHA256: hex.EncodeToString(sum[:])}, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/releases/ecs.tar.gz" {
			t.Fatalf("download path = %s", req.URL.Path)
		}
		return bytesResponse(http.StatusOK, body), nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := VerifySHA256(path, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(path, strings.Repeat("0", 64)); err == nil {
		t.Fatal("VerifySHA256 returned nil error for mismatch")
	}
	if err := VerifySHA256(filepath.Join(t.TempDir(), "missing"), hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("VerifySHA256 returned nil error for missing file")
	}
	if err := VerifySHA256(t.TempDir(), hex.EncodeToString(sum[:])); err == nil {
		t.Fatal("VerifySHA256 returned nil error for unreadable directory")
	}
	if _, _, err := PrepareArtifact("https://example.test/releases", Artifact{Name: "ecs", URL: "ecs.tar.gz"}, nil); err == nil {
		t.Fatal("PrepareArtifact returned nil error without checksum")
	}
}

func TestHTTPAndURLHelpers(t *testing.T) {
	if got := JoinURL("https://example.test/root/", "/index.json"); got != "https://example.test/root/index.json" {
		t.Fatalf("JoinURL = %q", got)
	}
	if got := JoinURL("https://example.test", "index.json"); got != "https://example.test/index.json" {
		t.Fatalf("JoinURL root = %q", got)
	}
	if got := JoinURL("://bad", "index.json"); got != "://bad/index.json" {
		t.Fatalf("JoinURL fallback = %q", got)
	}
	if !IsHTTPURL("https://example.test") || IsHTTPURL("file:///tmp/x") || IsHTTPURL("://bad") {
		t.Fatal("IsHTTPURL result mismatch")
	}
	for _, raw := range []string{"plugin.tar.gz", "nested/plugin.tar.gz"} {
		if !SafeRelativePath(raw) {
			t.Fatalf("SafeRelativePath(%q) = false", raw)
		}
	}
	for _, raw := range []string{"/tmp/plugin.tar.gz", "../plugin.tar.gz", "nested\\plugin.tar.gz", "."} {
		if SafeRelativePath(raw) {
			t.Fatalf("SafeRelativePath(%q) = true", raw)
		}
	}
	for _, raw := range []string{"https://example.test/plugin.tar.gz", "plugin.tar.gz", "nested/plugin.tar.gz"} {
		if !ValidArtifactURL(raw) {
			t.Fatalf("ValidArtifactURL(%q) = false", raw)
		}
	}
	for _, raw := range []string{"file:///tmp/plugin.tar.gz", "/tmp/plugin.tar.gz", "../plugin.tar.gz", "nested\\plugin.tar.gz", "."} {
		if ValidArtifactURL(raw) {
			t.Fatalf("ValidArtifactURL(%q) = true", raw)
		}
	}
	if got := ArtifactBase("nested/plugin.tar.gz"); got != "plugin.tar.gz" {
		t.Fatalf("ArtifactBase = %q", got)
	}
	if got := ArtifactBase(""); got != "" {
		t.Fatalf("ArtifactBase empty = %q", got)
	}

	if _, err := HTTPGetBytes("://bad", nil); err == nil {
		t.Fatal("HTTPGetBytes returned nil error for bad URL")
	}
	if _, _, err := DownloadArtifact("://bad", nil); err == nil {
		t.Fatal("DownloadArtifact returned nil error for bad URL")
	}
	if _, err := HTTPGetBytes("https://example.test/index.json", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network")
	})); err == nil {
		t.Fatal("HTTPGetBytes returned nil error for network error")
	}
	if _, err := HTTPGetBytes("https://example.test/index.json", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return stringResponse(http.StatusNotFound, "missing"), nil
	})); err == nil {
		t.Fatal("HTTPGetBytes returned nil error for 404")
	}

	closeErr := errors.New("close response body")
	if _, err := HTTPGetBytes("https://example.test/index.json", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return responseWithBody(http.StatusOK, failingReadCloser{Reader: strings.NewReader("index"), closeErr: closeErr}), nil
	})); !errors.Is(err, closeErr) {
		t.Fatalf("HTTPGetBytes close error = %v, want %v", err, closeErr)
	}
	if _, err := HTTPGetBytes("https://example.test/index.json", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return responseWithBody(http.StatusNotFound, failingReadCloser{Reader: strings.NewReader("missing"), closeErr: closeErr}), nil
	})); err == nil {
		t.Fatal("HTTPGetBytes returned nil error for 404 with close error")
	} else {
		requireDiagnosticKey(t, err, "error.http_get_status")
		if !errors.Is(err, closeErr) {
			t.Fatalf("HTTPGetBytes joined error = %v, want close error", err)
		}
	}
}

func TestDownloadArtifactCleansUp(t *testing.T) {
	path, cleanup, err := DownloadArtifact("https://example.test/plugin.tar.gz", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return stringResponse(http.StatusOK, "archive"), nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("downloaded path missing: %v", err)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup stat error = %v, want not exist", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func stringResponse(status int, body string) *http.Response {
	return bytesResponse(status, []byte(body))
}

func bytesResponse(status int, body []byte) *http.Response {
	return responseWithBody(status, io.NopCloser(strings.NewReader(string(body))))
}

func responseWithBody(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: body}
}

type failingReadCloser struct {
	*strings.Reader
	closeErr error
}

func (rc failingReadCloser) Close() error {
	return rc.closeErr
}
