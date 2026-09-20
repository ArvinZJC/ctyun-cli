/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// productTransferOptions is the shared declaration for download-only CLI options.
func productTransferOptions(command plugin.Command) []commandOption {
	if !command.Download {
		return nil
	}
	return []commandOption{{Name: "output-file", TakesValue: true}, {Name: "overwrite"}}
}

// transferParameter maps command-owned options to private parsed-value keys.
func transferParameter(name string) string { return "__ctyun_" + strings.ReplaceAll(name, "-", "_") }

// validateTransportOutput rejects incompatible controls before IO or confirmation.
func validateTransportOutput(bundle plugin.Bundle, command plugin.Command, values map[string]string, opts globalOptions) error {
	operation := bundle.APIs.Operations[command.Operation]
	destination := values[transferParameter("output-file")]
	if values[transferParameter("overwrite")] != "" && destination == "" {
		return apicontract.Invalid("overwrite")
	}
	if opts.Output == "raw" && operation.Response == nil {
		return diagnostic.New("error.unsupported_output", opts.Output)
	}
	binary := false
	if operation.Response != nil {
		for _, variant := range operation.Response.Variants {
			binary = binary || variant.Format == "binary"
		}
	}
	if binary && destination == "" && opts.Output != "raw" {
		return diagnostic.New("error.download_destination")
	}
	if destination != "" || opts.Output == "raw" {
		for _, name := range []string{"cols", "filter", "sort", "no-header", "table", "wait"} {
			if _, ok := opts.Seen[name]; ok {
				return apicontract.Invalid(name)
			}
		}
		if opts.Waiter != "" {
			return apicontract.Invalid("wait")
		}
		if destination != "" {
			if _, explicit := opts.Seen["output"]; explicit {
				return apicontract.Invalid("output")
			}
		}
	}
	return nil
}

// prepareCommandBody resolves explicit sources after ordinary argument validation.
func prepareCommandBody(operation plugin.Operation, command plugin.Command, args, values map[string]string, profile coreconfig.Profile, fields map[string]any) (*client.PreparedBody, error) {
	request := operation.Request
	input := client.BodyInput{Encoding: request.Encoding, JSONFields: request.JSONFields, ContentType: operation.ContentType, Fields: fields}
	resolve := func(source string) string {
		if name, ok := strings.CutPrefix(source, "$param."); ok {
			return values[name]
		}
		if name, ok := strings.CutPrefix(source, "$arg."); ok {
			return args[name]
		}
		if source == "$profile.region" {
			return profile.Region
		}
		return ""
	}
	if name, parameter := strings.CutPrefix(request.Document, "$param."); parameter {
		if _, present := values[name]; !present {
			return nil, nil
		}
	}
	input.Document = resolve(request.Document)
	if request.Encoding == "xml" {
		for _, parameter := range command.Parameters {
			if request.Document == "$param."+parameter.Name && parameter.Input == "file" {
				file, err := client.OpenRegularFile(input.Document)
				if err != nil {
					return nil, err
				}
				data, readErr := io.ReadAll(io.LimitReader(file, client.MaxStructuredBody+1))
				closeErr := file.Close()
				if readErr != nil {
					return nil, readErr
				}
				if closeErr != nil {
					return nil, closeErr
				}
				if len(data) > client.MaxStructuredBody {
					return nil, apicontract.Invalid("request.size")
				}
				input.Document = string(data)
			}
		}
	}
	for _, part := range request.Parts {
		value := resolve(part.Source)
		name, parameter := strings.CutPrefix(part.Source, "$param.")
		_, present := values[name]
		if parameter && !present {
			continue
		}
		input.Parts = append(input.Parts, client.BodyPart{Name: part.Name, Value: value, File: part.File, ContentType: part.ContentType})
	}
	return client.PrepareBody(input)
}

