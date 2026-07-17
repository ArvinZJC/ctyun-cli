/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"slices"
	"strings"
	"unicode"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// descriptionQualityFinding returns one objective reason that an English
// command description is not ready for generated plugin help.
func descriptionQualityFinding(catalog Catalog, operation Operation, command plugin.Command, description string) string {
	words := descriptionWords(description)
	actionWords := descriptionWords(commandAction(operation))
	mechanical := append(actionWords, descriptionWords(catalog.Product.APIProduct)...)
	mechanical = append(mechanical, descriptionWords(operation.Category)...)
	if slices.Equal(foldDescriptionWords(words), foldDescriptionWords(mechanical)) {
		return "matches the old mechanical action/product/category fallback"
	}
	pathWords := descriptionWords(strings.Join(command.Path, " "))
	if slices.Equal(foldDescriptionWords(words), foldDescriptionWords(pathWords)) {
		return "is a title-cased command path"
	}
	if strings.Contains(description, "/v4/") || strings.Contains(description, "/global-trust-authority/") {
		return "contains a raw API URI"
	}
	for _, word := range words {
		if slices.Contains([]string{"Vm", "Cpu", "Gpu", "Dns", "Id", "Ipv6", "Jwt", "Tpm", "Tee", "Vpc", "Vtpm"}, word) {
			return "uses non-conventional acronym casing"
		}
	}
	return ""
}

// descriptionWords extracts Unicode letter and number words while preserving
// their original casing for acronym checks.
func descriptionWords(value string) []string {
	return strings.FieldsFunc(value, func(char rune) bool {
		return !unicode.IsLetter(char) && !unicode.IsDigit(char)
	})
}

// foldDescriptionWords lowercases words for exact mechanical-form comparison.
func foldDescriptionWords(words []string) []string {
	folded := make([]string, len(words))
	for index, word := range words {
		folded[index] = strings.ToLower(word)
	}
	return folded
}

// operationMissingExampleEvidence returns required source inputs whose
// placeholder has not been backed by concrete evidence or an explicit reviewed
// absence.
func operationMissingExampleEvidence(operation Operation, command plugin.Command) []string {
	required := make(map[string]bool)
	for _, parameter := range command.Parameters {
		if parameter.Required {
			required[parameter.Name] = true
		}
	}
	var missing []string
	for _, parameter := range operation.Parameters {
		if parameter.Profile != "" {
			continue
		}
		name := parameter.CLIName
		if parameter.Argument != "" {
			name = parameter.Argument
			required[name] = parameter.Required
		}
		if name == "" || !required[name] || parameterHasExampleEvidence(operation, parameter) {
			continue
		}
		missing = append(missing, name)
	}
	return missing
}

// parameterHasExampleEvidence reports whether one required source input has a
// concrete value source or an explicit reviewed unavailable marker.
func parameterHasExampleEvidence(operation Operation, parameter Parameter) bool {
	if parameter.ExampleUnavailable || len(parameter.Example) > 0 || parameter.Default != "" || len(parameter.Enum) > 0 {
		return true
	}
	if _, ok := sourceExampleText(parameter, operation.RequestExample[parameter.Name]); ok {
		return true
	}
	if parameter.Argument != "" && exampleArgumentValues(operation)[parameter.Argument] != "" {
		return true
	}
	return false
}
