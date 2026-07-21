/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestParameterValueTypeMapsNormalizedTypes verifies the closed mapping from
// captured CTyun parameter types to the plugin request contract.
func TestParameterValueTypeMapsNormalizedTypes(t *testing.T) {
	tests := []struct {
		source string
		want   plugin.ParameterValueType
	}{
		{source: "String", want: plugin.ParameterValueString},
		{source: "Integer", want: plugin.ParameterValueInteger},
		{source: "Float", want: plugin.ParameterValueNumber},
		{source: "Boolean", want: plugin.ParameterValueBoolean},
		{source: "Array of Strings", want: plugin.ParameterValueStringArray},
		{source: "Array of Integers", want: plugin.ParameterValueIntegerArray},
		{source: "Array of Objects", want: plugin.ParameterValueObjectArray},
		{source: "Map of String", want: plugin.ParameterValueStringMap},
		{source: " Object ", want: plugin.ParameterValueJSON},
		{source: "jSoN", want: plugin.ParameterValueJSON},
	}
	for _, tc := range tests {
		t.Run(tc.source, func(t *testing.T) {
			got, err := parameterValueType(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("parameterValueType(%q) = %q, want %q", tc.source, got, tc.want)
			}
		})
	}
	if _, err := parameterValueType("Binary"); err == nil {
		t.Fatal("parameterValueType accepted unsupported source type")
	}
}

// TestGeneratedJSONParameterExamplesUseRuntimeValidation verifies that Object
// and JSON source evidence becomes a concrete typed option example.
func TestGeneratedJSONParameterExamplesUseRuntimeValidation(t *testing.T) {
	for _, sourceType := range []string{"Object", "JSON"} {
		t.Run(sourceType, func(t *testing.T) {
			catalog := loadCatalogFixture(t)
			operation := &catalog.Operations[0]
			operation.Examples = nil
			operation.Parameters = []Parameter{{
				Name:     "payload",
				Location: "body",
				Required: true,
				Type:     sourceType,
				CLIName:  "payload",
				CLIFlag:  "payload",
				Example:  json.RawMessage(`{"nested":[true,2]}`),
			}}

			command := buildCommands(catalog).Commands[0]
			if got := command.Parameters[0].ValueType; got != plugin.ParameterValueJSON {
				t.Fatalf("generated %s value type = %q, want %q", sourceType, got, plugin.ParameterValueJSON)
			}
			example := command.Examples[0]
			if !strings.Contains(example, `{"nested":[true,2]}`) {
				t.Fatalf("generated %s example does not contain concrete JSON: %q", sourceType, example)
			}
			if err := plugin.ValidateCommandExample(command, example); err != nil {
				t.Fatalf("ValidateCommandExample rejected generated %s example: %v", sourceType, err)
			}
			invalid := strings.Replace(example, `{"nested":[true,2]}`, `{"nested":}`, 1)
			if err := plugin.ValidateCommandExample(command, invalid); err == nil {
				t.Fatalf("ValidateCommandExample accepted invalid generated %s JSON", sourceType)
			}
		})
	}
}

