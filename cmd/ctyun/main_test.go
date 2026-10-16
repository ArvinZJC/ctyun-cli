/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"os"
	"strings"
	"testing"
)

// TestMainExitStatus verifies that the CLI preserves exit status and separates
// successful output on stdout from failure diagnostics on stderr.
func TestMainExitStatus(t *testing.T) {
	originalExit, originalArgs, originalOut, originalErr := exitProcess, os.Args, os.Stdout, os.Stderr
	t.Cleanup(func() {
		exitProcess, os.Args, os.Stdout, os.Stderr = originalExit, originalArgs, originalOut, originalErr
	})
	for _, tc := range []struct {
		name           string
		args           []string
		status         int
		stdout, stderr string
	}{
		{"version", []string{"ctyun", "--version"}, 0, "ctyun", ""},
		{"invalid timeout", []string{"ctyun", "--timeout", "invalid"}, 1, "", "--timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			streams := []*os.File{nil, nil}
			for index := range streams {
				file, err := os.CreateTemp(t.TempDir(), "stream")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = file.Close() })
				streams[index] = file
			}
			os.Stdout, os.Stderr = streams[0], streams[1]
			status := -1
			exitProcess = func(code int) { status = code }
			os.Args = tc.args
			main()
			if status != tc.status {
				t.Fatalf("exit status=%d want=%d", status, tc.status)
			}
			for index, want := range []string{tc.stdout, tc.stderr} {
				data, err := os.ReadFile(streams[index].Name())
				if err != nil {
					t.Fatal(err)
				}
				if want == "" && len(data) != 0 || want != "" && !strings.Contains(string(data), want) {
					t.Errorf("stream %d=%q want=%q", index, data, want)
				}
			}
		})
	}
}
