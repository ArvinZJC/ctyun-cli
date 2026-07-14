/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugin

import "github.com/ArvinZJC/ctyun-cli/internal/diagnostic"

// Validate checks that a target can be matched safely against bundle metadata.
func (target CommandTarget) Validate() error {
	if !ValidName(target.Plugin) {
		return diagnostic.New("error.recommendation_target_plugin", target.Plugin)
	}
	if len(target.Path) == 0 {
		return diagnostic.New("error.recommendation_target_path", target.Plugin, "")
	}
	for _, segment := range target.Path {
		if !validCommandPathSegment(segment) {
			return diagnostic.New("error.recommendation_target_path", target.Plugin, segment)
		}
	}
	return nil
}

// FindCommandTarget resolves an exact declared command path in the named bundle.
func FindCommandTarget(bundles []Bundle, target CommandTarget) (Bundle, Command, bool) {
	for _, bundle := range bundles {
		if bundle.Manifest.Name != target.Plugin {
			continue
		}
		for _, command := range bundle.Commands.Commands {
			if equalStrings(command.Path, target.Path) {
				return bundle, command, true
			}
		}
	}
	return Bundle{}, Command{}, false
}

// CommandIsDeprecated reports command- or operation-level deprecation.
func CommandIsDeprecated(bundle Bundle, command Command) bool {
	if command.Deprecation.Active() {
		return true
	}
	operation, ok := bundle.APIs.Operations[command.Operation]
	return ok && operation.Deprecation.Active()
}

// validateRecommendation checks the local shape of optional command guidance.
func validateRecommendation(recommendation *Recommendation) error {
	if !recommendation.Active() {
		return nil
	}
	return recommendation.TargetCommand.Validate()
}
