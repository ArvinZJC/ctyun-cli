/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// findRegistryArtifact loads the registry index and selects one artifact.
func findRegistryArtifact(source any, name, channel string, transport http.RoundTripper, publicKey string) (distribution.Source, registry.Artifact, error) {
	selectedSource, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return distribution.Source{}, registry.Artifact{}, err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return distribution.Source{}, registry.Artifact{}, err
	}
	artifact, ok := idx.Find(name, channel)
	if !ok {
		return distribution.Source{}, registry.Artifact{}, diagnostic.New("error.plugin_not_found_registry", name)
	}
	return selectedSource, artifact, nil
}

// searchPlugins renders matching artifacts from a registry.
func searchPlugins(stdout io.Writer, root string, source any, channel, query string, transport http.RoundTripper, publicKey string, opts globalOptions) error {
	_, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	rows, err := availablePluginRows(root, idx.Search(query, channel))
	if err != nil {
		return err
	}
	return renderPluginRows(stdout, rows, availablePluginColumns(opts.Language), opts)
}

// reinstallPluginsFromHostedSource reinstalls selected installed plugins from
// one hosted registry read without comparing versions.
func reinstallPluginsFromHostedSource(stdout io.Writer, root string, source distribution.Source, names []string, all bool, channel string, transport http.RoundTripper, publicKey string, language string) error {
	selectedSource, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	targets, err := reinstallTargets(root, names, all)
	if err != nil {
		return err
	}
	for _, target := range targets {
		artifact, ok := idx.Find(target, channel)
		if !ok {
			return diagnostic.New("error.plugin_not_found_registry", target)
		}
		if err := reinstallRegistryArtifact(stdout, root, selectedSource, artifact, transport, language); err != nil {
			return err
		}
	}
	return nil
}

// reinstallRegistryArtifact verifies and reinstalls one selected registry
// artifact.
func reinstallRegistryArtifact(stdout io.Writer, root string, selectedSource distribution.Source, artifact registry.Artifact, transport http.RoundTripper, language string) error {
	if err := installVerifiedRegistryArtifact(root, selectedSource, artifact, transport); err != nil {
		return err
	}
	return writeLine(stdout, pluginReinstalledMessage(language, artifact.Name))
}

// updateAllPlugins installs newer registry artifacts for every installed plugin.
func updateAllPlugins(stdout io.Writer, root string, source any, channel string, transport http.RoundTripper, publicKey string, language string) error {
	selectedSource, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bundlePath := filepath.Join(root, entry.Name())
		bundle, err := plugin.LoadBundle(bundlePath, version.Version)
		if err != nil {
			return err
		}
		artifact, ok := idx.Find(bundle.Manifest.Name, channel)
		if !ok || version.CompareSemanticVersions(artifact.Version, bundle.Manifest.Version) <= 0 {
			continue
		}
		artifactSource, cleanup, err := prepareRegistryArtifact(selectedSource.URL, artifact, transport)
		if err != nil {
			return err
		}
		if err := verifyArtifact(artifactSource, artifact); err != nil {
			cleanup()
			return err
		}
		if _, err := plugin.InstallVerifiedLocalBundle(artifactSource, root, version.Version); err != nil {
			cleanup()
			return err
		}
		cleanup()
		if err := writeLine(stdout, pluginUpdatedMessage(language, bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version)); err != nil {
			return err
		}
	}
	return nil
}

// updateOnePlugin installs a newer registry artifact for one plugin.
func updateOnePlugin(stdout io.Writer, root string, source any, name string, channel string, transport http.RoundTripper, publicKey string, language string) error {
	bundle, err := plugin.LoadBundle(filepath.Join(root, name), version.Version)
	if err != nil {
		return err
	}
	selectedSource, artifact, err := findRegistryArtifact(source, bundle.Manifest.Name, channel, transport, publicKey)
	if err != nil {
		return err
	}
	if version.CompareSemanticVersions(artifact.Version, bundle.Manifest.Version) <= 0 {
		return writeLine(stdout, pluginCurrentMessage(language, bundle.Manifest.Name))
	}
	artifactSource, cleanup, err := prepareRegistryArtifact(selectedSource.URL, artifact, transport)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := verifyArtifact(artifactSource, artifact); err != nil {
		return err
	}
	if _, err := plugin.InstallVerifiedLocalBundle(artifactSource, root, version.Version); err != nil {
		return err
	}
	return writeLine(stdout, pluginUpdatedMessage(language, bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version))
}

// listPluginUpdates prints available updates for installed plugins.
func listPluginUpdates(stdout io.Writer, root string, source any, channel string, transport http.RoundTripper, publicKey string, language string) error {
	_, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	return listPluginUpdatesFromIndex(stdout, root, idx, channel, language)
}

