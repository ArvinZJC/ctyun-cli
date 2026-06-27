/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestCatalogValidationAcceptsFixture(t *testing.T) {
	catalog := loadCatalogFixture(t, "ecs-source.json")
	if err := catalog.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if catalog.Product.PluginName != "ecs" {
		t.Fatalf("plugin name = %q, want ecs", catalog.Product.PluginName)
	}
	if len(catalog.Operations) != 3 {
		t.Fatalf("operation count = %d, want 3", len(catalog.Operations))
	}
}

func TestCatalogValidationRejectsMissingRequiredFields(t *testing.T) {
	cases := []struct {
		name    string
		catalog Catalog
		want    string
	}{
		{name: "missing schema", catalog: Catalog{}, want: "schema_version is required"},
		{name: "missing plugin", catalog: Catalog{SchemaVersion: 1}, want: "product.plugin_name is required"},
		{name: "bad plugin", catalog: Catalog{SchemaVersion: 1, Product: Product{PluginName: "../ecs"}}, want: "product.plugin_name ../ecs is invalid"},
		{name: "missing operation id", catalog: Catalog{SchemaVersion: 1, Product: Product{PluginName: "ecs", APIProduct: "ecs", CtyunProductID: 25}, Operations: []Operation{{Method: "GET", Path: "/v4/demo"}}}, want: "operation id is required"},
		{name: "bad method", catalog: Catalog{SchemaVersion: 1, Product: Product{PluginName: "ecs", APIProduct: "ecs", CtyunProductID: 25}, Operations: []Operation{{ID: "op", Method: "TRACE", Path: "/v4/demo"}}}, want: "operation op method TRACE is unsupported"},
		{name: "bad path", catalog: Catalog{SchemaVersion: 1, Product: Product{PluginName: "ecs", APIProduct: "ecs", CtyunProductID: 25}, Operations: []Operation{{ID: "op", Method: "GET", Path: "v4/demo"}}}, want: "operation op path must start with /"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.catalog.Validate()
			if err == nil {
				t.Fatal("Validate returned nil error")
			}
			if err.Error() != tc.want {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestCatalogValidationRejectsAdditionalInvalidShapes(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Catalog)
		want   string
	}{
		{name: "missing api product", mutate: func(catalog *Catalog) { catalog.Product.APIProduct = "" }, want: "product.api_product is required"},
		{name: "missing product id", mutate: func(catalog *Catalog) { catalog.Product.CtyunProductID = 0 }, want: "product.ctyun_product_id is required"},
		{name: "duplicate operation", mutate: func(catalog *Catalog) { catalog.Operations = append(catalog.Operations, catalog.Operations[0]) }, want: "operation v4.ecs.instance.list is duplicated"},
		{name: "missing method", mutate: func(catalog *Catalog) { catalog.Operations[0].Method = "" }, want: "operation v4.ecs.instance.list method is required"},
		{name: "missing path", mutate: func(catalog *Catalog) { catalog.Operations[0].Path = "" }, want: "operation v4.ecs.instance.list path is required"},
		{name: "invalid path", mutate: func(catalog *Catalog) { catalog.Operations[0].Path = "/v4/../demo" }, want: "operation v4.ecs.instance.list path /v4/../demo is invalid"},
		{name: "missing parameter", mutate: func(catalog *Catalog) { catalog.Operations[0].Parameters[0].Name = "" }, want: "operation v4.ecs.instance.list parameter name is required"},
		{name: "unsupported parameter location", mutate: func(catalog *Catalog) { catalog.Operations[0].Parameters[0].Location = "cookie" }, want: "operation v4.ecs.instance.list parameter regionID location cookie is unsupported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			catalog := loadCatalogFixture(t, "ecs-source.json")
			tc.mutate(&catalog)
			err := catalog.Validate()
			if err == nil {
				t.Fatal("Validate returned nil error")
			}
			if err.Error() != tc.want {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestWorkspaceHarvestCopiesInputToSource(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	data, err := os.ReadFile(filepath.Join("testdata", "ecs-source.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(input, data, 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	workspace := Workspace{Root: root}
	if err := workspace.HarvestFromFile("ecs", input); err != nil {
		t.Fatalf("HarvestFromFile returned error: %v", err)
	}

	got := workspace.ProductPath("ecs", "source.json")
	catalog := readCatalogFile(t, got)
	if catalog.Product.PluginName != "ecs" {
		t.Fatalf("source plugin = %q, want ecs", catalog.Product.PluginName)
	}
}

func TestWorkspaceRejectsProductMismatch(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	data, err := os.ReadFile(filepath.Join("testdata", "ecs-source.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(input, data, 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	workspace := Workspace{Root: root}
	err = workspace.HarvestFromFile("region", input)
	if err == nil {
		t.Fatal("HarvestFromFile returned nil error")
	}
	if err.Error() != "input product ecs does not match requested product region" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestWorkspaceReportsReadWriteAndNoBaselineDiffCases(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	if err := workspace.HarvestFromFile("ecs", filepath.Join(root, "missing.json")); err == nil {
		t.Fatal("HarvestFromFile returned nil error for missing input")
	}
	if err := workspace.WriteCatalog(workspace.ProductPath("bad", "source.json"), Catalog{}); err == nil {
		t.Fatal("WriteCatalog returned nil error for invalid catalog")
	}
	if err := writeJSON(filepath.Join(root, "bad.json"), func() {}); err == nil {
		t.Fatal("writeJSON returned nil error for unmarshalable value")
	}
	parentFile := filepath.Join(root, "parent-file")
	if err := os.WriteFile(parentFile, []byte("file"), 0o644); err != nil {
		t.Fatalf("write parent file: %v", err)
	}
	if err := writeText(filepath.Join(parentFile, "child.md"), "text"); err == nil {
		t.Fatal("writeText returned nil error for parent file conflict")
	}
	invalidPath := filepath.Join(root, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte(`{`), 0o644); err != nil {
		t.Fatalf("write invalid json: %v", err)
	}
	if _, err := readCatalog(invalidPath); err == nil {
		t.Fatal("readCatalog returned nil error for invalid JSON")
	}
	invalidCatalogPath := filepath.Join(root, "invalid-catalog.json")
	if err := os.WriteFile(invalidCatalogPath, []byte(`{"schema_version":1}`), 0o644); err != nil {
		t.Fatalf("write invalid catalog: %v", err)
	}
	if _, err := readCatalog(invalidCatalogPath); err == nil {
		t.Fatal("readCatalog returned nil error for invalid catalog")
	}
	if _, err := workspace.ReadBaseline("ecs"); err == nil {
		t.Fatal("ReadBaseline returned nil error for missing baseline")
	}
	if _, err := workspace.WriteDiff("ecs"); err == nil {
		t.Fatal("WriteDiff returned nil error without source")
	}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t, "ecs-source.json")); err != nil {
		t.Fatalf("write source: %v", err)
	}
	report, err := workspace.WriteDiff("ecs")
	if err != nil {
		t.Fatalf("WriteDiff without baseline returned error: %v", err)
	}
	if !strings.Contains(report.Markdown(), "No baseline exists") {
		t.Fatalf("missing no-baseline text:\n%s", report.Markdown())
	}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "baseline.json"), loadCatalogFixture(t, "ecs-source.json")); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	report, err = workspace.WriteDiff("ecs")
	if err != nil {
		t.Fatalf("WriteDiff with baseline returned error: %v", err)
	}
	if got := report.Markdown(); !strings.Contains(got, "No drift detected.") {
		t.Fatalf("missing no-drift text:\n%s", got)
	}
}

func TestDiffCatalogsReportsOperationAndParameterChanges(t *testing.T) {
	baseline := loadCatalogFixture(t, "ecs-source.json")
	source := loadCatalogFixture(t, "ecs-source.json")
	source.Product.DocsVersion = "82"
	source.Operations[0].Path = "/v4/ecs/list-instances-v2"
	source.Operations[0].Parameters = append(source.Operations[0].Parameters, Parameter{Name: "azName", Location: "body", Type: "string"})
	source.Operations = append(source.Operations, Operation{ID: "v4.ecs.instance.stop", Method: "POST", Path: "/v4/ecs/stop-instance"})
	source.Operations = source.Operations[:2]
	source.Operations = append(source.Operations, Operation{ID: "v4.ecs.instance.stop", Method: "POST", Path: "/v4/ecs/stop-instance"})

	report := DiffCatalogs(baseline, source)
	got := report.Markdown()
	for _, want := range []string{
		"# OpenAPI Drift Report: ecs",
		"- Docs version changed from `81` to `82`.",
		"- Removed operation `v4.ecs.instance.start`.",
		"- Added operation `v4.ecs.instance.stop`.",
		"- Operation `v4.ecs.instance.list` path changed from `/v4/ecs/list-instances` to `/v4/ecs/list-instances-v2`.",
		"- Operation `v4.ecs.instance.list` added parameter `body.azName`.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("report missing %q:\n%s", want, got)
		}
	}
}

func TestDiffCatalogsReportsMethodParameterAndResponseChanges(t *testing.T) {
	baseline := loadCatalogFixture(t, "ecs-source.json")
	source := loadCatalogFixture(t, "ecs-source.json")
	source.Operations[0].Method = "GET"
	source.Operations[0].Parameters[0].Required = false
	source.Operations[0].Parameters[0].Type = "integer"
	source.Operations[0].Parameters = source.Operations[0].Parameters[:1]
	source.Operations[0].Response.RowPath = "returnObj.items"
	source.Operations[0].Response.JobIDPath = "returnObj.jobID"

	got := DiffCatalogs(baseline, source).Markdown()
	for _, want := range []string{
		"- Operation `v4.ecs.instance.list` method changed from `POST` to `GET`.",
		"- Operation `v4.ecs.instance.list` removed parameter `body.keyword`.",
		"- Operation `v4.ecs.instance.list` parameter `body.regionID` required changed from `true` to `false`.",
		"- Operation `v4.ecs.instance.list` parameter `body.regionID` type changed from `string` to `integer`.",
		"- Operation `v4.ecs.instance.list` row path changed from `returnObj.results` to `returnObj.items`.",
		"- Operation `v4.ecs.instance.list` job ID path changed from `` to `returnObj.jobID`.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("report missing %q:\n%s", want, got)
		}
	}
}

func TestGenerateDraftWritesPluginMetadata(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t, "ecs-source.json")); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("GenerateDraft returned error: %v", err)
	}

	manifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	if manifest.Name != "ecs" || manifest.Quality != "generated" || manifest.API.CtyunProductID != 25 {
		t.Fatalf("manifest = %#v", manifest)
	}
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	if len(commands.Commands) != 3 {
		t.Fatalf("command count = %d, want 3", len(commands.Commands))
	}
	if got := commands.Commands[2].Dangerous.Confirm; got != "yes" {
		t.Fatalf("start dangerous confirm = %q, want yes", got)
	}
	apis := readJSONFile[plugin.APIs](t, workspace.ProductPath("ecs", "draft", "apis.json"))
	if apis.Operations["v4.ecs.instance.list"].Body["regionID"] != "$profile.region" {
		t.Fatalf("list region mapping = %#v", apis.Operations["v4.ecs.instance.list"].Body)
	}
	tables := readJSONFile[plugin.Tables](t, workspace.ProductPath("ecs", "draft", "tables.json"))
	if tables.Tables["ecs.instance.list"].Columns[0].Key != "instance_id" {
		t.Fatalf("first column = %#v", tables.Tables["ecs.instance.list"].Columns[0])
	}
}

func TestGenerateDraftCoversFallbacksAndErrors(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	if err := workspace.GenerateDraft("missing"); err == nil {
		t.Fatal("GenerateDraft returned nil error for missing source")
	}
	catalog := loadCatalogFixture(t, "ecs-source.json")
	catalog.Operations = append(catalog.Operations, Operation{
		ID:          "v4.ecs.metadata",
		Category:    "metadata",
		Method:      "GET",
		Path:        "/v4/ecs/metadata",
		ContentType: "application/json",
		Parameters: []Parameter{
			{Name: "X-Ctyun-Test", Location: "header", Type: "string", CLIName: "token"},
			{Name: "ignored", Location: "query", Type: "string"},
		},
		Response:        Response{RowPath: "returnObj", Columns: []Column{{Key: "id", Path: "id", LabelEN: "ID", LabelZH: "ID"}}},
		ExampleResponse: json.RawMessage(`{}`),
	})
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("GenerateDraft returned error: %v", err)
	}
	apis := readJSONFile[plugin.APIs](t, workspace.ProductPath("ecs", "draft", "apis.json"))
	if got := apis.Operations["v4.ecs.metadata"].Headers["X-Ctyun-Test"]; got != "$param.token" {
		t.Fatalf("header binding = %q, want $param.token", got)
	}
	if _, ok := apis.Operations["v4.ecs.metadata"].Query["ignored"]; ok {
		t.Fatalf("unbound parameter was emitted: %#v", apis.Operations["v4.ecs.metadata"].Query)
	}
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	if got := commands.Commands[len(commands.Commands)-1].Path; strings.Join(got, " ") != "ecs metadata metadata" {
		t.Fatalf("fallback command path = %#v", got)
	}
	emptyFixture := readJSONFile[map[string]any](t, workspace.ProductPath("ecs", "draft", "fixtures", "v4-ecs-metadata.json"))
	if len(emptyFixture) != 0 {
		t.Fatalf("empty fixture = %#v, want empty object", emptyFixture)
	}

	conflictRoot := t.TempDir()
	conflictWorkspace := Workspace{Root: conflictRoot}
	if err := conflictWorkspace.WriteCatalog(conflictWorkspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t, "ecs-source.json")); err != nil {
		t.Fatalf("write conflict source: %v", err)
	}
	if err := os.WriteFile(conflictWorkspace.ProductPath("ecs", "draft"), []byte("file"), 0o644); err != nil {
		t.Fatalf("write draft conflict: %v", err)
	}
	if err := conflictWorkspace.GenerateDraft("ecs"); err == nil {
		t.Fatal("GenerateDraft returned nil error for draft path conflict")
	}
	i18nConflictRoot := t.TempDir()
	i18nConflictWorkspace := Workspace{Root: i18nConflictRoot}
	if err := i18nConflictWorkspace.WriteCatalog(i18nConflictWorkspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t, "ecs-source.json")); err != nil {
		t.Fatalf("write i18n conflict source: %v", err)
	}
	if err := os.MkdirAll(i18nConflictWorkspace.ProductPath("ecs", "draft"), 0o755); err != nil {
		t.Fatalf("make draft dir: %v", err)
	}
	if err := os.WriteFile(i18nConflictWorkspace.ProductPath("ecs", "draft", "i18n"), []byte("file"), 0o644); err != nil {
		t.Fatalf("write i18n conflict: %v", err)
	}
	if err := i18nConflictWorkspace.GenerateDraft("ecs"); err == nil {
		t.Fatal("GenerateDraft returned nil error for i18n path conflict")
	}
	if got := string(compactRawMessage(json.RawMessage(`{`))); got != "{" {
		t.Fatalf("compactRawMessage invalid = %q, want raw", got)
	}
	if err := writeFixtures(t.TempDir(), Catalog{Operations: []Operation{{ID: "v4.empty"}}}); err != nil {
		t.Fatalf("writeFixtures empty raw returned error: %v", err)
	}
	if err := writeRawJSON(filepath.Join(t.TempDir(), "bad.json"), json.RawMessage(`{`)); err == nil {
		t.Fatal("writeRawJSON returned nil error for invalid JSON")
	}
	fixtureConflict := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixtureConflict, "fixtures"), []byte("file"), 0o644); err != nil {
		t.Fatalf("write fixture conflict: %v", err)
	}
	if err := writeFixtures(fixtureConflict, Catalog{Operations: []Operation{{ID: "v4.bad"}}}); err == nil {
		t.Fatal("writeFixtures returned nil error for fixture path conflict")
	}
	rawParent := filepath.Join(t.TempDir(), "parent")
	if err := os.WriteFile(rawParent, []byte("file"), 0o644); err != nil {
		t.Fatalf("write raw parent: %v", err)
	}
	if err := writeRawJSON(filepath.Join(rawParent, "child.json"), json.RawMessage(`{}`)); err == nil {
		t.Fatal("writeRawJSON returned nil error for parent file conflict")
	}
	if got := commandAction(Operation{}); got != "call" {
		t.Fatalf("empty operation action = %q, want call", got)
	}
}

