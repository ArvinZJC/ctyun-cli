/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
	"strings"
)

// UsesTransport identifies operations requiring the explicit HTTP contract core.
func UsesTransport(operation Operation) bool {
	return operation.Native != nil || operation.Request != nil || operation.Response != nil || operation.Method == "HEAD" || operation.Method == "OPTIONS"
}

// validateTransport prevents mixing legacy and explicit HTTP success policies.
func validateTransport(operation Operation) error {
	if operation.Native != nil && operation.Native.PolicyAuth && (operation.Request == nil || operation.Request.PostPolicy == nil) {
		return apicontract.Invalid("native.policy_auth")
	}
	if err := apicontract.ValidateNative(operation.Native, operation.Path, operation.Response); err != nil {
		return err
	}
	if err := apicontract.Validate(operation.Method, operation.ContentType, operation.Request, operation.Response); err != nil {
		return err
	}
	if operation.Response != nil && len(operation.AcceptedStatuses) != 0 {
		return apicontract.Invalid("accepted_statuses")
	}
	if operation.Request != nil && operation.Request.Encoding != "json" && operation.Request.Encoding != "form" && len(operation.Body) != 0 {
		return apicontract.Invalid("body")
	}
	return nil
}

// validateTransportBindings checks sources and CLI file roles before file access.
func validateTransportBindings(command Command, operation Operation) error {
	parameters := map[string]Parameter{}
	for _, parameter := range command.Parameters {
		if parameter.Input != "" && parameter.Input != "value" && parameter.Input != "file" {
			return apicontract.Invalid("parameter.input")
		}
		if command.Download && (parameter.Flag == "output-file" || parameter.Flag == "overwrite") {
			return apicontract.Invalid("parameter.flag")
		}
		parameters[parameter.Name] = parameter
	}
	if request := operation.Request; request != nil {
		for _, field := range request.JSONFields {
			if _, exists := operation.Body[field]; !exists {
				return apicontract.Invalid("request.json_fields")
			}
		}
		checkFile := func(source string, wantFile bool, allowXML bool) error {
			name, isParameter := strings.CutPrefix(source, "$param.")
			parameter, exists := parameters[name]
			if wantFile && (!isParameter || !exists || parameter.Input != "file") {
				return apicontract.Invalid("request.file_source")
			}
			if !wantFile && !allowXML && isParameter && parameter.Input == "file" {
				return apicontract.Invalid("request.file_source")
			}
			return nil
		}
		if request.Document != "" {
			if err := checkFile(request.Document, request.Encoding == "file", request.Encoding == "xml"); err != nil {
				return err
			}
		}
		for _, part := range request.Parts {
			if request.PostPolicy != nil && (part.Source == request.PostPolicy.AccessKey || part.Source == request.PostPolicy.Policy || part.Source == request.PostPolicy.Signature) {
				parameter := parameters[strings.TrimPrefix(part.Source, "$param.")]
				if parameter.ValueType != "" && parameter.ValueType != ParameterValueString {
					return apicontract.Invalid("request.post_policy.source")
				}
			}
			if part.Expand {
				name, ok := strings.CutPrefix(part.Source, "$param.")
				if !ok || parameters[name].ValueType != ParameterValueStringMap {
					return apicontract.Invalid("request.parts.expand")
				}
			}
			if err := checkFile(part.Source, part.File, false); err != nil {
				return err
			}
		}
	}
	sources := []string{}
	if operation.Native != nil {
		if operation.Native.Metadata != "" {
			name, ok := strings.CutPrefix(operation.Native.Metadata, "$param.")
			if !ok || parameters[name].ValueType != ParameterValueStringMap || parameters[name].Input == "file" {
				return apicontract.Invalid("native.metadata")
			}
		}
		sources = append(sources, operation.Native.Bucket, operation.Native.Object, operation.Native.Metadata, operation.Native.ContentType)
		for _, source := range []string{operation.Native.Bucket, operation.Native.Object, operation.Native.ContentType} {
			if name, ok := strings.CutPrefix(source, "$param."); ok {
				parameter := parameters[name]
				if parameter.Input == "file" || parameter.ValueType != "" && parameter.ValueType != ParameterValueString {
					return apicontract.Invalid("native.source")
				}
			}
		}
	}
	if operation.Request != nil {
		sources = append(sources, operation.Request.Document)
		for _, part := range operation.Request.Parts {
			sources = append(sources, part.Source)
		}
	}
	for _, source := range sources {
		if source == "" {
			continue
		}
		if name, ok := strings.CutPrefix(source, "$param."); ok {
			if _, exists := parameters[name]; !exists {
				return apicontract.Invalid("request.source")
			}
			continue
		}
		if name, ok := strings.CutPrefix(source, "$arg."); ok {
			found := false
			for _, part := range command.Path {
				found = found || part == "{"+name+"}"
			}
			if !found {
				return apicontract.Invalid("request.source")
			}
			continue
		}
		if source != "$profile.region" {
			return apicontract.Invalid("request.source")
		}
	}
	for _, parameter := range command.Parameters {
		if parameter.Input != "file" {
			continue
		}
		used := false
		for _, source := range sources {
			used = used || source == "$param."+parameter.Name
		}
		for _, fields := range []map[string]string{operation.Query, operation.Body, operation.Headers} {
			for _, source := range fields {
				if source == "$param."+parameter.Name {
					return apicontract.Invalid("request.file_source")
				}
			}
		}
		if !used || parameter.ValueType != "" && parameter.ValueType != ParameterValueString {
			return apicontract.Invalid("parameter.input")
		}
	}
	if command.Download {
		if operation.Response == nil {
			return apicontract.Invalid("download")
		}
		for _, variant := range operation.Response.Variants {
			if variant.Format == "empty" {
				return apicontract.Invalid("download")
			}
		}
	}
	return nil
}

