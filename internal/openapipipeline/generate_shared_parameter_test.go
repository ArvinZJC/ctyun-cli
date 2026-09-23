/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestGeneratedSharedRegionParameterPreservesRequestBindings verifies that a
// header and body sharing one region input emit one usable option and two bindings.
func TestGeneratedSharedRegionParameterPreservesRequestBindings(t *testing.T) {
	catalog := sharedRegionParameterCatalog(t)
	directory := t.TempDir()
	if err := writeDraft(directory, catalog, buildManifest(catalog)); err != nil {
		t.Fatal(err)
	}
	bundle, err := plugin.LoadBundle(directory, "0.5.0")
	if err != nil {
		t.Fatalf("load shared-region draft: %v", err)
	}
	command := bundle.Commands.Commands[0]
	if len(command.Parameters) != 1 || command.Parameters[0].Flag != "region" {
		t.Fatalf("parameters = %#v, want one region option", command.Parameters)
	}
	operation := bundle.APIs.Operations[command.Operation]
	if operation.Headers["regionId"] != "$profile.region" || operation.Body["regionId"] != "$profile.region" {
		t.Fatalf("shared region bindings lost: headers=%#v body=%#v", operation.Headers, operation.Body)
	}
	if len(command.Examples) != 1 || strings.Count(command.Examples[0], "--region") != 1 {
		t.Fatalf("examples = %#v, want one region option", command.Examples)
	}
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		if got := bundle.I18N[language]["parameter."+command.ID+".region.description"]; got != catalog.Operations[0].Parameters[0].Descriptions[language] {
			t.Fatalf("%s region description = %q", language, got)
		}
	}
}

// TestGeneratedSharedRegionParameterRetainsConflicts prevents deduplication
// from hiding incompatible constraints behind the same visible option.
func TestGeneratedSharedRegionParameterRetainsConflicts(t *testing.T) {
	catalog := sharedRegionParameterCatalog(t)
	catalog.Operations[0].Parameters[1].Enum = []string{"region-only"}
	directory := t.TempDir()
	if err := writeDraft(directory, catalog, buildManifest(catalog)); err != nil {
		t.Fatal(err)
	}
	if _, err := plugin.LoadBundle(directory, "0.5.0"); err == nil {
		t.Fatal("conflicting shared region declarations loaded successfully")
	}
	if parameters := buildCommands(catalog).Commands[0].Parameters; len(parameters) != 2 || len(parameters[1].AllowedValues) != 1 {
		t.Fatalf("conflicting constraints were discarded: %#v", parameters)
	}
}

// sharedRegionParameterCatalog supplies equivalent region declarations in two
// request locations without depending on any product-specific metadata.
func sharedRegionParameterCatalog(t *testing.T) Catalog {
	t.Helper()
	catalog := loadCatalogFixture(t)
	catalog.Operations = catalog.Operations[:1]
	parameter := Parameter{
		Name: "regionId", Location: "header", Required: true, Type: "string", Profile: "region",
		Descriptions: map[string]string{"en-US": "Region", "en-GB": "Region", "zh-CN": "资源池"},
		Example:      json.RawMessage(`"example-region"`),
	}
	catalog.Operations[0].Parameters = []Parameter{parameter, parameter}
	catalog.Operations[0].Parameters[1].Location = "body"
	catalog.Operations[0].Examples = []string{"ctyun ecs instance list --region example-region"}
	return catalog
}
