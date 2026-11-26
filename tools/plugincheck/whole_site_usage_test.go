/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestWholeSiteUsageTimeColumns checks the schema and captured timestamp spellings through public table controls.
func TestWholeSiteUsageTimeColumns(t *testing.T) {
	const start = "2023-02-26T06:36:25Z"
	const end = "2023-02-26T13:14:06Z"
	args := []string{"whole-site-acceleration", "function", "usage", "show", "--name", "js_test", "--start-time", start, "--end-time", end, "--lang", "en-US"}
	for _, tc := range []struct {
		name    string
		options []string
		want    []string
	}{
		{"defaults", []string{"--offline"}, []string{start, end, "10"}},
		{"example keys", []string{"--offline", "--cols", "start_time_example,end_time_example", "--filter", "end_time_example=" + end}, []string{start, end}},
		{"documented names", []string{"--cols", "start_time,end_time"}, []string{start, end}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			cfg := cli.Config{Args: append(slices.Clone(args), tc.options...), Stdout: &out, PluginRoot: t.TempDir(), Config: []byte("{}"),
				Env: func(key string) string {
					if key == "CTYUN_AK" || key == "CTYUN_SK" || key == "CTYUN_REGION" {
						return "test"
					}
					return ""
				},
				HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
					if tc.name != "documented names" {
						t.Fatal("fixture command attempted a request")
					}
					body := `{"statusCode":100000,"returnObj":{"startTime":"` + start + `","endTime":"` + end + `","times":10}}`
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				})}
			if err := cli.Run(cfg); err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("missing %q in %s", want, out.String())
				}
			}
		})
	}
}
