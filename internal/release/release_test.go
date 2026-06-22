/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package release

import (
	"strings"
	"testing"
)

func TestLoadIndexValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "missing schema", raw: `{"releases":[]}`, want: "unsupported release index schema"},
		{name: "missing version", raw: `{"schema":1,"releases":[{"channel":"stable","artifacts":[]}]}`, want: "missing version"},
		{name: "invalid version", raw: `{"schema":1,"releases":[{"version":"v0.2","channel":"stable","artifacts":[]}]}`, want: "invalid version"},
		{name: "bad channel", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"nightly","artifacts":[]}]}`, want: "unsupported channel"},
		{name: "missing artifact", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[]}]}`, want: "has no artifacts"},
		{name: "bad artifact url", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"../ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "invalid artifact url"},
		{name: "bad artifact scheme", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"file:///tmp/ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "invalid artifact url"},
		{name: "absolute artifact path", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"/tmp/ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "invalid artifact url"},
		{name: "backslash artifact path", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"bin\\ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "invalid artifact url"},
		{name: "missing os", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"arch":"arm64","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "missing os"},
		{name: "missing arch", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "missing arch"},
		{name: "missing url", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "missing url"},
		{name: "short sha", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"ctyun.tar.gz","sha256":"BAD"}]}]}`, want: "invalid sha256"},
		{name: "bad sha", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"ctyun.tar.gz","sha256":"` + strings.Repeat("g", 64) + `"}]}]}`, want: "invalid sha256"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadIndex([]byte(tt.raw))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadIndex error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestLoadIndexRejectsMalformedJSON(t *testing.T) {
	if _, err := LoadIndex([]byte(`{`)); err == nil {
		t.Fatal("LoadIndex returned nil error for malformed JSON")
	}
}

func TestFindLatestSelectsPlatformAndChannel(t *testing.T) {
	idx, err := LoadIndex([]byte(`{"schema":1,"releases":[
		{"version":"0.1.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"old.tar.gz","sha256":"` + strings.Repeat("1", 64) + `"}]},
		{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"new.tar.gz","sha256":"` + strings.Repeat("2", 64) + `"}]},
		{"version":"0.3.0","channel":"beta","artifacts":[{"os":"darwin","arch":"arm64","url":"beta.tar.gz","sha256":"` + strings.Repeat("3", 64) + `"}]}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	rel, art, ok := idx.FindLatest("stable", "darwin", "arm64")
	if !ok {
		t.Fatal("FindLatest returned no artifact")
	}
	if rel.Version != "0.2.0" || art.URL != "new.tar.gz" {
		t.Fatalf("FindLatest = %s %s, want 0.2.0 new.tar.gz", rel.Version, art.URL)
	}
}

func TestFindLatestDefaultsChannelAndReportsMissing(t *testing.T) {
	idx, err := LoadIndex([]byte(`{"schema":1,"releases":[
		{"version":"0.2.0","channel":"stable","artifacts":[{"os":"linux","arch":"amd64","url":"linux.tar.gz","sha256":"` + strings.Repeat("4", 64) + `"}]}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	rel, _, ok := idx.FindLatest("", "linux", "amd64")
	if !ok || rel.Version != "0.2.0" {
		t.Fatalf("FindLatest default channel = %s, %v; want 0.2.0 true", rel.Version, ok)
	}
	if _, _, ok := idx.FindLatest("beta", "linux", "amd64"); ok {
		t.Fatal("FindLatest returned beta artifact from stable release")
	}
}

func TestVersionNewer(t *testing.T) {
	if !VersionNewer("0.2.0", "0.1.0-dev") {
		t.Fatal("VersionNewer rejected newer version")
	}
	if !VersionNewer("0.2.0", "0.2.0-beta.1") {
		t.Fatal("VersionNewer rejected stable release after prerelease")
	}
	if VersionNewer("0.1.0", "0.2.0") {
		t.Fatal("VersionNewer accepted older version")
	}
	if VersionNewer("0.1.0", "0.2.0-beta.1") {
		t.Fatal("VersionNewer accepted stable downgrade from newer beta")
	}
	if VersionNewer("0.2.0", "0.2.0") {
		t.Fatal("VersionNewer accepted equal version")
	}
}
