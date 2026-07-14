/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestRepoPluginRecommendationsResolve verifies relationships between loaded
// repository plugin commands before their bundles are released.
func TestRepoPluginRecommendationsResolve(t *testing.T) {
	for _, problem := range recommendationProblems(loadRepoBundles(t)) {
		t.Error(problem)
	}
}

// TestRecommendationProblemsAllowMissingPlugin preserves soft dependencies
// on recommendation targets that are not part of the loaded bundle set.
func TestRecommendationProblemsAllowMissingPlugin(t *testing.T) {
	bundles := []plugin.Bundle{
		recommendationBundle("ecs", recommendationCommand("ecs.metric.history", []string{"ecs", "metric", "history"}, "monitor", []string{"monitor", "metric", "history"})),
	}

	if problems := recommendationProblems(bundles); len(problems) != 0 {
		t.Fatalf("recommendationProblems() = %#v, want no problem for unloaded plugin", problems)
	}
}

// TestRecommendationProblemsRejectWrongLoadedPlugin verifies that a command
// path in another loaded plugin does not satisfy the declared plugin target.
func TestRecommendationProblemsRejectWrongLoadedPlugin(t *testing.T) {
	source := recommendationBundle("ecs", recommendationCommand("ecs.metric.history", []string{"ecs", "metric", "history"}, "ecs", []string{"monitor", "metric", "history"}))
	target := recommendationBundle("monitor", plugin.Command{ID: "monitor.metric.history", Path: []string{"monitor", "metric", "history"}})

	problems := recommendationProblems([]plugin.Bundle{target, source})
	if len(problems) != 1 || !strings.Contains(problems[0], "unresolved recommendation") {
		t.Fatalf("recommendationProblems() = %#v, want wrong-plugin problem", problems)
	}
}

// TestRecommendationProblemsRejectStalePath verifies that a loaded target
// plugin must contain the exact declared visible command path.
func TestRecommendationProblemsRejectStalePath(t *testing.T) {
	source := recommendationBundle("ecs", recommendationCommand("ecs.metric.history", []string{"ecs", "metric", "history"}, "monitor", []string{"monitor", "metric", "stale"}))
	target := recommendationBundle("monitor", plugin.Command{ID: "monitor.metric.history", Path: []string{"monitor", "metric", "history"}})

	problems := recommendationProblems([]plugin.Bundle{source, target})
	if len(problems) != 1 || !strings.Contains(problems[0], "unresolved recommendation") {
		t.Fatalf("recommendationProblems() = %#v, want stale-path problem", problems)
	}
}

// TestRecommendationProblemsRejectDeprecatedTarget verifies that visible
// recommendations never direct users to deprecated commands.
func TestRecommendationProblemsRejectDeprecatedTarget(t *testing.T) {
	source := recommendationBundle("ecs", recommendationCommand("ecs.metric.history", []string{"ecs", "metric", "history"}, "monitor", []string{"monitor", "metric", "history"}))
	targetCommand := plugin.Command{
		ID:          "monitor.metric.history",
		Path:        []string{"monitor", "metric", "history"},
		Deprecation: &plugin.Deprecation{Status: "deprecated"},
	}
	target := recommendationBundle("monitor", targetCommand)

	problems := recommendationProblems([]plugin.Bundle{source, target})
	if len(problems) != 1 || !strings.Contains(problems[0], "recommends deprecated command") {
		t.Fatalf("recommendationProblems() = %#v, want deprecated-target problem", problems)
	}
}

// TestRecommendationCyclesRejectTwoCommandCycle verifies deterministic cycle
// reporting across commands in different loaded plugins.
func TestRecommendationCyclesRejectTwoCommandCycle(t *testing.T) {
	left := recommendationBundle("ecs", recommendationCommand("ecs.metric.history", []string{"ecs", "metric", "history"}, "monitor", []string{"monitor", "metric", "history"}))
	right := recommendationBundle("monitor", recommendationCommand("monitor.metric.history", []string{"monitor", "metric", "history"}, "ecs", []string{"ecs", "metric", "history"}))
	want := []string{"ecs:ecs metric history", "monitor:monitor metric history", "ecs:ecs metric history"}

	cycles := recommendationCycles([]plugin.Bundle{right, left})
	if len(cycles) != 1 || !slices.Equal(cycles[0], want) {
		t.Fatalf("recommendationCycles() = %#v, want %#v", cycles, [][]string{want})
	}
	problems := recommendationProblems([]plugin.Bundle{right, left})
	if len(problems) != 1 || !strings.Contains(problems[0], "recommendation cycle") {
		t.Fatalf("recommendationProblems() = %#v, want cycle problem", problems)
	}
}

// recommendationBundle builds the narrow bundle shape used by graph tests.
func recommendationBundle(name string, commands ...plugin.Command) plugin.Bundle {
	return plugin.Bundle{
		Manifest: plugin.Manifest{Name: name},
		Commands: plugin.Commands{Commands: commands},
		APIs:     plugin.APIs{Operations: map[string]plugin.Operation{}},
	}
}

// recommendationCommand builds a command with one visible recommendation.
func recommendationCommand(id string, path []string, targetPlugin string, targetPath []string) plugin.Command {
	return plugin.Command{
		ID:   id,
		Path: path,
		Recommendation: &plugin.Recommendation{TargetCommand: plugin.CommandTarget{
			Plugin: targetPlugin,
			Path:   targetPath,
		}},
	}
}

