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
	"sort"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

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
	return reinstallPluginsFromHostedSourceWithProgress(stdout, io.Discard, root, source, names, all, channel, transport, publicKey, language)
}

// reinstallPluginsFromHostedSourceWithProgress replaces selected installed
// plugins from one hosted registry read and reports one batch result.
func reinstallPluginsFromHostedSourceWithProgress(stdout, stderr io.Writer, root string, source distribution.Source, names []string, all bool, channel string, transport http.RoundTripper, publicKey string, language string) error {
	targets := installedPluginTargets(root, names, all)
	needsIndex := false
	for _, target := range targets {
		if target.Err == nil {
			needsIndex = true
			break
		}
	}
	var selectedSource distribution.Source
	var idx registry.Index
	if needsIndex {
		var indexBytes []byte
		var err error
		selectedSource, indexBytes, err = readRegistryIndex(source, transport, publicKey)
		if err != nil {
			return err
		}
		idx, err = registry.LoadIndex(indexBytes)
		if err != nil {
			return err
		}
	}
	tasks := make([]operationTask, 0, len(targets))
	for _, target := range targets {
		target := target
		tasks = append(tasks, operationTask{
			Target: target.Name,
			Label:  operationProgressLabel(language, "reinstall", target.Name),
			Run: func() operationResult {
				if target.Err != nil {
					return operationResult{Target: target.Name, Err: target.Err}
				}
				artifact, ok := idx.Find(target.Name, channel)
				if !ok {
					return operationResult{Target: target.Name, Err: diagnostic.New("error.plugin_not_found_registry", target.Name)}
				}
				if err := installVerifiedRegistryArtifact(root, selectedSource, artifact, transport); err != nil {
					return operationResult{Target: target.Name, Err: err}
				}
				return operationResult{Target: target.Name, Outcome: operationChanged, OldVersion: target.Bundle.Manifest.Version, NewVersion: artifact.Version}
			},
		})
	}
	return executeAndReportOperationTasks(stdout, stderr, tasks, language, pluginReinstallSummary)
}

// updateAllPlugins installs newer registry artifacts for every installed plugin.
func updateAllPlugins(stdout io.Writer, root string, source any, channel string, transport http.RoundTripper, publicKey string, language string) error {
	return updatePluginsFromHostedSourceWithProgress(stdout, io.Discard, root, source, nil, true, channel, transport, publicKey, language)
}

// updateOnePlugin installs a newer registry artifact for one plugin.
func updateOnePlugin(stdout io.Writer, root string, source any, name string, channel string, transport http.RoundTripper, publicKey string, language string) error {
	return updatePluginsFromHostedSourceWithProgress(stdout, io.Discard, root, source, []string{name}, false, channel, transport, publicKey, language)
}

// updatePluginsFromHostedSourceWithProgress updates installed plugins only
// when the selected registry artifact has a strictly newer semantic version.
func updatePluginsFromHostedSourceWithProgress(stdout, stderr io.Writer, root string, source any, names []string, all bool, channel string, transport http.RoundTripper, publicKey string, language string) error {
	if all {
		if err := validatePluginRootForList(root); err != nil {
			return err
		}
	}
	targets := installedPluginTargets(root, names, all)
	needsIndex := false
	for _, target := range targets {
		if target.Err == nil {
			needsIndex = true
			break
		}
	}
	var selectedSource distribution.Source
	var idx registry.Index
	if needsIndex {
		var indexBytes []byte
		var err error
		selectedSource, indexBytes, err = readRegistryIndex(source, transport, publicKey)
		if err != nil {
			return err
		}
		idx, err = registry.LoadIndex(indexBytes)
		if err != nil {
			return err
		}
	}
	tasks := make([]operationTask, 0, len(targets))
	for _, target := range targets {
		target := target
		tasks = append(tasks, operationTask{
			Target: target.Name,
			Label:  operationProgressLabel(language, "update", target.Name),
			Run: func() operationResult {
				if target.Err != nil {
					return operationResult{Target: target.Name, Err: target.Err}
				}
				artifact, ok := idx.Find(target.Name, channel)
				if !ok {
					if all {
						return operationResult{Target: target.Name, Outcome: operationUnchanged, OldVersion: target.Bundle.Manifest.Version}
					}
					return operationResult{Target: target.Name, Err: diagnostic.New("error.plugin_not_found_registry", target.Name)}
				}
				if version.CompareSemanticVersions(artifact.Version, target.Bundle.Manifest.Version) <= 0 {
					return operationResult{Target: target.Name, Outcome: operationUnchanged, OldVersion: target.Bundle.Manifest.Version, NewVersion: artifact.Version}
				}
				if err := installVerifiedRegistryArtifact(root, selectedSource, artifact, transport); err != nil {
					return operationResult{Target: target.Name, Err: err}
				}
				return operationResult{Target: target.Name, Outcome: operationChanged, OldVersion: target.Bundle.Manifest.Version, NewVersion: artifact.Version}
			},
		})
	}
	return executeAndReportOperationTasks(stdout, stderr, tasks, language, pluginUpdateSummary)
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
	return installPluginsFromHostedSourceWithProgress(stdout, io.Discard, root, source, names, all, channel, transport, publicKey, language)
}

// installPluginsFromHostedSourceWithProgress installs only absent hosted
// plugins from one registry read and reports one structured batch result.
func installPluginsFromHostedSourceWithProgress(stdout, stderr io.Writer, root string, source distribution.Source, names []string, all bool, channel string, transport http.RoundTripper, publicKey string, language string) error {
	installed := make(map[string]bool, len(names))
	precheckErrors := make(map[string]error)
	needsIndex := all
	if !all {
		for _, name := range names {
			_, ok, err := loadInstalledPlugin(root, name)
			installed[name] = ok
			precheckErrors[name] = err
			if err == nil && !ok {
				needsIndex = true
			}
		}
	}

	var selectedSource distribution.Source
	var idx registry.Index
	if needsIndex {
		var indexBytes []byte
		var err error
		selectedSource, indexBytes, err = readRegistryIndex(source, transport, publicKey)
		if err != nil {
			return err
		}
		idx, err = registry.LoadIndex(indexBytes)
		if err != nil {
			return err
		}
	}

	targets := names
	if all {
		artifacts := idx.Search("", channel)
		sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
		targets = make([]string, 0, len(artifacts))
		for _, artifact := range artifacts {
			targets = append(targets, artifact.Name)
		}
	}
	tasks := make([]operationTask, 0, len(targets))
	for _, name := range targets {
		name := name
		tasks = append(tasks, operationTask{
			Target: name,
			Label:  operationProgressLabel(language, "install", name),
			Run: func() operationResult {
				if err := precheckErrors[name]; err != nil {
					return operationResult{Target: name, Err: err}
				}
				isInstalled := installed[name]
				if all {
					_, isInstalled, precheckErrors[name] = loadInstalledPlugin(root, name)
					if precheckErrors[name] != nil {
						return operationResult{Target: name, Err: precheckErrors[name]}
					}
				}
				if isInstalled {
					return operationResult{Target: name, Outcome: operationSkipped}
				}
				artifact, ok := idx.Find(name, channel)
				if !ok {
					return operationResult{Target: name, Err: diagnostic.New("error.plugin_not_found_registry", name)}
				}
				if err := installVerifiedRegistryArtifact(root, selectedSource, artifact, transport); err != nil {
					return operationResult{Target: name, Err: err}
				}
				return operationResult{Target: name, Outcome: operationChanged}
			},
		})
	}
	return executeAndReportOperationTasks(stdout, stderr, tasks, language, pluginInstallSummary)
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
