/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// TestBinaryDownloadPreservesStoredContentEncoding prevents transparent object decompression.
func TestBinaryDownloadPreservesStoredContentEncoding(t *testing.T) {
	var stored bytes.Buffer
	compressed := gzip.NewWriter(&stored)
	if _, err := io.WriteString(compressed, "stored object"); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		if _, closeErr := w.Write(stored.Bytes()); closeErr != nil {
			t.Error(closeErr)
		}
	}))
	defer server.Close()
	spec := RequestSpec{BaseURL: server.URL, Method: "GET", Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "binary"}}}}
	response, err := Do(nil, spec)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if _, err := CopyHTTPResponse(&got, response, spec); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), stored.Bytes()) {
		t.Fatalf("object encoding changed: received %d bytes, stored %d", got.Len(), stored.Len())
	}
}
