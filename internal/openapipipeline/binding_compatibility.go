/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// catalogNeedsBindingIsolation identifies explicit request mappings that older
// cores could overwrite or copy into another location through parameter targets.
func catalogNeedsBindingIsolation(catalog Catalog) bool {
	for _, operation := range catalog.Operations {
		body := map[string]string{}
		for _, parameter := range operation.Parameters {
			if parameter.Location == "body" {
				body[parameter.Name] = parameterBinding(parameter)
			}
		}
		inputsByTarget := map[string]string{}
		for _, parameter := range operation.Parameters {
			name, _, target, _ := commandParameterMetadata(parameter)
			if name == "" {
				continue
			}
			if parameter.TableTarget != "" && parameter.TableTarget != parameter.Name {
				return true
			}
			if previous, ok := inputsByTarget[target]; ok && previous != name {
				return true
			}
			inputsByTarget[target] = name
			if operation.Request == nil && len(body) > 0 && parameter.Location != "body" && body[parameter.Name] != parameterBinding(parameter) {
				return true
			}
		}
	}
	return false
}

// minimumCoreRequirement raises the lower bound while preserving upper bounds
// and any stricter existing minimum.
func minimumCoreRequirement(constraint, minimum string) string {
	for part := range strings.FieldsSeq(constraint) {
		if strings.HasPrefix(part, ">=") && version.CompareSemanticVersions(strings.TrimPrefix(part, ">="), minimum) >= 0 {
			return constraint
		}
	}
	parts := []string{">=" + minimum}
	for part := range strings.FieldsSeq(constraint) {
		if !strings.HasPrefix(part, ">=") {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " ")
}
