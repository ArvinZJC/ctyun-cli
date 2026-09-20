/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package apicontract

import (
	"encoding/json"
	"strings"
)

// XMLName identifies an element or attribute by namespace URI and local name.
type XMLName struct {
	Space string `json:"space,omitempty"`
	Local string `json:"local"`
}

// XMLSelector selects all matching child elements and their text or one attribute.
type XMLSelector struct {
	Name      bool      `json:"name,omitempty"`
	Path      []XMLName `json:"path,omitempty"`
	Attribute *XMLName  `json:"attribute,omitempty"`
}

// XMLTable declares an absolute row path and relative column selectors.
// Empty column paths select the row itself; repeated values remain ordered.
type XMLTable struct {
	RowPaths [][]XMLName            `json:"row_paths,omitempty"`
	Rows     []XMLName              `json:"rows,omitempty"`
	Columns  map[string]XMLSelector `json:"columns"`
}

// Validate rejects malformed XML names and ambiguous projection declarations.
func (table XMLTable) Validate() error {
	if (len(table.Rows) == 0) == (len(table.RowPaths) == 0) || len(table.Columns) == 0 {
		return Invalid("table.xml")
	}
	validName := func(name XMLName) bool { return name.Local != "" && !strings.ContainsAny(name.Local, "{}:/ \t\r\n") }
	paths := table.RowPaths
	if len(paths) == 0 {
		paths = [][]XMLName{table.Rows}
	}
	seen := map[string]bool{}
	for _, path := range paths {
		if len(path) == 0 {
			return Invalid("table.xml.rows")
		}
		for _, name := range path {
			if !validName(name) {
				return Invalid("table.xml.rows")
			}
		}
		encoded, _ := json.Marshal(path)
		if seen[string(encoded)] {
			return Invalid("table.xml.rows")
		}
		seen[string(encoded)] = true
	}
	for key, selector := range table.Columns {
		if key == "" || selector.Name && selector.Attribute != nil {
			return Invalid("table.xml.columns")
		}
		for _, name := range selector.Path {
			if !validName(name) {
				return Invalid("table.xml.columns")
			}
		}
		if selector.Attribute != nil && !validName(*selector.Attribute) {
			return Invalid("table.xml.attribute")
		}
	}
	return nil
}
