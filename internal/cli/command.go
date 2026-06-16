/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// runPluginCommand resolves a metadata-defined command and renders its result.
func runPluginCommand(stdout, stderr io.Writer, opts globalOptions, args []string, installedRoot string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper) error {
	bundle, command, commandArgs, parameterValues, ok, err := findPluginCommand(args, installedRoot, opts.Language)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unknown command %q", strings.Join(args, " "))
	}
	if command.Dangerous.Confirm != "" && !opts.Yes {
		message := command.Dangerous.Message
		if message == "" {
			message = command.ID
		}
		return localizedConfirmationRequired(message, opts.Language)
	}

	table := bundle.Tables.Tables[command.Table]
	loadResponse := func() (map[string]any, error) {
		return loadCommandResponse(bundle, command, commandArgs, parameterValues, opts, profile, getenv, transport, debugWriter(opts, stderr))
	}
	// Keep loading separate from rendering so waiters can poll the same command
	// without duplicating metadata resolution.
	payload, err := loadResponse()
	if err != nil {
		return err
	}

	switch opts.Output {
	case "json":
		rendered, _ := output.RenderJSON(payload)
		if _, err = io.WriteString(stdout, rendered); err != nil {
			return err
		}
		return renderWaiter(stderr, bundle, opts.Waiter, payload, loadResponse)
	case "table":
		rows, err := rowsFromPayload(payload, table)
		if err != nil {
			return err
		}
		// Fixture output is filtered with the same stable table keys that live
		// requests use for parameterized API calls.
		rows = filterRowsByParameters(rows, table, command.Parameters, parameterValues)
		if err := validateFilterSortKeys(table, opts.Filter, opts.Sort); err != nil {
			return err
		}
		rows, err = output.FilterRows(rows, opts.Filter)
		if err != nil {
			return err
		}
		rows, err = output.SortRows(rows, opts.Sort)
		if err != nil {
			return err
		}
		columns := tableColumns(table, opts.Language)
		rendered, err := output.RenderTable(rows, columns, output.TableOptions{
			Columns:  opts.Columns,
			NoHeader: opts.NoHeader,
			Style:    opts.Table,
		})
		if err != nil {
			return err
		}
		if _, err = io.WriteString(stdout, rendered); err != nil {
			return err
		}
		return renderWaiter(stdout, bundle, opts.Waiter, payload, loadResponse)
	default:
		return fmt.Errorf("unsupported output %q", opts.Output)
	}
}

// filterRowsByParameters applies parameter-derived filters to fixture rows.
func filterRowsByParameters(rows []map[string]string, table plugin.Table, parameters []plugin.Parameter, values map[string]string) []map[string]string {
	if len(rows) == 0 || len(parameters) == 0 || len(values) == 0 {
		return rows
	}
	columnKeyByPath := make(map[string]string, len(table.Columns))
	for _, column := range table.Columns {
		columnKeyByPath[column.Path] = column.Key
	}
	targetByName := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		targetByName[parameter.Name] = columnKeyByPath[parameter.Target]
	}
	filtered := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		matches := true
		for name, value := range values {
			if value == "" {
				continue
			}
			target := targetByName[name]
			if target == "" {
				continue
			}
			if row[target] != value {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

// validateFilterSortKeys ensures filter and sort expressions use stable table
// keys.
func validateFilterSortKeys(table plugin.Table, filter, sort string) error {
	keys := make(map[string]bool, len(table.Columns))
	for _, column := range table.Columns {
		keys[column.Key] = true
	}
	if key := filterKey(filter); key != "" && !keys[key] {
		return fmt.Errorf("unknown filter key %q; use stable column keys", key)
	}
	if key := sortKey(sort); key != "" && !keys[key] {
		return fmt.Errorf("unknown sort key %q; use stable column keys", key)
	}
	return nil
}

// filterKey extracts the stable key from a filter expression.
func filterKey(expression string) string {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return ""
	}
	parts := strings.SplitN(expression, "!=", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(expression, "=", 2)
	}
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

// sortKey extracts the stable key from a sort expression.
func sortKey(expression string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(expression), "-"))
}

