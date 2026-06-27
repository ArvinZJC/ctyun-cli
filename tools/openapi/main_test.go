/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHarvestRequiresInput(t *testing.T) {
	var stdout bytes.Buffer
	err := run([]string{"harvest", "ecs"}, t.TempDir(), &stdout)
	if err == nil {
		t.Fatal("run returned nil error")
	}
	if err.Error() != "harvest requires --input" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRunHarvestAndDiff(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "ecs-source.json")
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "openapi", "testdata", "ecs-source.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(input, data, 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	if err := run([]string{"harvest", "ecs", "--input", input}, root, &stdout); err != nil {
		t.Fatalf("harvest returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "wrote openapi/products/ecs/source.json") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	stdout.Reset()
	if err := run([]string{"diff", "ecs"}, root, &stdout); err != nil {
		t.Fatalf("diff returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "wrote openapi/products/ecs/changes.md") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
