/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

// configKeyHelp describes one supported config key in command help.
type configKeyHelp struct {
	Name           string
	Value          string
	DescriptionKey string
	Default        string
}

// globalConfigKeyHelp returns the keys accepted without --profile.
func globalConfigKeyHelp() []configKeyHelp {
	return []configKeyHelp{
		{Name: "active_profile", Value: "name", DescriptionKey: "config.key.active_profile"},
		{Name: "ak", Value: "value", DescriptionKey: "config.key.global_ak"},
		{Name: "sk", Value: "value", DescriptionKey: "config.key.global_sk"},
		{Name: "warn_config_credentials", Value: "true|false", DescriptionKey: "config.key.warn_config_credentials", Default: "true"},
		{Name: "warn_deprecated", Value: "true|false", DescriptionKey: "config.key.warn_deprecated", Default: "true"},
	}
}

// profileConfigKeyHelp returns the keys accepted for a named profile.
func profileConfigKeyHelp() []configKeyHelp {
	return []configKeyHelp{
		{Name: "region", Value: "id", DescriptionKey: "config.key.region"},
		{Name: "language", Value: "zh-CN|en-US|en-GB", DescriptionKey: "config.key.language"},
		{Name: "endpoint_url", Value: "url", DescriptionKey: "config.key.endpoint_url"},
		{Name: "timeout_seconds", Value: "seconds", DescriptionKey: "config.key.timeout_seconds"},
		{Name: "ak", Value: "value", DescriptionKey: "config.key.profile_ak"},
		{Name: "sk", Value: "value", DescriptionKey: "config.key.profile_sk"},
		{Name: "warn_config_credentials", Value: "true|false", DescriptionKey: "config.key.warn_config_credentials", Default: "true"},
		{Name: "warn_deprecated", Value: "true|false", DescriptionKey: "config.key.warn_deprecated", Default: "true"},
	}
}

// writeConfigKeyHelp writes global and profile key sections for a config
// mutation command.
func writeConfigKeyHelp(writer *outputWriter, profileOnly, includeValues bool, language string) {
	if !profileOnly {
		writer.Format("\n%s:\n", helpText("config.global_keys.heading", language))
		writeAlignedHelpRows(writer, configKeyHelpRows(globalConfigKeyHelp(), includeValues, language), "  ")
	}
	writer.Format("\n%s:\n", helpText("config.profile_keys.heading", language))
	writeAlignedHelpRows(writer, configKeyHelpRows(profileConfigKeyHelp(), includeValues, language), "  ")
}

// configKeyHelpRows resolves and sorts config key help rows.
func configKeyHelpRows(keys []configKeyHelp, includeValues bool, language string) []helpRow {
	rows := make([]helpRow, 0, len(keys))
	for _, key := range keys {
		name := key.Name
		if includeValues {
			name += " {" + key.Value + "}"
		}
		rows = append(rows, helpRow{
			Name:        name,
			Description: optionHelpText(key.DescriptionKey, key.Default, language),
			SortKey:     key.Name,
		})
	}
	sortHelpRows(rows)
	return rows
}
