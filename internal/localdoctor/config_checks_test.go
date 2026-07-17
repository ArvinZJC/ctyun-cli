/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package localdoctor

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
)

func TestConfigChecksHandleMissingZeroByteAndMalformedFiles(t *testing.T) {
	missing := Run(Input{ConfigPath: filepath.Join(t.TempDir(), "missing"), Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, missing, CheckConfigFile, doctor.StatusSkipped)
	assertFindingStatus(t, missing, CheckCredentials, doctor.StatusFailed)

	zero := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(zero, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	empty := Run(Input{ConfigPath: zero, Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, empty, CheckConfigFile, doctor.StatusPassed)

	malformed := Run(Input{InjectedConfig: []byte(`{"profiles":`), UseInjectedConfig: true, Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, malformed, CheckConfigFile, doctor.StatusFailed)
	assertFindingStatus(t, malformed, CheckProfile, doctor.StatusSkipped)
	assertFindingStatus(t, malformed, CheckCredentials, doctor.StatusSkipped)
}

func TestConfigChecksEvaluatePOSIXPermissionsButNotWindowsACLs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"ak":"stored-ak","sk":"stored-sk"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	deps := DefaultDependencies()
	deps.GOOS = "linux"
	posix := Run(Input{ConfigPath: path, Getenv: func(string) string { return "" }}, deps)
	assertFindingStatus(t, posix, CheckConfigFile, doctor.StatusWarning)

	deps.GOOS = "windows"
	windows := Run(Input{ConfigPath: path, Getenv: func(string) string { return "" }}, deps)
	assertFindingStatus(t, windows, CheckConfigFile, doctor.StatusPassed)
}

func TestConfigChecksResolveProfilesCredentialsRegionAndEndpoint(t *testing.T) {
	raw := []byte(`{"active_profile":"prod","profiles":{"prod":{"region":"region-1","endpoint_url":"https://example.test","ak":"stored-ak","sk":"stored-sk"}}}`)
	report := Run(Input{InjectedConfig: raw, UseInjectedConfig: true, Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, report, CheckProfile, doctor.StatusPassed)
	assertFindingStatus(t, report, CheckCredentials, doctor.StatusWarning)
	assertFindingStatus(t, report, CheckRegion, doctor.StatusPassed)
	assertFindingStatus(t, report, CheckEndpoint, doctor.StatusPassed)

	badEndpoint := Run(Input{InjectedConfig: []byte(`{"profiles":{"dev":{"endpoint_url":"http://example.test"}}}`), UseInjectedConfig: true, Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, badEndpoint, CheckEndpoint, doctor.StatusFailed)

	ambiguous := Run(Input{InjectedConfig: []byte(`{"profiles":{"a":{},"b":{}}}`), UseInjectedConfig: true, Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, ambiguous, CheckProfile, doctor.StatusFailed)
	assertFindingStatus(t, ambiguous, CheckRegion, doctor.StatusSkipped)
}

func TestConfigChecksNeverRetainCredentialValues(t *testing.T) {
	secrets := []string{"environment-ak", "environment-sk", "stored-ak", "stored-sk"}
	report := Run(Input{
		InjectedConfig: []byte(`{"ak":"stored-ak","sk":"stored-sk"}`), UseInjectedConfig: true,
		Getenv: func(key string) string {
			if key == "CTYUN_AK" {
				return secrets[0]
			}
			if key == "CTYUN_SK" {
				return secrets[1]
			}
			return ""
		},
	}, DefaultDependencies())
	for _, finding := range report.Findings {
		for _, arg := range finding.DetailArgs {
			value := strings.TrimSpace(arg.(string))
			for _, secret := range secrets {
				if strings.Contains(value, secret) {
					t.Fatalf("finding %#v retained secret %q", finding, secret)
				}
			}
		}
	}
}

func TestCredentialFindingsDistinguishStoredAndMixedSources(t *testing.T) {
	stored := Run(Input{
		InjectedConfig: []byte(`{"ak":"stored-ak","sk":"stored-sk"}`), UseInjectedConfig: true,
		Getenv: func(string) string { return "" },
	}, DefaultDependencies())
	storedFinding := findingByKey(t, stored, CheckCredentials)
	if storedFinding.DetailKey != "doctor.local.detail.credentials_stored" || len(storedFinding.DetailArgs) != 1 || storedFinding.DetailArgs[0] != "config" {
		t.Fatalf("stored credential finding = %#v", storedFinding)
	}

	mixed := Run(Input{
		InjectedConfig: []byte(`{"sk":"stored-sk"}`), UseInjectedConfig: true,
		Getenv: func(key string) string {
			if key == "CTYUN_AK" {
				return "environment-ak"
			}
			return ""
		},
	}, DefaultDependencies())
	mixedFinding := findingByKey(t, mixed, CheckCredentials)
	if mixedFinding.DetailKey != "doctor.local.detail.credentials_mixed" || len(mixedFinding.DetailArgs) != 2 {
		t.Fatalf("mixed credential finding = %#v", mixedFinding)
	}
}

func TestConfigChecksCoverUnavailableFilesystemStates(t *testing.T) {
	emptyPath := Run(Input{Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, emptyPath, CheckConfigFile, doctor.StatusFailed)

	statError := DefaultDependencies()
	statError.Stat = func(string) (fs.FileInfo, error) { return nil, errors.New("stat") }
	report := Run(Input{ConfigPath: "/config", Getenv: completeEnvironment}, statError)
	assertFindingStatus(t, report, CheckConfigFile, doctor.StatusFailed)
	assertFindingStatus(t, report, CheckCredentials, doctor.StatusPassed)

	nonRegular := Run(Input{ConfigPath: t.TempDir(), Getenv: func(string) string { return "" }}, DefaultDependencies())
	assertFindingStatus(t, nonRegular, CheckConfigFile, doctor.StatusFailed)

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	readError := DefaultDependencies()
	readError.ReadFile = func(string) ([]byte, error) { return nil, errors.New("read") }
	assertFindingStatus(t, Run(Input{ConfigPath: path, Getenv: func(string) string { return "" }}, readError), CheckConfigFile, doctor.StatusFailed)

	malformedPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(malformedPath, []byte(`{"profiles":`), 0o600); err != nil {
		t.Fatal(err)
	}
	assertFindingStatus(t, Run(Input{ConfigPath: malformedPath, Getenv: func(string) string { return "" }}, DefaultDependencies()), CheckConfigFile, doctor.StatusFailed)
}

func TestRunFillsOmittedReadOnlyDependencies(t *testing.T) {
	report := Run(Input{UseInjectedConfig: true, Getenv: completeEnvironment}, Dependencies{})
	assertFindingStatus(t, report, CheckConfigFile, doctor.StatusPassed)
}

// assertFindingStatus finds one stable check and verifies its status.
func assertFindingStatus(t *testing.T, report Report, key CheckKey, want doctor.Status) {
	t.Helper()
	finding := findingByKey(t, report, key)
	if finding.Status != want {
		t.Fatalf("finding %q status = %s, want %s: %#v", key, finding.Status.String(), want.String(), finding)
	}
}

// findingByKey returns the first finding for a stable check key.
func findingByKey(t *testing.T, report Report, key CheckKey) Finding {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Key == key {
			return finding
		}
	}
	t.Fatalf("finding %q not found in %#v", key, report.Findings)
	return Finding{}
}