func TestReviewDraftRequiresGeneratedQualityAndDangerousConfirmation(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	source := loadCatalogFixture(t, "ecs-source.json")
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), source); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("generate draft: %v", err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatalf("ReviewDraft returned error: %v", err)
	}
	if !report.Ready {
		t.Fatalf("review should be ready: %#v", report)
	}

	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	commands.Commands[2].Dangerous.Confirm = ""
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "commands.json"), commands); err != nil {
		t.Fatalf("write commands: %v", err)
	}
	report, err = workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatalf("ReviewDraft returned error after edit: %v", err)
	}
	if report.Ready {
		t.Fatalf("review should not be ready: %#v", report)
	}
	if !slices.Contains(report.Findings, "operation v4.ecs.instance.start is dangerous but command ecs.instance.start lacks confirmation") {
		t.Fatalf("findings = %#v", report.Findings)
	}
}

func TestReviewDraftReportsMissingCommandTableAndQuality(t *testing.T) {
	if _, err := (Workspace{Root: t.TempDir()}).ReviewDraft("ecs"); err == nil {
		t.Fatal("ReviewDraft returned nil error without source")
	}
	root := t.TempDir()
	workspace := Workspace{Root: root}
	source := loadCatalogFixture(t, "ecs-source.json")
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), source); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := workspace.ReviewDraft("ecs"); err == nil {
		t.Fatal("ReviewDraft returned nil error without draft")
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("generate draft: %v", err)
	}
	if err := os.WriteFile(workspace.ProductPath("ecs", "draft", "commands.json"), []byte(`{`), 0o644); err != nil {
		t.Fatalf("write invalid commands: %v", err)
	}
	if _, err := workspace.ReviewDraft("ecs"); err == nil {
		t.Fatal("ReviewDraft returned nil error for invalid commands")
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("regenerate draft: %v", err)
	}
	if err := os.Mkdir(workspace.ProductPath("ecs", "review.md"), 0o755); err != nil {
		t.Fatalf("make review path directory: %v", err)
	}
	if _, err := workspace.ReviewDraft("ecs"); err == nil {
		t.Fatal("ReviewDraft returned nil error when review.md is a directory")
	}
	if err := os.Remove(workspace.ProductPath("ecs", "review.md")); err != nil {
		t.Fatalf("remove review path directory: %v", err)
	}
	manifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	manifest.Quality = "raw"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	commands.Commands = commands.Commands[1:]
	commands.Commands[0].Table = ""
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "commands.json"), commands); err != nil {
		t.Fatalf("write commands: %v", err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatalf("ReviewDraft returned error: %v", err)
	}
	for _, want := range []string{
		"plugin quality raw is unsupported",
		"operation v4.ecs.instance.list has no command",
		"operation v4.ecs.instance.show has response rows but no table",
	} {
		if !slices.Contains(report.Findings, want) {
			t.Fatalf("findings missing %q: %#v", want, report.Findings)
		}
	}
}

