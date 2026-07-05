/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/output"
)

// TestListPluginsPropagatesHelperErrors covers helper failures in plugin lists.
func TestListPluginsPropagatesHelperErrors(t *testing.T) {
	t.Run("render", func(t *testing.T) {
		originalRenderOutputTable := renderOutputTable
		t.Cleanup(func() { renderOutputTable = originalRenderOutputTable })
		renderOutputTable = func([]map[string]string, []output.Column, output.TableOptions) (string, error) {
			return "", errors.New("render failed")
		}
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "table", Language: "en-US"})
		if err == nil || !strings.Contains(err.Error(), "render failed") {
			t.Fatalf("listPlugins render error = %v, want render failed", err)
		}
	})
	t.Run("json", func(t *testing.T) {
		originalRenderOutputJSON := renderOutputJSON
		t.Cleanup(func() { renderOutputJSON = originalRenderOutputJSON })
		renderOutputJSON = func(any) (string, error) {
			return "", errors.New("json failed")
		}
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "json", Language: "en-US"})
		if err == nil || !strings.Contains(err.Error(), "json failed") {
			t.Fatalf("listPlugins json error = %v, want json failed", err)
		}
	})
	t.Run("output", func(t *testing.T) {
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "yaml", Language: "en-US"})
		if err == nil {
			t.Fatalf("listPlugins output error = %v, want unsupported output", err)
		}
		requireDiagnosticKey(t, err, "error.unsupported_output")
	})
	t.Run("stat", func(t *testing.T) {
		originalOSStat := osStat
		t.Cleanup(func() { osStat = originalOSStat })
		osStat = func(string) (os.FileInfo, error) {
			return nil, errors.New("stat failed")
		}
		err := listPlugins(io.Discard, t.TempDir(), globalOptions{Output: "table", Language: "en-US"})
		if err == nil || !strings.Contains(err.Error(), "stat failed") {
			t.Fatalf("listPlugins stat error = %v, want stat failed", err)
		}
	})
}
