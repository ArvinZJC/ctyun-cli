/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestPreparedFileSnapshotsSource pins stable signing and replay after source mutation.
func TestPreparedFileSnapshotsSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	body := prepareTestFileBody(t, path, []byte("original"))
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		reader, err := body.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(reader)
		if closeErr := reader.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		if err != nil || string(got) != "original" {
			t.Fatalf("%q %v", got, err)
		}
	}
}

// prepareTestBody registers snapshot cleanup before a test can fail.
func prepareTestBody(t *testing.T, input BodyInput) *PreparedBody {
	t.Helper()
	body, err := PrepareBody(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := body.Close(); err != nil {
			t.Error(err)
		}
	})
	return body
}

// prepareTestFileBody snapshots controlled input while retaining its source for mutation tests.
func prepareTestFileBody(t *testing.T, path string, data []byte) *PreparedBody {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return prepareTestBody(t, BodyInput{Encoding: "file", Document: path})
}

// readPreparedTestBody verifies both reading and closing a fresh snapshot reader.
func readPreparedTestBody(t *testing.T, body *PreparedBody) []byte {
	t.Helper()
	reader, err := body.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	}()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
