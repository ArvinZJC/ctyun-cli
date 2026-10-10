/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"sort"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// requestLocationParameters maps explicitly bound inputs to their wire fields in
// one request location. Inputs bound elsewhere cannot leak through legacy target
// fallback; unbound parameters retain that fallback for compact plugin metadata.
func requestLocationParameters(operation plugin.Operation, bindings map[string]string, parameters []plugin.Parameter) []plugin.Parameter {
	bound := map[string]bool{}
	explicit := map[string]bool{}
	for _, fields := range []map[string]string{operation.Body, operation.Query, operation.Headers} {
		for field, source := range fields {
			if name, ok := strings.CutPrefix(source, "$param."); ok {
				bound[name] = true
				explicit[name] = true
			}
			if source == "$profile.region" {
				for _, parameter := range parameters {
					if parameter.Target == field {
						bound[parameter.Name] = true
					}
				}
			}
		}
	}
	if operation.Request != nil {
		sources := []string{operation.Request.Document}
		for _, part := range operation.Request.Parts {
			sources = append(sources, part.Source)
		}
		for _, source := range sources {
			if name, ok := strings.CutPrefix(source, "$param."); ok {
				bound[name] = true
				explicit[name] = true
			}
		}
	}
	keys := make([]string, 0, len(bindings))
	for key := range bindings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]plugin.Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		matched := false
		for _, key := range keys {
			source := bindings[key]
			if source != "$param."+parameter.Name && !(source == "$profile.region" && key == parameter.Target && (!explicit[parameter.Name] || parameter.Name == "region")) {
				continue
			}
			copyParameter := parameter
			copyParameter.Target = key
			result = append(result, copyParameter)
			matched = true
		}
		if matched || bound[parameter.Name] {
			continue
		}
		if source := bindings[parameter.Target]; strings.HasPrefix(source, "$param.") || strings.HasPrefix(source, "$arg.") {
			continue
		}
		result = append(result, parameter)
	}
	return result
}
