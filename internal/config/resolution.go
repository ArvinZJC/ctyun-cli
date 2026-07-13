/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package config

import (
	"strconv"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
)

// SourceKind identifies the precedence layer that supplied a resolved value.
type SourceKind string

const (
	// SourceOption identifies an explicit command-line option.
	SourceOption SourceKind = "option"
	// SourceProcess identifies a value injected by the embedding process.
	SourceProcess SourceKind = "process"
	// SourceEnvironment identifies an environment variable.
	SourceEnvironment SourceKind = "environment"
	// SourceProfile identifies a field in the selected profile.
	SourceProfile SourceKind = "profile"
	// SourceConfig identifies a supported top-level config fallback.
	SourceConfig SourceKind = "config"
	// SourceImplicit identifies the only configured profile selected implicitly.
	SourceImplicit SourceKind = "implicit"
	// SourceOS identifies an operating-system locale hint.
	SourceOS SourceKind = "os"
	// SourceDefault identifies a built-in default.
	SourceDefault SourceKind = "default"
	// SourceUnset identifies a setting with no resolved value.
	SourceUnset SourceKind = "unset"
)

// Source describes the winning precedence layer and its safe source name.
type Source struct {
	Kind SourceKind `json:"kind"`
	Name string     `json:"name,omitempty"`
}

// PathInput lists config path candidates in descending precedence.
type PathInput struct {
	Option      string
	Process     string
	Environment string
	Default     string
}

// Setting describes one resolved base configuration value and its provenance.
type Setting struct {
	Key        string
	Value      string
	Configured bool
	Sensitive  bool
	Effective  bool
	Source     Source
	Profile    string
}

// ResolveInput contains already-collected process inputs for config resolution.
type ResolveInput struct {
	Raw              []byte
	ConfigPath       string
	ConfigPathSource Source
	ProfileOption    string
	LanguageOption   string
	Getenv           func(string) string
	OSLocale         string
}

// Resolution contains ordered safe settings and the selected runtime profile.
type Resolution struct {
	Settings    []Setting
	profile     Profile
	profileName string
}

// Inspection contains best-effort resolution plus independent config and profile errors.
type Inspection struct {
	Resolution   Resolution
	ConfigError  error
	ProfileError error
	config       Config
}

// candidate is one internal value and provenance option in a precedence chain.
type candidate struct {
	value   string
	source  Source
	profile string
}

// settingKeys is the canonical config explanation order.
var settingKeys = []string{
	"config_path",
	"profile",
	"language",
	"region",
	"endpoint_url",
	"timeout_seconds",
	"ak",
	"sk",
	"warn_config_credentials",
	"warn_deprecated",
	"registry_url",
	"registry_public_key",
}

// ResolvePath returns the highest-precedence config path and its provenance.
func ResolvePath(input PathInput) (string, Source) {
	return firstCandidate(
		candidate{value: input.Option, source: Source{Kind: SourceOption, Name: "--config"}},
		candidate{value: input.Process, source: Source{Kind: SourceProcess, Name: "embedded config path"}},
		candidate{value: input.Environment, source: Source{Kind: SourceEnvironment, Name: "CTYUN_CONFIG"}},
		candidate{value: input.Default, source: Source{Kind: SourceDefault, Name: "default config path"}},
	).valueAndSource()
}

// SettingKeys returns the stable config explanation setting order.
func SettingKeys() []string {
	return append([]string(nil), settingKeys...)
}

// Profile returns the selected profile with supported top-level fallbacks applied.
func (resolution Resolution) Profile() Profile {
	return resolution.profile
}

// ProfileName returns the explicit, active, or implicit selected profile name.
func (resolution Resolution) ProfileName() string {
	return resolution.profileName
}

// Setting returns one setting by its stable key.
func (resolution Resolution) Setting(key string) (Setting, bool) {
	for _, setting := range resolution.Settings {
		if setting.Key == key {
			return setting, true
		}
	}
	return Setting{}, false
}

// HasStoredCredentials reports whether supported config fields contain AK/SK material.
func (inspection Inspection) HasStoredCredentials() bool {
	if inspection.config.AccessKey != "" || inspection.config.SecretKey != "" {
		return true
	}
	for _, profile := range inspection.config.Profiles {
		if profile.AccessKey != "" || profile.SecretKey != "" {
			return true
		}
	}
	return false
}

