/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// commandOption defines one command-owned option accepted by the shared token
// parser. Name is the canonical long spelling without leading hyphens.
type commandOption struct {
	Name       string
	Aliases    []string
	TakesValue bool
}

// parsedCommandTokens contains canonical option values and positional tokens.
type parsedCommandTokens struct {
	Options     map[string]string
	Present     map[string]bool
	Positionals []string
}

// commandBoundaryError classifies an unresolved token at a command boundary
// and reports the full visible path for command-shaped input.
func commandBoundaryError(path []string) error {
	token := path[len(path)-1]
	if strings.HasPrefix(token, "-") {
		return diagnostic.New("error.unknown_option", token)
	}
	return diagnostic.New("error.unknown_command", strings.Join(path, " "))
}

// parseCommandTokens classifies command-owned options and positional tokens.
func parseCommandTokens(args []string, options []commandOption) (parsedCommandTokens, error) {
	parsed := parsedCommandTokens{
		Options: make(map[string]string),
		Present: make(map[string]bool),
	}
	byName := make(map[string]commandOption)
	for _, option := range options {
		byName["--"+option.Name] = option
		for _, alias := range option.Aliases {
			byName[alias] = option
		}
	}
	for index := 0; index < len(args); index++ {
		token := args[index]
		if !strings.HasPrefix(token, "-") {
			parsed.Positionals = append(parsed.Positionals, token)
			continue
		}
		name, inlineValue, inline := strings.Cut(token, "=")
		option, ok := byName[name]
		if !ok {
			return parsed, diagnostic.New("error.unknown_option", token)
		}
		if !option.TakesValue {
			if inline {
				return parsed, diagnostic.New("error.unknown_option", token)
			}
			parsed.Present[option.Name] = true
			continue
		}
		value := inlineValue
		if !inline {
			index++
			if index >= len(args) {
				return parsed, diagnostic.New("error.option_requires_value", name)
			}
			value = args[index]
		}
		parsed.Present[option.Name] = true
		parsed.Options[option.Name] = value
	}
	return parsed, nil
}

// rejectUnexpectedPositionals rejects tokens beyond the declared maximum.
func rejectUnexpectedPositionals(values []string, maximum int) error {
	if maximum >= 0 && len(values) > maximum {
		return diagnostic.New("error.unexpected_argument", values[maximum])
	}
	return nil
}

// requirePositional reports one missing required positional by visible name.
func requirePositional(values []string, minimum int, name string) error {
	if len(values) < minimum {
		return diagnostic.New("error.missing_required_argument", name)
	}
	return nil
}

// validatePositionalArguments applies option-shape, minimum, and maximum
// checks to one command's already separated positional input.
func validatePositionalArguments(values, names []string, minimum, maximum int) error {
	for _, value := range values {
		if strings.HasPrefix(value, "-") {
			return diagnostic.New("error.unknown_option", value)
		}
	}
	if len(values) < minimum {
		missingIndex := len(values)
		if missingIndex >= len(names) {
			missingIndex = len(names) - 1
		}
		return diagnostic.New("error.missing_required_argument", names[missingIndex])
	}
	return rejectUnexpectedPositionals(values, maximum)
}
