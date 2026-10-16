/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
)

// TestIndexVersionMaps verifies channel isolation, semantic ordering, and
// compatibility with readers that ignore the additive badge metadata.
func TestIndexVersionMaps(t *testing.T) {
	plugins := registry.Index{Plugins: []registry.Artifact{
		{Name: "sample", Version: "1.2.0", Channel: "stable", Quality: "curated", URL: "sample.tar.gz"},
		{Name: "sample", Version: "1.10.0", Channel: "stable", Quality: "curated", URL: "sample-new.tar.gz"},
		{Name: "sample", Version: "2.0.0-beta.2", Channel: "beta", Quality: "curated", URL: "sample-beta.tar.gz"},
		{Name: "other-plugin", Version: "0.1.0", Channel: "stable", Quality: "curated", URL: "other.tar.gz"},
	}}
	core := corerelease.Index{Schema: 1, Releases: []corerelease.Release{
		{Version: "1.2.0", Channel: "stable"}, {Version: "1.10.0", Channel: "stable"},
		{Version: "2.0.0-beta.2", Channel: "beta"},
	}}
	for i := range core.Releases {
		core.Releases[i].Artifacts = []corerelease.Artifact{{OS: "linux", Arch: "amd64", URL: "core.tar.gz", SHA256: strings.Repeat("a", 64)}}
	}
	for range 2 {
		pluginDoc := registryIndexDocument(plugins)
		wantPlugins := map[string]map[string]string{"sample": {"stable": "1.10.0", "beta": "2.0.0-beta.2"}, "other-plugin": {"stable": "0.1.0"}}
		if !reflect.DeepEqual(pluginDoc.Versions, wantPlugins) {
			t.Fatalf("plugin versions: %#v", pluginDoc.Versions)
		}
		raw, err := json.Marshal(pluginDoc)
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := registry.LoadIndex(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(loaded, plugins) {
			t.Fatal("plugin artifact metadata changed")
		}
		coreDoc := coreIndexDocument(core)
		if !reflect.DeepEqual(coreDoc.Versions, map[string]string{"stable": "1.10.0", "beta": "2.0.0-beta.2"}) {
			t.Fatalf("core versions: %#v", coreDoc.Versions)
		}
		raw, err = json.Marshal(coreDoc)
		if err != nil {
			t.Fatal(err)
		}
		loadedCore, err := corerelease.LoadIndex(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(loadedCore, core) {
			t.Fatal("core artifact metadata changed")
		}
		slices.Reverse(plugins.Plugins)
		slices.Reverse(core.Releases)
	}
}

// TestEmptyIndexVersionMaps ensures empty indexes serialize an object rather
// than null, without inventing channel or product versions.
func TestEmptyIndexVersionMaps(t *testing.T) {
	if got := registryIndexDocument(registry.Index{}).Versions; got == nil || len(got) != 0 {
		t.Fatalf("plugin versions: %#v", got)
	}
	if got := coreIndexDocument(corerelease.Index{}).Versions; got == nil || len(got) != 0 {
		t.Fatalf("core versions: %#v", got)
	}
}

// assertPublishedVersion checks a badge path in the actual signed publication bytes.
func assertPublishedVersion(t *testing.T, raw []byte, keys []string, want string) {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	value := document["versions"]
	for _, key := range keys {
		object, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("missing version map at %q", key)
		}
		value = object[key]
	}
	if value != want {
		t.Fatalf("version at %v = %v, want %s", keys, value, want)
	}
}