// Inspect resolves every safe setting while retaining config and profile failures separately.
func Inspect(input ResolveInput) Inspection {
	if input.Getenv == nil {
		input.Getenv = func(string) string { return "" }
	}
	cfg := Config{Profiles: make(map[string]Profile)}
	var configErr error
	if len(input.Raw) > 0 {
		cfg, configErr = Load(input.Raw)
	}
	inspection := Inspection{ConfigError: configErr, config: cfg}
	profile, profileName, profileSource, profileErr := selectProfile(cfg, input.ProfileOption, configErr)
	inspection.ProfileError = profileErr
	resolvedProfile := profile
	if configErr == nil && profileErr == nil {
		resolvedProfile = cfg.ApplyProfileDefaults(profile)
	}
	inspection.Resolution = buildResolution(input, cfg, profile, resolvedProfile, profileName, profileSource, configErr, profileErr)
	return inspection
}

// Resolve performs strict config resolution, returning config errors before profile errors.
func Resolve(input ResolveInput) (Resolution, error) {
	inspection := Inspect(input)
	if inspection.ConfigError != nil {
		return Resolution{}, inspection.ConfigError
	}
	if inspection.ProfileError != nil {
		return Resolution{}, inspection.ProfileError
	}
	return inspection.Resolution, nil
}

// selectProfile applies explicit, active, and implicit-single precedence.
func selectProfile(cfg Config, option string, configErr error) (Profile, string, Source, error) {
	if configErr != nil {
		return Profile{}, "", Source{Kind: SourceUnset}, nil
	}
	if option != "" {
		profile, ok := cfg.Profiles[option]
		if !ok {
			return Profile{}, "", Source{Kind: SourceUnset}, diagnostic.New("error.profile_not_found", option)
		}
		return profile, option, Source{Kind: SourceOption, Name: "--profile"}, nil
	}
	if cfg.ActiveProfileName != "" {
		profile, ok := cfg.Profiles[cfg.ActiveProfileName]
		if !ok {
			return Profile{}, "", Source{Kind: SourceUnset}, diagnostic.New("error.active_profile_missing", cfg.ActiveProfileName)
		}
		return profile, cfg.ActiveProfileName, Source{Kind: SourceConfig, Name: "active_profile"}, nil
	}
	if len(cfg.Profiles) == 1 {
		for name, profile := range cfg.Profiles {
			return profile, name, Source{Kind: SourceImplicit, Name: name}, nil
		}
	}
	if len(cfg.Profiles) > 1 {
		return Profile{}, "", Source{Kind: SourceUnset}, diagnostic.New("error.config_multiple_profiles")
	}
	return Profile{}, "", Source{Kind: SourceUnset}, nil
}

// buildResolution constructs the ordered, secret-safe setting catalogue.
func buildResolution(input ResolveInput, cfg Config, profile, resolvedProfile Profile, profileName string, profileSource Source, configErr, profileErr error) Resolution {
	settings := make([]Setting, 0, len(settingKeys))
	settings = append(settings, settingFromCandidate("config_path", candidate{value: input.ConfigPath, source: normalizedSource(input.ConfigPathSource)}))
	settings = append(settings, settingFromCandidate("profile", candidate{value: profileName, source: profileSource}))
	profileAvailable := configErr == nil && profileErr == nil

	language := firstSupportedLanguage(
		candidate{value: input.LanguageOption, source: Source{Kind: SourceOption, Name: "--lang"}},
		candidate{value: input.Getenv("CTYUN_LANGUAGE"), source: Source{Kind: SourceEnvironment, Name: "CTYUN_LANGUAGE"}},
		conditionalCandidate(profileAvailable, profile.Language, Source{Kind: SourceProfile, Name: profileName}, profileName),
		candidate{value: input.OSLocale, source: Source{Kind: SourceOS, Name: input.OSLocale}},
		candidate{value: "zh-CN", source: Source{Kind: SourceDefault, Name: "zh-CN"}},
	)
	settings = append(settings, settingFromCandidate("language", language))
	settings = append(settings, settingFromCandidate("region", conditionalCandidate(profileAvailable, profile.Region, Source{Kind: SourceProfile, Name: profileName}, profileName)))
	settings = append(settings, settingFromCandidate("endpoint_url", conditionalCandidate(profileAvailable, profile.EndpointURL, Source{Kind: SourceProfile, Name: profileName}, profileName)))
	timeout := ""
	if profileAvailable && profile.TimeoutSeconds != 0 {
		timeout = strconv.Itoa(profile.TimeoutSeconds)
	}
	settings = append(settings, settingFromCandidate("timeout_seconds", conditionalCandidate(profileAvailable, timeout, Source{Kind: SourceProfile, Name: profileName}, profileName)))

	ak := credentialCandidate(input.Getenv("CTYUN_AK"), "CTYUN_AK", profileAvailable, profile.AccessKey, cfg.AccessKey, profileName, "ak")
	sk := credentialCandidate(input.Getenv("CTYUN_SK"), "CTYUN_SK", profileAvailable, profile.SecretKey, cfg.SecretKey, profileName, "sk")
	settings = append(settings, sensitiveSetting("ak", ak, true))
	settings = append(settings, sensitiveSetting("sk", sk, true))
	settings = append(settings, warningSetting("warn_config_credentials", "CTYUN_WARN_CONFIG_CREDENTIALS", input.Getenv, profileAvailable, profile.WarnConfigCredentials, cfg.WarnConfigCredentials, profileName))
	settings = append(settings, warningSetting("warn_deprecated", "CTYUN_WARN_DEPRECATED", input.Getenv, profileAvailable, profile.WarnDeprecated, cfg.WarnDeprecated, profileName))
	settings = append(settings, inactiveSetting("registry_url", conditionalCandidate(profileAvailable, profile.RegistryURL, Source{Kind: SourceProfile, Name: profileName}, profileName), false))
	settings = append(settings, inactiveSetting("registry_public_key", conditionalCandidate(profileAvailable, profile.RegistryPublicKey, Source{Kind: SourceProfile, Name: profileName}, profileName), true))
	return Resolution{Settings: settings, profile: resolvedProfile, profileName: profileName}
}

