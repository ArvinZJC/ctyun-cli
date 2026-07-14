/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

// catalogOperationRef identifies one tracked operation and its owning catalog.
type catalogOperationRef struct {
	Catalog   Catalog
	Operation Operation
}

// readTrackedCatalogs reads source evidence for every directly tracked OpenAPI
// catalog in deterministic product-directory order.
func (workspace Workspace) readTrackedCatalogs() ([]Catalog, error) {
	root := filepath.Join(workspace.Root, "openapi-catalogs")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	catalogs := make([]Catalog, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "source.json")
		catalog, err := readCatalog(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, catalog)
	}
	sort.SliceStable(catalogs, func(left, right int) bool {
		return catalogs[left].Product.PluginName < catalogs[right].Product.PluginName
	})
	return catalogs, nil
}

// targetAPIOwners returns tracked operations whose HTTP method and path exactly
// match target.
func targetAPIOwners(catalogs []Catalog, target APIReference) []catalogOperationRef {
	var owners []catalogOperationRef
	for _, catalog := range catalogs {
		for _, operation := range catalog.Operations {
			if operation.Method == target.Method && operation.Path == target.Path {
				owners = append(owners, catalogOperationRef{Catalog: catalog, Operation: operation})
			}
		}
	}
	sort.SliceStable(owners, func(left, right int) bool {
		leftPlugin := owners[left].Catalog.Product.PluginName
		rightPlugin := owners[right].Catalog.Product.PluginName
		if leftPlugin != rightPlugin {
			return leftPlugin < rightPlugin
		}
		return owners[left].Operation.ID < owners[right].Operation.ID
	})
	return owners
}

// reviewOperationRecommendation validates source evidence, tracked ownership,
// promoted visible-command resolution, and generated metadata agreement.
func (workspace Workspace) reviewOperationRecommendation(report *ReviewReport, catalogs []Catalog, _ Catalog, operation Operation, command plugin.Command) {
	defer reviewGeneratedRecommendation(report, operation, command)
	if operationHasUnclassifiedRecommendation(operation) {
		addReviewFinding(report, fmt.Sprintf("operation %s has recommendation wording without recommendation metadata", operation.ID))
		return
	}
	recommendation := operation.Recommendation
	if recommendation == nil {
		return
	}
	targetLabel := recommendationAPIKey(recommendation.TargetAPI)
	owners := targetAPIOwners(catalogs, recommendation.TargetAPI)
	if len(owners) == 0 {
		if recommendation.TargetCommand == nil {
			addReviewNote(report, fmt.Sprintf("operation %s recommendation target %s is outside tracked catalogs", operation.ID, targetLabel))
			return
		}
		addReviewFinding(report, fmt.Sprintf("operation %s recommendation target %s is outside tracked catalogs but declares target command %s", operation.ID, targetLabel, commandTargetLabel(*recommendation.TargetCommand)))
		return
	}
	if len(owners) > 1 {
		plugins := make([]string, 0, len(owners))
		for _, owner := range owners {
			plugins = append(plugins, owner.Catalog.Product.PluginName)
		}
		addReviewFinding(report, fmt.Sprintf("operation %s recommendation target %s is owned by multiple tracked catalogs: %s", operation.ID, targetLabel, strings.Join(plugins, ", ")))
		return
	}
	ownerPlugin := owners[0].Catalog.Product.PluginName
	if recommendation.TargetCommand == nil {
		addReviewFinding(report, fmt.Sprintf("operation %s recommendation target %s is tracked by plugin %s but has no target_command", operation.ID, targetLabel, ownerPlugin))
		return
	}
	target := *recommendation.TargetCommand
	if target.Plugin != ownerPlugin {
		addReviewFinding(report, fmt.Sprintf("operation %s target command plugin %s does not match tracked owner %s for %s", operation.ID, target.Plugin, ownerPlugin, targetLabel))
		return
	}
	bundle, err := plugin.LoadBundle(filepath.Join(workspace.Root, "plugins", ownerPlugin), coreversion.Version)
	if err != nil {
		addReviewFinding(report, fmt.Sprintf("operation %s target command %s has no valid promoted plugin %s", operation.ID, commandTargetLabel(target), ownerPlugin))
		return
	}
	resolvedBundle, resolvedCommand, ok := plugin.FindCommandTarget([]plugin.Bundle{bundle}, target)
	if !ok {
		addReviewFinding(report, fmt.Sprintf("operation %s target command %s does not resolve in promoted plugin %s", operation.ID, commandTargetLabel(target), ownerPlugin))
		return
	}
	targetOperation, ok := resolvedBundle.APIs.Operations[resolvedCommand.Operation]
	if !ok || targetOperation.Method != recommendation.TargetAPI.Method || targetOperation.Path != recommendation.TargetAPI.Path {
		got := "no API operation"
		if ok {
			got = targetOperation.Method + " " + targetOperation.Path
		}
		addReviewFinding(report, fmt.Sprintf("operation %s target command %s maps to %s, want %s", operation.ID, commandTargetLabel(target), got, targetLabel))
		return
	}
	if plugin.CommandIsDeprecated(resolvedBundle, resolvedCommand) {
		addReviewFinding(report, fmt.Sprintf("operation %s target command %s is deprecated", operation.ID, commandTargetLabel(target)))
	}
}

