/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestGenerateDraftWritesPluginMetadata(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t)); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("GenerateDraft returned error: %v", err)
	}

	manifest := readJSONFile[plugin.Manifest](t, workspace.ProductPath("ecs", "draft", "plugin.json"))
	if manifest.Name != "ecs" || manifest.Quality != "generated" || manifest.API.CtyunProductID != 25 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if !apiScopeEqual(manifest.API.Scope, loadCatalogFixture(t).Product.APIScope) {
		t.Fatalf("manifest API scope = %#v, want catalog scope", manifest.API.Scope)
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
	if len(commands.Commands[3].Parameters) != 0 {
		t.Fatalf("argument region command parameters = %#v, want no duplicate region option", commands.Commands[3].Parameters)
	}
	if got := commands.Commands[0].Parameters[0]; got.Name != "region" || got.Flag != "region" || got.Target != "regionID" || got.Required {
		t.Fatalf("generated profile region parameter = %#v", got)
	}
	if got := commands.Commands[0].Parameters[1].Flag; got != "name" {
		t.Fatalf("generated flag = %q, want source flag hint", got)
	}
	if got := commands.Commands[0].Examples; !slices.Equal(got, []string{"ctyun ecs instance list", "ctyun ecs instance list --name demo"}) {
		t.Fatalf("generated examples = %#v", got)
	}
	if got := commands.Commands[1].Examples; !slices.Equal(got, []string{"ctyun ecs instance show ins-demo-1"}) {
		t.Fatalf("generated concrete argument example = %#v", got)
	}
	if got := commands.Commands[0].FixtureResponse; got != "fixtures/ecs-instance-list.json" {
		t.Fatalf("fixture response = %q, want command-id-based path", got)
	}
	if got := commands.Commands[0].Parameters[2].Description; got != "Page No" {
		t.Fatalf("generated English parameter description = %q, want generated English fallback", got)
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
	if got := tables.Tables["ecs.instance.list"].Columns[3].Labels["zh-CN"]; got != "创建时间" {
		t.Fatalf("generated zh-CN table label = %q, want Chinese fallback", got)
	}
	waiters := readJSONFile[plugin.Waiters](t, workspace.ProductPath("ecs", "draft", "waiters.json"))
	if waiter := waiters.Waiters["ecs.instance.running"]; waiter.Path != "returnObj.instanceStatus" || waiter.Success != "running" {
		t.Fatalf("generated running waiter = %#v", waiter)
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
	if got := i18n["parameter.ecs.instance.list.region.description"]; got != "资源池 ID" {
		t.Fatalf("generated region parameter description = %q", got)
	}
	englishI18N := readJSONFile[map[string]string](t, workspace.ProductPath("ecs", "draft", "i18n", "en-US.json"))
	if got := englishI18N["argument.ecs.instance.show.instance_id.description"]; got != "Instance ID" {
		t.Fatalf("generated English argument description = %q, want generated English fallback", got)
	}
	if got := englishI18N["parameter.ecs.instance.list.region.description"]; got != "Region ID" {
		t.Fatalf("generated English region parameter description = %q, want generated English fallback", got)
	}
	if got := englishI18N["parameter.ecs.instance.list.page_no.description"]; got != "Page No" {
		t.Fatalf("generated English i18n parameter description = %q, want generated English fallback", got)
	}
}

func TestGenerateDraftCarriesDeprecationMetadata(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	catalog := loadCatalogFixture(t)
	catalog.Operations = catalog.Operations[:1]
	catalog.Operations[0].Description["zh-CN"] = "旧接口，后续可能下线"
	catalog.Operations[0].Parameters[2].Name = "pageNumber"
	catalog.Operations[0].Parameters[2].CLIName = "page_number"
	catalog.Operations[0].Parameters[2].CLIFlag = "page-number"
	catalog.Operations[0].Parameters[2].TableTarget = "pageNumber"
	catalog.Operations[0].Parameters[2].Description = "页码。建议使用pageNo，该字段未来将会下线。"
	catalog.Operations[0].Parameters[2].Descriptions = map[string]string{
		"zh-CN": "页码。建议使用pageNo，该字段未来将会下线。",
	}
	catalog.Operations[0].Response.Columns[0].Description = "实例 ID（废弃该字段）"

	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("GenerateDraft returned error: %v", err)
	}

	apis := readJSONFile[plugin.APIs](t, workspace.ProductPath("ecs", "draft", "apis.json"))
	if got := apis.Operations["v4.ecs.instance.list"].Deprecation; got.Status != "deprecated" || got.Notice != "旧接口，后续可能下线" {
		t.Fatalf("operation deprecation = %#v", got)
	}
	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	if got := commands.Commands[0].Parameters[2].Deprecation; got.Status != "deprecated" || got.Replacement.Label != "pageNo" {
		t.Fatalf("parameter deprecation = %#v", got)
	}
	tables := readJSONFile[plugin.Tables](t, workspace.ProductPath("ecs", "draft", "tables.json"))
	if got := tables.Tables["ecs.instance.list"].Columns[0].Deprecation; got.Status != "deprecated" || got.Notice != "实例 ID（废弃该字段）" {
		t.Fatalf("column deprecation = %#v", got)
	}
}

func TestDeprecationTextHelpersCoverFallbacks(t *testing.T) {
	if got := deprecationNotice("current field", map[string]string{"zh-CN": "仍然可用"}); got != "" {
		t.Fatalf("deprecationNotice = %q, want empty", got)
	}
	if got := leadingReplacementToken("!pageNo"); got != "" {
		t.Fatalf("leadingReplacementToken invalid start = %q, want empty", got)
	}
	if got := leadingReplacementToken("（ pageNo"); got != "pageNo" {
		t.Fatalf("leadingReplacementToken punctuation skip = %q, want pageNo", got)
	}
}

func TestGenerateDraftExposesAllArgumentParameters(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	catalog := loadCatalogFixture(t)
	catalog.Operations = append(catalog.Operations, Operation{
		ID:          "v4.ecs.job.info",
		APIID:       "5865",
		Title:       "通用任务状态查询",
		Method:      "GET",
		Path:        "/v4/ecs/job-info",
		ContentType: "application/json",
		Parameters: []Parameter{
			{Name: "regionID", Location: "query", Required: true, Type: "String", Argument: "region_id"},
			{Name: "jobID", Location: "query", Required: true, Type: "String", Argument: "job_id"},
		},
		Response:        Response{RowPath: "returnObj", Columns: []Column{{Key: "job_id", Path: "jobID", LabelEN: "Job ID", LabelZH: "任务 ID"}}},
		ExampleResponse: json.RawMessage(`{"statusCode":800,"returnObj":{"jobID":"job-demo-1"}}`),
	})
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatalf("GenerateDraft returned error: %v", err)
	}

	commands := readJSONFile[plugin.Commands](t, workspace.ProductPath("ecs", "draft", "commands.json"))
	command := commands.Commands[len(commands.Commands)-1]
	if got := strings.Join(command.Path, " "); got != "ecs info {region_id} {job_id}" {
		t.Fatalf("multi-argument command path = %q, want both required arguments", got)
	}
	if len(command.Parameters) != 0 {
		t.Fatalf("multi-argument command parameters = %#v, want no region option for region_id argument", command.Parameters)
	}
}

