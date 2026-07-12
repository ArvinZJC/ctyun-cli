/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

func TestRunVerifiesSignedSourcesAndCTyunReachability(t *testing.T) {
	plan, dependencies := signedTestPlan(t, nil)
	var progressIDs []string
	var observing atomic.Bool
	report, err := Run(context.Background(), plan, time.Second, dependencies, func(progress Progress) error {
		if !observing.CompareAndSwap(false, true) {
			t.Fatal("observer called concurrently")
		}
		defer observing.Store(false)
		progressIDs = append(progressIDs, progress.Check.ID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.FailedCapabilities) != 0 || report.Counts.Failed != 0 {
		t.Fatalf("report = %#v", report)
	}
	if len(progressIDs) != len(plan.Checks) {
		t.Fatalf("progress events = %d, want %d", len(progressIDs), len(plan.Checks))
	}
	want := []struct {
		kind       CheckKind
		subject    string
		sourceName string
		target     string
	}{
		{kind: CheckSource, subject: "core", sourceName: "github", target: "https://github.example.test"},
		{kind: CheckSource, subject: "core", sourceName: "gitee", target: "https://gitee.example.test"},
		{kind: CheckEndpoint, subject: "ctyun", target: "https://ctapi.example.test"},
	}
	if len(report.Results) != len(want) {
		t.Fatalf("visible results = %#v, want %d rows", report.Results, len(want))
	}
	for index, expected := range want {
		check := report.Results[index].Check
		if check.Kind != expected.kind || check.Subject != expected.subject || check.SourceName != expected.sourceName || check.Target != expected.target {
			t.Fatalf("result %d check = %#v, want %#v", index, check, expected)
		}
		if report.Results[index].DetailKey != "doctor.detail."+string(expected.kind)+"_passed" {
			t.Fatalf("result %d detail = %q", index, report.Results[index].DetailKey)
		}
	}
}

func TestRunFoldsMissingPublicKeyIntoVisibleSourceResults(t *testing.T) {
	plan, dependencies := signedTestPlan(t, nil)
	plan.PublicKey = ""
	report, err := Run(context.Background(), plan, time.Second, dependencies, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 3 || report.Counts != (Counts{Passed: 1, Failed: 2}) {
		t.Fatalf("report = %#v", report)
	}
	for _, result := range report.Results[:2] {
		if result.Check.Kind != CheckSource || result.Status != StatusFailed || result.Category != FailurePublicKey || result.DetailKey != "doctor.detail.public_key" {
			t.Fatalf("source result = %#v", result)
		}
	}
	if report.Results[2].Check.Kind != CheckEndpoint || report.Results[2].Check.Target != "https://ctapi.example.test" {
		t.Fatalf("endpoint result = %#v", report.Results[2])
	}
}

func TestRunShowsOneVisibleResultPerUniqueCTyunOrigin(t *testing.T) {
	basePlan, dependencies := signedTestPlan(t, nil)
	plan, err := Build(Input{
		CTyunEndpoints: []string{
			"https://first.ctapi.example.test/v1",
			"https://second.ctapi.example.test/v2",
			"https://first.ctapi.example.test/other",
		},
		PublicKey: basePlan.PublicKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), plan, time.Second, dependencies, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantTargets := []string{"https://first.ctapi.example.test", "https://second.ctapi.example.test"}
	if len(report.Results) != len(wantTargets) {
		t.Fatalf("results = %#v", report.Results)
	}
	for index, want := range wantTargets {
		if result := report.Results[index]; result.Check.Kind != CheckEndpoint || result.Check.Target != want || result.Status != StatusPassed {
			t.Fatalf("result %d = %#v, want endpoint %q", index, result, want)
		}
	}
}

func TestRunTreatsWorkingFallbackAsWarning(t *testing.T) {
	plan, dependencies := signedTestPlan(t, func(req *http.Request) error {
		if req.URL.Host == "github.example.test" {
			return errors.New("github unavailable")
		}
		return nil
	})
	report, err := Run(context.Background(), plan, time.Second, dependencies, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.FailedCapabilities) != 0 || report.Counts != (Counts{Passed: 2, Warning: 1}) {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunFailsCapabilityWhenEveryMirrorFails(t *testing.T) {
	plan, dependencies := signedTestPlan(t, func(req *http.Request) error {
		if strings.Contains(req.URL.Host, "example.test") && req.URL.Host != "ctapi.example.test" {
			return errors.New("source unavailable")
		}
		return nil
	})
	report, err := Run(context.Background(), plan, 100*time.Millisecond, dependencies, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(report.FailedCapabilities, "core-source") || report.Counts.Failed == 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunUsesProxyRouteWithoutResolvingDestination(t *testing.T) {
	proxyURL, err := url.Parse("https://proxy.example.test:8443")
	if err != nil {
		t.Fatal(err)
	}
	resolved := make(map[string]int)
	plan, dependencies := signedTestPlan(t, nil)
	plan, err = Build(Input{
		Sources:        []SourceInput{{Capability: "core-source", Subject: "core", Source: distribution.Source{Name: "github", URL: "https://github.example.test/core"}, IndexName: "core-index.json", SignatureName: "core-index.sig"}},
		CTyunEndpoints: []string{"https://ctapi.example.test"},
		PublicKey:      plan.PublicKey,
		Proxy: func(*http.Request) (*url.URL, error) {
			return proxyURL, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	dependencies.Proxy = func(*http.Request) (*url.URL, error) { return proxyURL, nil }
	dependencies.Resolver = resolverFunc(func(_ context.Context, host string) ([]string, error) {
		resolved[host]++
		if host != "proxy.example.test" {
			return nil, errors.New("destination DNS unavailable")
		}
		return []string{"192.0.2.10"}, nil
	})
	report, err := Run(context.Background(), plan, time.Second, dependencies, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.Failed != 0 || resolved["github.example.test"] != 0 || resolved["ctapi.example.test"] != 0 || resolved["proxy.example.test"] != 1 {
		t.Fatalf("report = %#v, resolved = %#v", report, resolved)
	}
}

func TestRunPropagatesObserverErrorAndCancellation(t *testing.T) {
	plan, dependencies := signedTestPlan(t, nil)
	want := errors.New("progress writer")
	if _, err := Run(context.Background(), plan, time.Second, dependencies, func(Progress) error { return want }); !errors.Is(err, want) {
		t.Fatalf("observer error = %v, want %v", err, want)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := Run(ctx, plan, time.Second, dependencies, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.Failed == 0 && report.Counts.Skipped == 0 {
		t.Fatalf("cancelled report = %#v", report)
	}
}

func TestRunCoversDefaultsSkippedObserverAndDependencyCycle(t *testing.T) {
	report, err := Run(context.Background(), Plan{}, 0, Dependencies{}, nil)
	if err != nil || len(report.Results) != 0 {
		t.Fatalf("empty run = %#v, %v", report, err)
	}
	want := errors.New("skipped observer")
	plan := Plan{PublicKey: "", Checks: []Check{
		{ID: "key", Kind: CheckConfiguration, Sequence: 0},
		{ID: "dependent", Kind: CheckHTTPS, Sequence: 1, DependsOn: []string{"key"}},
	}}
	events := 0
	_, err = Run(context.Background(), plan, time.Second, Dependencies{}, func(Progress) error {
		events++
		if events == 2 {
			return want
		}
		return nil
	})
	if !errors.Is(err, want) {
		t.Fatalf("observer error = %v, want %v", err, want)
	}
	cycle := Plan{Checks: []Check{{ID: "cycle", Kind: CheckHTTPS, DependsOn: []string{"missing"}}}}
	if _, err := Run(context.Background(), cycle, time.Second, Dependencies{}, nil); err == nil {
		t.Fatal("Run accepted an unsatisfied dependency cycle")
	}
}

func signedTestPlan(t *testing.T, failure func(*http.Request) error) (Plan, Dependencies) {
	t.Helper()
	index := []byte(`{"schema":1}`)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	signature := []byte(base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, index)))
	plan, err := Build(Input{
		Sources: []SourceInput{{
			Capability: "core-source", Subject: "core",
			Source:    distribution.Source{Name: "github", URL: "https://github.example.test/core", Fallbacks: []distribution.Source{{Name: "gitee", URL: "https://gitee.example.test/core"}}},
			IndexName: "core-index.json", SignatureName: "core-index.sig",
		}},
		CTyunEndpoints: []string{"https://ctapi.example.test"},
		PublicKey:      base64.StdEncoding.EncodeToString(publicKey),
	})
	if err != nil {
		t.Fatal(err)
	}
	dependencies := Dependencies{
		Resolver: resolverFunc(func(context.Context, string) ([]string, error) { return []string{"192.0.2.1"}, nil }),
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if failure != nil {
				if err := failure(req); err != nil {
					return nil, err
				}
			}
			if req.Method == http.MethodHead {
				return response(http.StatusNotFound, nil), nil
			}
			if strings.HasSuffix(req.URL.Path, ".sig") {
				return response(http.StatusOK, signature), nil
			}
			return response(http.StatusOK, index), nil
		}),
	}
	return plan, dependencies
}

type resolverFunc func(context.Context, string) ([]string, error)

func (resolver resolverFunc) LookupHost(ctx context.Context, host string) ([]string, error) {
	return resolver(ctx, host)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return roundTrip(req)
}

func response(status int, body []byte) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}
}
