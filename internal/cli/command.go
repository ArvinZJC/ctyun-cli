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
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// developmentBundledPluginsEnabled reports whether command discovery should
// include source-tree bundled plugins for development builds.
var developmentBundledPluginsEnabled = version.IsDevelopmentBuild

// runPluginCommand resolves a metadata-defined command and renders its result.
func runPluginCommand(stdout, stderr io.Writer, stdin io.Reader, opts globalOptions, args []string, installedRoot string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper) error {
	bundle, command, commandArgs, parameterValues, ok, err := findPluginCommand(args, installedRoot, opts.Language)
	if err != nil {
		return err
	}
	if !ok {
		return runHelp(stdout, args, installedRoot, opts.Language)
	}
	if parameterValues[fixtureModeParameter] != "" {
		opts.Fixture = true
		delete(parameterValues, fixtureModeParameter)
	}
	if err := validateTransportOutput(bundle, command, parameterValues, opts); err != nil {
		return err
	}
	selectorValue := ""
	if opts.Waiter != "" {
		spec, exists := bundle.Waiters.Waiters[opts.Waiter]
		if !exists {
			return diagnostic.New("error.unknown_waiter", opts.Waiter)
		}
		if !plugin.WaiterApplies(bundle, command, spec) {
			return diagnostic.New("error.waiter_not_applicable", opts.Waiter, commandDisplayPath(command))
		}
		if spec.Selector != nil {
			selectorValue, err = resolveWaiterSelection(command, spec, commandArgs, parameterValues)
			if err != nil {
				return err
			}
		}
	}
	if command.Dangerous.Confirm != "" && !opts.Yes {
		message := command.Dangerous.Message
		if message == "" {
			message = commandDisplayPath(command)
		}
		if err := confirmDangerousOperation(stderr, stdin, opts, message); err != nil {
			return err
		}
	}
	if err := warnDeprecatedCommandUsage(stderr, bundle, command, parameterValues, getenv, profile, opts.Language); err != nil {
		return err
	}

	if plugin.UsesTransport(bundle.APIs.Operations[command.Operation]) {
		return runTransportCommand(stdout, stderr, bundle, command, commandArgs, parameterValues, opts, profile, getenv, transport, debugWriter(opts, stderr), selectorValue)
	}
	loadResponse := func() (map[string]any, error) {
		return loadCommandResponse(bundle, command, commandArgs, parameterValues, opts, profile, getenv, transport, stderr, debugWriter(opts, stderr))
	}
	// Keep loading separate from rendering so waiters can poll the same command
	// without duplicating metadata resolution.
	payload, err := loadResponse()
	if err != nil {
		return err
	}

	return renderCommandPayload(stdout, stderr, bundle, command, parameterValues, opts, payload, loadResponse, selectorValue, profile, getenv)
}

