/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package version

import "testing"

func TestIsDevelopmentBuild(t *testing.T) {
	original := Version
	t.Cleanup(func() { Version = original })

	Version = "0.1.0-dev"
	if !IsDevelopmentBuild() {
		t.Fatal("IsDevelopmentBuild returned false for dev version")
	}

	Version = "0.1.0"
	if IsDevelopmentBuild() {
		t.Fatal("IsDevelopmentBuild returned true for release version")
	}
}
