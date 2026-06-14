/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/coverprofile"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

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
	filteredProfile := filepath.Join(coverDir, "coverage.out")

	if err := runGo(root, os.Stdout, os.Stderr, "test", "-coverprofile="+rawProfile, "./..."); err != nil {
		return err
	}
	if err := filter(rawProfile, filteredProfile); err != nil {
		return err
	}

	var report bytes.Buffer
	if err := runGo(root, &report, os.Stderr, "tool", "cover", "-func="+filteredProfile); err != nil {
		return err
	}
	fmt.Print(report.String())
	if total := coverprofile.TotalPercent(report.String()); total != "100.0%" {
		return fmt.Errorf("coverage gate failed: got %s, want 100.0%%", total)
	}
	return nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
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

func runGo(root string, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = goEnv(root)
	return cmd.Run()
}

func filter(rawProfile, filteredProfile string) error {
	in, err := os.Open(rawProfile)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(filteredProfile)
	if err != nil {
		return err
	}
	if err := coverprofile.Filter(in, out, coverprofile.DefaultExclusions()); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

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
