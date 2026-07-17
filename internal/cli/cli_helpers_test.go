/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/testarchive"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

func testCompatibleCoreConstraint() string {
	return ">=" + testCoreVersion() + " <1.0.0"
}

func testCoreVersion() string {
	return version.Version
}

func writeArgumentBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create argument bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ims",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "generated",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ims", "ctyun_product_id": 23, "source_revision": "89"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ims.image.show",
      "path": ["ims", "image", "show", "{image_id}"],
      "operation": "v4.ims.image.show",
      "table": "ims.image.show"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ims.image.show": {
      "method": "POST",
      "path": "/v4/ims/image/show",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "imageID": "$arg.image_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ims.image.show": {
      "row_path": "returnObj.images",
      "columns": [
        {"key": "image_id", "path": "imageID", "labels": {"zh-CN": "镜像ID", "en-US": "Image ID", "en-GB": "Image ID"}},
        {"key": "name", "path": "name", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
}`)
}

func writeFlagBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create flag bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "parameters": [
        {"name": "name", "flag": "name", "target": "displayName", "required": true, "description": "Filter by instance name"}
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "POST",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "displayName": "$param.name"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}},
        {"key": "name", "path": "displayName", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
}`)
}

func writeQueryHeaderBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create query/header bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "parameters": [
        {"name": "page", "flag": "page", "target": "pageNo", "required": true, "description": "Page number"}
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "GET",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "query": {"regionID": "$profile.region", "pageNo": "$param.page"},
      "headers": {"x-ctyun-resource": "ecs"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
}

func writeValidationBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)

	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create validation bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "fixture_response": "fixtures/list.json",
      "parameters": [
        {"name": "status", "flag": "status", "target": "status", "allowed_values": ["running", "stopped"], "description": "Status"},
        {"name": "name", "flag": "name", "target": "displayName", "pattern": "^[A-Za-z0-9-]+$", "description": "Instance name"}
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "POST",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "body": {"status": "$param.status", "displayName": "$param.name"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "list.json"), `{"returnObj":{"instances":[]}}`)
}

