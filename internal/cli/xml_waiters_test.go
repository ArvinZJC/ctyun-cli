/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package cli

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// xmlWaitTransport returns successive XML states and records safe retrieval requests.
type xmlWaitTransport struct {
	calls    int
	terminal string
}

// RoundTrip simulates an asynchronous resource independently of any product API.
func (transport *xmlWaitTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls++
	state := "working"
	if transport.calls > 1 {
		state = transport.terminal
	}
	if request.Method != "GET" {
		panic("waiter submitted a mutation")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/xml"}}, Body: io.NopCloser(strings.NewReader("<Object><State>" + state + "</State></Object>"))}, nil
}

// TestXMLWaiterThroughCommandEngine covers polling, completion, timeout and incompatible output.
func TestXMLWaiterThroughCommandEngine(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ecs")
	writeWaitBundle(t, dir)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{"operations":{"v4.ecs.instance.show":{"method":"GET","path":"/","retryable":true,"native":{"service":"s3","addressing":"path"},"response":{"variants":[{"status":200,"format":"xml","xml_root":{"local":"Object"}}]}}}}`)
	mustWrite(t, filepath.Join(dir, "waiters.json"), `{"waiters":{"ready":{"commands":["ecs.instance.show"],"xml_path":[{"local":"Object"},{"local":"State"}],"success":"ready","failure":"failed","max_attempts":2}}}`)
	for _, tc := range []struct{ terminal, want string }{{"ready", "success"}, {"failed", "failure"}, {"unknown", "timeout"}} {
		transport := &xmlWaitTransport{terminal: tc.terminal}
		var out, stderr bytes.Buffer
		err := Run(Config{Args: []string{"ecs", "instance", "show", "id", "--output", "json", "--wait", "ready", "--lang", "en-US"}, PluginRoot: root, Stdout: &out, Stderr: &stderr, HTTPTransport: transport, Env: func(k string) string {
			return map[string]string{"CTYUN_STORAGE_AK": "ak", "CTYUN_STORAGE_SK": "sk", "CTYUN_STORAGE_ENDPOINT": "https://storage.example", "CTYUN_STORAGE_REGION": "cn"}[k]
		}})
		if err != nil || transport.calls != 2 || !strings.Contains(stderr.String(), ": "+tc.want) || !strings.Contains(out.String(), "working") {
			t.Fatal(err, transport.calls, out.String(), stderr.String())
		}
	}
	got := completeArgs([]string{"ecs", "instance", "show", "id", "--wait", ""}, root)
	if len(got) != 1 || got[0] != "ready" {
		t.Fatal(got)
	}
}
