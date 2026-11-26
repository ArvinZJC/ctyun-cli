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
	"slices"
	"strings"
	"testing"
)

// TestCoverageMainReportsFailures verifies the process status and diagnostic
// produced by the real gate, while keeping the enclosing test process alive.
func TestCoverageMainReportsFailures(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			root := coverageWorkspace(t)
			oldExecute, oldExit, oldStderr := executeGo, exitProcess, os.Stderr
			t.Cleanup(func() { executeGo, exitProcess, os.Stderr = oldExecute, oldExit, oldStderr })
			diagnostic, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = diagnostic.Close() })
			os.Stderr = diagnostic
			status := 0
			exitProcess = func(code int) { status = code }
			executeGo = func(_ string, _, _ io.Writer, args ...string) error {
				if fail {
					return errors.New("test suite failed")
				}
				if args[0] == "test" {
					return os.WriteFile(filepath.Join(root, ".cache", "coverage", "raw.out"), []byte("mode: set\nf.go:1.1,2.1 1 1\n"), 0o600)
				}
				return nil
			}
			main()
			message, err := os.ReadFile(diagnostic.Name())
			if err != nil {
				t.Fatal(err)
			}
			if fail && (status != 1 || !strings.Contains(string(message), "test suite failed")) || !fail && (status != 0 || len(message) != 0) {
				t.Fatalf("exit=%d stderr=%q", status, message)
			}
		})
	}
}

// TestRepositoryDiscovery verifies upward lookup and rejects running the gate
// outside a repository or after its working directory has disappeared.
func TestRepositoryDiscovery(t *testing.T) {
	t.Run("parent", func(t *testing.T) {
		root := coverageWorkspace(t)
		nested := filepath.Join(root, "nested")
		if err := os.Mkdir(nested, 0o700); err != nil {
			t.Fatal(err)
		}
		t.Chdir(nested)
		got, err := repoRoot()
		if err != nil {
			t.Fatal(err)
		}
		if got != root {
			t.Fatalf("root=%q want=%q", got, root)
		}
	})
	t.Run("missing", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := run(); err == nil || !strings.Contains(err.Error(), "could not find go.mod") {
			t.Fatalf("missing root: %v", err)
		}
	})
	t.Run("unavailable working directory", func(t *testing.T) {
		old := workingDirectory
		t.Cleanup(func() { workingDirectory = old })
		want := errors.New("working directory unavailable")
		workingDirectory = func() (string, error) { return "", want }
		if err := run(); !errors.Is(err, want) {
			t.Fatalf("directory failure = %v, want %v", err, want)
		}
	})
}

// TestRunGoUsesWorkspaceAndCache exercises the real child process, checking
// repository selection, writable-cache defaults, overrides, and failure output.
func TestRunGoUsesWorkspaceAndCache(t *testing.T) {
	root := coverageWorkspace(t)
	t.Setenv("GOCACHE", "")
	if err := os.Unsetenv("GOCACHE"); err != nil {
		t.Fatal(err)
	}
	for index, cache := range []string{filepath.Join(root, ".cache", "go-build"), t.TempDir()} {
		if index > 0 {
			if err := os.Setenv("GOCACHE", cache); err != nil {
				t.Fatal(err)
			}
		}
		var stdout, stderr bytes.Buffer
		if err := runGo(root, &stdout, &stderr, "env", "GOMOD", "GOCACHE"); err != nil {
			t.Fatalf("go env: %v: %s", err, &stderr)
		}
		want := filepath.Join(root, "go.mod") + "\n" + cache + "\n"
		if stdout.String() != want {
			t.Fatalf("child environment=%q want=%q", stdout.String(), want)
		}
	}
	var stdout, stderr bytes.Buffer
	if err := runGo(root, &stdout, &stderr, "not-a-go-command"); err == nil || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("command failure=%v stderr=%q", err, stderr.String())
	}
}

// TestCoverageGateUsesRawCounters prevents the gate from hiding missed
// statements, accepting a rounded percentage, or masking command failures.
func TestCoverageGateUsesRawCounters(t *testing.T) {
	for _, tc := range []struct {
		name, profile, want          string
		commandFailure               string
		missingProfile, badDirectory bool
	}{
		{name: "covered", profile: "mode: set\nf.go:1.1,2.1 1 1\n"},
		{name: "uncovered", profile: "mode: set\nf.go:1.1,2.1 99999 1\nmiss.go:1.1,2.1 1 0\n", want: "miss.go"},
		{name: "test failure", commandFailure: "test", want: "command failed"},
		{name: "report failure", commandFailure: "tool", want: "command failed"},
		{name: "missing evidence", missingProfile: true, want: "raw.out"},
		{name: "bad directory", badDirectory: true, want: ".cache"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := coverageWorkspace(t)
			if tc.badDirectory {
				if err := os.WriteFile(filepath.Join(root, ".cache"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			old := executeGo
			t.Cleanup(func() { executeGo = old })
			executeGo = func(dir string, _, _ io.Writer, args ...string) error {
				if dir != root {
					t.Fatalf("command escaped workspace: %q", dir)
				}
				if args[0] == tc.commandFailure {
					return errors.New("command failed")
				}
				profilePath := filepath.Join(root, ".cache", "coverage", "raw.out")
				if args[0] == "test" {
					if !slices.Equal(args, []string{"test", "-timeout=30m", "-coverprofile=" + profilePath, "./..."}) {
						t.Fatalf("incomplete test command: %v", args)
					}
					if tc.missingProfile {
						return nil
					}
					return os.WriteFile(profilePath, []byte(tc.profile), 0o600)
				}
				if !slices.Equal(args, []string{"tool", "cover", "-func=" + profilePath}) {
					t.Fatalf("report did not use raw profile: %v", args)
				}
				return nil
			}
			err := run()
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("run error = %v, want %q", err, tc.want)
			}
		})
	}
}

// coverageWorkspace provides a real isolated repository root to gate tests.
func coverageWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/coverage\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}
