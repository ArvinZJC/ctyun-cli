package i18n

import "strings"

type LanguageOptions struct {
	Flag     string
	Env      string
	Profile  string
	OSLocale string
}

type Catalog map[string]map[string]string

func ResolveLanguage(options LanguageOptions) string {
	for _, candidate := range []string{options.Flag, options.Env, options.Profile, options.OSLocale} {
		if resolved := matchLanguage(candidate); resolved != "" {
			return resolved
		}
	}
	return "zh-CN"
}

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
