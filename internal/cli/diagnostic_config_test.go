/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestDiagnosticPathsDoNotWaitForConfigPipe exercises help, path reporting, and scope errors with no config producer.
func TestDiagnosticPathsDoNotWaitForConfigPipe(t *testing.T) {
	const pipeEnv = "CTYUN_TEST_DIAGNOSTIC_PIPE"
	if path := os.Getenv(pipeEnv); path != "" {
		var stderr bytes.Buffer
		for _, extra := range [][]string{nil, {"--lang=en-US"}} {
			args := append([]string{"--config", path, "version", "--output", "json"}, extra...)
			stderr.Reset()
			code := Execute(Config{Args: args, Env: func(string) string { return "" }, Stdout: io.Discard, Stderr: &stderr})
			if code != 1 || !strings.Contains(stderr.String(), "--output") {
				t.Fatalf("scope error = %d, %q", code, stderr.String())
			}
		}
		for _, pathArgs := range [][]string{{"doctor", "local", "--help"}, {"help", "doctor", "local"}, {"config", "path"}} {
			var stdout bytes.Buffer
			stderr.Reset()
			args := append([]string{"--config", path, "--lang", "en-US"}, pathArgs...)
			code := Execute(Config{Args: args, Stdout: &stdout, Stderr: &stderr})
			expected := "Usage:"
			if pathArgs[0] == "config" {
				expected = path
			}
			if code != 0 || !strings.Contains(stdout.String(), expected) || stderr.Len() != 0 {
				t.Fatalf("%v: exit %d, stdout %q stderr %q", pathArgs, code, stdout.String(), stderr.String())
			}
		}
		return
	}
	mkfifo, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("mkfifo is unavailable")
	}
	path := filepath.Join(t.TempDir(), "config.fifo")
	if err := exec.Command(mkfifo, path).Run(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDiagnosticPathsDoNotWaitForConfigPipe$")
	cmd.Env = append(os.Environ(), pipeEnv+"="+path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("reading help, path or scope diagnostics with an unread config pipe: %v (%v), %s", err, ctx.Err(), output)
	}
}

// TestErrorUsesCommandConfigSnapshot verifies that language and redaction survive a selected profile changing during output.
func TestErrorUsesCommandConfigSnapshot(t *testing.T) {
	for _, useLink := range []bool{false, true} {
		dir := t.TempDir()
		path := filepath.Join(dir, "selected.json")
		mustWrite(t, path, `{"profiles":{"work":{"language":"zh-CN","ak":"old-test-key","sk":"old-test-secret"}}}`)
		alias := path
		if useLink {
			alias = filepath.Join(dir, "link.json")
			if err := os.Symlink(path, alias); err != nil {
				t.Logf("cannot create test symlink: %v", err)
				continue
			}
		}
		fallback := filepath.Join(dir, "fallback.json")
		mustWrite(t, fallback, `{"language":"en-US"}`)
		reader, writer := io.Pipe()
		done := make(chan int, 1)
		var stderr bytes.Buffer
		go func() {
			done <- Execute(Config{Args: []string{"--config", alias, "--profile", "work", "version"}, ConfigPath: fallback, Env: func(string) string { return "" }, Stdout: writer, Stderr: &stderr})
		}()
		if _, err := io.ReadFull(reader, make([]byte, 1)); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, path, `{"profiles":{"work":{"language":"en-US","ak":"new-test-key","sk":"new-test-secret"}}}`)
		if err := reader.CloseWithError(errors.New("output failed for old-test-key")); err != nil {
			t.Fatal(err)
		}
		code := <-done
		_ = writer.Close()
		if code != 1 || !strings.Contains(stderr.String(), "错误：") || !strings.Contains(stderr.String(), "[REDACTED]") || strings.Contains(stderr.String(), "old-test-key") {
			t.Errorf("symlink=%t, exit %d, diagnostic %q", useLink, code, stderr.String())
		}
	}
}

// TestInlineConfigKeepsDiagnosticArguments checks reuse of process arguments when the runtime provides spare slice capacity.
func TestInlineConfigKeepsDiagnosticArguments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-config")
	for _, inputs := range [][]string{
		{"--config=" + path, "version", "extra", "--lang", "zh-CN"},
		{"version", "--config=" + path, "--language=zh-CN", "extra"},
	} {
		// A caller may pass a subslice of a larger argument vector and retain the surrounding entries.
		backing := make([]string, 12)
		backing[0], backing[len(inputs)+1] = "before", "after"
		args := backing[1 : 1+len(inputs)]
		copy(args, inputs)
		want := slices.Clone(backing)
		var stderr bytes.Buffer
		code := Execute(Config{Args: args, Env: func(string) string { return "" }, Stdout: io.Discard, Stderr: &stderr})
		if code != 1 || !strings.Contains(stderr.String(), "错误：") || !slices.Equal(backing, want) {
			t.Errorf("exit %d, stderr %q, backing %q; expected %q", code, stderr.String(), backing, want)
		}
	}
}
