/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestUnavailableFixturePreservesPublishedCommand requires explicit evidence instead of fabricated success fixtures.
func TestUnavailableFixturePreservesPublishedCommand(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations = catalog.Operations[:1]
	op := &catalog.Operations[0]
	op.ExampleResponse = nil
	op.Response = Response{RowPath: "$", Columns: []Column{{Key: "message", Path: "message", LabelEN: "Message", LabelZH: "消息"}}, HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}, Success: []apicontract.Check{{Path: "statusCode", Values: []string{"0"}}}}}
	if err := json.Unmarshal([]byte(`{"fixture_unavailable":"Published example contains only a business error; original retained in captured evidence"}`), op); err != nil {
		t.Fatal(err)
	}
	workspace := Workspace{Root: t.TempDir()}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatal(err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatal(err)
	}
	bundle, err := plugin.LoadBundle(workspace.ProductPath("ecs", "draft"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Commands.Commands[0].FixtureResponse != "" {
		t.Fatal("fabricated fixture advertised")
	}
	entries, err := os.ReadDir(filepath.Join(workspace.ProductPath("ecs", "draft"), "fixtures"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("fabricated fixture written")
	}
	report, err := workspace.ReviewDraft("ecs")
	if err != nil || !report.Ready {
		t.Fatalf("%+v %v", report, err)
	}
}

// TestUnavailableFixtureRejectsConflictingEvidence prevents absence markers from bypassing captured-response validation.
func TestUnavailableFixtureRejectsConflictingEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Operation)
	}{
		{"blank reason", func(op *Operation) { op.FixtureUnavailable = " " }},
		{"legacy contract", func(op *Operation) { op.Response.HTTP = nil }},
		{"captured response", func(op *Operation) { op.ExampleResponse = json.RawMessage(`{}`) }},
		{"captured fixture", func(op *Operation) { op.Fixture = &client.HTTPFixture{SchemaVersion: 1, Status: 200} }},
		{"xml projection", func(op *Operation) { op.Response.XML = &apicontract.XMLTable{} }},
		{"legacy status override", func(op *Operation) { op.Response.AcceptedStatuses = []plugin.AcceptedStatusRule{{}} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := Operation{Method: "GET", Path: "/example", FixtureUnavailable: "Only an error response was captured", Response: Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}}}}
			tc.change(&op)
			if err := validateOperationTransport(op); err == nil {
				t.Fatal("conflicting evidence accepted")
			}
		})
	}
}
