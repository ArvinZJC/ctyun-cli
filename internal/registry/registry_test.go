/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package registry

import (
	"strings"
	"testing"

	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

func TestFindArtifactDefaultsToStableReviewedOrCurated(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "generated", "url": "generated.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "reviewed.tar.gz"},
    {"name": "ecs", "version": "0.3.0", "channel": "beta", "quality": "curated", "url": "beta.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	artifact, ok := idx.Find("ecs", "")
	if !ok {
		t.Fatal("Find returned false")
	}
	if artifact.Version != "0.2.0" {
		t.Fatalf("version = %q, want 0.2.0", artifact.Version)
	}
}

func TestLoadIndexRejectsInvalidSemanticVersion(t *testing.T) {
	_, err := LoadIndex([]byte(`{"plugins":[{"name":"ecs","version":"v0.2","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`))
	if err == nil {
		t.Fatal("LoadIndex returned nil error")
	}
	requireDiagnosticKey(t, err, "error.registry_invalid_version")
}

func TestFindArtifactOrdersSemanticVersions(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "old.tar.gz"},
    {"name": "ecs", "version": "0.10.0", "channel": "stable", "quality": "reviewed", "url": "new.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	artifact, ok := idx.Find("ecs", "")
	if !ok {
		t.Fatal("Find returned false")
	}
	if artifact.Version != "0.10.0" {
		t.Fatalf("version = %q, want 0.10.0", artifact.Version)
	}
}

func TestFindArtifactSupportsExplicitChannel(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "stable.tar.gz"},
    {"name": "ecs", "version": "0.3.0", "channel": "beta", "quality": "reviewed", "url": "beta.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	artifact, ok := idx.Find("ecs", "beta")
	if !ok {
		t.Fatal("Find beta returned false")
	}
	if artifact.URL != "beta.tar.gz" {
		t.Fatalf("url = %q, want beta.tar.gz", artifact.URL)
	}
	if _, ok := idx.Find("missing", "beta"); ok {
		t.Fatal("Find returned true for missing plugin")
	}
}

func TestSearchDefaultsToStableReviewedOrCuratedLatestPerPlugin(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "generated", "url": "generated.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "reviewed.tar.gz"},
    {"name": "ecs", "version": "0.3.0", "channel": "stable", "quality": "curated", "url": "curated.tar.gz"},
    {"name": "vpc", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "vpc.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	results := idx.Search("", "")
	if len(results) != 2 {
		t.Fatalf("Search returned %d results, want 2: %#v", len(results), results)
	}
	if results[0].Name != "ecs" || results[0].Version != "0.3.0" {
		t.Fatalf("first result = %#v, want latest curated ecs", results[0])
	}
	for _, artifact := range results {
		if artifact.Quality == "generated" {
			t.Fatalf("Search exposed generated stable artifact: %#v", artifact)
		}
	}
}

func TestSearchFiltersByQuery(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "product": "ecs", "display_name": "Elastic Cloud Server", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs.tar.gz"},
    {"name": "vpc", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "vpc.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	results := idx.Search("vp", "")
	if len(results) != 1 || results[0].Name != "vpc" {
		t.Fatalf("Search query results = %#v, want only vpc", results)
	}
	if strings.Contains(results[0].URL, "ecs") {
		t.Fatalf("Search returned ecs for vpc query: %#v", results[0])
	}

	results = idx.Search("elc", "")
	if len(results) != 1 || results[0].Name != "ecs" || results[0].Product != "ecs" || results[0].DisplayName != "Elastic Cloud Server" {
		t.Fatalf("Search fuzzy display-name results = %#v, want ecs storefront metadata", results)
	}

	if results := idx.Search("missing", ""); len(results) != 0 {
		t.Fatalf("Search missing returned %#v, want none", results)
	}
	if results := idx.Search("", "beta"); len(results) != 0 {
		t.Fatalf("Search beta returned %#v, want none", results)
	}
	if !isSubsequence("", "ecs") {
		t.Fatal("isSubsequence returned false for empty query")
	}
}

func TestSearchSortsSameNameByNewestVersion(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "old.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "new.tar.gz"},
    {"name": "abc", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "abc.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}
	results := idx.Search("", "")
	if len(results) != 2 {
		t.Fatalf("Search returned %#v, want two latest plugin rows", results)
	}
	if results[1].Name != "ecs" || results[1].Version != "0.2.0" {
		t.Fatalf("ecs result = %#v, want latest 0.2.0", results[1])
	}
}

func TestLoadIndexRejectsInvalidArtifactMetadata(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "missing name",
			raw:  `{"plugins":[{"version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "error.registry_missing_name",
		},
		{
			name: "missing version",
			raw:  `{"plugins":[{"name":"ecs","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "error.registry_missing_version",
		},
		{
			name: "bad channel",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"nightly","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "error.registry_unsupported_channel",
		},
		{
			name: "bad quality",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"raw","url":"ecs.tar.gz"}]}`,
			want: "error.registry_unsupported_quality",
		},
		{
			name: "missing url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed"}]}`,
			want: "error.registry_missing_url",
		},
		{
			name: "unsafe name",
			raw:  `{"plugins":[{"name":"../ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "error.registry_invalid_plugin_name",
		},
		{
			name: "traversal url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"../ecs.tar.gz"}]}`,
			want: "error.registry_invalid_artifact_url",
		},
		{
			name: "absolute local url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"/tmp/ecs.tar.gz"}]}`,
			want: "error.registry_invalid_artifact_url",
		},
		{
			name: "unsupported url scheme",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"file:///tmp/ecs.tar.gz"}]}`,
			want: "error.registry_invalid_artifact_url",
		},
		{
			name: "backslash url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"plugins\\ecs.tar.gz"}]}`,
			want: "error.registry_invalid_artifact_url",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadIndex([]byte(tc.raw))
			if err == nil {
				t.Fatal("LoadIndex returned nil error")
			}
			requireDiagnosticKey(t, err, tc.want)
		})
	}
}

func TestLoadIndexRejectsMalformedJSON(t *testing.T) {
	_, err := LoadIndex([]byte(`{"plugins":`))
	if err == nil {
		t.Fatal("LoadIndex returned nil error for malformed JSON")
	}
	requireDiagnosticKey(t, err, "error.parse_registry_index")
}

func TestCompareVersionEqualVersions(t *testing.T) {
	if got := coreversion.CompareSemanticVersions("0.1.0+build.2", "0.1.0+build.1"); got != 0 {
		t.Fatalf("CompareSemanticVersions equal = %d, want 0", got)
	}
	if got := coreversion.CompareSemanticVersions("0.2.0", "0.2.0-beta.1"); got <= 0 {
		t.Fatalf("CompareSemanticVersions stable after prerelease = %d, want positive", got)
	}
}

func requireDiagnosticKey(t *testing.T, err error, want string) {
	t.Helper()
	got, ok := err.(interface{ MessageKey() string })
	if !ok {
		t.Fatalf("error %T does not expose a diagnostic key: %v", err, err)
	}
	if got.MessageKey() != want {
		t.Fatalf("diagnostic key = %q, want %q", got.MessageKey(), want)
	}
}
