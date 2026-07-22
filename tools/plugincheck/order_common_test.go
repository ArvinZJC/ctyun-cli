/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// documentedPluginOperation pins one visible command to its upstream request
// and generated response table.
type documentedPluginOperation struct {
	command   string
	path      []string
	operation string
	method    string
	uri       string
	rowPath   string
}

// TestOrderAndCommonPluginsCoverDocumentedAPIs pins the approved command and
// HTTP surface selected from the official online ECS documentation.
func TestOrderAndCommonPluginsCoverDocumentedAPIs(t *testing.T) {
	cases := []struct {
		plugin     string
		version    string
		channel    string
		quality    string
		scope      []string
		operations []documentedPluginOperation
	}{
		{
			plugin:  "order",
			version: "0.1.1",
			channel: "stable",
			quality: "curated",
			scope:   []string{"/v4/order/", "/v4/new-order/", "/v4/renew-order/", "/v4/upgrade-order/"},
			operations: []documentedPluginOperation{
				{command: "order.price.new", path: []string{"order", "price", "new"}, operation: "v4.order.price.new", method: "POST", uri: "/v4/order/new-query-price", rowPath: "returnObj"},
				{command: "order.price.renew", path: []string{"order", "price", "renew"}, operation: "v4.order.price.renew", method: "POST", uri: "/v4/order/renew-query-price", rowPath: "returnObj"},
				{command: "order.price.upgrade", path: []string{"order", "price", "upgrade"}, operation: "v4.order.price.upgrade", method: "POST", uri: "/v4/order/upgrade-query-price", rowPath: "returnObj"},
				{command: "order.resource-uuid.list", path: []string{"order", "resource-uuid", "list", "{master_order_id}"}, operation: "v4.order.resource-uuid.list", method: "GET", uri: "/v4/order/query-uuid", rowPath: "returnObj"},
				{command: "order.new.price", path: []string{"order", "new", "price"}, operation: "v4.order.new.price", method: "POST", uri: "/v4/new-order/query-price", rowPath: "returnObj"},
				{command: "order.renew.price", path: []string{"order", "renew", "price"}, operation: "v4.order.renew.price", method: "POST", uri: "/v4/renew-order/query-price", rowPath: "returnObj"},
				{command: "order.upgrade.price", path: []string{"order", "upgrade", "price"}, operation: "v4.order.upgrade.price", method: "POST", uri: "/v4/upgrade-order/query-price", rowPath: "returnObj"},
			},
		},
		{
			plugin:  "common",
			version: "0.1.0-beta.2",
			channel: "beta",
			quality: "generated",
			scope:   []string{"/v4/common/"},
			operations: []documentedPluginOperation{
				{command: "common.ecs-flavor.list", path: []string{"common", "ecs-flavor", "list"}, operation: "v4.common.ecs-flavor.list", method: "GET", uri: "/v4/common/get-ecs-flavors", rowPath: "returnObj.results"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.plugin, func(t *testing.T) {
			bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", tc.plugin)), version.Version)
			if err != nil {
				t.Fatalf("load %s plugin: %v", tc.plugin, err)
			}
			if bundle.Manifest.Version != tc.version || bundle.Manifest.Channel != tc.channel || bundle.Manifest.Quality != tc.quality {
				t.Fatalf("manifest release = %s/%s/%s, want %s/%s/%s", bundle.Manifest.Version, bundle.Manifest.Channel, bundle.Manifest.Quality, tc.version, tc.channel, tc.quality)
			}
			if !slices.Equal(bundle.Manifest.API.Scope.IncludeURIPrefixes, tc.scope) {
				t.Fatalf("API scope = %#v, want %#v", bundle.Manifest.API.Scope.IncludeURIPrefixes, tc.scope)
			}
			if len(bundle.Commands.Commands) != len(tc.operations) || len(bundle.APIs.Operations) != len(tc.operations) {
				t.Fatalf("commands/operations = %d/%d, want %d/%d", len(bundle.Commands.Commands), len(bundle.APIs.Operations), len(tc.operations), len(tc.operations))
			}
			commands := make(map[string]plugin.Command, len(bundle.Commands.Commands))
			for _, command := range bundle.Commands.Commands {
				commands[command.ID] = command
			}
			for _, expected := range tc.operations {
				command, ok := commands[expected.command]
				if !ok {
					t.Fatalf("missing command %s", expected.command)
				}
				if !slices.Equal(command.Path, expected.path) || command.Operation != expected.operation {
					t.Fatalf("command %s path/operation = %#v/%s, want %#v/%s", expected.command, command.Path, command.Operation, expected.path, expected.operation)
				}
				operation, ok := bundle.APIs.Operations[expected.operation]
				if !ok {
					t.Fatalf("missing operation %s", expected.operation)
				}
				if operation.Method != expected.method || operation.Path != expected.uri || !operation.Retryable {
					t.Fatalf("operation %s = %s %s retryable=%t, want %s %s retryable=true", expected.operation, operation.Method, operation.Path, operation.Retryable, expected.method, expected.uri)
				}
				if len(operation.AcceptedStatuses) != 0 {
					t.Fatalf("operation %s accepted statuses = %#v, want none", expected.operation, operation.AcceptedStatuses)
				}
				if got := bundle.Tables.Tables[command.Table].RowPath; got != expected.rowPath {
					t.Fatalf("command %s row path = %q, want %q", expected.command, got, expected.rowPath)
				}
			}
			assertOrderAndCommonBindings(t, bundle)
		})
	}
}

// assertOrderAndCommonBindings verifies the identifiers that distinguish the
// two order quote families and the product-neutral common query contract.
func assertOrderAndCommonBindings(t *testing.T, bundle plugin.Bundle) {
	t.Helper()
	switch bundle.Manifest.Name {
	case "order":
		for _, operationID := range []string{"v4.order.price.renew", "v4.order.price.upgrade"} {
			if got := bundle.APIs.Operations[operationID].Body["resourceUUID"]; got != "$param.resource_uuid" {
				t.Fatalf("operation %s resourceUUID binding = %q, want $param.resource_uuid", operationID, got)
			}
		}
		for _, operationID := range []string{"v4.order.renew.price", "v4.order.upgrade.price"} {
			if got := bundle.APIs.Operations[operationID].Body["resourceID"]; got != "$param.resource_id" {
				t.Fatalf("operation %s resourceID binding = %q, want $param.resource_id", operationID, got)
			}
		}
		if got := bundle.APIs.Operations["v4.order.resource-uuid.list"].Query["masterOrderId"]; got != "$arg.master_order_id" {
			t.Fatalf("resource UUID masterOrderId binding = %q, want $arg.master_order_id", got)
		}
	case "common":
		operation := bundle.APIs.Operations["v4.common.ecs-flavor.list"]
		for field, want := range map[string]string{
			"regionID": "$profile.region",
			"azName":   "$param.az_name",
			"series":   "$param.series",
		} {
			if got := operation.Query[field]; got != want {
				t.Fatalf("common %s binding = %q, want %q", field, got, want)
			}
		}
	}
}
