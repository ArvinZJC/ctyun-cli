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

// recommendationTargetAmbiguity records one HTTP API identity that names more
// than one distinct recommendation target across tracked catalogs.
type recommendationTargetAmbiguity struct {
	Source  string
	Targets []string
}

// recommendationGraphReview contains deterministic graph validation results
// for tracked API recommendations.
type recommendationGraphReview struct {
	Ambiguities []recommendationTargetAmbiguity
	Cycles      [][]string
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
	if operationHasDeprecationText(owners[0].Operation) {
		addReviewFinding(report, fmt.Sprintf("operation %s recommendation target %s is deprecated in current tracked source", operation.ID, targetLabel))
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

// analyzeRecommendationGraph rejects branching API identities and reports
// every cycle in the remaining functional graph once. Each cycle repeats its
// starting API as the final element.
func analyzeRecommendationGraph(catalogs []Catalog) recommendationGraphReview {
	targetSets := make(map[string]map[string]struct{})
	for _, catalog := range catalogs {
		for _, operation := range catalog.Operations {
			if operation.Recommendation == nil {
				continue
			}
			source := operation.Method + " " + operation.Path
			target := recommendationAPIKey(operation.Recommendation.TargetAPI)
			if targetSets[source] == nil {
				targetSets[source] = make(map[string]struct{})
			}
			targetSets[source][target] = struct{}{}
		}
	}

	sources := make([]string, 0, len(targetSets))
	for source := range targetSets {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	review := recommendationGraphReview{}
	edges := make(map[string]string, len(sources))
	for _, source := range sources {
		targets := make([]string, 0, len(targetSets[source]))
		for target := range targetSets[source] {
			targets = append(targets, target)
		}
		sort.Strings(targets)
		if len(targets) > 1 {
			review.Ambiguities = append(review.Ambiguities, recommendationTargetAmbiguity{Source: source, Targets: targets})
			continue
		}
		edges[source] = targets[0]
	}

	const (
		graphUnvisited uint8 = iota
		graphVisiting
		graphDone
	)
	states := make(map[string]uint8, len(edges))
	for _, start := range sources {
		if _, ok := edges[start]; !ok || states[start] != graphUnvisited {
			continue
		}
		path := make([]string, 0)
		pathIndexes := make(map[string]int)
		node := start
		for states[node] == graphUnvisited {
			states[node] = graphVisiting
			pathIndexes[node] = len(path)
			path = append(path, node)
			target, ok := edges[node]
			if !ok {
				node = ""
				break
			}
			node = target
		}
		if states[node] == graphVisiting {
			if index, ok := pathIndexes[node]; ok {
				cycle := append(slices.Clone(path[index:]), node)
				review.Cycles = append(review.Cycles, canonicalRecommendationCycle(cycle))
			}
		}
		for _, visited := range path {
			states[visited] = graphDone
		}
	}
	sort.Slice(review.Cycles, func(left, right int) bool {
		return strings.Join(review.Cycles[left], "\x00") < strings.Join(review.Cycles[right], "\x00")
	})
	return review
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
