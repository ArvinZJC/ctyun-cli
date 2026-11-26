/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// brokenOutput exposes command output failures after successful maintenance.
type brokenOutput struct{}

// Write reports an unavailable output destination.
func (brokenOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// prepareCatalog writes an independent, valid source catalog for maintenance tests.
func prepareCatalog(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	if err := os.WriteFile(input, []byte(toolCatalogFixtureJSON), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"harvest", "ecs", "--input", input}, root, io.Discard); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestDispatchLifecycle verifies generated evidence, reviewed readiness, and promotion.
func TestDispatchLifecycle(t *testing.T) {
	root := prepareCatalog(t)
	for _, tc := range []struct{ cmd, want, path string }{{"generate", "wrote openapi-catalogs/ecs/draft", "openapi-catalogs/ecs/draft/plugin.json"}, {"review", "review ready for ecs", "openapi-catalogs/ecs/review.md"}, {"promote", "promoted plugins/ecs", "plugins/ecs/plugin.json"}} {
		var out bytes.Buffer
		if err := run([]string{tc.cmd, "ecs"}, root, &out); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("%s output=%q", tc.cmd, out.String())
		}
		if _, err := os.Stat(filepath.Join(root, tc.path)); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDispatchErrors ensures invalid inputs and missing evidence fail without success output.
func TestDispatchErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown", "ecs"}, {"harvest", "ecs", "--unknown"}, {"harvest", "ecs", "--input", "missing.json"}, {"diff", "ecs"}, {"normalize-labels", "ecs"}, {"generate", "ecs"}, {"review", "ecs"}, {"promote", "ecs"}} {
		var out bytes.Buffer
		if err := run(args, t.TempDir(), &out); err == nil || out.Len() != 0 {
			t.Fatalf("%v error=%v output=%q", args, err, out.String())
		}
	}
	root := prepareCatalog(t)
	if err := run([]string{"generate", "ecs"}, root, io.Discard); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "openapi-catalogs/ecs/draft/plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte(`"source_fingerprint": "`), []byte(`"source_fingerprint": "changed`))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"review", "ecs"}, root, io.Discard); err == nil || !strings.Contains(err.Error(), "review blocked") {
		t.Fatalf("review=%v", err)
	}
}

// TestDispatchOutputFailures verifies every command reports output write failures.
func TestDispatchOutputFailures(t *testing.T) {
	for _, command := range []string{"harvest", "diff", "normalize-labels", "generate", "review", "promote"} {
		t.Run(command, func(t *testing.T) {
			root := prepareCatalog(t)
			if err := run([]string{"generate", "ecs"}, root, io.Discard); err != nil {
				t.Fatal(err)
			}
			args := []string{command, "ecs"}
			if command == "harvest" {
				args = append(args, "--input", filepath.Join(root, "input.json"))
			}
			if err := run(args, root, brokenOutput{}); !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

// TestMainProcessBoundary checks successful command output and failed-command exit status.
func TestMainProcessBoundary(t *testing.T) {
	originalExit, originalArgs, originalOut, originalErr := exitProcess, os.Args, os.Stdout, os.Stderr
	t.Cleanup(func() {
		exitProcess, os.Args, os.Stdout, os.Stderr = originalExit, originalArgs, originalOut, originalErr
	})
	root := prepareCatalog(t)
	t.Chdir(root)
	for _, tc := range []struct {
		args   []string
		want   string
		status int
	}{{[]string{"openapi", "generate", "ecs"}, "wrote openapi-catalogs/ecs/draft", -1}, {[]string{"openapi"}, "usage: openapi", 1}} {
		file, err := os.CreateTemp(t.TempDir(), "output")
		if err != nil {
			t.Fatal(err)
		}
		status := -1
		exitProcess = func(code int) { status = code }
		os.Args = tc.args
		os.Stdout, os.Stderr = file, file
		main()
		t.Cleanup(func() { _ = file.Close() })
		data, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if status != tc.status || !strings.Contains(string(data), tc.want) {
			t.Fatalf("status=%d output=%s", status, data)
		}
	}
}
