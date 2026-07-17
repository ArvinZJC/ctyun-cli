/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Upstream-only recommendation values ensure help does not leak ignored metadata.
const (
	recommendationNoticeForTest = "use the upstream replacement API"
	recommendationPathForTest   = "/v4/monitor/private/metric-history"
	recommendationDocsForTest   = "https://example.test/upstream/recommendation"
)

// recommendationBundleOptions controls the temporary source and target bundles.
type recommendationBundleOptions struct {
	includeTarget            bool
	targetCommandPath        []string
	recommendationTargetPath []string
	recommendationScope      string
	localizedScopes          map[string]string
	deprecateSourceCommand   bool
	deprecateTargetCommand   bool
	deprecateTargetOperation bool
}

// TestRecommendationHelpQualifiesApplicability localizes the condition that
// narrows when the alternative command should be used.
func TestRecommendationHelpQualifiesApplicability(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	wants := map[string]string{
		"en-US": "For physical-machine images, use: ctyun monitor metric history",
		"en-GB": "For physical-machine images, use: ctyun monitor metric history",
		"zh-CN": "如需查询物理机镜像，请使用：ctyun monitor metric history",
	}
	for language, want := range wants {
		t.Run(language, func(t *testing.T) {
			root := t.TempDir()
			writeRecommendationBundles(t, root, recommendationBundleOptions{
				includeTarget:       true,
				recommendationScope: "physical-machine images",
				localizedScopes: map[string]string{
					"en-US": "physical-machine images",
					"en-GB": "physical-machine images",
					"zh-CN": "物理机镜像",
				},
			})

			got := recommendationHelpOutput(t, root, language)
			if !strings.Contains(got, want) {
				t.Fatalf("qualified help output missing %q:\n%s", want, got)
			}
			if strings.Contains(got, "Recommended alternative:") || strings.Contains(got, "推荐替代命令：") {
				t.Fatalf("qualified help retained unconditional wording:\n%s", got)
			}
		})
	}
}

// TestRecommendationHelpApplicabilityFallsBackToMetadata keeps qualified help
// usable when an optional plugin i18n entry is absent.
func TestRecommendationHelpApplicabilityFallsBackToMetadata(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	root := t.TempDir()
	writeRecommendationBundles(t, root, recommendationBundleOptions{
		includeTarget:       true,
		recommendationScope: "physical-machine images",
	})

	got := recommendationHelpOutput(t, root, "en-US")
	want := "For physical-machine images, use: ctyun monitor metric history"
	if !strings.Contains(got, want) {
		t.Fatalf("fallback qualified help output missing %q:\n%s", want, got)
	}
}

// TestRecommendationHelpRequiresLoadedVisibleCommand checks exact loaded-target resolution.
func TestRecommendationHelpRequiresLoadedVisibleCommand(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	root := t.TempDir()
	writeRecommendationBundles(t, root, recommendationBundleOptions{})

	withoutTarget := recommendationHelpOutput(t, root, "en-US")
	if strings.Contains(withoutTarget, "Recommended alternative:") {
		t.Fatalf("help rendered recommendation without loaded target:\n%s", withoutTarget)
	}

	writeRecommendationTargetBundle(t, root, recommendationBundleOptions{includeTarget: true})
	withTarget := recommendationHelpOutput(t, root, "en-US")
	want := "Recommended alternative: ctyun monitor metric history"
	if !strings.Contains(withTarget, want) {
		t.Fatalf("help output missing %q:\n%s", want, withTarget)
	}
}

// TestRecommendationHelpLocalizesLabel checks the help-only prose in every supported locale.
func TestRecommendationHelpLocalizesLabel(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	wants := map[string]string{
		"en-US": "Recommended alternative: ctyun monitor metric history",
		"en-GB": "Recommended alternative: ctyun monitor metric history",
		"zh-CN": "推荐替代命令：ctyun monitor metric history",
	}
	for language, want := range wants {
		t.Run(language, func(t *testing.T) {
			root := t.TempDir()
			writeRecommendationBundles(t, root, recommendationBundleOptions{includeTarget: true})

			got := recommendationHelpOutput(t, root, language)
			if !strings.Contains(got, want) {
				t.Fatalf("localized help output missing %q:\n%s", want, got)
			}
		})
	}
}