// credentialCandidate applies environment, profile, config, and unset precedence.
func credentialCandidate(environment, environmentName string, profileAvailable bool, profileValue, configValue, profileName, configName string) candidate {
	return firstCandidate(
		candidate{value: environment, source: Source{Kind: SourceEnvironment, Name: environmentName}},
		conditionalCandidate(profileAvailable, profileValue, Source{Kind: SourceProfile, Name: profileName}, profileName),
		conditionalCandidate(profileAvailable, configValue, Source{Kind: SourceConfig, Name: configName}, ""),
	)
}

// warningSetting resolves one boolean warning control with its provenance.
func warningSetting(key, environmentName string, getenv func(string) string, profileAvailable bool, profileValue, configValue *bool, profileName string) Setting {
	if value := getenv(environmentName); value != "" {
		return settingFromCandidate(key, candidate{value: strconv.FormatBool(parseWarningBool(value, true)), source: Source{Kind: SourceEnvironment, Name: environmentName}})
	}
	if profileAvailable && profileValue != nil {
		return settingFromCandidate(key, candidate{value: strconv.FormatBool(*profileValue), source: Source{Kind: SourceProfile, Name: profileName}, profile: profileName})
	}
	if configValue != nil {
		return settingFromCandidate(key, candidate{value: strconv.FormatBool(*configValue), source: Source{Kind: SourceConfig, Name: key}})
	}
	return settingFromCandidate(key, candidate{value: "true", source: Source{Kind: SourceDefault, Name: "true"}})
}

// conditionalCandidate suppresses a candidate whose prerequisite is unavailable.
func conditionalCandidate(enabled bool, value string, source Source, profile string) candidate {
	if !enabled {
		return candidate{}
	}
	return candidate{value: value, source: source, profile: profile}
}

// firstSupportedLanguage returns the first candidate matching a supported locale.
func firstSupportedLanguage(candidates ...candidate) candidate {
	for _, item := range candidates {
		if resolved := i18n.MatchLanguage(item.value); resolved != "" {
			item.value = resolved
			return item
		}
	}
	return candidate{source: Source{Kind: SourceUnset}}
}

// firstCandidate returns the first non-empty value and its provenance.
func firstCandidate(candidates ...candidate) candidate {
	for _, item := range candidates {
		if item.value != "" {
			return item
		}
	}
	return candidate{source: Source{Kind: SourceUnset}}
}

// valueAndSource projects a candidate onto the public path result shape.
func (item candidate) valueAndSource() (string, Source) {
	return item.value, item.source
}

// normalizedSource converts an absent source into the stable unset source.
func normalizedSource(source Source) Source {
	if source.Kind == "" {
		return Source{Kind: SourceUnset}
	}
	return source
}

// settingFromCandidate builds a non-sensitive effective setting.
func settingFromCandidate(key string, item candidate) Setting {
	if item.value == "" {
		item.source = Source{Kind: SourceUnset}
		item.profile = ""
	} else {
		item.source = normalizedSource(item.source)
	}
	return Setting{Key: key, Value: item.value, Configured: item.value != "", Effective: true, Source: item.source, Profile: item.profile}
}

// sensitiveSetting removes the resolved value while retaining configured state.
func sensitiveSetting(key string, item candidate, effective bool) Setting {
	setting := settingFromCandidate(key, item)
	setting.Value = ""
	setting.Sensitive = true
	setting.Effective = effective
	return setting
}

// inactiveSetting marks an accepted stored field that does not affect execution.
func inactiveSetting(key string, item candidate, sensitive bool) Setting {
	setting := settingFromCandidate(key, item)
	setting.Effective = false
	if sensitive {
		setting.Value = ""
		setting.Sensitive = true
	}
	return setting
}
