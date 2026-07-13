/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package config

import (
	"slices"
	"testing"
)

func TestResolvePathReportsWinningSource(t *testing.T) {
	tests := []struct {
		in   PathInput
		path string
		kind SourceKind
	}{
		{PathInput{Option: "/flag", Process: "/process", Environment: "/env", Default: "/default"}, "/flag", SourceOption},
		{PathInput{Process: "/process", Environment: "/env", Default: "/default"}, "/process", SourceProcess},
		{PathInput{Environment: "/env", Default: "/default"}, "/env", SourceEnvironment},
		{PathInput{Default: "/default"}, "/default", SourceDefault},
	}
	for _, tc := range tests {
		path, source := ResolvePath(tc.in)
		if path != tc.path || source.Kind != tc.kind {
			t.Fatalf("ResolvePath() = %q, %#v", path, source)
		}
	}
}

func TestInspectResolvesSettingsInStableOrderWithoutSecrets(t *testing.T) {
	disabled := "false"
	env := map[string]string{
		"CTYUN_AK":                      "environment-ak",
		"CTYUN_LANGUAGE":                "en-GB",
		"CTYUN_WARN_CONFIG_CREDENTIALS": disabled,
	}
	inspection := Inspect(ResolveInput{
		Raw:              []byte(`{"active_profile":"prod","sk":"top-sk","warn_deprecated":false,"profiles":{"prod":{"region":"region-1","language":"zh-CN","endpoint_url":"https://example.test","timeout_seconds":12,"ak":"profile-ak","registry_url":"https://registry.test","registry_public_key":"public-key"}}}`),
		ConfigPath:       "/config.json",
		ConfigPathSource: Source{Kind: SourceOption, Name: "--config"},
		LanguageOption:   "en-US",
		Getenv:           func(key string) string { return env[key] },
		OSLocale:         "zh-CN",
	})
	if inspection.ConfigError != nil || inspection.ProfileError != nil {
		t.Fatalf("Inspect() errors = config %v, profile %v", inspection.ConfigError, inspection.ProfileError)
	}
	wantKeys := []string{"config_path", "profile", "language", "region", "endpoint_url", "timeout_seconds", "ak", "sk", "warn_config_credentials", "warn_deprecated", "registry_url", "registry_public_key"}
	gotKeys := make([]string, 0, len(inspection.Resolution.Settings))
	for _, setting := range inspection.Resolution.Settings {
		gotKeys = append(gotKeys, setting.Key)
		if setting.Sensitive && setting.Value != "" {
			t.Fatalf("sensitive setting %q retained value %q", setting.Key, setting.Value)
		}
	}
	if !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("setting keys = %v, want %v", gotKeys, wantKeys)
	}
	assertSettingSource(t, inspection.Resolution, "config_path", SourceOption)
	assertSettingSource(t, inspection.Resolution, "profile", SourceConfig)
	assertSettingSource(t, inspection.Resolution, "language", SourceOption)
	assertSettingSource(t, inspection.Resolution, "region", SourceProfile)
	assertSettingSource(t, inspection.Resolution, "ak", SourceEnvironment)
	assertSettingSource(t, inspection.Resolution, "sk", SourceConfig)
	assertSettingSource(t, inspection.Resolution, "warn_config_credentials", SourceEnvironment)
	assertSettingSource(t, inspection.Resolution, "warn_deprecated", SourceConfig)
	registry, _ := inspection.Resolution.Setting("registry_url")
	if registry.Effective || !registry.Configured {
		t.Fatalf("registry_url = %#v, want configured inactive setting", registry)
	}
	if !inspection.HasStoredCredentials() {
		t.Fatal("HasStoredCredentials() = false, want true")
	}
}

