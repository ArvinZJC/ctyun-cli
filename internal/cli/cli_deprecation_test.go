/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

func TestPluginCommandWarnsForDeprecatedCommandOperationAndParameter(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDeprecatedWarningBundle(t, filepath.Join(pluginRoot, "demo"))
	transport := deprecatedWarningTransport()

	var stderr bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "en-US", "demo", "list", "--page", "1", "--cols", "id"},
		Stdout:        io.Discard,
		Stderr:        &stderr,
		PluginRoot:    pluginRoot,
		HTTPTransport: transport,
		Env:           envCredentialsForDeprecatedWarningTest,
		Config:        deprecatedWarningConfigForTest(false),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stderr.String()
	for _, want := range []string{
		"Warning: this plugin command is deprecated.",
		"Warning: the CTyun API used by this command is deprecated.",
		"Warning: option --page is deprecated.",
		"Recommended CLI replacement: ctyun demo new-list.",
		"ctyun config set warn_deprecated false",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"old command", "old API", "old page field", "newer demo command", "--page-no"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("stderr exposed upstream deprecation metadata %q:\n%s", unwanted, got)
		}
	}
}

func TestPluginCommandLocalizesDeprecatedWarning(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDeprecatedWarningBundle(t, filepath.Join(pluginRoot, "demo"))

	var stderr bytes.Buffer
	err := Run(Config{
		Args:          []string{"--lang", "zh-CN", "demo", "list", "--page", "1", "--cols", "id"},
		Stdout:        io.Discard,
		Stderr:        &stderr,
		PluginRoot:    pluginRoot,
		HTTPTransport: deprecatedWarningTransport(),
		Env:           envCredentialsForDeprecatedWarningTest,
		Config:        deprecatedWarningConfigForTest(false),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stderr.String()
	for _, want := range []string{
		"警告：此插件命令已弃用。",
		"警告：此命令使用的天翼云 API 已弃用。",
		"警告：选项 --page 已弃用。",
		"建议使用 CLI 替代项：ctyun demo new-list。",
		"ctyun config set warn_deprecated false",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"old command", "old API", "old page field", "newer demo command", "--page-no"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("stderr exposed upstream deprecation metadata %q:\n%s", unwanted, got)
		}
	}
}

func TestPluginCommandDoesNotWarnWhenDeprecatedWarningDisabled(t *testing.T) {
	cases := []struct {
		name   string
		env    func(string) string
		config []byte
	}{
		{
			name: "env",
			env: func(key string) string {
				if key == "CTYUN_WARN_DEPRECATED" {
					return "false"
				}
				return envCredentialsForDeprecatedWarningTest(key)
			},
			config: deprecatedWarningConfigForTest(false),
		},
		{
			name:   "config",
			env:    envCredentialsForDeprecatedWarningTest,
			config: deprecatedWarningConfigForTest(true),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pluginRoot := t.TempDir()
			writeDeprecatedWarningBundle(t, filepath.Join(pluginRoot, "demo"))

			var stderr bytes.Buffer
			err := Run(Config{
				Args:          []string{"--lang", "en-US", "demo", "list", "--page", "1", "--cols", "id"},
				Stdout:        io.Discard,
				Stderr:        &stderr,
				PluginRoot:    pluginRoot,
				HTTPTransport: deprecatedWarningTransport(),
				Env:           tc.env,
				Config:        tc.config,
			})
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if strings.Contains(stderr.String(), "deprecated") {
				t.Fatalf("stderr contained deprecated warning: %q", stderr.String())
			}
		})
	}
}

