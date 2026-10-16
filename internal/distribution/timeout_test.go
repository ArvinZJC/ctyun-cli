/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package distribution

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestHostedFetchBoundsAndReleasesContext catches unbounded update requests and
// timers left running after either a successful fetch or a transport failure.
func TestHostedFetchBoundsAndReleasesContext(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			var requestContext context.Context
			data, err := HTTPGetBytes("https://example.test/archive.tar.gz", roundTripFunc(func(request *http.Request) (*http.Response, error) {
				requestContext = request.Context()
				deadline, bounded := requestContext.Deadline()
				if !bounded || time.Until(deadline) <= 0 {
					t.Error("hosted request has no future deadline")
				}
				if fail {
					return nil, context.DeadlineExceeded
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("artifact")), Header: make(http.Header)}, nil
			}))
			if fail && err == nil || !fail && (err != nil || string(data) != "artifact") {
				t.Fatalf("fetch = %q, %v", data, err)
			}
			if !errors.Is(requestContext.Err(), context.Canceled) {
				t.Errorf("completed request context = %v, want canceled", requestContext.Err())
			}
		})
	}
}

// TestSignedIndexFallsBackAfterTimeout verifies that a bounded primary request
// can fail over to a mirror whose index still requires signature validation.
func TestSignedIndexFallsBackAfterTimeout(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	index := `{"plugins":[]}`
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(index)))
	source := Source{Name: "primary", URL: "https://primary.test", Fallbacks: []Source{{Name: "mirror", URL: "https://mirror.test"}}}
	selected, data, err := ReadSignedIndexWithFallbacks(source, "index.json", "index.sig", base64.StdEncoding.EncodeToString(publicKey), "registry", roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if _, bounded := request.Context().Deadline(); !bounded {
			t.Error("index request has no deadline")
		}
		if request.URL.Host == "primary.test" {
			return nil, context.DeadlineExceeded
		}
		body := index
		if request.URL.Path == "/index.sig" {
			body = signature
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if selected.Name != "mirror" || string(data) != index {
		t.Fatalf("fallback = %q, %q, %v", selected.Name, data, err)
	}
}
