/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBundledECSInstanceShowUsesDetailsAPI(t *testing.T) {
	var seen bool
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seen = true
		if req.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/v4/ecs/instance-details" {
			t.Fatalf("path = %q, want /v4/ecs/instance-details", req.URL.Path)
		}
		if req.URL.RawQuery != "instanceID=ins-live-1&regionID=cn-huadong1" {
			t.Fatalf("query = %q, want instanceID=ins-live-1&regionID=cn-huadong1", req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"instanceID":"ins-live-1","displayName":"live","instanceStatus":"running"}}`)),
		}, nil
	})

	var stdout bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "ecs", "instance", "show", "ins-live-1", "--cols", "instance_id"},
		Stdout:        &stdout,
		HTTPTransport: transport,
		Env: func(key string) string {
			switch key {
			case "CTYUN_AK":
				return "ak-test"
			case "CTYUN_SK":
				return "sk-test"
			default:
				return ""
			}
		},
		Config: []byte(`{
  "active_profile": "default",
  "profiles": {
    "default": {
      "region": "cn-huadong1",
      "endpoint_url": "https://ctapi.example.test"
    }
  }
}`),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !seen {
		t.Fatal("HTTP transport was not used")
	}
	if !strings.Contains(stdout.String(), "ins-live-1") {
		t.Fatalf("show output missing instance id:\n%s", stdout.String())
	}
}
