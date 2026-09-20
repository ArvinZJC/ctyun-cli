/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"testing"
)

// TestTransferOptionsAreCommandOwned tests command capability based parsing.
func TestTransferOptionsAreCommandOwned(t *testing.T) {
	command := plugin.Command{Download: true}
	values, err := parseCommandParameters(command, []string{"--output-file", "download.bin", "--overwrite"}, "en-US")
	if err != nil || values["__ctyun_output_file"] != "download.bin" || values["__ctyun_overwrite"] != "true" {
		t.Fatalf("%v %v", values, err)
	}
	if _, err := parseCommandParameters(plugin.Command{}, []string{"--output-file", "download.bin"}, "en-US"); err == nil {
		t.Fatal("unrelated command accepted transfer flags")
	}
}