// TestGeneratedExamplesIncludeRequiredTypedInputs verifies that the first
// published example is canonical, complete, typed, and shell-safe.
func TestGeneratedExamplesIncludeRequiredTypedInputs(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations = []Operation{{
		ID:          "v4.ecs.instance.vm-cpu-history-metric-data",
		APIID:       "test-cpu-history",
		Title:       "查询云主机 CPU 历史监控数据",
		Description: map[string]string{"en-US": "List CPU metrics for ECS instances over a specified time range", "en-GB": "List CPU metrics for ECS instances over a specified time range", "zh-CN": "查询云主机指定时间范围内的 CPU 监控数据"},
		Category:    "instance",
		Method:      "POST",
		Path:        "/v4/ecs/vm-cpu-history-metric-data",
		ContentType: "application/json",
		Examples:    []string{"ctyun ecs instance vm-cpu-history-metric-data"},
		RequestExample: map[string]json.RawMessage{
			"deviceIDList": json.RawMessage(`["3c496231-8fe5-44b9-8d6b-6129f6e319a8","8887670d-34cd-432f-b977-aa1f62d2e13c"]`),
			"startTime":    json.RawMessage(`1667486264`),
			"endTime":      json.RawMessage(`1667491264`),
		},
		Parameters: []Parameter{
			{Name: "regionID", Location: "body", Required: true, Type: "String", Profile: "region"},
			{Name: "deviceIDList", Location: "body", Required: true, Type: "Array of Strings", CLIName: "device_idlist", CLIFlag: "device-idlist"},
			{Name: "startTime", Location: "body", Required: true, Type: "Integer", CLIName: "start_time", CLIFlag: "start-time"},
			{Name: "endTime", Location: "body", Required: true, Type: "Integer", CLIName: "end_time", CLIFlag: "end-time"},
		},
		Response: Response{RowPath: "returnObj", Columns: []Column{{Key: "metric", Path: "metric", LabelEN: "Metric", LabelZH: "指标"}}},
	}}

	commands := buildCommands(catalog)
	want := []string{`ctyun ecs instance vm-cpu-history-metric-data --device-idlist '["3c496231-8fe5-44b9-8d6b-6129f6e319a8","8887670d-34cd-432f-b977-aa1f62d2e13c"]' --start-time 1667486264 --end-time 1667491264`}
	if got := commands.Commands[0].Examples; !slices.Equal(got, want) {
		t.Fatalf("generated CPU examples = %#v, want %#v", got, want)
	}
}

// TestGeneratedExamplesPreferCompleteCapturedRequiredExample verifies a
// complete reviewed source example is not replaced by a smaller generated
// command when required options are present.
func TestGeneratedExamplesPreferCompleteCapturedRequiredExample(t *testing.T) {
	const captured = "ctyun demo create --name official --mode subscription --count 6"
	operation := Operation{Examples: []string{captured}}
	command := plugin.Command{
		Path: []string{"demo", "create"},
		Parameters: []plugin.Parameter{
			{Name: "name", Flag: "name", Required: true},
			{Name: "mode", Flag: "mode"},
			{Name: "count", Flag: "count", ValueType: plugin.ParameterValueInteger},
		},
	}
	if got := generatedCommandExamples(operation, command); !slices.Equal(got, []string{captured}) {
		t.Fatalf("generated examples = %#v, want complete captured example", got)
	}
}

// TestGeneratedExamplesOmitBareCommandPaths verifies that a command without
// source examples or required user input does not publish its own invocation
// as a redundant example.
func TestGeneratedExamplesOmitBareCommandPaths(t *testing.T) {
	operation := Operation{}
	command := plugin.Command{Path: []string{"acs", "flavor", "list"}}
	if got := generatedCommandExamples(operation, command); len(got) != 0 {
		t.Fatalf("generated bare examples = %#v, want none", got)
	}
}

// TestGeneratedExamplesOmitCapturedBareCommandPaths verifies upstream example
// evidence does not reintroduce an invocation that duplicates Usage.
func TestGeneratedExamplesOmitCapturedBareCommandPaths(t *testing.T) {
	operation := Operation{Examples: []string{
		"ctyun region list",
		"ctyun region list --name 华东1",
		"  ctyun region list  ",
	}}
	command := plugin.Command{Path: []string{"region", "list"}}
	want := []string{"ctyun region list --name 华东1"}
	if got := generatedCommandExamples(operation, command); !slices.Equal(got, want) {
		t.Fatalf("generated examples = %#v, want %#v", got, want)
	}
}

// TestGeneratedExamplesRetainRequiredInputs verifies that positional and
// required option evidence still produces useful examples without source CLI
// example seeds.
func TestGeneratedExamplesRetainRequiredInputs(t *testing.T) {
	argumentOperation := Operation{
		Parameters: []Parameter{{Name: "instanceID", Type: "String", Argument: "instance_id", Example: json.RawMessage(`"instance-demo"`)}},
	}
	argumentCommand := plugin.Command{Path: []string{"acs", "instance", "show", "{instance_id}"}}
	if got := generatedCommandExamples(argumentOperation, argumentCommand); !slices.Equal(got, []string{"ctyun acs instance show instance-demo"}) {
		t.Fatalf("generated argument examples = %#v", got)
	}
	placeholderOperation := Operation{
		Examples:   []string{"ctyun acs instance show {instance_id}"},
		Parameters: []Parameter{{Name: "instanceID", Type: "String", Argument: "instance_id", ExampleUnavailable: true}},
	}
	if got := generatedCommandExamples(placeholderOperation, argumentCommand); len(got) != 0 {
		t.Fatalf("generated placeholder-only argument examples = %#v, want none", got)
	}

	requiredOperation := Operation{
		Parameters: []Parameter{{Name: "name", Type: "String", CLIName: "name", Example: json.RawMessage(`"captured-name"`)}},
	}
	requiredCommand := plugin.Command{
		Path:       []string{"cbr", "storage", "create"},
		Parameters: []plugin.Parameter{{Name: "name", Flag: "name", Required: true}},
	}
	if got := generatedCommandExamples(requiredOperation, requiredCommand); !slices.Equal(got, []string{"ctyun cbr storage create --name captured-name"}) {
		t.Fatalf("generated required-option examples = %#v", got)
	}
}