func writeIMSBundleWithoutFixture(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create ims bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ims",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ims", "ctyun_product_id": 23, "source_revision": "89"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ims.image.list",
      "path": ["ims", "image", "list"],
      "operation": "v4.ims.image.list",
      "table": "ims.image.list",
      "parameters": [
        {"name": "region", "flag": "region", "target": "regionID", "required": false, "description": "Region ID"}
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ims.image.list": {
      "method": "POST",
      "path": "/v4/ims/image/list",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ims.image.list": {
      "row_path": "returnObj.images",
      "columns": [
        {"key": "image_id", "path": "imageID", "labels": {"zh-CN": "镜像ID", "en-US": "Image ID", "en-GB": "Image ID"}},
        {"key": "name", "path": "name", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
	}`)
}

func writeDeprecatedWarningBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create deprecated warning bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "demo",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "generated",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "demo", "ctyun_product_id": 25, "endpoint_url": "https://ctapi.example.test"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "demo.list",
      "path": ["demo", "list"],
      "operation": "v4.demo.list",
      "table": "demo.list",
      "fixture_response": "fixtures/list.json",
      "deprecation": {
        "status": "deprecated",
        "notice": "old command",
        "replacement": {"kind": "command", "label": "ctyun demo new-list"}
      },
      "parameters": [
        {
          "name": "page",
          "flag": "page",
          "target": "page",
          "required": false,
          "description": "Page number",
          "deprecation": {
            "status": "deprecated",
            "notice": "old page field",
            "replacement": {"kind": "parameter", "label": "--page-no"}
          }
        }
      ]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.demo.list": {
      "method": "GET",
      "path": "/v4/demo/list",
      "content_type": "application/json",
      "query": {"regionID": "$profile.region", "page": "$param.page"},
      "retryable": true,
      "deprecation": {
        "status": "deprecated",
        "notice": "old API",
        "replacement": {"kind": "api", "label": "newer CTyun API"}
      }
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "demo.list": {
      "row_path": "returnObj.items",
      "columns": [
        {"key": "id", "path": "id", "labels": {"zh-CN": "ID", "en-US": "ID", "en-GB": "ID"}},
        {
          "key": "old_size",
          "path": "oldSize",
          "labels": {"zh-CN": "旧大小", "en-US": "Old Size", "en-GB": "Old Size"},
          "deprecation": {
            "status": "deprecated",
            "notice": "old response field",
            "replacement": {"kind": "field", "label": "remainingSize"}
          }
        }
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "list.json"), `{"returnObj":{"items":[{"id":"one","oldSize":"1"}]}}`)
}

func envCredentialsForDeprecatedWarningTest(key string) string {
	switch key {
	case "CTYUN_AK":
		return "env-ak"
	case "CTYUN_SK":
		return "env-sk"
	default:
		return ""
	}
}

func deprecatedWarningConfigForTest(disableWarning bool) []byte {
	warningSetting := ""
	if disableWarning {
		warningSetting = `"warn_deprecated": false,`
	}
	return []byte(`{
  ` + warningSetting + `
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "81f7728662dd11ec810800155d307d5b",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`)
}

func writeArgumentRegionOptionBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create region bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "region",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "curated",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "region", "ctyun_product_id": 25, "source_revision": "81", "endpoint_url": "https://ctapi.example.test"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "region.show",
      "path": ["region", "show", "{region_id}"],
      "operation": "v4.region.show",
      "table": "region.show"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.region.show": {
      "method": "GET",
      "path": "/v4/region/show",
      "content_type": "application/json",
      "query": {"regionID": "$arg.region_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "region.show": {
      "row_path": "returnObj",
      "columns": [
        {"key": "region_id", "path": "regionID", "labels": {"zh-CN": "资源池ID", "en-US": "Region ID", "en-GB": "Region ID"}}
      ]
    }
  }
}`)
}

func writeVersionedBundle(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "`+name+`",
  "version": "`+version+`",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "`+name+`", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands": []}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{"tables": {}}`)
}

func writeDangerBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create danger fixture dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.delete",
      "path": ["ecs", "instance", "delete", "{instance_id}"],
      "operation": "v4.ecs.instance.delete",
      "table": "ecs.instance.delete",
      "fixture_response": "fixtures/delete.json",
      "dangerous": {"confirm": "yes", "message": "delete instance"}
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.delete": {
      "method": "POST",
      "path": "/v4/ecs/delete-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "instanceID": "$arg.instance_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.delete": {
      "row_path": "returnObj.jobs",
      "columns": [
        {"key": "job_id", "path": "jobID", "labels": {"zh-CN": "任务ID", "en-US": "Job ID", "en-GB": "Job ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "delete.json"), `{
  "returnObj": {
    "jobs": [{"jobID": "job-demo-1"}]
  }
}`)
}

func writeWaitBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)
	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create wait fixture dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.show",
      "path": ["ecs", "instance", "show", "{instance_id}"],
      "operation": "v4.ecs.instance.show",
      "table": "ecs.instance.show",
      "fixture_response": "fixtures/show.json"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.show": {
      "method": "POST",
      "path": "/v4/ecs/show-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "instanceID": "$arg.instance_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.show": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{
  "waiters": {
    "ecs.instance.running": {"path": "returnObj.status", "success": "running", "failure": "error"}
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "show.json"), `{
  "returnObj": {
    "status": "running",
    "instances": [{"instanceID": "ins-demo-1"}]
  }
}`)
}

func writePollingWaitBundle(t *testing.T, dir string) {
	t.Helper()
	disableDevelopmentBundledPluginsForTest(t)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create polling wait bundle dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.show",
      "path": ["ecs", "instance", "show", "{instance_id}"],
      "operation": "v4.ecs.instance.show",
      "table": "ecs.instance.show"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.show": {
      "method": "POST",
      "path": "/v4/ecs/show-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region", "instanceID": "$arg.instance_id"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.show": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{
  "waiters": {
    "ecs.instance.running": {
      "path": "returnObj.status",
      "success": "running",
      "failure": "error",
      "max_attempts": 3,
      "interval_seconds": 0
    }
  }
}`)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func writeVPCBundle(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(dir, "fixtures"), 0o755); err != nil {
		t.Fatalf("create vpc fixture dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "vpc",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "vpc", "ctyun_product_id": 18, "source_revision": "94"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "vpc.subnet.list",
      "path": ["vpc", "subnet", "list"],
      "operation": "v4.vpc.subnet.list",
      "table": "vpc.subnet.list",
      "fixture_response": "fixtures/subnet-list.json"
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.vpc.subnet.list": {
      "method": "POST",
      "path": "/v4/vpc/list-subnet",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "vpc.subnet.list": {
      "row_path": "returnObj.subnets",
      "columns": [
        {"key": "subnet_id", "path": "subnetID", "labels": {"zh-CN": "子网ID", "en-US": "Subnet ID", "en-GB": "Subnet ID"}},
        {"key": "name", "path": "name", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "fixtures", "subnet-list.json"), `{
  "statusCode": 800,
  "message": "success",
  "returnObj": {
    "subnets": [
      {"subnetID": "subnet-demo-1", "name": "app-subnet"}
    ]
  }
}`)
}

func testBundleDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "generated",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands": []}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{"tables": {}}`)
	if err := os.Mkdir(filepath.Join(dir, "i18n"), 0o755); err != nil {
		t.Fatalf("mkdir i18n: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "i18n", "en-US.json"), `{"name":"Elastic Cloud Server"}`)
	mustWrite(t, filepath.Join(dir, "i18n", "zh-CN.json"), `{"name":"弹性云主机"}`)
	return dir
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func disableDevelopmentBundledPluginsForTest(t *testing.T) {
	t.Helper()

	original := developmentBundledPluginsEnabled
	developmentBundledPluginsEnabled = func() bool { return false }
	t.Cleanup(func() { developmentBundledPluginsEnabled = original })
}

func signedRegistryIndex(t *testing.T, index []byte) (string, string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate registry signing key: %v", err)
	}
	signature := ed25519.Sign(privateKey, index)
	return base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(signature)
}

func hostedPluginArtifact(t *testing.T, name, version string) (string, []byte, string) {
	t.Helper()
	bundleDir := filepath.Join(t.TempDir(), name+"-"+version)
	writeVersionedBundle(t, bundleDir, name, version)
	artifactName := name + "-" + version + ".tar.gz"
	archivePath := filepath.Join(t.TempDir(), artifactName)
	writeTarGz(t, archivePath, bundleDir)
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read hosted plugin artifact: %v", err)
	}
	sum := sha256.Sum256(data)
	return artifactName, data, hex.EncodeToString(sum[:])
}

func hostedPluginRegistry(t *testing.T, index []byte, artifacts map[string][]byte) (string, http.RoundTripper) {
	t.Helper()
	publicKey, signature := signedRegistryIndex(t, index)
	return publicKey, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch filepath.Base(req.URL.Path) {
		case "index.json":
			return &http.Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK), Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(index))}, nil
		case "index.sig":
			return &http.Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(signature))}, nil
		default:
			if body, ok := artifacts[filepath.Base(req.URL.Path)]; ok {
				return &http.Response{StatusCode: http.StatusOK, Status: http.StatusText(http.StatusOK), Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
			}
			return &http.Response{StatusCode: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound), Header: make(http.Header), Body: io.NopCloser(strings.NewReader("not found"))}, nil
		}
	})
}

