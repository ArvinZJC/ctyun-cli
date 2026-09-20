/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// HTTPResponse owns the response body and its active timeout until Close.
type HTTPResponse struct {
	Status    int
	Headers   http.Header
	Body      io.ReadCloser
	requestID string
	cancel    context.CancelFunc
	once      sync.Once
	closeErr  error
}

// Close releases the response and timeout exactly once.
func (response *HTTPResponse) Close() error {
	response.once.Do(func() {
		if response.Body != nil {
			response.closeErr = response.Body.Close()
		}
		if response.cancel != nil {
			response.cancel()
		}
	})
	return response.closeErr
}

// Do executes signed requests without following redirects or buffering successful bodies.
func Do(transport http.RoundTripper, spec RequestSpec) (*HTTPResponse, error) {
	if spec.Response != nil {
		method := spec.Method
		if method == "" {
			method = http.MethodPost
		}
		if err := apicontract.Validate(method, "", nil, spec.Response); err != nil {
			return nil, err
		}
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	for attempt := 0; attempt <= spec.Retries; attempt++ {
		req, err := BuildRequest(spec)
		if err != nil {
			return nil, err
		}
		attemptSpec := spec
		attemptSpec.RequestID = req.Header.Get("ctyun-eop-request-id")
		if err := writeDebugRequest(spec.Debug, req, attemptSpec); err != nil {
			if req.Body != nil {
				// Preserve the primary failure while releasing this resource.
				_ = req.Body.Close()
			}
			return nil, err
		}
		cancel := context.CancelFunc(func() {})
		if spec.Timeout > 0 {
			var ctx context.Context
			ctx, cancel = context.WithTimeout(req.Context(), spec.Timeout)
			req = req.WithContext(ctx)
		}
		resp, err := transport.RoundTrip(req)
		if err != nil {
			if resp != nil && resp.Body != nil {
				// Preserve the primary failure while releasing this resource.
				_ = resp.Body.Close()
			}
			cancel()
			if debugErr := writeDebugTransportError(spec.Debug, err, attemptSpec); debugErr != nil {
				return nil, debugErr
			}
			if attempt < spec.Retries {
				continue
			}
			return nil, err
		}
		if resp == nil || resp.Body == nil {
			cancel()
			return nil, apicontract.Invalid("response.body")
		}
		response := &HTTPResponse{Status: resp.StatusCode, Headers: resp.Header.Clone(), Body: resp.Body, cancel: cancel, requestID: attemptSpec.RequestID}
		accepted := resp.StatusCode >= 200 && resp.StatusCode < 300
		if spec.Response != nil {
			accepted = false
			for _, v := range spec.Response.Variants {
				accepted = accepted || v.Status == resp.StatusCode
			}
		}
		if accepted {
			return response, nil
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 64<<10))
		closeErr := response.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if err := writeDebugResponse(spec.Debug, resp.StatusCode, body, attemptSpec); err != nil {
			return nil, err
		}
		if attempt < spec.Retries && isRetryableStatus(resp.StatusCode) {
			continue
		}
		return nil, diagnostic.New("error.api_http", strconv.Itoa(resp.StatusCode), RedactHTTPDetails(string(body), spec.Credentials, attemptSpec.RequestID, spec.sensitiveValues()...))
	}
	return nil, diagnostic.New("error.api_request_failed")
}
