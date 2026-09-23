/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// TestXMLPreservesNamesOrderAndProjection checks namespace identity and lexical values.
func TestXMLPreservesNamesOrderAndProjection(t *testing.T) {
	root, err := DecodeXML([]byte(`<r xmlns="urn:a">before<x a="1">001</x>between<x>2</x>after</r>`))
	if err != nil {
		t.Fatal(err)
	}
	if root.Name.Space != "urn:a" || len(root.Content) != 5 || root.Content[2].Text != "between" {
		t.Fatalf("root: %#v", root)
	}
	table := apicontract.XMLTable{Rows: []XMLName{{Space: "urn:a", Local: "r"}, {Space: "urn:a", Local: "x"}}, Columns: map[string]apicontract.XMLSelector{"text": {}, "attr": {Attribute: &XMLName{Local: "a"}}}}
	rows, err := ProjectXML(root, table)
	if err != nil || len(rows) != 2 || rows[0]["text"] != "001" || rows[0]["attr"] != "1" || rows[1]["text"] != "2" {
		t.Fatalf("rows=%#v err=%v", rows, err)
	}
	table.Rows = table.Rows[:1]
	table.Columns["text"] = apicontract.XMLSelector{Path: []XMLName{{Space: "urn:a", Local: "x"}}}
	rows, err = ProjectXML(root, table)
	if err != nil || rows[0]["text"] != "001, 2" {
		t.Fatalf("repeated=%#v %v", rows, err)
	}
	table.Columns["text"] = apicontract.XMLSelector{}
	rows, err = ProjectXML(root, table)
	if err != nil || rows[0]["text"] != "before001between2after" {
		t.Fatalf("mixed=%#v %v", rows, err)
	}
	table.Rows[0].Space = "wrong"
	if _, err = ProjectXML(root, table); err == nil {
		t.Fatal("wrong namespace accepted")
	}
	if _, err = ProjectXML(root, apicontract.XMLTable{}); err == nil {
		t.Fatal("invalid table accepted")
	}
}

// TestXMLRejectsUnsafeOrMalformedDocuments exercises bounded parsing, not entity expansion.
func TestXMLRejectsUnsafeOrMalformedDocuments(t *testing.T) {
	for _, input := range []string{"", "plain", "<a/ ><b/>", "<a/><b/>", "<a>", "<a>&missing;</a>", "<!DOCTYPE a><a/>", "<?other x?><a/>", "<a/><?xml x?>", strings.Repeat("<a>", 129) + strings.Repeat("</a>", 129), strings.Repeat("x", MaxStructuredBody+1)} {
		if _, err := DecodeXML([]byte(input)); err == nil {
			t.Errorf("accepted %.80q", input)
		}
	}
	if _, err := DecodeXML([]byte("<?xml version=\"1.0\"?>\n<!-- comment --><a/>\n")); err != nil {
		t.Fatal(err)
	}
}

// TestXMLRowAlternativesPreserveDocumentOrder keeps heterogeneous result collections visible.
func TestXMLRowAlternativesPreserveDocumentOrder(t *testing.T) {
	root, err := DecodeXML([]byte(`<r><group><id>g</id></group><user><id>u</id></user><group><id>g2</id></group></r>`))
	if err != nil {
		t.Fatal(err)
	}
	table := apicontract.XMLTable{RowPaths: [][]XMLName{{{Local: "r"}, {Local: "user"}}, {{Local: "r"}, {Local: "group"}}}, Columns: map[string]apicontract.XMLSelector{"kind": {Name: true}, "id": {Path: []XMLName{{Local: "id"}}}}}
	rows, err := ProjectXML(root, table)
	if err != nil || len(rows) != 3 || rows[0]["kind"] != "group" || rows[0]["id"] != "g" || rows[1]["id"] != "u" || rows[2]["id"] != "g2" {
		t.Fatalf("rows=%#v %v", rows, err)
	}
	table.Rows = []XMLName{{Local: "r"}}
	if _, err = ProjectXML(root, table); err == nil {
		t.Fatal("ambiguous rows accepted")
	}
}

// TestXMLOverlappingRowPathsRetainDescendants makes path order independent of traversal.
func TestXMLOverlappingRowPathsRetainDescendants(t *testing.T) {
	root, err := DecodeXML([]byte(`<r><child>value</child></r>`))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := ProjectXML(root, apicontract.XMLTable{RowPaths: [][]XMLName{{{Local: "r"}}, {{Local: "r"}, {Local: "child"}}}, Columns: map[string]apicontract.XMLSelector{"name": {Name: true}}})
	if err != nil || len(rows) != 2 || rows[1]["name"] != "child" {
		t.Fatalf("%#v %v", rows, err)
	}
}
