/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package i18n resolves CLI languages and localized message catalogs.
package i18n

import "strings"

// LanguageOptions lists language hints in descending precedence.
type LanguageOptions struct {
	Flag     string
	Env      string
	Profile  string
	OSLocale string
}

// Catalog maps message keys to localized labels by language tag.
type Catalog map[string]map[string]string

// ResolveLanguage chooses the first supported language from the supplied
// options and falls back to Simplified Chinese.
func ResolveLanguage(options LanguageOptions) string {
	for _, candidate := range []string{options.Flag, options.Env, options.Profile, options.OSLocale} {
		if resolved := matchLanguage(candidate); resolved != "" {
			return resolved
		}
	}
	return "zh-CN"
}

// Text returns a localized label for key, falling back to zh-CN and then the
// key itself.
func (c Catalog) Text(key, language string) string {
	labels, ok := c[key]
	if !ok {
		return key
	}
	if text := labels[language]; text != "" {
		return text
	}
	if text := labels["zh-CN"]; text != "" {
		return text
	}
	return key
}

func matchLanguage(candidate string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(candidate), "_", "-")
	if base, _, ok := strings.Cut(normalized, "."); ok {
		normalized = base
	}
	if normalized == "" {
		return ""
	}

	switch normalized {
	case "zh-CN", "zh-Hans", "zh":
		return "zh-CN"
	case "en-US":
		return "en-US"
	case "en-GB", "en-AU", "en-NZ", "en-IE":
		return "en-GB"
	}
	if strings.HasPrefix(normalized, "en-") {
		return "en-US"
	}
	if strings.HasPrefix(normalized, "zh-") {
		return "zh-CN"
	}
	return ""
}
