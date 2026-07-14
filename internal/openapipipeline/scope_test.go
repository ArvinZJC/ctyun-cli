/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestValidateAPIScope(t *testing.T) {
	cases := []struct {
		name  string
		scope plugin.APIScope
		want  string
	}{
		{name: "empty"},
		{name: "valid include and exclude", scope: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/ecs/"}, ExcludeURIPrefixes: []string{"/v4/ecs/internal/"}}},
		{name: "bad include", scope: plugin.APIScope{IncludeURIPrefixes: []string{""}}, want: "product.api_scope.include_uri_prefixes contains an empty prefix"},
		{name: "bad exclude", scope: plugin.APIScope{ExcludeURIPrefixes: []string{"v4/ecs/"}}, want: "product.api_scope.exclude_uri_prefixes prefix v4/ecs/ must start with /"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAPIScope(tc.scope)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateAPIScope returned error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("validateAPIScope returned nil error")
			}
			if err.Error() != tc.want {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestValidateURIPrefix(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		want   string
	}{
		{name: "valid", prefix: "/v4/ecs/"},
		{name: "empty", want: "field contains an empty prefix"},
		{name: "relative", prefix: "v4/ecs/", want: "field prefix v4/ecs/ must start with /"},
		{name: "query", prefix: "/v4/ecs/?", want: "field prefix /v4/ecs/? is invalid"},
		{name: "parent", prefix: "/v4/../ecs/", want: "field prefix /v4/../ecs/ is invalid"},
		{name: "current", prefix: "/v4/./ecs/", want: "field prefix /v4/./ecs/ is invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateURIPrefix("field", tc.prefix)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateURIPrefix returned error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("validateURIPrefix returned nil error")
			}
			if err.Error() != tc.want {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestOperationInAPIScope(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		scope plugin.APIScope
		want  bool
	}{
		{name: "empty scope", path: "/v4/region/list-regions", want: true},
		{name: "included", path: "/v4/ecs/list-instances", scope: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/ecs/"}}, want: true},
		{name: "not included", path: "/v4/region/list-regions", scope: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/ecs/"}}},
		{name: "excluded", path: "/v4/ecs/internal/list", scope: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/ecs/"}, ExcludeURIPrefixes: []string{"/v4/ecs/internal/"}}},
		{name: "exclude misses", path: "/v4/ecs/list-instances", scope: plugin.APIScope{ExcludeURIPrefixes: []string{"/v4/ecs/internal/"}}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := operationInAPIScope(tc.path, tc.scope); got != tc.want {
				t.Fatalf("operationInAPIScope = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAPIScopeEqual(t *testing.T) {
	base := plugin.APIScope{
		IncludeURIPrefixes: []string{"/v4/ecs/"},
		ExcludeURIPrefixes: []string{"/v4/ecs/internal/"},
		Notes:              "scope",
	}
	cases := []struct {
		name string
		next plugin.APIScope
		want bool
	}{
		{name: "same", next: base, want: true},
		{name: "notes", next: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/ecs/"}, ExcludeURIPrefixes: []string{"/v4/ecs/internal/"}, Notes: "other"}},
		{name: "include length", next: plugin.APIScope{ExcludeURIPrefixes: []string{"/v4/ecs/internal/"}, Notes: "scope"}},
		{name: "exclude length", next: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/ecs/"}, Notes: "scope"}},
		{name: "include value", next: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/region/"}, ExcludeURIPrefixes: []string{"/v4/ecs/internal/"}, Notes: "scope"}},
		{name: "exclude value", next: plugin.APIScope{IncludeURIPrefixes: []string{"/v4/ecs/"}, ExcludeURIPrefixes: []string{"/v4/ecs/private/"}, Notes: "scope"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := apiScopeEqual(base, tc.next); got != tc.want {
				t.Fatalf("apiScopeEqual = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCatalogValidationRejectsAPIScopeDrift(t *testing.T) {
	cases := []catalogValidationCase{
		{name: "invalid API scope prefix", mutate: func(catalog *Catalog) {
			catalog.Product.APIScope.IncludeURIPrefixes = []string{"v4/ecs/"}
		}, want: "product.api_scope.include_uri_prefixes prefix v4/ecs/ must start with /"},
		{name: "operation outside API scope", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Path = "/v4/region/list-regions"
		}, want: "operation v4.ecs.instance.list path /v4/region/list-regions is outside product.api_scope"},
	}
	runCatalogValidationCases(t, cases)
}

// TestPromotedPluginMetadataMatchesBaseline verifies that released plugin
// provenance matches the last promoted catalog rather than newer source drift.
func TestPromotedPluginMetadataMatchesBaseline(t *testing.T) {
	for _, product := range []string{"ecs", "job", "region"} {
		t.Run(product, func(t *testing.T) {
			baseline := readCatalogFile(t, filepath.Join("..", "..", "openapi-catalogs", product, "baseline.json"))
			manifest := readJSONFile[plugin.Manifest](t, filepath.Join("..", "..", "plugins", product, "plugin.json"))
			if manifest.API.SourceFingerprint != catalogFingerprint(baseline) {
				t.Fatalf("promoted source fingerprint = %q, want baseline fingerprint %q", manifest.API.SourceFingerprint, catalogFingerprint(baseline))
			}
			if !apiScopeEqual(manifest.API.Scope, baseline.Product.APIScope) {
				t.Fatalf("promoted manifest API scope = %#v, want baseline API scope %#v", manifest.API.Scope, baseline.Product.APIScope)
			}
		})
	}
}