func TestPluginCommandWarnsForDeprecatedDisplayedColumn(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDeprecatedWarningBundle(t, filepath.Join(pluginRoot, "demo"))

	var stderr bytes.Buffer
	err := Run(Config{
		Args:       []string{"--lang", "en-US", "--offline", "demo", "list", "--cols", "Old Size"},
		Stdout:     io.Discard,
		Stderr:     &stderr,
		PluginRoot: pluginRoot,
		Env:        envCredentialsForDeprecatedWarningTest,
		Config:     deprecatedWarningConfigForTest(false),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Warning: output field Old Size is deprecated.") {
		t.Fatalf("stderr missing deprecated field warning: %q", stderr.String())
	}
}

func TestRunPluginCommandReturnsDeprecatedWarningWriteError(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDeprecatedWarningBundle(t, filepath.Join(pluginRoot, "demo"))

	err := runPluginCommand(
		io.Discard,
		failingWriter{},
		strings.NewReader(""),
		globalOptions{Language: "en-US", Output: "json", Offline: true},
		[]string{"demo", "list", "--page", "1"},
		pluginRoot,
		coreconfig.Profile{},
		envCredentialsForDeprecatedWarningTest,
		nil,
	)
	if err == nil {
		t.Fatal("runPluginCommand returned nil error")
	}
}

func TestRunPluginCommandReturnsTableRenderError(t *testing.T) {
	pluginRoot := t.TempDir()
	writeDeprecatedWarningBundle(t, filepath.Join(pluginRoot, "demo"))

	err := runPluginCommand(
		io.Discard,
		io.Discard,
		strings.NewReader(""),
		globalOptions{Language: "en-US", Output: "table", Table: "unknown", Offline: true},
		[]string{"demo", "list"},
		pluginRoot,
		coreconfig.Profile{WarnDeprecated: boolPtrForDeprecatedWarningTest(false)},
		envCredentialsForDeprecatedWarningTest,
		nil,
	)
	if err == nil {
		t.Fatal("runPluginCommand returned nil error")
	}
}

func TestDeprecatedWarningWriteErrors(t *testing.T) {
	deprecated := &plugin.Deprecation{Status: "deprecated", Notice: "old"}
	cases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "command",
			fn: func() error {
				return warnDeprecatedCommandUsage(
					failingWriter{},
					plugin.Bundle{},
					plugin.Command{Deprecation: deprecated},
					nil,
					envCredentialsForDeprecatedWarningTest,
					coreconfig.Profile{},
					"en-US",
				)
			},
		},
		{
			name: "api",
			fn: func() error {
				return warnDeprecatedCommandUsage(
					failingWriter{},
					plugin.Bundle{APIs: plugin.APIs{Operations: map[string]plugin.Operation{"demo": {Deprecation: deprecated}}}},
					plugin.Command{Operation: "demo"},
					nil,
					envCredentialsForDeprecatedWarningTest,
					coreconfig.Profile{},
					"en-US",
				)
			},
		},
		{
			name: "option",
			fn: func() error {
				return warnDeprecatedCommandUsage(
					failingWriter{},
					plugin.Bundle{},
					plugin.Command{Parameters: []plugin.Parameter{{Name: "page", Flag: "page", Deprecation: deprecated}}},
					map[string]string{"page": "1"},
					envCredentialsForDeprecatedWarningTest,
					coreconfig.Profile{},
					"en-US",
				)
			},
		},
		{
			name: "field",
			fn: func() error {
				return warnDeprecatedDisplayedColumns(
					failingWriter{},
					plugin.Table{Columns: []plugin.TableColumn{{Key: "old", Deprecation: deprecated}}},
					[]output.Column{{Key: "old", Label: "Old"}},
					[]string{"old"},
					envCredentialsForDeprecatedWarningTest,
					coreconfig.Profile{},
					"en-US",
				)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err == nil {
				t.Fatal("warning returned nil error")
			}
		})
	}
}

func TestDeprecationTextHelpersCoverEmptyAndCasingBranches(t *testing.T) {
	if got := localizedDeprecationDetails(nil, "en-US"); got != "" {
		t.Fatalf("nil deprecation details = %q, want empty", got)
	}
	if got := localizedDeprecationDetails(&plugin.Deprecation{Status: "deprecated", Notice: "old", Replacement: &plugin.Replacement{Kind: "parameter", Label: "pageNo"}}, "en-US"); got != "" {
		t.Fatalf("upstream replacement details = %q, want empty", got)
	}
	if got := localizedDeprecationDetails(&plugin.Deprecation{Status: "deprecated", Replacement: &plugin.Replacement{Kind: "option", Label: "--page-no"}}, "en-US"); !strings.Contains(got, "Recommended CLI replacement: --page-no.") {
		t.Fatalf("CLI option replacement details = %q", got)
	}
	if got := capitalizeHelpSentencePart("1 already sentence"); got != "1 already sentence" {
		t.Fatalf("capitalized numeric fragment = %q", got)
	}
	if got := capitalizeHelpSentencePart(""); got != "" {
		t.Fatalf("capitalized empty fragment = %q", got)
	}
	if got := localizedErrorText("parse warn_deprecated: invalid boolean", "en-US"); !strings.Contains(got, "warn_deprecated") {
		t.Fatalf("localized parse warning error = %q", got)
	}
	if !cliReplacementKind("command") || !cliReplacementKind("option") || cliReplacementKind("parameter") {
		t.Fatal("CLI replacement kind classification is incorrect")
	}
}

func deprecatedWarningTransport() http.RoundTripper {
	return roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"returnObj":{"items":[{"id":"one","oldSize":"1"}]}}`)),
		}, nil
	})
}

func boolPtrForDeprecatedWarningTest(value bool) *bool {
	return &value
}
