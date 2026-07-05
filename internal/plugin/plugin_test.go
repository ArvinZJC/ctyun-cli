/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/testarchive"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

func testCompatibleCoreConstraint() string {
	return ">=" + testCoreVersion() + " <1.0.0"
}

func testCoreVersion() string {
	return coreversion.Version
}

func TestLoadBundleReadsMetadataCommandsAndTables(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())

	bundle, err := LoadBundle(dir, testCoreVersion())
	if err != nil {
		t.Fatalf("LoadBundle returned error: %v", err)
	}

	if bundle.Manifest.Name != "ecs" {
		t.Fatalf("bundle name = %q, want ecs", bundle.Manifest.Name)
	}
	if len(bundle.Commands.Commands) != 1 {
		t.Fatalf("loaded %d commands, want 1", len(bundle.Commands.Commands))
	}
	if len(bundle.Tables.Tables["ecs.instance.list"].Columns) != 3 {
		t.Fatalf("loaded table columns = %d, want 3", len(bundle.Tables.Tables["ecs.instance.list"].Columns))
	}
	if bundle.Commands.Commands[0].DocsURL != "https://eop.ctyun.cn/ecs/list" {
		t.Fatalf("DocsURL = %q", bundle.Commands.Commands[0].DocsURL)
	}
	if bundle.Commands.Commands[0].Dangerous.Confirm != "" {
		t.Fatalf("list command should not require confirmation")
	}
	if bundle.Commands.Commands[0].Parameters[0].Flag != "name" {
		t.Fatalf("parameter flag = %q, want name", bundle.Commands.Commands[0].Parameters[0].Flag)
	}
	if bundle.I18N["zh-CN"]["command.ecs.instance.list.description"] != "列出云主机" {
		t.Fatalf("i18n command description was not loaded: %#v", bundle.I18N)
	}
	if bundle.Waiters.Waiters["ecs.instance.running"].Success != "running" {
		t.Fatalf("waiter metadata was not loaded")
	}
}

func TestLoadBundleRejectsWaiterTimeoutSeconds(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{
  "waiters": {
    "ecs.instance.running": {
      "path": "returnObj.status",
      "success": "running",
      "failure": "error",
      "max_attempts": 3,
      "interval_seconds": 1,
      "timeout_seconds": 30
    }
  }
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for waiter timeout_seconds")
	}
	requireDiagnosticKey(t, err, "error.waiter_unsupported_timeout_seconds")
}

func TestCloseWithError(t *testing.T) {
	closeErr := errors.New("close failed")
	if err := closeWithError(nil, func() error { return nil }); err != nil {
		t.Fatalf("closeWithError nil close error = %v, want nil", err)
	}
	if err := closeWithError(nil, func() error { return closeErr }); !errors.Is(err, closeErr) {
		t.Fatalf("closeWithError close-only error = %v, want %v", err, closeErr)
	}

	primaryErr := errors.New("copy failed")
	err := closeWithError(primaryErr, func() error { return closeErr })
	if !errors.Is(err, primaryErr) || !errors.Is(err, closeErr) {
		t.Fatalf("closeWithError joined error = %v, want primary and close errors", err)
	}
}

func TestLoadBundleRejectsInvalidParameterMetadata(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "parameters": [
        {"name": "name", "flag": "name", "required": true}
      ]
    }
  ]
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for parameter missing target")
	}
	requireDiagnosticKey(t, err, "error.command_parameter_missing_target")
}

func TestLoadBundleRejectsInvalidParameterPattern(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "parameters": [
        {"name": "name", "flag": "name", "target": "displayName", "pattern": "["}
      ]
    }
  ]
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for invalid parameter pattern")
	}
	requireDiagnosticKey(t, err, "error.command_parameter_invalid_pattern")
}

func TestLoadBundleRejectsMissingOperationEvenWithFixture(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.missing",
      "table": "ecs.instance.list",
      "fixture_response": "fixtures/list.json"
    }
  ]
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for fixture command with missing operation")
	}
	requireDiagnosticKey(t, err, "error.command_missing_operation_ref")
}

func TestLoadBundleRejectsUnsafeFixtureResponsePath(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "fixture_response": "../secret.json"
    }
  ]
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for unsafe fixture response path")
	}
	requireDiagnosticKey(t, err, "error.command_invalid_fixture_response")
}

func TestLoadBundleRejectsInvalidManifestMetadata(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "ecs",
  "version": "0.1.0",
  "channel": "nightly",
  "quality": "raw",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for invalid manifest metadata")
	}
	requireDiagnosticKey(t, err, "error.plugin_unsupported_channel")
}

