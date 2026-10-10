/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestCRSGenerationSuccessIsolation prevents one API generation's success field
// or value from silently making a different generation's error look successful.
func TestCRSGenerationSuccessIsolation(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/crs"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, good, bad string }{
		{"crs.instance.show", `{"statusCode":800,"returnObj":{}}`, `{"statusCode":0,"returnObj":{}}`},
		{"crs.v1.instance.show", `{"statusCode":0,"returnObj":{}}`, `{"statusCode":800,"returnObj":{}}`},
		{"crs.legacy.instance.show", `{"code":0,"data":{}}`, `{"statusCode":800,"data":{}}`},
	} {
		op, ok := b.APIs.Operations[tc.id]
		if !ok {
			t.Fatalf("missing %s", tc.id)
		}
		for i, body := range []string{tc.good, tc.bad} {
			resp := &client.HTTPResponse{Status: 200, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
			_, err := client.DecodeHTTPResponse(resp, client.RequestSpec{Method: op.Method, Response: op.Response})
			if (err == nil) != (i == 0) {
				t.Errorf("%s response %s: %v", tc.id, body, err)
			}
		}
	}
}

// TestCRSCredentialIssuanceSafety keeps password issuance and cleanup execution
// confirmed and non-retryable even when their API names resemble retrievals.
func TestCRSCredentialIssuanceSafety(t *testing.T) {
	b, err := plugin.LoadBundle(repoPath(t, "plugins/crs"), version.Version)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{"crs.temporary-password.create": false, "crs.v1.temporary-password.create": false, "crs.cleanup.execution.create": false}
	for _, c := range b.Commands.Commands {
		if _, ok := expected[c.ID]; !ok {
			continue
		}
		expected[c.ID] = true
		if b.APIs.Operations[c.Operation].Retryable || c.Dangerous.Confirm != "yes" {
			t.Errorf("unsafe operation %s", c.ID)
		}
	}
	for id, found := range expected {
		if !found {
			t.Errorf("missing %s", id)
		}
	}
}
