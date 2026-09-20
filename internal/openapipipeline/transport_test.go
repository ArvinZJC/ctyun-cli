/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestTransportCatalogLifecycle validates round-trip serialization, minimum core, and evidence integrity.
func TestTransportCatalogLifecycle(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations = catalog.Operations[:1]
	op := &catalog.Operations[0]
	op.Parameters = nil
	op.Examples = nil
	op.ExampleResponse = nil
	op.Method = "GET"
	op.Request = nil
	op.Response = Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "xml", XMLRoot: &apicontract.XMLName{Local: "items"}}}}, XML: &apicontract.XMLTable{Rows: []apicontract.XMLName{{Local: "items"}, {Local: "item"}}, Columns: map[string]apicontract.XMLSelector{"value": {}}}, Columns: []Column{{Key: "value", LabelEN: "Value", LabelZH: "值"}}}
	op.Fixture = &client.HTTPFixture{SchemaVersion: 1, Status: 200, BodyBase64: base64.StdEncoding.EncodeToString([]byte("<items><item>001</item></items>"))}
	workspace := Workspace{Root: t.TempDir()}
	if err := workspace.WriteCatalog(workspace.ProductPath("ecs", "source.json"), catalog); err != nil {
		t.Fatal(err)
	}
	if err := workspace.GenerateDraft("ecs"); err != nil {
		t.Fatal(err)
	}
	draft := workspace.ProductPath("ecs", "draft")
	bundle, err := plugin.LoadBundle(draft, version.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(bundle.Manifest.Requires.Ctyun, ">=0.5.0") {
		t.Fatal("missing core floor")
	}
	if _, err = plugin.LoadBundle(draft, "0.4.0"); err == nil {
		t.Fatal("old core accepted")
	}
	review, err := workspace.ReviewDraft("ecs")
	if err != nil || !review.Ready {
		t.Fatalf("review=%+v err=%v", review, err)
	}
	if err = workspace.PromoteDraft("ecs", filepath.Join(workspace.Root, "plugins", "ecs")); err != nil {
		t.Fatal(err)
	}
	baseline, err := workspace.ReadBaseline("ecs")
	if err != nil || !sameExecutionJSON(catalog, baseline) {
		t.Fatalf("baseline mismatch %v", err)
	}
	fixture := op.Fixture
	fixture.BodyBase64 = base64.StdEncoding.EncodeToString([]byte("<items><item>tampered</item></items>"))
	if err = writeJSON(filepath.Join(draft, fixturePath(*op)), fixture); err != nil {
		t.Fatal(err)
	}
	review, err = workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatal(err)
	}
	if review.Ready {
		t.Fatal("tampered fixture accepted")
	}
	if err = os.Remove(filepath.Join(draft, fixturePath(*op))); err != nil {
		t.Fatal(err)
	}
	review, err = workspace.ReviewDraft("ecs")
	if err != nil {
		t.Fatal(err)
	}
	if review.Ready {
		t.Fatal("missing fixture accepted")
	}
}

// TestNumericResponseLabelsRemainData permits protocol status values in localized labels.
func TestNumericResponseLabelsRemainData(t *testing.T) {
	if finding := DisplayLabelQualityFinding("zh-CN", "200 响应次数"); finding != "" {
		t.Fatal(finding)
	}
	if finding := DisplayLabelQualityFinding("zh-CN", "200Custom 响应次数"); finding == "" {
		t.Fatal("untranslated word accepted")
	}
}

// TestTransportEvidenceValidation checks the same representation rules as runtime execution.
func TestTransportEvidenceValidation(t *testing.T) {
	fixture := func(body string) *client.HTTPFixture {
		return &client.HTTPFixture{SchemaVersion: 1, Status: 200, BodyBase64: base64.StdEncoding.EncodeToString([]byte(body))}
	}
	for _, tc := range []struct {
		operation Operation
		valid     bool
	}{
		{Operation{Method: "BAD"}, false},
		{Operation{Method: "GET", Fixture: fixture("{}")}, false},
		{Operation{Method: "GET", Response: Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}}}}, false},
		{Operation{Method: "GET", Fixture: &client.HTTPFixture{}, Response: Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}}}}, false},
		{Operation{Method: "GET", Fixture: fixture("broken"), Response: Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}}}}, false},
		{Operation{Method: "GET", Fixture: fixture("binary"), Response: Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "binary"}}}}}, true},
		{Operation{Method: "GET", Fixture: fixture(`{"code":0}`), Response: Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "json"}}}}}, true},
	} {
		if err := validateOperationTransport(tc.operation); (err == nil) != tc.valid {
			t.Errorf("%+v %v", tc, err)
		}
		if tc.operation.Response.HTTP != nil {
			payload, err := operationExamplePayload(tc.operation)
			want := tc.valid && tc.operation.Response.HTTP.Variants[0].Format == "json"
			if (err == nil) != want {
				t.Errorf("payload %#v, %v", payload, err)
			}
		}
	}
}

// TestBinaryCatalogLifecycle omits meaningless tables and records transport drift.
func TestBinaryCatalogLifecycle(t *testing.T) {
	catalog := loadCatalogFixture(t)
	catalog.Operations = catalog.Operations[:1]
	op := &catalog.Operations[0]
	op.Method = "GET"
	op.Parameters = nil
	op.Examples = nil
	op.ExampleResponse = nil
	op.Download = true
	op.Response = Response{HTTP: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "binary"}}}}
	op.Fixture = &client.HTTPFixture{SchemaVersion: 1, Status: 200, BodyBase64: base64.StdEncoding.EncodeToString([]byte("bytes"))}
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
	if len(bundle.Tables.Tables) != 0 || bundle.Commands.Commands[0].Table != "" {
		t.Fatal("binary table generated")
	}
	var report DiffReport
	compareOperationMetadata(&report, Operation{}, *op)
	if len(report.Changes) == 0 {
		t.Fatal("missing transport drift")
	}
	if err := os.WriteFile(filepath.Join(workspace.Root, "blocked"), []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeFixtures(filepath.Join(workspace.Root, "blocked"), catalog); err == nil {
		t.Fatal("missing fixture destination accepted")
	}
}

// TestXMLProjectionDriftBlocksReview prevents table edits from bypassing source evidence.
func TestXMLProjectionDriftBlocksReview(t *testing.T) {
	operation := loadCatalogFixture(t).Operations[0]
	command := plugin.Command{Operation: operation.ID, Table: "table"}
	var report ReviewReport
	reviewTableLabels(&report, Catalog{Operations: []Operation{operation}}, plugin.Commands{Commands: []plugin.Command{command}}, plugin.Tables{Tables: map[string]plugin.Table{"table": {XML: &apicontract.XMLTable{Rows: []apicontract.XMLName{{Local: "unexpected"}}}}}})
	if len(report.Findings) == 0 {
		t.Fatal("projection drift ignored")
	}
	operation.Examples = []string{"ctyun ecs instance list --region region-example"}
	command.Path = []string{"ecs", "instance", "list"}
	if !operationHasSourceCommandExample(operation, command) {
		t.Fatal("informative example lost")
	}
	catalog := loadCatalogFixture(t)
	catalog.Operations[0].Response.HTTP = &apicontract.Response{}
	if err := catalog.Validate(); err == nil {
		t.Fatal("invalid HTTP contract accepted")
	}
}
