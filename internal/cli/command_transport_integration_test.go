/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/apicontract"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// transportFixture creates synthetic HTTP evidence independently of product bundles.
func transportFixture(t *testing.T, format, body string, status int) (plugin.Bundle, plugin.Command) {
	t.Helper()
	dir := t.TempDir()
	data, err := json.Marshal(client.HTTPFixture{SchemaVersion: 1, Status: status, Headers: http.Header{"Etag": {"one", "two"}}, BodyBase64: base64.StdEncoding.EncodeToString([]byte(body))})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "response.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	command := plugin.Command{ID: "synthetic.show", Path: []string{"synthetic", "show"}, Operation: "show", Table: "show", FixtureResponse: "response.json", Download: format != "empty"}
	operation := plugin.Operation{Method: "GET", Path: "/result", Response: &apicontract.Response{Variants: []apicontract.Variant{{Status: status, Format: format, Headers: map[string]string{"etag": "ETag"}}}}}
	table := plugin.Table{RowPath: "$", Columns: []plugin.TableColumn{{Key: "status", Path: "status", Labels: map[string]string{"en-US": "Status"}}}}
	return plugin.Bundle{Dir: dir, Manifest: plugin.Manifest{API: plugin.APIInfo{EndpointURL: "https://example.test"}}, APIs: plugin.APIs{Operations: map[string]plugin.Operation{"show": operation}}, Tables: plugin.Tables{Tables: map[string]plugin.Table{"show": table}}}, command
}

// TestTransportFixtureOutput preserves response bytes and validates before file publication.
func TestTransportFixtureOutput(t *testing.T) {
	for _, tc := range []struct {
		format, body string
		status       int
	}{{"xml", "<r>001</r>\n", 200}, {"text", "hello\n", 200}, {"json", "{\"n\": 9007199254740993}\n", 200}, {"binary", "\x00\xffabc", 200}, {"empty", "", 204}} {
		t.Run(tc.format, func(t *testing.T) {
			bundle, command := transportFixture(t, tc.format, tc.body, tc.status)
			var out bytes.Buffer
			opts := globalOptions{Fixture: true, Output: "raw", Language: "en-US"}
			run := func(values map[string]string, writer io.Writer) error {
				return runTransportCommand(writer, io.Discard, bundle, command, nil, values, opts, coreconfig.Profile{}, func(string) string { return "" }, nil, nil, "")
			}
			if err := run(nil, &out); err != nil || out.String() != tc.body {
				t.Fatalf("raw %q %v", out.String(), err)
			}
			if tc.format == "empty" {
				return
			}
			path := filepath.Join(t.TempDir(), "result")
			values := map[string]string{transferParameter("output-file"): path}
			opts.Output = "table"
			if err := run(values, &out); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != tc.body {
				t.Fatalf("file=%q %v", data, err)
			}
			if err = run(values, &out); err == nil {
				t.Fatal("existing file overwritten")
			}
			values[transferParameter("overwrite")] = "true"
			if err = run(values, &out); err != nil {
				t.Fatal(err)
			}
			opts.Output = "raw"
			if err = run(nil, failingWriter{}); err == nil {
				t.Fatal("writer failure ignored")
			}
		})
	}
	bundle, command := transportFixture(t, "json", `{"statusCode":0,"returnObj":{"code":1}}`, 200)
	op := bundle.APIs.Operations["show"]
	op.Response.Success = []apicontract.Check{{Path: "statusCode", Values: []string{"0"}}, {Path: "returnObj.code", Values: []string{"0"}}}
	bundle.APIs.Operations["show"] = op
	path := filepath.Join(t.TempDir(), "failed")
	err := runTransportCommand(io.Discard, io.Discard, bundle, command, nil, map[string]string{transferParameter("output-file"): path}, globalOptions{Fixture: true}, coreconfig.Profile{}, func(string) string { return "" }, nil, nil, "")
	if err == nil {
		t.Fatal("failed application accepted")
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("failed result published")
	}
}

