/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package client builds and executes signed CTyun API requests.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/signing"
)

// RequestSpec describes one CTyun API request after CLI metadata has resolved
// profiles, arguments, flags, and credentials into HTTP fields.
type RequestSpec struct {
	Method      string
	BaseURL     string
	Path        string
	Query       string
	ContentType string
	Body        []byte
	Headers     map[string]string
	Credentials config.Credentials
	RequestID   string
	Now         time.Time
	Timeout     time.Duration
	Retries     int
	Debug       io.Writer
	Language    string
}

// BuildRequest creates an HTTP request with CTyun EOP headers and optional
// authorization when credentials are present.
func BuildRequest(spec RequestSpec) (*http.Request, error) {
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
	req, err := http.NewRequest(spec.Method, url, bytes.NewReader(spec.Body))
	if err != nil {
		return nil, err
	}

	date := spec.Now.UTC().Format("20060102T150405Z")
	req.Header.Set("ctyun-eop-request-id", spec.RequestID)
	req.Header.Set("Eop-date", date)
	req.Header.Set("User-Agent", "ctyun-cli")
	if spec.ContentType != "" {
		req.Header.Set("Content-Type", spec.ContentType)
	}
	for key, value := range spec.Headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	if auth := signing.GenerateEOPAuthorization(signing.EOPRequest{
		Query:     spec.Query,
		Body:      spec.Body,
		Date:      date,
		RequestID: spec.RequestID,
	}, spec.Credentials); auth != "" {
		req.Header.Set("Eop-Authorization", auth)
	}
	return req, nil
}

// DoJSON sends a request, applies retry and timeout settings from spec, and
// decodes a successful JSON object response.
func DoJSON(transport http.RoundTripper, spec RequestSpec) (map[string]any, error) {
	if transport == nil {
		transport = http.DefaultTransport
	}

	attempts := spec.Retries + 1
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		// Rebuild each attempt so timeout contexts, generated request IDs, and
		// debug output reflect the actual request being sent.
		req, err := BuildRequest(spec)
		if err != nil {
			return nil, err
		}
		writeDebugRequest(spec.Debug, req, spec)
		cancel := func() {}
		if spec.Timeout > 0 {
			ctx, cancelFunc := context.WithTimeout(req.Context(), spec.Timeout)
			req = req.WithContext(ctx)
			cancel = cancelFunc
		}

		resp, err := transport.RoundTrip(req)
		if err != nil {
			cancel()
			writeDebugTransportError(spec.Debug, err, spec)
			lastErr = err
			if attempt+1 < attempts {
				continue
			}
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		cancel()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		writeDebugResponse(spec.Debug, resp.StatusCode, body, spec)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				return nil, diagnostic.Wrap("error.parse_response_json", err)
			}
			return payload, nil
		}
		lastErr = diagnostic.New("error.api_http", strconv.Itoa(resp.StatusCode), RedactHTTPDetails(string(body), spec.Credentials, spec.RequestID))
		// Retry only transient response classes; callers decide whether an
		// operation is safe to retry by setting RequestSpec.Retries.
		if attempt+1 < attempts && isRetryableStatus(resp.StatusCode) {
			continue
		}
		return nil, lastErr
	}

	return nil, diagnostic.New("error.api_request_failed")
}

// isRetryableStatus reports whether an HTTP status code is transient enough for
// metadata-approved retries.
func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// writeDebugRequest emits the redacted request line, headers, and body.
func writeDebugRequest(debug io.Writer, req *http.Request, spec RequestSpec) {
	if debug == nil {
		return
	}
	fmt.Fprintf(debug, "%s %s %s\n", debugText("debug.request", spec.Language), req.Method, req.URL.String())
	fmt.Fprintf(debug, "%s ctyun-eop-request-id=%s eop-authorization=%s\n",
		debugText("debug.request_headers", spec.Language),
		RedactHTTPDetails(req.Header.Get("ctyun-eop-request-id"), spec.Credentials, spec.RequestID),
		RedactHTTPDetails(req.Header.Get("Eop-Authorization"), spec.Credentials, spec.RequestID),
	)
	if len(spec.Body) > 0 {
		fmt.Fprintf(debug, "%s %s\n", debugText("debug.request_body", spec.Language), RedactHTTPDetails(string(spec.Body), spec.Credentials, spec.RequestID))
	}
}

// writeDebugResponse emits the redacted HTTP response status and body.
func writeDebugResponse(debug io.Writer, status int, body []byte, spec RequestSpec) {
	if debug == nil {
		return
	}
	fmt.Fprintf(debug, "%s %d\n", debugText("debug.response", spec.Language), status)
	if len(body) > 0 {
		fmt.Fprintf(debug, "%s %s\n", debugText("debug.response_body", spec.Language), RedactHTTPDetails(string(body), spec.Credentials, spec.RequestID))
	}
}

// writeDebugTransportError emits a redacted transport error.
func writeDebugTransportError(debug io.Writer, err error, spec RequestSpec) {
	if debug == nil {
		return
	}
	fmt.Fprintf(debug, "%s %s\n", debugText("debug.transport_error", spec.Language), RedactHTTPDetails(err.Error(), spec.Credentials, spec.RequestID))
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
	if idx := strings.Index(redacted, "Signature="); idx >= 0 {
		end := strings.IndexAny(redacted[idx:], " \n\t")
		if end < 0 {
			return redacted[:idx] + "Signature=[REDACTED]"
		}
		return redacted[:idx] + "Signature=[REDACTED]" + redacted[idx+end:]
	}
	return redacted
}