func TestConcreteExamplesUseOnlyCapturedScalarValues(t *testing.T) {
	original := []string{"ctyun demo show {id}"}
	for _, operation := range []Operation{
		{Examples: original, Parameters: []Parameter{{Name: "id", Argument: "id"}}},
		{Examples: original, Parameters: []Parameter{{Name: "id", Argument: "id"}}, ExampleResponse: json.RawMessage(`{`)},
		{Examples: original, Parameters: []Parameter{{Name: "id"}}},
		{Examples: original, Parameters: []Parameter{{Name: "id", Argument: "id"}}, ExampleResponse: json.RawMessage(`{"id":{"nested":"value"}}`)},
	} {
		if got := concreteExamples(operation); !slices.Equal(got, original) {
			t.Fatalf("concreteExamples without scalar evidence = %#v, want %#v", got, original)
		}
	}

	operation := Operation{
		Examples: []string{"ctyun demo show {enabled} {count}"},
		Parameters: []Parameter{
			{Name: "enabled", Argument: "enabled"},
			{Name: "count", Argument: "count"},
		},
		ExampleResponse: json.RawMessage(`{"returnObj":{"items":[{"enabled":true,"count":2}]}}`),
	}
	if got := concreteExamples(operation); !slices.Equal(got, []string{"ctyun demo show true 2"}) {
		t.Fatalf("concreteExamples scalar replacements = %#v", got)
	}
	if value, ok := scalarExampleValue(nil); ok || value != "" {
		t.Fatalf("scalarExampleValue(nil) = %q %v", value, ok)
	}
}

