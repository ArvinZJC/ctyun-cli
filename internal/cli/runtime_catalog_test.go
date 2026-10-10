/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
)

// TestRuntimePluginColumnsUseMessageCatalog keeps runtime table labels outside help-only copy.
func TestRuntimePluginColumnsUseMessageCatalog(t *testing.T) {
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		for _, column := range availablePluginColumns(language) {
			key := "plugin.column." + column.Key
			if _, exists := helpCatalog[key]; exists {
				t.Fatalf("runtime table label %q is in helpCatalog", key)
			}
			if got := messageText(key, language); got == key || got != column.Label {
				t.Fatalf("runtime label %q = %q, table label = %q", key, got, column.Label)
			}
		}
	}
}

// TestRedirectDiagnosticLocalizesThroughHTTPWrapper covers the error wrapping used by net/http.
func TestRedirectDiagnosticLocalizesThroughHTTPWrapper(t *testing.T) {
	err := &url.Error{Op: "Get", URL: "https://example.test", Err: distribution.ErrUnsafeRedirect}
	if got := localizedError(err, "zh-CN"); got != "HTTP 重定向不安全或次数过多" {
		t.Fatalf("redirect error = %q, want localized safety diagnostic", got)
	}
}

// TestRuntimeMessagesQuoteUserValues prevents names and paths from creating misleading output lines.
func TestRuntimeMessagesQuoteUserValues(t *testing.T) {
	value := "name\n\"with quotes\""
	quoted := fmt.Sprintf("%q", value)
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		for _, key := range []string{"config.active_profile", "config.profile_reset", "config.reset", "error.plugin_root_not_directory", "error.plugin_not_installed"} {
			got := messagef(key, language, value)
			if !strings.Contains(got, quoted) || strings.Contains(got, "\n") {
				t.Errorf("%s %s did not quote the user value: %q", language, key, got)
			}
		}
		for _, key := range []string{"warning.config_credentials", "warning.deprecated_command"} {
			got := messageText(key, language)
			if !strings.Contains(got, "\n  ctyun config set ") {
				t.Errorf("%s %s does not put its suggested command on a separate line", language, key)
			}
		}
	}
}
