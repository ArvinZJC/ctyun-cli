/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestFixtureFilterDocumentedSpelling checks exact fields first and uses only an unambiguous declared alternate.
func TestFixtureFilterDocumentedSpelling(t *testing.T) {
	table := plugin.Table{Columns: []plugin.TableColumn{
		{Key: "date", Path: "date"}, {Key: "sample", Path: "date "}, {Key: "other", Path: " date"},
	}}
	parameters := []plugin.Parameter{{Name: "date", Target: "date"}}
	padded := map[string]string{"sample": "today "}
	if rows := filterRowsByParameters([]map[string]string{padded}, table, parameters, map[string]string{"date": "today "}); len(rows) != 1 {
		t.Fatal("an exact value in the alternate field must also match")
	}
	for _, tc := range []struct {
		name string
		row  map[string]string
		want bool
	}{
		{"sample", map[string]string{"sample": "today "}, true},
		{"absent", map[string]string{}, false},
		{"different sample", map[string]string{"sample": "yesterday "}, false},
		{"unrelated key", map[string]string{"another": "today"}, false},
		{"exact wins", map[string]string{"date": "yesterday", "sample": "today "}, false},
		{"exact empty wins", map[string]string{"date": "", "sample": "today "}, false},
		{"exact whitespace is significant", map[string]string{"date": "today "}, false},
		{"ambiguous", map[string]string{"sample": "today ", "other": "today"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := filterRowsByParameters([]map[string]string{tc.row}, table, parameters, map[string]string{"date": "today"})
			if (len(got) == 1) != tc.want {
				t.Fatalf("matched %t, want %t", len(got) == 1, tc.want)
			}
		})
	}
}
