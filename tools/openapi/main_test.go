/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHarvestRequiresInput(t *testing.T) {
	var stdout bytes.Buffer
	err := run([]string{"harvest", "ecs"}, t.TempDir(), &stdout)
	if err == nil {
		t.Fatal("run returned nil error")
	}
	if err.Error() != "harvest requires --input" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRunHarvestAndDiff(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	if err := os.WriteFile(input, []byte(toolCatalogFixtureJSON), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	if err := run([]string{"harvest", "ecs", "--input", input}, root, &stdout); err != nil {
		t.Fatalf("harvest returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "wrote openapi-catalogs/ecs/source.json") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	stdout.Reset()
	if err := run([]string{"diff", "ecs"}, root, &stdout); err != nil {
		t.Fatalf("diff returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "wrote openapi-catalogs/ecs/changes.md") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

// TestRunNormalizeLabelsRepairsTrustedSourceEvidence verifies the maintenance
// command updates source labels without generating or promoting a plugin.
func TestRunNormalizeLabelsRepairsTrustedSourceEvidence(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.json")
	content := strings.ReplaceAll(toolCatalogFixtureJSON, `"label_en": "Instance ID"`, `"label_en": "Cmk UUID"`)
	content = strings.ReplaceAll(content, `"label_zh": "实例 ID"`, `"label_zh": "Instance ID"`)
	if err := os.WriteFile(input, []byte(content), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	if err := run([]string{"harvest", "ecs", "--input", input}, root, &stdout); err != nil {
		t.Fatalf("harvest returned error: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"normalize-labels", "ecs"}, root, &stdout); err != nil {
		t.Fatalf("normalize-labels returned error: %v", err)
	}
	if stdout.String() != "normalized 2 labels in openapi-catalogs/ecs/source.json\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	source, err := os.ReadFile(filepath.Join(root, "openapi-catalogs", "ecs", "source.json"))
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !bytes.Contains(source, []byte(`"label_en": "CMK UUID"`)) || !bytes.Contains(source, []byte(`"label_zh": "云主机 ID"`)) {
		t.Fatalf("normalized source = %s", source)
	}
}

const toolCatalogFixtureJSON = `{
  "schema_version": 1,
  "product": {
    "plugin_name": "ecs",
    "api_product": "ecs",
    "ctyun_product_id": 25,
    "source_revision": "81",
    "display_name": {
      "en-US": "Elastic Cloud Server",
      "en-GB": "Elastic Cloud Server",
      "zh-CN": "弹性云主机"
    },
    "endpoint_url": "https://ctecs-global.ctapi.ctyun.cn",
    "source_url": "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25",
    "api_scope": {
      "include_uri_prefixes": [
        "/v4/ecs/"
      ],
      "notes": "All official ECS APIs whose URI starts with /v4/ecs/."
    }
  },
  "operations": [
    {
      "id": "v4.ecs.instance.list",
      "api_id": "8309",
      "title": "查询云主机列表",
      "description": {
        "en-US": "List ECS instances",
        "en-GB": "List ECS instances",
        "zh-CN": "列出云主机"
      },
      "category": "instance",
      "method": "POST",
      "path": "/v4/ecs/list-instances",
      "content_type": "application/json",
      "docs_url": "https://eop.ctyun.cn/ebp/ctapiDocument/search?sid=25&api=8309&data=87",
      "retryable": true,
      "parameters": [
        {
          "name": "regionID",
          "location": "body",
          "required": true,
          "type": "string",
          "profile": "region"
        }
      ],
      "response": {
        "success_code": "800",
        "result_path": "returnObj",
        "row_path": "returnObj.results",
        "columns": [
          {
            "key": "instance_id",
            "path": "instanceID",
            "label_en": "Instance ID",
            "label_zh": "实例 ID"
          }
        ]
      },
      "example_response": {
        "statusCode": 800,
        "returnObj": {
          "results": [
            {
              "instanceID": "ins-demo-1"
            }
          ]
        }
      }
    }
  ]
}`
