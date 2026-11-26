/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package distribution

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"testing/iotest"
)

// failingArtifact retains a real file while failing selected persistence operations.
type failingArtifact struct {
	artifactFile
	writeErr, closeErr error
	closed             bool
}

// TestDownloadPartialResponse checks that a first-chunk disk failure stops subsequent network reads.
func TestDownloadPartialResponse(t *testing.T) {
	original := createArtifactFile
	t.Cleanup(func() { createArtifactFile = original })
	responseFailure := errors.New("interrupted response")
	writeFailure := errors.New("disk full")
	var staged *failingArtifact
	createArtifactFile = func() (artifactFile, error) {
		file, err := os.CreateTemp(t.TempDir(), "partial-*")
		if err != nil {
			return nil, err
		}
		staged = &failingArtifact{artifactFile: file, writeErr: writeFailure}
		return staged, nil
	}
	body := &trackingReadCloser{Reader: io.MultiReader(strings.NewReader("first chunk"), iotest.ErrReader(responseFailure))}
	path, cleanup, err := DownloadArtifact("https://example.test/archive", roundTripFunc(func(*http.Request) (*http.Response, error) {
		return responseWithBody(http.StatusOK, body), nil
	}))
	defer cleanup()
	if !errors.Is(err, writeFailure) || path != "" || !body.Closed {
		t.Errorf("download returned %q, %v, response closed=%t; want write failure on first chunk", path, err, body.Closed)
	}
	if staged == nil || !staged.closed {
		t.Fatal("staging file was never opened and closed")
	}
	if _, err := os.Stat(staged.Name()); !os.IsNotExist(err) {
		t.Errorf("partial file remains: %v", err)
	}
}

// TestDownloadResponseFailureRemovesStaging verifies cleanup after a partial read or a failed response close.
func TestDownloadResponseFailureRemovesStaging(t *testing.T) {
	failure := errors.New("response failed")
	for _, test := range []struct {
		name string
		body io.ReadCloser
	}{
		{"read", io.NopCloser(io.MultiReader(strings.NewReader("prefix"), iotest.ErrReader(failure)))},
		{"close", failingReadCloser{Reader: strings.NewReader("complete"), closeErr: failure}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			original := createArtifactFile
			t.Cleanup(func() { createArtifactFile = original })
			createArtifactFile = func() (artifactFile, error) { return os.CreateTemp(dir, "partial-*") }
			path, cleanup, err := DownloadArtifact("https://example.test/archive", roundTripFunc(func(*http.Request) (*http.Response, error) {
				return responseWithBody(http.StatusOK, test.body), nil
			}))
			defer cleanup()
			if !errors.Is(err, failure) || path != "" {
				t.Fatalf("returned %q, %v", path, err)
			}
			if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
				t.Fatalf("remaining staging files: %v, %v", entries, err)
			}
		})
	}
}

// Write injects a storage failure without changing the temporary file's identity.
func (file *failingArtifact) Write(data []byte) (int, error) {
	if file.writeErr != nil {
		return 0, file.writeErr
	}
	return file.artifactFile.Write(data)
}

// Close releases the real file before reporting a simulated close failure.
func (file *failingArtifact) Close() error {
	file.closed = true
	err := file.artifactFile.Close()
	if file.closeErr != nil {
		return file.closeErr
	}
	return err
}

// TestDownloadStorageFailures checks propagation and removal of incomplete artifacts.
func TestDownloadStorageFailures(t *testing.T) {
	original := createArtifactFile
	t.Cleanup(func() { createArtifactFile = original })
	failure := errors.New("storage failure")
	for _, stage := range []string{"create", "write", "close"} {
		t.Run(stage, func(t *testing.T) {
			var file *failingArtifact
			createArtifactFile = func() (artifactFile, error) {
				if stage == "create" {
					return nil, failure
				}
				temporary, err := os.CreateTemp(t.TempDir(), "artifact-*")
				if err != nil {
					return nil, err
				}
				file = &failingArtifact{artifactFile: temporary}
				if stage == "write" {
					file.writeErr = failure
				} else {
					file.closeErr = failure
				}
				return file, nil
			}
			path, cleanup, err := DownloadArtifact("https://example.test/archive", roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("archive"))}, nil
			}))
			cleanup()
			if !errors.Is(err, failure) || path != "" {
				t.Fatalf("download=%q, %v", path, err)
			}
			if file != nil {
				if !file.closed {
					t.Error("file not closed")
				}
				if _, err := os.Stat(file.Name()); !os.IsNotExist(err) {
					t.Errorf("partial artifact remains: %v", err)
				}
			}
		})
	}
}
