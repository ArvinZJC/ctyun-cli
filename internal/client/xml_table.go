/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// ProjectXML preserves repeated rows and joins repeated column values in document order.
// Paths match exact namespace URIs; they never infer namespaces from local names.
func ProjectXML(root *XMLNode, table apicontract.XMLTable) ([]map[string]string, error) {
	if err := table.Validate(); err != nil {
		return nil, err
	}
	paths := table.RowPaths
	if len(paths) == 0 {
		paths = [][]XMLName{table.Rows}
	}
	for _, path := range paths {
		if root == nil || root.Name != path[0] {
			return nil, apicontract.Invalid("table.xml.root")
		}
	}
	nodes := xmlRows(root, paths, nil)
	rows := make([]map[string]string, 0, len(nodes))
	for _, node := range nodes {
		row := make(map[string]string, len(table.Columns))
		for key, selector := range table.Columns {
			var values []string
			for _, selected := range xmlChildren([]*XMLNode{node}, selector.Path) {
				if selector.Name {
					values = append(values, selected.Name.Local)
				} else if selector.Attribute != nil {
					for _, attr := range selected.Attributes {
						if attr.Name == *selector.Attribute {
							values = append(values, attr.Value)
						}
					}
				} else {
					values = append(values, xmlText(selected))
				}
			}
			row[key] = strings.Join(values, ", ")
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// xmlChildren follows an exact sequence of child names, retaining every match.
func xmlChildren(nodes []*XMLNode, path []apicontract.XMLName) []*XMLNode {
	for _, name := range path {
		var next []*XMLNode
		for _, node := range nodes {
			for _, item := range node.Content {
				if item.Element != nil && item.Element.Name == name {
					next = append(next, item.Element)
				}
			}
		}
		nodes = next
	}
	return nodes
}

// xmlText concatenates descendant character data without trimming meaningful whitespace.
func xmlText(node *XMLNode) string {
	var text strings.Builder
	for _, item := range node.Content {
		if item.Element != nil {
			text.WriteString(xmlText(item.Element))
		} else {
			text.WriteString(item.Text)
		}
	}
	return text.String()
}

// xmlRows follows declared absolute paths while retaining the source document order.
func xmlRows(node *XMLNode, paths [][]XMLName, prefix []XMLName) []*XMLNode {
	current := append(slices.Clone(prefix), node.Name)
	var rows []*XMLNode
	descend := false
	for _, path := range paths {
		if slices.Equal(current, path) {
			rows = append(rows, node)
		}
		if len(path) > len(current) && slices.Equal(current, path[:len(current)]) {
			descend = true
		}
	}
	if descend {
		for _, item := range node.Content {
			if item.Element != nil {
				rows = append(rows, xmlRows(item.Element, paths, current)...)
			}
		}
	}
	return rows
}
