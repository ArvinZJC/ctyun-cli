package i18n

import "testing"

func TestResolveLanguageOrderAndFallbacks(t *testing.T) {
	tests := []struct {
		name    string
		options LanguageOptions
		want    string
	}{
		{name: "flag wins", options: LanguageOptions{Flag: "en-GB", Env: "zh-CN", Profile: "en-US", OSLocale: "zh"}, want: "en-GB"},
		{name: "env second", options: LanguageOptions{Env: "en-US", Profile: "zh-CN", OSLocale: "en-AU"}, want: "en-US"},
		{name: "profile third", options: LanguageOptions{Profile: "en-GB", OSLocale: "zh-CN"}, want: "en-GB"},
		{name: "locale suffix is ignored", options: LanguageOptions{OSLocale: "en_GB.UTF-8"}, want: "en-GB"},
		{name: "english commonwealth maps to en-GB", options: LanguageOptions{OSLocale: "en-AU"}, want: "en-GB"},
		{name: "other english maps to en-US", options: LanguageOptions{OSLocale: "en-SG"}, want: "en-US"},
		{name: "zh family maps to zh-CN", options: LanguageOptions{OSLocale: "zh-Hans"}, want: "zh-CN"},
		{name: "unknown falls back to zh-CN", options: LanguageOptions{OSLocale: "fr-FR"}, want: "zh-CN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveLanguage(tt.options); got != tt.want {
				t.Fatalf("ResolveLanguage(%+v) = %q, want %q", tt.options, got, tt.want)
			}
		})
	}
}

func TestCatalogLooksUpLanguageThenFallback(t *testing.T) {
	catalog := Catalog{
		"instance_id": {
			"zh-CN": "实例ID",
			"en-GB": "Instance ID",
		},
	}

	if got := catalog.Text("instance_id", "en-US"); got != "实例ID" {
		t.Fatalf("fallback text = %q, want zh-CN label", got)
	}
	if got := catalog.Text("instance_id", "en-GB"); got != "Instance ID" {
		t.Fatalf("language text = %q, want en-GB label", got)
	}
	if got := catalog.Text("missing", "en-US"); got != "missing" {
		t.Fatalf("missing text = %q, want key", got)
	}
}

func TestCatalogFallsBackToKeyWhenNoUsableLabelExists(t *testing.T) {
	catalog := Catalog{
		"status": {
			"en-GB": "Status",
		},
	}

	if got := catalog.Text("status", "en-US"); got != "status" {
		t.Fatalf("fallback text = %q, want key", got)
	}
}

func TestMatchLanguageTrimsWhitespaceAndAcceptsChineseRegionalLocales(t *testing.T) {
	if got := ResolveLanguage(LanguageOptions{Flag: " zh-HK.UTF-8 "}); got != "zh-CN" {
		t.Fatalf("ResolveLanguage zh-HK = %q, want zh-CN", got)
	}
}
