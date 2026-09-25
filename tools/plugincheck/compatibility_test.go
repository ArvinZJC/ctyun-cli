/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestTypedRequestBodyPluginsRequireCore040 prevents plugins from advertising
// compatibility with cores that serialize every command option as a JSON string.
func TestTypedRequestBodyPluginsRequireCore040(t *testing.T) {
	for _, pluginDir := range pluginDirs(t, repoPath(t, "plugins")) {
		bundle, err := plugin.LoadBundle(pluginDir, version.Version)
		if err != nil {
			t.Fatalf("load plugin %s with current core: %v", filepath.Base(pluginDir), err)
		}
		if !usesTypedRequestBody(bundle) {
			continue
		}
		t.Run(bundle.Manifest.Name, func(t *testing.T) {
			if _, err := plugin.LoadBundle(pluginDir, "0.3.999"); err == nil {
				t.Errorf("plugin with typed request-body options accepts a pre-0.4.0 core")
			}
		})
	}
}

// usesTypedRequestBody reports whether a bundle relies on core conversion of
// command-line text into non-string JSON request-body values.
func usesTypedRequestBody(bundle plugin.Bundle) bool {
	for _, command := range bundle.Commands.Commands {
		operation := bundle.APIs.Operations[command.Operation]
		if len(operation.Body) == 0 {
			continue
		}
		for _, parameter := range command.Parameters {
			if parameter.Target != "" && parameter.ValueType != "" && parameter.ValueType != plugin.ParameterValueString {
				return true
			}
		}
	}
	return false
}

// TestIndependentRequestBindingsRequireCore051 rejects a core that lets body
// values overwrite separately documented header or query values.
func TestIndependentRequestBindingsRequireCore051(t *testing.T) {
	for _, name := range []string{"yunxiao", "model-training", "dts", "cloud-audit"} {
		if _, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), "0.5.0"); err == nil {
			t.Errorf("%s accepts a core without independent request binding resolution", name)
		}
		if _, err := plugin.LoadBundle(repoPath(t, "plugins/"+name), version.Version); err != nil {
			t.Errorf("%s rejects current core: %v", name, err)
		}
	}
}