// TestGeneratedExamplesUseExplicitUnavailablePlaceholders verifies that a
// reviewed absence remains visible instead of dropping a required option.
func TestGeneratedExamplesUseExplicitUnavailablePlaceholders(t *testing.T) {
	catalog := loadCatalogFixture(t)
	operation := &catalog.Operations[0]
	operation.Parameters[1].Required = true
	operation.Parameters[1].ExampleUnavailable = true
	operation.Examples = nil
	commands := buildCommands(catalog)
	if got := commands.Commands[0].Examples; !slices.Equal(got, []string{"ctyun ecs instance list --name {name}"}) {
		t.Fatalf("generated unavailable example = %#v", got)
	}
}

// TestGeneratedExamplesUseTypedUnavailablePlaceholders verifies that explicit
// source gaps remain visible while the published example still parses.
func TestGeneratedExamplesUseTypedUnavailablePlaceholders(t *testing.T) {
	catalog := loadCatalogFixture(t)
	operation := &catalog.Operations[0]
	operation.Examples = nil
	operation.RequestExample = nil
	operation.Parameters = []Parameter{
		{Name: "count", Location: "body", Required: true, Type: "Integer", CLIName: "count", CLIFlag: "count", ExampleUnavailable: true},
		{Name: "desktopOids", Location: "body", Required: true, Type: "Array of Strings", CLIName: "desktop_oids", CLIFlag: "desktop-oids", ExampleUnavailable: true},
	}

	command := buildCommands(catalog).Commands[0]
	want := "ctyun ecs instance list --count 0 --desktop-oids '[\"{desktop_oids}\"]'"
	if got := command.Examples[0]; got != want {
		t.Fatalf("generated typed unavailable example = %q, want %q", got, want)
	}
	if err := plugin.ValidateCommandExample(command, command.Examples[0]); err != nil {
		t.Fatalf("ValidateCommandExample rejected typed unavailable example: %v", err)
	}

	placeholderCases := []struct {
		typeName string
		want     string
	}{
		{typeName: "Float", want: "0"},
		{typeName: "Boolean", want: "false"},
		{typeName: "Array of Integers", want: "[0]"},
		{typeName: "Array of Objects", want: `[{"value":"{value}"}]`},
		{typeName: "Map of String", want: `{"key":"{value}"}`},
		{typeName: "JSON", want: `{"value":"{value}"}`},
		{typeName: "String", want: "{value}"},
	}
	for _, testCase := range placeholderCases {
		t.Run(testCase.typeName, func(t *testing.T) {
			if got := unavailableParameterExample(Parameter{Type: testCase.typeName}, "value"); got != testCase.want {
				t.Fatalf("unavailableParameterExample(%q) = %q, want %q", testCase.typeName, got, testCase.want)
			}
		})
	}
}

// TestCatalogExampleEvidenceValidation verifies that source examples are typed
// JSON evidence and cannot also be marked unavailable.
func TestCatalogExampleEvidenceValidation(t *testing.T) {
	catalog := loadCatalogFixture(t)
	parameter := &catalog.Operations[0].Parameters[1]
	parameter.Example = json.RawMessage(`"demo"`)
	parameter.ExampleUnavailable = true
	if err := catalog.Validate(); err == nil {
		t.Fatal("Catalog.Validate accepted conflicting example evidence")
	}

	parameter.ExampleUnavailable = false
	parameter.Example = json.RawMessage(`1`)
	if err := catalog.Validate(); err == nil {
		t.Fatal("Catalog.Validate accepted example with the wrong source type")
	}
}

