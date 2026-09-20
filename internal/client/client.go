/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package client builds and executes signed CTyun API requests.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/jsonvalue"
	"github.com/ArvinZJC/ctyun-cli/internal/signing"
)

// RequestSpec describes one CTyun API request after CLI metadata has resolved
// profiles, arguments, flags, and credentials into HTTP fields.
type RequestSpec struct {
	PreparedBody *PreparedBody
	Response     *apicontract.Response
	Method       string
	BaseURL      string
	Path         string
	Query        string
	ContentType  string
	Body         []byte
	Headers      map[string]string
	Credentials  config.Credentials
	RequestID    string
	Now          time.Time
	Timeout      time.Duration
	Retries      int
	Debug        io.Writer
	Language     string
	// AcceptedStatuses extends CTyun application success handling for APIs
	// whose useful result can come with a non-800 status and a verified body
	// shape.
	AcceptedStatuses []AcceptedStatusRule
}

// AcceptedStatusRule declares one non-default CTyun application status that
// can be treated as successful when its optional response path is present.
type AcceptedStatusRule struct {
	Code         string
	RequiredPath string
}

// BuildRequest creates an HTTP request with CTyun EOP headers and optional
// authorization when credentials are present.
func BuildRequest(spec RequestSpec) (*http.Request, error) {
	for name, value := range spec.Headers {
		if !apicontract.HeaderName(name) || strings.ContainsAny(value, "\r\n\x00") {
			return nil, apicontract.Invalid("headers")
		}
		switch strings.ToLower(name) {
		case "host", "content-length", "transfer-encoding", "eop-authorization", "eop-date", "ctyun-eop-request-id":
			return nil, apicontract.Invalid("headers." + name)
		}
		if spec.PreparedBody != nil && strings.EqualFold(name, "Content-Type") {
			return nil, apicontract.Invalid("headers.content-type")
		}
	}
	if spec.Method == "" {
		spec.Method = http.MethodPost
	}
	if spec.Now.IsZero() {
		spec.Now = time.Now().UTC()
	}
	if spec.RequestID == "" {
		spec.RequestID = strconv.FormatInt(spec.Now.UnixNano(), 36)
	}

	url := strings.TrimRight(spec.BaseURL, "/") + spec.Path
	if spec.Query != "" {
		url += "?" + spec.Query
	}
	var reader io.Reader = bytes.NewReader(spec.Body)
	var prepared io.ReadCloser
	if spec.PreparedBody != nil {
		if len(spec.Body) != 0 {
			return nil, apicontract.Invalid("request.body")
		}
		var err error
		prepared, err = spec.PreparedBody.Open()
		if err != nil {
			return nil, err
		}
		reader = prepared
	}
	req, err := http.NewRequest(spec.Method, url, reader)
	if err != nil {
		if prepared != nil {
			prepared.Close()
		}
		return nil, err
	}
	if spec.PreparedBody != nil {
		req.ContentLength = spec.PreparedBody.Length
		if req.ContentLength == 0 {
			req.Body.Close()
			req.Body = http.NoBody
		}
		req.Header.Set("Content-Type", spec.PreparedBody.ContentType)
	}

	date := spec.Now.UTC().Format("20060102T150405Z")
	req.Header.Set("ctyun-eop-request-id", spec.RequestID)
	req.Header.Set("Eop-date", date)
	req.Header.Set("User-Agent", "ctyun-cli")
	if spec.ContentType != "" && spec.PreparedBody == nil {
		req.Header.Set("Content-Type", spec.ContentType)
	}
	for key, value := range spec.Headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	// Explicit binary transfers preserve stored Content-Encoding bytes. Setting
	// Accept-Encoding prevents Go from transparently requesting and decoding gzip.
	if spec.Response != nil && req.Header.Get("Accept-Encoding") == "" {
		for _, variant := range spec.Response.Variants {
			if variant.Format == "binary" {
				req.Header.Set("Accept-Encoding", "identity")
				break
			}
		}
	}
	signingRequest := signing.EOPRequest{
		Query:     spec.Query,
		Body:      spec.Body,
		Date:      date,
		RequestID: spec.RequestID,
	}
	auth := signing.GenerateEOPAuthorization(signingRequest, spec.Credentials)
	if spec.PreparedBody != nil {
		auth = signing.GenerateEOPAuthorizationDigest(signingRequest, spec.PreparedBody.SHA256, spec.Credentials)
	}
	if auth != "" {
		req.Header.Set("Eop-Authorization", auth)
	}
	return req, nil
}