// renderWaiter evaluates optional waiter metadata and writes the final state.
func renderWaiter(stdout io.Writer, bundle plugin.Bundle, waiterID string, payload map[string]any, loadResponse func() (map[string]any, error)) error {
	if waiterID == "" {
		return nil
	}
	spec, ok := bundle.Waiters.Waiters[waiterID]
	if !ok {
		return fmt.Errorf("unknown waiter %q", waiterID)
	}
	attempts := spec.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	var state waiter.State
	for attempt := 1; attempt <= attempts; attempt++ {
		var err error
		state, err = waiter.Evaluate(waiter.Spec{
			Path:    spec.Path,
			Success: spec.Success,
			Failure: spec.Failure,
		}, payload)
		if err != nil {
			return err
		}
		if state != waiter.Pending {
			break
		}
		if attempt == attempts {
			state = waiter.Timeout
			break
		}
		if spec.IntervalSeconds > 0 {
			time.Sleep(time.Duration(spec.IntervalSeconds) * time.Second)
		}
		payload, err = loadResponse()
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "waiter %s: %s\n", waiterID, state)
	return nil
}

// findPluginCommand matches command arguments to a plugin command and parses
// command-specific flags.
func findPluginCommand(args []string, installedRoot, language string) (plugin.Bundle, plugin.Command, map[string]string, map[string]string, bool, error) {
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return plugin.Bundle{}, plugin.Command{}, nil, nil, false, err
	}
	for _, bundle := range bundles {
		command, commandArgs, rest, ok := plugin.FindCommandPrefixWithArgs(bundle, args)
		if ok {
			parameterValues, err := parseCommandParameters(command, rest, language)
			if err != nil {
				return plugin.Bundle{}, plugin.Command{}, nil, nil, false, err
			}
			return bundle, command, commandArgs, parameterValues, true, nil
		}
	}
	return plugin.Bundle{}, plugin.Command{}, nil, nil, false, nil
}

// parseCommandParameters parses metadata-defined command flags.
func parseCommandParameters(command plugin.Command, args []string, language string) (map[string]string, error) {
	byFlag := make(map[string]plugin.Parameter, len(command.Parameters))
	for _, parameter := range command.Parameters {
		byFlag[parameter.Flag] = parameter
	}

	values := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return nil, localizedUnexpectedArgument(arg, command.ID, language)
		}
		flag := strings.TrimPrefix(arg, "--")
		value := ""
		if name, inline, ok := strings.Cut(flag, "="); ok {
			flag = name
			value = inline
		} else {
			i++
			if i >= len(args) {
				return nil, localizedFlagRequiresValue(flag, language)
			}
			value = args[i]
		}
		parameter, ok := byFlag[flag]
		if !ok {
			return nil, localizedUnknownOption(flag, command.ID, language)
		}
		if err := validateParameterValue(command, parameter, value, language); err != nil {
			return nil, err
		}
		values[parameter.Name] = value
	}

	for _, parameter := range command.Parameters {
		if parameter.Required && values[parameter.Name] == "" {
			return nil, localizedMissingRequiredFlag(command.ID, parameter.Flag, language)
		}
	}
	return values, nil
}

// validateParameterValue applies allowed-value and pattern validation for one
// parameter.
func validateParameterValue(command plugin.Command, parameter plugin.Parameter, value, language string) error {
	if len(parameter.AllowedValues) > 0 && !slices.Contains(parameter.AllowedValues, value) {
		return localizedAllowedValuesError(command.ID, parameter.Flag, strings.Join(parameter.AllowedValues, ","), language)
	}
	if parameter.Pattern != "" {
		matched, err := regexp.MatchString(parameter.Pattern, value)
		if err != nil {
			return localizedInvalidPattern(command.ID, parameter.Flag, err, language)
		}
		if !matched {
			return localizedPatternMismatch(command.ID, parameter.Flag, parameter.Pattern, language)
		}
	}
	return nil
}

// localizedConfirmationRequired returns the dangerous-command confirmation
// error.
func localizedConfirmationRequired(message, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 需要确认：请使用 --yes 重新执行", message)
	}
	return fmt.Errorf("confirmation required for %s: rerun with --yes", message)
}

// localizedUnexpectedArgument returns an error for extra command arguments.
func localizedUnexpectedArgument(arg, commandID, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 不支持参数 %q", commandID, arg)
	}
	return fmt.Errorf("unexpected argument %q for %s", arg, commandID)
}

