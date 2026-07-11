/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package config

import "testing"

func TestResolveCredentialsUsesOfficialEnvNames(t *testing.T) {
	env := map[string]string{
		"CTYUN_AK": "ak-test",
		"CTYUN_SK": "sk-test",
	}

	creds, err := ResolveCredentials(func(key string) string {
		return env[key]
	}, Profile{})
	if err != nil {
		t.Fatalf("ResolveCredentials returned error: %v", err)
	}

	if creds.AccessKey != "ak-test" {
		t.Fatalf("AccessKey = %q, want %q", creds.AccessKey, "ak-test")
	}
	if creds.SecretKey != "sk-test" {
		t.Fatalf("SecretKey = %q, want %q", creds.SecretKey, "sk-test")
	}
	if creds.UsesConfig() {
		t.Fatal("UsesConfig returned true for environment credentials")
	}
}

func TestResolveCredentialsRejectsMissingValues(t *testing.T) {
	_, err := ResolveCredentials(func(string) string { return "" }, Profile{})
	if err == nil {
		t.Fatal("ResolveCredentials returned nil error for missing credentials")
	}
}

func TestResolveCredentialsFallsBackToProfileConfig(t *testing.T) {
	profile := Profile{AccessKey: "profile-ak", SecretKey: "profile-sk"}

	creds, err := ResolveCredentials(func(string) string { return "" }, profile)
	if err != nil {
		t.Fatalf("ResolveCredentials returned error: %v", err)
	}

	if creds.AccessKey != "profile-ak" || creds.SecretKey != "profile-sk" {
		t.Fatalf("credentials = %#v, want profile credentials", creds)
	}
	if !creds.UsesConfig() {
		t.Fatal("UsesConfig returned false for profile credentials")
	}
}

func TestResolveCredentialsUsesEnvironmentBeforeConfigFallbacks(t *testing.T) {
	env := map[string]string{
		"CTYUN_AK": "env-ak",
		"CTYUN_SK": "env-sk",
	}
	profile := Profile{AccessKey: "profile-ak", SecretKey: "profile-sk"}

	creds, err := ResolveCredentials(func(key string) string { return env[key] }, profile)
	if err != nil {
		t.Fatalf("ResolveCredentials returned error: %v", err)
	}

	if creds.AccessKey != "env-ak" || creds.SecretKey != "env-sk" {
		t.Fatalf("credentials = %#v, want environment credentials", creds)
	}
	if creds.UsesConfig() {
		t.Fatal("UsesConfig returned true when both credentials came from env")
	}
}

func TestResolveCredentialsAllowsMixedEnvironmentAndConfigFallbacks(t *testing.T) {
	env := map[string]string{"CTYUN_AK": "env-ak"}
	profile := Profile{AccessKey: "profile-ak", SecretKey: "profile-sk"}

	creds, err := ResolveCredentials(func(key string) string { return env[key] }, profile)
	if err != nil {
		t.Fatalf("ResolveCredentials returned error: %v", err)
	}

	if creds.AccessKey != "env-ak" || creds.SecretKey != "profile-sk" {
		t.Fatalf("credentials = %#v, want env AK and profile SK", creds)
	}
	if !creds.UsesConfig() {
		t.Fatal("UsesConfig returned false when SK came from config")
	}
}

func TestConfigCredentialWarningDefaultsToEnabled(t *testing.T) {
	if !ShouldWarnConfigCredentials(func(string) string { return "" }, Profile{}) {
		t.Fatal("ShouldWarnConfigCredentials returned false by default")
	}
}

func TestConfigCredentialWarningCanBeDisabledByEnv(t *testing.T) {
	if ShouldWarnConfigCredentials(func(key string) string {
		if key == "CTYUN_WARN_CONFIG_CREDENTIALS" {
			return "0"
		}
		return ""
	}, Profile{}) {
		t.Fatal("ShouldWarnConfigCredentials returned true with env disabled")
	}
}

