/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpgradeCheckDevelopmentBuildWithoutSource(t *testing.T) {
	var stdout bytes.Buffer
	err := Run(Config{Args: []string{"upgrade", "--check"}, Stdout: &stdout})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "self-upgrade is unavailable for development builds") {
		t.Fatalf("stdout = %q, want development-build guidance", stdout.String())
	}
}

func TestUpgradeCheckUsesExplicitSignedSource(t *testing.T) {
	source, publicKey := writeSignedReleaseSource(t, `{"schema":1,"releases":[{"version":"0.2.0","channel":"stable","artifacts":[{"os":"`+runtime.GOOS+`","arch":"`+runtime.GOARCH+`","url":"ctyun.tar.gz","sha256":"`+strings.Repeat("0", 64)+`"}]}]}`)
	var stdout bytes.Buffer
	err := Run(Config{
		Args:   []string{"upgrade", "--check", "--source", source},
		Stdout: &stdout,
		Env: func(key string) string {
			if key == "CTYUN_RELEASE_PUBLIC_KEY" {
				return publicKey
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "0.2.0") {
		t.Fatalf("stdout = %q, want available version", stdout.String())
	}
}

func writeSignedReleaseSource(t *testing.T, index string) (string, string) {
	t.Helper()
	root := t.TempDir()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, []byte(index))
	if err := os.WriteFile(filepath.Join(root, "core-index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core-index.sig"), []byte(base64.StdEncoding.EncodeToString(signature)), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, base64.StdEncoding.EncodeToString(publicKey)
}
