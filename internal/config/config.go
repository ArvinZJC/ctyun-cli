/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package config loads CTyun CLI credentials and profile configuration.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Credentials contains process-provided CTyun AK/SK values for request signing.
type Credentials struct {
	AccessKey string
	SecretKey string
}

// Config is the on-disk profile configuration after JSON decoding.
type Config struct {
	ActiveProfileName string             `json:"active_profile"`
	Profiles          map[string]Profile `json:"profiles"`
}

// Profile contains user-selectable defaults for command execution and plugin
// registry access.
type Profile struct {
	Region            string         `json:"region"`
	Language          string         `json:"language"`
	RegistryURL       string         `json:"registry_url"`
	RegistryPublicKey string         `json:"registry_public_key"`
	Registry          RegistryConfig `json:"registry"`
	EndpointURL       string         `json:"endpoint_url"`
	TimeoutSeconds    int            `json:"timeout_seconds"`
}

// RegistryConfig is the nested registry configuration accepted in profile JSON.
type RegistryConfig struct {
	URL       string `json:"url"`
	PublicKey string `json:"public_key"`
}

// LoadCredentialsFromEnv reads CTYUN_AK and CTYUN_SK from the supplied
// environment lookup.
func LoadCredentialsFromEnv(getenv func(string) string) (Credentials, error) {
	creds := Credentials{
		AccessKey: getenv("CTYUN_AK"),
		SecretKey: getenv("CTYUN_SK"),
	}
	if creds.AccessKey == "" || creds.SecretKey == "" {
		return Credentials{}, errors.New("missing CTyun credentials: set CTYUN_AK and CTYUN_SK")
	}
	return creds, nil
}

// Load decodes profile configuration and rejects persisted credential material.
func Load(raw []byte) (Config, error) {
	if containsPersistedSecret(raw) {
		return Config{}, errors.New("config must not contain AK/SK or secret key material; use CTYUN_AK and CTYUN_SK")
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
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

// containsPersistedSecret reports whether raw config JSON contains credential
// fields or secret-like key names.
func containsPersistedSecret(raw []byte) bool {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return false
	}
	return valueContainsSecretKey(decoded)
}

// valueContainsSecretKey recursively searches decoded config values for
// forbidden secret-bearing keys.
func valueContainsSecretKey(value any) bool {
	object, ok := value.(map[string]any)
	if !ok {
		values, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range values {
			if valueContainsSecretKey(item) {
				return true
			}
		}
		return false
	}

	for key, item := range object {
		normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
		if normalized == "ak" || normalized == "sk" || normalized == "accesskey" || normalized == "secretkey" {
			return true
		}
		if bytes.Contains([]byte(normalized), []byte("secret")) {
			return true
		}
		if valueContainsSecretKey(item) {
			return true
		}
	}
	return false
}
