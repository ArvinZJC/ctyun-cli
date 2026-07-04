/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

func TestReadSignedIndexReadsLocalDirectory(t *testing.T) {
	root := t.TempDir()
	index := []byte(`{"schema":1,"releases":[]}`)
	publicKey := writeSignedIndex(t, root, index)

	got, err := ReadSignedIndex(root, publicKey, nil)
	if err != nil {
		t.Fatalf("ReadSignedIndex returned error: %v", err)
	}
	if string(got) != string(index) {
		t.Fatalf("index = %s, want %s", got, index)
	}
}

func TestReadSignedIndexReadsHTTPRoot(t *testing.T) {
	index := []byte(`{"schema":1,"releases":[]}`)
	publicKey, signature := signIndex(t, index)
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/releases/core-index.json":
			return stringResponse(http.StatusOK, string(index)), nil
		case "/releases/core-index.sig":
			return stringResponse(http.StatusOK, signature), nil
		default:
			return stringResponse(http.StatusNotFound, ""), nil
		}
	})

	got, err := ReadSignedIndex("https://example.test/releases", publicKey, transport)
	if err != nil {
		t.Fatalf("ReadSignedIndex returned error: %v", err)
	}
	if string(got) != string(index) {
		t.Fatalf("index = %s, want %s", got, index)
	}
}

func TestReadSignedIndexReadsLocalFileAndRejectsMissingSignature(t *testing.T) {
	root := t.TempDir()
	index := []byte(`{"schema":1,"releases":[]}`)
	publicKey := writeSignedIndex(t, root, index)
	got, err := ReadSignedIndex(filepath.Join(root, "core-index.json"), publicKey, nil)
	if err != nil {
		t.Fatalf("ReadSignedIndex file returned error: %v", err)
	}
	if string(got) != string(index) {
		t.Fatalf("index = %s, want %s", got, index)
	}
	if err := os.Remove(filepath.Join(root, "core-index.sig")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSignedIndex(root, publicKey, nil); err == nil {
		t.Fatal("ReadSignedIndex returned nil error without signature")
	}
}

func TestReadSignedIndexRejectsBadSignature(t *testing.T) {
	root := t.TempDir()
	index := []byte(`{"schema":1,"releases":[]}`)
	publicKey := writeSignedIndex(t, root, index)
	if err := os.WriteFile(filepath.Join(root, "core-index.sig"), []byte("bad-signature"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadSignedIndex(root, publicKey, nil)
	if err == nil {
		t.Fatal("ReadSignedIndex returned nil error for bad signature")
	}
	requireDiagnosticKey(t, err, "error.index_signature")
}

func TestReadSignedIndexPropagatesReadErrors(t *testing.T) {
	if _, err := ReadSignedIndex(filepath.Join(t.TempDir(), "missing"), "key", nil); err == nil {
		t.Fatal("ReadSignedIndex returned nil error for missing local source")
	}
	if _, err := ReadSignedIndex("https://example.test/releases", "key", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return stringResponse(http.StatusNotFound, ""), nil
	})); err == nil {
		t.Fatal("ReadSignedIndex returned nil error for missing HTTP index")
	}
	index := []byte(`{"schema":1,"releases":[]}`)
	if _, err := ReadSignedIndex("https://example.test/releases", "key", roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if filepath.Base(req.URL.Path) == "core-index.json" {
			return stringResponse(http.StatusOK, string(index)), nil
		}
		return stringResponse(http.StatusNotFound, ""), nil
	})); err == nil {
		t.Fatal("ReadSignedIndex returned nil error for missing HTTP signature")
	}
}

func TestPrepareArtifactResolvesLocalAndDownloadsHTTP(t *testing.T) {
	localPath, cleanup, err := PrepareArtifact("/tmp/releases", Artifact{URL: "ctyun.tar.gz"}, nil)
	if err != nil {
		t.Fatalf("PrepareArtifact local returned error: %v", err)
	}
	defer cleanup()
	if localPath != filepath.Join("/tmp/releases", "ctyun.tar.gz") {
		t.Fatalf("local path = %q", localPath)
	}

	for _, tc := range []struct {
		name   string
		source string
		url    string
		want   string
	}{
		{name: "absolute artifact", source: "/tmp/releases", url: "https://example.test/ctyun.tar.gz", want: "/ctyun.tar.gz"},
		{name: "relative http source", source: "https://example.test/releases", url: "ctyun.tar.gz", want: "/releases/ctyun.tar.gz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, cleanup, err := PrepareArtifact(tc.source, Artifact{URL: tc.url}, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != tc.want {
					t.Fatalf("request path = %q, want %q", req.URL.Path, tc.want)
				}
				return stringResponse(http.StatusOK, "archive"), nil
			}))
			if err != nil {
				t.Fatalf("PrepareArtifact returned error: %v", err)
			}
			defer cleanup()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "archive" {
				t.Fatalf("downloaded data = %q", data)
			}
		})
	}
	if _, _, err := PrepareArtifact("https://example.test/releases", Artifact{URL: "ctyun.tar.gz"}, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return stringResponse(http.StatusNotFound, ""), nil
	})); err == nil {
		t.Fatal("PrepareArtifact returned nil error for failed download")
	}
}

func TestPrepareArtifactDownloadsHTTPWithChecksum(t *testing.T) {
	body := []byte("archive")
	sum := sha256.Sum256(body)
	path, cleanup, err := PrepareArtifact("https://example.test/releases", Artifact{URL: "ctyun.tar.gz", SHA256: hex.EncodeToString(sum[:])}, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/releases/ctyun.tar.gz" {
			t.Fatalf("download path = %s", req.URL.Path)
		}
		return stringResponse(http.StatusOK, string(body)), nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := distribution.VerifySHA256(path, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func stringResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       ioNopCloser{strings.NewReader(body)},
		Header:     make(http.Header),
	}
}

type ioNopCloser struct {
	*strings.Reader
}

func (ioNopCloser) Close() error {
	return nil
}

func writeSignedIndex(t *testing.T, root string, index []byte) string {
	t.Helper()
	publicKey, signature := signIndex(t, index)
	if err := os.WriteFile(filepath.Join(root, "core-index.json"), index, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core-index.sig"), []byte(signature), 0o644); err != nil {
		t.Fatal(err)
	}
	return publicKey
}

func signIndex(t *testing.T, index []byte) (string, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, index)
	return base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(signature)
}
