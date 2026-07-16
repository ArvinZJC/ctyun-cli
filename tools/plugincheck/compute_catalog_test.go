/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// computePluginExpectation pins one Compute product's released identity and
// exact upstream API inventory.
type computePluginExpectation struct {
	name          string
	displayNameEN string
	productID     int
	revision      string
	endpoint      string
	apiIDs        []string
}

// assertComputePluginMatchesCatalog verifies that one promoted Compute plugin
// preserves its approved catalog and release identity.
func assertComputePluginMatchesCatalog(t *testing.T, expected computePluginExpectation) {
	t.Helper()
	baselinePath := repoPath(t, filepath.Join("openapi-catalogs", expected.name, "baseline.json"))
	content, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read %s baseline: %v", expected.name, err)
	}
	var baseline openapipipeline.Catalog
	if err := json.Unmarshal(content, &baseline); err != nil {
		t.Fatalf("parse %s baseline: %v", expected.name, err)
	}
	bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", expected.name)), version.Version)
	if err != nil {
		t.Fatalf("load %s plugin: %v", expected.name, err)
	}
	manifest := bundle.Manifest
	if manifest.Name != expected.name || manifest.Version != "0.1.0-beta.1" || manifest.Channel != "beta" || manifest.Quality != "generated" {
		t.Fatalf("manifest release = %s/%s/%s/%s", manifest.Name, manifest.Version, manifest.Channel, manifest.Quality)
	}
	if manifest.API.CtyunProductID != expected.productID || manifest.API.SourceRevision != expected.revision || manifest.API.EndpointURL != expected.endpoint {
		t.Fatalf("manifest API identity = %d/%s/%s", manifest.API.CtyunProductID, manifest.API.SourceRevision, manifest.API.EndpointURL)
	}
	if expected.displayNameEN != "" {
		if got := baseline.Product.DisplayName["en-US"]; got != expected.displayNameEN {
			t.Fatalf("baseline English product name = %q, want %q", got, expected.displayNameEN)
		}
		for _, language := range []string{"en-GB", "en-US"} {
			if got := bundle.I18N[language]["name"]; got != expected.displayNameEN {
				t.Fatalf("%s plugin product name = %q, want %q", language, got, expected.displayNameEN)
			}
		}
	}
	if !reflect.DeepEqual(manifest.API.Scope, baseline.Product.APIScope) {
		t.Fatalf("manifest scope = %#v, want %#v", manifest.API.Scope, baseline.Product.APIScope)
	}

	gotAPIIDs := make([]string, 0, len(baseline.Operations))
	commands := make(map[string]plugin.Command, len(bundle.Commands.Commands))
	for _, command := range bundle.Commands.Commands {
		if _, exists := commands[command.Operation]; exists {
			t.Fatalf("duplicate command operation %s", command.Operation)
		}
		commands[command.Operation] = command
	}
	if len(bundle.APIs.Operations) != len(expected.apiIDs) || len(commands) != len(expected.apiIDs) {
		t.Fatalf("commands/operations = %d/%d, want %d/%d", len(commands), len(bundle.APIs.Operations), len(expected.apiIDs), len(expected.apiIDs))
	}
	for _, source := range baseline.Operations {
		gotAPIIDs = append(gotAPIIDs, source.APIID)
		operation, ok := bundle.APIs.Operations[source.ID]
		if !ok {
			t.Fatalf("missing operation %s", source.ID)
		}
		if operation.Method != source.Method || operation.Path != source.Path || operation.Retryable != source.Retryable || !reflect.DeepEqual(operation.AcceptedStatuses, source.Response.AcceptedStatuses) {
			t.Fatalf("operation %s differs from baseline", source.ID)
		}
		command, ok := commands[source.ID]
		if !ok {
			t.Fatalf("missing command for operation %s", source.ID)
		}
		if got := bundle.Tables.Tables[command.Table].RowPath; got != source.Response.RowPath {
			t.Fatalf("operation %s row path = %q, want %q", source.ID, got, source.Response.RowPath)
		}
		if dangerous := command.Dangerous.Confirm != ""; dangerous != source.Dangerous {
			t.Fatalf("operation %s dangerous = %t, want %t", source.ID, dangerous, source.Dangerous)
		}
	}
	slices.Sort(gotAPIIDs)
	wantAPIIDs := slices.Clone(expected.apiIDs)
	slices.Sort(wantAPIIDs)
	if !slices.Equal(gotAPIIDs, wantAPIIDs) {
		t.Fatalf("API IDs = %#v, want %#v", gotAPIIDs, wantAPIIDs)
	}
}