func TestInspectSeparatesConfigAndProfileErrors(t *testing.T) {
	input := ResolveInput{Raw: []byte(`{"active_profile":"missing","ak":"top-ak"}`), ConfigPath: "/config.json", Getenv: func(string) string { return "" }}
	inspection := Inspect(input)
	if inspection.ProfileError == nil || inspection.ConfigError != nil {
		t.Fatalf("inspection errors = config %v, profile %v", inspection.ConfigError, inspection.ProfileError)
	}
	if _, err := Resolve(input); err == nil {
		t.Fatal("strict Resolve returned nil error")
	}
	for _, setting := range inspection.Resolution.Settings {
		if setting.Sensitive && setting.Value != "" {
			t.Fatalf("sensitive setting %q retained a value", setting.Key)
		}
	}

	malformed := Inspect(ResolveInput{Raw: []byte(`{"profiles":`), Getenv: func(string) string { return "" }})
	if malformed.ConfigError == nil || malformed.ProfileError != nil {
		t.Fatalf("malformed errors = config %v, profile %v", malformed.ConfigError, malformed.ProfileError)
	}

	empty := Inspect(ResolveInput{Getenv: func(string) string { return "" }})
	if empty.ConfigError != nil || empty.ProfileError != nil {
		t.Fatalf("zero-byte errors = config %v, profile %v", empty.ConfigError, empty.ProfileError)
	}
	for _, key := range []string{"region", "endpoint_url", "timeout_seconds", "ak", "sk", "registry_url", "registry_public_key"} {
		assertSettingSource(t, empty.Resolution, key, SourceUnset)
	}

	ambiguous := Inspect(ResolveInput{Raw: []byte(`{"profiles":{"a":{},"b":{}}}`), Getenv: func(string) string { return "" }})
	if ambiguous.ConfigError != nil || ambiguous.ProfileError == nil {
		t.Fatalf("ambiguous errors = config %v, profile %v", ambiguous.ConfigError, ambiguous.ProfileError)
	}
}

func TestInspectSelectsExplicitAndImplicitProfiles(t *testing.T) {
	raw := []byte(`{"profiles":{"dev":{"region":"region-1"}}}`)
	explicit := Inspect(ResolveInput{Raw: raw, ProfileOption: "dev", Getenv: func(string) string { return "" }})
	if explicit.Resolution.ProfileName() != "dev" {
		t.Fatalf("explicit profile = %q", explicit.Resolution.ProfileName())
	}
	assertSettingSource(t, explicit.Resolution, "profile", SourceOption)

	implicit := Inspect(ResolveInput{Raw: raw, Getenv: func(string) string { return "" }})
	if implicit.Resolution.ProfileName() != "dev" {
		t.Fatalf("implicit profile = %q", implicit.Resolution.ProfileName())
	}
	assertSettingSource(t, implicit.Resolution, "profile", SourceImplicit)
}

func TestResolutionAccessorsAndRemainingPrecedenceBranches(t *testing.T) {
	keys := SettingKeys()
	keys[0] = "changed"
	if SettingKeys()[0] != "config_path" {
		t.Fatal("SettingKeys returned shared mutable storage")
	}

	enabled := true
	resolution, err := Resolve(ResolveInput{
		Raw:    []byte(`{"profiles":{"dev":{"warn_deprecated":true,"ak":"profile-ak"}}}`),
		Getenv: func(string) string { return "" },
	})
	if err != nil || resolution.Profile().WarnDeprecated == nil || *resolution.Profile().WarnDeprecated != enabled {
		t.Fatalf("Resolve() = %#v, %v", resolution, err)
	}
	if _, ok := resolution.Setting("missing"); ok {
		t.Fatal("Setting returned true for an unknown key")
	}
	if !Inspect(ResolveInput{Raw: []byte(`{"profiles":{"dev":{"ak":"stored"}}}`)}).HasStoredCredentials() {
		t.Fatal("profile-only stored credential was not detected")
	}
	if Inspect(ResolveInput{}).HasStoredCredentials() {
		t.Fatal("empty config reported stored credentials")
	}
	if _, err := Resolve(ResolveInput{Raw: []byte(`{"profiles":`)}); err == nil {
		t.Fatal("Resolve returned nil for malformed config")
	}
	if _, err := Resolve(ResolveInput{Raw: []byte(`{"profiles":{"dev":{}}}`), ProfileOption: "missing"}); err == nil {
		t.Fatal("Resolve returned nil for a missing explicit profile")
	}
	if got := firstSupportedLanguage(); got.source.Kind != SourceUnset {
		t.Fatalf("firstSupportedLanguage() = %#v", got)
	}
}

func assertSettingSource(t *testing.T, resolution Resolution, key string, want SourceKind) {
	t.Helper()
	setting, ok := resolution.Setting(key)
	if !ok || setting.Source.Kind != want {
		t.Fatalf("setting %q = %#v, want source %q", key, setting, want)
	}
}
