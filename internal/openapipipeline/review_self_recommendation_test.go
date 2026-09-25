/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestNewPluginRecommendationsResolveWithinDraft allows a first promotion to
// reference another command in the same reviewed candidate bundle.
func TestNewPluginRecommendationsResolveWithinDraft(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := loadCatalogFixture(t)
	target := catalog.Operations[0]
	target.ID = "ecs.instance.list-new"
	target.APIID = "9999"
	target.Path = "/v4/ecs/instance/list-new"
	target.CommandPath = []string{"instance", "list-new"}
	target.Examples = nil
	catalog.Operations = append(catalog.Operations, target)
	catalog.Operations[0].Recommendation = &APIRecommendation{Notice: "Recommend the replacement query", TargetAPI: APIReference{Method: target.Method, Path: target.Path}, TargetCommand: &plugin.CommandTarget{Plugin: "ecs", Path: []string{"ecs", "instance", "list-new"}}}
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)
	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("self-contained first draft rejected: %#v %v", report, err)
	}
	if err := workspace.PromoteDraft("ecs", filepath.Join(workspace.Root, "plugins", "ecs")); err != nil {
		t.Fatal(err)
	}
}
