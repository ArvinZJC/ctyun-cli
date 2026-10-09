/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestECPCSharedBandwidthRequests verifies the documented Boolean binding and
// conditional subnet requirement without sending requests to a live service.
func TestECPCSharedBandwidthRequests(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options []string
		fails   bool
		shared  any
	}{
		{"shared without subnets", []string{"--is-share-type", "true"}, false, true},
		{"ordinary omitted selector", nil, true, nil},
		{"ordinary explicit selector", []string{"--is-share-type", "false"}, true, false},
		{"ordinary with subnets", []string{"--subnet-oid-list", `["subnet"]`}, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			args := []string{"--lang", "en-US", "ecpc", "bandwidth", "create-bandwidth", "--region", "region", "--name", "bandwidth", "--size", "2", "--vpc-oid", "vpc", "--resource-pack-oid", "pack", "--yes", "--output", "json"}
			args = append(args, tc.options...)
			err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if r.Method != "POST" || r.URL.Path != "/v3/bandwidth/create" || r.URL.Query().Get("regionId") != "region" || body["isShareType"] != tc.shared {
					t.Fatalf("unexpected request %s %v", r.URL, body)
				}
				if _, ok := body["isShareType"]; tc.shared == nil && ok {
					t.Fatal("omitted selector was injected into request")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"submitted":true}}`))}, nil
			})})
			if tc.fails {
				if err == nil || !strings.Contains(err.Error(), "--subnet-oid-list") || calls != 0 {
					t.Fatalf("err=%v calls=%d", err, calls)
				}
			} else if err != nil || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
	t.Run("list query", func(t *testing.T) {
		calls := 0
		err := cli.Run(cli.Config{Args: []string{"ecpc", "bandwidth", "describe-bandwidths", "--region", "region", "--is-share-type", "true", "--output", "json"}, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "GET" || r.URL.Path != "/v3/bandwidth/describe" || r.URL.Query().Get("isShareType") != "true" {
				t.Fatalf("unexpected request %s", r.URL)
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"list":[]}}`))}, nil
		})})
		if err != nil || calls != 1 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	})
}

// TestECPCSharedBandwidthHelpAndCompletion protects localized conditional help
// and identical option discovery from empty and partially typed option tokens.
func TestECPCSharedBandwidthHelpAndCompletion(t *testing.T) {
	for _, command := range []string{"create-bandwidth", "describe-bandwidths"} {
		for _, token := range []string{"", "--is-share"} {
			var out bytes.Buffer
			err := cli.Run(cli.Config{Args: []string{"__complete", "ecpc", "bandwidth", command, token}, PluginRoot: t.TempDir(), Stdout: &out})
			if err != nil || !strings.Contains(out.String(), "--is-share-type") {
				t.Fatalf("completion: %v %s", err, out.String())
			}
		}
	}
	for _, token := range []string{"", "--share-band"} {
		var out bytes.Buffer
		err := cli.Run(cli.Config{Args: []string{"__complete", "ecpc", "floating-ip", "describe-floating-ip-create-price", token}, PluginRoot: t.TempDir(), Stdout: &out})
		if err != nil || !strings.Contains(out.String(), "--share-bandwidth-oid") {
			t.Fatalf("pricing option completion: %v %s", err, out.String())
		}
	}
	for _, token := range []string{"", "share"} {
		var out bytes.Buffer
		err := cli.Run(cli.Config{Args: []string{"__complete", "ecpc", "floating-ip", "describe-floating-ip-create-price", "--charge-type", token}, PluginRoot: t.TempDir(), Stdout: &out})
		if err != nil || !strings.Contains(out.String(), "shareBandWidth") {
			t.Fatalf("pricing value completion: %v %s", err, out.String())
		}
	}
	for _, tc := range []struct{ lang, phrase string }{{"en-US", "ordinary bandwidth"}, {"zh-CN", "普通带宽"}} {
		var out bytes.Buffer
		err := cli.Run(cli.Config{Args: []string{"--lang", tc.lang, "ecpc", "bandwidth", "create-bandwidth", "--help"}, PluginRoot: t.TempDir(), Stdout: &out})
		if err != nil || !strings.Contains(out.String(), "--is-share-type") || !strings.Contains(out.String(), tc.phrase) {
			t.Fatalf("help: %v %s", err, out.String())
		}
	}
}

// TestECPCSharedFloatingIPPricing permits shared-bandwidth pricing without a
// dedicated bandwidth body and preserves validation for the ordinary branches.
func TestECPCSharedFloatingIPPricing(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options []string
		fails   bool
	}{
		{"shared", []string{"--bill-mode", "OnDemand", "--charge-type", "shareBandWidth", "--share-bandwidth-oid", "shared"}, false},
		{"cycle missing bandwidth", []string{"--bill-mode", "Cycle"}, true},
		{"on demand unspecified", []string{"--bill-mode", "OnDemand"}, true},
		{"ordinary fixed bandwidth", []string{"--bill-mode", "OnDemand", "--charge-type", "Bandwidth"}, true},
		{"ordinary traffic", []string{"--bill-mode", "OnDemand", "--charge-type", "Traffic"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			args := append([]string{"ecpc", "floating-ip", "describe-floating-ip-create-price", "--region", "region", "--count", "1", "--output", "json"}, tc.options...)
			err := cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: io.Discard, Stderr: io.Discard, Env: func(k string) string { return map[string]string{"CTYUN_AK": "ak", "CTYUN_SK": "sk"}[k] }, HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if r.URL.Path != "/v3/fip/describeCreatePrice" || body["shareBandWidthOid"] != "shared" || body["chargeType"] != "shareBandWidth" {
					t.Fatalf("unexpected pricing request: %s %v", r.URL, body)
				}
				if _, ok := body["band"]; ok {
					t.Fatal("shared pricing sent an invented bandwidth object")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":800,"returnObj":{"subOrderPrices":[]}}`))}, nil
			})})
			if tc.fails {
				if err == nil || calls != 0 {
					t.Fatalf("err=%v calls=%d", err, calls)
				}
			} else if err != nil || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}
