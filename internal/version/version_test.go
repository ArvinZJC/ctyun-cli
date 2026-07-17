/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package version

import "testing"

// TestDefaultVersionMatchesCurrentRelease keeps source-build identity aligned
// with the release currently being prepared.
func TestDefaultVersionMatchesCurrentRelease(t *testing.T) {
	if Version != "0.4.0" {
		t.Fatalf("Version = %q, want 0.4.0", Version)
	}
}

func TestIsSemanticVersion(t *testing.T) {
	for _, value := range []string{"0.1.0-dev", "0.1.0-alpha.1", "0.2.0", "0.2.0-beta.1", "0.2.0+build.1", "0.2.0-beta.1+build.1"} {
		if !IsSemanticVersion(value) {
			t.Fatalf("IsSemanticVersion(%q) = false, want true", value)
		}
	}

	for _, value := range []string{"v0.2.0", "0.2", "0.02.0", "0.2.0-01"} {
		if IsSemanticVersion(value) {
			t.Fatalf("IsSemanticVersion(%q) = true, want false", value)
		}
	}
}

func TestCompareSemanticVersions(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "newer minor", left: "0.2.0", right: "0.1.0", want: 1},
		{name: "older stable than beta", left: "0.1.0", right: "0.2.0-beta.1", want: -1},
		{name: "stable release after prerelease", left: "0.2.0", right: "0.2.0-beta.1", want: 1},
		{name: "prerelease before stable", left: "0.2.0-beta.1", right: "0.2.0", want: -1},
		{name: "numeric prerelease order", left: "0.2.0-beta.2", right: "0.2.0-beta.11", want: -1},
		{name: "numeric identifiers before text", left: "0.2.0-1", right: "0.2.0-alpha", want: -1},
		{name: "longer prerelease wins when equal prefix", left: "0.2.0-alpha.1", right: "0.2.0-alpha", want: 1},
		{name: "equal prerelease", left: "0.2.0-alpha.1", right: "0.2.0-alpha.1", want: 0},
		{name: "text prerelease order", left: "0.2.0-beta", right: "0.2.0-alpha", want: 1},
		{name: "build metadata ignored", left: "0.2.0+build.2", right: "0.2.0+build.1", want: 0},
		{name: "invalid fallback", left: "bad", right: "0.0.0", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareSemanticVersions(tt.left, tt.right); got != tt.want {
				t.Fatalf("CompareSemanticVersions(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
			}
			if reverse := CompareSemanticVersions(tt.right, tt.left); reverse != -tt.want {
				t.Fatalf("reverse CompareSemanticVersions(%q, %q) = %d, want %d", tt.right, tt.left, reverse, -tt.want)
			}
		})
	}
}

func TestIsDevelopmentBuild(t *testing.T) {
	original := Version
	originalChannel := Channel
	t.Cleanup(func() {
		Version = original
		Channel = originalChannel
	})

	Version = "0.1.0"
	Channel = "dev"
	if !IsDevelopmentBuild() {
		t.Fatal("IsDevelopmentBuild returned false for dev channel")
	}

	Version = "0.1.0-dev"
	Channel = "stable"
	if IsDevelopmentBuild() {
		t.Fatal("IsDevelopmentBuild returned true for release channel")
	}
}
