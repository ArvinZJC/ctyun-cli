/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// parameterValueType maps one normalized CTyun parameter type into the stable
// plugin request-value vocabulary.
func parameterValueType(sourceType string) (plugin.ParameterValueType, error) {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "", "string":
		return plugin.ParameterValueString, nil
	case "integer":
		return plugin.ParameterValueInteger, nil
	case "float", "number":
		return plugin.ParameterValueNumber, nil
	case "boolean":
		return plugin.ParameterValueBoolean, nil
	case "array of strings":
		return plugin.ParameterValueStringArray, nil
	case "array of integers":
		return plugin.ParameterValueIntegerArray, nil
	case "array of objects":
		return plugin.ParameterValueObjectArray, nil
	case "map of string":
		return plugin.ParameterValueStringMap, nil
	case "object", "json":
		return plugin.ParameterValueJSON, nil
	default:
		return "", fmt.Errorf("unsupported parameter type %q", sourceType)
	}
}

// generatedParameterValueType omits the default string value from generated
// metadata while preserving every non-string request shape.
func generatedParameterValueType(parameter Parameter) plugin.ParameterValueType {
	valueType, _ := parameterValueType(parameter.Type)
	if valueType == plugin.ParameterValueString {
		return ""
	}
	return valueType
}

// validateParameterExample checks that captured JSON evidence matches the
// normalized parameter type.
func validateParameterExample(parameter Parameter) error {
	if len(parameter.Example) == 0 {
		return nil
	}
	if !json.Valid(parameter.Example) {
		return fmt.Errorf("example is invalid JSON")
	}
	valueType, err := parameterValueType(parameter.Type)
	if err != nil {
		return err
	}
	if valueType == plugin.ParameterValueString {
		var value string
		if err := json.Unmarshal(parameter.Example, &value); err != nil {
			return fmt.Errorf("example must be a JSON string")
		}
		return nil
	}
	_, err = plugin.ParseParameterValue(plugin.Parameter{ValueType: valueType}, string(parameter.Example))
	return err
}

// generatedCommandExamples prefers complete captured source examples for
// commands without required options and otherwise builds one complete minimal
// example from typed evidence.
func generatedCommandExamples(operation Operation, command plugin.Command) []string {
	first := generatedCommandExample(operation, command)
	if hasRequiredCommandParameter(command) {
		return []string{first}
	}
	var examples []string
	for _, example := range concreteExamples(operation) {
		if example == "" || slices.Contains(examples, example) {
			continue
		}
		examples = append(examples, example)
	}
	if len(examples) == 0 {
		if commandNeedsExample(command) {
			return []string{first}
		}
		return nil
	}
	return examples
}

// generatedCommandExample builds the canonical command path plus every
// required option and the requirements of any selected conditional branch.
func generatedCommandExample(operation Operation, command plugin.Command) string {
	parts := []string{"ctyun"}
	for _, segment := range command.Path {
		if name, ok := examplePathArgument(segment); ok {
			segment = operationArgumentExample(operation, name)
		}
		parts = append(parts, shellQuoteExampleValue(segment))
	}
	selected := make(map[string]string)
	for _, parameter := range command.Parameters {
		if !parameter.Required {
			continue
		}
		value := operationParameterExample(operation, parameter.Name)
		selected[parameter.Name] = value
		parts = append(parts, "--"+parameter.Flag, shellQuoteExampleValue(value))
	}
	for _, requirement := range command.ConditionalRequirements {
		if !exampleConditionMatches(requirement.When, selected[requirement.When.Parameter]) {
			continue
		}
		for _, name := range requirement.Required {
			parts = appendExampleParameter(operation, command, selected, parts, name)
		}
		if len(requirement.AnyOf) > 0 {
			parts = appendExampleParameter(operation, command, selected, parts, requirement.AnyOf[0])
		}
	}
	return strings.Join(parts, " ")
}

// appendExampleParameter adds one conditional option unless it is already in
// the example.
func appendExampleParameter(operation Operation, command plugin.Command, selected map[string]string, parts []string, name string) []string {
	if selected[name] != "" {
		return parts
	}
	for _, parameter := range command.Parameters {
		if parameter.Name != name {
			continue
		}
		value := operationParameterExample(operation, name)
		selected[name] = value
		return append(parts, "--"+parameter.Flag, shellQuoteExampleValue(value))
	}
	return parts
}

