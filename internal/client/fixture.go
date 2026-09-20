/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"io"
	"net/http"
	"strings"
)

// HTTPFixture records exact HTTP response evidence in a portable JSON file.
type HTTPFixture struct {
	SchemaVersion int         `json:"schema_version"`
	Status        int         `json:"status"`
	Headers       http.Header `json:"headers"`
	BodyBase64    string      `json:"body_base64"`
}

// DecodeFixture strictly decodes HTTP evidence without guessing JSON body shapes.
func DecodeFixture(data []byte) (*HTTPResponse, error) {
	var fixture HTTPFixture
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, apicontract.Invalid("fixture")
	}
	if fixture.SchemaVersion != 1 || fixture.Status < 100 || fixture.Status > 599 {
		return nil, apicontract.Invalid("fixture")
	}
	headers := make(http.Header)
	for key, values := range fixture.Headers {
		if !apicontract.HeaderName(key) {
			return nil, apicontract.Invalid("fixture.headers")
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n\x00") {
				return nil, apicontract.Invalid("fixture.headers")
			}
			headers.Add(key, value)
		}
	}
	body, err := base64.StdEncoding.Strict().DecodeString(fixture.BodyBase64)
	if err != nil {
		return nil, err
	}
	return &HTTPResponse{Status: fixture.Status, Headers: headers, Body: io.NopCloser(bytes.NewReader(body))}, nil
}
