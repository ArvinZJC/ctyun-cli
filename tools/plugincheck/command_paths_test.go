/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestPluginCommandLeavesDoNotShadowChildren rejects child commands that the
// parser cannot reach after resolving an earlier path as a complete command.
func TestPluginCommandLeavesDoNotShadowChildren(t *testing.T) {
	for _, dir := range pluginDirs(t, repoPath(t, "plugins")) {
		b, err := plugin.LoadBundle(dir, version.Version)
		if err != nil {
			t.Fatal(err)
		}
		leaves := make(map[string]bool, len(b.Commands.Commands))
		for _, c := range b.Commands.Commands {
			leaves[strings.Join(c.Path, " ")] = true
		}
		for _, c := range b.Commands.Commands {
			for i := 1; i < len(c.Path); i++ {
				prefix := strings.Join(c.Path[:i], " ")
				if leaves[prefix] {
					t.Errorf("command %q is shadowed by leaf %q", strings.Join(c.Path, " "), prefix)
				}
			}
		}
	}
}