// validateTransportCore requires an explicit minimum that older loaders enforce.
func validateTransportCore(bundle Bundle, coreVersion string) error {
	uses := false
	for _, op := range bundle.APIs.Operations {
		uses = uses || UsesTransport(op)
	}
	for _, command := range bundle.Commands.Commands {
		uses = uses || command.Download
		for _, parameter := range command.Parameters {
			uses = uses || parameter.Input == "file"
		}
	}
	for _, table := range bundle.Tables.Tables {
		uses = uses || table.XML != nil
	}
	if !uses {
		return nil
	}
	floor := false
	for _, part := range strings.Fields(bundle.Manifest.Requires.Ctyun) {
		if minimum, ok := strings.CutPrefix(part, ">="); ok && coreversion.IsSemanticVersion(minimum) && coreversion.CompareSemanticVersions(minimum, "0.5.0") >= 0 {
			floor = true
		}
	}
	if !floor {
		return apicontract.Invalid("requires.ctyun")
	}
	if coreversion.CompareSemanticVersions(compatibilityVersion(coreVersion), "0.5.0") < 0 {
		return diagnostic.New("error.plugin_version", bundle.Manifest.Name, bundle.Manifest.Requires.Ctyun, coreVersion)
	}
	return nil
}

// BinaryOnly identifies operations with no structured response projection.
func BinaryOnly(operation Operation) bool {
	if operation.Response == nil || len(operation.Response.Variants) == 0 {
		return false
	}
	for _, variant := range operation.Response.Variants {
		if variant.Format != "binary" {
			return false
		}
	}
	return true
}

// OutputFormats is the shared output vocabulary for a product command.
func OutputFormats(operation Operation) []string {
	if operation.Response == nil {
		return []string{"json", "table"}
	}
	for _, variant := range operation.Response.Variants {
		if variant.Format == "binary" {
			return []string{"raw"}
		}
	}
	return []string{"json", "raw", "table"}
}
