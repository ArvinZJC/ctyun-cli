/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// recommendationHelpLine resolves one help-only visible-command recommendation.
func recommendationHelpLine(source plugin.Bundle, command plugin.Command, bundles []plugin.Bundle, language string) string {
	if !command.Recommendation.Active() || plugin.CommandIsDeprecated(source, command) {
		return ""
	}
	targetBundle, targetCommand, ok := plugin.FindCommandTarget(bundles, command.Recommendation.TargetCommand)
	if !ok || plugin.CommandIsDeprecated(targetBundle, targetCommand) {
		return ""
	}
	return helpf("recommendation.command", language, visibleCommandLine(targetCommand))
}

// visibleCommandLine formats a declared plugin command path as command data.
func visibleCommandLine(command plugin.Command) string {
	return "ctyun " + strings.Join(command.Path, " ")
}