// reviewGeneratedRecommendation requires the draft command to contain exactly
// the help metadata derived from the source operation.
func reviewGeneratedRecommendation(report *ReviewReport, operation Operation, command plugin.Command) {
	if command.ID == "" {
		return
	}
	expected := generatedRecommendation(operation)
	if recommendationMetadataEqual(command.Recommendation, expected) {
		return
	}
	addReviewFinding(report, fmt.Sprintf("operation %s generated recommendation does not match source target command", operation.ID))
}

// recommendationMetadataEqual compares the complete narrow command target
// representation emitted into generated plugin metadata.
func recommendationMetadataEqual(left, right *plugin.Recommendation) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.TargetCommand.Plugin == right.TargetCommand.Plugin && slices.Equal(left.TargetCommand.Path, right.TargetCommand.Path)
}

// recommendationGraphCycles reports deterministic cycles between tracked API
// identities. Each cycle repeats its starting API as the final element.
func recommendationGraphCycles(catalogs []Catalog) [][]string {
	graph := make(map[string][]string)
	for _, catalog := range catalogs {
		for _, operation := range catalog.Operations {
			if operation.Recommendation == nil {
				continue
			}
			source := operation.Method + " " + operation.Path
			target := recommendationAPIKey(operation.Recommendation.TargetAPI)
			graph[source] = append(graph[source], target)
			if _, ok := graph[target]; !ok {
				graph[target] = nil
			}
		}
	}
	keys := make([]string, 0, len(graph))
	for key, targets := range graph {
		sort.Strings(targets)
		graph[key] = slices.Compact(targets)
		keys = append(keys, key)
	}
	sort.Strings(keys)

	state := make(map[string]uint8, len(graph))
	position := make(map[string]int, len(graph))
	stack := make([]string, 0, len(graph))
	cyclesByKey := make(map[string][]string)
	var visit func(string)
	visit = func(node string) {
		state[node] = 1
		position[node] = len(stack)
		stack = append(stack, node)
		for _, target := range graph[node] {
			switch state[target] {
			case 0:
				visit(target)
			case 1:
				cycle := append(slices.Clone(stack[position[target]:]), target)
				cycle = canonicalRecommendationCycle(cycle)
				cyclesByKey[strings.Join(cycle, "\x00")] = cycle
			}
		}
		stack = stack[:len(stack)-1]
		delete(position, node)
		state[node] = 2
	}
	for _, key := range keys {
		if state[key] == 0 {
			visit(key)
		}
	}
	cycleKeys := make([]string, 0, len(cyclesByKey))
	for key := range cyclesByKey {
		cycleKeys = append(cycleKeys, key)
	}
	sort.Strings(cycleKeys)
	cycles := make([][]string, 0, len(cycleKeys))
	for _, key := range cycleKeys {
		cycles = append(cycles, cyclesByKey[key])
	}
	return cycles
}

// canonicalRecommendationCycle rotates a directed cycle to start at its
// lexicographically smallest API key.
func canonicalRecommendationCycle(cycle []string) []string {
	nodes := cycle[:len(cycle)-1]
	start := 0
	for index := 1; index < len(nodes); index++ {
		if nodes[index] < nodes[start] {
			start = index
		}
	}
	canonical := make([]string, 0, len(cycle))
	canonical = append(canonical, nodes[start:]...)
	canonical = append(canonical, nodes[:start]...)
	return append(canonical, canonical[0])
}

// recommendationAPIKey formats an upstream API identity for graph and review
// output.
func recommendationAPIKey(target APIReference) string {
	return target.Method + " " + target.Path
}