// localizedFlagRequiresValue returns an error for a missing flag value.
func localizedFlagRequiresValue(flag, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("--%s 需要一个值", flag)
	}
	return fmt.Errorf("--%s requires a value", flag)
}

// localizedUnknownOption returns an error for an unsupported command flag.
func localizedUnknownOption(flag, commandID, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 不支持选项 --%s", commandID, flag)
	}
	return fmt.Errorf("unknown option --%s for %s", flag, commandID)
}

// localizedMissingRequiredFlag returns an error for a missing required flag.
func localizedMissingRequiredFlag(commandID, flag, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 需要 --%s", commandID, flag)
	}
	return fmt.Errorf("%s requires --%s", commandID, flag)
}

// localizedAllowedValuesError returns an error for a value outside the allowed
// set.
func localizedAllowedValuesError(commandID, flag, allowed, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s --%s 必须是以下值之一 %s", commandID, flag, allowed)
	}
	return fmt.Errorf("%s --%s must be one of %s", commandID, flag, allowed)
}

// localizedInvalidPattern returns an error for invalid plugin parameter regex.
func localizedInvalidPattern(commandID, flag string, err error, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s --%s 的校验表达式无效: %w", commandID, flag, err)
	}
	return fmt.Errorf("%s --%s has invalid validation pattern: %w", commandID, flag, err)
}

// localizedPatternMismatch returns an error for a value that fails regex
// validation.
func localizedPatternMismatch(commandID, flag, pattern, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s --%s 不匹配 %s", commandID, flag, pattern)
	}
	return fmt.Errorf("%s --%s does not match %s", commandID, flag, pattern)
}

// loadBundles loads user-installed bundles before bundled example plugins.
func loadBundles(installedRoot string) ([]plugin.Bundle, error) {
	dirs := append(pluginDirs(installedRoot), pluginDirs(defaultPluginRoot())...)
	bundles := make([]plugin.Bundle, 0, len(dirs))
	seen := make(map[string]bool, len(dirs))

	for _, dir := range dirs {
		bundle, err := plugin.LoadBundle(dir, version.Version)
		if err != nil {
			return nil, err
		}
		// User-installed plugins are scanned first and intentionally shadow the
		// bundled examples by manifest name.
		if seen[bundle.Manifest.Name] {
			continue
		}
		seen[bundle.Manifest.Name] = true
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}

// pluginDirs lists plugin bundle directories under root.
func pluginDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return nil
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(root, entry.Name()))
		}
	}
	return dirs
}

// loadCommandResponse chooses live API execution or fixture loading.
func loadCommandResponse(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, opts globalOptions, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, debug io.Writer) (map[string]any, error) {
	if !opts.Offline {
		if opts.Timeout > 0 {
			profile.TimeoutSeconds = opts.Timeout
		}
		return executeAPICommand(bundle, command, commandArgs, parameterValues, profile, getenv, transport, debug)
	}
	if command.FixtureResponse == "" {
		return nil, fmt.Errorf("command %s has no fixture response for offline mode", command.ID)
	}

	data, err := os.ReadFile(filepath.Join(bundle.Dir, command.FixtureResponse))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse fixture response for %s: %w", command.ID, err)
	}
	return payload, nil
}

