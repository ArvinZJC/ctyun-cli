/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestPluginCommandTypedBodyValues verifies that API request bodies preserve
// the scalar and composite shapes declared by command metadata.
func TestPluginCommandTypedBodyValues(t *testing.T) {
	command := typedValueCommand()
	bundle := plugin.Bundle{
		Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://ctapi.example.test"}},
		APIs: plugin.APIs{Operations: map[string]plugin.Operation{
			command.Operation: {
				Method:      http.MethodPost,
				Path:        "/v1/typed",
				ContentType: "application/json",
				Body: map[string]string{
					"count":   "$param.count",
					"ratio":   "$param.ratio",
					"enabled": "$param.enabled",
					"names":   "$param.names",
					"counts":  "$param.counts",
					"objects": "$param.objects",
					"labels":  "$param.labels",
					"payload": "$param.payload",
				},
			},
		}},
	}
	values := map[string]string{
		"count":   "9223372036854775808",
		"ratio":   "1.25",
		"enabled": "true",
		"names":   `["one","two"]`,
		"counts":  `[1,9223372036854775808]`,
		"objects": `[{"id":1}]`,
		"labels":  `{"env":"test"}`,
		"payload": `{"nested":[false,2]}`,
	}

	var got map[string]any
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		decoder := json.NewDecoder(request.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		return typedValueResponse(), nil
	})
	_, err := executeAPICommand(bundle, command, nil, values, typedValueProfile(), func(string) string { return "" }, transport, nil, nil, "en-US")
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"count":   json.Number("9223372036854775808"),
		"ratio":   json.Number("1.25"),
		"enabled": true,
		"names":   []any{"one", "two"},
		"counts":  []any{json.Number("1"), json.Number("9223372036854775808")},
		"objects": []any{map[string]any{"id": json.Number("1")}},
		"labels":  map[string]any{"env": "test"},
		"payload": map[string]any{"nested": []any{false, json.Number("2")}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request body = %#v, want %#v", got, want)
	}
}

// TestPluginCommandInvalidTypedValueStopsBeforeRequest verifies that invalid
// typed input produces a localized visible-option error without network I/O.
func TestPluginCommandInvalidTypedValueStopsBeforeRequest(t *testing.T) {
	command := typedValueCommand()
	if _, err := parseCommandParameters(command, []string{"--enabled", "yes"}, "en-US"); err == nil || !strings.Contains(err.Error(), `invalid value "yes" for option "--enabled": expected boolean`) {
		t.Fatalf("parseCommandParameters error = %v", err)
	}
	called := false
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return typedValueResponse(), nil
	})
	bundle := plugin.Bundle{
		Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://ctapi.example.test"}},
		APIs: plugin.APIs{Operations: map[string]plugin.Operation{
			command.Operation: {Method: http.MethodPost, Path: "/v1/typed", Body: map[string]string{"enabled": "$param.enabled"}},
		}},
	}

	_, err := executeAPICommand(bundle, command, nil, map[string]string{"enabled": "yes"}, typedValueProfile(), func(string) string { return "" }, transport, nil, nil, "en-US")
	if err == nil || !strings.Contains(err.Error(), `invalid value "yes" for option "--enabled": expected boolean`) {
		t.Fatalf("executeAPICommand error = %v", err)
	}
	if called {
		t.Fatal("transport called for invalid typed input")
	}

	called = false
	bundle.APIs.Operations[command.Operation] = plugin.Operation{
		Method: http.MethodGet,
		Path:   "/v1/typed",
		Query:  map[string]string{"enabled": "$param.enabled"},
	}
	_, err = executeAPICommand(bundle, command, nil, map[string]string{"enabled": "yes"}, typedValueProfile(), func(string) string { return "" }, transport, nil, nil, "en-US")
	if err == nil || !strings.Contains(err.Error(), `invalid value "yes" for option "--enabled": expected boolean`) {
		t.Fatalf("query executeAPICommand error = %v", err)
	}
	if called {
		t.Fatal("transport called for invalid typed query input")
	}
}

