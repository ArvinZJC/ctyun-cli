/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package version

import "testing"

func TestIsSemanticVersion(t *testing.T) {
	for _, value := range []string{"0.2.0", "0.2.0-beta.1", "0.2.0+build.1", "0.2.0-beta.1+build.1"} {
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
