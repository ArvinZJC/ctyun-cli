package config

import "testing"

func TestLoadCredentialsFromEnvUsesOfficialNames(t *testing.T) {
	env := map[string]string{
		"CTYUN_AK": "ak-test",
		"CTYUN_SK": "sk-test",
	}

	creds, err := LoadCredentialsFromEnv(func(key string) string {
		return env[key]
	})
	if err != nil {
		t.Fatalf("LoadCredentialsFromEnv returned error: %v", err)
	}

	if creds.AccessKey != "ak-test" {
		t.Fatalf("AccessKey = %q, want %q", creds.AccessKey, "ak-test")
	}
	if creds.SecretKey != "sk-test" {
		t.Fatalf("SecretKey = %q, want %q", creds.SecretKey, "sk-test")
	}
}

func TestLoadCredentialsFromEnvRejectsMissingValues(t *testing.T) {
	_, err := LoadCredentialsFromEnv(func(string) string { return "" })
	if err == nil {
		t.Fatal("LoadCredentialsFromEnv returned nil error for missing credentials")
	}
}

func TestLoadConfigReadsProfilesWithoutSecrets(t *testing.T) {
	raw := []byte(`{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-huadong1",
      "language": "en-GB",
      "registry_url": "https://registry.example.test",
      "registry_public_key": "pubkey-test",
      "endpoint_url": "https://ctapi-global.ctapi.ctyun.cn",
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
	if profile.Region != "cn-huadong1" {
		t.Fatalf("Region = %q, want cn-huadong1", profile.Region)
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
	if profile.TimeoutSeconds != 20 {
		t.Fatalf("TimeoutSeconds = %d, want 20", profile.TimeoutSeconds)
	}
}

func TestLoadConfigReadsNestedRegistryProfile(t *testing.T) {
	raw := []byte(`{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-huadong1",
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
    "dev": {"region": "cn-dev"}
  }
}`))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	profile, ok := cfg.ActiveProfile()
	if !ok {
		t.Fatal("ActiveProfile returned false")
	}
	if profile.Region != "cn-dev" {
		t.Fatalf("Region = %q, want cn-dev", profile.Region)
	}
}

func TestActiveProfileRejectsMultipleProfilesWithoutName(t *testing.T) {
	cfg, err := Load([]byte(`{
  "profiles": {
    "dev": {"region": "cn-dev"},
    "prod": {"region": "cn-prod"}
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

func TestLoadConfigRejectsPersistedSecrets(t *testing.T) {
	raw := []byte(`{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-huadong1",
      "ak": "must-not-be-here"
    }
  }
}`)

	_, err := Load(raw)
	if err == nil {
		t.Fatal("Load returned nil error for persisted secret material")
	}
}
