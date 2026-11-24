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
	"reflect"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestEventBridgeAndKafkaRequests checks provider-dependent source configuration
// and the wire types and sold-out result of the Kafka capacity query.
func TestEventBridgeAndKafkaRequests(t *testing.T) {
	sourceArgs := []string{"--yes", "eventbridge", "source", "create", "--event-source-name", "test1", "--event-bus-name", "test0723", "--type"}
	for _, tc := range []struct {
		name, path, project, result string
		args                        []string
		body                        map[string]any
		missingParams               bool
	}{
		{
			name: "simple source", path: "/v1/busSource/create", result: "{}",
			args: append(sourceArgs, "simple"),
			body: map[string]any{"eventSourceName": "test1", "eventBusName": "test0723", "type": "simple"},
		},
		{
			name: "Kafka source", path: "/v1/busSource/create", result: "{}",
			args: append(sourceArgs, "kafka", "--event-source-params", `[{"resourceKey":"instanceId","value":"instance","form":"CONSTANT"}]`),
			body: map[string]any{
				"eventSourceName": "test1", "eventBusName": "test0723", "type": "kafka",
				"eventSourceParams": []any{map[string]any{"resourceKey": "instanceId", "value": "instance", "form": "CONSTANT"}},
			},
		},
		{
			name: "missing provider parameters", args: append(sourceArgs, "kafka"), missingParams: true,
		},
		{
			name: "sold-out capacity", path: "/v3/instances/checkResourceSoldOut", project: "project",
			result: `{"soldOut":true,"message":"insufficient capacity"}`,
			args: []string{
				"kafka", "instance", "capacity", "check", "--project-id", "project", "--deploy-mode", "1",
				"--az-info", `["az1"]`, "--spec-name", "kafka.8u16g.cluster", "--host-type", "C",
				"--node-num", "3", "--data-disk-size", "100",
			},
			body: map[string]any{
				"deployMode": "1", "azInfo": []any{"az1"}, "specName": "kafka.8u16g.cluster",
				"hostType": "C", "nodeNum": float64(3), "dataDiskSize": "100",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			calls := 0
			err := cli.Run(cli.Config{
				Args: append([]string{"--lang", "en-US", "--output", "json"}, tc.args...), Stdout: &stdout, Stderr: io.Discard, PluginRoot: t.TempDir(),
				Config: []byte(`{"active_profile":"test","profiles":{"test":{"region":"test-region"}}}`),
				Env:    func(k string) string { return map[string]string{"CTYUN_AK": "test-ak", "CTYUN_SK": "test-sk"}[k] },
				HTTPTransport: nativeWireTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if tc.missingParams || r.Method != "POST" || r.URL.Path != tc.path || r.URL.RawQuery != "" || r.Header.Get("regionId") != "test-region" || r.Header.Get("projectId") != tc.project || r.Header.Get("Eop-Authorization") == "" {
						t.Fatalf("unexpected request %s %s", r.Method, r.URL)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(body, tc.body) {
						t.Fatalf("body=%#v, want %#v", body, tc.body)
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"statusCode":"800","returnObj":` + tc.result + `}`))}, nil
				}),
			})
			if tc.missingParams {
				if err == nil || !strings.Contains(err.Error(), "event-source-params") || calls != 0 {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
				return
			}
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			if tc.project != "" {
				var result struct{ ReturnObj struct{ SoldOut bool } }
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.ReturnObj.SoldOut {
					t.Fatalf("capacity result=%s err=%v", stdout.String(), err)
				}
			}
		})
	}
}