// DoJSON sends a request, applies retry and timeout settings from spec, and
// decodes a successful JSON object response.
func DoJSON(transport http.RoundTripper, spec RequestSpec) (map[string]any, error) {
	response, err := Do(transport, spec)
	if err != nil {
		return nil, err
	}
	result, err := DecodeHTTPResponse(response, spec)
	if err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// validateCTyunStatusCode treats CTyun API statusCode 800 as success and
// applies operation-specific body-shape checks for any other accepted status.
func validateCTyunStatusCode(payload map[string]any, body []byte, spec RequestSpec) error {
	value, ok := payload["statusCode"]
	if !ok {
		return nil
	}
	status := ctyunStatusCode(value)
	if status == "800" {
		return nil
	}
	for _, rule := range spec.AcceptedStatuses {
		if status != "900" || rule.Code != "900" || rule.RequiredPath == "" {
			continue
		}
		if responsePathExists(payload, rule.RequiredPath) {
			return nil
		}
	}
	return diagnostic.New("error.api_status", status, RedactHTTPDetails(string(body), spec.Credentials, spec.RequestID))
}

// ctyunStatusCode returns the string form of a CTyun application status code.
func ctyunStatusCode(value any) string {
	switch typed := value.(type) {
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return jsonvalue.NumberText(typed)
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

// responsePathExists reports whether a dotted JSON object path exists.
func responsePathExists(payload map[string]any, path string) bool {
	var current any = payload
	for part := range strings.SplitSeq(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return false
		}
		next, ok := object[part]
		if !ok {
			return false
		}
		current = next
	}
	return true
}

// isRetryableStatus reports whether an HTTP status code is transient enough for
// metadata-approved retries.
func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// writeDebugRequest emits the redacted request line, headers, and body.
func writeDebugRequest(debug io.Writer, req *http.Request, spec RequestSpec) error {
	if debug == nil {
		return nil
	}
	err := writeDebugf(debug, "%s %s %s\n", debugText("debug.request", spec.Language), req.Method, RedactHTTPDetails(req.URL.String(), spec.Credentials, spec.RequestID))
	if err == nil {
		err = writeDebugf(debug, "%s ctyun-eop-request-id=%s eop-authorization=%s\n",
			debugText("debug.request_headers", spec.Language),
			RedactHTTPDetails(req.Header.Get("ctyun-eop-request-id"), spec.Credentials, spec.RequestID),
			RedactHTTPDetails(req.Header.Get("Eop-Authorization"), spec.Credentials, spec.RequestID),
		)
	}
	if err == nil && len(spec.Body) > 0 {
		err = writeDebugf(debug, "%s %s\n", debugText("debug.request_body", spec.Language), RedactHTTPDetails(string(spec.Body), spec.Credentials, spec.RequestID))
	}
	return err
}

// writeDebugResponse emits the redacted HTTP response status and body.
func writeDebugResponse(debug io.Writer, status int, body []byte, spec RequestSpec) error {
	if debug == nil {
		return nil
	}
	err := writeDebugf(debug, "%s %d\n", debugText("debug.response", spec.Language), status)
	if err == nil && len(body) > 0 {
		err = writeDebugf(debug, "%s %s\n", debugText("debug.response_body", spec.Language), RedactHTTPDetails(string(body), spec.Credentials, spec.RequestID))
	}
	return err
}

// writeDebugTransportError emits a redacted transport error.
func writeDebugTransportError(debug io.Writer, err error, spec RequestSpec) error {
	if debug == nil {
		return nil
	}
	return writeDebugf(debug, "%s %s\n", debugText("debug.transport_error", spec.Language), RedactHTTPDetails(err.Error(), spec.Credentials, spec.RequestID))
}

// writeDebugf writes one formatted debug line and returns writer failures.
func writeDebugf(debug io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(debug, format, args...)
	return err
}

var debugCatalog = i18n.Catalog{
	"debug.request":         {"en-US": "request", "en-GB": "request", "zh-CN": "请求"},
	"debug.request_headers": {"en-US": "request headers:", "en-GB": "request headers:", "zh-CN": "请求头："},
	"debug.request_body":    {"en-US": "request body:", "en-GB": "request body:", "zh-CN": "请求体："},
	"debug.response":        {"en-US": "response", "en-GB": "response", "zh-CN": "响应"},
	"debug.response_body":   {"en-US": "response body:", "en-GB": "response body:", "zh-CN": "响应体："},
	"debug.transport_error": {"en-US": "transport error:", "en-GB": "transport error:", "zh-CN": "传输错误："},
}

// debugText returns localized debug labels for HTTP diagnostics.
func debugText(key, language string) string {
	return debugCatalog.Text(key, language)
}

// RedactHTTPDetails removes credentials, request IDs, and CTyun signatures from
// debug or error text before it is shown to users.
func RedactHTTPDetails(input string, creds config.Credentials, requestID string) string {
	redacted := signing.RedactSecrets(input, []string{
		creds.AccessKey,
		creds.SecretKey,
		requestID,
	})
	return redactCredentialFields(redacted)
}
