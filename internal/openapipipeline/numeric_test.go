/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestPromotionPreservesNumericIdentityChanges prevents adjacent large numeric
// identifiers from being treated as equivalent during artifact promotion.
func TestPromotionPreservesNumericIdentityChanges(t *testing.T) {
	before := []byte(`{"id":9007199254740992}`)
	after := []byte(`{"id":9007199254740993}`)
	if equivalentJSON(before, after) {
		t.Fatal("different numeric identities considered equivalent")
	}
	if !equivalentJSON([]byte(`{"values":[1,1.25]}`), []byte(`{"values":[1.0,125e-2]}`)) {
		t.Fatal("equivalent numeric spellings considered different")
	}
	dir := t.TempDir()
	source, target := filepath.Join(dir, "source.json"), filepath.Join(dir, "target.json")
	if err := os.WriteFile(source, after, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := copyJSONFileIfChanged(source, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != string(after) {
		t.Fatalf("got %s: %v", got, err)
	}
}

// TestNumericExampleIdentity retains the upstream identity in generated command
// examples rather than rounding it or writing scientific notation.
func TestNumericExampleIdentity(t *testing.T) {
	operation := Operation{Parameters: []Parameter{{Name: "id", Argument: "id"}}, ExampleResponse: json.RawMessage(`{"id":9007199254740993}`)}
	if got := exampleArgumentValues(operation)["id"]; got != "9007199254740993" {
		t.Fatal(got)
	}
}