func TestConfigCredentialWarningCanBeEnabledByEnv(t *testing.T) {
	disabled := false
	if !ShouldWarnConfigCredentials(func(key string) string {
		if key == "CTYUN_WARN_CONFIG_CREDENTIALS" {
			return "yes"
		}
		return ""
	}, Profile{WarnConfigCredentials: &disabled}) {
		t.Fatal("ShouldWarnConfigCredentials returned false with env enabled")
	}
}

func TestConfigCredentialWarningFallsBackOnInvalidEnvValue(t *testing.T) {
	if !ShouldWarnConfigCredentials(func(key string) string {
		if key == "CTYUN_WARN_CONFIG_CREDENTIALS" {
			return "maybe"
		}
		return ""
	}, Profile{}) {
		t.Fatal("ShouldWarnConfigCredentials returned false for invalid env value")
	}
}

func TestConfigCredentialWarningCanBeDisabledByConfig(t *testing.T) {
	disabled := false
	if ShouldWarnConfigCredentials(func(string) string { return "" }, Profile{WarnConfigCredentials: &disabled}) {
		t.Fatal("ShouldWarnConfigCredentials returned true with config disabled")
	}
}

func TestDeprecatedWarningDefaultsToEnabled(t *testing.T) {
	if !ShouldWarnDeprecated(func(string) string { return "" }, Profile{}) {
		t.Fatal("ShouldWarnDeprecated returned false by default")
	}
}

func TestDeprecatedWarningCanBeDisabledByEnv(t *testing.T) {
	if ShouldWarnDeprecated(func(key string) string {
		if key == "CTYUN_WARN_DEPRECATED" {
			return "0"
		}
		return ""
	}, Profile{}) {
		t.Fatal("ShouldWarnDeprecated returned true with env disabled")
	}
}

func TestDeprecatedWarningCanBeEnabledByEnv(t *testing.T) {
	disabled := false
	if !ShouldWarnDeprecated(func(key string) string {
		if key == "CTYUN_WARN_DEPRECATED" {
			return "yes"
		}
		return ""
	}, Profile{WarnDeprecated: &disabled}) {
		t.Fatal("ShouldWarnDeprecated returned false with env enabled")
	}
}

func TestDeprecatedWarningFallsBackOnInvalidEnvValue(t *testing.T) {
	if !ShouldWarnDeprecated(func(key string) string {
		if key == "CTYUN_WARN_DEPRECATED" {
			return "maybe"
		}
		return ""
	}, Profile{}) {
		t.Fatal("ShouldWarnDeprecated returned false for invalid env value")
	}
}

func TestDeprecatedWarningCanBeDisabledByConfig(t *testing.T) {
	disabled := false
	if ShouldWarnDeprecated(func(string) string { return "" }, Profile{WarnDeprecated: &disabled}) {
		t.Fatal("ShouldWarnDeprecated returned true with config disabled")
	}
}

func TestLoadConfigReadsProfilesWithoutSecrets(t *testing.T) {
	raw := []byte(`{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "81f7728662dd11ec810800155d307d5b",
      "language": "en-GB",
      "registry_url": "https://registry.example.test",
      "registry_public_key": "pubkey-test",
      "endpoint_url": "https://ctapi-global.ctapi.ctyun.cn",
      "ak": "profile-ak",
      "sk": "profile-sk",
      "timeout_seconds": 20
    }
  }
}`)

	cfg, err := Load(raw)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	profile, ok := cfg.ActiveProfile()
	if !ok {
		t.Fatal("ActiveProfile returned false")
	}
	if profile.Region != "81f7728662dd11ec810800155d307d5b" {
		t.Fatalf("Region = %q, want 81f7728662dd11ec810800155d307d5b", profile.Region)
	}
	if profile.Language != "en-GB" {
		t.Fatalf("Language = %q, want en-GB", profile.Language)
	}
	if profile.RegistryURL != "https://registry.example.test" {
		t.Fatalf("RegistryURL = %q", profile.RegistryURL)
	}
	if profile.RegistryPublicKey != "pubkey-test" {
		t.Fatalf("RegistryPublicKey = %q", profile.RegistryPublicKey)
	}
	if profile.EndpointURL != "https://ctapi-global.ctapi.ctyun.cn" {
		t.Fatalf("EndpointURL = %q", profile.EndpointURL)
	}
	if profile.AccessKey != "profile-ak" {
		t.Fatalf("AccessKey = %q", profile.AccessKey)
	}
	if profile.SecretKey != "profile-sk" {
		t.Fatalf("SecretKey = %q", profile.SecretKey)
	}
	if profile.TimeoutSeconds != 20 {
		t.Fatalf("TimeoutSeconds = %d, want 20", profile.TimeoutSeconds)
	}
}

