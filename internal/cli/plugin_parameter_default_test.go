/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestInvalidPluginParameterDefaultDiagnosticIsFullyLocalized(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ecs")
	writeValidationBundle(t, dir)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [{
    "id": "ecs.instance.list",
    "path": ["ecs", "instance", "list"],
    "operation": "v4.ecs.instance.list",
    "table": "ecs.instance.list",
    "parameters": [{
      "name": "count",
      "flag": "count",
      "target": "count",
      "value_type": "integer",
      "default": "three",
      "description": "Count"
    }]
  }]
}`)

	_, err := plugin.LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error")
	}
	got := localizedError(err, "zh-CN")
	if want := `选项 --count 的默认值 "three" 对值类型 integer 无效`; !strings.Contains(got, want) {
		t.Fatalf("localized diagnostic = %q, want %q", got, want)
	}
	if strings.Contains(got, "expected JSON integer") {
		t.Fatalf("localized diagnostic contains raw English parser cause: %q", got)
	}
}

func TestPluginParameterHelpLocalizesDefaultHintAndPreservesToken(t *testing.T) {
	command := plugin.Command{ID: "demo.create"}
	parameter := plugin.Parameter{
		Name:        "mode",
		Flag:        "mode",
		Target:      "mode",
		Description: "Mode",
		Default:     "CrossAZ",
	}
	bundle := plugin.Bundle{I18N: map[string]map[string]string{
		"zh-CN": {"parameter.demo.create.mode.description": "模式"},
	}}

	if got := parameterHelpDescription(bundle, command, parameter, "en-US"); got != "Mode (default: CrossAZ)" {
		t.Fatalf("English parameter help = %q", got)
	}
	if got := parameterHelpDescription(bundle, command, parameter, "zh-CN"); got != "模式（默认：CrossAZ）" {
		t.Fatalf("Chinese parameter help = %q", got)
	}
}

func TestPluginParameterDefaultDoesNotPopulateOmittedRequestInput(t *testing.T) {
	command := plugin.Command{
		ID:        "demo.create",
		Operation: "demo.create",
		Parameters: []plugin.Parameter{{
			Name:    "mode",
			Flag:    "mode",
			Target:  "mode",
			Default: "CrossAZ",
		}},
	}
	values, err := parseCommandParameters(command, nil, "en-US")
	if err != nil {
		t.Fatalf("parseCommandParameters returned error: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("parsed omitted values = %#v, want empty", values)
	}

	bundle := plugin.Bundle{
		Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://ctapi.example.test"}},
		APIs: plugin.APIs{Operations: map[string]plugin.Operation{
			command.Operation: {
				Method: http.MethodPost,
				Path:   "/v1/demo",
				Body: map[string]string{
					"name": "demo",
					"mode": "$param.mode",
				},
			},
		}},
	}
	var requestBody map[string]any
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		return typedValueResponse(), nil
	})
	profile := coreconfig.Profile{AccessKey: "ak-test", SecretKey: "sk-test"}
	if _, err := executeAPICommand(bundle, command, nil, values, profile, func(string) string { return "" }, transport, nil, nil, "en-US"); err != nil {
		t.Fatalf("executeAPICommand returned error: %v", err)
	}
	if _, exists := requestBody["mode"]; exists {
		t.Fatalf("request body = %#v, default metadata populated omitted mode", requestBody)
	}
	if requestBody["name"] != "demo" {
		t.Fatalf("request body = %#v, want static request input", requestBody)
	}
}
