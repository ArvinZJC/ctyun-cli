/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// expandMultipartPart expands a string map at its declared position in stable key order.
func expandMultipartPart(prefix, value string) ([]client.BodyPart, error) {
	parsed, err := plugin.ParseParameterValue(plugin.Parameter{ValueType: plugin.ParameterValueStringMap}, value)
	if err != nil {
		return nil, apicontract.Invalid("request.parts.map")
	}
	fields := parsed.(map[string]any)
	keys := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for key := range fields {
		name := prefix + key
		if key == "" || !apicontract.HeaderName(name) || seen[strings.ToLower(name)] {
			return nil, apicontract.Invalid("request.parts.name")
		}
		seen[strings.ToLower(name)] = true
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]client.BodyPart, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, client.BodyPart{Name: prefix + key, Value: fields[key].(string)})
	}
	return parts, nil
}