func TestGenerateDraftCoversFallbacksAndErrors(t *testing.T) {
	root := t.TempDir()
	workspace := Workspace{Root: root}
	if err := workspace.GenerateDraft("missing"); err == nil {
		t.Fatal("GenerateDraft returned nil error for missing source")
	}
	catalog := loadCatalogFixture(t)
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
	if err := conflictWorkspace.WriteCatalog(conflictWorkspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t)); err != nil {
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
	if err := i18nConflictWorkspace.WriteCatalog(i18nConflictWorkspace.ProductPath("ecs", "source.json"), loadCatalogFixture(t)); err != nil {
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
	if err := writeDraft(metadataConflict, loadCatalogFixture(t)); err == nil {
		t.Fatal("writeDraft returned nil error for metadata file conflict")
	}
	i18nConflict := t.TempDir()
	if err := os.WriteFile(filepath.Join(i18nConflict, "i18n"), []byte("file"), 0o644); err != nil {
		t.Fatalf("create i18n conflict: %v", err)
	}
	if err := writeDraft(i18nConflict, loadCatalogFixture(t)); err == nil {
		t.Fatal("writeDraft returned nil error for i18n path conflict")
	}
	fixtureConflict := t.TempDir()
	if err := os.MkdirAll(filepath.Join(fixtureConflict, "fixtures"), 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(fixtureConflict, "fixtures", "ecs-instance-list.json"), 0o755); err != nil {
		t.Fatalf("create fixture conflict: %v", err)
	}
	if err := writeFixtures(fixtureConflict, loadCatalogFixture(t)); err == nil {
		t.Fatal("writeFixtures returned nil error for fixture file conflict")
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
	if got := string(collapseIndentedScalarArrays("  \"items\": [\n    {\n  ]")); got != "  \"items\": [\n    {\n  ]" {
		t.Fatalf("non-scalar array collapsed to %q", got)
	}
	if _, _, ok := scalarArrayItems([]string{`  "unterminated"`}, 0); ok {
		t.Fatal("scalarArrayItems accepted an unterminated array")
	}
	if _, _, ok := scalarArrayItems([]string{"", "]"}, 0); ok {
		t.Fatal("scalarArrayItems accepted an empty scalar item")
	}
	if err := writeRawJSON(filepath.Join(t.TempDir(), "bad.json"), json.RawMessage(`{`)); err == nil {
		t.Fatal("writeRawJSON returned nil error for invalid JSON")
	}
	fixtureParentConflict := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixtureParentConflict, "fixtures"), []byte("file"), 0o644); err != nil {
		t.Fatalf("write fixture parent conflict: %v", err)
	}
	if err := writeFixtures(fixtureParentConflict, Catalog{Operations: []Operation{{ID: "v4.bad"}}}); err == nil {
		t.Fatal("writeFixtures returned nil error for fixture parent conflict")
	}
	rawParent := filepath.Join(t.TempDir(), "parent")
	if err := os.WriteFile(rawParent, []byte("file"), 0o644); err != nil {
		t.Fatalf("write raw parent: %v", err)
	}
	if err := writeRawJSON(filepath.Join(rawParent, "child.json"), json.RawMessage(`{}`)); err == nil {
		t.Fatal("writeRawJSON returned nil error for parent file conflict")
	}
	if got := string(compactRawMessage(nil)); got != "" {
		t.Fatalf("compactRawMessage empty = %q, want empty", got)
	}
	if got := commandAction(Operation{}); got != "call" {
		t.Fatalf("empty operation action = %q, want call", got)
	}
}

func TestGenerateLabelFallbacks(t *testing.T) {
	cases := []struct {
		name         string
		column       Column
		englishLabel string
		want         string
	}{
		{
			name:         "English source label fallback",
			column:       Column{Key: "foo_bar", LabelZH: "Foo Bar"},
			englishLabel: "Foo Bar",
			want:         "Foo Bar",
		},
		{
			name:         "unknown technical identifier fallback",
			column:       Column{Key: "tenant_zone"},
			englishLabel: "Tenant Zone",
			want:         "Tenant Zone",
		},
		{
			name:         "empty generated label fallback",
			column:       Column{Key: " "},
			englishLabel: "Fallback",
			want:         "Fallback",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := chineseColumnLabel(tc.column, tc.englishLabel); got != tc.want {
				t.Fatalf("chineseColumnLabel = %q, want %q", got, tc.want)
			}
		})
	}
}