func TestLoadBundleRejectsUnsafePluginName(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "../ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for unsafe plugin name")
	}
	requireDiagnosticKey(t, err, "error.plugin_name")
}

func TestLoadBundleRejectsDuplicateCommandPaths(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list"
    },
    {
      "id": "ecs.instance.search",
      "path": ["ecs", "instance", "list"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list"
    }
  ]
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for duplicate command path")
	}
	requireDiagnosticKey(t, err, "error.duplicate_command_path")
}

func TestLoadBundleRejectsUnsafeCommandPathSegment(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list;rm"],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list"
    }
  ]
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for unsafe command path segment")
	}
	requireDiagnosticKey(t, err, "error.command_invalid_path_segment")
}

func TestLoadBundleRejectsInvalidOperationMetadata(t *testing.T) {
	tests := []struct {
		name    string
		apis    string
		wantErr string
	}{
		{
			name: "unsupported method",
			apis: `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "TRACE",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region"}
    }
  }
}`,
			wantErr: "error.operation_unsupported_method",
		},
		{
			name: "scheme-relative path",
			apis: `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "POST",
      "path": "//evil.example/v4/ecs/list-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region"}
    }
  }
}`,
			wantErr: "error.operation_invalid_path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
			mustWrite(t, filepath.Join(dir, "apis.json"), tt.apis)

			_, err := LoadBundle(dir, testCoreVersion())
			if err == nil {
				t.Fatal("LoadBundle returned nil error for invalid operation metadata")
			}
			requireDiagnosticKey(t, err, tt.wantErr)
		})
	}
}

func TestLoadBundleRejectsIncompleteTableLabels(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID"}}
      ]
    }
  }
}`)

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for incomplete table labels")
	}
	requireDiagnosticKey(t, err, "error.table_column_missing_label")
}

func TestFindCommandMatchesCanonicalPathOnly(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	bundle, err := LoadBundle(dir, testCoreVersion())
	if err != nil {
		t.Fatalf("LoadBundle returned error: %v", err)
	}

	command, ok := FindCommand(bundle, []string{"ecs", "instance", "list"})
	if !ok {
		t.Fatal("FindCommand did not match canonical path")
	}
	if command.ID != "ecs.instance.list" {
		t.Fatalf("command id = %q", command.ID)
	}
	if _, ok := FindCommand(bundle, []string{"ecs", "server", "ls"}); ok {
		t.Fatal("FindCommand matched unsupported alias path")
	}
	if missing, ok := matchMissingPathArgs([]string{"ecs", "instance", "show", "{instance_id}"}, []string{"ecs", "instance", "show", "extra"}); ok || missing != nil {
		t.Fatalf("matchMissingPathArgs matched complete path: %v, %v", missing, ok)
	}
}

func TestLoadBundleIgnoresCommandAliasesField(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "aliases": [["ecs", "server", "ls"]],
      "operation": "v4.ecs.instance.list",
      "table": "ecs.instance.list",
      "parameters": [
        {"name": "name", "flag": "name", "target": "displayName", "required": false, "description": "Filter by instance name"}
      ],
      "docs_url": "https://eop.ctyun.cn/ecs/list",
      "examples": ["ctyun ecs instance list"],
      "dangerous": {"confirm": ""}
    }
  ]
}`)

	bundle, err := LoadBundle(dir, testCoreVersion())
	if err != nil {
		t.Fatalf("LoadBundle returned error for unused aliases field: %v", err)
	}
	if _, ok := FindCommand(bundle, []string{"ecs", "server", "ls"}); ok {
		t.Fatal("FindCommand matched ignored aliases field")
	}
}

func TestLoadBundleRejectsIncompatibleCoreVersion(t *testing.T) {
	dir := writeBundle(t, "ecs", ">=99.0.0 <100.0.0")

	_, err := LoadBundle(dir, testCoreVersion())
	if err == nil {
		t.Fatal("LoadBundle returned nil error for incompatible version")
	}
}

func TestInstallLocalBundleCopiesDirectory(t *testing.T) {
	src := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	destRoot := t.TempDir()

	installed, err := InstallLocalBundle(src, destRoot)
	if err != nil {
		t.Fatalf("InstallLocalBundle returned error: %v", err)
	}

	if installed != filepath.Join(destRoot, "ecs") {
		t.Fatalf("installed path = %q, want %q", installed, filepath.Join(destRoot, "ecs"))
	}
	if _, err := os.Stat(filepath.Join(installed, "plugin.json")); err != nil {
		t.Fatalf("plugin.json was not copied: %v", err)
	}
}