// TestCatalogRequiresSupportedDescriptions verifies that every generated
// command has explicit prose for the public localization scope.
func TestCatalogRequiresSupportedDescriptions(t *testing.T) {
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		catalog := loadCatalogFixture(t)
		delete(catalog.Operations[0].Description, language)
		if err := catalog.Validate(); err == nil {
			t.Fatalf("Catalog.Validate accepted missing %s description", language)
		}
	}
}

// TestCatalogFingerprintIncludesExampleEvidence verifies that accepted request
// evidence advances source provenance.
func TestCatalogFingerprintIncludesExampleEvidence(t *testing.T) {
	catalog := loadCatalogFixture(t)
	before := catalogFingerprint(catalog)
	catalog.Operations[0].Parameters[1].Example = json.RawMessage(`"demo"`)
	if after := catalogFingerprint(catalog); after == before {
		t.Fatal("catalog fingerprint did not change with parameter example evidence")
	}
	catalog = loadCatalogFixture(t)
	before = catalogFingerprint(catalog)
	catalog.Operations[0].RequestExample = map[string]json.RawMessage{"keyword": json.RawMessage(`"demo"`)}
	if after := catalogFingerprint(catalog); after == before {
		t.Fatal("catalog fingerprint did not change with request example evidence")
	}
}

