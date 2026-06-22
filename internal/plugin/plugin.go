/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package plugin loads, validates, and matches metadata-defined CTyun command
// bundles.
package plugin

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

// Manifest is the plugin.json contract for one plugin bundle.
type Manifest struct {
	Name     string       `json:"name"`
	Version  string       `json:"version"`
	Channel  string       `json:"channel"`
	Quality  string       `json:"quality"`
	Requires Requirements `json:"requires"`
	API      APIInfo      `json:"api"`
}

// Requirements declares core ctyun compatibility for a plugin bundle.
type Requirements struct {
	Ctyun string `json:"ctyun"`
}

// APIInfo describes the CTyun product and endpoint behind a plugin bundle.
type APIInfo struct {
	Product        string `json:"product"`
	CtyunProductID int    `json:"ctyun_product_id"`
	DocsVersion    string `json:"docs_version"`
	EndpointURL    string `json:"endpoint_url"`
}

// Commands is the top-level commands.json document.
type Commands struct {
	Commands []Command `json:"commands"`
}

// APIs is the top-level apis.json document.
type APIs struct {
	Operations map[string]Operation `json:"operations"`
}

// Operation maps a command to one CTyun HTTP request shape.
type Operation struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	ContentType string            `json:"content_type"`
	Query       map[string]string `json:"query"`
	Headers     map[string]string `json:"headers"`
	Body        map[string]string `json:"body"`
	Retryable   bool              `json:"retryable"`
}

// Command describes one metadata-defined CLI command path and its bindings.
type Command struct {
	ID              string      `json:"id"`
	Path            []string    `json:"path"`
	Operation       string      `json:"operation"`
	Table           string      `json:"table"`
	Parameters      []Parameter `json:"parameters"`
	FixtureResponse string      `json:"fixture_response"`
	DocsURL         string      `json:"docs_url"`
	Examples        []string    `json:"examples"`
	Dangerous       Dangerous   `json:"dangerous"`
}

// Parameter defines one command flag and how its value binds into a request or
// table operation.
type Parameter struct {
	Name          string   `json:"name"`
	Flag          string   `json:"flag"`
	Target        string   `json:"target"`
	Required      bool     `json:"required"`
	AllowedValues []string `json:"allowed_values"`
	Pattern       string   `json:"pattern"`
	Description   string   `json:"description"`
}

// Dangerous declares the confirmation contract for state-changing commands.
type Dangerous struct {
	Confirm string `json:"confirm"`
	Message string `json:"message"`
}

// Waiters is the top-level waiters.json document.
type Waiters struct {
	Waiters map[string]Waiter `json:"waiters"`
}

