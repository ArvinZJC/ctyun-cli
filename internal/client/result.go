/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/jsonvalue"
)

// Result retains exact bytes alongside a structured view for presentation.
type Result struct {
	Status  int
	Headers http.Header
	Format  string
	Payload map[string]any
	Raw     []byte
	XML     *XMLNode
}

// responseVariant selects the explicit successful representation or legacy JSON.
func responseVariant(response *HTTPResponse, spec RequestSpec) (apicontract.Variant, error) {
	if spec.Response == nil {
		if response.Status < 200 || response.Status >= 300 {
			return apicontract.Variant{}, diagnostic.New("error.api_http", strconv.Itoa(response.Status), "")
		}
		return apicontract.Variant{Status: response.Status, Format: "json"}, nil
	}
	for _, variant := range spec.Response.Variants {
		if variant.Status == response.Status {
			if variant.MediaType != "" {
				values := response.Headers.Values("Content-Type")
				actual := strings.Join(values, ", ")
				mediaType, _, err := mime.ParseMediaType(actual)
				if len(values) != 1 || err != nil || !strings.EqualFold(mediaType, variant.MediaType) {
					return apicontract.Variant{}, diagnostic.New("error.response_media_type", actual, variant.MediaType)
				}
			}
			return variant, nil
		}
	}
	return apicontract.Variant{}, diagnostic.New("error.api_http", strconv.Itoa(response.Status), "")
}

// DecodeHTTPResponse consumes and closes a structured response using its declared policy.
func DecodeHTTPResponse(response *HTTPResponse, spec RequestSpec) (_ *Result, err error) {
	defer func() {
		if closeErr := response.Close(); err == nil {
			err = closeErr
		}
	}()
	variant, err := responseVariant(response, spec)
	if err != nil {
		return nil, err
	}
	if variant.Format == "binary" {
		return nil, apicontract.Invalid("response.binary_output")
	}
	reader := io.Reader(response.Body)
	if spec.Response != nil {
		reader = io.LimitReader(reader, MaxStructuredBody+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if spec.Response != nil && len(data) > MaxStructuredBody {
		return nil, apicontract.Invalid("response.size")
	}
	if response.requestID != "" {
		spec.RequestID = response.requestID
	}
	if err := writeDebugResponse(spec.Debug, response.Status, data, spec); err != nil {
		return nil, err
	}
	result := &Result{Status: response.Status, Headers: response.Headers, Format: variant.Format, Raw: data}
	switch variant.Format {
	case "json":
		result.Payload, err = DecodeResponse(data)
		if err != nil {
			return nil, diagnostic.Wrap("error.parse_response_json", err)
		}
		if spec.Response == nil {
			err = validateCTyunStatusCode(result.Payload, data, spec)
		} else {
			err = validateSuccess(result.Payload, spec.Response.Success)
		}
	case "xml":
		result.XML, err = DecodeXML(data)
		if err == nil && variant.XMLRoot != nil && result.XML.Name != *variant.XMLRoot {
			return nil, apicontract.Invalid("response.xml_root")
		}
		if err == nil {
			var encoded []byte
			encoded, err = json.Marshal(result.XML)
			if err == nil {
				result.Payload, err = DecodeResponse(encoded)
				if err == nil {
					result.Payload["status"] = json.Number(strconv.Itoa(response.Status))
				}
			}
		}
	case "text":
		result.Payload = map[string]any{"text": string(data)}
	case "empty":
		if response.Status == 303 && response.Headers.Get("Location") == "" {
			return nil, apicontract.Invalid("response.redirect.location")
		}
		if len(data) != 0 {
			return nil, apicontract.Invalid("response.empty")
		}
		headers := map[string]any{}
		for name, header := range variant.Headers {
			values := response.Headers.Values(header)
			if len(values) == 1 {
				headers[name] = values[0]
			} else {
				headers[name] = values
			}
		}
		result.Payload = map[string]any{"status": json.Number(strconv.Itoa(response.Status)), "headers": headers}
	default:
		return nil, apicontract.Invalid("response.format")
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// validateSuccess requires every declared path and value, including nested application codes.
func validateSuccess(payload map[string]any, checks []apicontract.Check) error {
	for _, check := range checks {
		var value any = payload
		for part := range strings.SplitSeq(check.Path, ".") {
			object, ok := value.(map[string]any)
			if !ok {
				return apicontract.Invalid("response.success")
			}
			value, ok = object[part]
			if !ok {
				return apicontract.Invalid("response.success")
			}
		}
		var text string
		switch value.(type) {
		case string, json.Number, bool:
			text = ctyunStatusCode(value)
		default:
			return apicontract.Invalid("response.success")
		}
		matched := false
		for _, allowed := range check.Values {
			matched = matched || text == allowed
			if number, ok := value.(json.Number); ok && json.Valid([]byte(allowed)) {
				matched = matched || jsonvalue.Equal(number, json.Number(allowed))
			}
		}
		if !matched {
			return diagnostic.New("error.api_status", text, "")
		}
	}
	return nil
}

// CopyHTTPResponse streams one binary body and never retries after exposing bytes.
func CopyHTTPResponse(destination io.Writer, response *HTTPResponse, spec RequestSpec) (_ int64, err error) {
	defer func() {
		if closeErr := response.Close(); err == nil {
			err = closeErr
		}
	}()
	variant, err := responseVariant(response, spec)
	if err != nil {
		return 0, err
	}
	if variant.Format != "binary" {
		return 0, apicontract.Invalid("response.binary_output")
	}
	if err = writeDebugResponse(spec.Debug, response.Status, nil, spec); err != nil {
		return 0, err
	}
	return io.Copy(destination, response.Body)
}
