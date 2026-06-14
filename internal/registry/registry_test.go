package registry

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindArtifactDefaultsToStableReviewedOrCurated(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "generated", "url": "generated.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "reviewed.tar.gz"},
    {"name": "ecs", "version": "0.3.0", "channel": "beta", "quality": "curated", "url": "beta.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	artifact, ok := idx.Find("ecs", "")
	if !ok {
		t.Fatal("Find returned false")
	}
	if artifact.Version != "0.2.0" {
		t.Fatalf("version = %q, want 0.2.0", artifact.Version)
	}
}

func TestFindArtifactOrdersSemanticVersions(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "old.tar.gz"},
    {"name": "ecs", "version": "0.10.0", "channel": "stable", "quality": "reviewed", "url": "new.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	artifact, ok := idx.Find("ecs", "")
	if !ok {
		t.Fatal("Find returned false")
	}
	if artifact.Version != "0.10.0" {
		t.Fatalf("version = %q, want 0.10.0", artifact.Version)
	}
}

func TestFindArtifactSupportsExplicitChannel(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "stable.tar.gz"},
    {"name": "ecs", "version": "0.3.0", "channel": "beta", "quality": "reviewed", "url": "beta.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	artifact, ok := idx.Find("ecs", "beta")
	if !ok {
		t.Fatal("Find beta returned false")
	}
	if artifact.URL != "beta.tar.gz" {
		t.Fatalf("url = %q, want beta.tar.gz", artifact.URL)
	}
	if _, ok := idx.Find("missing", "beta"); ok {
		t.Fatal("Find returned true for missing plugin")
	}
}

func TestSearchDefaultsToStableReviewedOrCuratedLatestPerPlugin(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "generated", "url": "generated.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "reviewed.tar.gz"},
    {"name": "ecs", "version": "0.3.0", "channel": "stable", "quality": "curated", "url": "curated.tar.gz"},
    {"name": "vpc", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "vpc.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	results := idx.Search("", "")
	if len(results) != 2 {
		t.Fatalf("Search returned %d results, want 2: %#v", len(results), results)
	}
	if results[0].Name != "ecs" || results[0].Version != "0.3.0" {
		t.Fatalf("first result = %#v, want latest curated ecs", results[0])
	}
	for _, artifact := range results {
		if artifact.Quality == "generated" {
			t.Fatalf("Search exposed generated stable artifact: %#v", artifact)
		}
	}
}

func TestSearchFiltersByQuery(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "ecs.tar.gz"},
    {"name": "vpc", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "vpc.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}

	results := idx.Search("vp", "")
	if len(results) != 1 || results[0].Name != "vpc" {
		t.Fatalf("Search query results = %#v, want only vpc", results)
	}
	if strings.Contains(results[0].URL, "ecs") {
		t.Fatalf("Search returned ecs for vpc query: %#v", results[0])
	}

	if results := idx.Search("missing", ""); len(results) != 0 {
		t.Fatalf("Search missing returned %#v, want none", results)
	}
	if results := idx.Search("", "beta"); len(results) != 0 {
		t.Fatalf("Search beta returned %#v, want none", results)
	}
}

func TestSearchSortsSameNameByNewestVersion(t *testing.T) {
	idx, err := LoadIndex([]byte(`{
  "plugins": [
    {"name": "ecs", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "old.tar.gz"},
    {"name": "ecs", "version": "0.2.0", "channel": "stable", "quality": "reviewed", "url": "new.tar.gz"},
    {"name": "abc", "version": "0.1.0", "channel": "stable", "quality": "reviewed", "url": "abc.tar.gz"}
  ]
}`))
	if err != nil {
		t.Fatalf("LoadIndex returned error: %v", err)
	}
	results := idx.Search("", "")
	if len(results) != 2 {
		t.Fatalf("Search returned %#v, want two latest plugin rows", results)
	}
	if results[1].Name != "ecs" || results[1].Version != "0.2.0" {
		t.Fatalf("ecs result = %#v, want latest 0.2.0", results[1])
	}
}

func TestLoadIndexRejectsInvalidArtifactMetadata(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "missing name",
			raw:  `{"plugins":[{"version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "missing name",
		},
		{
			name: "missing version",
			raw:  `{"plugins":[{"name":"ecs","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "missing version",
		},
		{
			name: "bad channel",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"nightly","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "unsupported channel",
		},
		{
			name: "bad quality",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"raw","url":"ecs.tar.gz"}]}`,
			want: "unsupported quality",
		},
		{
			name: "missing url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed"}]}`,
			want: "missing url",
		},
		{
			name: "unsafe name",
			raw:  `{"plugins":[{"name":"../ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"ecs.tar.gz"}]}`,
			want: "invalid plugin name",
		},
		{
			name: "traversal url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"../ecs.tar.gz"}]}`,
			want: "invalid artifact url",
		},
		{
			name: "absolute local url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"/tmp/ecs.tar.gz"}]}`,
			want: "invalid artifact url",
		},
		{
			name: "unsupported url scheme",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"file:///tmp/ecs.tar.gz"}]}`,
			want: "invalid artifact url",
		},
		{
			name: "backslash url",
			raw:  `{"plugins":[{"name":"ecs","version":"0.1.0","channel":"stable","quality":"reviewed","url":"plugins\\ecs.tar.gz"}]}`,
			want: "invalid artifact url",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadIndex([]byte(tc.raw))
			if err == nil {
				t.Fatal("LoadIndex returned nil error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestLoadIndexRejectsMalformedJSON(t *testing.T) {
	_, err := LoadIndex([]byte(`{"plugins":`))
	if err == nil {
		t.Fatal("LoadIndex returned nil error for malformed JSON")
	}
}

func TestVerifySHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(path, []byte("plugin artifact"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	const good = "839524ad87034f8d290fce6b67e9f4581606895a7c490802e1e298b93c5b6c10"
	if err := VerifySHA256(path, good); err != nil {
		t.Fatalf("VerifySHA256 returned error: %v", err)
	}
	if err := VerifySHA256(path, "bad"); err == nil {
		t.Fatal("VerifySHA256 returned nil error for bad checksum")
	}
}

func TestVerifySHA256HandlesOpenErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.tar.gz")
	if err := VerifySHA256(missing, hex.EncodeToString([]byte("bad"))); err == nil {
		t.Fatal("VerifySHA256 returned nil error for missing file")
	}

	if err := VerifySHA256(t.TempDir(), hex.EncodeToString([]byte("bad"))); err == nil {
		t.Fatal("VerifySHA256 returned nil error for directory read")
	}
}

func TestVerifyIndexSignatureAcceptsTrustedEd25519Signature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	index := []byte(`{"plugins":[]}`)
	signature := ed25519.Sign(privateKey, index)

	err = VerifyIndexSignature(
		index,
		[]byte(base64.StdEncoding.EncodeToString(signature)),
		base64.StdEncoding.EncodeToString(publicKey),
	)
	if err != nil {
		t.Fatalf("VerifyIndexSignature returned error: %v", err)
	}
}

func TestVerifyIndexSignatureRejectsMissingKeyOrBadSignature(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	index := []byte(`{"plugins":[]}`)

	if err := VerifyIndexSignature(index, []byte("bad-signature"), ""); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error without a public key")
	}
	if err := VerifyIndexSignature(index, []byte(base64.StdEncoding.EncodeToString([]byte("bad-signature"))), base64.StdEncoding.EncodeToString(publicKey)); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for bad signature")
	}
}

func TestCompareVersionEqualVersions(t *testing.T) {
	if got := compareVersion("0.1.0", "0.1"); got != 0 {
		t.Fatalf("compareVersion equal = %d, want 0", got)
	}
}

func TestVerifyIndexSignatureRejectsMalformedKeyAndWrongLength(t *testing.T) {
	index := []byte(`{"plugins":[]}`)

	if err := VerifyIndexSignature(index, []byte("bad-signature"), "not-base64"); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for malformed key")
	}
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	if err := VerifyIndexSignature(index, []byte("bad-signature"), shortKey); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for short key")
	}
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	if err := VerifyIndexSignature(index, []byte("not-base64"), base64.StdEncoding.EncodeToString(publicKey)); err == nil {
		t.Fatal("VerifyIndexSignature returned nil error for malformed signature")
	}
}
