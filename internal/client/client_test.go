package client

import (
	"bytes"
	"io"
	"net/http"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
