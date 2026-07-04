/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapi

import (
	"bytes"
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
	if len(catalog.Operations) != 4 {
		t.Fatalf("operation count = %d, want 4", len(catalog.Operations))
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
		{name: "offline example", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Examples = append(catalog.Operations[0].Examples, "ctyun --offline ecs instance list")
		}, want: "operation v4.ecs.instance.list example uses dev-only fixture flag --offline"},
		{name: "fixture example", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Examples = append(catalog.Operations[0].Examples, "ctyun --fixture ecs instance list")
		}, want: "operation v4.ecs.instance.list example uses dev-only fixture flag --fixture"},
		{name: "short fixture example", mutate: func(catalog *Catalog) {
			catalog.Operations[0].Examples = append(catalog.Operations[0].Examples, "ctyun -O ecs instance list")
		}, want: "operation v4.ecs.instance.list example uses dev-only fixture flag -O"},
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
	source.Product.SourceRevision = "82"
	source.Operations[0].Path = "/v4/ecs/list-instances-v2"
	source.Operations[0].Parameters = append(source.Operations[0].Parameters, Parameter{Name: "azName", Location: "body", Type: "string"})
	source.Operations = append(source.Operations, Operation{ID: "v4.ecs.instance.stop", Method: "POST", Path: "/v4/ecs/stop-instance"})
	source.Operations = source.Operations[:2]
	source.Operations = append(source.Operations, Operation{ID: "v4.ecs.instance.stop", Method: "POST", Path: "/v4/ecs/stop-instance"})

	report := DiffCatalogs(baseline, source)
	got := report.Markdown()
	for _, want := range []string{
		"# OpenAPI Drift Report: ecs",
		"- Source revision changed from `81` to `82`.",
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

func TestCatalogFingerprintIgnoresWeakProvenanceAndTracksContent(t *testing.T) {
	catalog := loadCatalogFixture(t, "ecs-source.json")
	fingerprint := catalogFingerprint(catalog)
	if !strings.HasPrefix(fingerprint, "sha256:") {
		t.Fatalf("fingerprint = %q, want sha256 prefix", fingerprint)
	}

	weakProvenance := catalog
	weakProvenance.Product.SourceRevision = "82"
	weakProvenance.Product.SourceURL = "https://example.test/changed"
	if got := catalogFingerprint(weakProvenance); got != fingerprint {
		t.Fatalf("weak provenance fingerprint = %q, want %q", got, fingerprint)
	}

	contentChange := catalog
	contentChange.Operations[0].Path = "/v4/ecs/list-instances-v2"
	if got := catalogFingerprint(contentChange); got == fingerprint {
		t.Fatalf("content-change fingerprint still = %q", got)
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
	if len(commands.Commands) != 4 {
		t.Fatalf("command count = %d, want 4", len(commands.Commands))
	}
	if got := commands.Commands[2].Dangerous.Confirm; got != "yes" {
		t.Fatalf("start dangerous confirm = %q, want yes", got)
	}
	if got := strings.Join(commands.Commands[3].Path, " "); got != "ecs zone list {region_id}" {
		t.Fatalf("argument list command path = %q, want scoped list path", got)
	}
	if got := commands.Commands[0].Parameters[0].Flag; got != "name" {
		t.Fatalf("generated flag = %q, want source flag hint", got)
	}
	if got := commands.Commands[0].Examples; !slices.Equal(got, []string{"ctyun ecs instance list", "ctyun ecs instance list --name demo"}) {
		t.Fatalf("generated examples = %#v", got)
	}
	if got := commands.Commands[0].FixtureResponse; got != "fixtures/ecs-instance-list.json" {
		t.Fatalf("fixture response = %q, want command-id-based path", got)
	}
	commandsData, err := os.ReadFile(workspace.ProductPath("ecs", "draft", "commands.json"))
	if err != nil {
		t.Fatalf("read commands JSON: %v", err)
	}
	for _, unwanted := range []string{`"parameters": null`, `"allowed_values": null`, `"pattern": ""`, `"message": ""`} {
		if strings.Contains(string(commandsData), unwanted) {
			t.Fatalf("commands JSON contains avoidable empty field %s:\n%s", unwanted, commandsData)
		}
	}
	apis := readJSONFile[plugin.APIs](t, workspace.ProductPath("ecs", "draft", "apis.json"))
	if apis.Operations["v4.ecs.instance.list"].Body["regionID"] != "$profile.region" {
		t.Fatalf("list region mapping = %#v", apis.Operations["v4.ecs.instance.list"].Body)
	}
	apisData, err := os.ReadFile(workspace.ProductPath("ecs", "draft", "apis.json"))
	if err != nil {
		t.Fatalf("read APIs JSON: %v", err)
	}
	for _, unwanted := range []string{`"headers": {}`, `"body": {}`} {
		if strings.Contains(string(apisData), unwanted) {
			t.Fatalf("APIs JSON contains avoidable empty field %s:\n%s", unwanted, apisData)
		}
	}
	tables := readJSONFile[plugin.Tables](t, workspace.ProductPath("ecs", "draft", "tables.json"))
	if tables.Tables["ecs.instance.list"].Columns[0].Key != "instance_id" {
		t.Fatalf("first column = %#v", tables.Tables["ecs.instance.list"].Columns[0])
	}
	i18n := readJSONFile[map[string]string](t, workspace.ProductPath("ecs", "draft", "i18n", "zh-CN.json"))
	if i18n["name"] != "弹性云主机" {
		t.Fatalf("generated i18n name = %q, want product display name", i18n["name"])
	}
	if i18n["command.ecs.instance.list.description"] != "列出云主机" {
		t.Fatalf("generated command description = %q", i18n["command.ecs.instance.list.description"])
	}
	if i18n["parameter.ecs.instance.list.name.description"] != "按云主机名称过滤" {
		t.Fatalf("generated parameter description = %q", i18n["parameter.ecs.instance.list.name.description"])
	}
}

func TestRegionPluginSourceFingerprintMatchesCatalog(t *testing.T) {
	catalog := readCatalogFile(t, filepath.Join("..", "..", "openapi", "products", "region", "source.json"))
	manifest := readJSONFile[plugin.Manifest](t, filepath.Join("..", "..", "plugins", "region", "plugin.json"))
	if manifest.API.SourceFingerprint != catalogFingerprint(catalog) {
		t.Fatalf("source fingerprint = %q, want %q", manifest.API.SourceFingerprint, catalogFingerprint(catalog))
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
	emptyFixture := readJSONFile[map[string]any](t, workspace.ProductPath("ecs", "draft", "fixtures", "ecs-metadata.json"))
	if len(emptyFixture) != 0 {
		t.Fatalf("empty fixture = %#v, want empty object", emptyFixture)
	}
	staleFixture := workspace.ProductPath("ecs", "draft", "fixtures", "v4-ecs-metadata.json")
	if err := os.WriteFile(staleFixture, []byte(`{"stale":true}`), 0o644); err != nil {
		t.Fatalf("write stale fixture: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("regenerate draft with stale fixture: %v", err)
	}
	if _, err := os.Stat(staleFixture); !os.IsNotExist(err) {
		t.Fatalf("stale fixture still exists after regeneration: %v", err)
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
	if err := i18nConflictWorkspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("GenerateDraft should replace stale draft contents: %v", err)
	}
	if _, err := os.Stat(i18nConflictWorkspace.ProductPath("ecs", "draft", "i18n", "en-US.json")); err != nil {
		t.Fatalf("generated i18n after replacing stale draft contents: %v", err)
	}
	if got := string(compactRawMessage(json.RawMessage(`{`))); got != "{" {
		t.Fatalf("compactRawMessage invalid = %q, want raw", got)
	}
	if err := prepareDraftDir(string([]byte{0})); err == nil {
		t.Fatal("prepareDraftDir returned nil error for invalid path")
	}
	metadataConflict := t.TempDir()
	if err := os.Mkdir(filepath.Join(metadataConflict, "plugin.json"), 0o755); err != nil {
		t.Fatalf("create metadata conflict: %v", err)
	}
	if err := writeDraft(metadataConflict, loadCatalogFixture(t, "ecs-source.json")); err == nil {
		t.Fatal("writeDraft returned nil error for metadata file conflict")
	}
	i18nConflict := t.TempDir()
	if err := os.WriteFile(filepath.Join(i18nConflict, "i18n"), []byte("file"), 0o644); err != nil {
		t.Fatalf("create i18n conflict: %v", err)
	}
	if err := writeDraft(i18nConflict, loadCatalogFixture(t, "ecs-source.json")); err == nil {
		t.Fatal("writeDraft returned nil error for i18n path conflict")
	}
	fixtureWriteConflict := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixtureWriteConflict, "fixtures"), []byte("file"), 0o644); err != nil {
		t.Fatalf("create fixture write conflict: %v", err)
	}
	if err := writeDraft(fixtureWriteConflict, loadCatalogFixture(t, "ecs-source.json")); err == nil {
		t.Fatal("writeDraft returned nil error for fixture path conflict")
	}
	if err := writeFixtures(t.TempDir(), Catalog{Operations: []Operation{{ID: "v4.empty"}}}); err != nil {
		t.Fatalf("writeFixtures empty raw returned error: %v", err)
	}
	inlineArrayPath := filepath.Join(t.TempDir(), "fixture.json")
	if err := writeRawJSON(inlineArrayPath, json.RawMessage(`{"returnObj":{"items":[{"zoneList":["az1","az2"],"cpuArches":["x86_64"],"nested":[{"id":"keep"}]}]}}`)); err != nil {
		t.Fatalf("writeRawJSON inline arrays returned error: %v", err)
	}
	inlineArrayFixture, err := os.ReadFile(inlineArrayPath)
	if err != nil {
		t.Fatalf("read inline array fixture: %v", err)
	}
	wantInlineArrayFixture := "{\n" +
		"  \"returnObj\": {\n" +
		"    \"items\": [\n" +
		"      {\n" +
		"        \"zoneList\": [\"az1\", \"az2\"],\n" +
		"        \"cpuArches\": [\"x86_64\"],\n" +
		"        \"nested\": [\n" +
		"          {\n" +
		"            \"id\": \"keep\"\n" +
		"          }\n" +
		"        ]\n" +
		"      }\n" +
		"    ]\n" +
		"  }\n" +
		"}\n"
	if string(inlineArrayFixture) != wantInlineArrayFixture {
		t.Fatalf("inline array fixture =\n%s\nwant\n%s", inlineArrayFixture, wantInlineArrayFixture)
	}
	if _, _, ok := scalarArrayItems([]string{`  "unterminated"`}, 0); ok {
		t.Fatal("scalarArrayItems accepted an unterminated array")
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

	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("regenerate draft: %v", err)
	}
	commands = readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	commands.Commands[1].Path = []string{"ecs", "instance", "show"}
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "commands.json"), commands); err != nil {
		t.Fatalf("write commands without argument: %v", err)
	}
	report, err = workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatalf("ReviewDraft returned error after dropping argument: %v", err)
	}
	if !slices.Contains(report.Findings, "operation v4.ecs.instance.show argument instance_id is not exposed by command ecs.instance.show") {
		t.Fatalf("findings missing command argument drift: %#v", report.Findings)
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

func TestReviewDraftRequiresReviewedSourceFingerprint(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	source := loadCatalogFixture(t, "ecs-source.json")
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), source); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("generate draft: %v", err)
	}
	manifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	manifest.Quality = "reviewed"
	manifest.API.SourceFingerprint = ""
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), manifest); err != nil {
		t.Fatalf("write missing fingerprint manifest: %v", err)
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatalf("ReviewDraft missing fingerprint returned error: %v", err)
	}
	sourceFingerprint := catalogFingerprint(source)
	if !slices.Contains(report.Findings, "reviewed plugin must include source fingerprint "+sourceFingerprint) {
		t.Fatalf("findings missing source fingerprint requirement: %#v", report.Findings)
	}

	manifest.API.SourceFingerprint = "sha256:stale"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), manifest); err != nil {
		t.Fatalf("write stale fingerprint manifest: %v", err)
	}
	report, err = workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatalf("ReviewDraft stale fingerprint returned error: %v", err)
	}
	if !slices.Contains(report.Findings, "reviewed plugin source fingerprint sha256:stale does not match source catalog "+sourceFingerprint) {
		t.Fatalf("findings missing stale source fingerprint: %#v", report.Findings)
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
	targetDir := filepath.Join(root, "plugins", "ecs")
	commandsData, err := os.ReadFile(workspace.ProductPath("ecs", "draft", "commands.json"))
	commandsRaw := compactJSONBytes(t, commandsData, err)
	targetCommands := filepath.Join(targetDir, "commands.json")
	if err := os.MkdirAll(filepath.Dir(targetCommands), 0o755); err != nil {
		t.Fatalf("create target metadata dir: %v", err)
	}
	if err := os.WriteFile(targetCommands, commandsRaw, 0o644); err != nil {
		t.Fatalf("write target commands: %v", err)
	}
	i18nData, err := os.ReadFile(workspace.ProductPath("ecs", "draft", "i18n", "en-US.json"))
	i18nRaw := compactJSONBytes(t, i18nData, err)
	targetI18N := filepath.Join(targetDir, "i18n", "en-US.json")
	if err := os.MkdirAll(filepath.Dir(targetI18N), 0o755); err != nil {
		t.Fatalf("create target i18n dir: %v", err)
	}
	if err := os.WriteFile(targetI18N, i18nRaw, 0o644); err != nil {
		t.Fatalf("write target i18n: %v", err)
	}
	if err := workspace.PromoteDraft("ecs", targetDir); err != nil {
		t.Fatalf("PromoteDraft returned error: %v", err)
	}
	if got := readFileBytes(t, targetCommands); !bytes.Equal(got, commandsRaw) {
		t.Fatalf("semantically equal commands were rewritten:\n%s", got)
	}
	if got := readFileBytes(t, targetI18N); !bytes.Equal(got, i18nRaw) {
		t.Fatalf("semantically equal i18n was rewritten:\n%s", got)
	}
	manifest := readJSONFile[plugin.Manifest](t, filepath.Join(targetDir, "plugin.json"))
	if manifest.Name != "ecs" || manifest.Quality != "reviewed" {
		t.Fatalf("promoted manifest = %#v", manifest)
	}
	if manifest.API.SourceRevision != source.Product.SourceRevision {
		t.Fatalf("source revision = %q, want %q", manifest.API.SourceRevision, source.Product.SourceRevision)
	}
	if manifest.API.SourceFingerprint != catalogFingerprint(source) {
		t.Fatalf("source fingerprint = %q, want %q", manifest.API.SourceFingerprint, catalogFingerprint(source))
	}
	baseline := readCatalogFile(t, workspace.ProductPath("ecs", "baseline.json"))
	if baseline.Product.SourceRevision != source.Product.SourceRevision {
		t.Fatalf("baseline source revision = %q, want %q", baseline.Product.SourceRevision, source.Product.SourceRevision)
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
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("regenerate draft for remove error: %v", err)
	}
	manifest = readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	manifest.Quality = "reviewed"
	if err := writeJSON(workspace.ProductPath("ecs", "draft", "plugin.json"), manifest); err != nil {
		t.Fatalf("write reviewed manifest for remove error: %v", err)
	}
	removeErrorTarget := filepath.Join(root, "plugins", "remove-error")
	protectedFixtureChild := filepath.Join(removeErrorTarget, "fixtures", "child")
	if err := os.MkdirAll(protectedFixtureChild, 0o755); err != nil {
		t.Fatalf("create protected fixture child: %v", err)
	}
	if err := os.WriteFile(filepath.Join(protectedFixtureChild, "file.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write protected fixture child: %v", err)
	}
	protectedFixtures := filepath.Join(removeErrorTarget, "fixtures")
	if err := os.Chmod(protectedFixtures, 0o500); err != nil {
		t.Fatalf("protect fixtures: %v", err)
	}
	defer func() {
		_ = os.Chmod(protectedFixtures, 0o755)
	}()
	if err := workspace.PromoteDraft("ecs", removeErrorTarget); err == nil {
		t.Fatal("PromoteDraft returned nil error for protected fixtures")
	}
	syncRoot := t.TempDir()
	syncSource := filepath.Join(syncRoot, "source")
	syncTarget := filepath.Join(syncRoot, "target")
	if err := os.MkdirAll(filepath.Join(syncSource, "nested"), 0o755); err != nil {
		t.Fatalf("create sync source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(syncSource, "keep.json"), []byte(`{"value":1}`), 0o644); err != nil {
		t.Fatalf("write source keep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(syncSource, "nested", "copy.json"), []byte(`{"copy":true}`), 0o644); err != nil {
		t.Fatalf("write source copy: %v", err)
	}
	keptRaw := []byte("{\n  \"value\": 1\n}\n")
	if err := os.MkdirAll(filepath.Join(syncTarget, "nested", "stale-dir"), 0o755); err != nil {
		t.Fatalf("create sync target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(syncTarget, "keep.json"), keptRaw, 0o644); err != nil {
		t.Fatalf("write target keep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(syncTarget, "stale.json"), []byte(`{"stale":true}`), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(syncTarget, "nested", "stale-dir", "file.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write stale nested file: %v", err)
	}
	if err := syncJSONTree(syncSource, syncTarget); err != nil {
		t.Fatalf("syncJSONTree returned error: %v", err)
	}
	if got := readFileBytes(t, filepath.Join(syncTarget, "keep.json")); !bytes.Equal(got, keptRaw) {
		t.Fatalf("equivalent JSON was rewritten: %s", got)
	}
	if _, err := os.Stat(filepath.Join(syncTarget, "stale.json")); !os.IsNotExist(err) {
		t.Fatalf("stale file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(syncTarget, "nested", "stale-dir")); !os.IsNotExist(err) {
		t.Fatalf("stale directory still exists: %v", err)
	}
	removeRoot := t.TempDir()
	removeSource := filepath.Join(removeRoot, "source")
	removeTarget := filepath.Join(removeRoot, "target")
	if err := os.Mkdir(removeSource, 0o755); err != nil {
		t.Fatalf("create remove source: %v", err)
	}
	if err := os.Mkdir(removeTarget, 0o755); err != nil {
		t.Fatalf("create remove target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(removeTarget, "stale.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write protected stale file: %v", err)
	}
	if err := os.Chmod(removeTarget, 0o500); err != nil {
		t.Fatalf("protect remove target: %v", err)
	}
	defer func() {
		_ = os.Chmod(removeTarget, 0o755)
	}()
	if err := syncJSONTree(removeSource, removeTarget); err == nil {
		t.Fatal("syncJSONTree returned nil error for protected stale target")
	}
	walkErrorRoot := t.TempDir()
	walkErrorSource := filepath.Join(walkErrorRoot, "source")
	walkErrorTarget := filepath.Join(walkErrorRoot, "target")
	if err := os.Mkdir(walkErrorSource, 0o755); err != nil {
		t.Fatalf("create walk error source: %v", err)
	}
	blockedTargetDir := filepath.Join(walkErrorTarget, "blocked")
	if err := os.MkdirAll(blockedTargetDir, 0o755); err != nil {
		t.Fatalf("create blocked target dir: %v", err)
	}
	if err := os.Chmod(blockedTargetDir, 0); err != nil {
		t.Fatalf("block target dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(blockedTargetDir, 0o755)
	}()
	if err := syncJSONTree(walkErrorSource, walkErrorTarget); err == nil {
		t.Fatal("syncJSONTree returned nil error for unreadable target child")
	}
	copySource := filepath.Join(root, "copy-source.json")
	if err := os.WriteFile(copySource, []byte(`{"value":2}`), 0o644); err != nil {
		t.Fatalf("write copy source: %v", err)
	}
	copyTarget := filepath.Join(root, "copy-target.json")
	if err := os.WriteFile(copyTarget, []byte(`{"value":1}`), 0o644); err != nil {
		t.Fatalf("write copy target: %v", err)
	}
	if err := copyJSONFileIfChanged(copySource, copyTarget); err != nil {
		t.Fatalf("copyJSONFileIfChanged returned error: %v", err)
	}
	if got := readFileBytes(t, copyTarget); string(got) != `{"value":2}` {
		t.Fatalf("changed JSON was not copied: %s", got)
	}
	if err := copyJSONFileIfChanged(filepath.Join(root, "missing-source.json"), filepath.Join(root, "dest.json")); err == nil {
		t.Fatal("copyJSONFileIfChanged returned nil error for missing source")
	}
	copyParentFile := filepath.Join(root, "copy-parent-file")
	if err := os.WriteFile(copyParentFile, []byte("file"), 0o644); err != nil {
		t.Fatalf("write copy parent file: %v", err)
	}
	if err := copyJSONFileIfChanged(copySource, filepath.Join(copyParentFile, "child.json")); err == nil {
		t.Fatal("copyJSONFileIfChanged returned nil error for parent file conflict")
	}
	if err := copyJSONFileIfChanged(copySource, string([]byte{0})); err == nil {
		t.Fatal("copyJSONFileIfChanged returned nil error for invalid destination")
	}
	if equivalentJSON([]byte(`{`), []byte(`{}`)) {
		t.Fatal("equivalentJSON accepted invalid left JSON")
	}
	if equivalentJSON([]byte(`{}`), []byte(`{`)) {
		t.Fatal("equivalentJSON accepted invalid right JSON")
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

func readFileBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func compactJSONBytes(t *testing.T, data []byte, err error) []byte {
	t.Helper()
	if err != nil {
		t.Fatalf("read JSON to compact: %v", err)
	}
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, data); err != nil {
		t.Fatalf("compact JSON: %v", err)
	}
	return append(buffer.Bytes(), '\n')
}
