/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestCloudAssistantExecutionTimeoutIsDistinctFromHTTPTimeout verifies command
// execution deadlines remain separate from the shared HTTP timeout option.
func TestCloudAssistantExecutionTimeoutIsDistinctFromHTTPTimeout(t *testing.T) {
	context := loadStorageReviewContext(t, "cloud-assistant")
	defaults := map[string]string{
		"v4.cloud-assistant.command.create": "60",
		"v4.cloud-assistant.command.invoke": "",
		"v4.cloud-assistant.command.modify": "",
		"v4.cloud-assistant.command.run":    "60",
	}
	for operationID, wantDefault := range defaults {
		parameter := context.commandParameter(operationID, "timeout")
		if parameter.Name != "execution_timeout" || parameter.Flag != "execution-timeout" || parameter.Default != wantDefault {
			t.Errorf("%s execution timeout = %s/%s default %q, want execution_timeout/execution-timeout default %q", operationID, parameter.Name, parameter.Flag, parameter.Default, wantDefault)
		}
	}

	var stdout bytes.Buffer
	if err := cli.Run(cli.Config{
		Args:       []string{"--lang", "en-US", "help", "cloud-assistant", "command", "create"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("render Cloud Assistant command help: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "--execution-timeout <value>                   Command execution timeout in seconds (default: 60)") {
		t.Fatalf("Cloud Assistant execution-timeout help missing:\n%s", got)
	}
	if strings.Count(got, "--timeout") != 1 {
		t.Fatalf("Cloud Assistant help contains an ambiguous --timeout option:\n%s", got)
	}
}
