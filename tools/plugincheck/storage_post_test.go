/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */
package plugincheck

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
	"github.com/ArvinZJC/ctyun-cli/internal/signing"
)

// postWireTransport validates independent gateway and policy signatures on the encoded upload.
type postWireTransport struct {
	t      *testing.T
	status int
	policy string
	calls  int
}

// RoundTrip consumes one multipart upload and returns an isolated protocol response.
func (transport *postWireTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls++
	defer func() {
		if closeErr := request.Body.Close(); closeErr != nil {
			transport.t.Error(closeErr)
		}
	}()
	if request.URL.Path != "/v2/testBucket" || request.Header.Get("ctyun-eop-ak") != "gateway-ak" || !strings.HasPrefix(request.Header.Get("Eop-Authorization"), "gateway-ak ") {
		transport.t.Fatal(request.URL, request.Header)
	}
	_, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	reader := multipart.NewReader(request.Body, params["boundary"])
	values := map[string]string{}
	last := ""
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}
		last = part.FormName()
		values[last] = string(data)
	}
	if last != "file" || values["file"] != "\x00\xffpayload" || values["x-amz-meta-location"] != "中文" || values["AWSAccessKeyId"] != "storage-ak" || values["Signature"] != signing.SignPOSTPolicyV2(transport.policy, "storage-sk") || values["success-action-status"] != "201" || values["x-amz-security-token"] != "temporary-token" {
		transport.t.Fatal("incorrect multipart fields")
	}
	headers := http.Header{"Location": {"https://example.test/result"}}
	body := ""
	if transport.status == 403 {
		body = `{"message":"temporary-token","x-amz-security-token":"temporary-token","policy":"` + transport.policy + `"}`
	}
	if transport.status == 201 {
		body = `<PostResponse><Bucket>testBucket</Bucket><Key>object</Key><Location>https://example.test/object</Location></PostResponse>`
	}
	return &http.Response{StatusCode: transport.status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}, nil
}

// TestMediaPOSTWireContract verifies every success representation without a live upload.
func TestMediaPOSTWireContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upload.bin")
	if err := os.WriteFile(path, []byte("\x00\xffpayload"), 0600); err != nil {
		t.Fatal(err)
	}
	policy := base64.StdEncoding.EncodeToString([]byte(`{"expiration":"2030-01-01T00:00:00Z","conditions":[]}`))
	for _, status := range []int{200, 201, 204, 303, 403} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			transport := &postWireTransport{t: t, status: status, policy: policy}
			var out, debug bytes.Buffer
			args := []string{"--yes", "--debug", "--lang", "en-US", "media-storage", "object", "post", "testBucket", "--key", "object", "--file", path, "--metadata", `{"location":"中文"}`, "--storage-access-key", "storage-ak", "--policy", policy, "--success-action-status", "201", "--security-token", "temporary-token"}
			err := cli.Run(cli.Config{Args: args, Stdout: &out, Stderr: &debug, PluginRoot: t.TempDir(), HTTPTransport: transport, Env: func(key string) string {
				return map[string]string{"CTYUN_AK": "gateway-ak", "CTYUN_SK": "gateway-sk", "CTYUN_STORAGE_SK": "storage-sk"}[key]
			}})
			if (err != nil) != (status == 403) || transport.calls != 1 {
				t.Fatal(err, transport.calls)
			}
			want := strconv.Itoa(status)
			if status != 403 && !strings.Contains(out.String(), want) {
				t.Fatal(out.String())
			}
			for _, secret := range []string{"storage-sk", "gateway-sk", "storage-ak", "temporary-token", policy} {
				if strings.Contains(debug.String(), secret) || err != nil && strings.Contains(err.Error(), secret) {
					t.Fatal("debug leaked a credential")
				}
			}
		})
	}
}
