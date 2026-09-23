/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/jsonvalue"
)

// DecodeResponse decodes one response object while preserving numeric resource
// identities exactly, including integers beyond float64's precision.
func DecodeResponse(data []byte) (map[string]any, error) {
	value, err := jsonvalue.Decode(data)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	payload, ok := value.(map[string]any)
	if !ok {
		return nil, diagnostic.New("error.expected_json_response_object")
	}
	return payload, nil
}
