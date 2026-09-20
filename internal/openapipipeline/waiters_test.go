/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// TestCatalogWaitersGenerateExplicitDefinitions prevents both lost declarations
// and lifecycle guesses based only on a response column name.
func TestCatalogWaitersGenerateExplicitDefinitions(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Waiters = nil
	if err := json.Unmarshal([]byte(`{"waiters":{"ecs.instance.ready":{"commands":["ecs.instance.show"],"path":"returnObj.instanceStatus","success":"running","failure":"error","max_attempts":20,"interval_seconds":3}}}`), &catalog); err != nil {
		t.Fatal(err)
	}
	got := buildWaiters(catalog).Waiters
	if len(got) != 1 || got["ecs.instance.ready"].Success != "running" {
		t.Fatalf("explicit waiters not preserved: %#v", got)
	}
	catalog.Waiters = nil
	if got := buildWaiters(catalog).Waiters; len(got) != 0 {
		t.Fatalf("invented waiters: %#v", got)
	}
}

// TestWaiterReviewRejectsDrift prevents promotion from silently replacing
// reviewed polling semantics with edited draft metadata.
func TestWaiterReviewRejectsDrift(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := loadCatalogFixture(t)
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
	path := workspace.ProductPath("ecs", "draft", "waiters.json")
	if err := os.WriteFile(path, []byte(`{"waiters":{"invented":{"path":"status","success":"ok"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready {
		t.Fatal("review accepted invented waiter")
	}
}

// TestWaiterDiffReportsChanges makes lifecycle changes visible during review.
func TestWaiterDiffReportsChanges(t *testing.T) {
	baseline := loadCatalogFixture(t)
	source := loadCatalogFixture(t)
	if err := json.Unmarshal([]byte(`{"waiters":{"ready":{"path":"status","success":"ok"}}}`), &source); err != nil {
		t.Fatal(err)
	}
	if report := DiffCatalogs(baseline, source); len(report.Changes) == 0 {
		t.Fatal("waiter change missing from diff")
	}
}

// TestCatalogWaiterValidation checks that declarations remain safe and match
// captured response shape rather than accidentally traversing collection rows.
func TestCatalogWaiterValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Catalog)
		valid  bool
	}{
		{"valid", func(*Catalog) {}, true},
		{"large numeric identities", func(c *Catalog) {
			w := c.Waiters["ready"]
			w.Path = "state"
			w.Selector = &waiter.Selector{Path: "returnObj", Key: "id", Value: "$arg.instance_id"}
			c.Waiters["ready"] = w
			c.Operations[1].ExampleResponse = json.RawMessage(`{"returnObj":[{"id":9007199254740992,"state":"running"},{"id":9007199254740993,"state":"running"}]}`)
		}, true},
		{"missing evidence", func(c *Catalog) { w := c.Waiters["ready"]; w.Evidence = ""; c.Waiters["ready"] = w }, false},
		{"missing command", func(c *Catalog) { w := c.Waiters["ready"]; w.Commands = nil; c.Waiters["ready"] = w }, false},
		{"unbounded", func(c *Catalog) { w := c.Waiters["ready"]; w.MaxAttempts = 0; c.Waiters["ready"] = w }, false},
		{"wrong binding", func(c *Catalog) { w := c.Waiters["ready"]; w.Commands = []string{"missing"}; c.Waiters["ready"] = w }, false},
		{"non-object example", func(c *Catalog) { c.Operations[1].ExampleResponse = json.RawMessage(`[]`) }, false},
		{"array response", func(c *Catalog) {
			c.Operations[1].ExampleResponse = json.RawMessage(`{"returnObj":[{"instanceStatus":"running"}]}`)
		}, false},
		{"missing state", func(c *Catalog) { c.Operations[1].ExampleResponse = json.RawMessage(`{"returnObj":{}}`) }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := loadCatalogFixture(t)
			c.Waiters = map[string]CatalogWaiter{"ready": {Waiter: plugin.Waiter{Commands: []string{"ecs.instance.show"}, Path: "returnObj.instanceStatus", Success: "running", MaxAttempts: 3, IntervalSeconds: 1}, Evidence: "Documented state"}}
			tc.mutate(&c)
			if err := c.Validate(); (err == nil) != tc.valid {
				t.Fatalf("valid=%t error=%v", tc.valid, err)
			}
		})
	}
}

// TestReviewMissingWaiterFile rejects incomplete drafts rather than dropping waits.
func TestReviewMissingWaiterFile(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", loadCatalogFixture(t))
	if err := os.Remove(workspace.ProductPath("ecs", "draft", "waiters.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.ReviewDraft("ecs"); err == nil {
		t.Fatal("accepted missing waiter document")
	}
}

// TestWaiterDraftRaisesCompatibilityFloor protects old cores from silently
// ignoring command bindings and additional terminal values.
func TestWaiterDraftRaisesCompatibilityFloor(t *testing.T) {
	c := loadCatalogFixture(t)
	c.Waiters = map[string]CatalogWaiter{"ready": {Waiter: plugin.Waiter{Commands: []string{"ecs.instance.show"}, Path: "returnObj.instanceStatus", Success: "running", MaxAttempts: 3, IntervalSeconds: 1}, Evidence: "Documented lifecycle"}}
	for _, constraint := range []string{">=0.4.0 <1.0.0", ">=0.6.0 <1.0.0"} {
		workspace := Workspace{Root: t.TempDir()}
		existing := buildManifest(c)
		existing.Requires.Ctyun = constraint
		if err := writeJSON(filepath.Join(workspace.Root, "plugins", "ecs", "plugin.json"), existing); err != nil {
			t.Fatal(err)
		}
		writeCatalogAndGenerateDraft(t, workspace, "ecs", c)
		manifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
		if !strings.Contains(manifest.Requires.Ctyun, ">=0.5.0") && !strings.Contains(manifest.Requires.Ctyun, ">=0.6.0") {
			t.Fatalf("unsafe constraint: %s", manifest.Requires.Ctyun)
		}
		manifest.Requires.Ctyun = ">=0.4.0 <1.0.0"
		if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), manifest); err != nil {
			t.Fatal(err)
		}
		report, err := workspace.ReviewDraft("ecs")
		if err != nil {
			t.Fatal(err)
		}
		if report.Ready {
			t.Fatal("review accepts old-core compatibility")
		}
	}
}