// executeAPICommand builds and sends a signed CTyun request from plugin
// metadata.
func executeAPICommand(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, debug io.Writer) (map[string]any, error) {
	operation, ok := bundle.APIs.Operations[command.Operation]
	if !ok {
		return nil, fmt.Errorf("command %s references missing operation %s", command.ID, command.Operation)
	}
	endpointURL := profile.EndpointURL
	if endpointURL == "" {
		endpointURL = bundle.Manifest.API.EndpointURL
	}
	if endpointURL == "" {
		return nil, fmt.Errorf("command %s requires plugin api.endpoint_url or profile endpoint_url for live API execution", command.ID)
	}
	creds, err := coreconfig.LoadCredentialsFromEnv(getenv)
	if err != nil {
		return nil, err
	}

	// Operation metadata is the single source of truth for translating CLI
	// arguments and flags into the CTyun request.
	bodyMap := resolveMap(operation.Body, profile, commandArgs, parameterValues, command.Parameters, len(operation.Body) > 0)
	var body []byte
	if len(bodyMap) > 0 {
		body, _ = json.Marshal(bodyMap)
	}
	query := encodeQuery(resolveMap(operation.Query, profile, commandArgs, parameterValues, command.Parameters, false))
	headers := resolveMap(operation.Headers, profile, commandArgs, parameterValues, command.Parameters, false)
	contentType := operation.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	retries := 0
	if operation.Retryable {
		retries = 2
	}
	var timeout time.Duration
	if profile.TimeoutSeconds > 0 {
		timeout = time.Duration(profile.TimeoutSeconds) * time.Second
	}
	// Only operations marked retryable in metadata get automatic retries; this
	// keeps state-changing APIs opt-in.
	return client.DoJSON(transport, client.RequestSpec{
		Method:      operation.Method,
		BaseURL:     endpointURL,
		Path:        operation.Path,
		Query:       query,
		ContentType: contentType,
		Body:        body,
		Headers:     headers,
		Credentials: creds,
		Timeout:     timeout,
		Retries:     retries,
		Debug:       debug,
	})
}

// debugWriter returns stderr only when HTTP debug logging is enabled.
func debugWriter(opts globalOptions, stderr io.Writer) io.Writer {
	if !opts.Debug {
		return nil
	}
	return stderr
}

// resolveMap expands metadata placeholders into request field values.
func resolveMap(values map[string]string, profile coreconfig.Profile, commandArgs, parameterValues map[string]string, parameters []plugin.Parameter, includeParameterTargets bool) map[string]string {
	resolved := make(map[string]string, len(values)+len(parameterValues))
	for key, value := range values {
		switch value {
		case "$profile.region":
			resolved[key] = profile.Region
		default:
			if strings.HasPrefix(value, "$arg.") {
				resolved[key] = commandArgs[strings.TrimPrefix(value, "$arg.")]
			} else if strings.HasPrefix(value, "$param.") {
				if parameterValue := parameterValues[strings.TrimPrefix(value, "$param.")]; parameterValue != "" {
					resolved[key] = parameterValue
				}
			} else {
				resolved[key] = value
			}
		}
	}
	if includeParameterTargets {
		// Body maps can include parameter targets that are not explicitly listed
		// in apis.json, which keeps simple plugin flags compact.
		for _, parameter := range parameters {
			if value, ok := parameterValues[parameter.Name]; ok && value != "" {
				resolved[parameter.Target] = value
			}
		}
	}
	return resolved
}

// encodeQuery encodes non-empty request query values.
func encodeQuery(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	query := url.Values{}
	for key, value := range values {
		if value != "" {
			query.Set(key, value)
		}
	}
	return query.Encode()
}

// rowsFromPayload converts decoded JSON into stable-key table rows.
func rowsFromPayload(payload map[string]any, table plugin.Table) ([]map[string]string, error) {
	rawRows, err := valueAtPath(payload, table.RowPath)
	if err != nil {
		return nil, err
	}
	rowValues, ok := rawRows.([]any)
	if !ok {
		return nil, fmt.Errorf("row path %q is not an array", table.RowPath)
	}

	rows := make([]map[string]string, 0, len(rowValues))
	for _, rawRow := range rowValues {
		rowMap, ok := rawRow.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("row path %q contains a non-object row", table.RowPath)
		}
		row := make(map[string]string, len(table.Columns))
		for _, column := range table.Columns {
			// Missing optional paths render as empty cells; malformed row paths
			// were already rejected above.
			value, err := valueAtPath(rowMap, column.Path)
			if err == nil {
				row[column.Key] = fmt.Sprint(value)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// tableColumns localizes table column labels for rendering.
func tableColumns(table plugin.Table, language string) []output.Column {
	columns := make([]output.Column, 0, len(table.Columns))
	for _, column := range table.Columns {
		catalog := i18n.Catalog{column.Key: column.Labels}
		columns = append(columns, output.Column{
			Key:   column.Key,
			Label: catalog.Text(column.Key, language),
		})
	}
	return columns
}

// valueAtPath walks a dot-separated path through decoded JSON objects.
func valueAtPath(value any, path string) (any, error) {
	current := value
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path %q cannot read %q", path, part)
		}
		current, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("path %q is missing %q", path, part)
		}
	}
	return current, nil
}