// operationParameterExample returns source-backed CLI text for one generated
// parameter, or a visible placeholder when evidence is unavailable.
func operationParameterExample(operation Operation, cliName string) string {
	for _, parameter := range operation.Parameters {
		if parameter.CLIName != cliName && !(parameter.Profile == "region" && cliName == "region") {
			continue
		}
		if raw, ok := operation.RequestExample[parameter.Name]; ok {
			if value, ok := sourceExampleText(parameter, raw); ok {
				return canonicalParameterExample(parameter, value)
			}
		}
		if value, ok := sourceExampleText(parameter, parameter.Example); ok {
			return canonicalParameterExample(parameter, value)
		}
		if parameter.Default != "" {
			return parameter.Default
		}
		if len(parameter.Enum) > 0 {
			return parameter.Enum[0]
		}
		if parameter.ExampleUnavailable {
			return unavailableParameterExample(parameter, cliName)
		}
		return "{" + cliName + "}"
	}
	return "{" + cliName + "}"
}

// unavailableParameterExample returns an executable, type-correct placeholder
// when reviewed source evidence explicitly records that no example is present.
func unavailableParameterExample(parameter Parameter, cliName string) string {
	valueType, _ := parameterValueType(parameter.Type)
	switch valueType {
	case plugin.ParameterValueInteger, plugin.ParameterValueNumber:
		return "0"
	case plugin.ParameterValueBoolean:
		return "false"
	case plugin.ParameterValueStringArray:
		return `["{` + cliName + `}"]`
	case plugin.ParameterValueIntegerArray:
		return "[0]"
	case plugin.ParameterValueObjectArray:
		return `[{"value":"{` + cliName + `}"}]`
	case plugin.ParameterValueStringMap:
		return `{"key":"{` + cliName + `}"}`
	case plugin.ParameterValueJSON:
		return `{"value":"{` + cliName + `}"}`
	default:
		return "{" + cliName + "}"
	}
}

// canonicalParameterExample resolves an upstream example to the declared enum
// spelling when the documentation differs only by letter case.
func canonicalParameterExample(parameter Parameter, value string) string {
	for _, allowed := range parameter.Enum {
		if strings.EqualFold(allowed, value) {
			return allowed
		}
	}
	return value
}

// operationArgumentExample returns captured request or response evidence for a
// path argument, falling back to its explicit visible placeholder.
func operationArgumentExample(operation Operation, argument string) string {
	for _, parameter := range operation.Parameters {
		if parameter.Argument != argument {
			continue
		}
		if raw, ok := operation.RequestExample[parameter.Name]; ok {
			if value, ok := sourceExampleText(parameter, raw); ok {
				return value
			}
		}
		if value, ok := sourceExampleText(parameter, parameter.Example); ok {
			return value
		}
		if value := exampleArgumentValues(operation)[argument]; value != "" {
			return value
		}
		return "{" + argument + "}"
	}
	return "{" + argument + "}"
}

// sourceExampleText converts typed JSON evidence into the exact command-line
// text accepted by the generated parameter declaration.
func sourceExampleText(parameter Parameter, raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || !json.Valid(raw) {
		return "", false
	}
	valueType, err := parameterValueType(parameter.Type)
	if err != nil {
		return "", false
	}
	if valueType == plugin.ParameterValueString {
		var value string
		if json.Unmarshal(raw, &value) != nil {
			return "", false
		}
		if value == "" {
			return "", false
		}
		return value, true
	}
	var compact bytes.Buffer
	_ = json.Compact(&compact, raw)
	return compact.String(), true
}

// examplePathArgument unwraps a generated {argument} path segment.
func examplePathArgument(segment string) (string, bool) {
	if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
	return name, name != ""
}

// exampleConditionMatches reports whether a selected example value activates
// one conditional requirement.
func exampleConditionMatches(condition plugin.ParameterCondition, value string) bool {
	if condition.Equals != "" {
		return value == condition.Equals
	}
	return slices.Contains(condition.In, value)
}

// hasRequiredCommandParameter reports whether source examples could omit a
// command-owned requirement and therefore should not be retained blindly.
func hasRequiredCommandParameter(command plugin.Command) bool {
	for _, parameter := range command.Parameters {
		if parameter.Required {
			return true
		}
	}
	return false
}

// commandNeedsExample reports whether omitting every example would leave
// users without a concrete invocation for required command input.
func commandNeedsExample(command plugin.Command) bool {
	if hasRequiredCommandParameter(command) {
		return true
	}
	for _, segment := range command.Path {
		if _, ok := examplePathArgument(segment); ok {
			return true
		}
	}
	return false
}

// operationHasSourceCommandExample reports whether the source supplied an
// example that generation would retain for an input-free command.
func operationHasSourceCommandExample(operation Operation) bool {
	for _, example := range concreteExamples(operation) {
		if example != "" {
			return true
		}
	}
	return false
}

// shellQuoteExampleValue quotes values only when they contain characters that
// would otherwise be interpreted by a POSIX shell.
func shellQuoteExampleValue(value string) string {
	if value != "" && strings.IndexFunc(value, func(char rune) bool {
		return !unicode.IsLetter(char) && !unicode.IsDigit(char) && !strings.ContainsRune("-._:/@%+=,{}", char)
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
