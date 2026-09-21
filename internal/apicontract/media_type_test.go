/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package apicontract

import (
	"encoding/json"
	"testing"
)

// TestResponseMediaTypeDeclaration accepts MIME essences and rejects ambiguous declarations.
func TestResponseMediaTypeDeclaration(t *testing.T) {
	for _, tc := range []struct {
		media string
		valid bool
	}{{"", true}, {"application/octet-stream", true}, {"Application/Octet-Stream", true}, {" application/octet-stream", false}, {"application/octet-stream ", false}, {"application/octet-stream;", false}, {"invalid", false}, {"application/*", false}, {"*/octet-stream", false}, {"application/octet-stream; charset=binary", false}, {"application/octet-stream\r\n", false}} {
		t.Run(tc.media, func(t *testing.T) {
			data, _ := json.Marshal(map[string]any{"variants": []any{map[string]any{"status": 200, "format": "binary", "media_type": tc.media}}})
			var response Response
			if err := json.Unmarshal(data, &response); err != nil {
				t.Fatal(err)
			}
			if err := Validate("GET", "", nil, &response); (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}
