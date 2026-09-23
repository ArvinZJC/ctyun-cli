/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
)

// TestStreamingUploadOwnership permits the transport to finish reading after headers arrive.
func TestStreamingUploadOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upload")
	if err := os.WriteFile(path, []byte("complete upload"), 0600); err != nil {
		t.Fatal(err)
	}
	body, err := PrepareBody(BodyInput{Encoding: "file", Document: path})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	var request *http.Request
	response, err := Do(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		request = req
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("download"))}, nil
	}), RequestSpec{BaseURL: "https://example.test", PreparedBody: body, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := response.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	defer func() {
		if closeErr := request.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	data, err := io.ReadAll(request.Body)
	if err != nil || string(data) != "complete upload" {
		t.Fatalf("upload = %q, %v", data, err)
	}
	if request.Context().Err() != nil {
		t.Fatal("timeout cancelled before response closed")
	}
	if err := response.Close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(request.Context().Err(), context.Canceled) {
		t.Fatal("response close did not cancel request")
	}
}

// TestResponseStreamingDeadline keeps the timeout active until the body has completed.
func TestResponseStreamingDeadline(t *testing.T) {
	var ctx context.Context
	response, err := Do(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		ctx = req.Context()
		if req.Body != nil {
			if closeErr := req.Body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("data"))}, nil
	}), RequestSpec{BaseURL: "https://example.test", Timeout: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := response.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	<-ctx.Done()
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatal(ctx.Err())
	}
}

// TestPreparedRetryUsesIdenticalBytes verifies a file mutation cannot alter a signed retry.
func TestPreparedRetryUsesIdenticalBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upload")
	body := prepareTestFileBody(t, path, []byte("original"))
	attempts := 0
	response, err := Do(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		defer func() {
			if closeErr := req.Body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}()
		data, err := io.ReadAll(req.Body)
		if err != nil || string(data) != "original" {
			t.Fatalf("attempt %d: %q %v", attempts, data, err)
		}
		if attempts == 1 {
			if err := os.WriteFile(path, []byte("mutated"), 0600); err != nil {
				t.Fatal(err)
			}
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("retry"))}, nil
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	}), RequestSpec{BaseURL: "https://example.test", PreparedBody: body, Retries: 1, Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: 204, Format: "empty"}}}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := response.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	if attempts != 2 {
		t.Fatalf("attempts %d", attempts)
	}
}
