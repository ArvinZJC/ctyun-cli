/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package distribution

import "testing"

func TestResolveSourceVariants(t *testing.T) {
	opts := SourceOptions{
		Label:     "test",
		GitHubURL: "https://github.example.test/root",
		GiteeURL:  "https://gitee.example.test/root",
	}
	got, err := ResolveSource(opts)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "github" || len(got.Fallbacks) != 1 || got.Fallbacks[0].Name != "gitee" {
		t.Fatalf("auto source = %#v, want github with gitee fallback", got)
	}

	got, err = ResolveSource(SourceOptions{Requested: "gitee", GitHubURL: opts.GitHubURL, GiteeURL: opts.GiteeURL})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "gitee" || got.URL != opts.GiteeURL {
		t.Fatalf("gitee source = %#v", got)
	}

	got, err = ResolveSource(SourceOptions{EnvName: "CTYUN_TEST_SOURCE", GitHubURL: opts.GitHubURL, GiteeURL: opts.GiteeURL, Getenv: func(key string) string {
		if key == "CTYUN_TEST_SOURCE" {
			return "github"
		}
		return ""
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "github" {
		t.Fatalf("env source = %#v, want github", got)
	}
	got, err = ResolveSource(SourceOptions{EnvName: "CTYUN_TEST_SOURCE", GitHubURL: opts.GitHubURL, GiteeURL: opts.GiteeURL})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "github" {
		t.Fatalf("nil getenv env source = %#v, want default auto github", got)
	}

	got, err = ResolveSource(SourceOptions{DevelopmentBuild: true, DisableDevAuto: true, GitHubURL: opts.GitHubURL, GiteeURL: opts.GiteeURL})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != SourceDevelopmentUnavailable {
		t.Fatalf("dev auto source = %#v, want unavailable", got)
	}

	if _, err := ResolveSource(SourceOptions{Requested: "./registry"}); err == nil {
		t.Fatal("ResolveSource returned nil error for release custom source")
	}
	if _, err := ResolveSource(SourceOptions{Requested: "./registry", DevelopmentBuild: true}); err == nil {
		t.Fatal("ResolveSource returned nil error for dev custom source")
	}
}