// listPluginUpdatesFromIndex prints updates from an already loaded index.
func listPluginUpdatesFromIndex(stdout io.Writer, root string, idx registry.Index, channel string, language string) error {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bundle, err := plugin.LoadBundle(filepath.Join(root, entry.Name()), version.Version)
		if err != nil {
			return err
		}
		artifact, ok := idx.Find(bundle.Manifest.Name, channel)
		if !ok || version.CompareSemanticVersions(artifact.Version, bundle.Manifest.Version) <= 0 {
			continue
		}
		if err := writeLine(stdout, pluginUpdateAvailableMessage(language, bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version)); err != nil {
			return err
		}
	}
	return nil
}

// installPluginsFromHostedSource installs selected plugins from one hosted
// registry read.
func installPluginsFromHostedSource(stdout io.Writer, root string, source distribution.Source, names []string, all bool, channel string, transport http.RoundTripper, publicKey string, language string) error {
	selectedSource, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	artifacts := make([]registry.Artifact, 0, len(names))
	if all {
		artifacts = idx.Search("", channel)
	} else {
		for _, name := range names {
			artifact, ok := idx.Find(name, channel)
			if !ok {
				return diagnostic.New("error.plugin_not_found_registry", name)
			}
			artifacts = append(artifacts, artifact)
		}
	}
	for _, artifact := range artifacts {
		if err := installRegistryArtifact(stdout, root, selectedSource, artifact, transport, language); err != nil {
			return err
		}
	}
	return nil
}

// installRegistryArtifact verifies and installs one selected registry artifact.
func installRegistryArtifact(stdout io.Writer, root string, selectedSource distribution.Source, artifact registry.Artifact, transport http.RoundTripper, language string) error {
	if err := installVerifiedRegistryArtifact(root, selectedSource, artifact, transport); err != nil {
		return err
	}
	return writeLine(stdout, pluginInstalledMessage(language, artifact.Name))
}

// installVerifiedRegistryArtifact prepares, verifies, and installs one selected
// registry artifact.
func installVerifiedRegistryArtifact(root string, selectedSource distribution.Source, artifact registry.Artifact, transport http.RoundTripper) error {
	artifactSource, cleanup, err := prepareRegistryArtifact(selectedSource.URL, artifact, transport)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := verifyArtifact(artifactSource, artifact); err != nil {
		return err
	}
	if _, err := plugin.InstallVerifiedLocalBundle(artifactSource, root, version.Version); err != nil {
		return err
	}
	return nil
}

// readRegistryIndex reads the first signed hosted registry index.
func readRegistryIndex(source any, transport http.RoundTripper, publicKey string) (distribution.Source, []byte, error) {
	resolved, err := registrySource(source)
	if err != nil {
		return distribution.Source{}, nil, err
	}
	if !distribution.IsHTTPURL(resolved.URL) {
		index, err := os.ReadFile(filepath.Join(resolved.URL, "index.json"))
		return resolved, index, err
	}
	selected, index, err := distribution.ReadSignedIndexWithFallbacks(resolved, "index.json", "index.sig", publicKey, "registry", transport)
	return selected, index, err
}

// registrySource normalizes hosted sources and local test fixture roots.
func registrySource(source any) (distribution.Source, error) {
	switch value := source.(type) {
	case distribution.Source:
		if value.URL == "" {
			return distribution.Source{}, diagnostic.New("error.plugin_source_empty")
		}
		return value, nil
	case string:
		if value == "" {
			return distribution.Source{}, diagnostic.New("error.plugin_source_empty")
		}
		return distribution.Source{Name: "test", URL: value, Kind: distribution.SourceReady}, nil
	default:
		return distribution.Source{}, diagnostic.New("error.unsupported_plugin_source", source)
	}
}

// prepareRegistryArtifact resolves an artifact to a local path and cleanup
// callback.
func prepareRegistryArtifact(registryRoot string, artifact registry.Artifact, transport http.RoundTripper) (string, func(), error) {
	if !distribution.IsHTTPURL(registryRoot) && !distribution.IsHTTPURL(artifact.URL) {
		return filepath.Join(registryRoot, artifact.URL), func() {}, nil
	}
	return distribution.PrepareArtifact(registryRoot, distribution.Artifact{Name: artifact.Name, URL: artifact.URL, SHA256: artifact.SHA256}, transport)
}

// verifyArtifact checks artifact checksums when registry metadata provides one.
func verifyArtifact(path string, artifact registry.Artifact) error {
	if artifact.SHA256 == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return distribution.VerifySHA256(filepath.Join(path, "plugin.json"), artifact.SHA256)
	}
	return distribution.VerifySHA256(path, artifact.SHA256)
}