func TestApplyProfileDefaultsUsesGlobalCredentialFallbacks(t *testing.T) {
	disabled := false
	cfg := Config{
		AccessKey:             "global-ak",
		SecretKey:             "global-sk",
		WarnConfigCredentials: &disabled,
		WarnDeprecated:        &disabled,
		Profiles:              map[string]Profile{"prod": {AccessKey: "profile-ak"}},
		ActiveProfileName:     "prod",
	}

	profile, ok := cfg.ActiveProfile()
	if !ok {
		t.Fatal("ActiveProfile returned false")
	}
	profile = cfg.ApplyProfileDefaults(profile)

	if profile.AccessKey != "profile-ak" {
		t.Fatalf("AccessKey = %q, want profile-ak", profile.AccessKey)
	}
	if profile.SecretKey != "global-sk" {
		t.Fatalf("SecretKey = %q, want global-sk", profile.SecretKey)
	}
	if profile.WarnConfigCredentials == nil || *profile.WarnConfigCredentials {
		t.Fatal("WarnConfigCredentials did not inherit disabled global config")
	}
	if profile.WarnDeprecated == nil || *profile.WarnDeprecated {
		t.Fatal("WarnDeprecated did not inherit disabled global config")
	}
}

func TestApplyProfileDefaultsKeepsProfileCredentialSettings(t *testing.T) {
	enabled := true
	disabled := false
	cfg := Config{
		AccessKey:             "global-ak",
		SecretKey:             "global-sk",
		WarnConfigCredentials: &disabled,
		WarnDeprecated:        &disabled,
	}
	profile := cfg.ApplyProfileDefaults(Profile{
		AccessKey:             "profile-ak",
		SecretKey:             "profile-sk",
		WarnConfigCredentials: &enabled,
		WarnDeprecated:        &enabled,
	})

	if profile.AccessKey != "profile-ak" || profile.SecretKey != "profile-sk" {
		t.Fatalf("profile credentials = %#v, want original profile values", profile)
	}
	if profile.WarnConfigCredentials == nil || !*profile.WarnConfigCredentials {
		t.Fatal("WarnConfigCredentials did not keep profile value")
	}
	if profile.WarnDeprecated == nil || !*profile.WarnDeprecated {
		t.Fatal("WarnDeprecated did not keep profile value")
	}
}

