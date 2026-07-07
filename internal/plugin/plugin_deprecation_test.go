/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBundleAcceptsDeprecationMetadata(t *testing.T) {
	root := deprecationBundleDir(t)
	mustWrite(t, filepath.Join(root, "commands.json"), `{
  "commands": [{
    "id": "demo.show",
    "path": ["demo", "show"],
    "operation": "demo.show",
    "table": "demo.show",
    "fixture_response": "fixtures/show.json",
    "docs_url": "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25&api=1",
    "deprecation": {
      "status": "deprecated",
      "notice": "old API",
      "replacement": {"kind": "generic", "label": "newer CTyun OpenAPI"}
    },
    "parameters": [{
      "name": "page",
      "flag": "page",
      "target": "page",
      "required": false,
      "description": "Page",
      "deprecation": {"status": "deprecated", "notice": "use pageNo"}
    }],
    "examples": ["ctyun demo show"]
  }]
}`)
	mustWrite(t, filepath.Join(root, "apis.json"), `{
  "operations": {
    "demo.show": {
      "method": "GET",
      "path": "/v4/demo/show",
      "query": {"page": "$param.page"},
      "retryable": true,
      "deprecation": {"status": "deprecated", "notice": "old API"}
    }
  }
}`)
	mustWrite(t, filepath.Join(root, "tables.json"), `{
  "tables": {
    "demo.show": {
      "row_path": "returnObj.items",
      "columns": [{
        "key": "old_size",
        "path": "oldSize",
        "labels": {"en-US": "Old Size", "en-GB": "Old Size", "zh-CN": "旧大小"},
        "default": true,
        "deprecation": {"status": "deprecated", "notice": "use remainingSize"}
      }]
    }
  }
}`)
	mustWrite(t, filepath.Join(root, "fixtures", "show.json"), `{"statusCode":800,"returnObj":{"items":[{"oldSize":"1"}]}}`)

	bundle, err := LoadBundle(root, "9.9.9")
	if err != nil {
		t.Fatalf("LoadBundle returned error: %v", err)
	}
	if bundle.Commands.Commands[0].Deprecation.Status != "deprecated" {
		t.Fatalf("command deprecation = %#v", bundle.Commands.Commands[0].Deprecation)
	}
	if bundle.Commands.Commands[0].Parameters[0].Deprecation.Notice != "use pageNo" {
		t.Fatalf("parameter deprecation = %#v", bundle.Commands.Commands[0].Parameters[0].Deprecation)
	}
	if bundle.APIs.Operations["demo.show"].Deprecation.Notice != "old API" {
		t.Fatalf("operation deprecation = %#v", bundle.APIs.Operations["demo.show"].Deprecation)
	}
	if bundle.Tables.Tables["demo.show"].Columns[0].Deprecation.Notice != "use remainingSize" {
		t.Fatalf("column deprecation = %#v", bundle.Tables.Tables["demo.show"].Columns[0].Deprecation)
	}
}

func TestLoadBundleRejectsUnsupportedDeprecationStatus(t *testing.T) {
	root := deprecationBundleDir(t)
	mustWrite(t, filepath.Join(root, "commands.json"), `{
  "commands": [{
    "id": "demo.show",
    "path": ["demo", "show"],
    "operation": "demo.show",
    "table": "demo.show",
    "fixture_response": "fixtures/show.json",
    "deprecation": {"status": "removed"},
    "examples": ["ctyun demo show"]
  }]
}`)
	mustWrite(t, filepath.Join(root, "fixtures", "show.json"), `{"statusCode":800,"returnObj":{"items":[]}}`)

	_, err := LoadBundle(root, "9.9.9")
	if err == nil {
		t.Fatal("LoadBundle returned nil error")
	}
	assertDiagnosticKey(t, err, "error.deprecation_status")
}

func TestLoadBundleRejectsUnsupportedDeprecationReplacementKind(t *testing.T) {
	cases := []struct {
		name  string
		write func(t *testing.T, root string)
	}{
		{
			name: "parameter",
			write: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "commands.json"), `{
  "commands": [{
    "id": "demo.show",
    "path": ["demo", "show"],
    "operation": "demo.show",
    "table": "demo.show",
    "fixture_response": "fixtures/show.json",
    "parameters": [{
      "name": "page",
      "flag": "page",
      "target": "page",
      "deprecation": {"status": "deprecated", "replacement": {"kind": "service", "label": "pageNo"}}
    }]
  }]
}`)
			},
		},
		{
			name: "operation",
			write: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "apis.json"), `{
  "operations": {
    "demo.show": {
      "method": "GET",
      "path": "/v4/demo/show",
      "deprecation": {"status": "deprecated", "replacement": {"kind": "service", "label": "v2"}}
    }
  }
}`)
			},
		},
		{
			name: "table column",
			write: func(t *testing.T, root string) {
				mustWrite(t, filepath.Join(root, "tables.json"), `{
  "tables": {
    "demo.show": {
      "row_path": "returnObj.items",
      "columns": [{
        "key": "old_size",
        "path": "oldSize",
        "labels": {"en-US": "Old Size", "en-GB": "Old Size", "zh-CN": "旧大小"},
        "deprecation": {"status": "deprecated", "replacement": {"kind": "service", "label": "newSize"}}
      }]
    }
  }
}`)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := deprecationBundleDir(t)
			tc.write(t, root)
			_, err := LoadBundle(root, "9.9.9")
			if err == nil {
				t.Fatal("LoadBundle returned nil error")
			}
			assertDiagnosticKey(t, err, "error.deprecation_replacement_kind")
		})
	}
}

func deprecationBundleDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "fixtures"), 0o755); err != nil {
		t.Fatalf("create fixtures: %v", err)
	}
	mustWrite(t, filepath.Join(root, "plugin.json"), `{
  "name": "demo",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "generated",
  "requires": {"ctyun": ">=0.0.0"},
  "api": {
    "product": "demo",
    "ctyun_product_id": 25,
    "endpoint_url": "https://ctecs-global.ctapi.ctyun.cn"
  }
}`)
	mustWrite(t, filepath.Join(root, "commands.json"), `{"commands":[]}`)
	mustWrite(t, filepath.Join(root, "apis.json"), `{"operations":{}}`)
	mustWrite(t, filepath.Join(root, "tables.json"), `{"tables":{}}`)
	return root
}

func assertDiagnosticKey(t *testing.T, err error, want string) {
	t.Helper()
	got, ok := err.(interface{ MessageKey() string })
	if !ok {
		t.Fatalf("error %T does not expose a diagnostic key: %v", err, err)
	}
	if got.MessageKey() != want {
		t.Fatalf("diagnostic key = %q, want %q", got.MessageKey(), want)
	}
}
