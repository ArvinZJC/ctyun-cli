/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// CatalogWaiter keeps reviewed upstream state semantics next to the definition
// that is promoted into a runtime bundle. Evidence is not shipped to users.
type CatalogWaiter struct {
	plugin.Waiter
	Evidence string `json:"evidence"`
}

// buildWaiters preserves explicit reviewed definitions without inferring state
// semantics from a command name or response field name.
func buildWaiters(catalog Catalog) plugin.Waiters {
	specs := make(map[string]plugin.Waiter, len(catalog.Waiters))
	for id, spec := range catalog.Waiters {
		specs[id] = spec.Waiter
	}
	return plugin.Waiters{Waiters: specs}
}

// validateWaiters requires evidence, bounded polling, and explicit safe command
// bindings for catalog-maintained waiters, beyond legacy runtime validation.
func (catalog Catalog) validateWaiters() error {
	if len(catalog.Waiters) == 0 {
		return nil
	}
	for id, spec := range catalog.Waiters {
		if strings.TrimSpace(spec.Evidence) == "" || len(spec.Commands) == 0 || spec.MaxAttempts <= 0 || spec.IntervalSeconds <= 0 {
			return fmt.Errorf("waiter %s requires evidence, command bindings, and positive polling limits", id)
		}
	}
	for id, spec := range catalog.Waiters {
		for _, operation := range catalog.Operations {
			if !slices.Contains(spec.Commands, commandID(operation)) {
				continue
			}
			payload, err := client.DecodeResponse(operation.ExampleResponse)
			if err != nil {
				return fmt.Errorf("waiter %s requires an object response example: %w", id, err)
			}
			if err := waiter.ValidateExample(waiter.Spec{Path: spec.Path, Selector: spec.Selector}, payload); err != nil {
				return fmt.Errorf("waiter %s path %s is not supported by the example for %s: %w", id, spec.Path, operation.ID, err)
			}
		}
	}
	return plugin.ValidateWaiterBindings(plugin.Bundle{Commands: buildCommands(catalog), APIs: buildAPIs(catalog), Waiters: buildWaiters(catalog)})
}

// waiterCoreRequirement preserves existing constraints while requiring the
// first core release that interprets explicit bindings and multiple outcomes.
func waiterCoreRequirement(constraint string) string {
	for part := range strings.FieldsSeq(constraint) {
		if strings.HasPrefix(part, ">=") && version.CompareSemanticVersions(strings.TrimPrefix(part, ">="), "0.5.0") >= 0 {
			return constraint
		}
	}
	return strings.TrimSpace(constraint + " >=0.5.0")
}
