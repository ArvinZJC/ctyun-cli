/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package plugin loads, validates, and matches metadata-defined CTyun command
// bundles.
package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
		return Bundle{}, fmt.Errorf("plugin %s requires ctyun %s, current version is %s", bundle.Manifest.Name, bundle.Manifest.Requires.Ctyun, coreVersion)
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
			return Bundle{}, fmt.Errorf("duplicate command id %s", command.ID)
		}
		seenCommandIDs[command.ID] = true
		key := strings.Join(command.Path, " ")
		if seenCommandPaths[key] {
			return Bundle{}, fmt.Errorf("duplicate command path %s", key)
		}
		seenCommandPaths[key] = true
		if err := validateCommandParameters(command); err != nil {
			return Bundle{}, err
		}
		if _, ok := bundle.Tables.Tables[command.Table]; !ok {
			return Bundle{}, fmt.Errorf("command %s references missing table %s", command.ID, command.Table)
		}
		if command.Operation != "" {
			if _, ok := bundle.APIs.Operations[command.Operation]; !ok {
				return Bundle{}, fmt.Errorf("command %s references missing operation %s", command.ID, command.Operation)
			}
		}
	}
	return bundle, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("plugin manifest is missing name")
	}
	if !ValidName(manifest.Name) {
		return fmt.Errorf("invalid plugin name %q", manifest.Name)
	}
	if manifest.Version == "" {
		return fmt.Errorf("plugin %s manifest is missing version", manifest.Name)
	}
	if !oneOf(manifest.Channel, "stable", "beta", "edge") {
		return fmt.Errorf("plugin %s has unsupported channel %q", manifest.Name, manifest.Channel)
	}
	if !oneOf(manifest.Quality, "generated", "reviewed", "curated") {
		return fmt.Errorf("plugin %s has unsupported quality %q", manifest.Name, manifest.Quality)
	}
	if manifest.Requires.Ctyun == "" {
		return fmt.Errorf("plugin %s manifest is missing requires.ctyun", manifest.Name)
	}
	if manifest.API.Product == "" {
		return fmt.Errorf("plugin %s manifest is missing api.product", manifest.Name)
	}
	if manifest.API.CtyunProductID <= 0 {
		return fmt.Errorf("plugin %s manifest is missing api.ctyun_product_id", manifest.Name)
	}
	if manifest.API.EndpointURL != "" && !validEndpointURL(manifest.API.EndpointURL) {
		return fmt.Errorf("plugin %s has invalid api.endpoint_url %q", manifest.Name, manifest.API.EndpointURL)
	}
	return nil
}

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

func validateCommandShape(command Command) error {
	if command.ID == "" {
		return fmt.Errorf("command is missing id")
	}
	if len(command.Path) == 0 {
		return fmt.Errorf("command %s is missing path", command.ID)
	}
	for _, part := range command.Path {
		if !validCommandPathSegment(part) {
			return fmt.Errorf("command %s has invalid path segment %q", command.ID, part)
		}
	}
	if command.Table == "" {
		return fmt.Errorf("command %s is missing table", command.ID)
	}
	if command.FixtureResponse != "" && !safeRelativePath(command.FixtureResponse) {
		return fmt.Errorf("command %s has invalid fixture_response %q", command.ID, command.FixtureResponse)
	}
	return nil
}

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

func validateCommandParameters(command Command) error {
	seen := make(map[string]bool, len(command.Parameters))
	for _, parameter := range command.Parameters {
		if parameter.Name == "" {
			return fmt.Errorf("command %s has parameter missing name", command.ID)
		}
		if parameter.Flag == "" {
			return fmt.Errorf("command %s parameter %s is missing flag", command.ID, parameter.Name)
		}
		if parameter.Target == "" {
			return fmt.Errorf("command %s parameter %s is missing target", command.ID, parameter.Name)
		}
		if seen[parameter.Flag] {
			return fmt.Errorf("command %s has duplicate parameter flag %s", command.ID, parameter.Flag)
		}
		if parameter.Pattern != "" {
			if _, err := regexp.Compile(parameter.Pattern); err != nil {
				return fmt.Errorf("command %s parameter %s has invalid pattern: %w", command.ID, parameter.Name, err)
			}
		}
		seen[parameter.Flag] = true
	}
	return nil
}