func TestLoadConfigReadsTopLevelCredentialFallbacks(t *testing.T) {
	cfg, err := Load([]byte(`{"ak":"global-ak","sk":"global-sk"}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	profile := cfg.ApplyProfileDefaults(Profile{})
	if profile.AccessKey != "global-ak" || profile.SecretKey != "global-sk" {
		t.Fatalf("profile credentials = %#v, want global config credentials", profile)
	}
}

func TestLoadConfigReadsNestedRegistryProfile(t *testing.T) {
	raw := []byte(`{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "81f7728662dd11ec810800155d307d5b",
      "registry": {
        "url": "https://mirror.example.cn/ctyun-cli",
        "public_key": "pubkey-test"
      }
    }
  }
}`)

	cfg, err := Load(raw)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	profile, ok := cfg.ActiveProfile()
	if !ok {
		t.Fatal("ActiveProfile returned false")
	}
	if profile.RegistryURL != "https://mirror.example.cn/ctyun-cli" {
		t.Fatalf("RegistryURL = %q", profile.RegistryURL)
	}
	if profile.RegistryPublicKey != "pubkey-test" {
		t.Fatalf("RegistryPublicKey = %q", profile.RegistryPublicKey)
	}
}

func TestActiveProfileAllowsSingleProfileWithoutName(t *testing.T) {
	cfg, err := Load([]byte(`{
  "profiles": {
    "dev": {"region": "41f64827f25f468595ffa3a5deb5d15d"}
  }
}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	profile, ok := cfg.ActiveProfile()
	if !ok {
		t.Fatal("ActiveProfile returned false")
	}
	if profile.Region != "41f64827f25f468595ffa3a5deb5d15d" {
		t.Fatalf("Region = %q, want 41f64827f25f468595ffa3a5deb5d15d", profile.Region)
	}
}

func TestActiveProfileRejectsMultipleProfilesWithoutName(t *testing.T) {
	cfg, err := Load([]byte(`{
  "profiles": {
    "dev": {"region": "41f64827f25f468595ffa3a5deb5d15d"},
    "prod": {"region": "100054c0416811e9a6690242ac110002"}
  }
}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	_, ok := cfg.ActiveProfile()
	if ok {
		t.Fatal("ActiveProfile returned true for ambiguous profiles")
	}
}

func TestLoadConfigRejectsUnsupportedPersistedSecrets(t *testing.T) {
	raw := []byte(`{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "81f7728662dd11ec810800155d307d5b",
      "secret_token": "must-not-be-here"
    }
  }
}`)

	_, err := Load(raw)
	if err == nil {
		t.Fatal("Load returned nil error for unsupported secret material")
	}
}

func TestLoadConfigRejectsNestedAndArraySecrets(t *testing.T) {
	cases := []string{
		`{"profiles":{"prod":{"metadata":{"secret_token":"bad"}}}}`,
		`{"profiles":{"prod":{"values":[{"secret_key":"bad"}]}}}`,
		`{"profiles":{"prod":{"credentials":[{"access_key":"bad"}]}}}`,
		`{"profiles":{"prod":{"credentials":[{"ak":"bad"}]}}}`,
	}

	for _, raw := range cases {
		if _, err := Load([]byte(raw)); err == nil {
			t.Fatalf("Load returned nil error for secret-bearing config: %s", raw)
		}
	}
}

func TestLoadConfigAllowsArraysWithoutSecrets(t *testing.T) {
	raw := []byte(`{
  "profiles": {
    "prod": {
      "metadata": [
        {"name": "public"},
        "plain-value"
      ]
    }
  }
}`)
	if _, err := Load(raw); err != nil {
		t.Fatalf("Load returned error for non-secret array: %v", err)
	}
}

func TestLoadConfigMalformedJSONReportsParseError(t *testing.T) {
	_, err := Load([]byte(`{"profiles":`))
	if err == nil {
		t.Fatal("Load returned nil error for malformed JSON")
	}
}

func TestActiveProfileReturnsFalseForUnknownConfiguredName(t *testing.T) {
	cfg, err := Load([]byte(`{
  "active_profile": "missing",
  "profiles": {
    "dev": {"region": "41f64827f25f468595ffa3a5deb5d15d"}
  }
}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if _, ok := cfg.ActiveProfile(); ok {
		t.Fatal("ActiveProfile returned true for an unknown active_profile")
	}
}

func TestLoadConfigInitializesMissingProfiles(t *testing.T) {
	cfg, err := Load([]byte(`{}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Profiles == nil {
		t.Fatal("Profiles map was not initialized")
	}
	if _, ok := cfg.ActiveProfile(); ok {
		t.Fatal("ActiveProfile returned true without profiles")
	}
}