// renderCommandPayload applies the shared table and JSON output controls.
func renderCommandPayload(stdout, stderr io.Writer, bundle plugin.Bundle, command plugin.Command, parameterValues map[string]string, opts globalOptions, payload map[string]any, loadResponse func() (map[string]any, error), selectorValue string, profile coreconfig.Profile, getenv func(string) string) error {
	table := bundle.Tables.Tables[command.Table]
	var err error

	switch opts.Output {
	case "json":
		rendered, _ := output.RenderJSON(payload)
		if _, err = io.WriteString(stdout, rendered); err != nil {
			return err
		}
		return renderWaiter(stderr, bundle, opts.Waiter, payload, loadResponse, opts.Language, selectorValue)
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
		if err := warnDeprecatedDisplayedColumns(stderr, table, columns, selectedColumns, getenv, profile, opts.Language); err != nil {
			return err
		}
		rendered, err := renderTableOutput(stdout, rows, columns, output.TableOptions{
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
		return renderWaiter(stdout, bundle, opts.Waiter, payload, loadResponse, opts.Language, selectorValue)
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

// findPluginCommand matches command arguments to a plugin command and parses
// command-specific flags.
func findPluginCommand(args []string, installedRoot, language string) (plugin.Bundle, plugin.Command, map[string]string, map[string]string, bool, error) {
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return plugin.Bundle{}, plugin.Command{}, nil, nil, false, err
	}
	for _, bundle := range bundles {
		optionalCommand, optionalCommandArgs, optionalRest, ok := findOptionalRegionCommand(bundle, args)
		if ok {
			parameterValues, err := parseCommandParameters(optionalCommand, optionalRest, language)
			if err != nil {
				return plugin.Bundle{}, plugin.Command{}, nil, nil, false, err
			}
			return bundle, optionalCommand, optionalCommandArgs, parameterValues, true, nil
		}
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
		_, missing, ok := plugin.FindCommandMissingPathArgs(bundle, args)
		if ok {
			return plugin.Bundle{}, plugin.Command{}, nil, nil, false, diagnostic.New("error.missing_path_argument", strings.Join(missing, ","))
		}
	}
	return plugin.Bundle{}, plugin.Command{}, nil, nil, false, nil
}

// findOptionalRegionCommand matches commands whose trailing region_id argument
// can come from a profile instead of the positional path.
func findOptionalRegionCommand(bundle plugin.Bundle, args []string) (plugin.Command, map[string]string, []string, bool) {
	for _, command := range bundle.Commands.Commands {
		if len(command.Path) == 0 || !isRegionPathPlaceholder(command.Path[len(command.Path)-1]) {
			continue
		}
		operation, ok := bundle.APIs.Operations[command.Operation]
		if !ok || len(argRegionTargets(operation)) == 0 {
			continue
		}
		prefix := command.Path[:len(command.Path)-1]
		commandArgs, rest, ok := pluginPathPrefix(prefix, args)
		if !ok || (len(rest) > 0 && !strings.HasPrefix(rest[0], "--")) {
			continue
		}
		return command, commandArgs, rest, true
	}
	return plugin.Command{}, nil, nil, false
}

// pluginPathPrefix matches a command path prefix and returns remaining tokens.
func pluginPathPrefix(pattern, args []string) (map[string]string, []string, bool) {
	if len(pattern) > len(args) {
		return nil, nil, false
	}
	commandArgs := make(map[string]string)
	for index, segment := range pattern {
		if isPathPlaceholder(segment) {
			commandArgs[commandPathPlaceholderName(segment)] = args[index]
			continue
		}
		if segment != args[index] {
			return nil, nil, false
		}
	}
	return commandArgs, args[len(pattern):], true
}

// isRegionPathPlaceholder reports whether segment is the region_id argument.
func isRegionPathPlaceholder(segment string) bool {
	return isPathPlaceholder(segment) && commandPathPlaceholderName(segment) == "region_id"
}

// commandPathPlaceholderName returns the placeholder name without braces.
func commandPathPlaceholderName(segment string) string {
	return strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
}

// commandDisplayPath returns a visible command path for prompts and diagnostics.
func commandDisplayPath(command plugin.Command) string {
	if len(command.Path) == 0 {
		return "plugin command"
	}
	return strings.Join(command.Path, " ")
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
		handled := false
		for _, option := range productTransferOptions(command) {
			name, inline, hasInline := strings.Cut(arg, "=")
			if name != "--"+option.Name {
				continue
			}
			value := "true"
			if option.TakesValue {
				value = inline
				if !hasInline {
					i++
					if i >= len(args) {
						return nil, diagnostic.New("error.option_requires_value", name)
					}
					value = args[i]
				}
				if value == "" {
					return nil, diagnostic.New("error.option_requires_value", name)
				}
			} else if hasInline {
				return nil, diagnostic.New("error.unknown_option", arg)
			}
			values[transferParameter(option.Name)] = value
			handled = true
			break
		}
		if handled {
			continue
		}

		if arg == "--offline" || arg == "--fixture" {
			if !version.IsDevelopmentBuild() {
				return nil, diagnostic.New("error.unknown_option", arg)
			}
			values[fixtureModeParameter] = arg
			continue
		}
		if !strings.HasPrefix(arg, "--") {
			if strings.HasPrefix(arg, "-") {
				return nil, diagnostic.New("error.unknown_option", arg)
			}
			return nil, diagnostic.New("error.unexpected_argument", arg)
		}
		flag := strings.TrimPrefix(arg, "--")
		value := ""
		if name, inline, ok := strings.Cut(flag, "="); ok {
			flag = name
			value = inline
		} else {
			if _, ok := byFlag[flag]; !ok {
				return nil, diagnostic.New("error.unknown_option", "--"+flag)
			}
			i++
			if i >= len(args) {
				return nil, diagnostic.New("error.option_requires_value", "--"+flag)
			}
			value = args[i]
		}
		parameter, ok := byFlag[flag]
		if !ok {
			return nil, diagnostic.New("error.unknown_option", "--"+flag)
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

const fixtureModeParameter = "__ctyun_fixture_mode"

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
	if _, err := plugin.ParseParameterValue(parameter, value); err != nil {
		return localizedInvalidOptionValueType(parameter, value, language)
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
		conditionValue := plugin.ParameterValueOrDefault(requirement.When.Parameter, command.Parameters, values)
		if !plugin.ParameterConditionMatches(requirement.When, command.Parameters, values) {
			continue
		}
		conditionFlag := byName[requirement.When.Parameter].Flag
		for _, name := range requirement.Required {
			parameter := byName[name]
			if values[name] == "" {
				if requirement.When.Always {
					return fmt.Errorf(messageText("error.missing_unconditional_flag", language), parameter.Flag)
				}
				if len(requirement.When.All) > 0 {
					return fmt.Errorf(messageText("error.missing_composite_flag", language), parameter.Flag, parameterConditionDescription(command, requirement.When, language, false))
				}
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
			if requirement.When.Always {
				return fmt.Errorf(messageText("error.missing_unconditional_any", language), strings.Join(flags, ", "))
			}
			if len(requirement.When.All) > 0 {
				return fmt.Errorf(messageText("error.missing_composite_any", language), strings.Join(flags, ", "), parameterConditionDescription(command, requirement.When, language, false))
			}
			return localizedMissingConditionalAny(command.ID, strings.Join(flags, ", "), conditionFlag, conditionValue, language)
		}
	}
	return nil
}

// loadBundles loads user-installed bundles and, for development builds, bundled
// repo plugins.
func loadBundles(installedRoot string) ([]plugin.Bundle, error) {
	dirs := pluginDirs(installedRoot)
	if developmentBundledPluginsEnabled() {
		dirs = append(pluginDirs(defaultPluginRoot()), dirs...)
	}
	bundles := make([]plugin.Bundle, 0, len(dirs))
	seen := make(map[string]bool, len(dirs))

	for _, dir := range dirs {
		bundle, err := plugin.LoadBundle(dir, version.Version)
		if err != nil {
			return nil, err
		}
		// Development builds scan bundled plugins first so worktree metadata
		// remains visible even when released plugins are installed.
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
	if !opts.Fixture {
		if opts.Timeout > 0 {
			profile.TimeoutSeconds = opts.Timeout
		}
		return executeAPICommand(bundle, command, commandArgs, parameterValues, profile, getenv, transport, stderr, debug, opts.Language)
	}
	if command.FixtureResponse == "" {
		return nil, diagnostic.New("error.command_missing_fixture_response")
	}

	data, err := os.ReadFile(filepath.Join(bundle.Dir, command.FixtureResponse))
	if err != nil {
		return nil, err
	}
	operation := bundle.APIs.Operations[command.Operation]
	if operation.Response != nil {
		// DecodeHTTPResponse takes ownership and closes the response on every path.
		//goland:noinspection GoResourceLeak
		response, err := client.DecodeFixture(data)
		if err != nil {
			return nil, err
		}
		result, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: operation.Method, Response: operation.Response})
		if err != nil {
			return nil, err
		}
		return result.Payload, nil
	}
	payload, err := client.DecodeResponse(data)
	if err != nil {
		return nil, diagnostic.Wrap("error.parse_fixture_response", err)
	}
	return payload, nil
}

// executeAPICommand builds and sends a signed CTyun request from plugin
// metadata.
func executeAPICommand(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, stderr, debug io.Writer, language string) (map[string]any, error) {
	spec, err := buildAPIRequest(bundle, command, commandArgs, parameterValues, profile, getenv, stderr, debug, language)
	if err != nil {
		return nil, err
	}
	if spec.PreparedBody != nil {
		// Snapshot removal is best-effort after the request has completed.
		defer func() { _ = spec.PreparedBody.Close() }()
	}
	return client.DoJSON(transport, spec)
}

// buildAPIRequest resolves credentials, routing, and metadata for both response paths.
func buildAPIRequest(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, profile coreconfig.Profile, getenv func(string) string, stderr, debug io.Writer, language string) (client.RequestSpec, error) {
	operation, ok := bundle.APIs.Operations[command.Operation]
	if !ok {
		return client.RequestSpec{}, diagnostic.New("error.command_missing_operation_ref")
	}
	path, err := resolveRequestPath(operation.Path, commandArgs)
	if err != nil {
		return client.RequestSpec{}, err
	}
	endpointURL := profile.EndpointURL
	if endpointURL == "" {
		endpointURL = bundle.Manifest.API.EndpointURL
	}
	if endpointURL == "" && operation.Native == nil {
		return client.RequestSpec{}, diagnostic.New("error.command_missing_live_endpoint")
	}
	if profile.Region == "" && operationMissingProfileRegion(operation, commandArgs, command.Parameters, parameterValues) {
		return client.RequestSpec{}, diagnostic.New("error.missing_profile_region")
	}
	var creds coreconfig.Credentials
	var native *client.NativeRequest
	if operation.Native != nil {
		native, endpointURL, creds, err = nativeRequest(operation.Native, commandArgs, parameterValues, getenv)
	} else {
		creds, err = coreconfig.ResolveCredentials(getenv, profile)
	}
	if err != nil {
		return client.RequestSpec{}, err
	}
	if err := warnConfigCredentials(stderr, creds, getenv, profile, language); err != nil {
		return client.RequestSpec{}, err
	}

	// Operation metadata is the single source of truth for translating CLI
	// arguments and flags into the CTyun request.
	var bodyMap map[string]any
	if operation.Request != nil {
		bodyMap, err = resolveExplicitBody(operation.Body, profile, commandArgs, parameterValues, command.Parameters, language)
	} else {
		bodyMap, err = resolveRequestBody(operation.Body, profile, commandArgs, parameterValues, command.Parameters, language)
	}
	if err != nil {
		return client.RequestSpec{}, err
	}
	var body []byte
	if len(bodyMap) > 0 {
		body, _ = json.Marshal(bodyMap)
	}
	queryMap, err := resolveQueryMap(operation.Query, profile, commandArgs, parameterValues, command.Parameters, language)
	if err != nil {
		return client.RequestSpec{}, err
	}
	query := encodeQuery(queryMap)
	if operation.Native != nil {
		for _, key := range operation.Native.Subresources {
			if query != "" {
				query += "&"
			}
			query += url.QueryEscape(key) + "="
		}
	}

	headers := resolveMap(operation.Headers, profile, commandArgs, parameterValues, command.Parameters, false)
	if operation.Native != nil {
		nativeType, err := nativeRequestHeaders(operation.Native, parameterValues, headers)
		if err != nil {
			return client.RequestSpec{}, err
		}
		if nativeType != "" {
			operation.ContentType = nativeType
		}
	}
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
	spec := client.RequestSpec{
		Method:           operation.Method,
		BaseURL:          endpointURL,
		Path:             path,
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
	}
	spec.Native = native
	spec.Response = operation.Response
	if operation.Request != nil {
		parameterValues, err = resolvePOSTPolicy(operation.Request, parameterValues, getenv)
		if err != nil {
			return client.RequestSpec{}, err
		}
		prepared, err := prepareCommandBody(operation, command, commandArgs, parameterValues, profile, bodyMap)
		if err != nil {
			return client.RequestSpec{}, err
		}
		spec.Body = nil
		spec.PreparedBody = prepared
	}
	return spec, nil
}

// operationMissingProfileRegion reports whether an operation needs a profile
// region and no command option overrides that request field.
func operationMissingProfileRegion(operation plugin.Operation, commandArgs map[string]string, parameters []plugin.Parameter, parameterValues map[string]string) bool {
	for _, target := range profileRegionTargets(operation) {
		if !parameterTargetHasValue(parameters, parameterValues, target) {
			return true
		}
	}
	for range argRegionTargets(operation) {
		if commandArgs["region_id"] == "" {
			return true
		}
	}
	return false
}

// profileRegionTargets returns request field names sourced from profile region.
func profileRegionTargets(operation plugin.Operation) []string {
	return operationTargetsBySource(operation, "$profile.region")
}

// argRegionTargets returns request field names sourced from region_id argument.
func argRegionTargets(operation plugin.Operation) []string {
	return operationTargetsBySource(operation, "$arg.region_id")
}

// operationTargetsBySource returns request field names using source.
func operationTargetsBySource(operation plugin.Operation, source string) []string {
	var targets []string
	for _, values := range []map[string]string{operation.Body, operation.Query, operation.Headers} {
		for target, value := range values {
			if value == source {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

// parameterTargetHasValue reports whether command option values can fill target.
func parameterTargetHasValue(parameters []plugin.Parameter, parameterValues map[string]string, target string) bool {
	for _, parameter := range parameters {
		if parameter.Target == target && parameterValues[parameter.Name] != "" {
			return true
		}
	}
	return false
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

// warnDeprecatedCommandUsage writes command, API, and used-option deprecation
// warnings when warning output is enabled.
func warnDeprecatedCommandUsage(stderr io.Writer, bundle plugin.Bundle, command plugin.Command, parameterValues map[string]string, getenv func(string) string, profile coreconfig.Profile, language string) error {
	if stderr == nil || !coreconfig.ShouldWarnDeprecated(getenv, profile) {
		return nil
	}
	if command.Deprecation.Active() {
		if err := writeLine(stderr, localizedDeprecationWarning("warning.deprecated_command", command.Deprecation, language)); err != nil {
			return err
		}
	}
	if operation, ok := bundle.APIs.Operations[command.Operation]; ok && operation.Deprecation.Active() {
		if err := writeLine(stderr, localizedDeprecationWarning("warning.deprecated_api", operation.Deprecation, language)); err != nil {
			return err
		}
	}
	for _, parameter := range command.Parameters {
		if parameterValues[parameter.Name] == "" || !parameter.Deprecation.Active() {
			continue
		}
		message := messagef("warning.deprecated_option", language, parameter.Flag, localizedDeprecationDetails(parameter.Deprecation, language))
		if err := writeLine(stderr, message); err != nil {
			return err
		}
	}
	return nil
}

// warnDeprecatedDisplayedColumns writes response-field deprecation warnings for
// table columns selected for rendering.
func warnDeprecatedDisplayedColumns(stderr io.Writer, table plugin.Table, columns []output.Column, requested []string, getenv func(string) string, profile coreconfig.Profile, language string) error {
	if stderr == nil || !coreconfig.ShouldWarnDeprecated(getenv, profile) {
		return nil
	}
	selected, err := output.ResolveColumnSelectors(columns, requested)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		selected = make([]string, 0, len(table.Columns))
		for _, column := range table.Columns {
			selected = append(selected, column.Key)
		}
	}
	selectedSet := make(map[string]bool, len(selected))
	for _, key := range selected {
		selectedSet[key] = true
	}
	labels := make(map[string]string, len(columns))
	for _, column := range columns {
		labels[column.Key] = column.Label
	}
	for _, column := range table.Columns {
		if !selectedSet[column.Key] || !column.Deprecation.Active() {
			continue
		}
		message := messagef("warning.deprecated_field", language, labels[column.Key], localizedDeprecationDetails(column.Deprecation, language))
		if err := writeLine(stderr, message); err != nil {
			return err
		}
	}
	return nil
}

// localizedDeprecationWarning returns a generic deprecation warning line.
func localizedDeprecationWarning(key string, deprecation *plugin.Deprecation, language string) string {
	return messagef(key, language, localizedDeprecationDetails(deprecation, language))
}

// localizedDeprecationDetails formats optional CLI-facing replacement guidance.
func localizedDeprecationDetails(deprecation *plugin.Deprecation, language string) string {
	if deprecation == nil {
		return ""
	}
	var builder strings.Builder
	if deprecation.Replacement != nil && cliReplacementKind(deprecation.Replacement.Kind) && deprecation.Replacement.Label != "" {
		builder.WriteString(messagef("warning.deprecated_replacement", language, deprecation.Replacement.Label))
	}
	return builder.String()
}

// cliReplacementKind reports whether replacement metadata is safe to present as
// a CLI recommendation instead of raw upstream API guidance.
func cliReplacementKind(kind string) bool {
	return kind == "command" || kind == "option"
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
			if argName, ok := strings.CutPrefix(value, "$arg."); ok {
				resolved[key] = commandArgs[argName]
				if argName == "region_id" && resolved[key] == "" {
					resolved[key] = profile.Region
				}
			} else if parameterName, ok := strings.CutPrefix(value, "$param."); ok {
				if parameterValue := parameterValues[parameterName]; parameterValue != "" {
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
			if value, ok := parameterValues[parameter.Name]; ok && value != "" && parameter.Target != "" {
				resolved[parameter.Target] = value
			}
		}
	}
	if !includeParameterTargets {
		for _, parameter := range parameters {
			if value, ok := parameterValues[parameter.Name]; ok && value != "" && parameter.Target != "" {
				if _, exists := resolved[parameter.Target]; exists {
					resolved[parameter.Target] = value
				}
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
