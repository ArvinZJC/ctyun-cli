/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import "testing"

func TestValidateCommandParametersRejectsInvalidTypedDefault(t *testing.T) {
	command := Command{
		ID: "demo.create",
		Parameters: []Parameter{{
			Name:      "count",
			Flag:      "count",
			Target:    "count",
			ValueType: ParameterValueInteger,
			Default:   "three",
		}},
	}

	err := validateCommandParameters(command)
	if err == nil {
		t.Fatal("validateCommandParameters returned nil error")
	}
	requireDiagnosticKey(t, err, "error.command_parameter_invalid_default")
}

func TestValidateCommandParametersRejectsDefaultOutsideAllowedValues(t *testing.T) {
	command := Command{
		ID: "demo.create",
		Parameters: []Parameter{{
			Name:          "mode",
			Flag:          "mode",
			Target:        "mode",
			AllowedValues: []string{"CrossAZ", "SingleAZ"},
			Default:       "crossaz",
		}},
	}

	err := validateCommandParameters(command)
	if err == nil {
		t.Fatal("validateCommandParameters returned nil error")
	}
	requireDiagnosticKey(t, err, "error.command_parameter_default_disallowed")
}

func TestValidateCommandParametersAcceptsValidDefault(t *testing.T) {
	command := Command{
		ID: "demo.create",
		Parameters: []Parameter{{
			Name:          "enabled",
			Flag:          "enabled",
			Target:        "enabled",
			ValueType:     ParameterValueBoolean,
			AllowedValues: []string{"true", "false"},
			Default:       "false",
		}},
	}

	if err := validateCommandParameters(command); err != nil {
		t.Fatalf("validateCommandParameters returned error: %v", err)
	}
}