func hostedPluginEnv(publicKey string) func(string) string {
	return func(key string) string {
		if key == "CTYUN_RELEASE_PUBLIC_KEY" {
			return publicKey
		}
		return ""
	}
}

func writeTarGz(t *testing.T, archivePath, srcDir string) {
	t.Helper()
	testarchive.WriteTarGzFromDir(t, archivePath, srcDir)
}

// reportRenderer renders one config or diagnostic report with shared options.
type reportRenderer func(io.Writer, globalOptions) error

// assertReportRendererErrors verifies the shared table-control, format, render,
// and writer failure contract for report renderers.
func assertReportRendererErrors(t *testing.T, render reportRenderer) {
	t.Helper()
	for _, test := range []struct {
		name string
		opts globalOptions
	}{
		{name: "filter", opts: globalOptions{Output: "table", Language: "en-US", Filter: "missing=value"}},
		{name: "sort", opts: globalOptions{Output: "table", Language: "en-US", Sort: "missing"}},
		{name: "output", opts: globalOptions{Output: "xml", Language: "en-US"}},
	} {
		if err := render(io.Discard, test.opts); err == nil {
			t.Fatalf("renderer accepted invalid %s", test.name)
		}
	}

	originalJSON := renderOutputJSON
	originalTable := renderOutputTable
	t.Cleanup(func() { renderOutputJSON = originalJSON; renderOutputTable = originalTable })
	want := errors.New("render")
	renderOutputJSON = func(any) (string, error) { return "", want }
	if err := render(io.Discard, globalOptions{Output: "json", Language: "en-US"}); !errors.Is(err, want) {
		t.Fatalf("JSON render error = %v", err)
	}
	renderOutputJSON = originalJSON
	if err := render(failingWriter{}, globalOptions{Output: "json", Language: "en-US"}); err == nil {
		t.Fatal("JSON renderer ignored stdout failure")
	}
	renderOutputTable = func([]map[string]string, []output.Column, output.TableOptions) (string, error) { return "", want }
	if err := render(io.Discard, globalOptions{Output: "table", Language: "en-US"}); !errors.Is(err, want) {
		t.Fatalf("table render error = %v", err)
	}
	renderOutputTable = originalTable
	if err := render(failingWriter{}, globalOptions{Output: "table", Language: "en-US"}); err == nil {
		t.Fatal("table renderer ignored stdout failure")
	}
}