// runTransportCommand shares response validation between fixtures and live calls.
func runTransportCommand(stdout, stderr io.Writer, bundle plugin.Bundle, command plugin.Command, args, values map[string]string, opts globalOptions, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, debug io.Writer, selectorValue string) error {
	operation := bundle.APIs.Operations[command.Operation]
	spec := client.RequestSpec{Method: operation.Method, Response: operation.Response, AcceptedStatuses: acceptedStatusRules(operation.AcceptedStatuses), Language: opts.Language, Debug: debug}
	var response *client.HTTPResponse
	var err error
	if opts.Fixture {
		data, readErr := os.ReadFile(filepath.Join(bundle.Dir, command.FixtureResponse))
		if readErr != nil {
			return readErr
		}
		if operation.Response != nil {
			response, err = client.DecodeFixture(data)
		} else {
			response = &client.HTTPResponse{Status: 200, Headers: make(http.Header), Body: io.NopCloser(bytes.NewReader(data))}
		}
	} else {
		if opts.Timeout > 0 {
			profile.TimeoutSeconds = opts.Timeout
		}
		spec, err = buildAPIRequest(bundle, command, args, values, profile, getenv, transport, stderr, debug, opts.Language)
		if err != nil {
			return err
		}
		if spec.PreparedBody != nil {
			defer spec.PreparedBody.Close()
		}
		response, err = client.Do(transport, spec)
	}
	if err != nil {
		return err
	}
	defer response.Close()
	destination := values[transferParameter("output-file")]
	binary := false
	if operation.Response != nil {
		for _, variant := range operation.Response.Variants {
			if variant.Status == response.Status {
				binary = variant.Format == "binary"
			}
		}
	}
	if binary {
		copyBody := func(writer io.Writer) error { _, err := client.CopyHTTPResponse(writer, response, spec); return err }
		if destination != "" {
			return output.WriteFile(destination, values[transferParameter("overwrite")] != "", copyBody)
		}
		return copyBody(stdout)
	}
	result, err := client.DecodeHTTPResponse(response, spec)
	if err != nil {
		return err
	}
	if destination != "" {
		return output.WriteFile(destination, values[transferParameter("overwrite")] != "", func(writer io.Writer) error { _, err := writer.Write(result.Raw); return err })
	}
	if opts.Output == "raw" {
		_, err := stdout.Write(result.Raw)
		return err
	}
	reload := func() (map[string]any, error) {
		return loadCommandResponse(bundle, command, args, values, opts, profile, getenv, transport, stderr, debug)
	}
	return renderCommandPayload(stdout, stderr, bundle, command, values, opts, result.Payload, reload, selectorValue, profile, getenv)
}

// printProductGlobalOptions keeps help output values aligned with runtime capabilities.
func printProductGlobalOptions(writer *outputWriter, language string, args []string, operation plugin.Operation) {
	rows := globalOptionHelpRows(language, args, false)
	rows = slices.DeleteFunc(rows, func(row helpRow) bool {
		return !productGlobalOptionAllowed(operation, strings.TrimPrefix(row.SortKey, "--"))
	})
	for i := range rows {
		if rows[i].SortKey == "--output" {
			option := globalOptionHelp{Short: "-o", Long: "--output", Value: strings.Join(plugin.OutputFormats(operation), "|")}
			rows[i].Name = formatGlobalOptionNames(option)
			if operation.Response != nil {
				rows[i].Description = helpText("option.output.transport", language)
			}
		}
	}
	writer.Format("\n%s:\n", helpText("global.heading", language))
	writeAlignedHelpRows(writer, rows, "  ")
}

// resolveExplicitBody sends only declared fields and preserves present empty strings.
func resolveExplicitBody(bindings map[string]string, profile coreconfig.Profile, args, values map[string]string, parameters []plugin.Parameter, language string) (map[string]any, error) {
	fields := make(map[string]any, len(bindings))
	for key, source := range bindings {
		if name, ok := strings.CutPrefix(source, "$param."); ok {
			value, present := values[name]
			if !present {
				continue
			}
			for _, parameter := range parameters {
				if parameter.Name == name {
					typed, err := plugin.ParseParameterValue(parameter, value)
					if err != nil {
						return nil, localizedInvalidOptionValueType(parameter, value, language)
					}
					fields[key] = typed
				}
			}
		} else {
			for name, value := range resolveMap(map[string]string{key: source}, profile, args, values, nil, false) {
				fields[name] = value
			}
		}
	}
	return fields, nil
}

// productGlobalOptionAllowed keeps binary command help and completion free of table controls.
func productGlobalOptionAllowed(operation plugin.Operation, name string) bool {
	if slices.Equal(plugin.OutputFormats(operation), []string{"raw"}) {
		switch name {
		case "cols", "filter", "sort", "no-header", "table", "wait":
			return false
		}
	}
	return true
}
