/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestReviewDraftRejectsInvalidLabels verifies every objective label defect
// blocks an otherwise review-ready generated draft.
func TestReviewDraftRejectsInvalidLabels(t *testing.T) {
	tests := []struct {
		name     string
		language string
		label    string
		want     string
	}{
		{"raw camel case", "zh-CN", "sharePathV6", "raw identifier"},
		{"raw snake case", "zh-CN", "share_path", "raw identifier"},
		{"ordinary English", "zh-CN", "Created Time", "untranslated English"},
		{"URL", "zh-CN", "https://example.invalid/label", "URL"},
		{"prose", "zh-CN", "您可以查看接口文档获取该值", "prose"},
		{"unknown mixed word", "zh-CN", "资源 Custom 状态", "unknown ASCII word Custom"},
		{"noncanonical English", "en-US", "Cmk UUID", "non-canonical technical casing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := Workspace{Root: t.TempDir()}
			catalog := loadCatalogFixture(t)
			column := &catalog.Operations[0].Response.Columns[0]
			if test.language == "zh-CN" {
				column.LabelZH = test.label
			} else {
				column.LabelEN = test.label
			}
			writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)

			report, err := workspace.ReviewDraft("ecs")
			if err != nil {
				t.Fatalf("ReviewDraft returned error: %v", err)
			}
			if report.Ready {
				t.Fatalf("ReviewDraft accepted %s label %q", test.language, test.label)
			}
			if !slices.ContainsFunc(report.Findings, func(finding string) bool {
				return strings.Contains(finding, "operation v4.ecs.instance.list response column instanceID") &&
					strings.Contains(finding, test.language) && strings.Contains(finding, test.want)
			}) {
				t.Fatalf("findings %#v do not contain %q", report.Findings, test.want)
			}
		})
	}
}

// TestReviewDraftReportsMissingLabelInputs covers required table and localized
// help files that must exist before label review can begin.
func TestReviewDraftReportsMissingLabelInputs(t *testing.T) {
	for _, relativePath := range []string{"tables.json", filepath.Join("i18n", "zh-CN.json")} {
		t.Run(relativePath, func(t *testing.T) {
			workspace := Workspace{Root: t.TempDir()}
			writeCatalogAndGenerateDraft(t, workspace, "ecs", loadCatalogFixture(t))
			if err := os.Remove(workspace.ProductPath("ecs", "draft", relativePath)); err != nil {
				t.Fatalf("remove draft input: %v", err)
			}
			if _, err := workspace.ReviewDraft("ecs"); err == nil {
				t.Fatal("ReviewDraft returned nil for a missing draft input")
			}
		})
	}
}

// TestReviewDraftAcceptsLocalizedAndTechnicalLabels verifies the shared policy
// permits localized labels and canonical language-independent technical text.
func TestReviewDraftAcceptsLocalizedAndTechnicalLabels(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := loadCatalogFixture(t)
	columns := catalog.Operations[0].Response.Columns
	columns[0].LabelEN, columns[0].LabelZH = "HPFS Share Path", "HPFS 文件系统共享路径"
	columns[1].LabelEN, columns[1].LabelZH = "VPC FUID", "VPC FUID"
	columns[2].LabelEN, columns[2].LabelZH = "ETag", "ETag"
	columns[3].LabelEN, columns[3].LabelZH = "Windows Share Path", "Windows 共享路径"
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)

	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("ReviewDraft = %#v, %v", report, err)
	}
}

// TestReviewDraftRejectsSourceLabelDisagreement verifies a draft cannot hide
// localized source evidence behind a different visible label.
func TestReviewDraftRejectsSourceLabelDisagreement(t *testing.T) {
	workspace := Workspace{Root: t.TempDir()}
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].Response.Columns[0].LabelZH = "加入 AD 域的操作号"
	writeCatalogAndGenerateDraft(t, workspace, "ecs", catalog)

	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	tables := readJSONFile[plugin.Tables](t, workspace.ProductPath("ecs", "draft", "tables.json"))
	command := slices.IndexFunc(commands.Commands, func(command plugin.Command) bool {
		return command.Operation == "v4.ecs.instance.list"
	})
	if command < 0 {
		t.Fatal("generated command not found")
	}
	table := tables.Tables[commands.Commands[command].Table]
	table.Columns[0].Labels["zh-CN"] = "Operation ID"
	tables.Tables[commands.Commands[command].Table] = table
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "tables.json"), tables); err != nil {
		t.Fatalf("write drifted tables: %v", err)
	}

	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatalf("ReviewDraft returned error: %v", err)
	}
	if report.Ready || !slices.ContainsFunc(report.Findings, func(finding string) bool {
		return strings.Contains(finding, "source label disagreement")
	}) {
		t.Fatalf("ReviewDraft accepted source label drift: %#v", report.Findings)
	}
}

// TestReviewTableLabelsRejectsStructuralDrift verifies missing tables, missing
// columns, and duplicate columns are independently review-blocking.
func TestReviewTableLabelsRejectsStructuralDrift(t *testing.T) {
	catalog := loadCatalogFixture(t)
	operation := catalog.Operations[0]
	command := plugin.Command{Operation: operation.ID, Table: "ecs.instance.list"}
	tests := []struct {
		name   string
		tables plugin.Tables
		want   string
	}{
		{"missing table", plugin.Tables{Tables: map[string]plugin.Table{}}, "table ecs.instance.list is missing"},
		{"missing column", plugin.Tables{Tables: map[string]plugin.Table{"ecs.instance.list": {}}}, "has no generated table column"},
		{"duplicate column", plugin.Tables{Tables: map[string]plugin.Table{"ecs.instance.list": {Columns: []plugin.TableColumn{
			{Key: operation.Response.Columns[0].Key, Path: operation.Response.Columns[0].Path},
			{Key: operation.Response.Columns[0].Key, Path: operation.Response.Columns[0].Path},
		}}}}, "has duplicate generated table columns"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := ReviewReport{Ready: true}
			reviewTableLabels(&report, Catalog{Operations: []Operation{operation}}, plugin.Commands{Commands: []plugin.Command{command}}, test.tables)
			if report.Ready || !slices.ContainsFunc(report.Findings, func(finding string) bool { return strings.Contains(finding, test.want) }) {
				t.Fatalf("findings = %#v, want %q", report.Findings, test.want)
			}
		})
	}

	report := ReviewReport{Ready: true}
	reviewSourceColumnLabels(&report, operation.ID, Column{LabelZH: "状态"})
	if !report.Ready || len(report.Findings) != 0 {
		t.Fatalf("empty optional source label produced findings: %#v", report.Findings)
	}
}
