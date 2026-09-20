/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package waiter

import (
	"encoding/json"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// xmlState reads one exact scalar element from the lossless HTTP XML representation.
// Missing, repeated and structured states are errors; empty scalar text stays pending.
func xmlState(payload map[string]any, path []apicontract.XMLName) (any, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var root client.XMLNode
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(path) == 0 || root.Name != path[0] {
		return nil, apicontract.Invalid("waiter.xml_path")
	}
	nodes := []*client.XMLNode{&root}
	for _, name := range path[1:] {
		var next []*client.XMLNode
		for _, node := range nodes {
			for _, item := range node.Content {
				if item.Element != nil && item.Element.Name == name {
					next = append(next, item.Element)
				}
			}
		}
		nodes = next
	}
	if len(nodes) != 1 {
		return nil, apicontract.Invalid("waiter.xml_path")
	}
	var value strings.Builder
	for _, item := range nodes[0].Content {
		if item.Element != nil {
			return nil, diagnostic.New("error.waiter_expected_scalar", "xml_path")
		}
		value.WriteString(item.Text)
	}
	if value.Len() == 0 {
		return nil, nil
	}
	return value.String(), nil
}
