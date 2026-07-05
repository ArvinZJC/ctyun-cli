/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import "strings"

// commonCatalog contains localized tokens shared by help and runtime output.
var commonCatalog = map[string]map[string]string{
	"sentence.terminator": {"en-US": ".", "en-GB": ".", "zh-CN": "。"},
	"table.field":         {"en-US": "Field", "en-GB": "Field", "zh-CN": "字段"},
	"table.value":         {"en-US": "Value", "en-GB": "Value", "zh-CN": "值"},
}

// localizedCatalogText resolves one localized string from a catalog with
// language fallbacks.
func localizedCatalogText(catalog map[string]map[string]string, key, language string) string {
	translations, ok := catalog[key]
	if !ok {
		return key
	}
	if text := translations[language]; text != "" {
		return text
	}
	if strings.HasPrefix(language, "en-") {
		if text := translations["en-US"]; text != "" {
			return text
		}
	}
	if text := translations["zh-CN"]; text != "" {
		return text
	}
	return key
}

// commonText resolves localized text shared by help and runtime messages.
func commonText(key, language string) string {
	return localizedCatalogText(commonCatalog, key, language)
}
