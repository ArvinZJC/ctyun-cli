/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import "testing"

// TestParseScopedOptionValues verifies finite-value option parsing.
func TestParseScopedOptionValues(t *testing.T) {
	if _, _, err := parseGlobalOptions([]string{"--output", "yaml"}); err == nil {
		t.Fatal("parseGlobalOptions returned nil error for bad output")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_output")
	}
	if _, _, err := parseGlobalOptions([]string{"--table", "grid"}); err == nil {
		t.Fatal("parseGlobalOptions returned nil error for bad table style")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_table_style")
	}
	if opts, _, err := parseGlobalOptions([]string{"--output", "json", "--table", "compact", "version"}); err != nil || opts.Output != "json" || opts.Table != "compact" {
		t.Fatalf("parseGlobalOptions scoped values = %+v, %v", opts, err)
	}
	if _, err := parseUpgradeOptions([]string{"--source", "bad"}); err == nil {
		t.Fatal("parseUpgradeOptions returned nil error for bad source")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_source")
	}
	if _, err := parseUpgradeOptions([]string{"--channel", "all"}); err == nil {
		t.Fatal("parseUpgradeOptions returned nil error for all channel")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_channel")
	}
	if opts, err := parseUpgradeOptions([]string{"--source", "gitee", "--channel", "beta"}); err != nil || opts.Source != "gitee" || opts.Channel != "beta" {
		t.Fatalf("parseUpgradeOptions scoped values = %+v, %v", opts, err)
	}
}

// TestResolvePluginSourceRejectsBadEnvSource covers env-based source errors.
func TestResolvePluginSourceRejectsBadEnvSource(t *testing.T) {
	getenv := func(key string) string {
		if key == "CTYUN_PLUGIN_SOURCE" {
			return "bad"
		}
		return ""
	}
	if _, err := resolvePluginSource("", getenv); err == nil {
		t.Fatal("resolvePluginSource returned nil error for bad env source")
	} else {
		requireDiagnosticKey(t, err, "error.unsupported_source")
	}
}
