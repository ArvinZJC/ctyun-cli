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
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	body, err := PrepareBody(BodyInput{Encoding: "file", Document: path})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		reader, err := body.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || string(got) != "original" {
			t.Fatalf("%q %v", got, err)
		}
	}
}
