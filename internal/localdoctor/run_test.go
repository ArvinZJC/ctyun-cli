/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package localdoctor

import "testing"

func TestRunResolvesReportLanguage(t *testing.T) {
	report := Run(Input{UseInjectedConfig: true, LanguageOption: "en-GB", Getenv: func(string) string { return "" }}, DefaultDependencies())
	if report.Language != "en-GB" {
		t.Fatalf("Language = %q, want en-GB", report.Language)
	}
}
