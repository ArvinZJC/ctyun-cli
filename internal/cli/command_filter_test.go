/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestCommandTablePreservesLiveMatches leaves API filtering to the service while
// retaining local parameter filtering for static fixture responses.
func TestCommandTablePreservesLiveMatches(t *testing.T) {
	bundle := plugin.Bundle{Tables: plugin.Tables{Tables: map[string]plugin.Table{
		"instances": {RowPath: "items", Columns: []plugin.TableColumn{{Key: "name", Path: "name", Labels: map[string]string{"en-US": "Name"}}}},
	}}}
	command := plugin.Command{Table: "instances", Parameters: []plugin.Parameter{{Name: "name", Target: "name"}}}
	payload := map[string]any{"items": []any{map[string]any{"name": "web"}, map[string]any{"name": "web-01"}}}
	for _, fixture := range []bool{false, true} {
		t.Run(map[bool]string{false: "live fuzzy matches", true: "fixture exact matches"}[fixture], func(t *testing.T) {
			var out bytes.Buffer
			opts := globalOptions{Output: "table", Language: "en-US", Fixture: fixture}
			err := renderCommandPayload(&out, io.Discard, bundle, command, map[string]string{"name": "web"}, opts, payload, nil, "", config.Profile{}, func(string) string { return "" })
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "web") || strings.Contains(out.String(), "web-01") == fixture {
				t.Fatalf("fixture=%v output=%s", fixture, out.String())
			}
		})
	}
}
