package client

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/config"
)

func TestBuildRequestSignsAndRedacts(t *testing.T) {
	req, err := BuildRequest(RequestSpec{
		Method:      http.MethodPost,
		BaseURL:     "https://ctapi.example.test",
		Path:        "/v4/ecs/list-instance",
		ContentType: "application/json",
		Body:        []byte(`{"regionID":"cn-huadong1"}`),
		Credentials: config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"},
		RequestID:   "request-123",
		Now:         time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if req.Method != http.MethodPost {
		t.Fatalf("method = %s, want POST", req.Method)
	}
	if got := req.Header.Get("ctyun-eop-request-id"); got != "request-123" {
		t.Fatalf("request id header = %q", got)
	}
	if got := req.Header.Get("Eop-date"); got != "20260613T010203Z" {
		t.Fatalf("Eop-date = %q", got)
	}
	if got := req.Header.Get("Eop-Authorization"); got == "" || bytes.Contains([]byte(got), []byte("sk-test")) {
		t.Fatalf("authorization header is empty or leaked SK: %q", got)
	}

	redacted := RedactHTTPDetails("ak-test sk-test request-123 Signature=abc", config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"}, "request-123")
	for _, secret := range []string{"ak-test", "sk-test", "request-123", "abc"} {
		if bytes.Contains([]byte(redacted), []byte(secret)) {
			t.Fatalf("redacted details still contain %q: %s", secret, redacted)
		}
	}
}

func TestBuildRequestIncludesQueryAndExtraHeaders(t *testing.T) {
	req, err := BuildRequest(RequestSpec{
		Method:      http.MethodGet,
		BaseURL:     "https://ctapi.example.test",
		Path:        "/v4/demo",
		Query:       "pageNo=2&regionID=cn-huadong1",
		ContentType: "application/json",
		Headers: map[string]string{
			"x-ctyun-resource": "ecs",
		},
		Credentials: config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"},
		RequestID:   "request-123",
		Now:         time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if req.URL.RawQuery != "pageNo=2&regionID=cn-huadong1" {
		t.Fatalf("query = %q, want pageNo=2&regionID=cn-huadong1", req.URL.RawQuery)
	}
	if got := req.Header.Get("x-ctyun-resource"); got != "ecs" {
		t.Fatalf("extra header = %q, want ecs", got)
	}
	if req.Header.Get("Eop-Authorization") == "" {
		t.Fatal("request was not signed")
	}
}

func TestBuildRequestDefaultsMethodTimeAndRequestID(t *testing.T) {
	req, err := BuildRequest(RequestSpec{
		BaseURL:     "https://ctapi.example.test/",
		Path:        "/v4/demo",
		Credentials: config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"},
	})
	if err != nil {
		t.Fatalf("BuildRequest returned error: %v", err)
	}

	if req.Method != http.MethodPost {
		t.Fatalf("method = %q, want POST default", req.Method)
	}
	if req.Header.Get("ctyun-eop-request-id") == "" {
		t.Fatal("request id header was not defaulted")
	}
	if req.Header.Get("Eop-date") == "" {
		t.Fatal("Eop-date header was not defaulted")
	}
}

func TestBuildRequestRejectsInvalidURL(t *testing.T) {
	_, err := BuildRequest(RequestSpec{
		BaseURL: "://bad",
		Path:    "/v4/demo",
	})
	if err == nil {
		t.Fatal("BuildRequest returned nil error for invalid URL")
	}
}

func TestDoJSONUsesInjectedHTTPClient(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Eop-Authorization") == "" {
			t.Fatal("request was not signed")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"returnObj":{"ok":true}}`)),
		}, nil
	})

	payload, err := DoJSON(transport, RequestSpec{
		Method:      http.MethodPost,
		BaseURL:     "https://ctapi.example.test",
		Path:        "/v4/demo",
		ContentType: "application/json",
		Body:        []byte(`{}`),
		Credentials: config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"},
		RequestID:   "request-123",
		Now:         time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("DoJSON returned error: %v", err)
	}
	if payload["returnObj"] == nil {
		t.Fatalf("payload = %#v, want returnObj", payload)
	}
}

func TestDoJSONAppliesTimeoutAndRetriesRetryableRequests(t *testing.T) {
	attempts := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("request context has no deadline")
		}
		if attempts == 1 {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(`temporary`)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{"returnObj":{"ok":true}}`)),
		}, nil
	})

	payload, err := DoJSON(transport, RequestSpec{
		Method:      http.MethodPost,
		BaseURL:     "https://ctapi.example.test",
		Path:        "/v4/demo",
		ContentType: "application/json",
		Body:        []byte(`{}`),
		Credentials: config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"},
		RequestID:   "request-123",
		Now:         time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC),
		Timeout:     time.Second,
		Retries:     1,
	})
	if err != nil {
		t.Fatalf("DoJSON returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if payload["returnObj"] == nil {
		t.Fatalf("payload = %#v, want returnObj", payload)
	}
}

