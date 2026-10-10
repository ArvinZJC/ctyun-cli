/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestExplicitRequestLocationsRemainIndependent prevents header and query inputs
// from overwriting body fields that have the same wire name or table target.
func TestExplicitRequestLocationsRemainIndependent(t *testing.T) {
	command := plugin.Command{Operation: "demo", Parameters: []plugin.Parameter{
		{Name: "header_id", Target: "id"},
		{Name: "body_id", Target: "id", ValueType: plugin.ParameterValueInteger},
		{Name: "query_ids", Target: "ids", ValueType: plugin.ParameterValueIntegerArray},
		{Name: "legacy", Target: "legacy"},
		{Name: "region", Target: "regionID"},
	}}
	operation := plugin.Operation{Method: "POST", Path: "/demo", Headers: map[string]string{"id": "$param.header_id"}, Query: map[string]string{"ids": "$param.query_ids"}, Body: map[string]string{"id": "$param.body_id", "regionID": "$profile.region"}}
	bundle := plugin.Bundle{Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://example.test"}}, APIs: plugin.APIs{Operations: map[string]plugin.Operation{"demo": operation}}}
	spec, err := buildAPIRequest(bundle, command, nil, map[string]string{"header_id": "header", "body_id": "42", "query_ids": "[1,2]", "legacy": "retained", "region": "explicit-region"}, coreconfig.Profile{Region: "profile-region"}, func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] }, nil, nil, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Headers["id"] != "header" {
		t.Fatalf("header overwritten: %#v", spec.Headers)
	}
	var body map[string]any
	if err := json.Unmarshal(spec.Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 3 || body["id"] != float64(42) || body["legacy"] != "retained" || body["regionID"] != "explicit-region" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if spec.Query != "ids=%5B1%2C2%5D" {
		t.Fatalf("unexpected query: %s", spec.Query)
	}
}

// TestExplicitBindingNamesControlTypes preserves typed values when wire field
// names differ from table targets and prevents cross-location profile overrides.
func TestExplicitBindingNamesControlTypes(t *testing.T) {
	parameters := []plugin.Parameter{
		{Name: "query_region", Target: "regionID"},
		{Name: "region", Target: "regionID"},
		{Name: "items", Target: "display_items", ValueType: plugin.ParameterValueIntegerArray},
		{Name: "unused", Target: "wire_items"},
	}
	operation := plugin.Operation{Body: map[string]string{"wire_items": "$param.items"}, Query: map[string]string{"regionID": "$param.query_region"}, Headers: map[string]string{"regionID": "$profile.region"}}
	values := map[string]string{"query_region": "query-only", "region": "header-only", "items": "[1,9223372036854775808]", "unused": "must-not-overwrite"}
	headers := resolveMap(operation.Headers, coreconfig.Profile{Region: "profile"}, nil, values, requestLocationParameters(operation, operation.Headers, parameters), false)
	if headers["regionID"] != "header-only" {
		t.Fatalf("header=%v", headers)
	}
	query, err := resolveQueryMap(operation.Query, coreconfig.Profile{}, nil, values, requestLocationParameters(operation, operation.Query, parameters), "en-US")
	if err != nil || query["regionID"] != "query-only" {
		t.Fatalf("query=%v err=%v", query, err)
	}
	body, err := resolveRequestBody(operation.Body, coreconfig.Profile{}, nil, values, requestLocationParameters(operation, operation.Body, parameters), "en-US")
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(body)
	if string(encoded) != `{"wire_items":[1,9223372036854775808]}` {
		t.Fatalf("body=%s", encoded)
	}
}

// TestMultipartInputsDoNotOverrideHeaders keeps file and expanded-part inputs
// scoped to the multipart encoder when they share a target with an HTTP header.
func TestMultipartInputsDoNotOverrideHeaders(t *testing.T) {
	operation := plugin.Operation{Headers: map[string]string{"id": "fixed-header"}, Request: &apicontract.Request{Encoding: "multipart", Parts: []apicontract.Part{{Name: "file", Source: "$param.file"}, {Name: "meta-", Source: "$param.metadata", Expand: true}}}}
	parameters := []plugin.Parameter{{Name: "file", Target: "id"}, {Name: "metadata", Target: "id"}}
	got := resolveMap(operation.Headers, coreconfig.Profile{}, nil, map[string]string{"file": "private-file", "metadata": "private-metadata"}, requestLocationParameters(operation, operation.Headers, parameters), false)
	if len(got) != 1 || got["id"] != "fixed-header" {
		t.Fatalf("multipart input leaked into headers: %v", got)
	}
}
