/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

func TestBuildExpandsMirrorsAndDeduplicatesRoutes(t *testing.T) {
	coreSource := distribution.Source{
		Name: "github",
		URL:  "https://shared.example.test/core",
		Fallbacks: []distribution.Source{
			{Name: "gitee", URL: "https://gitee.example.test/core"},
		},
	}
	pluginSource := distribution.Source{Name: "github", URL: "https://shared.example.test/plugins"}
	plan, err := Build(Input{
		Sources: []SourceInput{
			{Capability: "core-source", Subject: "core", Source: coreSource, IndexName: "core-index.json", SignatureName: "core-index.sig"},
			{Capability: "plugin-source", Subject: "plugin", Source: pluginSource, IndexName: "index.json", SignatureName: "index.sig"},
		},
		CTyunEndpoints: []string{"https://ctapi.example.test", "https://ctapi.example.test/"},
		PublicKey:      "public-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := countChecks(plan.Checks, CheckRoute); got != 3 {
		t.Fatalf("route checks = %d, want 3", got)
	}
	if got := countChecks(plan.Checks, CheckVerification); got != 3 {
		t.Fatalf("verification checks = %d, want 3", got)
	}
	if got := countSubjectChecks(plan.Checks, CheckHTTPS, "ctyun"); got != 1 {
		t.Fatalf("CTyun HTTPS checks = %d, want 1", got)
	}
	for index, check := range plan.Checks {
		if check.Sequence != index {
			t.Fatalf("check %q sequence = %d, want %d", check.ID, check.Sequence, index)
		}
		if check.ID == "" {
			t.Fatal("plan contains an empty check ID")
		}
	}
}

func TestBuildSanitizesProxyAndEndpointTargets(t *testing.T) {
	proxyURL, err := url.Parse("https://user:secret@proxy.example.test:8443/path?token=secret#fragment")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := Build(Input{
		Sources: []SourceInput{{
			Capability: "core-source",
			Subject:    "core",
			Source:     distribution.Source{Name: "github", URL: "https://source.example.test/root?token=source-secret"},
			IndexName:  "core-index.json", SignatureName: "core-index.sig",
		}},
		CTyunEndpoints: []string{"https://api-user:api-secret@ctapi.example.test/root?token=api-secret#fragment"},
		PublicKey:      "public-key",
		Proxy: func(*http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range plan.Checks {
		for _, secret := range []string{"user", "secret", "token=", "/path", "fragment", "api-user"} {
			if strings.Contains(check.Target, secret) {
				t.Fatalf("check %q target leaks %q: %q", check.ID, secret, check.Target)
			}
		}
	}
	if got := countChecks(plan.Checks, CheckRoute); got != 1 {
		t.Fatalf("proxied route checks = %d, want 1", got)
	}
	for _, check := range plan.Checks {
		if check.Kind == CheckRoute && check.Role != "proxy" {
			t.Fatalf("proxy route role = %q", check.Role)
		}
	}
}

func TestBuildRejectsUnsafeEndpoints(t *testing.T) {
	for _, endpoint := range []string{"http://ctapi.example.test", "://bad", "file:///tmp/socket"} {
		_, err := Build(Input{CTyunEndpoints: []string{endpoint}})
		if err == nil {
			t.Fatalf("Build accepted endpoint %q", endpoint)
		}
	}
}

func TestBuildRejectsUnsafeSourcesAndProxyFailures(t *testing.T) {
	_, err := Build(Input{Sources: []SourceInput{{Capability: "core", Subject: "core", Source: distribution.Source{Name: "bad", URL: "http://source.example.test"}}}})
	if err == nil {
		t.Fatal("Build accepted an unsafe source")
	}
	proxyErr := func(*http.Request) (*url.URL, error) { return nil, errors.New("proxy configuration") }
	_, err = Build(Input{Sources: []SourceInput{{Capability: "core", Subject: "core", Source: distribution.Source{Name: "github", URL: "https://source.example.test"}}}, Proxy: proxyErr})
	if err == nil {
		t.Fatal("Build ignored a source proxy error")
	}
	_, err = Build(Input{CTyunEndpoints: []string{"https://ctapi.example.test"}, Proxy: proxyErr})
	if err == nil {
		t.Fatal("Build ignored a CTyun proxy error")
	}
	_, err = Build(Input{CTyunEndpoints: []string{"https://ctapi.example.test"}, Proxy: func(*http.Request) (*url.URL, error) {
		return &url.URL{Scheme: "http", Host: "proxy.example.test"}, nil
	}})
	if err == nil {
		t.Fatal("Build accepted an unsafe proxy URL")
	}
}

func TestBuildPreservesSafeEscapedSourcePathsAndReusesHTTPS(t *testing.T) {
	plan, err := Build(Input{Sources: []SourceInput{
		{Capability: "core-a", Subject: "core", Source: distribution.Source{Name: "first", URL: "https://source.example.test/root%20space"}, IndexName: "index.json", SignatureName: "index.sig"},
		{Capability: "core-b", Subject: "core", Source: distribution.Source{Name: "second", URL: "https://source.example.test/root%20space"}, IndexName: "other.json", SignatureName: "other.sig"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := countSubjectChecks(plan.Checks, CheckHTTPS, "core"); got != 1 {
		t.Fatalf("HTTPS checks = %d, want 1", got)
	}
	for _, check := range plan.Checks {
		if check.Kind == CheckIndex && !strings.Contains(check.RequestURL, "/root%20space/") {
			t.Fatalf("index URL lost escaped path: %q", check.RequestURL)
		}
	}
}

func TestPlanBuilderRejectsInvalidRouteRequest(t *testing.T) {
	builder := planBuilder{input: Input{}, routeIDs: make(map[string]string), httpsIDs: make(map[string]string), capabilities: make(map[string][]string)}
	if _, err := builder.ensureRoute("://bad", "core"); err == nil {
		t.Fatal("ensureRoute accepted an invalid request URL")
	}
	if id := builder.ensureHTTPS("https://source.example.test", "https://source.example.test", "core", "route"); id == "" {
		t.Fatal("ensureHTTPS returned an empty ID")
	}
	if id := builder.ensureHTTPS("https://source.example.test", "https://source.example.test", "core", "route"); id != "https-0" {
		t.Fatalf("duplicate HTTPS ID = %q", id)
	}
	if id := builder.ensureHTTPSForSubject("https://source.example.test", "https://source.example.test", "core", "route"); id != "https-0" {
		t.Fatalf("direct duplicate HTTPS ID = %q", id)
	}
}

func countChecks(checks []Check, kind CheckKind) int {
	count := 0
	for _, check := range checks {
		if check.Kind == kind {
			count++
		}
	}
	return count
}

func countSubjectChecks(checks []Check, kind CheckKind, subject string) int {
	count := 0
	for _, check := range checks {
		if check.Kind == kind && check.Subject == subject {
			count++
		}
	}
	return count
}
