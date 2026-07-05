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
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

// runPluginCommand resolves a metadata-defined command and renders its result.
func runPluginCommand(stdout, stderr io.Writer, stdin io.Reader, opts globalOptions, args []string, installedRoot string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper) error {
	bundle, command, commandArgs, parameterValues, ok, err := findPluginCommand(args, installedRoot, opts.Language)
	if err != nil {
		return err
	}
	if !ok {
		return diagnostic.New("error.unknown_command", strings.Join(args, " "))
	}
	if command.Dangerous.Confirm != "" && !opts.Yes {
		message := command.Dangerous.Message
		if message == "" {
			message = command.ID
		}
		if err := confirmDangerousOperation(stderr, stdin, opts, message); err != nil {
			return err
		}
	}

	table := bundle.Tables.Tables[command.Table]
	loadResponse := func() (map[string]any, error) {
		return loadCommandResponse(bundle, command, commandArgs, parameterValues, opts, profile, getenv, transport, stderr, debugWriter(opts, stderr))
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
		return renderWaiter(stderr, bundle, opts.Waiter, payload, loadResponse, opts.Language)
	case "table":
		rows, err := rowsFromPayload(payload, table)
		if err != nil {
			return err
		}
		// Fixture output is filtered with the same stable table keys that live
		// requests use for parameterized API calls.
		rows = filterRowsByParameters(rows, table, command.Parameters, parameterValues)
		columns := tableColumns(table, opts.Language)
		opts.Filter, err = output.ResolveFilterExpression(columns, opts.Filter)
		if err != nil {
			return err
		}
		opts.Sort, err = output.ResolveSortExpression(columns, opts.Sort)
		if err != nil {
			return err
		}
		rows, _ = output.FilterRows(rows, opts.Filter)
		rows, _ = output.SortRows(rows, opts.Sort)
		selectedColumns := opts.Columns
		if len(selectedColumns) == 0 {
			selectedColumns = table.DefaultColumns
		}
		rendered, err := output.RenderTable(rows, columns, output.TableOptions{
			Columns:    selectedColumns,
			NoHeader:   opts.NoHeader,
			Style:      opts.Table,
			Vertical:   table.Layout == "vertical",
			FieldLabel: commonText("table.field", opts.Language),
			ValueLabel: commonText("table.value", opts.Language),
		})
		if err != nil {
			return err
		}
		if _, err = io.WriteString(stdout, rendered); err != nil {
			return err
		}
		return renderWaiter(stdout, bundle, opts.Waiter, payload, loadResponse, opts.Language)
	default:
		return diagnostic.New("error.unsupported_output", opts.Output)
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

// renderWaiter evaluates optional waiter metadata and writes the final state.
func renderWaiter(stdout io.Writer, bundle plugin.Bundle, waiterID string, payload map[string]any, loadResponse func() (map[string]any, error), language string) error {
	if waiterID == "" {
		return nil
	}
	spec, ok := bundle.Waiters.Waiters[waiterID]
	if !ok {
		return diagnostic.New("error.unknown_waiter", waiterID)
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
	return writeLine(stdout, waiterStatusMessage(language, waiterID, string(state)))
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
	for _, bundle := range bundles {
		command, missing, ok := plugin.FindCommandMissingPathArgs(bundle, args)
		if ok {
			return plugin.Bundle{}, plugin.Command{}, nil, nil, false, diagnostic.New("error.missing_path_argument", command.ID, strings.Join(missing, ","))
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
	if err := validateConditionalParameterValues(command, values, language); err != nil {
		return nil, err
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

// localizedUnexpectedArgument returns an error for extra command arguments.
func localizedUnexpectedArgument(arg, _ string, language string) error {
	return fmt.Errorf(messageText("error.unexpected_argument", language), arg)
}

// localizedFlagRequiresValue returns an error for a missing flag value.
func localizedFlagRequiresValue(flag, language string) error {
	return fmt.Errorf(messageText("error.flag_requires_value", language), flag)
}

// localizedUnknownOption returns an error for an unsupported command flag.
func localizedUnknownOption(flag, _ string, language string) error {
	return fmt.Errorf(messageText("error.unknown_option", language), flag)
}

// localizedMissingRequiredFlag returns an error for a missing required flag.
func localizedMissingRequiredFlag(_ string, flag, language string) error {
	return fmt.Errorf(messageText("error.missing_required_flag", language), flag)
}

// localizedMissingConditionalFlag returns an error for a missing conditional
// parameter.
func localizedMissingConditionalFlag(_ string, flag, conditionFlag, conditionValue, language string) error {
	return fmt.Errorf(messageText("error.missing_conditional_flag", language), flag, conditionFlag, conditionValue)
}

// localizedMissingConditionalAny returns an error for a missing conditional
// parameter group.
func localizedMissingConditionalAny(_ string, flags, conditionFlag, conditionValue, language string) error {
	return fmt.Errorf(messageText("error.missing_conditional_any", language), flags, conditionFlag, conditionValue)
}

// localizedAllowedValuesError returns an error for a value outside the allowed
// set.
func localizedAllowedValuesError(_ string, flag, allowed, language string) error {
	return fmt.Errorf(messageText("error.allowed_values", language), flag, allowed)
}

// localizedInvalidPattern returns an error for invalid plugin parameter regex.
func localizedInvalidPattern(_ string, flag string, err error, language string) error {
	return fmt.Errorf(messageText("error.invalid_pattern", language), flag, err)
}

// localizedPatternMismatch returns an error for a value that fails regex
// validation.
func localizedPatternMismatch(_ string, flag, pattern, language string) error {
	return fmt.Errorf(messageText("error.pattern_mismatch", language), flag, pattern)
}

// validateConditionalParameterValues applies command-level conditional
// requirement metadata after normal flag parsing.
func validateConditionalParameterValues(command plugin.Command, values map[string]string, language string) error {
	byName := make(map[string]plugin.Parameter, len(command.Parameters))
	for _, parameter := range command.Parameters {
		byName[parameter.Name] = parameter
	}
	for _, requirement := range command.ConditionalRequirements {
		conditionValue := values[requirement.When.Parameter]
		if !parameterConditionMatches(requirement.When, conditionValue) {
			continue
		}
		conditionFlag := byName[requirement.When.Parameter].Flag
		for _, name := range requirement.Required {
			parameter := byName[name]
			if values[name] == "" {
				return localizedMissingConditionalFlag(command.ID, parameter.Flag, conditionFlag, conditionValue, language)
			}
		}
		if len(requirement.AnyOf) == 0 {
			continue
		}
		flags := make([]string, 0, len(requirement.AnyOf))
		satisfied := false
		for _, name := range requirement.AnyOf {
			parameter := byName[name]
			flags = append(flags, "--"+parameter.Flag)
			if values[name] != "" {
				satisfied = true
			}
		}
		if !satisfied {
			return localizedMissingConditionalAny(command.ID, strings.Join(flags, ", "), conditionFlag, conditionValue, language)
		}
	}
	return nil
}

// parameterConditionMatches reports whether a parsed value activates a rule.
func parameterConditionMatches(condition plugin.ParameterCondition, value string) bool {
	if value == "" {
		return false
	}
	if condition.Equals != "" {
		return value == condition.Equals
	}
	return slices.Contains(condition.In, value)
}

// loadBundles loads user-installed bundles and, for development builds, bundled
// repo plugins.
func loadBundles(installedRoot string) ([]plugin.Bundle, error) {
	dirs := pluginDirs(installedRoot)
	if version.IsDevelopmentBuild() {
		dirs = append(dirs, pluginDirs(defaultPluginRoot())...)
	}
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
func loadCommandResponse(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, opts globalOptions, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, stderr, debug io.Writer) (map[string]any, error) {
	if !opts.Offline {
		if opts.Timeout > 0 {
			profile.TimeoutSeconds = opts.Timeout
		}
		return executeAPICommand(bundle, command, commandArgs, parameterValues, profile, getenv, transport, stderr, debug, opts.Language)
	}
	if command.FixtureResponse == "" {
		return nil, diagnostic.New("error.command_missing_fixture_response", command.ID)
	}

	data, err := os.ReadFile(filepath.Join(bundle.Dir, command.FixtureResponse))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, diagnostic.Wrap("error.parse_fixture_response", err, command.ID)
	}
	return payload, nil
}

// executeAPICommand builds and sends a signed CTyun request from plugin
// metadata.
func executeAPICommand(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, stderr, debug io.Writer, language string) (map[string]any, error) {
	operation, ok := bundle.APIs.Operations[command.Operation]
	if !ok {
		return nil, diagnostic.New("error.command_missing_operation_ref", command.ID, command.Operation)
	}
	endpointURL := profile.EndpointURL
	if endpointURL == "" {
		endpointURL = bundle.Manifest.API.EndpointURL
	}
	if endpointURL == "" {
		return nil, diagnostic.New("error.command_missing_live_endpoint", command.ID)
	}
	creds, err := coreconfig.ResolveCredentials(getenv, profile)
	if err != nil {
		return nil, err
	}
	if err := warnConfigCredentials(stderr, creds, getenv, profile, language); err != nil {
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
		Method:           operation.Method,
		BaseURL:          endpointURL,
		Path:             operation.Path,
		Query:            query,
		ContentType:      contentType,
		Body:             body,
		Headers:          headers,
		Credentials:      creds,
		Timeout:          timeout,
		Retries:          retries,
		Debug:            debug,
		Language:         language,
		AcceptedStatuses: acceptedStatusRules(operation.AcceptedStatuses),
	})
}

// acceptedStatusRules converts plugin metadata into client request rules.
func acceptedStatusRules(rules []plugin.AcceptedStatusRule) []client.AcceptedStatusRule {
	converted := make([]client.AcceptedStatusRule, 0, len(rules))
	for _, rule := range rules {
		converted = append(converted, client.AcceptedStatusRule{
			Code:         rule.Code,
			RequiredPath: rule.RequiredPath,
		})
	}
	return converted
}

// warnConfigCredentials writes the localized runtime warning for config-backed
// AK/SK values when warning output is enabled.
func warnConfigCredentials(stderr io.Writer, creds coreconfig.Credentials, getenv func(string) string, profile coreconfig.Profile, language string) error {
	if stderr == nil || !creds.UsesConfig() || !coreconfig.ShouldWarnConfigCredentials(getenv, profile) {
		return nil
	}
	return writeLine(stderr, localizedConfigCredentialWarning(language))
}

// localizedConfigCredentialWarning returns the config credential warning text.
func localizedConfigCredentialWarning(language string) string {
	return messageText("warning.config_credentials", language)
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
		if rowMap, ok := rawRows.(map[string]any); ok {
			rowValues = []any{rowMap}
		} else {
			return nil, diagnostic.New("error.row_path_not_array", table.RowPath)
		}
	}

	rows := make([]map[string]string, 0, len(rowValues))
	for _, rawRow := range rowValues {
		rowMap, ok := rawRow.(map[string]any)
		if !ok {
			return nil, diagnostic.New("error.row_path_non_object", table.RowPath)
		}
		row := make(map[string]string, len(table.Columns))
		for _, column := range table.Columns {
			// Missing optional paths render as empty cells; malformed row paths
			// were already rejected above.
			value, err := valueAtPath(rowMap, column.Path)
			if err == nil {
				row[column.Key] = formatTableCell(value)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// formatTableCell converts decoded JSON values into readable table cells.
func formatTableCell(value any) string {
	return formatTableCellValue(value, false)
}

// formatTableCellValue formats nested JSON values with stable object ordering.
func formatTableCellValue(value any, nested bool) string {
	switch typed := value.(type) {
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, formatTableCellValue(item, true))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		parts := sortedMapCellParts(typed)
		if len(parts) == 0 {
			return "{}"
		}
		if nested {
			return "{" + strings.Join(parts, "; ") + "}"
		}
		return strings.Join(parts, "; ")
	}
	return fmt.Sprint(value)
}

// sortedMapCellParts returns stable key=value fragments for a JSON object cell.
func sortedMapCellParts(value map[string]any) []string {
	if len(value) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+formatTableCellValue(value[key], true))
	}
	return parts
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

// valueAtPath walks a dot-separated path through decoded JSON objects, and
// projects object paths through arrays so table columns can target leaf values.
func valueAtPath(value any, path string) (any, error) {
	return valueAtPathParts(value, strings.Split(path, "."), path)
}

// valueAtPathParts recursively reads path parts, flattening projected arrays.
func valueAtPathParts(value any, parts []string, fullPath string) (any, error) {
	if len(parts) == 0 {
		return value, nil
	}
	switch typed := value.(type) {
	case map[string]any:
		next, ok := typed[parts[0]]
		if !ok {
			return nil, diagnostic.New("error.path_missing", fullPath, parts[0])
		}
		return valueAtPathParts(next, parts[1:], fullPath)
	case []any:
		projected := make([]any, 0, len(typed))
		for _, item := range typed {
			next, err := valueAtPathParts(item, parts, fullPath)
			if err != nil {
				return nil, err
			}
			projected = appendProjectedValue(projected, next)
		}
		return projected, nil
	default:
		return nil, diagnostic.New("error.path_cannot_read", fullPath, parts[0])
	}
}

// appendProjectedValue appends value, flattening arrays from nested projection.
func appendProjectedValue(values []any, value any) []any {
	if nested, ok := value.([]any); ok {
		return append(values, nested...)
	}
	return append(values, value)
}
