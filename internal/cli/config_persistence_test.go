/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigSetWritesOnlyConfiguredProfileValues(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	runConfigForTest(t, configPath, nil, "config", "set", "region", "bb9fdb42056f11eda1610242ac110002", "--profile", "huadong1")

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "profiles": {
    "huadong1": {
      "region": "bb9fdb42056f11eda1610242ac110002"
    }
  }
}
`
	if string(raw) != want {
		t.Fatalf("config JSON =\n%s\nwant:\n%s", raw, want)
	}
}
