/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// catalogUsesTransport detects metadata requiring the explicit transport core.
func catalogUsesTransport(catalog Catalog) bool {
	for _, operation := range catalog.Operations {
		if plugin.UsesTransport(plugin.Operation{Native: operation.Native, Method: operation.Method, Request: operation.Request, Response: operation.Response.HTTP}) || operation.Download || operation.Response.XML != nil {
			return true
		}
	}
	return false
}

// validateOperationTransport validates captured evidence through runtime decoders.
func validateOperationTransport(operation Operation) error {
	if operation.FixtureUnavailable != "" && (strings.TrimSpace(operation.FixtureUnavailable) == "" || operation.Response.HTTP == nil || operation.Fixture != nil || (len(operation.ExampleResponse) != 0 && !bytes.Equal(bytes.TrimSpace(operation.ExampleResponse), []byte("null"))) || operation.Response.XML != nil || len(operation.Response.AcceptedStatuses) != 0) {
		return apicontract.Invalid("fixture_unavailable")
	}
	if err := apicontract.ValidateNative(operation.Native, operation.Path, operation.Response.HTTP); err != nil {
		return err
	}
	if err := apicontract.Validate(operation.Method, operation.ContentType, operation.Request, operation.Response.HTTP); err != nil {
		return err
	}
	if operation.Response.HTTP == nil {
		if operation.Fixture != nil {
			return apicontract.Invalid("http_fixture")
		}
		return nil
	}
	if operation.FixtureUnavailable != "" {
		return nil
	}
	if len(operation.ExampleResponse) != 0 && !bytes.Equal(bytes.TrimSpace(operation.ExampleResponse), []byte("null")) || operation.Fixture == nil || len(operation.Response.AcceptedStatuses) != 0 {
		return apicontract.Invalid("http_fixture")
	}
	// HTTPFixture contains only JSON-safe scalar and string-slice fields.
	data, _ := json.Marshal(operation.Fixture)
	response, err := client.DecodeFixture(data)
	if err != nil {
		return err
	}
	// Decode/copy helpers close the fixture; this also covers early validation failures.
	defer func() { _ = response.Close() }()
	spec := client.RequestSpec{Method: operation.Method, Response: operation.Response.HTTP}
	for _, variant := range operation.Response.HTTP.Variants {
		if variant.Status == response.Status && variant.Format == "binary" {
			_, err = client.CopyHTTPResponse(io.Discard, response, spec)
			return err
		}
	}
	result, err := client.DecodeHTTPResponse(response, spec)
	if err != nil {
		return err
	}
	if operation.Response.XML != nil && result.XML != nil {
		_, err = client.ProjectXML(result.XML, *operation.Response.XML)
	}
	return err
}

// operationExamplePayload decodes captured examples using the runtime representation contract.
func operationExamplePayload(operation Operation) (map[string]any, error) {
	if operation.Response.HTTP == nil {
		return client.DecodeResponse(operation.ExampleResponse)
	}
	// HTTPFixture contains only JSON-safe scalar and string-slice fields.
	data, _ := json.Marshal(operation.Fixture)
	// DecodeHTTPResponse takes ownership and closes the response on every path.
	//goland:noinspection GoResourceLeak
	response, err := client.DecodeFixture(data)
	if err != nil {
		return nil, err
	}
	result, err := client.DecodeHTTPResponse(response, client.RequestSpec{Method: operation.Method, Response: operation.Response.HTTP})
	if err != nil {
		return nil, err
	}
	return result.Payload, nil
}
