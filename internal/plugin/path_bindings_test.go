/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"path/filepath"
	"testing"
)

// TestLoadBundleRejectsUnboundHTTPPathArguments keeps malformed templates and
// undeclared argument references out of executable plugin bundles.
func TestLoadBundleRejectsUnboundHTTPPathArguments(t *testing.T) {
	for _, path := range []string{"/v4/{missing}", "/v4/{instance_id", "/v4/prefix{instance_id}"} {
		t.Run(path, func(t *testing.T) {
			dir := writeBundle(t, "ecs", ">=0.4.0 <1.0.0")
			mustWrite(t, filepath.Join(dir, "apis.json"), `{"operations":{"show":{"method":"GET","path":"`+path+`"}}}`)
			mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[{"id":"show","path":["ecs","show","{instance_id}"],"operation":"show","table":"ecs.instance.list"}]}`)
			// The fixture's table identity is independent of the path contract.
			mustWrite(t, filepath.Join(dir, "tables.json"), `{"tables":{"ecs.instance.list":{"row_path":"returnObj","columns":[{"key":"id","path":"id","labels":{"en-US":"ID","en-GB":"ID","zh-CN":"ID"}}]}}}`)
			if _, err := LoadBundle(dir, "0.5.0"); err == nil {
				t.Fatal("accepted unresolved API path")
			}
		})
	}
	if err := validateOperationPathBindings(Command{Path: []string{"cf", "show", "{function_name}"}}, Operation{Path: "/openapi/v1/functions/{function_name}"}); err != nil {
		t.Fatal(err)
	}
}
