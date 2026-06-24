/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

func TestLoadBundleAllowsOptionalMetadataAndMissingI18N(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	if err := os.Remove(filepath.Join(dir, "apis.json")); err != nil {
		t.Fatalf("remove apis.json: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "waiters.json")); err != nil {
		t.Fatalf("remove waiters.json: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "i18n")); err != nil {
		t.Fatalf("remove i18n: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {
      "id": "ecs.instance.list",
      "path": ["ecs", "instance", "list"],
      "table": "ecs.instance.list"
    }
  ]
}`)

	bundle, err := LoadBundle(dir, testCoreVersion())
	if err != nil {
		t.Fatalf("LoadBundle returned error: %v", err)
	}
	if len(bundle.APIs.Operations) != 0 || len(bundle.Waiters.Waiters) != 0 || len(bundle.I18N) != 0 {
		t.Fatalf("optional metadata was not empty: %#v", bundle)
	}
}

func TestLoadBundleReportsReadAndParseErrors(t *testing.T) {
	if _, err := LoadBundle(filepath.Join(t.TempDir(), "missing"), testCoreVersion()); err == nil {
		t.Fatal("LoadBundle returned nil error for missing plugin.json")
	}

	dir := writeBundle(t, "ecs", ">=0.1.0 <1.0.0")
	mustWrite(t, filepath.Join(dir, "commands.json"), `{`)
	if _, err := LoadBundle(dir, testCoreVersion()); err == nil {
		t.Fatalf("LoadBundle error = %v, want parse error", err)
	} else {
		requireDiagnosticKey(t, err, "error.parse_json_file")
	}

	dir = writeBundle(t, "ecs", ">=0.1.0 <1.0.0")
	mustWrite(t, filepath.Join(dir, "i18n", "en-US.json"), `{`)
	if _, err := LoadBundle(dir, testCoreVersion()); err == nil {
		t.Fatalf("LoadBundle i18n error = %v, want parse error", err)
	} else {
		requireDiagnosticKey(t, err, "error.parse_json_file")
	}

	for _, file := range []string{"apis.json", "waiters.json", "tables.json"} {
		dir = writeBundle(t, "ecs", ">=0.1.0 <1.0.0")
		mustWrite(t, filepath.Join(dir, file), `{`)
		if _, err := LoadBundle(dir, testCoreVersion()); err == nil {
			t.Fatalf("LoadBundle %s error = %v, want parse error", file, err)
		} else {
			requireDiagnosticKey(t, err, "error.parse_json_file")
		}
	}
}

func TestLoadBundleRejectsDuplicateCommandIDs(t *testing.T) {
	dir := writeBundle(t, "ecs", testCompatibleCoreConstraint())
	mustWrite(t, filepath.Join(dir, "commands.json"), `{
  "commands": [
    {"id": "ecs.instance.list", "path": ["ecs", "instance", "list"], "operation": "v4.ecs.instance.list", "table": "ecs.instance.list"},
    {"id": "ecs.instance.list", "path": ["ecs", "instance", "ls"], "operation": "v4.ecs.instance.list", "table": "ecs.instance.list"}
  ]
}`)

	if _, err := LoadBundle(dir, testCoreVersion()); err == nil {
		t.Fatalf("LoadBundle duplicate id error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.duplicate_command_id")
	}
}

func TestValidateManifestRejectsMissingFieldsAndInvalidEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		manifest Manifest
		want     string
	}{
		{name: "missing name", manifest: Manifest{}, want: "error.plugin_manifest_missing_name"},
		{name: "missing version", manifest: Manifest{Name: "ecs", Channel: "stable", Quality: "reviewed", Requires: Requirements{Ctyun: ">=0.1.0"}, API: APIInfo{Product: "ecs", CtyunProductID: 1}}, want: "error.plugin_manifest_missing_version"},
		{name: "invalid version", manifest: Manifest{Name: "ecs", Version: "v0.1", Channel: "stable", Quality: "reviewed", Requires: Requirements{Ctyun: ">=0.1.0"}, API: APIInfo{Product: "ecs", CtyunProductID: 1}}, want: "error.plugin_invalid_version"},
		{name: "missing requires", manifest: Manifest{Name: "ecs", Version: "0.1.0", Channel: "stable", Quality: "reviewed", API: APIInfo{Product: "ecs", CtyunProductID: 1}}, want: "error.plugin_missing_requires_ctyun"},
		{name: "missing product", manifest: Manifest{Name: "ecs", Version: "0.1.0", Channel: "stable", Quality: "reviewed", Requires: Requirements{Ctyun: ">=0.1.0"}, API: APIInfo{CtyunProductID: 1}}, want: "error.plugin_missing_api_product"},
		{name: "missing product id", manifest: Manifest{Name: "ecs", Version: "0.1.0", Channel: "stable", Quality: "reviewed", Requires: Requirements{Ctyun: ">=0.1.0"}, API: APIInfo{Product: "ecs"}}, want: "error.plugin_missing_api_ctyun_product_id"},
		{name: "unsupported quality", manifest: Manifest{Name: "ecs", Version: "0.1.0", Channel: "stable", Quality: "raw", Requires: Requirements{Ctyun: ">=0.1.0"}, API: APIInfo{Product: "ecs", CtyunProductID: 1}}, want: "error.plugin_unsupported_quality"},
		{name: "invalid endpoint", manifest: Manifest{Name: "ecs", Version: "0.1.0", Channel: "stable", Quality: "reviewed", Requires: Requirements{Ctyun: ">=0.1.0"}, API: APIInfo{Product: "ecs", CtyunProductID: 1, EndpointURL: "http://ctapi.example.test"}}, want: "error.plugin_invalid_api_endpoint_url"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateManifest(tc.manifest)
			if err == nil {
				t.Fatal("validateManifest returned nil error")
			}
			requireDiagnosticKey(t, err, tc.want)
		})
	}
	if !validEndpointURL("https://ctapi.example.test") {
		t.Fatal("validEndpointURL rejected HTTPS endpoint")
	}
	if validEndpointURL("https://ctapi.example.test/bad path") {
		t.Fatal("validEndpointURL accepted endpoint with whitespace")
	}
}

func TestValidationHelpersCoverPathAndParameterShapes(t *testing.T) {
	for _, name := range []string{"", ".hidden", "bad/name", "bad\\name", "bad..name", "-bad"} {
		if ValidName(name) {
			t.Fatalf("ValidName accepted %q", name)
		}
	}
	for _, path := range []string{"", "/abs", ".", "..", "../secret"} {
		if safeRelativePath(path) {
			t.Fatalf("safeRelativePath accepted %q", path)
		}
	}
	for _, segment := range []string{"", "{9bad}", "{bad-name}", "bad.name"} {
		if validCommandPathSegment(segment) {
			t.Fatalf("validCommandPathSegment accepted %q", segment)
		}
	}
	if !validCommandPathSegment("{imageID}") {
		t.Fatal("validCommandPathSegment rejected path argument")
	}

	shapeCases := []struct {
		command Command
		want    string
	}{
		{command: Command{}, want: "error.command_missing_id"},
		{command: Command{ID: "demo"}, want: "error.command_missing_path"},
		{command: Command{ID: "demo", Path: []string{"demo"}, Table: ""}, want: "error.command_missing_table"},
	}
	for _, tc := range shapeCases {
		err := validateCommandShape(tc.command)
		if err == nil {
			t.Fatal("validateCommandShape returned nil error")
		}
		requireDiagnosticKey(t, err, tc.want)
	}

	paramCases := []struct {
		command Command
		want    string
	}{
		{command: Command{ID: "demo", Parameters: []Parameter{{Flag: "name", Target: "displayName"}}}, want: "error.command_parameter_missing_name"},
		{command: Command{ID: "demo", Parameters: []Parameter{{Name: "name", Target: "displayName"}}}, want: "error.command_parameter_missing_flag"},
		{command: Command{ID: "demo", Parameters: []Parameter{{Name: "name", Flag: "name", Target: "displayName"}, {Name: "other", Flag: "name", Target: "other"}}}, want: "error.command_duplicate_parameter_flag"},
	}
	for _, tc := range paramCases {
		err := validateCommandParameters(tc.command)
		if err == nil {
			t.Fatal("validateCommandParameters returned nil error")
		}
		requireDiagnosticKey(t, err, tc.want)
	}
}

func TestValidateOperationsTablesAndWaitersRejectMissingShapes(t *testing.T) {
	operationCases := []struct {
		apis APIs
		want string
	}{
		{apis: APIs{Operations: map[string]Operation{"": {Method: "GET", Path: "/v4/demo"}}}, want: "error.operation_missing_id"},
		{apis: APIs{Operations: map[string]Operation{"op": {Path: "/v4/demo"}}}, want: "error.operation_missing_method"},
		{apis: APIs{Operations: map[string]Operation{"op": {Method: "GET"}}}, want: "error.operation_missing_path"},
		{apis: APIs{Operations: map[string]Operation{"op": {Method: "GET", Path: "v4/demo"}}}, want: "error.operation_path_must_start_with_slash"},
		{apis: APIs{Operations: map[string]Operation{"op": {Method: "GET", Path: "/v4/../demo"}}}, want: "error.operation_invalid_path"},
	}
	for _, tc := range operationCases {
		err := validateOperations(tc.apis)
		if err == nil {
			t.Fatal("validateOperations returned nil error")
		}
		requireDiagnosticKey(t, err, tc.want)
	}
	for _, path := range []string{"", "v4/demo", "//host/path", "/v4/demo?q=1", "/v4/./demo"} {
		if validOperationPath(path) {
			t.Fatalf("validOperationPath accepted %q", path)
		}
	}

	tableCases := []struct {
		tables Tables
		want   string
	}{
		{tables: Tables{Tables: map[string]Table{"": {RowPath: "items", Columns: []TableColumn{{Key: "id", Path: "id", Labels: allLabels("ID")}}}}}, want: "error.table_missing_id"},
		{tables: Tables{Tables: map[string]Table{"t": {Columns: []TableColumn{{Key: "id", Path: "id", Labels: allLabels("ID")}}}}}, want: "error.table_missing_row_path"},
		{tables: Tables{Tables: map[string]Table{"t": {RowPath: "items"}}}, want: "error.table_missing_columns"},
		{tables: Tables{Tables: map[string]Table{"t": {RowPath: "items", Columns: []TableColumn{{Path: "id", Labels: allLabels("ID")}}}}}, want: "error.table_column_missing_key"},
		{tables: Tables{Tables: map[string]Table{"t": {RowPath: "items", Columns: []TableColumn{{Key: "id", Labels: allLabels("ID")}}}}}, want: "error.table_column_missing_path"},
		{tables: Tables{Tables: map[string]Table{"t": {RowPath: "items", Columns: []TableColumn{{Key: "id", Path: "id", Labels: allLabels("ID")}, {Key: "id", Path: "id2", Labels: allLabels("ID")}}}}}, want: "error.table_duplicate_column_key"},
	}
	for _, tc := range tableCases {
		err := validateTables(tc.tables)
		if err == nil {
			t.Fatal("validateTables returned nil error")
		}
		requireDiagnosticKey(t, err, tc.want)
	}

	if err := validateWaiters(Waiters{Waiters: map[string]Waiter{"w": {MaxAttempts: -1}}}); err == nil {
		t.Fatalf("validateWaiters max attempts error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.waiter_negative_max_attempts")
	}
	if err := validateWaiters(Waiters{Waiters: map[string]Waiter{"w": {IntervalSeconds: -1}}}); err == nil {
		t.Fatalf("validateWaiters interval error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.waiter_negative_interval_seconds")
	}
}

func TestCommandMatchingHelpersSupportArgumentsAndPrefixes(t *testing.T) {
	bundle := Bundle{Commands: Commands{Commands: []Command{{
		ID:    "ims.image.show",
		Path:  []string{"ims", "image", "show", "{imageID}"},
		Table: "images",
	}}}}

	command, args, ok := FindCommandWithArgs(bundle, []string{"ims", "image", "show", "img-1"})
	if !ok || command.ID != "ims.image.show" || args["imageID"] != "img-1" {
		t.Fatalf("FindCommandWithArgs = %#v %#v %v", command, args, ok)
	}
	command, args, rest, ok := FindCommandPrefixWithArgs(bundle, []string{"ims", "image", "show", "img-4", "--name", "base"})
	if !ok || command.ID != "ims.image.show" || args["imageID"] != "img-4" || strings.Join(rest, " ") != "--name base" {
		t.Fatalf("FindCommandPrefixWithArgs direct path = %#v %#v %#v %v", command, args, rest, ok)
	}
	if _, _, _, ok := FindCommandPrefixWithArgs(bundle, []string{"ims", "image"}); ok {
		t.Fatal("FindCommandPrefixWithArgs matched incomplete path")
	}
	command, missing, ok := FindCommandMissingPathArgs(bundle, []string{"ims", "image", "show"})
	if !ok || command.ID != "ims.image.show" || strings.Join(missing, ",") != "imageID" {
		t.Fatalf("FindCommandMissingPathArgs = %#v %#v %v", command, missing, ok)
	}
	if _, _, ok := FindCommandMissingPathArgs(bundle, []string{"ims", "image"}); ok {
		t.Fatal("FindCommandMissingPathArgs matched incomplete static path")
	}
	argumentPrefixBundle := Bundle{Commands: Commands{Commands: []Command{{
		ID:   "ims.image.tag.show",
		Path: []string{"ims", "image", "{imageID}", "tag", "{tagID}"},
	}}}}
	command, missing, ok = FindCommandMissingPathArgs(argumentPrefixBundle, []string{"ims", "image", "img-1", "tag"})
	if !ok || command.ID != "ims.image.tag.show" || strings.Join(missing, ",") != "tagID" {
		t.Fatalf("FindCommandMissingPathArgs argument prefix = %#v %#v %v", command, missing, ok)
	}
	if _, _, ok := FindCommandMissingPathArgs(argumentPrefixBundle, []string{"ims", "image", "img-1", "bad"}); ok {
		t.Fatal("FindCommandMissingPathArgs matched mismatched static segment after argument")
	}
	if _, _, ok := FindCommandWithArgs(bundle, []string{"ecs", "image", "show", "img-1"}); ok {
		t.Fatal("FindCommandWithArgs matched unrelated path")
	}
	if _, _, _, ok := FindCommandPrefixWithArgs(bundle, []string{"ecs", "image", "show", "img-1"}); ok {
		t.Fatal("FindCommandPrefixWithArgs matched unrelated path")
	}
	if _, _, ok := FindCommandWithArgs(bundle, []string{"ims", "img", "img-3"}); ok {
		t.Fatal("FindCommandWithArgs matched unsupported alias")
	}
	if _, _, _, ok := FindCommandPrefixWithArgs(bundle, []string{"ims", "img", "img-3", "--name", "base"}); ok {
		t.Fatal("FindCommandPrefixWithArgs matched unsupported alias")
	}
	if args, ok := matchPath([]string{"ims"}, []string{"ecs"}); ok || args != nil {
		t.Fatalf("matchPath mismatch = %#v %v", args, ok)
	}
	if args, ok := matchPath([]string{"ims", "image"}, []string{"ims"}); ok || args != nil {
		t.Fatalf("matchPath length mismatch = %#v %v", args, ok)
	}
}

func TestVersionHelpersAndEqualStrings(t *testing.T) {
	if !versionMatches("0.1.0", "") {
		t.Fatal("versionMatches rejected empty constraint")
	}
	if versionMatches("0.1.0", "bogus") {
		t.Fatal("versionMatches accepted unsupported constraint")
	}
	if !versionMatches("0.2.0", ">=0.1.0 <1.0.0") {
		t.Fatal("versionMatches rejected compatible version")
	}
	if !versionMatches("0.2.0", ">=0.2.0-beta.1 <1.0.0") {
		t.Fatal("versionMatches rejected stable release after prerelease")
	}
	if versionMatches("0.1.0-alpha.1", ">=0.1.0 <1.0.0") {
		t.Fatal("versionMatches accepted prerelease below stable lower bound")
	}
	if versionMatches("1.0.0", ">=0.1.0 <1.0.0") {
		t.Fatal("versionMatches accepted incompatible upper bound")
	}
	if coreversion.CompareSemanticVersions("0.1.0", "0.2.0") >= 0 ||
		coreversion.CompareSemanticVersions("0.3.0", "0.2.0") <= 0 ||
		coreversion.CompareSemanticVersions("0.2.0", "0.2.0-beta.1") <= 0 {
		t.Fatal("CompareSemanticVersions ordering failed")
	}
	if !equalStrings([]string{"a", "b"}, []string{"a", "b"}) {
		t.Fatal("equalStrings rejected equal slices")
	}
	if equalStrings([]string{"a"}, []string{"a", "b"}) || equalStrings([]string{"a", "c"}, []string{"a", "b"}) {
		t.Fatal("equalStrings accepted different slices")
	}
}

func TestReadI18NSkipsDirectoriesAndNonJSONFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("create nested i18n dir: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "README.txt"), "ignore")
	mustWrite(t, filepath.Join(dir, "en-US.json"), `{"name":"Elastic Compute"}`)

	catalogs, err := readI18N(dir)
	if err != nil {
		t.Fatalf("readI18N returned error: %v", err)
	}
	if len(catalogs) != 1 || catalogs["en-US"]["name"] != "Elastic Compute" {
		t.Fatalf("catalogs = %#v, want only en-US JSON catalog", catalogs)
	}
}

func TestReadOptionalJSONReturnsReadErrors(t *testing.T) {
	if err := readOptionalJSON(t.TempDir(), &APIs{}); err == nil {
		t.Fatal("readOptionalJSON returned nil error for directory path")
	}
	if _, err := readI18N(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("readI18N missing dir returned error: %v", err)
	}
	if _, err := readI18N(t.TempDir()); err != nil {
		t.Fatalf("readI18N empty dir returned error: %v", err)
	}
	unreadable := t.TempDir()
	if err := os.Chmod(unreadable, 0); err != nil {
		t.Fatalf("chmod unreadable i18n dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o755) })
	if _, err := readI18N(unreadable); err == nil {
		t.Fatal("readI18N returned nil error for unreadable dir")
	}
}

func TestInstallHelpersHandleArchiveAndFilesystemErrors(t *testing.T) {
	destRoot := t.TempDir()

	if _, err := InstallLocalBundle(filepath.Join(t.TempDir(), "missing.tar.gz"), destRoot); err == nil {
		t.Fatal("InstallLocalBundle returned nil error for missing archive")
	}

	tmpFile := filepath.Join(t.TempDir(), "tmp-file")
	mustWrite(t, tmpFile, "not a dir")
	t.Setenv("TMPDIR", tmpFile)
	if _, err := InstallLocalBundle(filepath.Join(t.TempDir(), "bundle.tar.gz"), destRoot); err == nil {
		t.Fatal("InstallLocalBundle returned nil error when TMPDIR is a file")
	}
	t.Setenv("TMPDIR", "")

	badArchive := filepath.Join(t.TempDir(), "bad.tar.gz")
	mustWrite(t, badArchive, "not gzip")
	if _, err := InstallLocalBundle(badArchive, destRoot); err == nil {
		t.Fatal("InstallLocalBundle returned nil error for invalid gzip")
	}

	emptyArchive := filepath.Join(t.TempDir(), "empty.tar.gz")
	writeCustomTarGz(t, emptyArchive, nil)
	if _, err := InstallLocalBundle(emptyArchive, destRoot); err == nil {
		t.Fatalf("InstallLocalBundle empty archive error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.archive_missing_plugin_json")
	}

	multiArchive := filepath.Join(t.TempDir(), "multi.tar.gz")
	writeCustomTarGz(t, multiArchive, []tarEntry{
		{name: "one/plugin.json", body: minimalManifest("one")},
		{name: "two/plugin.json", body: minimalManifest("two")},
	})
	if _, err := InstallLocalBundle(multiArchive, destRoot); err == nil {
		t.Fatalf("InstallLocalBundle multi-root error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.archive_multiple_plugin_roots")
	}

	destRootFile := filepath.Join(t.TempDir(), "plugins")
	mustWrite(t, destRootFile, "not a dir")
	if _, err := InstallLocalBundle(writeBundle(t, "ecs", testCompatibleCoreConstraint()), destRootFile); err == nil {
		t.Fatal("InstallLocalBundle returned nil error when destination root is a file")
	}

	noName := t.TempDir()
	mustWrite(t, filepath.Join(noName, "plugin.json"), `{"version":"0.1.0"}`)
	if _, err := InstallLocalBundle(noName, destRoot); err == nil {
		t.Fatalf("InstallLocalBundle no-name error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.plugin_manifest_missing_name")
	}

	if _, err := InstallLocalBundle(t.TempDir(), destRoot); err == nil {
		t.Fatal("InstallLocalBundle returned nil error without plugin.json")
	}

	unwritableRoot := filepath.Join(t.TempDir(), "plugins")
	if err := os.Mkdir(unwritableRoot, 0o555); err != nil {
		t.Fatalf("create unwritable root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unwritableRoot, 0o755) })
	if _, err := InstallLocalBundle(writeBundle(t, "ecs", testCompatibleCoreConstraint()), unwritableRoot); err == nil {
		t.Fatal("InstallLocalBundle returned nil error when temp dir could not be created")
	}
}

func TestFindExtractedBundleRootAndReplaceDirHelpers(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "plugin.json"), minimalManifest("root"))
	if got, err := findExtractedBundleRoot(root); err != nil || got != root {
		t.Fatalf("findExtractedBundleRoot root = %q, %v", got, err)
	}

	notDir := filepath.Join(t.TempDir(), "not-dir")
	mustWrite(t, notDir, "x")
	if _, err := findExtractedBundleRoot(notDir); err == nil {
		t.Fatal("findExtractedBundleRoot returned nil error for file root")
	}
	if _, err := findExtractedBundleRoot(filepath.Join(t.TempDir(), "missing-root")); err == nil {
		t.Fatal("findExtractedBundleRoot returned nil error for missing root")
	}
	fileOnly := t.TempDir()
	mustWrite(t, filepath.Join(fileOnly, "README.txt"), "not a plugin")
	if _, err := findExtractedBundleRoot(fileOnly); err == nil {
		t.Fatalf("findExtractedBundleRoot file-only error = %v", err)
	} else {
		requireDiagnosticKey(t, err, "error.archive_missing_plugin_json")
	}
	unreadableCandidateRoot := t.TempDir()
	candidate := filepath.Join(unreadableCandidateRoot, "candidate")
	if err := os.Mkdir(candidate, 0o755); err != nil {
		t.Fatalf("create candidate: %v", err)
	}
	if err := os.Chmod(candidate, 0); err != nil {
		t.Fatalf("chmod candidate: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(candidate, 0o755) })
	if _, err := findExtractedBundleRoot(unreadableCandidateRoot); err == nil {
		t.Fatal("findExtractedBundleRoot returned nil error for unreadable candidate")
	}

	parent := t.TempDir()
	src := filepath.Join(parent, "src")
	dest := filepath.Join(parent, "dest")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("create src: %v", err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("create dest: %v", err)
	}
	mustWrite(t, filepath.Join(dest, "old.txt"), "old")
	mustWrite(t, filepath.Join(src, "new.txt"), "new")
	if err := replaceDir(src, dest); err != nil {
		t.Fatalf("replaceDir returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "new.txt")); err != nil {
		t.Fatalf("replacement missing new file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old file still present or unexpected error: %v", err)
	}

	parentFile := filepath.Join(t.TempDir(), "parent")
	mustWrite(t, parentFile, "not-dir")
	if err := replaceDir(t.TempDir(), filepath.Join(parentFile, "dest")); err == nil {
		t.Fatal("replaceDir returned nil error when destination parent is a file")
	}
	if err := replaceDir(filepath.Join(t.TempDir(), "missing-src"), filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("replaceDir returned nil error for missing source")
	}
	dest = filepath.Join(t.TempDir(), "dest")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatalf("create dest for missing replacement: %v", err)
	}
	if err := replaceDir(filepath.Join(t.TempDir(), "missing-src"), dest); err == nil {
		t.Fatal("replaceDir returned nil error when replacement source disappeared")
	}
	unwritableParent := t.TempDir()
	dest = filepath.Join(unwritableParent, "dest")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatalf("create dest in unwritable parent: %v", err)
	}
	if err := os.Chmod(unwritableParent, 0o555); err != nil {
		t.Fatalf("chmod parent: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unwritableParent, 0o755) })
	if err := replaceDir(t.TempDir(), dest); err == nil {
		t.Fatal("replaceDir returned nil error when backup rename could not be created")
	}
}

func TestExtractTarGzRejectsEscapingPathsAndCopyFileErrors(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "escape.tar.gz")
	writeCustomTarGz(t, archive, []tarEntry{{name: "../evil.txt", body: "evil"}})
	if err := extractTarGz(archive, t.TempDir()); err == nil {
		t.Fatal("extractTarGz returned nil error for escaping path")
	}

	truncated := filepath.Join(t.TempDir(), "truncated.tar.gz")
	writeTruncatedGzip(t, truncated)
	if err := extractTarGz(truncated, t.TempDir()); err == nil {
		t.Fatal("extractTarGz returned nil error for truncated tar stream")
	}

	conflict := filepath.Join(t.TempDir(), "conflict.tar.gz")
	writeCustomTarGz(t, conflict, []tarEntry{{name: "parent", body: "file"}, {name: "parent/child", body: "child"}})
	if err := extractTarGz(conflict, t.TempDir()); err == nil {
		t.Fatal("extractTarGz returned nil error for file/directory conflict")
	}
	dirConflict := filepath.Join(t.TempDir(), "dir-conflict.tar.gz")
	writeCustomTarGz(t, dirConflict, []tarEntry{{name: "parent", typ: tar.TypeDir}})
	dirConflictDest := t.TempDir()
	mustWrite(t, filepath.Join(dirConflictDest, "parent"), "file")
	if err := extractTarGz(dirConflict, dirConflictDest); err == nil {
		t.Fatal("extractTarGz returned nil error for directory entry over file")
	}
	fileConflict := filepath.Join(t.TempDir(), "file-conflict.tar.gz")
	writeCustomTarGz(t, fileConflict, []tarEntry{{name: "parent", body: "file"}})
	fileConflictDest := t.TempDir()
	if err := os.Mkdir(filepath.Join(fileConflictDest, "parent"), 0o755); err != nil {
		t.Fatalf("create conflicting directory: %v", err)
	}
	if err := extractTarGz(fileConflict, fileConflictDest); err == nil {
		t.Fatal("extractTarGz returned nil error for file entry over directory")
	}
	oversized := filepath.Join(t.TempDir(), "oversized.tar.gz")
	writeOversizedTarGz(t, oversized)
	if err := extractTarGz(oversized, t.TempDir()); err == nil {
		t.Fatal("extractTarGz returned nil error for short file body")
	}

	if err := copyFile(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("copyFile returned nil error for missing source")
	}

	parentFile := filepath.Join(t.TempDir(), "parent")
	mustWrite(t, parentFile, "not-dir")
	source := filepath.Join(t.TempDir(), "source")
	mustWrite(t, source, "data")
	if err := copyFile(source, filepath.Join(parentFile, "child")); err == nil {
		t.Fatal("copyFile returned nil error when destination parent is a file")
	}
	destDir := filepath.Join(t.TempDir(), "dest")
	if err := os.Mkdir(destDir, 0o755); err != nil {
		t.Fatalf("create destination directory: %v", err)
	}
	if err := copyFile(source, destDir); err == nil {
		t.Fatal("copyFile returned nil error when destination is a directory")
	}
	unreadableSrc := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(filepath.Join(unreadableSrc, "blocked"), 0o755); err != nil {
		t.Fatalf("create blocked dir: %v", err)
	}
	if err := os.Chmod(filepath.Join(unreadableSrc, "blocked"), 0); err != nil {
		t.Fatalf("chmod blocked dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(unreadableSrc, "blocked"), 0o755) })
	if err := copyDir(unreadableSrc, t.TempDir()); err == nil {
		t.Fatal("copyDir returned nil error for unreadable source tree")
	}
}

func allLabels(value string) map[string]string {
	return map[string]string{"zh-CN": value, "en-US": value, "en-GB": value}
}

type tarEntry struct {
	name string
	body string
	typ  byte
}

func writeCustomTarGz(t *testing.T, archivePath string, entries []tarEntry) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer closeTestResource(t, "archive", file.Close)
	gzipWriter := gzip.NewWriter(file)
	defer closeTestResource(t, "gzip writer", gzipWriter.Close)
	tarWriter := tar.NewWriter(gzipWriter)
	defer closeTestResource(t, "tar writer", tarWriter.Close)
	for _, entry := range entries {
		typ := entry.typ
		if typ == 0 {
			typ = tar.TypeReg
		}
		size := int64(len(entry.body))
		if typ == tar.TypeDir {
			size = 0
		}
		if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o644, Typeflag: typ, Size: size}); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if size > 0 {
			if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
				t.Fatalf("write body: %v", err)
			}
		}
	}
}

func writeOversizedTarGz(t *testing.T, archivePath string) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create oversized archive: %v", err)
	}
	defer closeTestResource(t, "oversized archive", file.Close)
	gzipWriter := gzip.NewWriter(file)
	defer closeTestResource(t, "oversized gzip writer", gzipWriter.Close)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: "short.txt", Mode: 0o644, Typeflag: tar.TypeReg, Size: 1024}); err != nil {
		t.Fatalf("write oversized header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("short")); err != nil {
		t.Fatalf("write body: %v", err)
	}
	// Leave the tar writer open so the archive advertises a larger body than it contains.
}

func writeTruncatedGzip(t *testing.T, archivePath string) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create truncated archive: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	if _, err := gzipWriter.Write([]byte("not a tar stream")); err != nil {
		t.Fatalf("write truncated archive: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
}

func minimalManifest(name string) string {
	return `{"name":"` + name + `","version":"0.1.0","channel":"stable","quality":"reviewed","requires":{"ctyun":"` + testCompatibleCoreConstraint() + `"},"api":{"product":"ecs","ctyun_product_id":25}}`
}

func closeTestResource(t *testing.T, name string, close func() error) {
	t.Helper()

	if err := close(); err != nil {
		t.Fatalf("close %s: %v", name, err)
	}
}