// loadRepoBundles loads repository plugins in stable manifest-name order.
func loadRepoBundles(t *testing.T) []plugin.Bundle {
	t.Helper()
	pluginDirectories := pluginDirs(t, repoPath(t, "plugins"))
	sort.Slice(pluginDirectories, func(left, right int) bool {
		return filepath.Base(pluginDirectories[left]) < filepath.Base(pluginDirectories[right])
	})
	bundles := make([]plugin.Bundle, 0, len(pluginDirectories))
	for _, pluginDirectory := range pluginDirectories {
		bundle, err := plugin.LoadBundle(pluginDirectory, version.Version)
		if err != nil {
			t.Fatalf("load plugin %s: %v", filepath.Base(pluginDirectory), err)
		}
		bundles = append(bundles, bundle)
	}
	return sortedRecommendationBundles(bundles)
}

// recommendationProblems reports invalid relationships between loaded plugin
// commands while allowing targets whose plugin is not loaded.
func recommendationProblems(bundles []plugin.Bundle) []string {
	bundles = sortedRecommendationBundles(bundles)
	loadedPlugins := make(map[string]bool, len(bundles))
	for _, bundle := range bundles {
		loadedPlugins[bundle.Manifest.Name] = true
	}

	var problems []string
	for _, bundle := range bundles {
		for _, command := range sortedRecommendationCommands(bundle.Commands.Commands) {
			if !command.Recommendation.Active() {
				continue
			}
			target := command.Recommendation.TargetCommand
			if !loadedPlugins[target.Plugin] {
				continue
			}
			targetBundle, targetCommand, ok := plugin.FindCommandTarget(bundles, target)
			if !ok {
				problems = append(problems, fmt.Sprintf("plugin %s command %s has unresolved recommendation %s", bundle.Manifest.Name, command.ID, commandTargetKey(target)))
				continue
			}
			if plugin.CommandIsDeprecated(targetBundle, targetCommand) {
				problems = append(problems, fmt.Sprintf("plugin %s command %s recommends deprecated command %s", bundle.Manifest.Name, command.ID, targetCommand.ID))
			}
		}
	}
	for _, cycle := range recommendationCycles(bundles) {
		problems = append(problems, fmt.Sprintf("plugin recommendation cycle: %s", strings.Join(cycle, " -> ")))
	}
	sort.Strings(problems)
	return problems
}

// recommendationCycles reports deterministic cycles between exact visible
// command paths in loaded plugin bundles.
func recommendationCycles(bundles []plugin.Bundle) [][]string {
	bundles = sortedRecommendationBundles(bundles)
	graph := make(map[string][]string)
	for _, bundle := range bundles {
		for _, command := range sortedRecommendationCommands(bundle.Commands.Commands) {
			if !command.Recommendation.Active() {
				continue
			}
			targetBundle, targetCommand, ok := plugin.FindCommandTarget(bundles, command.Recommendation.TargetCommand)
			if !ok {
				continue
			}
			source := commandKey(bundle.Manifest.Name, command.Path)
			target := commandKey(targetBundle.Manifest.Name, targetCommand.Path)
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

	cyclesByKey := make(map[string][]string)
	for _, start := range keys {
		path := []string{start}
		inPath := map[string]bool{start: true}
		var visit func(string)
		visit = func(node string) {
			for _, target := range graph[node] {
				if target == start {
					cycle := canonicalRecommendationCycle(append(slices.Clone(path), start))
					cyclesByKey[strings.Join(cycle, "\x00")] = cycle
					continue
				}
				if target < start || inPath[target] {
					continue
				}
				inPath[target] = true
				path = append(path, target)
				visit(target)
				path = path[:len(path)-1]
				delete(inPath, target)
			}
		}
		visit(start)
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

// sortedRecommendationBundles returns a stable copy ordered by plugin name.
func sortedRecommendationBundles(bundles []plugin.Bundle) []plugin.Bundle {
	ordered := slices.Clone(bundles)
	sort.SliceStable(ordered, func(left, right int) bool {
		if ordered[left].Manifest.Name != ordered[right].Manifest.Name {
			return ordered[left].Manifest.Name < ordered[right].Manifest.Name
		}
		return ordered[left].Dir < ordered[right].Dir
	})
	return ordered
}

// sortedRecommendationCommands returns a stable copy ordered by declared path
// and then by internal command ID for deterministic release diagnostics.
func sortedRecommendationCommands(commands []plugin.Command) []plugin.Command {
	ordered := slices.Clone(commands)
	sort.SliceStable(ordered, func(left, right int) bool {
		leftPath := strings.Join(ordered[left].Path, "\x00")
		rightPath := strings.Join(ordered[right].Path, "\x00")
		if leftPath != rightPath {
			return leftPath < rightPath
		}
		return ordered[left].ID < ordered[right].ID
	})
	return ordered
}

// canonicalRecommendationCycle rotates a closed cycle to its smallest stable
// command key while retaining the repeated terminal node.
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

// commandKey formats a stable graph identity from plugin name and declared
// visible command path.
func commandKey(pluginName string, path []string) string {
	return pluginName + ":" + strings.Join(path, " ")
}

// commandTargetKey formats a declared target for release diagnostics.
func commandTargetKey(target plugin.CommandTarget) string {
	return commandKey(target.Plugin, target.Path)
}
