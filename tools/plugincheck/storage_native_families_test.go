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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
)

// TestNativeFamilyWireContracts checks service-specific protocols against independent expectations.
func TestNativeFamilyWireContracts(t *testing.T) {
	for _, tc := range []struct {
		slug, service, region string
		args                  []string
		check                 func(*testing.T, *http.Request, []byte)
	}{
		{"statistics.capacity.show", "s3", "cn-mg", []string{"--begin-date", "2026-05-21", "--end-date", "2026-05-22"}, func(t *testing.T, r *http.Request, b []byte) {
			if r.Method != "GET" || r.URL.Query().Get("Action") != "GetCapacity" || r.URL.Query().Get("BeginDate") != "2026-05-21" || len(b) != 0 {
				t.Fatal(r.URL, string(b))
			}
		}},
		{"iam.user-tags.set", "sts", "cn", []string{"--user-name", "test", "--tags", `[{"Key":"a&中","Value":""},{"Key":"b","Value":"a+b"}]`}, func(t *testing.T, r *http.Request, b []byte) {
			v, e := url.ParseQuery(string(b))
			if e != nil || v.Get("Action") != "TagUser" || v.Get("Tags.member.1.Key") != "a&中" || !v.Has("Tags.member.1.Value") || v.Get("Tags.member.2.Value") != "a+b" || r.Header.Get("Content-Type") != "application/octet-stream" {
				t.Fatal(string(b), e)
			}
		}},
		{"iam.session-token.show", "sts", "cn", []string{"--duration-seconds", "900", "--policy-document", `{"Statement":[]}`}, func(t *testing.T, r *http.Request, b []byte) {
			v, e := url.ParseQuery(string(b))
			if e != nil || v.Get("PolicyDocument") != `{"Statement":[]}` || v.Get("DurationSeconds") != "900" {
				t.Fatal(string(b), e)
			}
		}},
		{"tracking.event-selectors.set", "cloudtrail", "cn", []string{"--trail-name", "test", "--event-selectors", `[{"ReadWriteType":"All"}]`}, func(t *testing.T, r *http.Request, b []byte) {
			var v map[string]any
			e := json.Unmarshal(b, &v)
			if e != nil || len(v) != 2 || v["TrailName"] != "test" || !strings.HasSuffix(r.Header.Get("X-Amz-Target"), ".PutEventSelectors") {
				t.Fatal(string(b), e)
			}
		}},
	} {
		t.Run(tc.slug, func(t *testing.T) {
			ctx := loadStorageReviewContext(t, "classic-object-storage")
			id := "classic-object-storage.native." + tc.slug
			var args []string
			fixture := ""
			for _, cmd := range ctx.bundle.Commands.Commands {
				if cmd.Operation == id {
					args = append(args, cmd.Path...)
					fixture = cmd.FixtureResponse
				}
			}
			if fixture == "" {
				t.Fatal("missing command", id)
			}
			data, err := os.ReadFile(repoPath(t, filepath.Join("plugins/classic-object-storage", fixture)))
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var body []byte
				if r.Body != nil {
					body, _ = io.ReadAll(r.Body)
					if closeErr := r.Body.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				}
				tc.check(t, r, body)
				if !strings.Contains(r.Header.Get("Authorization"), "/"+tc.region+"/"+tc.service+"/aws4_request") || r.URL.Path != "/" {
					t.Fatal(r.URL, r.Header)
				}
				response, e := client.DecodeFixture(data)
				if e != nil {
					return nil, e
				}
				return &http.Response{StatusCode: response.Status, Header: response.Headers, Body: response.Body}, nil
			})
			args = append(args, tc.args...)
			args = append(args, "--yes", "--output", "json")
			var out bytes.Buffer
			err = cli.Run(cli.Config{Args: args, PluginRoot: t.TempDir(), Stdout: &out, Stderr: io.Discard, HTTPTransport: transport, Env: func(k string) string {
				return map[string]string{"CTYUN_STORAGE_AK": "ak", "CTYUN_STORAGE_SK": "sk", "CTYUN_STORAGE_ENDPOINT": "https://storage.example", "CTYUN_STORAGE_REGION": tc.region}[k]
			}})
			if err != nil || calls != 1 {
				t.Fatal(err, calls, args)
			}
		})
	}
}

// TestMediaPaginationBindings requires the documented wire keys and keeps listing filters out of V2 subresources.
func TestMediaPaginationBindings(t *testing.T) {
	ctx := loadStorageReviewContext(t, "media-storage")
	for slug, keys := range map[string][]string{"object.list": {"marker", "max-keys"}, "versions.list": {"key-marker", "version-id-marker", "max-keys"}, "parts.list": {"part-number-marker", "max-parts"}, "multipart-uploads.list": {"key-marker", "upload-id-marker", "max-uploads"}} {
		op := ctx.bundle.APIs.Operations["media-storage.native."+slug]
		for _, key := range keys {
			if op.Query[key] == "" || op.Headers[key] != "" {
				t.Fatal(slug, key)
			}
			for _, signed := range op.Native.V2QueryKeys {
				if signed == key {
					t.Fatal("listing filter became V2 subresource", slug, key)
				}
			}
		}
	}
}
