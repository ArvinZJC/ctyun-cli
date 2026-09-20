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
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// storageWireTransport checks the real promoted form contract without contacting a service.
type storageWireTransport struct {
	t     *testing.T
	calls int
}

// RoundTrip verifies one JSON serialization inside form encoding, then returns documented success.
func (transport *storageWireTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls++
	defer request.Body.Close()
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, err
	}
	var statement []map[string]any
	if err := json.Unmarshal([]byte(values.Get("statement")), &statement); err != nil {
		transport.t.Fatal(err)
	}
	if len(statement) != 1 || statement[0]["Resource"] != "arn:aws:s3:::bucket/a&b+中文" || values.Get("regionCode") != "0001" {
		transport.t.Fatal(values)
	}
	if request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || request.Header.Get("Eop-Authorization") == "" {
		transport.t.Fatal("missing encoding or signature")
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"statusCode":"0","returnObj":{"code":"0"}}`))}, nil
}

// TestMediaRefererWireEncoding checks CLI typing, metadata binding, form bytes and signing together.
func TestMediaRefererWireEncoding(t *testing.T) {
	transport := &storageWireTransport{t: t}
	err := cli.Run(cli.Config{Args: []string{"--yes", "--output", "json", "media-storage", "bucket-referer", "set", "--region-code", "0001", "--bucket-name", "bucket", "--statement", `[{"Effect":"Allow","Resource":"arn:aws:s3:::bucket/a&b+中文"}]`}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(key string) string {
		if key == "CTYUN_AK" {
			return "test-ak"
		}
		if key == "CTYUN_SK" {
			return "test-sk"
		}
		return ""
	}})
	if err != nil || transport.calls != 1 {
		t.Fatalf("calls=%d error=%v", transport.calls, err)
	}
}

// TestStorageGapEvidenceAndBinaryFixture keeps synthetic bytes and documentation corrections explicit.
func TestStorageGapEvidenceAndBinaryFixture(t *testing.T) {
	for _, name := range []string{"media-storage", "classic-object-storage"} {
		var inventory struct {
			IncludedCount int `json:"included_count"`
			Operations    []struct {
				ID       string         `json:"api_id"`
				Included bool           `json:"included"`
				Review   map[string]any `json:"evidence_review"`
			} `json:"operations"`
		}
		data, err := os.ReadFile(repoPath(t, "openapi-catalogs/"+name+"/coverage.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(data, &inventory); err != nil {
			t.Fatal(err)
		}
		bundle, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), version.Version)
		if err != nil {
			t.Fatal(err)
		}
		if len(bundle.Commands.Commands) != inventory.IncludedCount {
			t.Fatal("coverage count mismatch")
		}
		for _, op := range inventory.Operations {
			if !op.Included {
				t.Fatalf("unexpected remaining gap %s %s", name, op.ID)
			}
			if op.Review != nil && op.Review["live_verified"] != false {
				t.Fatal("inference presented as live verification")
			}
			if op.ID == "4862" && op.Review["fixture_basis"] == nil {
				t.Fatal("synthetic fixture basis missing")
			}
		}
	}
	var out bytes.Buffer
	if err := cli.Run(cli.Config{Args: []string{"media-storage", "object", "show", "bucket", "object", "--offline", "--output", "raw"}, Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), []byte{0, 255, 'C', 'T', 'Y', 'U', 'N', '\n'}) {
		t.Fatal("binary fixture changed")
	}
	path := filepath.Join(t.TempDir(), "download")
	if err := cli.Run(cli.Config{Args: []string{"media-storage", "object", "show", "bucket", "object", "--offline", "--output-file", path}, Stdout: io.Discard, Stderr: io.Discard, PluginRoot: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, out.Bytes()) {
		t.Fatal("file download changed bytes", err)
	}
}
