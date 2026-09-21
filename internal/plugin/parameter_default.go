/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

// ParameterValueOrDefault resolves a conditional selector from explicit input
// or its declared API default without adding default values to the request.
// An explicitly supplied empty value is preserved rather than replaced.
func ParameterValueOrDefault(name string, parameters []Parameter, values map[string]string) string {
	if value, present := values[name]; present {
		return value
	}
	for _, parameter := range parameters {
		if parameter.Name == name {
			return parameter.Default
		}
	}
	return ""
}