func TestInstallLocalBundleExtractsTarGz(t *testing.T) {
	src := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	archivePath := filepath.Join(t.TempDir(), "ctyun-plugin-ecs-0.1.0.tar.gz")
	writeTarGz(t, archivePath, src)
	destRoot := t.TempDir()

	installed, err := InstallLocalBundle(archivePath, destRoot)
	if err != nil {
		t.Fatalf("InstallLocalBundle archive returned error: %v", err)
	}
	if installed != filepath.Join(destRoot, "ecs") {
		t.Fatalf("installed path = %q, want %q", installed, filepath.Join(destRoot, "ecs"))
	}
	if _, err := os.Stat(filepath.Join(installed, "plugin.json")); err != nil {
		t.Fatalf("plugin.json was not extracted: %v", err)
	}
}

func TestInstallLocalBundleExtractsTarGzWithTopLevelDirectory(t *testing.T) {
	src := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	archivePath := filepath.Join(t.TempDir(), "ctyun-plugin-ecs-0.1.0.tar.gz")
	writeTarGzWithPrefix(t, archivePath, src, "ctyun-plugin-ecs")
	destRoot := t.TempDir()

	installed, err := InstallLocalBundle(archivePath, destRoot)
	if err != nil {
		t.Fatalf("InstallLocalBundle wrapped archive returned error: %v", err)
	}
	if installed != filepath.Join(destRoot, "ecs") {
		t.Fatalf("installed path = %q, want %q", installed, filepath.Join(destRoot, "ecs"))
	}
	if _, err := os.Stat(filepath.Join(installed, "plugin.json")); err != nil {
		t.Fatalf("plugin.json was not extracted from wrapped archive: %v", err)
	}
}

func TestInstallLocalBundleRejectsTarGzSymlinkEntries(t *testing.T) {
	src := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	archivePath := filepath.Join(t.TempDir(), "ctyun-plugin-ecs-0.1.0.tar.gz")
	writeTarGzWithSymlink(t, archivePath, src)
	destRoot := t.TempDir()

	_, err := InstallLocalBundle(archivePath, destRoot)
	if err == nil {
		t.Fatal("InstallLocalBundle returned nil error for tar symlink entry")
	}
	requireDiagnosticKey(t, err, "error.unsupported_archive_entry")
	if _, statErr := os.Stat(filepath.Join(destRoot, "ecs")); !os.IsNotExist(statErr) {
		t.Fatalf("symlink archive was copied, stat err: %v", statErr)
	}
}