func TestPromoteDraftCopiesMetadataAndAdvancesBaseline(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	source := loadCatalogFixture(t, "ecs-source.json")
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), source); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("generate draft: %v", err)
	}
	err := workspace.PromoteDraft("ecs", filepath.Join(root, "plugins", "ecs"))
	if err == nil {
		t.Fatal("PromoteDraft accepted generated quality")
	}
	if err.Error() != "promotion requires reviewed or curated quality, got generated" {
		t.Fatalf("generated promotion error = %q", err.Error())
	}

	draftManifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	draftManifest.Quality = "reviewed"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), draftManifest); err != nil {
		t.Fatalf("write reviewed manifest: %v", err)
	}
	if err := workspace.PromoteDraft("ecs", filepath.Join(root, "plugins", "ecs")); err != nil {
		t.Fatalf("PromoteDraft returned error: %v", err)
	}
	manifest := readJSONFile[plugin.Manifest](t, filepath.Join(root, "plugins", "ecs", "plugin.json"))
	if manifest.Name != "ecs" || manifest.Quality != "reviewed" {
		t.Fatalf("promoted manifest = %#v", manifest)
	}
	baseline := readCatalogFile(t, workspace.ProductPath("ecs", "baseline.json"))
	if baseline.Product.DocsVersion != source.Product.DocsVersion {
		t.Fatalf("baseline docs version = %q, want %q", baseline.Product.DocsVersion, source.Product.DocsVersion)
	}
	if _, err := os.Stat(filepath.Join(root, "plugins", "ecs", "review.md")); !os.IsNotExist(err) {
		t.Fatalf("review.md was promoted, stat err = %v", err)
	}
}

