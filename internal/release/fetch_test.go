/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyIndexSignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	index := []byte(`{"schema":1,"releases":[]}`)
	signature := ed25519.Sign(privateKey, index)
	err = VerifyIndexSignature(index, []byte(base64.StdEncoding.EncodeToString(signature)), base64.StdEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatalf("VerifyIndexSignature returned error: %v", err)
	}
	if err := VerifyIndexSignature(index, []byte("bad"), base64.StdEncoding.EncodeToString(publicKey)); err == nil {
		t.Fatal("VerifyIndexSignature accepted bad signature")
	}
}

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

func TestReadSignedIndexRejectsBadSignature(t *testing.T) {
	root := t.TempDir()
	index := []byte(`{"schema":1,"releases":[]}`)
	publicKey := writeSignedIndex(t, root, index)
	if err := os.WriteFile(filepath.Join(root, "core-index.sig"), []byte("bad-signature"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadSignedIndex(root, publicKey, nil); err == nil || !strings.Contains(err.Error(), "release index signature") {
		t.Fatalf("ReadSignedIndex error = %v, want signature failure", err)
	}
}

func TestVerifySHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.tar.gz")
	if err := os.WriteFile(path, []byte("artifact"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := VerifySHA256(path, "c7c5c1d70c5dec4416ab6158afd0b223ef40c29b1dc1f97ed9428b94d4cadb1c"); err != nil {
		t.Fatalf("VerifySHA256 returned error: %v", err)
	}
	if err := VerifySHA256(path, strings.Repeat("0", 64)); err == nil {
		t.Fatal("VerifySHA256 accepted bad digest")
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
