/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// pluginCommandUsage formats the runtime usage line for one product command.
func pluginCommandUsage(command plugin.Command, language string) string {
	usage := fmt.Sprintf("  ctyun [%s] %s", helpText("usage.global", language), pluginCommandPathUsage(command))
	for _, parameter := range command.Parameters {
		usage += " " + parameterUsageToken(parameter)
	}
	return usage
}

// pluginCommandPathUsage formats path placeholders for a usage line.
func pluginCommandPathUsage(command plugin.Command) string {
	segments := make([]string, 0, len(command.Path))
	for index, segment := range command.Path {
		if optionalTrailingRegionArgument(command, index) {
			segments = append(segments, "["+segment+"]")
			continue
		}
		segments = append(segments, segment)
	}
	return strings.Join(segments, " ")
}

// optionalTrailingRegionArgument reports whether a trailing region_id path
// argument can come from profile region instead.
func optionalTrailingRegionArgument(command plugin.Command, index int) bool {
	return index == len(command.Path)-1 && isRegionPathPlaceholder(command.Path[index])
}

// parameterUsageToken formats one product-command option for a usage line.
func parameterUsageToken(parameter plugin.Parameter) string {
	token := parameterOptionToken(parameter)
	if parameter.Required {
		return token
	}
	return "[" + token + "]"
}

// parameterOptionToken formats the visible option name and value placeholder.
func parameterOptionToken(parameter plugin.Parameter) string {
	token := "--" + parameter.Flag
	if placeholder := parameterValuePlaceholder(parameter); placeholder != "" {
		token += " <" + placeholder + ">"
	}
	return token
}

// parameterValuePlaceholder returns a scoped value marker from the command
// declaration without translating machine-oriented JSON syntax.
func parameterValuePlaceholder(parameter plugin.Parameter) string {
	if len(parameter.AllowedValues) > 0 {
		return strings.Join(parameter.AllowedValues, "|")
	}
	if compositeParameterValueType(parameter.ValueType) {
		return "json"
	}
	return "value"
}

// pluginCommandParameterHelpRows returns sorted option rows for a product
// command help page.
func pluginCommandParameterHelpRows(bundle plugin.Bundle, command plugin.Command, language string) []helpRow {
	rows := make([]helpRow, 0, len(command.Parameters))
	for _, parameter := range command.Parameters {
		rows = append(rows, helpRow{
			Name:        parameterOptionToken(parameter),
			Description: parameterHelpDescription(bundle, command, parameter, language),
			SortKey:     parameter.Flag,
		})
	}
	sortHelpRows(rows)
	return rows
}

// parameterHelpDescription resolves and annotates a product-command option
// description.
func parameterHelpDescription(bundle plugin.Bundle, command plugin.Command, parameter plugin.Parameter, language string) string {
	description := localizedPluginText(bundle, language, "parameter."+command.ID+"."+parameter.Name+".description", parameter.Description)
	marks := make([]string, 0, 2)
	if parameter.Required {
		marks = append(marks, helpText("required", language))
	}
	if hint := parameterConditionalHint(command, parameter, language); hint != "" {
		marks = append(marks, hint)
	}
	marks = append(marks, helpDeprecationMarks(parameter.Deprecation, language)...)
	if len(marks) > 0 {
		description = strings.TrimSpace(description + " (" + strings.Join(marks, "; ") + ")")
	}
	if hint := parameterValidationHint(parameter, language); hint != "" {
		description = strings.TrimSpace(description + hint)
	}
	return optionHelpDescription(description, parameter.Default, language)
}

// helpDeprecationSentence formats a standalone deprecation help notice.
func helpDeprecationSentence(kindKey string, deprecation *plugin.Deprecation, language string) string {
	parts := []string{helpText(kindKey, language)}
	for _, part := range helpDeprecationMarks(deprecation, language)[1:] {
		parts = append(parts, capitalizeHelpSentencePart(part))
	}
	return helpPageDescription(strings.Join(parts, ". "), language)
}

// capitalizeHelpSentencePart gives standalone English help fragments sentence
// casing without changing CJK text.
func capitalizeHelpSentencePart(part string) string {
	for index, char := range part {
		if char < 'a' || char > 'z' {
			return part
		}
		return string(char-'a'+'A') + part[index+1:]
	}
	return part
}

