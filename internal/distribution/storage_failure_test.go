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
)

// failingArtifact retains a real file while failing selected persistence operations.
type failingArtifact struct {
	*os.File
	writeErr, closeErr error
	closed             bool
}

// Write injects a storage failure without changing the temporary file's identity.
func (file *failingArtifact) Write(data []byte) (int, error) {
	if file.writeErr != nil {
		return 0, file.writeErr
	}
	return file.File.Write(data)
}

// Close releases the real file before reporting a simulated close failure.
func (file *failingArtifact) Close() error {
	file.closed = true
	err := file.File.Close()
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
				file = &failingArtifact{File: temporary}
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
