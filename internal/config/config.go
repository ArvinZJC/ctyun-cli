/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package config loads CTyun CLI credentials and profile configuration.
package config

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// CredentialSource records where one resolved credential value came from.
type CredentialSource string

const (
	credentialSourceEnv    CredentialSource = "env"
	credentialSourceConfig CredentialSource = "config"
)

// Credentials contains resolved CTyun AK/SK values for request signing.
type Credentials struct {
	AccessKey       string
	SecretKey       string
	AccessKeySource CredentialSource
	SecretKeySource CredentialSource
}

// UsesConfig reports whether either credential value came from config.
func (c Credentials) UsesConfig() bool {
	return c.AccessKeySource == credentialSourceConfig || c.SecretKeySource == credentialSourceConfig
}

// Config is the on-disk profile configuration after JSON decoding.
type Config struct {
	ActiveProfileName     string             `json:"active_profile"`
	AccessKey             string             `json:"ak"`
	SecretKey             string             `json:"sk"`
	WarnConfigCredentials *bool              `json:"warn_config_credentials"`
	WarnDeprecated        *bool              `json:"warn_deprecated"`
	Profiles              map[string]Profile `json:"profiles"`
}

// Profile contains user-selectable defaults for command execution and plugin
// registry access.
type Profile struct {
	Region                string         `json:"region"`
	Language              string         `json:"language"`
	RegistryURL           string         `json:"registry_url"`
	RegistryPublicKey     string         `json:"registry_public_key"`
	Registry              RegistryConfig `json:"registry"`
	EndpointURL           string         `json:"endpoint_url"`
	TimeoutSeconds        int            `json:"timeout_seconds"`
	AccessKey             string         `json:"ak"`
	SecretKey             string         `json:"sk"`
	WarnConfigCredentials *bool          `json:"warn_config_credentials"`
	WarnDeprecated        *bool          `json:"warn_deprecated"`
}

// RegistryConfig is the nested registry configuration accepted in profile JSON.
type RegistryConfig struct {
	URL       string `json:"url"`
	PublicKey string `json:"public_key"`
}

// ResolveCredentials resolves CTYUN_AK and CTYUN_SK with profile config
// fallbacks.
func ResolveCredentials(getenv func(string) string, profile Profile) (Credentials, error) {
	accessKey, accessKeySource := credentialValue(getenv("CTYUN_AK"), profile.AccessKey)
	secretKey, secretKeySource := credentialValue(getenv("CTYUN_SK"), profile.SecretKey)
	if accessKey == "" || secretKey == "" {
		return Credentials{}, diagnostic.New("error.missing_credentials")
	}
	return Credentials{
		AccessKey:       accessKey,
		SecretKey:       secretKey,
		AccessKeySource: accessKeySource,
		SecretKeySource: secretKeySource,
	}, nil
}

// ShouldWarnConfigCredentials reports whether config-backed credentials should
// emit a runtime warning.
func ShouldWarnConfigCredentials(getenv func(string) string, profile Profile) bool {
	if value := strings.TrimSpace(getenv("CTYUN_WARN_CONFIG_CREDENTIALS")); value != "" {
		return parseWarningBool(value, true)
	}
	if profile.WarnConfigCredentials != nil {
		return *profile.WarnConfigCredentials
	}
	return true
}

// ShouldWarnDeprecated reports whether deprecated metadata should emit runtime
// warnings.
func ShouldWarnDeprecated(getenv func(string) string, profile Profile) bool {
	if value := strings.TrimSpace(getenv("CTYUN_WARN_DEPRECATED")); value != "" {
		return parseWarningBool(value, true)
	}
	if profile.WarnDeprecated != nil {
		return *profile.WarnDeprecated
	}
	return true
}

// ApplyProfileDefaults fills profile fields from global config fallbacks.
func (c Config) ApplyProfileDefaults(profile Profile) Profile {
	if profile.AccessKey == "" {
		profile.AccessKey = c.AccessKey
	}
	if profile.SecretKey == "" {
		profile.SecretKey = c.SecretKey
	}
	if profile.WarnConfigCredentials == nil {
		profile.WarnConfigCredentials = c.WarnConfigCredentials
	}
	if profile.WarnDeprecated == nil {
		profile.WarnDeprecated = c.WarnDeprecated
	}
	return profile
}

// Load decodes profile configuration and rejects unsupported secret material.
func Load(raw []byte) (Config, error) {
	if containsUnsupportedSecret(raw) {
		return Config{}, diagnostic.New("error.config_unsupported_secret")
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, diagnostic.Wrap("error.parse_config", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	for name, profile := range cfg.Profiles {
		if profile.RegistryURL == "" {
			profile.RegistryURL = profile.Registry.URL
		}
		if profile.RegistryPublicKey == "" {
			profile.RegistryPublicKey = profile.Registry.PublicKey
		}
		cfg.Profiles[name] = profile
	}
	return cfg, nil
}

// ActiveProfile returns the explicitly configured profile, or the only profile
// when exactly one exists.
func (c Config) ActiveProfile() (Profile, bool) {
	if c.ActiveProfileName != "" {
		profile, ok := c.Profiles[c.ActiveProfileName]
		return profile, ok
	}
	var active Profile
	count := 0
	for _, profile := range c.Profiles {
		active = profile
		count++
	}
	if count != 1 {
		return Profile{}, false
	}
	return active, true
}

// credentialValue applies environment before config fallback precedence.
func credentialValue(envValue, configValue string) (string, CredentialSource) {
	if envValue != "" {
		return envValue, credentialSourceEnv
	}
	if configValue != "" {
		return configValue, credentialSourceConfig
	}
	return "", ""
}

// parseWarningBool parses common boolean strings and falls back on invalid
// values.
func parseWarningBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// containsUnsupportedSecret reports whether raw config JSON contains
// unsupported secret-like key names.
func containsUnsupportedSecret(raw []byte) bool {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return false
	}
	return valueContainsSecretKey(decoded, nil)
}

// valueContainsSecretKey recursively searches decoded config values for
// forbidden secret-bearing keys.
func valueContainsSecretKey(value any, path []string) bool {
	object, ok := value.(map[string]any)
	if !ok {
		values, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range values {
			if valueContainsSecretKey(item, path) {
				return true
			}
		}
		return false
	}

	for key, item := range object {
		normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
		if normalized == "ak" || normalized == "sk" {
			if !allowedCredentialPath(path) {
				return true
			}
		} else if normalized == "accesskey" || normalized == "secretkey" {
			return true
		} else if bytes.Contains([]byte(normalized), []byte("secret")) {
			return true
		}
		if valueContainsSecretKey(item, append(path, key)) {
			return true
		}
	}
	return false
}

// allowedCredentialPath permits ak/sk only at top level or directly inside a
// profile object.
func allowedCredentialPath(path []string) bool {
	if len(path) == 0 {
		return true
	}
	return len(path) == 2 && path[0] == "profiles"
}