// TestTransportControlsRejectIncompatibleOptions checks presence rather than only values.
func TestTransportControlsRejectIncompatibleOptions(t *testing.T) {
	bundle, command := transportFixture(t, "binary", "data", 200)
	for _, opts := range []globalOptions{{Output: "table"}, {Output: "json"}, {Output: "raw", Waiter: "ready"}, {Output: "raw", Seen: map[string]string{"table": "--table"}}, {Output: "raw", Seen: map[string]string{"no-header": "--no-header"}}} {
		if err := validateTransportOutput(bundle, command, nil, opts); err == nil {
			t.Fatalf("accepted %#v", opts)
		}
	}
	if err := validateTransportOutput(bundle, command, nil, globalOptions{Output: "raw"}); err != nil {
		t.Fatal(err)
	}
	if err := validateTransportOutput(bundle, command, map[string]string{transferParameter("overwrite"): "true"}, globalOptions{}); err == nil {
		t.Fatal("overwrite without path accepted")
	}
	values := map[string]string{transferParameter("output-file"): "out"}
	if err := validateTransportOutput(bundle, command, values, globalOptions{Seen: map[string]string{"output": "--output"}}); err == nil {
		t.Fatal("conflicting outputs accepted")
	}
	bundle.APIs.Operations["show"] = plugin.Operation{}
	if err := validateTransportOutput(bundle, command, nil, globalOptions{Output: "raw"}); err == nil {
		t.Fatal("legacy raw accepted")
	}
}

// TestExplicitBodyUsesOnlyBindings preserves empty strings without leaking query/header inputs.
func TestExplicitBodyUsesOnlyBindings(t *testing.T) {
	parameters := []plugin.Parameter{{Name: "empty", Target: "empty"}, {Name: "secret", Target: "header"}, {Name: "count", Target: "n", ValueType: plugin.ParameterValueInteger}}
	fields, err := resolveExplicitBody(map[string]string{"empty": "$param.empty", "absent": "$param.missing", "n": "$param.count", "literal": "fixed", "region": "$profile.region"}, coreconfig.Profile{Region: "region"}, nil, map[string]string{"empty": "", "secret": "hidden", "count": "123"}, parameters, "en-US")
	data, _ := json.Marshal(fields)
	if err != nil || string(data) != `{"empty":"","literal":"fixed","n":123,"region":"region"}` {
		t.Fatalf("fields=%s %v", data, err)
	}
	if strings.Contains(string(data), "hidden") {
		t.Fatal("unbound input leaked")
	}
	if _, err = resolveExplicitBody(map[string]string{"n": "$param.count"}, coreconfig.Profile{}, nil, map[string]string{"count": "bad"}, parameters, "en-US"); err == nil {
		t.Fatal("invalid typed input accepted")
	}
}

