/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import "testing"

func TestResolveSourceDevelopmentBuildRequiresExplicitSource(t *testing.T) {
	got, err := ResolveSource(SourceOptions{
		Requested:      "",
		CurrentVersion: "0.1.0-dev",
		Getenv:         func(string) string { return "" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != SourceDevelopmentUnavailable {
		t.Fatalf("kind = %v, want development unavailable", got.Kind)
	}
}

func TestResolveSourceRejectsCustomSources(t *testing.T) {
	if _, err := ResolveSource(SourceOptions{Requested: "./dist/releases", CurrentVersion: "0.2.0"}); err == nil {
		t.Fatal("ResolveSource returned nil error for local release source")
	}
	if _, err := ResolveSource(SourceOptions{Requested: "https://releases.example.test", CurrentVersion: "0.2.0"}); err == nil {
		t.Fatal("ResolveSource returned nil error for custom release URL")
	}
}

func TestResolveSourceNamedMirrors(t *testing.T) {
	for _, name := range []string{"github", "gitee"} {
		got, err := ResolveSource(SourceOptions{Requested: name, CurrentVersion: "0.2.0"})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.URL == "" || got.Name != name {
			t.Fatalf("%s source = %#v, want named URL", name, got)
		}
	}
}

func TestResolveSourceAutoForReleaseBuildUsesGitHubWithGiteeFallback(t *testing.T) {
	got, err := ResolveSource(SourceOptions{CurrentVersion: "0.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "github" || len(got.Fallbacks) != 1 || got.Fallbacks[0].Name != "gitee" {
		t.Fatalf("source = %#v, want github with gitee fallback", got)
	}
}

func TestResolveSourceUsesEnvironment(t *testing.T) {
	got, err := ResolveSource(SourceOptions{
		CurrentVersion: "0.2.0",
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
}

func TestResolveSourceRejectsEnvironmentURL(t *testing.T) {
	_, err := ResolveSource(SourceOptions{
		CurrentVersion: "0.2.0",
		Getenv: func(key string) string {
			if key == "CTYUN_UPGRADE_URL" {
				return "https://mirror.example.test/releases"
			}
			return ""
		},
	})
	if err == nil {
		t.Fatal("ResolveSource returned nil error for custom environment release URL")
	}
}