// Waiter describes how a command should poll and interpret operation state.
type Waiter struct {
	Path            string `json:"path"`
	Success         string `json:"success"`
	Failure         string `json:"failure"`
	MaxAttempts     int    `json:"max_attempts"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  *int   `json:"timeout_seconds"`
}

// Tables is the top-level tables.json document.
type Tables struct {
	Tables map[string]Table `json:"tables"`
}

// Table defines how response JSON becomes stable-key table rows.
type Table struct {
	RowPath string        `json:"row_path"`
	Columns []TableColumn `json:"columns"`
}

// TableColumn maps a stable output key to a response JSON path and localized
// labels.
type TableColumn struct {
	Key    string            `json:"key"`
	Path   string            `json:"path"`
	Labels map[string]string `json:"labels"`
}

// Bundle contains a loaded plugin directory and all validated metadata files.
type Bundle struct {
	Dir      string
	Manifest Manifest
	Commands Commands
	APIs     APIs
	Tables   Tables
	Waiters  Waiters
	I18N     map[string]map[string]string
}

// LoadBundle reads and validates a plugin bundle for the supplied core version.
func LoadBundle(dir, coreVersion string) (Bundle, error) {
	var bundle Bundle
	bundle.Dir = dir

	if err := readJSON(filepath.Join(dir, "plugin.json"), &bundle.Manifest); err != nil {
		return Bundle{}, err
	}
	if err := validateManifest(bundle.Manifest); err != nil {
		return Bundle{}, err
	}
	if err := readJSON(filepath.Join(dir, "commands.json"), &bundle.Commands); err != nil {
		return Bundle{}, err
	}
	if err := readOptionalJSON(filepath.Join(dir, "apis.json"), &bundle.APIs); err != nil {
		return Bundle{}, err
	}
	if err := readJSON(filepath.Join(dir, "tables.json"), &bundle.Tables); err != nil {
		return Bundle{}, err
	}
	if err := readOptionalJSON(filepath.Join(dir, "waiters.json"), &bundle.Waiters); err != nil {
		return Bundle{}, err
	}
	i18n, err := readI18N(filepath.Join(dir, "i18n"))
	if err != nil {
		return Bundle{}, err
	}
	bundle.I18N = i18n
	if !versionMatches(coreVersion, bundle.Manifest.Requires.Ctyun) {
		return Bundle{}, diagnostic.New("error.plugin_version", bundle.Manifest.Name, bundle.Manifest.Requires.Ctyun, coreVersion)
	}
	if err := validateTables(bundle.Tables); err != nil {
		return Bundle{}, err
	}
	if err := validateOperations(bundle.APIs); err != nil {
		return Bundle{}, err
	}
	if err := validateWaiters(bundle.Waiters); err != nil {
		return Bundle{}, err
	}
	seenCommandIDs := make(map[string]bool, len(bundle.Commands.Commands))
	seenCommandPaths := make(map[string]bool, len(bundle.Commands.Commands))
	for _, command := range bundle.Commands.Commands {
		if err := validateCommandShape(command); err != nil {
			return Bundle{}, err
		}
		if seenCommandIDs[command.ID] {
			return Bundle{}, diagnostic.New("error.duplicate_command_id", command.ID)
		}
		seenCommandIDs[command.ID] = true
		key := strings.Join(command.Path, " ")
		if seenCommandPaths[key] {
			return Bundle{}, diagnostic.New("error.duplicate_command_path", key)
		}
		seenCommandPaths[key] = true
		if err := validateCommandParameters(command); err != nil {
			return Bundle{}, err
		}
		if _, ok := bundle.Tables.Tables[command.Table]; !ok {
			return Bundle{}, diagnostic.New("error.command_missing_table_ref", command.ID, command.Table)
		}
		if command.Operation != "" {
			if _, ok := bundle.APIs.Operations[command.Operation]; !ok {
				return Bundle{}, diagnostic.New("error.command_missing_operation_ref", command.ID, command.Operation)
			}
		}
	}
	return bundle, nil
}

// validateManifest checks required plugin identity, release, and product
// metadata.
func validateManifest(manifest Manifest) error {
	if manifest.Name == "" {
		return diagnostic.New("error.plugin_manifest_missing_name")
	}
	if !ValidName(manifest.Name) {
		return diagnostic.New("error.plugin_name", manifest.Name)
	}
	if manifest.Version == "" {
		return diagnostic.New("error.plugin_manifest_missing_version", manifest.Name)
	}
	if !coreversion.IsSemanticVersion(manifest.Version) {
		return diagnostic.New("error.plugin_invalid_version", manifest.Name, manifest.Version)
	}
	if !oneOf(manifest.Channel, "stable", "beta", "alpha") {
		return diagnostic.New("error.plugin_unsupported_channel", manifest.Name, manifest.Channel)
	}
	if !oneOf(manifest.Quality, "generated", "reviewed", "curated") {
		return diagnostic.New("error.plugin_unsupported_quality", manifest.Name, manifest.Quality)
	}
	if manifest.Requires.Ctyun == "" {
		return diagnostic.New("error.plugin_missing_requires_ctyun", manifest.Name)
	}
	if manifest.API.Product == "" {
		return diagnostic.New("error.plugin_missing_api_product", manifest.Name)
	}
	if manifest.API.CtyunProductID <= 0 {
		return diagnostic.New("error.plugin_missing_api_ctyun_product_id", manifest.Name)
	}
	if manifest.API.EndpointURL != "" && !validEndpointURL(manifest.API.EndpointURL) {
		return diagnostic.New("error.plugin_invalid_api_endpoint_url", manifest.Name, manifest.API.EndpointURL)
	}
	return nil
}

// validEndpointURL reports whether raw is a safe HTTPS API endpoint.
func validEndpointURL(raw string) bool {
	if strings.ContainsAny(raw, " \t\r\n") {
		return false
	}
	return strings.HasPrefix(raw, "https://")
}

// ValidName reports whether name is safe for plugin directories and registry
// entries.
func ValidName(name string) bool {
	if name == "" || strings.HasPrefix(name, ".") {
		return false
	}
	for _, part := range []string{"/", "\\", ".."} {
		if strings.Contains(name, part) {
			return false
		}
	}
	matched, err := regexp.MatchString(`^[A-Za-z0-9][A-Za-z0-9._-]*$`, name)
	return err == nil && matched
}

// validateCommandShape checks command identity, path, table, and fixture shape.
func validateCommandShape(command Command) error {
	if command.ID == "" {
		return diagnostic.New("error.command_missing_id")
	}
	if len(command.Path) == 0 {
		return diagnostic.New("error.command_missing_path", command.ID)
	}
	for _, part := range command.Path {
		if !validCommandPathSegment(part) {
			return diagnostic.New("error.command_invalid_path_segment", command.ID, part)
		}
	}
	if command.Table == "" {
		return diagnostic.New("error.command_missing_table", command.ID)
	}
	if command.FixtureResponse != "" && !safeRelativePath(command.FixtureResponse) {
		return diagnostic.New("error.command_invalid_fixture_response", command.ID, command.FixtureResponse)
	}
	return nil
}

// validCommandPathSegment accepts literal command words or {argument}
// placeholders.
func validCommandPathSegment(segment string) bool {
	if segment == "" {
		return false
	}
	if strings.HasPrefix(segment, "{") || strings.HasSuffix(segment, "}") {
		matched, err := regexp.MatchString(`^\{[A-Za-z][A-Za-z0-9_]*\}$`, segment)
		return err == nil && matched
	}
	matched, err := regexp.MatchString(`^[A-Za-z0-9][A-Za-z0-9_-]*$`, segment)
	return err == nil && matched
}

// safeRelativePath rejects absolute paths and parent-directory escapes.
func safeRelativePath(path string) bool {
	if path == "" || filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return false
	}
	return true
}

// validateCommandParameters checks flag bindings and validation patterns.
func validateCommandParameters(command Command) error {
	seen := make(map[string]bool, len(command.Parameters))
	for _, parameter := range command.Parameters {
		if parameter.Name == "" {
			return diagnostic.New("error.command_parameter_missing_name", command.ID)
		}
		if parameter.Flag == "" {
			return diagnostic.New("error.command_parameter_missing_flag", command.ID, parameter.Name)
		}
		if parameter.Target == "" {
			return diagnostic.New("error.command_parameter_missing_target", command.ID, parameter.Name)
		}
		if seen[parameter.Flag] {
			return diagnostic.New("error.command_duplicate_parameter_flag", command.ID, parameter.Flag)
		}
		if parameter.Pattern != "" {
			if _, err := regexp.Compile(parameter.Pattern); err != nil {
				return diagnostic.Wrap("error.command_parameter_invalid_pattern", err, command.ID, parameter.Name)
			}
		}
		seen[parameter.Flag] = true
	}
	return nil
}

// validateOperations checks API operation method and path metadata.
func validateOperations(apis APIs) error {
	for id, operation := range apis.Operations {
		if id == "" {
			return diagnostic.New("error.operation_missing_id")
		}
		if operation.Method == "" {
			return diagnostic.New("error.operation_missing_method", id)
		}
		if !oneOf(operation.Method, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete) {
			return diagnostic.New("error.operation_unsupported_method", id, operation.Method)
		}
		if operation.Path == "" {
			return diagnostic.New("error.operation_missing_path", id)
		}
		if !strings.HasPrefix(operation.Path, "/") {
			return diagnostic.New("error.operation_path_must_start_with_slash", id)
		}
		if !validOperationPath(operation.Path) {
			return diagnostic.New("error.operation_invalid_path", id, operation.Path)
		}
	}
	return nil
}

// validOperationPath accepts clean absolute API paths without query fragments.
func validOperationPath(path string) bool {
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}
	if strings.ContainsAny(path, " \t\r\n?#") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	return true
}

// validateTables checks row paths, stable column keys, and required labels.
func validateTables(tables Tables) error {
	for id, table := range tables.Tables {
		if id == "" {
			return diagnostic.New("error.table_missing_id")
		}
		if table.RowPath == "" {
			return diagnostic.New("error.table_missing_row_path", id)
		}
		if len(table.Columns) == 0 {
			return diagnostic.New("error.table_missing_columns", id)
		}
		seen := make(map[string]bool, len(table.Columns))
		for _, column := range table.Columns {
			if column.Key == "" {
				return diagnostic.New("error.table_column_missing_key", id)
			}
			if column.Path == "" {
				return diagnostic.New("error.table_column_missing_path", id, column.Key)
			}
			if seen[column.Key] {
				return diagnostic.New("error.table_duplicate_column_key", id, column.Key)
			}
			seen[column.Key] = true
			for _, language := range []string{"zh-CN", "en-US", "en-GB"} {
				if column.Labels[language] == "" {
					return diagnostic.New("error.table_column_missing_label", id, column.Key, language)
				}
			}
		}
	}
	return nil
}

// validateWaiters checks polling limits and rejects unsupported timeout fields.
func validateWaiters(waiters Waiters) error {
	for id, waiter := range waiters.Waiters {
		if waiter.TimeoutSeconds != nil {
			return diagnostic.New("error.waiter_unsupported_timeout_seconds", id)
		}
		if waiter.MaxAttempts < 0 {
			return diagnostic.New("error.waiter_negative_max_attempts", id)
		}
		if waiter.IntervalSeconds < 0 {
			return diagnostic.New("error.waiter_negative_interval_seconds", id)
		}
	}
	return nil
}

// FindCommand returns the command whose path exactly matches path.
func FindCommand(bundle Bundle, path []string) (Command, bool) {
	command, _, ok := FindCommandWithArgs(bundle, path)
	return command, ok
}

// FindCommandWithArgs matches path against command templates and returns
// captured placeholder arguments.
func FindCommandWithArgs(bundle Bundle, path []string) (Command, map[string]string, bool) {
	for _, command := range bundle.Commands.Commands {
		if args, ok := matchPath(command.Path, path); ok {
			return command, args, true
		}
	}
	return Command{}, nil, false
}

// FindCommandPrefixWithArgs matches the longest command prefix and returns any
// remaining path segments for command-specific option parsing.
func FindCommandPrefixWithArgs(bundle Bundle, path []string) (Command, map[string]string, []string, bool) {
	for _, command := range bundle.Commands.Commands {
		if args, rest, ok := matchPathPrefix(command.Path, path); ok {
			return command, args, rest, true
		}
	}
	return Command{}, nil, nil, false
}

// matchPathPrefix matches a command path prefix and returns remaining tokens.
func matchPathPrefix(pattern, path []string) (map[string]string, []string, bool) {
	if len(pattern) > len(path) {
		return nil, nil, false
	}
	args, ok := matchPath(pattern, path[:len(pattern)])
	if !ok {
		return nil, nil, false
	}
	return args, path[len(pattern):], true
}

// matchPath matches a full command path and captures placeholder arguments.
func matchPath(pattern, path []string) (map[string]string, bool) {
	if len(pattern) != len(path) {
		return nil, false
	}
	args := make(map[string]string)
	for i := range pattern {
		if strings.HasPrefix(pattern[i], "{") && strings.HasSuffix(pattern[i], "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(pattern[i], "{"), "}")
			args[name] = path[i]
			continue
		}
		if pattern[i] != path[i] {
			return nil, false
		}
	}
	return args, true
}

// readJSON reads and unmarshals a required JSON metadata file.
func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return diagnostic.Wrap("error.parse_json_file", err, path)
	}
	return nil
}

// readOptionalJSON reads an optional JSON metadata file when it exists.
func readOptionalJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return diagnostic.Wrap("error.parse_json_file", err, path)
	}
	return nil
}

// readI18N reads plugin localization catalogs from an i18n directory.
func readI18N(dir string) (map[string]map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	catalogs := make(map[string]map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		language := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		var catalog map[string]string
		if err := readJSON(filepath.Join(dir, entry.Name()), &catalog); err != nil {
			return nil, err
		}
		catalogs[language] = catalog
	}
	return catalogs, nil
}

// versionMatches evaluates the simple version constraint language used by
// plugin manifests.
func versionMatches(current, constraint string) bool {
	if strings.TrimSpace(constraint) == "" {
		return true
	}
	for _, part := range strings.Fields(constraint) {
		switch {
		case strings.HasPrefix(part, ">="):
			if coreversion.CompareSemanticVersions(current, strings.TrimPrefix(part, ">=")) < 0 {
				return false
			}
		case strings.HasPrefix(part, "<"):
			if coreversion.CompareSemanticVersions(current, strings.TrimPrefix(part, "<")) >= 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// oneOf reports whether value is present in allowed.
func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

// equalStrings reports whether two string slices have identical contents.
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
