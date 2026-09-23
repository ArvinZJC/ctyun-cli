/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package plugincheck

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/client"
)

// nativeWireTransport provides a per-command request assertion seam without live storage calls.
type nativeWireTransport func(*http.Request) (*http.Response, error)

// RoundTrip executes the test's wire assertions and returns its captured response.
func (f nativeWireTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestNativeStorageCommandsOnWire checks every native binding through the shipped command engine.
func TestNativeStorageCommandsOnWire(t *testing.T) {
	for _, name := range []string{"media-storage", "classic-object-storage"} {
		context := loadStorageReviewContext(t, name)
		count := 0
		for _, command := range context.bundle.Commands.Commands {
			operation := context.bundle.APIs.Operations[command.Operation]
			if operation.Native == nil {
				continue
			}
			count++
			t.Run(command.ID, func(t *testing.T) {
				args := commandSmokeArgs(t, command)
				for i, part := range command.Path {
					if strings.HasPrefix(part, "{") {
						if strings.Contains(part, "bucket") {
							args[i] = "test-bucket"
						} else {
							args[i] = "a//../中 %+?"
						}
					}
				}
				document := filepath.Join(t.TempDir(), "body.xml")
				if err := os.WriteFile(document, []byte("<Request/>"), 0600); err != nil {
					t.Fatal(err)
				}
				for _, parameter := range command.Parameters {
					if parameter.Input == "file" {
						for i, arg := range args {
							if arg == "--"+parameter.Flag {
								args[i+1] = document
							}
						}
					}
				}
				if operation.Native.ContentType != "" {
					args = append(args, "--content-type", "text/plain; charset=utf-8")
				}
				if operation.Native.Metadata != "" {
					args = append(args, "--metadata", `{"colour":"blue"}`)
				}
				args = append([]string{"--yes", "--lang", "en-US"}, args...)
				if command.Download {
					args = append(args, "--output", "raw")
				} else {
					args = append(args, "--output", "json")
				}
				fixture, err := os.ReadFile(repoPath(t, filepath.Join("plugins", name, command.FixtureResponse)))
				if err != nil {
					t.Fatal(err)
				}
				calls := 0
				transport := nativeWireTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Header.Get("Eop-Authorization") != "" || r.Header.Get("Eop-date") != "" || r.Header.Get("ctyun-eop-request-id") != "" {
						t.Fatal("EOP headers reached native service")
					}
					if operation.Native.ContentType != "" && r.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
						t.Fatal("MIME override lost", r.Header)
					}
					if operation.Native.Metadata != "" && r.Header.Get("X-Amz-Meta-Colour") != "blue" {
						t.Fatal("metadata map lost", r.Header)
					}
					wantHost := "storage.example"
					if operation.Native.Addressing == "virtual" && operation.Native.Bucket != "" {
						wantHost = "test-bucket." + wantHost
					}
					if r.URL.Host != wantHost || r.Method != operation.Method {
						t.Fatal(r.URL, r.Method)
					}
					if operation.Native.Object != "" && !strings.HasSuffix(r.URL.EscapedPath(), "/a//../%E4%B8%AD%20%25%2B%3F") {
						t.Fatal("object identity changed", r.URL)
					}
					for _, key := range operation.Native.Subresources {
						if !r.URL.Query().Has(key) {
							t.Fatal("missing native subresource", key)
						}
					}
					var body []byte
					if r.Body != nil {
						body, err = io.ReadAll(r.Body)
						if closeErr := r.Body.Close(); closeErr != nil {
							t.Error(closeErr)
						}
						if err != nil {
							t.Fatal(err)
						}
					}
					if !operation.Native.PolicyAuth {
						hash := sha256.Sum256(body)
						if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=native-ak/") || r.Header.Get("X-Amz-Content-Sha256") != hex.EncodeToString(hash[:]) {
							t.Fatal("incorrect body signing")
						}
					} else if r.Header.Get("Authorization") != "" {
						t.Fatal("policy POST has header authorization")
					}
					response, err := client.DecodeFixture(fixture)
					if err != nil {
						return nil, err
					}
					return &http.Response{StatusCode: response.Status, Header: response.Headers, Body: response.Body}, nil
				})
				var out bytes.Buffer
				err = cli.Run(cli.Config{Args: args, Stdout: &out, Stderr: io.Discard, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(k string) string {
					return map[string]string{"CTYUN_AK": "eop-ak", "CTYUN_SK": "eop-sk", "CTYUN_STORAGE_AK": "native-ak", "CTYUN_STORAGE_SK": "native-sk", "CTYUN_STORAGE_ENDPOINT": "https://storage.example", "CTYUN_STORAGE_REGION": "cn"}[k]
				}})
				if err != nil || calls != 1 {
					t.Fatalf("calls %d: %v; args %v", calls, err, args)
				}
			})
		}
		var inventory struct {
			Included int `json:"included_count"`
		}
		data, err := os.ReadFile(repoPath(t, "openapi-catalogs/"+name+"/native-inventory.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &inventory); err != nil {
			t.Fatal(err)
		}
		if count != inventory.Included {
			t.Fatal("native inventory count mismatch", count, inventory.Included)
		}
	}
}

// TestNativeClassicRequestLocations checks native URL/header/body semantics independently of table headings.
func TestNativeClassicRequestLocations(t *testing.T) {
	context := loadStorageReviewContext(t, "classic-object-storage")
	for _, slug := range []string{"bucket-logging.set", "bucket-cors.set", "multipart-upload.complete"} {
		op := context.bundle.APIs.Operations["classic-object-storage.native."+slug]
		want := 0
		if slug == "multipart-upload.complete" {
			want = 1
		}
		if len(op.Query) != want || op.Request == nil || op.Request.Encoding != "xml" {
			t.Fatalf("%s misplaced XML fields: %#v", slug, op.Query)
		}
	}
	create := context.bundle.APIs.Operations["classic-object-storage.native.bucket.create"]
	if create.Headers["x-amz-acl"] == "" || create.Query["x-amz-acl"] != "" {
		t.Fatal("ACL is not a header", create)
	}
	for _, verb := range []string{"set", "show", "delete", "list"} {
		op := context.bundle.APIs.Operations["classic-object-storage.native.bucket-inventory-configuration."+verb]
		key := "id"
		if verb == "list" {
			key = "continuation-token"
		}
		if op.Query[key] == "" || op.Headers[key] != "" {
			t.Fatalf("%s inventory query misplaced: %#v %#v", verb, op.Query, op.Headers)
		}
	}
}
