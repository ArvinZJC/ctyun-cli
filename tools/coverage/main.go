/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/coverprofile"
)

// exitProcess is the process boundary used to verify command failure status.
var exitProcess = os.Exit

// executeGo is the command boundary used to test the gate without recursively
// starting the repository test suite.
var executeGo = runGo

// workingDirectory allows repository discovery failures to be reported and tested.
var workingDirectory = os.Getwd

// main runs the repository coverage gate.
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		exitProcess(1)
	}
}

// run tests the complete repository and requires every statement in the raw
// coverage profile to execute, without exclusions or percentage rounding.
func run() error {
	root, err := repoRoot()
	if err != nil {
		return err
	}
	coverDir := filepath.Join(root, ".cache", "coverage")
	if err := os.MkdirAll(coverDir, 0o755); err != nil {
		return err
	}
	rawProfile := filepath.Join(coverDir, "raw.out")
	if err := executeGo(root, os.Stdout, os.Stderr, "test", "-timeout=30m", "-coverprofile="+rawProfile, "./..."); err != nil {
		return err
	}
	if err := executeGo(root, os.Stdout, os.Stderr, "tool", "cover", "-func="+rawProfile); err != nil {
		return err
	}
	data, err := os.ReadFile(rawProfile)
	if err != nil {
		return err
	}
	return coverprofile.CheckFullCoverage(strings.NewReader(string(data)))
}

// repoRoot walks upward from the current directory until it finds go.mod.
func repoRoot() (string, error) {
	dir, err := workingDirectory()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod")
		}
		dir = parent
	}
}

// runGo executes a go command from the repository root.
func runGo(root string, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = goEnv(root)
	return cmd.Run()
}

// goEnv ensures Go commands have a writable repository-local build cache.
func goEnv(root string) []string {
	env := os.Environ()
	hasGoCache := false
	for _, item := range env {
		if strings.HasPrefix(item, "GOCACHE=") {
			hasGoCache = true
		}
	}
	if !hasGoCache {
		env = append(env, "GOCACHE="+filepath.Join(root, ".cache", "go-build"))
	}
	return env
}
