/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package output

import "testing"

// TestFilterValuesPreserveOperators treats operator-like text after the first
// comparison as part of the value for both visible labels and stable keys.
func TestFilterValuesPreserveOperators(t *testing.T) {
	columns := []Column{{Key: "name", Label: "Name"}}
	rows := []map[string]string{{"name": "a!=b"}, {"name": "a=b"}, {"name": "other"}}
	for _, tc := range []struct {
		expression string
		want       []string
	}{
		{"Name=a!=b", []string{"a!=b"}},
		{"Name=a=b", []string{"a=b"}},
		{"Name!=a!=b", []string{"a=b", "other"}},
		{"Name!=a=b", []string{"a!=b", "other"}},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			resolved, err := ResolveFilterExpression(columns, tc.expression)
			if err != nil {
				t.Fatal(err)
			}
			got, err := FilterRows(rows, resolved)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("rows=%v want=%v", got, tc.want)
			}
			for i, want := range tc.want {
				if got[i]["name"] != want {
					t.Fatalf("row=%v want=%q", got[i], want)
				}
			}
		})
	}
}
