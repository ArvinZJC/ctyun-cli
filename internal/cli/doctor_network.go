/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/networkdoctor"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/release"
)

const (
	// defaultDoctorNetworkTimeout is the per-check deadline when --timeout is absent.
	defaultDoctorNetworkTimeout = 5 * time.Second
)

// doctorNetworkOptions contains command-owned doctor network options.
type doctorNetworkOptions struct {
	Source string
}

// runNetworkDoctor is a deterministic runner seam for CLI tests.
var runNetworkDoctor = networkdoctor.Run

// doctorNetworkDependenciesFactory creates proxy-aware diagnostic dependencies.
var doctorNetworkDependenciesFactory = func(transport http.RoundTripper) networkdoctor.Dependencies {
	return networkdoctor.Dependencies{Transport: transport, Proxy: http.ProxyFromEnvironment}
}

// writeDoctorNetworkReport is a deterministic reporter seam for CLI tests.
var writeDoctorNetworkReport = renderDoctorNetworkReport

// runDoctor executes supported core diagnostics.
func runDoctor(stdout, stderr io.Writer, args []string, installedRoot string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, global globalOptions) error {
	if len(args) == 0 {
		_, err := printDoctorHelp(stdout, []string{"doctor"}, global.Language)
		return err
	}
	opts, err := parseDoctorNetworkOptions(args)
	if err != nil {
		return err
	}
	coreSource, err := distribution.ResolveSource(distribution.SourceOptions{
		Label:     "core",
		Requested: opts.Source,
		EnvName:   "CTYUN_UPGRADE_SOURCE",
		GitHubURL: release.GitHubReleaseSource,
		GiteeURL:  release.GiteeReleaseSource,
		Getenv:    getenv,
	})
	if err != nil {
		return err
	}
	pluginSource, err := distribution.ResolveSource(distribution.SourceOptions{
		Label:     "plugin",
		Requested: opts.Source,
		EnvName:   "CTYUN_PLUGIN_SOURCE",
		GitHubURL: registry.GitHubPluginSource,
		GiteeURL:  registry.GiteePluginSource,
		Getenv:    getenv,
	})
	if err != nil {
		return err
	}
	endpoints, err := resolveDoctorEndpointURLs(installedRoot, profile)
	if err != nil {
		return err
	}
	dependencies := doctorNetworkDependenciesFactory(transport)
	plan, err := networkdoctor.Build(networkdoctor.Input{
		Sources: []networkdoctor.SourceInput{
			{Capability: "core-source", Subject: "core", Source: coreSource, IndexName: "core-index.json", SignatureName: "core-index.sig"},
			{Capability: "plugin-source", Subject: "plugin", Source: pluginSource, IndexName: "index.json", SignatureName: "index.sig"},
		},
		CTyunEndpoints: endpoints,
		PublicKey:      releasePublicKey(getenv),
		Proxy:          dependencies.Proxy,
	})
	if err != nil {
		return err
	}
	timeout := defaultDoctorNetworkTimeout
	if global.Timeout > 0 {
		timeout = time.Duration(global.Timeout) * time.Second
	}
	display := operationProgressFactory(stderr)
	if err := display.Start(len(plan.Checks)); err != nil {
		return err
	}
	observer := func(progress networkdoctor.Progress) error {
		return updateOperationProgress(display, progress.Completed, doctorProgressText(progress.Check, global.Language))
	}
	report, err := runNetworkDoctor(context.Background(), plan, timeout, dependencies, observer)
	if err != nil {
		_ = display.Clear()
		return err
	}
	if err := display.Clear(); err != nil {
		return err
	}
	if err := writeDoctorNetworkReport(stdout, report, global); err != nil {
		return err
	}
	if len(report.FailedCapabilities) > 0 {
		return silentExitError{}
	}
	return nil
}

// parseDoctorNetworkOptions parses the network topic and its owned options.
func parseDoctorNetworkOptions(args []string) (doctorNetworkOptions, error) {
	var opts doctorNetworkOptions
	if len(args) == 0 {
		return opts, nil
	}
	if args[0] != "network" {
		return opts, commandBoundaryError(append([]string{"doctor"}, args...))
	}
	parsed, err := parseCommandTokens(args[1:], []commandOption{{Name: "source", TakesValue: true}})
	if err != nil {
		return opts, err
	}
	if err := rejectUnexpectedPositionals(parsed.Positionals, 0); err != nil {
		return opts, err
	}
	opts.Source = parsed.Options["source"]
	if opts.Source != "" && opts.Source != "auto" && opts.Source != "github" && opts.Source != "gitee" {
		return opts, diagnostic.New("error.unsupported_source", "doctor", opts.Source)
	}
	return opts, nil
}

// resolveDoctorEndpointURLs loads available plugin endpoints when the selected
// profile does not override the API endpoint.
func resolveDoctorEndpointURLs(installedRoot string, profile coreconfig.Profile) ([]string, error) {
	if strings.TrimSpace(profile.EndpointURL) != "" {
		return doctorEndpointURLs(profile, nil), nil
	}
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return nil, diagnostic.New("error.doctor_plugin_metadata")
	}
	return doctorEndpointURLs(profile, bundles), nil
}

// doctorEndpointURLs applies profile, plugin, and canonical endpoint precedence.
func doctorEndpointURLs(profile coreconfig.Profile, bundles []plugin.Bundle) []string {
	if endpoint := normalizeDoctorEndpoint(profile.EndpointURL); endpoint != "" {
		return []string{endpoint}
	}
	seen := make(map[string]bool)
	endpoints := make([]string, 0, len(bundles))
	for _, bundle := range bundles {
		endpoint := normalizeDoctorEndpoint(bundle.Manifest.API.EndpointURL)
		if endpoint == "" || seen[endpoint] {
			continue
		}
		seen[endpoint] = true
		endpoints = append(endpoints, endpoint)
	}
	if len(endpoints) == 0 {
		return nil
	}
	slices.Sort(endpoints)
	return endpoints
}

// normalizeDoctorEndpoint trims whitespace and redundant trailing separators.
func normalizeDoctorEndpoint(endpoint string) string {
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}
