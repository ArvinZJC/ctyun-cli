/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"strings"
	"unicode"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// deprecationFromOperation infers API-level deprecation metadata from upstream
// operation title and localized descriptions.
func deprecationFromOperation(operation Operation) *plugin.Deprecation {
	deprecation := deprecationFromTextValues("api", operationLifecycleTexts(operation))
	if deprecation == nil {
		return nil
	}
	if operation.Recommendation != nil && operation.Recommendation.TargetCommand != nil {
		deprecation.Replacement = &plugin.Replacement{
			Kind:  "command",
			Label: commandTargetLabel(*operation.Recommendation.TargetCommand),
		}
	}
	return deprecation
}

// operationLifecycleTexts returns the title, localized descriptions, and
// preserved recommendation notice used for operation lifecycle classification
// in stable preference order.
func operationLifecycleTexts(operation Operation) []string {
	texts := deprecationTexts(operation.Title, operation.Description)
	if operation.Recommendation != nil {
		if notice := strings.TrimSpace(operation.Recommendation.Notice); notice != "" {
			texts = append(texts, notice)
		}
	}
	return texts
}

// operationHasDeprecationText reports whether any operation lifecycle text
// explicitly identifies the operation as deprecated or scheduled for removal.
func operationHasDeprecationText(operation Operation) bool {
	return hasDeprecationText(operationLifecycleTexts(operation))
}

// commandTargetLabel formats a reviewed visible command path for deprecation
// replacement guidance.
func commandTargetLabel(target plugin.CommandTarget) string {
	return "ctyun " + strings.Join(target.Path, " ")
}

// deprecationFromParameter infers option deprecation metadata from upstream
// parameter descriptions.
func deprecationFromParameter(parameter Parameter) *plugin.Deprecation {
	return deprecationFromTexts("parameter", parameter.Description, parameter.Descriptions)
}

// deprecationFromColumn infers response-field deprecation metadata from
// upstream response descriptions when catalogs preserve them.
func deprecationFromColumn(column Column) *plugin.Deprecation {
	return deprecationFromTexts("field", column.Description, column.Descriptions)
}

// deprecationFromTexts maps common CTyun documentation notices to shared plugin
// deprecation metadata.
func deprecationFromTexts(kind, description string, descriptions map[string]string) *plugin.Deprecation {
	return deprecationFromTextValues(kind, deprecationTexts(description, descriptions))
}

// deprecationFromTextValues maps ordered upstream text to shared plugin
// deprecation metadata.
func deprecationFromTextValues(kind string, texts []string) *plugin.Deprecation {
	if !hasDeprecationText(texts) {
		return nil
	}
	deprecation := plugin.Deprecation{
		Status: "deprecated",
		Notice: deprecationNoticeFromTextValues(texts),
	}
	if replacement := deprecationReplacementLabel(texts); replacement != "" {
		deprecation.Replacement = &plugin.Replacement{Kind: kind, Label: replacement}
	}
	return &deprecation
}

// deprecationTexts returns source descriptions in stable preference order.
func deprecationTexts(description string, descriptions map[string]string) []string {
	texts := make([]string, 0, 4)
	for _, language := range []string{"zh-CN", "en-US", "en-GB"} {
		if text := strings.TrimSpace(descriptions[language]); text != "" {
			texts = append(texts, text)
		}
	}
	if text := strings.TrimSpace(description); text != "" {
		texts = append(texts, text)
	}
	return texts
}

// hasDeprecationText reports whether any source text looks like an upstream
// deprecation notice.
func hasDeprecationText(texts []string) bool {
	for _, text := range texts {
		lower := strings.ToLower(text)
		for _, term := range []string{"弃用", "废弃", "下线", "退役", "deprecated", "obsolete"} {
			if strings.Contains(lower, term) {
				return true
			}
		}
	}
	return false
}

// deprecationNotice chooses the best original upstream notice from localized
// descriptions to preserve in metadata.
func deprecationNotice(description string, descriptions map[string]string) string {
	return deprecationNoticeFromTextValues(deprecationTexts(description, descriptions))
}

// deprecationNoticeFromTextValues chooses the first lifecycle notice from
// already ordered upstream text.
func deprecationNoticeFromTextValues(texts []string) string {
	for _, text := range texts {
		if hasDeprecationText([]string{text}) {
			return text
		}
	}
	return ""
}

// deprecationReplacementLabel extracts simple recommended replacement tokens
// from upstream prose such as 建议使用pageNo.
func deprecationReplacementLabel(texts []string) string {
	for _, text := range texts {
		for _, marker := range []string{"建议使用", "推荐使用", "请使用", "改用"} {
			index := strings.Index(text, marker)
			if index < 0 {
				continue
			}
			if label := leadingReplacementToken(text[index+len(marker):]); label != "" {
				return label
			}
		}
	}
	return ""
}

// leadingReplacementToken reads an identifier-like replacement from prose.
func leadingReplacementToken(text string) string {
	var builder strings.Builder
	started := false
	for _, char := range strings.TrimSpace(text) {
		if isReplacementTokenRune(char) {
			builder.WriteRune(char)
			started = true
			continue
		}
		if started {
			break
		}
		if !unicode.IsSpace(char) && !strings.ContainsRune("：:，,。.;；()（）[]【】\"'“”", char) {
			break
		}
	}
	return builder.String()
}

// isReplacementTokenRune reports whether char can belong to a short metadata
// replacement label.
func isReplacementTokenRune(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' || char == '.'
}
