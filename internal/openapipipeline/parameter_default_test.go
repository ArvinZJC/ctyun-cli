/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import "testing"

func TestBuildCommandsCarriesParameterDefaultExactly(t *testing.T) {
	catalog := loadCatalogFixture(t)
	parameter := &catalog.Operations[0].Parameters[1]
	parameter.Default = "CrossAZ"

	commands := buildCommands(catalog)
	for _, generated := range commands.Commands[0].Parameters {
		if generated.Name != parameter.CLIName {
			continue
		}
		if generated.Default != "CrossAZ" {
			t.Fatalf("generated parameter default = %q, want exact source token", generated.Default)
		}
		return
	}
	t.Fatalf("generated parameters = %#v, want %q", commands.Commands[0].Parameters, parameter.CLIName)
}