// TestRecommendationHelpOmitsMissingOrDeprecatedTarget checks soft target failures.
func TestRecommendationHelpOmitsMissingOrDeprecatedTarget(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	cases := []struct {
		name    string
		options recommendationBundleOptions
	}{
		{
			name: "missing exact command",
			options: recommendationBundleOptions{
				includeTarget:     true,
				targetCommandPath: []string{"monitor", "metric", "current"},
			},
		},
		{
			name:    "deprecated target command",
			options: recommendationBundleOptions{includeTarget: true, deprecateTargetCommand: true},
		},
		{
			name:    "deprecated target operation",
			options: recommendationBundleOptions{includeTarget: true, deprecateTargetOperation: true},
		},
		{
			name:    "deprecated source command",
			options: recommendationBundleOptions{includeTarget: true, deprecateSourceCommand: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeRecommendationBundles(t, root, tc.options)

			got := recommendationHelpOutput(t, root, "en-US")
			if strings.Contains(got, "Recommended alternative:") {
				t.Fatalf("help rendered unavailable recommendation:\n%s", got)
			}
		})
	}
}

// TestRecommendationHelpDoesNotExposeUpstreamMetadata checks the visible-command-only contract.
func TestRecommendationHelpDoesNotExposeUpstreamMetadata(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	root := t.TempDir()
	writeRecommendationBundles(t, root, recommendationBundleOptions{includeTarget: true})

	got := recommendationHelpOutput(t, root, "en-US")
	for _, unwanted := range []string{
		recommendationNoticeForTest,
		recommendationPathForTest,
		recommendationDocsForTest,
		"ecs.metric.history.operation",
		"monitor.metric.history.operation",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("help exposed upstream metadata %q:\n%s", unwanted, got)
		}
	}
}

// TestRecommendationHelpPreservesPositionalPlaceholders treats the command path as data.
func TestRecommendationHelpPreservesPositionalPlaceholders(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	root := t.TempDir()
	targetPath := []string{"monitor", "metric", "{metric_id}", "history"}
	writeRecommendationBundles(t, root, recommendationBundleOptions{
		includeTarget:            true,
		targetCommandPath:        targetPath,
		recommendationTargetPath: targetPath,
	})

	got := recommendationHelpOutput(t, root, "en-US")
	want := "Recommended alternative: ctyun monitor metric {metric_id} history"
	if !strings.Contains(got, want) {
		t.Fatalf("help output missing %q:\n%s", want, got)
	}
}

// TestRecommendationCommandExecutionWritesNoNotice keeps recommendations out of runtime output.
func TestRecommendationCommandExecutionWritesNoNotice(t *testing.T) {
	disableDevelopmentBundledPluginsForTest(t)
	for _, warnDeprecated := range []bool{true, false} {
		t.Run(strconv.FormatBool(warnDeprecated), func(t *testing.T) {
			root := t.TempDir()
			writeRecommendationBundles(t, root, recommendationBundleOptions{includeTarget: true})

			var stderr bytes.Buffer
			err := Run(Config{
				Args:       []string{"--lang", "en-US", "ecs", "metric", "history", "--offline"},
				Stdout:     io.Discard,
				Stderr:     &stderr,
				PluginRoot: root,
				Config:     []byte(`{"warn_deprecated": ` + strconv.FormatBool(warnDeprecated) + `}`),
				Env:        func(string) string { return "" },
			})
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if stderr.Len() != 0 {
				t.Fatalf("command execution wrote recommendation notice: %q", stderr.String())
			}
		})
	}
}

// recommendationHelpOutput renders source-command help from temporary bundles.
func recommendationHelpOutput(t *testing.T, root, language string) string {
	t.Helper()
	var stdout bytes.Buffer
	if err := Run(Config{
		Args:       []string{"--lang", language, "help", "ecs", "metric", "history"},
		Stdout:     &stdout,
		PluginRoot: root,
		Env:        func(string) string { return "" },
	}); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	return stdout.String()
}

// writeRecommendationBundles writes the source bundle and its optional target.
func writeRecommendationBundles(t *testing.T, root string, options recommendationBundleOptions) {
	t.Helper()
	writeRecommendationSourceBundle(t, root, options)
	if options.includeTarget {
		writeRecommendationTargetBundle(t, root, options)
	}
}