// TestTypedQueryValuesUseCompactJSON verifies that composite query values use
// one compact JSON string while scalar and header values remain text.
func TestTypedQueryValuesUseCompactJSON(t *testing.T) {
	command := plugin.Command{Parameters: []plugin.Parameter{
		{Name: "ids", Flag: "ids", Target: "ids", ValueType: plugin.ParameterValueStringArray},
		{Name: "token", Flag: "token", Target: "X-Token"},
	}}
	query, err := resolveQueryMap(
		map[string]string{"ids": "$param.ids"},
		coreconfig.Profile{}, nil,
		map[string]string{"ids": `[ "one", "two" ]`},
		command.Parameters, "en-US",
	)
	if err != nil {
		t.Fatal(err)
	}
	if query["ids"] != `["one","two"]` {
		t.Fatalf("query ids = %q", query["ids"])
	}
	header := resolveMap(map[string]string{"X-Token": "$param.token"}, coreconfig.Profile{}, nil, map[string]string{"token": "001"}, command.Parameters, false)
	if header["X-Token"] != "001" {
		t.Fatalf("header value = %q", header["X-Token"])
	}
}

// TestParameterValuePlaceholderUsesSharedSyntax verifies that plugin help uses
// fixed machine-oriented placeholders instead of repeating option names.
func TestParameterValuePlaceholderUsesSharedSyntax(t *testing.T) {
	tests := []struct {
		parameter plugin.Parameter
		want      string
	}{
		{parameter: plugin.Parameter{Flag: "name"}, want: "value"},
		{parameter: plugin.Parameter{Flag: "mode", AllowedValues: []string{"one", "two"}}, want: "one|two"},
		{parameter: plugin.Parameter{Flag: "ids", ValueType: plugin.ParameterValueStringArray}, want: "json"},
		{parameter: plugin.Parameter{Flag: "counts", ValueType: plugin.ParameterValueIntegerArray}, want: "json"},
		{parameter: plugin.Parameter{Flag: "objects", ValueType: plugin.ParameterValueObjectArray}, want: "json"},
		{parameter: plugin.Parameter{Flag: "labels", ValueType: plugin.ParameterValueStringMap}, want: "json"},
		{parameter: plugin.Parameter{Flag: "payload", ValueType: plugin.ParameterValueJSON}, want: "json"},
		{parameter: plugin.Parameter{Flag: "count", ValueType: plugin.ParameterValueInteger}, want: "value"},
	}
	for _, tc := range tests {
		if got := parameterValuePlaceholder(tc.parameter); got != tc.want {
			t.Fatalf("parameterValuePlaceholder(%#v) = %q, want %q", tc.parameter, got, tc.want)
		}
	}
	if got := localizedInvalidOptionValueType(plugin.Parameter{Flag: "name"}, "bad", "en-US").Error(); got != `invalid value "bad" for option "--name": expected string` {
		t.Fatalf("default string type error = %q", got)
	}
}

// typedValueCommand returns the synthetic command declaration shared by typed
// request tests.
func typedValueCommand() plugin.Command {
	return plugin.Command{
		ID:        "demo.typed.create",
		Operation: "demo.typed.create",
		Parameters: []plugin.Parameter{
			{Name: "count", Flag: "count", Target: "count", ValueType: plugin.ParameterValueInteger},
			{Name: "ratio", Flag: "ratio", Target: "ratio", ValueType: plugin.ParameterValueNumber},
			{Name: "enabled", Flag: "enabled", Target: "enabled", ValueType: plugin.ParameterValueBoolean},
			{Name: "names", Flag: "names", Target: "names", ValueType: plugin.ParameterValueStringArray},
			{Name: "counts", Flag: "counts", Target: "counts", ValueType: plugin.ParameterValueIntegerArray},
			{Name: "objects", Flag: "objects", Target: "objects", ValueType: plugin.ParameterValueObjectArray},
			{Name: "labels", Flag: "labels", Target: "labels", ValueType: plugin.ParameterValueStringMap},
			{Name: "payload", Flag: "payload", Target: "payload", ValueType: plugin.ParameterValueJSON},
		},
	}
}

// typedValueProfile returns credentials suitable for the in-memory signed
// request tests.
func typedValueProfile() coreconfig.Profile {
	return coreconfig.Profile{EndpointURL: "https://ctapi.example.test", AccessKey: "ak-test", SecretKey: "sk-test"}
}

// typedValueResponse returns a minimal successful CTyun JSON response.
func typedValueResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"returnObj":{}}`))}
}
