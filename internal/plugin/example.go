/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// ParseCommandExample splits one inert POSIX-style published example into the
// arguments a shell would pass to ctyun. Shell operators and substitutions are
// deliberately outside the supported example syntax.
func ParseCommandExample(example string) ([]string, error) {
	runes := []rune(example)
	var tokens []string
	var token strings.Builder
	active := false
	quote := rune(0)
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		if quote == '\'' {
			if char == '\'' {
				quote = 0
			} else {
				token.WriteRune(char)
			}
			continue
		}
		if quote == '"' {
			switch char {
			case '"':
				quote = 0
			case '`':
				return nil, fmt.Errorf("shell substitution is not allowed")
			case '\\':
				if index+1 >= len(runes) {
					return nil, fmt.Errorf("trailing escape")
				}
				index++
				token.WriteRune(runes[index])
			default:
				token.WriteRune(char)
			}
			continue
		}
		if unicode.IsSpace(char) {
			if active {
				tokens = append(tokens, token.String())
				token.Reset()
				active = false
			}
			continue
		}
		active = true
		switch char {
		case '\'', '"':
			quote = char
		case '\\':
			if index+1 >= len(runes) {
				return nil, fmt.Errorf("trailing escape")
			}
			index++
			token.WriteRune(runes[index])
		case ';', '|', '&', '<', '>':
			return nil, fmt.Errorf("shell operator %q is not allowed", char)
		case '`':
			return nil, fmt.Errorf("shell substitution is not allowed")
		default:
			token.WriteRune(char)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted value")
	}
	if active {
		tokens = append(tokens, token.String())
	}
	if containsCommandSubstitution(tokens) {
		return nil, fmt.Errorf("shell substitution is not allowed")
	}
	return tokens, nil
}

// containsCommandSubstitution reports whether tokens include an unexpanded
// command-substitution marker.
func containsCommandSubstitution(tokens []string) bool {
	for _, token := range tokens {
		if strings.Contains(token, "$(") {
			return true
		}
	}
	return false
}

// ValidateCommandExample checks one parsed example against the metadata-owned
// visible path, option declarations, requirements, and value types.
func ValidateCommandExample(command Command, example string) error {
	tokens, err := ParseCommandExample(example)
	if err != nil {
		return err
	}
	if len(tokens) == 0 || tokens[0] != "ctyun" {
		return fmt.Errorf("example must start with ctyun")
	}
	index := 1
	for _, segment := range command.Path {
		if index >= len(tokens) || (isPathPlaceholder(segment) && strings.HasPrefix(tokens[index], "--")) {
			return fmt.Errorf("example is missing command path segment %s", segment)
		}
		if !isPathPlaceholder(segment) && tokens[index] != segment {
			return fmt.Errorf("example command path does not match %s", strings.Join(command.Path, " "))
		}
		index++
	}

	byFlag := make(map[string]Parameter, len(command.Parameters))
	byName := make(map[string]Parameter, len(command.Parameters))
	for _, parameter := range command.Parameters {
		byFlag[parameter.Flag] = parameter
		byName[parameter.Name] = parameter
	}
	values := make(map[string]string)
	for index < len(tokens) {
		token := tokens[index]
		if !strings.HasPrefix(token, "--") {
			return fmt.Errorf("unexpected example argument %q", token)
		}
		flagValue := strings.TrimPrefix(token, "--")
		flag, value, inline := strings.Cut(flagValue, "=")
		parameter, ok := byFlag[flag]
		if !ok {
			return fmt.Errorf("unknown example option --%s", flag)
		}
		if _, duplicate := values[parameter.Name]; duplicate {
			return fmt.Errorf("duplicate example option --%s", flag)
		}
		if !inline {
			index++
			if index >= len(tokens) {
				return fmt.Errorf("example option --%s requires a value", flag)
			}
			value = tokens[index]
		}
		if err := validateExampleParameterValue(parameter, value); err != nil {
			return err
		}
		values[parameter.Name] = value
		index++
	}
	for _, parameter := range command.Parameters {
		if parameter.Required && values[parameter.Name] == "" {
			return fmt.Errorf("example is missing required option --%s", parameter.Flag)
		}
	}
	for _, requirement := range command.ConditionalRequirements {
		if !exampleParameterConditionMatches(requirement.When, values[requirement.When.Parameter]) {
			continue
		}
		for _, name := range requirement.Required {
			if values[name] == "" {
				return fmt.Errorf("example is missing conditionally required option --%s", byName[name].Flag)
			}
		}
		if len(requirement.AnyOf) > 0 && !exampleHasAnyValue(values, requirement.AnyOf) {
			return fmt.Errorf("example is missing a conditionally required option")
		}
	}
	return nil
}

// validateExampleParameterValue applies the declaration checks that do not
// depend on runtime localization.
func validateExampleParameterValue(parameter Parameter, value string) error {
	if exampleValuePlaceholder(parameter, value) {
		return nil
	}
	if len(parameter.AllowedValues) > 0 && !slices.Contains(parameter.AllowedValues, value) {
		return fmt.Errorf("example option --%s has unsupported value %q", parameter.Flag, value)
	}
	if parameter.Pattern != "" {
		matched, err := regexp.MatchString(parameter.Pattern, value)
		if err != nil || !matched {
			return fmt.Errorf("example option --%s does not match its pattern", parameter.Flag)
		}
	}
	if _, err := ParseParameterValue(parameter, value); err != nil {
		return fmt.Errorf("example option --%s has invalid %s value", parameter.Flag, parameter.ValueType)
	}
	return nil
}

// exampleValuePlaceholder reports whether value is the explicit source-review
// placeholder for parameter rather than purported concrete evidence.
func exampleValuePlaceholder(parameter Parameter, value string) bool {
	return value == "{"+parameter.Name+"}" || value == "{"+parameter.Flag+"}"
}

// exampleParameterConditionMatches applies conditional metadata to parsed
// example values.
func exampleParameterConditionMatches(condition ParameterCondition, value string) bool {
	if condition.Equals != "" {
		return value == condition.Equals
	}
	return slices.Contains(condition.In, value)
}

// exampleHasAnyValue reports whether at least one named parameter is present.
func exampleHasAnyValue(values map[string]string, names []string) bool {
	for _, name := range names {
		if values[name] != "" {
			return true
		}
	}
	return false
}
