/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestConfigCompletionOptionsRemainConsistent covers blank and partial option tokens.
func TestConfigCompletionOptionsRemainConsistent(t *testing.T) {
	root := t.TempDir()
	for _, path := range [][]string{
		{"config", "show"},
		{"config", "profile", "set-secret", "demo", "ak"},
		{"config", "profiles", "set-secret", "demo", "sk", "--from-stdin"},
	} {
		t.Run(path[1]+path[len(path)-1], func(t *testing.T) {
			blank := completeArgs(append(append([]string(nil), path...), ""), root)
			partial := completeArgs(append(append([]string(nil), path...), "-"), root)
			assertEqualCompletions(t, blank, partial)
		})
	}
}

// TestCompletionOptionParityAtValidPaths keeps group words and exposes every applicable unused option at either cursor.
func TestCompletionOptionParityAtValidPaths(t *testing.T) {
	root := t.TempDir()
	writeWaitBundle(t, filepath.Join(root, "ecs"))
	for _, path := range [][]string{
		nil, {"completion"}, {"completion", "bash"}, {"version"},
		{"config"}, {"config", "profiles"}, {"doctor"}, {"doctor", "network"},
		{"plugin"}, {"plugins", "list"}, {"help"}, {"help", "doctor"}, {"help", "doctor", "local"},
		{"help", "plugin", "list"}, {"help", "config", "profile", "set"}, {"help", "ecs", "instance", "show"},
		{"ecs"}, {"ecs", "instance"}, {"ecs", "instance", "show", "resource"},
		{"--lang", "en-US", "version"},
	} {
		t.Run(strings.Join(path, "_"), func(t *testing.T) {
			all := completeArgs(append(slices.Clone(path), ""), root)
			blank := slices.DeleteFunc(all, func(value string) bool { return !strings.HasPrefix(value, "-") })
			partial := completeArgs(append(slices.Clone(path), "-"), root)
			assertHasCompletions(t, partial, "--help")
			assertEqualCompletions(t, blank, partial)
		})
	}
	for _, path := range [][]string{
		{"version", "extra"}, {"completion", "unknown"}, {"config", "absent"},
		{"help", "plugin", "unknown"}, {"help", "doctor", "network", "bad"}, {"ecs", "wrong"},
	} {
		for _, prefix := range []string{"", "-"} {
			assertEqualCompletions(t, completeArgs(append(slices.Clone(path), prefix), root), nil)
		}
	}
}