// TestPrepareCommandBodyResolvesExplicitInputs covers file, argument and optional multipart roles.
func TestPrepareCommandBodyResolvesExplicitInputs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "document.xml")
	if err := os.WriteFile(path, []byte("<root/>"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		request      apicontract.Request
		parameters   []plugin.Parameter
		values, args map[string]string
		want         string
		valid        bool
	}{
		{apicontract.Request{Encoding: "xml", Document: "$param.data"}, []plugin.Parameter{{Name: "data", Input: "file"}}, map[string]string{"data": path}, nil, "<root/>", true},
		{apicontract.Request{Encoding: "xml", Document: "$param.data"}, []plugin.Parameter{{Name: "data", Input: "file"}}, map[string]string{"data": path + "missing"}, nil, "", false},
		{apicontract.Request{Encoding: "xml", Document: "$param.data"}, nil, nil, nil, "", true},
		{apicontract.Request{Encoding: "xml", Document: "$arg.data"}, nil, nil, map[string]string{"data": "<root/>"}, "<root/>", true},
		{apicontract.Request{Encoding: "xml", Document: "$profile.region"}, nil, nil, nil, "<root/>", true},
		{apicontract.Request{Encoding: "xml", Document: "unknown"}, nil, nil, nil, "", false},
		{apicontract.Request{Encoding: "multipart", Parts: []apicontract.Part{{Name: "region", Source: "$profile.region"}, {Name: "absent", Source: "$param.absent"}, {Name: "empty", Source: "$param.empty"}}}, nil, map[string]string{"empty": ""}, nil, "", true},
	} {
		body, err := prepareCommandBody(plugin.Operation{Request: &tc.request}, plugin.Command{Parameters: tc.parameters}, tc.args, tc.values, coreconfig.Profile{Region: "<root/>"}, nil)
		if (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
		if body == nil {
			continue
		}
		defer body.Close()
		reader, err := body.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != "" && string(data) != tc.want {
			t.Fatalf("%q", data)
		}
		if tc.request.Encoding == "multipart" && (!bytes.Contains(data, []byte(`name=empty`)) && !bytes.Contains(data, []byte(`name="empty"`)) || bytes.Contains(data, []byte("absent"))) {
			t.Fatalf("parts %s", data)
		}
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), client.MaxStructuredBody+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareCommandBody(plugin.Operation{Request: &apicontract.Request{Encoding: "xml", Document: "$param.data"}}, plugin.Command{Parameters: []plugin.Parameter{{Name: "data", Input: "file"}}}, nil, map[string]string{"data": path}, coreconfig.Profile{}, nil); err == nil {
		t.Fatal("oversized XML accepted")
	}
}

// TestTransportLiveAndLegacyFixture shares request preparation and response decoding at the CLI boundary.
func TestTransportLiveAndLegacyFixture(t *testing.T) {
	bundle, command := transportFixture(t, "json", `{"statusCode":0}`, 200)
	operation := bundle.APIs.Operations["show"]
	operation.Response.Variants[0].Headers = nil
	operation.Request = &apicontract.Request{Encoding: "form"}
	operation.Body = map[string]string{"Action": "Test"}
	bundle.APIs.Operations["show"] = operation
	var stdout bytes.Buffer
	calls := 0
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		defer req.Body.Close()
		data, err := io.ReadAll(req.Body)
		if err != nil || string(data) != "Action=Test" {
			t.Fatalf("body %q %v", data, err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"statusCode":0}`))}, nil
	})
	opts := globalOptions{Output: "json", Language: "en-US", Timeout: 1}
	if err := runTransportCommand(&stdout, io.Discard, bundle, command, nil, nil, opts, coreconfig.Profile{}, regionTestCredentials, transport, nil, ""); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !strings.Contains(stdout.String(), "statusCode") {
		t.Fatal(stdout.String())
	}
	if err := runTransportCommand(io.Discard, io.Discard, bundle, command, nil, nil, opts, coreconfig.Profile{}, func(string) string { return "" }, transport, nil, ""); err == nil {
		t.Fatal("missing credentials accepted")
	}
	operation.Response = nil
	bundle.APIs.Operations["show"] = operation
	if err := os.WriteFile(filepath.Join(bundle.Dir, command.FixtureResponse), []byte(`{"statusCode":800}`), 0600); err != nil {
		t.Fatal(err)
	}
	opts.Fixture = true
	if err := runTransportCommand(io.Discard, io.Discard, bundle, command, nil, nil, opts, coreconfig.Profile{}, regionTestCredentials, nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	command.FixtureResponse = "absent"
	if err := runTransportCommand(io.Discard, io.Discard, bundle, command, nil, nil, opts, coreconfig.Profile{}, regionTestCredentials, nil, nil, ""); err == nil {
		t.Fatal("missing fixture accepted")
	}
}

// TestTransportHelpAndCompletionMatchCapabilities checks the same options at both UI boundaries.
func TestTransportHelpAndCompletionMatchCapabilities(t *testing.T) {
	bundle, command := transportFixture(t, "binary", "data", 200)
	context := completionContext{Bundle: bundle, Command: command, CommandFound: true, Path: command.Path}
	options := completionOptions(context)
	foundFile, foundOverwrite := false, false
	for _, option := range options {
		for _, name := range option.Names {
			switch name {
			case "--table", "--cols", "--sort", "--filter", "--no-header", "--wait":
				t.Errorf("inapplicable %s", name)
			case "--output-file":
				foundFile = true
			case "--overwrite":
				foundOverwrite = true
			}
			if name == "--output" {
				values := option.Values(context)
				if len(values) != 1 || values[0] != "raw" {
					t.Fatal(values)
				}
			}
		}
	}
	if !foundFile || !foundOverwrite {
		t.Fatal("download options missing")
	}
	var text bytes.Buffer
	writer := newOutputWriter(&text)
	printProductGlobalOptions(writer, "en-US", command.Path, bundle.APIs.Operations[command.Operation])
	if strings.Contains(text.String(), "--table") || !strings.Contains(text.String(), "<raw>") {
		t.Fatal(text.String())
	}
	for _, args := range [][]string{{"--output-file"}, {"--overwrite=value"}} {
		if _, err := parseCommandParameters(command, args, "en-US"); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

// TestTransportFixtureReloadAndXMLTables preserves the same decoder on polling and display paths.
func TestTransportFixtureReloadAndXMLTables(t *testing.T) {
	bundle, command := transportFixture(t, "xml", "<root><value>001</value></root>", 200)
	operation := bundle.APIs.Operations["show"]
	operation.Response.Variants[0].Headers = nil
	bundle.APIs.Operations["show"] = operation
	opts := globalOptions{Fixture: true, Language: "en-US"}
	payload, err := loadCommandResponse(bundle, command, nil, nil, opts, coreconfig.Profile{}, regionTestCredentials, nil, io.Discard, nil)
	if err != nil {
		t.Fatal(err)
	}
	table := plugin.Table{XML: &apicontract.XMLTable{Rows: []apicontract.XMLName{{Local: "root"}}, Columns: map[string]apicontract.XMLSelector{"value": {Path: []apicontract.XMLName{{Local: "value"}}}}}}
	rows, err := rowsFromPayload(payload, table)
	if err != nil || rows[0]["value"] != "001" {
		t.Fatalf("%#v %v", rows, err)
	}
	for _, bad := range []map[string]any{{"bad": make(chan int)}, {"name": "bad"}} {
		if _, err := rowsFromPayload(bad, table); err == nil {
			t.Fatal("invalid XML view accepted")
		}
	}
	for _, data := range []string{"invalid", `{"schema_version":1,"status":200,"body_base64":"YnJva2Vu"}`} {
		if err := os.WriteFile(filepath.Join(bundle.Dir, command.FixtureResponse), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadCommandResponse(bundle, command, nil, nil, opts, coreconfig.Profile{}, regionTestCredentials, nil, io.Discard, nil); err == nil {
			t.Fatal("invalid fixture accepted")
		}
	}
	if _, err := parseCommandParameters(command, []string{"--output-file="}, "en-US"); err == nil {
		t.Fatal("empty destination accepted")
	}
	if err := validateGlobalOptionScope([]string{"config", "explain"}, globalOptions{Output: "raw"}); err == nil {
		t.Fatal("raw core output accepted")
	}
	if usage := pluginCommandUsage(command, "en-US"); !strings.Contains(usage, "--output-file <path>") {
		t.Fatal(usage)
	}
	if rows := pluginCommandParameterHelpRows(bundle, command, "en-US"); len(rows) != 2 {
		t.Fatal(rows)
	}
}

// TestInstalledTransportCommandDispatch routes metadata through the shared engine and exposes matching help.
func TestInstalledTransportCommandDispatch(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ims")
	writeIMSBundleWithoutFixture(t, dir)
	bundle, err := plugin.LoadBundle(dir, "0.5.0")
	if err != nil {
		t.Fatal(err)
	}
	command := &bundle.Commands.Commands[0]
	command.FixtureResponse = "response.json"
	command.Download = true
	operation := bundle.APIs.Operations[command.Operation]
	operation.Response = &apicontract.Response{Variants: []apicontract.Variant{{Status: 200, Format: "text"}}}
	bundle.APIs.Operations[command.Operation] = operation
	bundle.Manifest.Requires.Ctyun = ">=0.5.0 <1.0.0"
	writeRecommendationJSON(t, filepath.Join(dir, "plugin.json"), bundle.Manifest)
	writeRecommendationJSON(t, filepath.Join(dir, "apis.json"), bundle.APIs)
	writeRecommendationJSON(t, filepath.Join(dir, "commands.json"), bundle.Commands)
	writeRecommendationJSON(t, filepath.Join(dir, "response.json"), client.HTTPFixture{SchemaVersion: 1, Status: 200, BodyBase64: base64.StdEncoding.EncodeToString([]byte("exact text"))})
	var out bytes.Buffer
	args := append(append([]string{}, command.Path...), "--offline")
	if err := runPluginCommand(&out, io.Discard, strings.NewReader(""), globalOptions{Output: "raw", Language: "en-US"}, args, root, coreconfig.Profile{}, regionTestCredentials, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "exact text" {
		t.Fatal(out.String())
	}
	if err := runPluginCommand(io.Discard, io.Discard, strings.NewReader(""), globalOptions{Output: "raw", Language: "en-US", Seen: map[string]string{"cols": "--cols"}}, args, root, coreconfig.Profile{}, regionTestCredentials, nil); err == nil {
		t.Fatal("incompatible controls accepted")
	}
	if got := completeArgs(append(append([]string{}, command.Path...), "--output", ""), root); len(got) != 3 {
		t.Fatal(got)
	}
}

// TestTransportPollingAndRequestFailures exercises reloads and preparation failures without live services.
func TestTransportPollingAndRequestFailures(t *testing.T) {
	bundle, command := transportFixture(t, "json", `{"state":"pending"}`, 200)
	operation := bundle.APIs.Operations["show"]
	operation.Response.Variants[0].Headers = nil
	operation.Request = &apicontract.Request{Encoding: "form"}
	bundle.APIs.Operations["show"] = operation
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			req.Body.Close()
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"state":"ready"}`))}, nil
	})
	if _, err := executeAPICommand(bundle, command, nil, nil, coreconfig.Profile{}, regionTestCredentials, transport, io.Discard, nil, "en-US"); err != nil {
		t.Fatal(err)
	}
	bundle.Waiters = plugin.Waiters{Waiters: map[string]plugin.Waiter{"ready": {Path: "state", Success: "ready", MaxAttempts: 2}}}
	var pollOutput bytes.Buffer
	if err := runTransportCommand(io.Discard, &pollOutput, bundle, command, nil, nil, globalOptions{Output: "json", Fixture: true, Waiter: "ready", Language: "en-US"}, coreconfig.Profile{}, regionTestCredentials, nil, nil, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(pollOutput.String()), "timeout") {
		t.Fatal(pollOutput.String())
	}
	operation.Request = &apicontract.Request{Encoding: "xml", Document: "$param.doc"}
	bundle.APIs.Operations["show"] = operation
	if _, err := buildAPIRequest(bundle, command, nil, map[string]string{"doc": "malformed"}, coreconfig.Profile{}, regionTestCredentials, transport, io.Discard, nil, "en-US"); err == nil {
		t.Fatal("malformed XML prepared")
	}
	if err := os.WriteFile(filepath.Join(bundle.Dir, command.FixtureResponse), []byte("malformed fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runTransportCommand(io.Discard, io.Discard, bundle, command, nil, nil, globalOptions{Output: "json", Fixture: true}, coreconfig.Profile{}, regionTestCredentials, nil, nil, ""); err == nil {
		t.Fatal("malformed fixture accepted")
	}
	if got := configExplainValue(coreconfig.Setting{Value: "configured", Configured: true, Effective: false}, "en-US"); !strings.Contains(got, "configured") || got == "configured" {
		t.Fatal(got)
	}
}
