/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// reviewExecutionContract validates the runnable draft and requires request
// semantics and command bindings to originate in the tracked source catalog.
// Human review may refine help wording and table presentation independently.
func reviewExecutionContract(report *ReviewReport, source Catalog, draftDir string, manifest plugin.Manifest, commands plugin.Commands) {
	if _, err := plugin.LoadBundle(draftDir, version.Version); err != nil {
		addReviewFinding(report, fmt.Sprintf("draft bundle is invalid: %v", err))
	}
	apis, err := readDraftJSON[plugin.APIs](filepath.Join(draftDir, "apis.json"))
	if err != nil {
		addReviewFinding(report, fmt.Sprintf("draft APIs cannot be read: %v", err))
	} else if !sameExecutionJSON(apis, buildAPIs(source)) {
		addReviewFinding(report, "draft API execution contract does not match source catalog")
	}
	for _, operation := range source.Operations {
		if operation.Fixture == nil {
			continue
		}
		fixture, err := readDraftJSON[client.HTTPFixture](filepath.Join(draftDir, fixturePath(operation)))
		if err != nil || !sameExecutionJSON(fixture, operation.Fixture) {
			addReviewFinding(report, fmt.Sprintf("operation %s HTTP fixture does not match captured evidence", operation.ID))
		}
	}
	expectedManifest := buildManifest(source)
	if manifest.Name != expectedManifest.Name || !sameExecutionJSON(manifest.API, expectedManifest.API) {
		addReviewFinding(report, "draft plugin API identity, endpoint, scope, or provenance does not match source catalog")
	}
	expected := buildCommands(source)
	if len(commands.Commands) != len(expected.Commands) {
		addReviewFinding(report, "draft command inventory does not match source catalog")
	}
	byID := make(map[string]plugin.Command, len(commands.Commands))
	for _, command := range commands.Commands {
		byID[command.ID] = command
	}
	for _, command := range expected.Commands {
		actual, ok := byID[command.ID]
		if !ok || !sameExecutionJSON(commandExecutionContract(actual), commandExecutionContract(command)) {
			addReviewFinding(report, fmt.Sprintf("command %s execution contract does not match source catalog", command.ID))
		}
	}
}

// commandExecutionContract retains command dispatch, validation, confirmation,
// and request bindings while excluding separately reviewed presentation fields.
func commandExecutionContract(command plugin.Command) plugin.Command {
	command.Parameters = slices.Clone(command.Parameters)
	for index := range command.Parameters {
		command.Parameters[index].Description = ""
	}
	command.Examples = nil
	command.DocsURL = ""
	command.Dangerous.Message = ""
	command.Recommendation = nil
	return command
}

// sameExecutionJSON compares encoded metadata to normalize omitted empty maps
// and slices without weakening numeric or string value comparisons.
func sameExecutionJSON(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && equivalentJSON(leftJSON, rightJSON)
}
