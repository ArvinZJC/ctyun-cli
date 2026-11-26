/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestHelpFlagAcceptsCommandInvocations checks help routing without executing the supplied command.
func TestHelpFlagAcceptsCommandInvocations(t *testing.T) {
	root := t.TempDir()
	writeValidationBundle(t, filepath.Join(root, "ecs"))
	writeArgumentRegionOptionBundle(t, filepath.Join(root, "region"))
	transport := &waiterRejectTransport{}
	configPath := filepath.Join(t.TempDir(), "config.json")
	for _, tc := range []struct{ path, arguments []string }{
		{[]string{"plugin", "list"}, []string{"--available"}},
		{[]string{"plugins", "install"}, []string{"sample", "--channel", "beta", "second"}},
		{[]string{"plugin", "reinstall"}, []string{"--channel=stable"}},
		{[]string{"plugin", "remove"}, []string{"one", "two"}},
		{[]string{"plugins", "search"}, []string{"--source", "auto"}},
		{[]string{"plugin", "upgrade"}, []string{"sample", "--channel", "alpha"}},
		{[]string{"completion"}, []string{"bash"}},
		{[]string{"upgrade"}, []string{"--check", "--source", "auto"}},
		{[]string{"doctor", "network"}, []string{"--source", "github"}},
		{[]string{"config", "set"}, []string{"region", "use1"}},
		{[]string{"config", "profile", "use"}, []string{"prod"}},
		{[]string{"config", "profiles", "set-secret"}, []string{"prod", "ak", "--from-stdin"}},
		{[]string{"ecs", "instance", "list"}, []string{"--status", "running", "--name=demo"}},
		{[]string{"region", "show"}, []string{"explicit-region"}},
	} {
		t.Run(strings.Join(tc.path, "_"), func(t *testing.T) {
			invoke := func(args []string) string {
				var out, stderr bytes.Buffer
				err := Run(Config{Args: append([]string{"--lang", "en-US"}, args...), Config: []byte("{}"), PluginRoot: root, Stdout: &out, Stderr: &stderr, Stdin: strings.NewReader(""), ConfigPath: configPath, HTTPTransport: transport})
				if err != nil || stderr.Len() != 0 {
					t.Fatalf("args=%q stdout=%q stderr=%q error=%v", args, out.String(), stderr.String(), err)
				}
				return out.String()
			}
			want := invoke(append([]string{"help"}, tc.path...))
			path := append(slices.Clone(tc.path), tc.arguments...)
			for _, pos := range []int{0, len(tc.path), len(path)} {
				args := slices.Insert(slices.Clone(path), pos, "--help")
				if got := invoke(args); got != want || !strings.Contains(got, "Usage:") {
					t.Fatalf("args=%q: incorrect help page", args)
				}
			}
		})
	}
	if transport.calls != 0 {
		t.Fatalf("help sent %d requests", transport.calls)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("help changed configuration: %v", err)
	}
}

// TestHelpFlagChecksSuppliedInputs keeps malformed options, surplus operands and path-only help diagnostics.
func TestHelpFlagChecksSuppliedInputs(t *testing.T) {
	root := t.TempDir()
	writeValidationBundle(t, filepath.Join(root, "ecs"))
	for _, tc := range []struct {
		args []string
		key  string
	}{
		{[]string{"plugin", "list", "--available", "extra", "--help"}, "error.unexpected_argument"},
		{[]string{"plugin", "update", "one", "two", "--help"}, "error.unexpected_argument"},
		{[]string{"plugin", "install", "--channel", "--help"}, "error.option_requires_value"},
		{[]string{"plugin", "list", "--available=false", "--help"}, "error.unknown_option"},
		{[]string{"plugin", "search", "--source", "invalid", "--help"}, "error.unsupported_source"},
		{[]string{"plugin", "install", "--channel", "invalid", "--help"}, "error.unsupported_channel"},
		{[]string{"completion", "unknown", "--help"}, "error.unsupported_shell"},
		{[]string{"completion", "bash", "extra", "--help"}, "error.unexpected_argument"},
		{[]string{"ecs", "instance", "list", "--status", "running", "--unknown", "--help"}, "error.unknown_option"},
		{[]string{"config", "path", "extra", "--help"}, "error.unexpected_argument"},
		{[]string{"config", "profile", "use", "name", "extra", "--help"}, "error.unexpected_argument"},
		{[]string{"config", "--other", "profile", "--help"}, "error.unknown_option"},
		{[]string{"ecs", "instance", "list", "--status", "running", "bad", "--help"}, "error.unexpected_argument"},
		{[]string{"ecs", "instance", "list", "--status", "--help"}, "error.option_requires_value"},
		{[]string{"help", "plugin", "list", "--available"}, "error.unknown_option"},
		{[]string{"help", "ecs", "instance", "list", "--name", "data"}, "error.unknown_option"},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			err := Run(Config{Args: tc.args, Config: []byte("{}"), PluginRoot: root, Stdout: io.Discard, Stderr: io.Discard})
			requireDiagnosticKey(t, err, tc.key)
		})
	}
}

