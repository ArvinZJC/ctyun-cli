/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

func TestClassifyProbeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want FailureCategory
	}{
		{name: "cancelled", err: context.Canceled, want: FailureCancelled},
		{name: "timeout", err: context.DeadlineExceeded, want: FailureTimeout},
		{name: "dns", err: &net.DNSError{Err: "missing", Name: "example.test"}, want: FailureDNS},
		{name: "unknown authority", err: x509.UnknownAuthorityError{}, want: FailureTLS},
		{name: "invalid certificate", err: x509.CertificateInvalidError{}, want: FailureTLS},
		{name: "record header", err: tls.RecordHeaderError{}, want: FailureTLS},
		{name: "redirect", err: distribution.ErrUnsafeRedirect, want: FailureRedirect},
		{name: "proxy", err: errors.New("proxy"), want: FailureProxy},
		{name: "index", err: errors.New("index"), want: FailureIndex},
		{name: "signature", err: errors.New("signature"), want: FailureSignature},
		{name: "configuration", err: errors.New("configuration"), want: FailureConfiguration},
		{name: "connection", err: errors.New("connection refused"), want: FailureConnection},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			check := Check{Kind: CheckHTTPS}
			switch test.name {
			case "proxy":
				check = Check{Kind: CheckRoute, Role: "proxy"}
			case "index":
				check.Kind = CheckIndex
			case "signature":
				check.Kind = CheckVerification
			case "configuration":
				check.Kind = CheckConfiguration
			}
			if got := classifyProbeError(test.err, check); got != test.want {
				t.Fatalf("category = %q, want %q", got, test.want)
			}
		})
	}
}

func TestProbeErrorsAndRedirectSafety(t *testing.T) {
	dependencies := defaultDependencies(Dependencies{Resolver: resolverFunc(func(context.Context, string) ([]string, error) { return nil, nil }), Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "redirect.example.test" {
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"http://unsafe.example.test"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return response(http.StatusOK, nil), nil
	})})
	if err := probeRoute(context.Background(), Check{RequestURL: "://bad"}, dependencies); err == nil {
		t.Fatal("probeRoute accepted an invalid URL")
	}
	if err := probeHTTPS(context.Background(), Check{RequestURL: "://bad"}, dependencies); err == nil {
		t.Fatal("probeHTTPS accepted an invalid URL")
	}
	if err := probeHTTPS(context.Background(), Check{RequestURL: "https://redirect.example.test"}, dependencies); !errors.Is(err, distribution.ErrUnsafeRedirect) {
		t.Fatalf("probeHTTPS redirect error = %v, want ErrUnsafeRedirect", err)
	}

	redirects := 0
	dependencies.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		redirects++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://loop.example.test"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if err := probeHTTPS(context.Background(), Check{RequestURL: "https://loop.example.test"}, dependencies); !errors.Is(err, distribution.ErrUnsafeRedirect) || redirects < 10 {
		t.Fatalf("redirect error = %v, redirects = %d", err, redirects)
	}

	dependencies.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: failingProbeBody{readErr: errors.New("read")}}, nil
	})
	if err := probeHTTPS(context.Background(), Check{RequestURL: "https://read.example.test"}, dependencies); err == nil {
		t.Fatal("probeHTTPS ignored a body read error")
	}
	dependencies.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: failingProbeBody{closeErr: errors.New("close")}}, nil
	})
	if err := probeHTTPS(context.Background(), Check{RequestURL: "https://close.example.test"}, dependencies); err == nil {
		t.Fatal("probeHTTPS ignored a body close error")
	}
}

func TestExecuteCheckRejectsMissingPublicKey(t *testing.T) {
	dependencies := defaultDependencies(Dependencies{})
	output := executeCheck(context.Background(), Check{ID: "key", Kind: CheckConfiguration}, Plan{}, dependencies, nil)
	if output.result.Status != StatusFailed || output.result.Category != FailurePublicKey {
		t.Fatalf("result = %#v", output.result)
	}
}

func TestRunTimesOutSlowProbe(t *testing.T) {
	plan := Plan{Checks: []Check{{ID: "https", Kind: CheckHTTPS, Subject: "ctyun", Sequence: 0, Target: "https://slow.example.test", RequestURL: "https://slow.example.test"}}}
	report, err := Run(context.Background(), plan, time.Millisecond, Dependencies{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 || report.Results[0].Category != FailureTimeout {
		t.Fatalf("report = %#v", report)
	}
}

type failingProbeBody struct {
	readErr  error
	closeErr error
}

func (body failingProbeBody) Read([]byte) (int, error) {
	if body.readErr != nil {
		return 0, body.readErr
	}
	return 0, io.EOF
}

func (body failingProbeBody) Close() error {
	return body.closeErr
}