func TestPromoteDraftRejectsBlockedReviewAndCopyFailures(t *testing.T) {
	if err := (Workspace{Root: t.TempDir()}).PromoteDraft("ecs", filepath.Join(t.TempDir(), "plugins", "ecs")); err == nil {
		t.Fatal("PromoteDraft returned nil error without source")
	}
	root := t.TempDir()
	workspace := Workspace{Root: root}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t, "ecs-source.json")); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.PromoteDraft("ecs", filepath.Join(root, "plugins", "ecs")); err == nil {
		t.Fatal("PromoteDraft returned nil error without draft")
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("generate draft: %v", err)
	}
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	commands.Commands[2].Dangerous.Confirm = ""
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "commands.json"), commands); err != nil {
		t.Fatalf("write commands: %v", err)
	}
	if err := workspace.PromoteDraft("ecs", filepath.Join(root, "plugins", "ecs")); err == nil {
		t.Fatal("PromoteDraft returned nil error for blocked review")
	}
	commands.Commands[2].Dangerous.Confirm = "yes"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "commands.json"), commands); err != nil {
		t.Fatalf("restore commands: %v", err)
	}
	manifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	manifest.Quality = "reviewed"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.Remove(workspace.ProductPath("ecs", "draft", "apis.json")); err != nil {
		t.Fatalf("remove apis: %v", err)
	}
	if err := workspace.PromoteDraft("ecs", filepath.Join(root, "plugins", "ecs")); err == nil {
		t.Fatal("PromoteDraft returned nil error for missing apis.json")
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("regenerate draft for tree error: %v", err)
	}
	manifest = readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	manifest.Quality = "reviewed"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), manifest); err != nil {
		t.Fatalf("write reviewed manifest for tree error: %v", err)
	}
	if err := os.RemoveAll(workspace.ProductPath("ecs", "draft", "fixtures")); err != nil {
		t.Fatalf("remove fixtures: %v", err)
	}
	if err := workspace.PromoteDraft("ecs", filepath.Join(root, "plugins", "ecs")); err == nil {
		t.Fatal("PromoteDraft returned nil error for missing fixtures")
	}
	if err := copyTree(filepath.Join(root, "missing-tree"), filepath.Join(root, "dest")); err == nil {
		t.Fatal("copyTree returned nil error for missing source")
	}
	src := filepath.Join(root, "source-file")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	parentFile := filepath.Join(root, "copy-parent")
	if err := os.WriteFile(parentFile, []byte("file"), 0o644); err != nil {
		t.Fatalf("write copy parent: %v", err)
	}
	if err := copyFile(src, filepath.Join(parentFile, "child")); err == nil {
		t.Fatal("copyFile returned nil error for parent file conflict")
	}
	targetDir := filepath.Join(root, "target-dir")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("make target dir: %v", err)
	}
	if err := copyFile(src, targetDir); err == nil {
		t.Fatal("copyFile returned nil error for directory target")
	}
	treeDestParent := filepath.Join(root, "tree-parent")
	if err := os.WriteFile(treeDestParent, []byte("file"), 0o644); err != nil {
		t.Fatalf("write tree parent: %v", err)
	}
	treeSrc := filepath.Join(root, "tree-src")
	if err := os.Mkdir(treeSrc, 0o755); err != nil {
		t.Fatalf("make tree source: %v", err)
	}
	if err := copyTree(treeSrc, filepath.Join(treeDestParent, "child")); err == nil {
		t.Fatal("copyTree returned nil error for destination parent file")
	}
}

func loadCatalogFixture(t *testing.T, name string) Catalog {
	t.Helper()
	var catalog Catalog
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := decodeJSON(data, &catalog); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return catalog
}

func readCatalogFile(t *testing.T, path string) Catalog {
	t.Helper()
	var catalog Catalog
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	if err := decodeJSON(data, &catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	return catalog
}

func readJSONFile[T any](t *testing.T, path string) T {
	t.Helper()
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := decodeJSON(data, &value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}
