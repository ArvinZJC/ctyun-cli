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
		{name: "bad channel", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"nightly","artifacts":[]}]}`, want: "unsupported channel"},
		{name: "missing artifact", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[]}]}`, want: "has no artifacts"},
		{name: "bad artifact url", raw: `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","url":"../ctyun.tar.gz","sha256":"` + strings.Repeat("0", 64) + `"}]}]}`, want: "invalid artifact url"},
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