// writeRecommendationSourceBundle writes source command metadata with ignored upstream fields.
func writeRecommendationSourceBundle(t *testing.T, root string, options recommendationBundleOptions) {
	t.Helper()
	targetPath := options.recommendationTargetPath
	if len(targetPath) == 0 {
		targetPath = []string{"monitor", "metric", "history"}
	}
	command := map[string]any{
		"id":               "ecs.metric.history",
		"path":             []string{"ecs", "metric", "history"},
		"operation":        "ecs.metric.history.operation",
		"table":            "ecs.metric.history",
		"fixture_response": "fixtures/history.json",
		"recommendation": map[string]any{
			"notice":     recommendationNoticeForTest,
			"target_api": map[string]any{"path": recommendationPathForTest},
			"docs_url":   recommendationDocsForTest,
			"target_command": map[string]any{
				"plugin": "monitor",
				"path":   targetPath,
			},
		},
	}
	if options.recommendationScope != "" {
		command["recommendation"].(map[string]any)["applicability"] = options.recommendationScope
	}
	if options.deprecateSourceCommand {
		command["deprecation"] = recommendationDeprecationForTest()
	}
	writeRecommendationBundle(t, filepath.Join(root, "ecs"), "ecs", "Elastic Cloud Server", command, false, options.localizedScopes)
}

// writeRecommendationTargetBundle writes one visible target command for resolution.
func writeRecommendationTargetBundle(t *testing.T, root string, options recommendationBundleOptions) {
	t.Helper()
	targetPath := options.targetCommandPath
	if len(targetPath) == 0 {
		targetPath = []string{"monitor", "metric", "history"}
	}
	command := map[string]any{
		"id":               "monitor.metric.history",
		"path":             targetPath,
		"operation":        "monitor.metric.history.operation",
		"table":            "monitor.metric.history",
		"fixture_response": "fixtures/history.json",
	}
	if options.deprecateTargetCommand {
		command["deprecation"] = recommendationDeprecationForTest()
	}
	writeRecommendationBundle(t, filepath.Join(root, "monitor"), "monitor", "Cloud Monitor", command, options.deprecateTargetOperation, nil)
}

// writeRecommendationBundle writes one complete, loadable plugin bundle.
func writeRecommendationBundle(t *testing.T, dir, name, product string, command map[string]any, deprecateOperation bool, localizedScopes map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create recommendation fixture dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "i18n"), 0o755); err != nil {
		t.Fatalf("create recommendation i18n dir: %v", err)
	}
	writeRecommendationJSON(t, filepath.Join(dir, "plugin.json"), map[string]any{
		"name":    name,
		"version": "0.1.0",
		"channel": "stable",
		"quality": "generated",
		"requires": map[string]any{
			"ctyun": testCompatibleCoreConstraint(),
		},
		"api": map[string]any{
			"product":          name,
			"ctyun_product_id": 25,
			"endpoint_url":     "https://ctapi.example.test",
		},
	})
	writeRecommendationJSON(t, filepath.Join(dir, "commands.json"), map[string]any{"commands": []any{command}})
	operation := map[string]any{
		"method":       "GET",
		"path":         "/v4/" + name + "/metric/history",
		"content_type": "application/json",
	}
	if deprecateOperation {
		operation["deprecation"] = recommendationDeprecationForTest()
	}
	writeRecommendationJSON(t, filepath.Join(dir, "apis.json"), map[string]any{
		"operations": map[string]any{command["operation"].(string): operation},
	})
	writeRecommendationJSON(t, filepath.Join(dir, "tables.json"), map[string]any{
		"tables": map[string]any{
			command["table"].(string): map[string]any{
				"row_path": "returnObj.items",
				"columns": []any{
					map[string]any{
						"key":  "id",
						"path": "id",
						"labels": map[string]any{
							"en-US": "ID",
							"en-GB": "ID",
							"zh-CN": "ID",
						},
					},
				},
			},
		},
	})
	writeRecommendationJSON(t, filepath.Join(dir, "fixtures", "history.json"), map[string]any{
		"returnObj": map[string]any{"items": []any{map[string]any{"id": "metric-1"}}},
	})
	descriptionKey := "command." + command["id"].(string) + ".description"
	for language, values := range map[string]map[string]string{
		"en-US": {"name": product, descriptionKey: "Show metric history"},
		"en-GB": {"name": product, descriptionKey: "Show metric history"},
		"zh-CN": {"name": product, descriptionKey: "显示指标历史"},
	} {
		if scope := localizedScopes[language]; scope != "" {
			values["recommendation."+command["id"].(string)+".applicability"] = scope
		}
		writeRecommendationJSON(t, filepath.Join(dir, "i18n", language+".json"), values)
	}
}

// writeRecommendationJSON serializes one temporary bundle metadata file.
func writeRecommendationJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal recommendation fixture %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write recommendation fixture %s: %v", path, err)
	}
}

// recommendationDeprecationForTest returns active upstream deprecation metadata.
func recommendationDeprecationForTest() map[string]any {
	return map[string]any{"status": "deprecated", "notice": "upstream deprecation notice"}
}
