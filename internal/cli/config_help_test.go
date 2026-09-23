/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigSetHelpListsSupportedGlobalAndProfileKeys(t *testing.T) {
	var stdout bytes.Buffer
	handled, err := printCoreHelp(&stdout, []string{"config", "set"}, "en-US")
	if err != nil || !handled {
		t.Fatalf("config set help = handled %t, error %v", handled, err)
	}
	output := stdout.String()
	for _, want := range []string{
		"Global Keys:",
		"active_profile {name}",
		"warn_config_credentials {true|false}",
		"Profile Keys:",
		"region {id}",
		"language {zh-CN|en-US|en-GB}",
		"endpoint_url {url}",
		"default: true",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("config set help missing %q:\n%s", want, output)
		}
	}
}

func TestConfigProfileSetHelpExplainsAdvancedEndpointOverride(t *testing.T) {
	var stdout bytes.Buffer
	handled, err := printCoreHelp(&stdout, []string{"config", "profile", "set"}, "en-GB")
	if err != nil || !handled {
		t.Fatalf("config profile set help = handled %t, error %v", handled, err)
	}
	output := stdout.String()
	if strings.Contains(output, "Global Keys:") {
		t.Fatalf("profile-only help rendered global keys:\n%s", output)
	}
	for _, want := range []string{"Profile Keys:", "endpoint_url {url}", "Advanced profile-wide API endpoint override for testing or private environments"} {
		if !strings.Contains(output, want) {
			t.Fatalf("config profile set help missing %q:\n%s", want, output)
		}
	}
}

func TestConfigUnsetHelpListsKeyNamesWithoutValues(t *testing.T) {
	var stdout bytes.Buffer
	handled, err := printCoreHelp(&stdout, []string{"config", "unset"}, "en-US")
	if err != nil || !handled {
		t.Fatalf("config unset help = handled %t, error %v", handled, err)
	}
	output := stdout.String()
	for _, want := range []string{"Global Keys:", "  active_profile", "Profile Keys:", "  endpoint_url"} {
		if !strings.Contains(output, want) {
			t.Fatalf("config unset help missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "active_profile {name}") || strings.Contains(output, "endpoint_url {url}") {
		t.Fatalf("config unset help rendered value placeholders:\n%s", output)
	}
}

func TestConfigKeyHelpIsLocalized(t *testing.T) {
	var stdout bytes.Buffer
	handled, err := printCoreHelp(&stdout, []string{"config", "profile", "set"}, "zh-CN")
	if err != nil || !handled {
		t.Fatalf("localized config profile set help = handled %t, error %v", handled, err)
	}
	output := stdout.String()
	for _, want := range []string{"配置档案键:", "高级配置档案级 API 终端节点覆盖，用于测试或私有环境", "（默认：true）"} {
		if !strings.Contains(output, want) {
			t.Fatalf("localized config key help missing %q:\n%s", want, output)
		}
	}
}
