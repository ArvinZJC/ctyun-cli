/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestReviewRejectsExecutionDrift prevents promotion from certifying requests
// that differ from their tracked source while preserving the accepted baseline.
func TestReviewRejectsExecutionDrift(t *testing.T) {
	for _, name := range []string{"path", "method", "binding", "retryable", "accepted status", "missing APIs", "parameter", "endpoint"} {
		t.Run(name, func(t *testing.T) {
			workspace := Workspace{Root: t.TempDir()}
			writeCatalogAndGenerateDraft(t, workspace, "ecs", loadCatalogFixture(t))
			apiPath := workspace.ProductPath("ecs", "draft", "apis.json")
			apis, err := readDraftJSON[plugin.APIs](apiPath)
			if err != nil {
				t.Fatal(err)
			}
			operation := apis.Operations["v4.ecs.instance.list"]
			switch name {
			case "path":
				operation.Path = "/v4/outside-scope/delete"
			case "method":
				operation.Method = "DELETE"
			case "binding":
				operation.Body["regionID"] = "unreviewed-region"
			case "retryable":
				operation.Retryable = false
			case "accepted status":
				operation.AcceptedStatuses = []plugin.AcceptedStatusRule{{Code: "900", RequiredPath: "returnObj"}}
			}
			apis.Operations["v4.ecs.instance.list"] = operation
			if err := writeJSON(apiPath, apis); err != nil {
				t.Fatal(err)
			}
			if name == "missing APIs" {
				if err := os.Remove(apiPath); err != nil {
					t.Fatal(err)
				}
			}
			if name == "parameter" {
				path := workspace.ProductPath("ecs", "draft", "commands.json")
				commands, err := readDraftJSON[plugin.Commands](path)
				if err != nil {
					t.Fatal(err)
				}
				commands.Commands[0].Parameters[0].Target = "unreviewedTarget"
				if err := writeJSON(path, commands); err != nil {
					t.Fatal(err)
				}
			}
			if name == "endpoint" {
				path := workspace.ProductPath("ecs", "draft", "plugin.json")
				manifest, err := readDraftJSON[plugin.Manifest](path)
				if err != nil {
					t.Fatal(err)
				}
				manifest.API.EndpointURL = "https://unreviewed.example.com"
				if err := writeJSON(path, manifest); err != nil {
					t.Fatal(err)
				}
			}
			report, err := workspace.ReviewDraft("ecs")
			if err == nil && report.Ready {
				t.Fatal("review accepted unreviewed execution contract")
			}
			target := filepath.Join(workspace.Root, "plugins", "ecs")
			if err := workspace.PromoteDraft("ecs", target); err == nil {
				t.Fatal("promoted unreviewed execution contract")
			}
			if _, err := os.Stat(filepath.Join(target, "plugin.json")); !os.IsNotExist(err) {
				t.Fatal("failed promotion wrote plugin")
			}
			if _, err := os.Stat(workspace.ProductPath("ecs", "baseline.json")); !os.IsNotExist(err) {
				t.Fatal("failed promotion advanced baseline")
			}
		})
	}
}

// TestReviewAllowsPresentationEdits keeps reviewed wording and table layout
// independent from the source-controlled request and command contract.
func TestReviewAllowsPresentationEdits(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", loadCatalogFixture(t))
	path := workspace.ProductPath("ecs", "draft", "commands.json")
	commands, err := readDraftJSON[plugin.Commands](path)
	if err != nil {
		t.Fatal(err)
	}
	commands.Commands[0].Parameters[0].Description = "Reviewed help wording"
	if err := writeJSON(path, commands); err != nil {
		t.Fatal(err)
	}
	path = workspace.ProductPath("ecs", "draft", "tables.json")
	tables, err := readDraftJSON[plugin.Tables](path)
	if err != nil {
		t.Fatal(err)
	}
	for id, table := range tables.Tables {
		table.Layout = "vertical"
		tables.Tables[id] = table
	}
	if err := writeJSON(path, tables); err != nil {
		t.Fatal(err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("presentation edit blocked: %+v %v", report, err)
	}
}

// TestPromotionCopyFailureKeepsBaseline checks a valid reviewed draft against
// an unwritable destination without weakening the earlier execution-contract gate.
func TestPromotionCopyFailureKeepsBaseline(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", loadCatalogFixture(t))
	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("valid draft: %+v %v", report, err)
	}
	target := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(target, []byte("existing data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := workspace.PromoteDraft("ecs", target); err == nil {
		t.Fatal("accepted non-directory destination")
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "existing data" {
		t.Fatalf("destination changed: %s %v", data, err)
	}
	if _, err := os.Stat(workspace.ProductPath("ecs", "baseline.json")); !os.IsNotExist(err) {
		t.Fatalf("failed copy advanced baseline: %v", err)
	}
}