func TestInstallLocalBundleRejectsUnsafeManifestName(t *testing.T) {
	src := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(src, "plugin.json"), `{
  "name": "../ecs",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	destRoot := t.TempDir()

	_, err := InstallLocalBundle(src, destRoot)
	if err == nil {
		t.Fatal("InstallLocalBundle returned nil error for unsafe manifest name")
	}
	requireDiagnosticKey(t, err, "error.plugin_name")
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(destRoot), "ecs")); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe install wrote outside destination, stat err: %v", statErr)
	}
}

func TestInstallLocalBundleRejectsSymlinkEntries(t *testing.T) {
	src := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	if err := os.WriteFile(filepath.Join(src, "target.txt"), []byte("target"), 0o644); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Symlink("target.txt", filepath.Join(src, "linked.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	destRoot := t.TempDir()

	_, err := InstallLocalBundle(src, destRoot)
	if err == nil {
		t.Fatal("InstallLocalBundle returned nil error for symlink entry")
	}
	requireDiagnosticKey(t, err, "error.unsupported_bundle_entry")
	if _, statErr := os.Stat(filepath.Join(destRoot, "ecs")); !os.IsNotExist(statErr) {
		t.Fatalf("symlink bundle was copied, stat err: %v", statErr)
	}
}

func TestInstallVerifiedLocalBundleRejectsInvalidArchiveBeforeCopy(t *testing.T) {
	if _, err := InstallVerifiedLocalBundle(filepath.Join(t.TempDir(), "missing.tar.gz"), t.TempDir(), testCoreVersion()); err == nil {
		t.Fatal("InstallVerifiedLocalBundle returned nil error for missing archive")
	}

	src := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(src, "tables.json"), `{"tables": {}}`)
	archivePath := filepath.Join(t.TempDir(), "ctyun-plugin-ecs-0.1.0.tar.gz")
	writeTarGz(t, archivePath, src)
	destRoot := t.TempDir()

	_, err := InstallVerifiedLocalBundle(archivePath, destRoot, testCoreVersion())
	if err == nil {
		t.Fatal("InstallVerifiedLocalBundle returned nil error for invalid archive")
	}
	requireDiagnosticKey(t, err, "error.command_missing_table_ref")
	if _, statErr := os.Stat(filepath.Join(destRoot, "ecs")); !os.IsNotExist(statErr) {
		t.Fatalf("invalid archive was copied, stat err: %v", statErr)
	}
}

func TestInstallVerifiedLocalBundlePreservesExistingPluginOnCopyFailure(t *testing.T) {
	destRoot := t.TempDir()
	existing := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	if _, err := InstallVerifiedLocalBundle(existing, destRoot, testCoreVersion()); err != nil {
		t.Fatalf("install existing bundle: %v", err)
	}

	replacement := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(replacement, "plugin.json"), `{
  "name": "ecs",
  "version": "0.2.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+testCompatibleCoreConstraint()+`"},
  "api": {"product": "ecs", "ctyun_product_id": 25, "source_revision": "81"}
}`)
	if err := os.Symlink(filepath.Join(replacement, "missing-extra-file"), filepath.Join(replacement, "dangling-extra-file")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := InstallVerifiedLocalBundle(replacement, destRoot, testCoreVersion())
	if err == nil {
		t.Fatal("InstallVerifiedLocalBundle returned nil error for copy failure")
	}
	installed, loadErr := LoadBundle(filepath.Join(destRoot, "ecs"), testCoreVersion())
	if loadErr != nil {
		t.Fatalf("existing plugin was not loadable after failed replacement: %v", loadErr)
	}
	if installed.Manifest.Version != "0.1.0" {
		t.Fatalf("installed version = %q, want preserved 0.1.0", installed.Manifest.Version)
	}
}

func writeBundle(t *testing.T, name, requires string) string {
	t.Helper()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "plugin.json"), `{
  "name": "`+name+`",
  "version": "0.1.0",
  "channel": "stable",
  "quality": "reviewed",
  "requires": {"ctyun": "`+requires+`"},
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
        {"name": "name", "flag": "name", "target": "displayName", "required": false, "description": "Filter by instance name"}
      ],
      "docs_url": "https://eop.ctyun.cn/ecs/list",
      "examples": ["ctyun ecs instance list"],
      "dangerous": {"confirm": ""}
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{
  "operations": {
    "v4.ecs.instance.list": {
      "method": "POST",
      "path": "/v4/ecs/list-instance",
      "content_type": "application/json",
      "body": {"regionID": "$profile.region"}
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "tables.json"), `{
  "tables": {
    "ecs.instance.list": {
      "row_path": "returnObj.instances",
      "columns": [
        {"key": "instance_id", "path": "instanceID", "labels": {"zh-CN": "实例ID", "en-US": "Instance ID", "en-GB": "Instance ID"}},
        {"key": "name", "path": "displayName", "labels": {"zh-CN": "名称", "en-US": "Name", "en-GB": "Name"}},
        {"key": "status", "path": "status", "labels": {"zh-CN": "状态", "en-US": "Status", "en-GB": "Status"}}
      ]
    }
  }
}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{
  "waiters": {
    "ecs.instance.running": {
      "path": "returnObj.status",
      "success": "running",
      "failure": "error"
    }
  }
}`)
	if err := os.MkdirAll(filepath.Join(dir, "i18n"), 0o755); err != nil {
		t.Fatalf("create i18n dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "i18n", "zh-CN.json"), `{
  "name": "弹性云主机",
  "command.ecs.instance.list.description": "列出云主机",
  "parameter.ecs.instance.list.name.description": "按云主机名称过滤"
}`)
	return dir
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeTarGz(t *testing.T, archivePath, srcDir string) {
	t.Helper()
	testarchive.WriteTarGzFromDir(t, archivePath, srcDir)
}

func writeTarGzWithSymlink(t *testing.T, archivePath, srcDir string) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	if err := testarchive.WriteTarEntries(tarWriter, srcDir, ""); err != nil {
		t.Fatalf("write archive entries: %v", err)
	}
	if err := tarWriter.WriteHeader(&tar.Header{
		Name:     "linked.txt",
		Typeflag: tar.TypeSymlink,
		Linkname: "plugin.json",
		Mode:     0o777,
	}); err != nil {
		t.Fatalf("write symlink header: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
}

func writeTarGzWithPrefix(t *testing.T, archivePath, srcDir, prefix string) {
	t.Helper()
	testarchive.WriteTarGzFromDirWithPrefix(t, archivePath, srcDir, prefix)
}

func requireDiagnosticKey(t *testing.T, err error, want string) {
	t.Helper()
	got, ok := err.(interface{ MessageKey() string })
	if !ok {
		t.Fatalf("error %T does not expose a diagnostic key: %v", err, err)
	}
	if got.MessageKey() != want {
		t.Fatalf("diagnostic key = %q, want %q", got.MessageKey(), want)
	}
}
