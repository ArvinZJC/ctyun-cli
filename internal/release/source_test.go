/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

func TestResolveSourceUsesAuto(t *testing.T) {
	got, err := ResolveSource(SourceOptions{
		Requested: "",
		Getenv:    func(string) string { return "" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != distribution.SourceReady || got.Name != "github" || len(got.Fallbacks) != 1 || got.Fallbacks[0].Name != "gitee" {
		t.Fatalf("source = %#v, want ready github with gitee fallback", got)
	}
}

func TestResolveSourceAcceptsExplicitAuto(t *testing.T) {
	got, err := ResolveSource(SourceOptions{
		Requested: "auto",
		Getenv:    func(string) string { return "" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != distribution.SourceReady || got.Name != "github" || len(got.Fallbacks) != 1 || got.Fallbacks[0].Name != "gitee" {
		t.Fatalf("source = %#v, want ready github with gitee fallback", got)
	}
}

func TestResolveSourceRejectsCustomSources(t *testing.T) {
	if _, err := ResolveSource(SourceOptions{Requested: "./dist/releases"}); err == nil {
		t.Fatal("ResolveSource returned nil error for local release source")
	}
	if _, err := ResolveSource(SourceOptions{Requested: "https://releases.example.test"}); err == nil {
		t.Fatal("ResolveSource returned nil error for custom release URL")
	}
}

func TestResolveSourceNamedMirrors(t *testing.T) {
	for _, name := range []string{"github", "gitee"} {
		got, err := ResolveSource(SourceOptions{Requested: name})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.URL == "" || got.Name != name {
			t.Fatalf("%s source = %#v, want named URL", name, got)
		}
		if got.Kind != distribution.SourceReady {
			t.Fatalf("%s kind = %v, want ready", name, got.Kind)
		}
	}
}

func TestResolveSourceAutoForReleaseBuildUsesGitHubWithGiteeFallback(t *testing.T) {
	got, err := ResolveSource(SourceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "github" || len(got.Fallbacks) != 1 || got.Fallbacks[0].Name != "gitee" {
		t.Fatalf("source = %#v, want github with gitee fallback", got)
	}
	if got.Kind != distribution.SourceReady || got.Fallbacks[0].Kind != distribution.SourceReady {
		t.Fatalf("source kinds = %#v, want ready source and fallback", got)
	}
}

func TestResolveSourceUsesEnvironment(t *testing.T) {
	got, err := ResolveSource(SourceOptions{
		Getenv: func(key string) string {
			if key == "CTYUN_UPGRADE_SOURCE" {
				return "gitee"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "gitee" {
		t.Fatalf("source = %#v, want gitee from environment", got)
	}
	if got.Kind != distribution.SourceReady {
		t.Fatalf("kind = %v, want ready", got.Kind)
	}
}
