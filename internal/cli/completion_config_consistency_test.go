/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import "testing"

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