func validateOperations(apis APIs) error {
	for id, operation := range apis.Operations {
		if id == "" {
			return fmt.Errorf("operation is missing id")
		}
		if operation.Method == "" {
			return fmt.Errorf("operation %s is missing method", id)
		}
		if !oneOf(operation.Method, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete) {
			return fmt.Errorf("operation %s has unsupported method %q", id, operation.Method)
		}
		if operation.Path == "" {
			return fmt.Errorf("operation %s is missing path", id)
		}
		if !strings.HasPrefix(operation.Path, "/") {
			return fmt.Errorf("operation %s path must start with /", id)
		}
		if !validOperationPath(operation.Path) {
			return fmt.Errorf("operation %s has invalid path %q", id, operation.Path)
		}
	}
	return nil
}

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

func validateTables(tables Tables) error {
	for id, table := range tables.Tables {
		if id == "" {
			return fmt.Errorf("table is missing id")
		}
		if table.RowPath == "" {
			return fmt.Errorf("table %s is missing row_path", id)
		}
		if len(table.Columns) == 0 {
			return fmt.Errorf("table %s is missing columns", id)
		}
		seen := make(map[string]bool, len(table.Columns))
		for _, column := range table.Columns {
			if column.Key == "" {
				return fmt.Errorf("table %s has column missing key", id)
			}
			if column.Path == "" {
				return fmt.Errorf("table %s column %s is missing path", id, column.Key)
			}
			if seen[column.Key] {
				return fmt.Errorf("table %s has duplicate column key %s", id, column.Key)
			}
			seen[column.Key] = true
			for _, language := range []string{"zh-CN", "en-US", "en-GB"} {
				if column.Labels[language] == "" {
					return fmt.Errorf("table %s column %s is missing %s label", id, column.Key, language)
				}
			}
		}
	}
	return nil
}

func validateWaiters(waiters Waiters) error {
	for id, waiter := range waiters.Waiters {
		if waiter.TimeoutSeconds != nil {
			return fmt.Errorf("waiter %s uses unsupported timeout_seconds; use max_attempts and interval_seconds for polling, and profile timeout_seconds or --timeout for HTTP request timeouts", id)
		}
		if waiter.MaxAttempts < 0 {
			return fmt.Errorf("waiter %s has negative max_attempts", id)
		}
		if waiter.IntervalSeconds < 0 {
			return fmt.Errorf("waiter %s has negative interval_seconds", id)
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

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func readOptionalJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

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

func versionMatches(current, constraint string) bool {
	if strings.TrimSpace(constraint) == "" {
		return true
	}
	for _, part := range strings.Fields(constraint) {
		switch {
		case strings.HasPrefix(part, ">="):
			if compareVersion(current, strings.TrimPrefix(part, ">=")) < 0 {
				return false
			}
		case strings.HasPrefix(part, "<"):
			if compareVersion(current, strings.TrimPrefix(part, "<")) >= 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func compareVersion(left, right string) int {
	lv := parseVersion(left)
	rv := parseVersion(right)
	for i := 0; i < len(lv); i++ {
		if lv[i] < rv[i] {
			return -1
		}
		if lv[i] > rv[i] {
			return 1
		}
	}
	return 0
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func parseVersion(version string) [3]int {
	var parsed [3]int
	parts := strings.Split(version, ".")
	for i := 0; i < len(parsed) && i < len(parts); i++ {
		value, _ := strconv.Atoi(parts[i])
		parsed[i] = value
	}
	return parsed
}

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
