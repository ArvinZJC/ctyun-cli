/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestTypedRequestBodyPluginsRequireCore040 prevents plugins from advertising
// compatibility with cores that serialize every command option as a JSON string.
func TestTypedRequestBodyPluginsRequireCore040(t *testing.T) {
	for _, pluginDir := range pluginDirs(t, repoPath(t, "plugins")) {
		bundle, err := plugin.LoadBundle(pluginDir, "0.4.0")
		if err != nil {
			t.Fatalf("load plugin %s with core 0.4.0: %v", filepath.Base(pluginDir), err)
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