// TestProductHelpFlagValidatesOnlySuppliedInputs exercises transport options, pending requirements and build-specific controls.
func TestProductHelpFlagValidatesOnlySuppliedInputs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ims")
	writeArgumentBundle(t, dir)
	mustWrite(t, filepath.Join(dir, "commands.json"), `{"commands":[{"id":"ims.image.show","path":["ims","image","show","{image_id}"],"operation":"v4.ims.image.show","table":"ims.image.show","download":true,"dangerous":{"confirm":"yes"},"parameters":[{"name":"mode","flag":"mode","target":"mode","required":true,"allowed_values":["active","all"]}]}]}`)
	mustWrite(t, filepath.Join(dir, "apis.json"), `{"operations":{"v4.ims.image.show":{"method":"GET","path":"/v4/ims/image/{image_id}","query":{"mode":"$param.mode"},"response":{"variants":[{"status":200,"format":"binary"}]}}}}`)
	dest := filepath.Join(t.TempDir(), "download")
	transport := &waiterRejectTransport{}
	invoke := func(args ...string) (string, error) {
		var stdout bytes.Buffer
		err := Run(Config{Args: append([]string{"--lang", "en-US"}, args...), Config: []byte("{}"), PluginRoot: root, Stdout: &stdout, Stderr: io.Discard, HTTPTransport: transport})
		return stdout.String(), err
	}
	path := []string{"ims", "image", "show", "--option-shaped-id"}
	for _, rest := range [][]string{{"--output-file", dest, "--overwrite", "--help"}, {"--help", "--output-file=" + dest}, {"--help", "--fixture"}} {
		out, err := invoke(append(slices.Clone(path), rest...)...)
		if err != nil || !strings.Contains(out, "--output-file") || strings.Contains(out, "--fixture") {
			t.Fatalf("help %v: %q, %v", rest, out, err)
		}
	}
	if out, err := invoke("plugin", "install", "ims", "--bundled", "--help"); err != nil || strings.Contains(out, "--bundled") {
		t.Fatalf("development plugin help: %q, %v", out, err)
	}
	if _, err := invoke(append(slices.Clone(path), "--mode", "invalid", "--help")...); err == nil || !strings.Contains(err.Error(), "active,all") {
		t.Fatalf("supplied enum: %v", err)
	}
	if _, err := invoke(append(slices.Clone(path), "--output-file", "--help")...); err == nil {
		t.Fatal("missing transfer option value accepted")
	}
	t.Cleanup(patchVersion(testCoreVersion()))
	for _, args := range [][]string{append(slices.Clone(path), "--fixture", "--help"), {"plugin", "install", "--bundled", "--help"}, {"ims", "--fixture", "image", "show", "id", "--help"}} {
		_, err := invoke(args...)
		requireDiagnosticKey(t, err, "error.unknown_option")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) || transport.calls != 0 {
		t.Fatalf("help created output or sent requests: %v, requests=%d", err, transport.calls)
	}
}
