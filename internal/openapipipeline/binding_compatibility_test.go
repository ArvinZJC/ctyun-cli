/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"slices"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestBindingIsolationCoreRequirement prevents generated bundles from claiming
// compatibility with cores that leak or overwrite their explicitly bound inputs.
func TestBindingIsolationCoreRequirement(t *testing.T) {
	for _, tc := range []struct {
		name       string
		parameters []Parameter
		request    *apicontract.Request
		want       bool
	}{
		{"distinct locations", []Parameter{{Name: "id", Location: "header", CLIName: "header_id"}, {Name: "id", Location: "body", CLIName: "body_id"}}, nil, true},
		{"header and query identity", []Parameter{{Name: "id", Location: "header", CLIName: "header_id"}, {Name: "id", Location: "query", CLIName: "query_id"}}, nil, true},
		{"header leaks into body", []Parameter{{Name: "token", Location: "header", CLIName: "token"}, {Name: "id", Location: "body", CLIName: "id"}}, nil, true},
		{"typed alias", []Parameter{{Name: "ids", Location: "query", CLIName: "ids", TableTarget: "display_ids"}}, nil, true},
		{"shared region", []Parameter{{Name: "regionID", Location: "header", Profile: "region"}, {Name: "regionID", Location: "body", Profile: "region"}}, nil, false},
		{"body only", []Parameter{{Name: "id", Location: "body", CLIName: "id"}}, nil, false},
		{"header only", []Parameter{{Name: "id", Location: "header", CLIName: "id"}}, nil, false},
		{"constant header", []Parameter{{Name: "token", Location: "header", Constant: "fixed"}, {Name: "id", Location: "body", CLIName: "id"}}, nil, false},
		{"explicit body encoder", []Parameter{{Name: "token", Location: "header", CLIName: "token"}, {Name: "id", Location: "body", CLIName: "id"}}, &apicontract.Request{Encoding: "json"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := Catalog{Operations: []Operation{{Parameters: tc.parameters, Request: tc.request}}}
			if got := catalogNeedsBindingIsolation(catalog); got != tc.want {
				t.Fatalf("requires isolation=%v want %v", got, tc.want)
			}
			if tc.want && generatedCoreRequirement(catalog) != ">=0.5.1 <1.0.0" {
				t.Fatal("inaccurate core requirement")
			}
		})
	}
}

// TestMinimumCoreRequirementPreservesStricterBounds keeps generated compatibility
// ranges stable while replacing an obsolete lower bound.
func TestMinimumCoreRequirementPreservesStricterBounds(t *testing.T) {
	for _, tc := range []struct{ before, minimum, want string }{
		{">=0.4.0 <1.0.0", "0.5.1", ">=0.5.1 <1.0.0"},
		{">=0.6.0 <1.0.0", "0.5.1", ">=0.6.0 <1.0.0"},
		{"<1.0.0", "0.5.1", ">=0.5.1 <1.0.0"},
	} {
		if got := minimumCoreRequirement(tc.before, tc.minimum); got != tc.want {
			t.Errorf("%q => %q want %q", tc.before, got, tc.want)
		}
	}
}

// TestReviewRejectsObsoleteBindingCompatibility prevents an edited manifest from
// bypassing the generated minimum for separately bound query and body inputs.
func TestReviewRejectsObsoleteBindingCompatibility(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := loadCatalogFixture(t)
	if !catalogNeedsBindingIsolation(catalog) {
		t.Fatal("fixture no longer exercises binding isolation")
	}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatal(err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatal(err)
	}
	path := workspace.ProductPath("ecs", "draft", "plugin.json")
	manifest, err := readDraftJSON[plugin.Manifest](path)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Requires.Ctyun = ">=0.5.0 <1.0.0"
	if err := writeJSON(path, manifest); err != nil {
		t.Fatal(err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || !slices.Contains(report.Findings, "independent request bindings require core >=0.5.1") {
		t.Fatalf("obsolete compatibility not rejected: %#v", report)
	}
}