// helpDeprecationMarks returns compact help-row markers for deprecated
// metadata.
func helpDeprecationMarks(deprecation *plugin.Deprecation, language string) []string {
	if !deprecation.Active() {
		return nil
	}
	marks := []string{helpText("deprecated.marker", language)}
	if deprecation.Replacement != nil && cliReplacementKind(deprecation.Replacement.Kind) && deprecation.Replacement.Label != "" {
		marks = append(marks, helpf("deprecated.replacement", language, deprecation.Replacement.Label))
	}
	return marks
}

// parameterConditionalHint formats a conditional-required marker for one
// product-command option.
func parameterConditionalHint(command plugin.Command, parameter plugin.Parameter, language string) string {
	for _, requirement := range command.ConditionalRequirements {
		conditionParameter, ok := commandParameterByName(command, requirement.When.Parameter)
		if !ok {
			continue
		}
		conditionValue := parameterConditionValue(requirement.When)
		if conditionValue == "" {
			continue
		}
		if containsName(requirement.Required, parameter.Name) {
			return helpf("conditional.required", language, conditionParameter.Flag, conditionValue)
		}
		if containsName(requirement.AnyOf, parameter.Name) {
			return helpf("conditional.any_of", language, conditionalRequirementFlags(command, requirement.AnyOf), conditionParameter.Flag, conditionValue)
		}
	}
	return ""
}

// commandParameterByName returns the parameter metadata with the requested
// stable parameter name.
func commandParameterByName(command plugin.Command, name string) (plugin.Parameter, bool) {
	for _, parameter := range command.Parameters {
		if parameter.Name == name {
			return parameter, true
		}
	}
	return plugin.Parameter{}, false
}

// parameterConditionValue formats the matching value part of a conditional
// requirement.
func parameterConditionValue(condition plugin.ParameterCondition) string {
	if condition.Equals != "" {
		return condition.Equals
	}
	if len(condition.In) > 0 {
		return strings.Join(condition.In, "|")
	}
	return ""
}

// conditionalRequirementFlags formats the flags that can satisfy an any-of
// conditional requirement.
func conditionalRequirementFlags(command plugin.Command, names []string) string {
	flags := make([]string, 0, len(names))
	for _, name := range names {
		if parameter, ok := commandParameterByName(command, name); ok {
			flags = append(flags, "--"+parameter.Flag)
		}
	}
	return strings.Join(flags, ", ")
}

// containsName reports whether names contains name.
func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

// pluginCommandArgumentHelpRows returns positional argument help rows in path
// declaration order.
func pluginCommandArgumentHelpRows(bundle plugin.Bundle, command plugin.Command, language string) []helpRow {
	arguments := commandPathArguments(command.Path)
	rows := make([]helpRow, 0, len(arguments))
	for _, argument := range arguments {
		rows = append(rows, helpRow{
			Name:        "{" + argument + "}",
			Description: pluginCommandArgumentDescription(bundle, command, argument, language),
		})
	}
	return rows
}

// commandPathArguments extracts positional argument names from a command path.
func commandPathArguments(path []string) []string {
	arguments := make([]string, 0)
	for _, segment := range path {
		if argument, ok := commandPathArgumentName(segment); ok {
			arguments = append(arguments, argument)
		}
	}
	return arguments
}

// commandPathArgumentName unwraps one {argument} command path placeholder.
func commandPathArgumentName(segment string) (string, bool) {
	if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
	name = strings.TrimSuffix(name, "...")
	return name, name != ""
}

// pluginCommandArgumentDescription resolves a localized argument description,
// then falls back to table labels by stable key.
func pluginCommandArgumentDescription(bundle plugin.Bundle, command plugin.Command, argument, language string) string {
	key := "argument." + command.ID + "." + argument + ".description"
	if description := localizedPluginText(bundle, language, key, ""); description != "" {
		return description
	}
	if table, ok := bundle.Tables.Tables[command.Table]; ok {
		if label := tableColumnLabel(table, argument, language); label != "" {
			return label
		}
	}
	for _, table := range bundle.Tables.Tables {
		if label := tableColumnLabel(table, argument, language); label != "" {
			return label
		}
	}
	return ""
}