// TestGeneratedExampleHelpersCoverEvidenceFallbacks verifies the deterministic
// fallback order and conditional-branch construction used during draft review.
func TestGeneratedExampleHelpersCoverEvidenceFallbacks(t *testing.T) {
	if got := generatedParameterValueType(Parameter{Type: "String"}); got != "" {
		t.Fatalf("generated string value type = %q", got)
	}
	if got := generatedParameterValueType(Parameter{Type: "Integer"}); got != plugin.ParameterValueInteger {
		t.Fatalf("generated integer value type = %q", got)
	}

	for _, parameter := range []Parameter{
		{Type: "String", Example: json.RawMessage(`{`)},
		{Type: "Binary", Example: json.RawMessage(`1`)},
		{Type: "String", Example: json.RawMessage(`1`)},
		{Type: "Integer", Example: json.RawMessage(`"one"`)},
	} {
		if err := validateParameterExample(parameter); err == nil {
			t.Fatalf("validateParameterExample accepted %#v", parameter)
		}
	}
	if err := validateParameterExample(Parameter{Type: "String", Example: json.RawMessage(`"captured"`)}); err != nil {
		t.Fatalf("validateParameterExample rejected valid string: %v", err)
	}

	operation := Operation{
		RequestExample: map[string]json.RawMessage{
			"mode": json.RawMessage(`"branch"`),
		},
		Parameters: []Parameter{
			{Name: "mode", Type: "String", CLIName: "mode"},
			{Name: "required", Type: "String", CLIName: "required", Example: json.RawMessage(`"captured"`)},
			{Name: "choice", Type: "String", CLIName: "choice", Default: "default-choice"},
			{Name: "enum", Type: "String", CLIName: "enum", Enum: []string{"first", "second"}},
			{Name: "missing", Type: "String", CLIName: "missing"},
		},
	}
	command := plugin.Command{
		Path: []string{"demo", "create"},
		Parameters: []plugin.Parameter{
			{Name: "mode", Flag: "mode", Required: true},
			{Name: "required", Flag: "required"},
			{Name: "choice", Flag: "choice"},
			{Name: "enum", Flag: "enum"},
		},
		ConditionalRequirements: []plugin.ConditionalRequirement{
			{When: plugin.ParameterCondition{Parameter: "mode", Equals: "other"}, Required: []string{"required"}},
			{When: plugin.ParameterCondition{Parameter: "mode", Equals: "branch"}, Required: []string{"required", "mode"}, AnyOf: []string{"choice", "enum"}},
		},
	}
	want := "ctyun demo create --mode branch --required captured --choice default-choice"
	if got := generatedCommandExample(operation, command); got != want {
		t.Fatalf("conditional generated example = %q, want %q", got, want)
	}
	if got := operationParameterExample(operation, "enum"); got != "first" {
		t.Fatalf("enum parameter example = %q", got)
	}
	operation.Parameters[0].Enum = []string{"Demo", "Other"}
	operation.RequestExample["mode"] = json.RawMessage(`"demo"`)
	if got := operationParameterExample(operation, "mode"); got != "Demo" {
		t.Fatalf("case-normalized enum example = %q", got)
	}
	if got := operationParameterExample(operation, "missing"); got != "{missing}" {
		t.Fatalf("missing parameter example = %q", got)
	}
	if got := operationParameterExample(operation, "unknown"); got != "{unknown}" {
		t.Fatalf("unknown parameter example = %q", got)
	}
	parts := appendExampleParameter(operation, command, map[string]string{}, []string{"ctyun"}, "unknown")
	if !slices.Equal(parts, []string{"ctyun"}) {
		t.Fatalf("unknown conditional parameter changed example: %#v", parts)
	}

	additional := generatedCommandExamples(Operation{Examples: []string{"", "ctyun demo list"}}, plugin.Command{Path: []string{"demo", "list"}})
	if len(additional) != 0 {
		t.Fatalf("redundant examples = %#v, want none", additional)
	}
	preferred := generatedCommandExamples(Operation{Examples: []string{"ctyun demo attest --evidence captured"}}, plugin.Command{Path: []string{"demo", "attest"}})
	if !slices.Equal(preferred, []string{"ctyun demo attest --evidence captured"}) {
		t.Fatalf("captured example preference = %#v", preferred)
	}

	if _, ok := sourceExampleText(Parameter{Type: "String"}, json.RawMessage(`1`)); ok {
		t.Fatal("sourceExampleText accepted non-string JSON for string parameter")
	}
	if _, ok := sourceExampleText(Parameter{Type: "String"}, json.RawMessage(`""`)); ok {
		t.Fatal("sourceExampleText accepted an empty required string as concrete evidence")
	}
	if _, ok := sourceExampleText(Parameter{Type: "Binary"}, json.RawMessage(`1`)); ok {
		t.Fatal("sourceExampleText accepted unsupported type")
	}
	if _, ok := sourceExampleText(Parameter{Type: "String"}, json.RawMessage(`{`)); ok {
		t.Fatal("sourceExampleText accepted invalid JSON")
	}
	if got := shellQuoteExampleValue(""); got != "''" {
		t.Fatalf("empty shell value = %q", got)
	}
	if got := shellQuoteExampleValue("it's ready"); got != `'it'\''s ready'` {
		t.Fatalf("quoted shell value = %q", got)
	}
	if name, ok := examplePathArgument("{}"); ok || name != "" {
		t.Fatalf("empty path argument = %q %v", name, ok)
	}
	if exampleConditionMatches(plugin.ParameterCondition{In: []string{"one", "two"}}, "two") != true {
		t.Fatal("in-list example condition did not match")
	}
	if got := operationArgumentExample(Operation{}, "id"); got != "{id}" {
		t.Fatalf("unknown argument example = %q", got)
	}
	argumentOperation := Operation{
		RequestExample: map[string]json.RawMessage{"id": json.RawMessage(`"request-id"`)},
		Parameters:     []Parameter{{Name: "id", Type: "String", Argument: "id", Example: json.RawMessage(`"parameter-id"`)}},
	}
	if got := operationArgumentExample(argumentOperation, "id"); got != "request-id" {
		t.Fatalf("request argument example = %q", got)
	}
	argumentOperation.RequestExample = nil
	if got := operationArgumentExample(argumentOperation, "id"); got != "parameter-id" {
		t.Fatalf("parameter argument example = %q", got)
	}
}

// TestCatalogRejectsInvalidRequestExampleAndSourceType covers catalog-level
// evidence failures before draft generation.
func TestCatalogRejectsInvalidRequestExampleAndSourceType(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].RequestExample = map[string]json.RawMessage{"keyword": json.RawMessage(`{`)}
	if err := catalog.Validate(); err == nil {
		t.Fatal("Catalog.Validate accepted invalid request example JSON")
	}
	catalog = loadCatalogFixture(t)
	catalog.Operations[0].Parameters[0].Type = "Binary"
	if err := catalog.Validate(); err == nil {
		t.Fatal("Catalog.Validate accepted unsupported parameter type")
	}
}
