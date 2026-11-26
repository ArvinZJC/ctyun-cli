/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package networkdoctor

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDistinctEndpointFailuresAffectExitStatus keeps dotted and hyphenated hosts independent.
func TestDistinctEndpointFailuresAffectExitStatus(t *testing.T) {
	endpoints := []string{"https://api-east.example.test", "https://api.east.example.test"}
	plan, err := Build(Input{CTyunEndpoints: endpoints})
	if err != nil {
		t.Fatal(err)
	}
	for _, failed := range endpoints {
		_, dependencies := signedTestPlan(t, func(req *http.Request) error {
			if "https://"+req.URL.Host == failed {
				return errors.New("endpoint unavailable")
			}
			return nil
		})
		report, err := Run(t.Context(), plan, time.Second, dependencies, nil)
		if err != nil {
			t.Fatal(err)
		}
		if report.Counts != (Counts{Passed: 1, Failed: 1}) || len(report.FailedCapabilities) != 1 {
			t.Errorf("failed endpoint %s: counts %+v, failed capabilities %v", failed, report.Counts, report.FailedCapabilities)
		}
	}
}

// TestEndpointCertificateNameMismatchReportsTLS exercises real certificate verification on loopback.
func TestEndpointCertificateNameMismatchReportsTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(server.Close)
	transport := server.Client().Transport.(*http.Transport)
	transport.TLSClientConfig.ServerName = "wrong-host.example.test"
	plan, err := Build(Input{CTyunEndpoints: []string{server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(t.Context(), plan, 5*time.Second, Dependencies{Transport: transport}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := report.Results[0]
	if result.Status != StatusFailed || result.Category != FailureTLS || result.DetailKey != "doctor.detail.tls" {
		t.Fatalf("certificate mismatch: status %s, category %q, detail %q, error %v", result.Status, result.Category, result.DetailKey, result.Err)
	}
}
