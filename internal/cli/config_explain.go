/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"strconv"
	"strings"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
)

// configCommandInput contains shared config bytes and process resolution context.
type configCommandInput struct {
	Raw            []byte
	Path           string
	PathSource     coreconfig.Source
	LanguageOption string
	Getenv         func(string) string
	OSLocale       string
}

// configExplainJSONSetting is one secret-safe structured setting projection.
type configExplainJSONSetting struct {
	Key        string  `json:"key"`
	Value      *string `json:"value"`
	Configured bool    `json:"configured"`
	Sensitive  bool    `json:"sensitive"`
	Effective  bool    `json:"effective"`
	Source     string  `json:"source"`
	SourceName string  `json:"source_name,omitempty"`
	Profile    string  `json:"profile,omitempty"`
}

// configExplainJSONReport contains displayed config explanation settings.
type configExplainJSONReport struct {
	Settings []configExplainJSONSetting `json:"settings"`
}

// runConfigExplain resolves and renders safe effective config provenance.
func runConfigExplain(stdout io.Writer, args []string, opts globalOptions, input configCommandInput) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return diagnostic.New("error.unknown_option", arg)
		}
	}
	if err := validatePositionalArguments(args, []string{"key"}, 0, 1); err != nil {
		return err
	}
	resolution, err := coreconfig.Resolve(coreconfig.ResolveInput{
		Raw: input.Raw, ConfigPath: input.Path, ConfigPathSource: input.PathSource,
		ProfileOption: opts.Profile, LanguageOption: input.LanguageOption,
		Getenv: input.Getenv, OSLocale: input.OSLocale,
	})
	if err != nil {
		return err
	}
	settings := resolution.Settings
	if len(args) == 1 {
		setting, ok := resolution.Setting(args[0])
		if !ok {
			return diagnostic.New("error.unsupported_config_setting", args[0])
		}
		settings = []coreconfig.Setting{setting}
	}
	return renderConfigExplanation(stdout, settings, opts)
}

// renderConfigExplanation applies table controls and renders table or structured JSON.
func renderConfigExplanation(stdout io.Writer, settings []coreconfig.Setting, opts globalOptions) error {
	columns := configExplainColumns(opts.Language)
	rows := make([]map[string]string, 0, len(settings))
	jsonSettings := make([]configExplainJSONSetting, 0, len(settings))
	for index, setting := range settings {
		rows = append(rows, map[string]string{
			"setting": setting.Key,
			"value":   configExplainValue(setting, opts.Language),
			"source":  configExplainSource(setting.Source, opts.Language),
			"_index":  strconv.Itoa(index),
		})
		jsonSetting := configExplainJSONSetting{
			Key: setting.Key, Configured: setting.Configured, Sensitive: setting.Sensitive,
			Effective: setting.Effective, Source: string(setting.Source.Kind),
			SourceName: setting.Source.Name, Profile: setting.Profile,
		}
		if !setting.Sensitive {
			value := setting.Value
			jsonSetting.Value = &value
		}
		jsonSettings = append(jsonSettings, jsonSetting)
	}
	rows, filteredSettings, err := selectReportRows(rows, jsonSettings, columns, opts)
	if err != nil {
		return err
	}
	return renderReportOutput(stdout, rows, columns, configExplainJSONReport{Settings: filteredSettings}, "", opts)
}

// configExplainColumns returns stable setting columns with localized labels.
func configExplainColumns(language string) []output.Column {
	return []output.Column{
		{Key: "setting", Label: messageText("config.explain.column.setting", language)},
		{Key: "value", Label: messageText("config.explain.column.value", language)},
		{Key: "source", Label: messageText("config.explain.column.source", language)},
	}
}

// configExplainValue returns a localized value without exposing sensitive material.
func configExplainValue(setting coreconfig.Setting, language string) string {
	value := setting.Value
	if setting.Sensitive {
		if setting.Configured {
			value = messageText("config.explain.configured", language)
		} else {
			value = messageText("config.explain.not_configured", language)
		}
	} else if !setting.Configured {
		value = messageText("config.explain.not_configured", language)
	}
	if setting.Configured && !setting.Effective {
		return messagef("config.explain.inactive", language, value)
	}
	return value
}

// configExplainSource renders a human source label plus its safe source name.
func configExplainSource(source coreconfig.Source, language string) string {
	label := messageText("config.explain.source."+string(source.Kind), language)
	if source.Name == "" || source.Kind == coreconfig.SourceUnset {
		return label
	}
	name := source.Name
	switch source.Name {
	case "default config path":
		name = messageText("config.explain.name.default_path", language)
	case "embedded config path":
		name = messageText("config.explain.name.process_path", language)
	}
	return messagef("config.explain.source_with_name", language, label, name)
}