func TestDoJSONDebugLogRedactsSecrets(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`ak-test sk-test request-123 Signature=abc`)),
		}, nil
	})

	var debug bytes.Buffer
	_, err := DoJSON(transport, RequestSpec{
		Method:      http.MethodPost,
		BaseURL:     "https://ctapi.example.test",
		Path:        "/v4/demo",
		ContentType: "application/json",
		Body:        []byte(`{"secretEcho":"sk-test"}`),
		Credentials: config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"},
		RequestID:   "request-123",
		Now:         time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC),
		Debug:       &debug,
	})
	if err == nil {
		t.Fatal("DoJSON returned nil error for HTTP 400")
	}
	got := debug.String()
	for _, secret := range []string{"ak-test", "sk-test", "request-123", "abc"} {
		if bytes.Contains([]byte(got), []byte(secret)) {
			t.Fatalf("debug log still contains %q: %s", secret, got)
		}
	}
	for _, want := range []string{"request POST https://ctapi.example.test/v4/demo", "response 400"} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Fatalf("debug log missing %q: %s", want, got)
		}
	}
}

func TestDoJSONHandlesTransportAndResponseErrors(t *testing.T) {
	t.Run("transport error retries and logs", func(t *testing.T) {
		attempts := 0
		transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("dial sk-test request-123 Signature=abc")
		})
		var debug bytes.Buffer

		_, err := DoJSON(transport, RequestSpec{
			BaseURL:     "https://ctapi.example.test",
			Path:        "/v4/demo",
			Credentials: config.Credentials{AccessKey: "ak-test", SecretKey: "sk-test"},
			RequestID:   "request-123",
			Retries:     1,
			Debug:       &debug,
		})
		if err == nil {
			t.Fatal("DoJSON returned nil error for transport failure")
		}
		if attempts != 2 {
			t.Fatalf("attempts = %d, want retry", attempts)
		}
		if got := debug.String(); strings.Contains(got, "sk-test") || strings.Contains(got, "abc") {
			t.Fatalf("debug log leaked secret material: %s", got)
		}
	})

	t.Run("read body error", func(t *testing.T) {
		transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       errReadCloser{},
			}, nil
		})
		if _, err := DoJSON(transport, RequestSpec{BaseURL: "https://ctapi.example.test", Path: "/v4/demo"}); err == nil {
			t.Fatal("DoJSON returned nil error for body read failure")
		}
	})

	t.Run("close body error", func(t *testing.T) {
		transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       errCloseReadCloser{Reader: strings.NewReader(`{}`)},
			}, nil
		})
		if _, err := DoJSON(transport, RequestSpec{BaseURL: "https://ctapi.example.test", Path: "/v4/demo"}); err == nil {
			t.Fatal("DoJSON returned nil error for body close failure")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`not-json`)),
			}, nil
		})
		if _, err := DoJSON(transport, RequestSpec{BaseURL: "https://ctapi.example.test", Path: "/v4/demo"}); err == nil {
			t.Fatal("DoJSON returned nil error for invalid JSON")
		}
	})

	t.Run("non retryable error returns immediately", func(t *testing.T) {
		attempts := 0
		transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`bad`)),
			}, nil
		})
		if _, err := DoJSON(transport, RequestSpec{BaseURL: "https://ctapi.example.test", Path: "/v4/demo", Retries: 1}); err == nil {
			t.Fatal("DoJSON returned nil error for HTTP 400")
		}
		if attempts != 1 {
			t.Fatalf("attempts = %d, want no retry", attempts)
		}
	})
}

func TestDoJSONHandlesInvalidRequestAndNegativeRetries(t *testing.T) {
	if _, err := DoJSON(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("transport should not be called for invalid request")
		return nil, nil
	}), RequestSpec{BaseURL: "://bad", Path: "/v4/demo"}); err == nil {
		t.Fatal("DoJSON returned nil error for invalid request")
	}

	if _, err := DoJSON(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("transport should not be called when retries makes attempts zero")
		return nil, nil
	}), RequestSpec{BaseURL: "https://ctapi.example.test", Path: "/v4/demo", Retries: -1}); err == nil {
		t.Fatal("DoJSON returned nil error for zero attempts")
	}
}

func TestDoJSONUsesDefaultTransportWhenNoneInjected(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	})

	payload, err := DoJSON(nil, RequestSpec{BaseURL: "https://ctapi.example.test", Path: "/v4/demo"})
	if err != nil {
		t.Fatalf("DoJSON returned error: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("payload = %#v, want ok true", payload)
	}
}

func TestWriteDebugTransportErrorIgnoresNilWriter(t *testing.T) {
	writeDebugTransportError(nil, errors.New("sk-test"), RequestSpec{
		Credentials: config.Credentials{SecretKey: "sk-test"},
	})
}

func TestRedactHTTPDetailsHandlesSignatureInMiddle(t *testing.T) {
	got := RedactHTTPDetails("prefix Signature=abc suffix", config.Credentials{}, "")
	if strings.Contains(got, "abc") {
		t.Fatalf("signature was not redacted: %s", got)
	}
	if !strings.Contains(got, "suffix") {
		t.Fatalf("redaction removed trailing text: %s", got)
	}
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (errReadCloser) Close() error {
	return nil
}

type errCloseReadCloser struct {
	io.Reader
}

func (errCloseReadCloser) Close() error {
	return errors.New("close failed")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
