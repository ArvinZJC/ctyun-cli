/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// APIReference identifies an upstream API target preserved as source catalog
// evidence.
type APIReference struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	DocsURL string `json:"docs_url,omitempty"`
}

// APIRecommendation preserves upstream recommendation evidence and an
// optional reviewed visible command mapping.
type APIRecommendation struct {
	Notice        string                `json:"notice"`
	TargetAPI     APIReference          `json:"target_api"`
	TargetCommand *plugin.CommandTarget `json:"target_command,omitempty"`
	Applicability map[string]string     `json:"applicability,omitempty"`
}

// validateRecommendation checks the source evidence and optional reviewed
// command target for one operation.
func (operation Operation) validateRecommendation() error {
	recommendation := operation.Recommendation
	if recommendation == nil {
		return nil
	}
	if strings.TrimSpace(recommendation.Notice) == "" {
		return fmt.Errorf("operation %s recommendation notice is required", operation.ID)
	}
	if err := validateRecommendationApplicability(operation.ID, recommendation.Applicability); err != nil {
		return err
	}
	target := recommendation.TargetAPI
	if target.Method == "" {
		return fmt.Errorf("operation %s recommendation target method is required", operation.ID)
	}
	if !oneOf(target.Method, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete) {
		return fmt.Errorf("operation %s recommendation target method %s is unsupported", operation.ID, target.Method)
	}
	if target.Path == "" {
		return fmt.Errorf("operation %s recommendation target path is required", operation.ID)
	}
	if !validRecommendationAPIPath(target.Path) {
		return fmt.Errorf("operation %s recommendation target path %s is invalid", operation.ID, target.Path)
	}
	if operation.Method == target.Method && operation.Path == target.Path {
		return fmt.Errorf("operation %s recommendation target must differ from source operation", operation.ID)
	}
	if target.DocsURL != "" && !validRecommendationDocsURL(target.DocsURL) {
		return fmt.Errorf("operation %s recommendation target docs URL must be HTTPS", operation.ID)
	}
	if recommendation.TargetCommand != nil {
		if err := recommendation.TargetCommand.Validate(); err != nil {
			return fmt.Errorf("operation %s recommendation target command: %w", operation.ID, err)
		}
	}
	return nil
}

// validateRecommendationApplicability requires a complete set of public
// locales whenever recommendation applicability is present.
func validateRecommendationApplicability(operationID string, applicability map[string]string) error {
	if len(applicability) == 0 {
		return nil
	}
	for language := range applicability {
		switch language {
		case "en-US", "en-GB", "zh-CN":
		default:
			return fmt.Errorf("operation %s recommendation applicability locale %s is unsupported", operationID, language)
		}
	}
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		value, ok := applicability[language]
		if !ok {
			return fmt.Errorf("operation %s recommendation applicability %s is required", operationID, language)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("operation %s recommendation applicability %s must not be blank", operationID, language)
		}
	}
	return nil
}

// validRecommendationAPIPath accepts clean absolute API paths without query
// fragments, network-path references, or traversal segments.
func validRecommendationAPIPath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, " \t\r\n?#") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

// validRecommendationDocsURL reports whether raw is a complete HTTPS URL.
func validRecommendationDocsURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

// hasRecommendationText reports whether any source text contains common
// recommendation-only wording.
func hasRecommendationText(texts []string) bool {
	for _, text := range texts {
		lower := strings.ToLower(text)
		for _, term := range []string{"推荐使用", "建议使用", "请使用", "recommend", "prefer"} {
			if strings.Contains(lower, term) {
				return true
			}
		}
		if hasChineseSwitchRecommendation(lower) {
			return true
		}
	}
	return false
}

// hasChineseSwitchRecommendation distinguishes standalone 改用 guidance from
// the same adjacent characters across the ordinary 修改用户 phrase.
func hasChineseSwitchRecommendation(text string) bool {
	for {
		index := strings.Index(text, "改用")
		if index < 0 {
			return false
		}
		after := text[index+len("改用"):]
		if !strings.HasPrefix(after, "户") {
			return true
		}
		text = strings.TrimPrefix(after, "户")
	}
}

// operationHasUnclassifiedRecommendation reports recommendation wording that
// has neither explicit recommendation evidence nor lifecycle precedence.
func operationHasUnclassifiedRecommendation(operation Operation) bool {
	texts := operationLifecycleTexts(operation)
	return operation.Recommendation == nil && hasRecommendationText(texts) && !operationHasDeprecationText(operation)
}
