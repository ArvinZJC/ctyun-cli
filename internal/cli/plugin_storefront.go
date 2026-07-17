/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"maps"
	"net/http"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// listAvailablePlugins renders registry plugins with local installation
// status.
func listAvailablePlugins(stdout io.Writer, root string, source any, channel string, transport http.RoundTripper, publicKey string, opts globalOptions) error {
	_, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	rows, err := availablePluginRows(root, idx.Search("", channel))
	if err != nil {
		return err
	}
	return renderPluginRows(stdout, rows, availablePluginColumns(opts.Language), opts)
}

// listBundledAvailablePlugins renders development bundled plugins with local
// installation status.
func listBundledAvailablePlugins(stdout io.Writer, root string, channel string, opts globalOptions) error {
	idx, err := bundledPluginIndex(opts.Language)
	if err != nil {
		return err
	}
	rows, err := availablePluginRows(root, idx.Search("", bundledPluginChannel(channel)))
	if err != nil {
		return err
	}
	return renderPluginRows(stdout, rows, availablePluginColumns(opts.Language), opts)
}

// searchBundledPlugins renders matching development bundled plugins.
func searchBundledPlugins(stdout io.Writer, root string, channel, query string, opts globalOptions) error {
	idx, err := bundledPluginIndex(opts.Language)
	if err != nil {
		return err
	}
	rows, err := availablePluginRows(root, idx.Search(query, bundledPluginChannel(channel)))
	if err != nil {
		return err
	}
	return renderPluginRows(stdout, rows, availablePluginColumns(opts.Language), opts)
}

// listBundledPluginUpdates prints available updates from development bundled
// plugin metadata.
func listBundledPluginUpdates(stdout io.Writer, root string, channel string, language string) error {
	idx, err := bundledPluginIndex(language)
	if err != nil {
		return err
	}
	return listPluginUpdatesFromIndex(stdout, root, idx, bundledPluginChannel(channel), language)
}

// bundledPluginIndex converts repo-local bundled plugins into registry-shaped
// artifacts for development discovery.
func bundledPluginIndex(language string) (registry.Index, error) {
	if !version.IsDevelopmentBuild() {
		return registry.Index{}, diagnostic.New("error.bundled_dev_only")
	}
	dirs := pluginDirs(defaultPluginRoot())
	artifacts := make([]registry.Artifact, 0, len(dirs))
	for _, dir := range dirs {
		bundle, err := plugin.LoadBundle(dir, version.Version)
		if err != nil {
			return registry.Index{}, err
		}
		artifacts = append(artifacts, registry.Artifact{
			Name:        bundle.Manifest.Name,
			Product:     bundle.Manifest.API.Product,
			DisplayName: localizedPluginText(bundle, language, "name", bundle.Manifest.Name),
			Version:     bundle.Manifest.Version,
			Channel:     bundle.Manifest.Channel,
			Quality:     bundle.Manifest.Quality,
			URL:         dir,
		})
	}
	return registry.Index{Plugins: artifacts}, nil
}

// bundledPluginChannel picks the stable bundle channel when callers do not
// request a specific bundled plugin channel.
func bundledPluginChannel(channel string) string {
	if channel != "" {
		return channel
	}
	return "stable"
}

// availablePluginRows converts registry artifacts into output rows and joins
// installed plugin versions when available.
func availablePluginRows(root string, artifacts []registry.Artifact) ([]map[string]string, error) {
	installed, err := installedPluginVersions(root)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		installedVersion := installed[artifact.Name]
		rows = append(rows, map[string]string{
			"name":              registryDisplayName(artifact),
			"plugin":            artifact.Name,
			"product":           registryProduct(artifact),
			"version":           artifact.Version,
			"channel":           artifact.Channel,
			"quality":           artifact.Quality,
			"status":            pluginInstallStatus(installedVersion, artifact.Version),
			"installed_version": installedVersion,
		})
	}
	return rows, nil
}

// installedPluginVersions maps installed plugin names to their versions.
func installedPluginVersions(root string) (map[string]string, error) {
	if err := validatePluginRootForList(root); err != nil {
		return nil, err
	}
	bundles, err := loadInstalledBundles(root)
	if err != nil {
		return nil, err
	}
	installed := make(map[string]string, len(bundles))
	for _, bundle := range bundles {
		installed[bundle.Manifest.Name] = bundle.Manifest.Version
	}
	return installed, nil
}

// registryDisplayName returns a friendly storefront name with a stable
// fallback for older registry indexes.
func registryDisplayName(artifact registry.Artifact) string {
	if artifact.DisplayName != "" {
		return artifact.DisplayName
	}
	return artifact.Name
}

// registryProduct returns the CTyun product value with a stable fallback for
// older registry indexes.
func registryProduct(artifact registry.Artifact) string {
	if artifact.Product != "" {
		return artifact.Product
	}
	return artifact.Name
}

// pluginInstallStatus summarizes local state for one available registry
// artifact.
func pluginInstallStatus(installedVersion, availableVersion string) string {
	if installedVersion == "" {
		return "available"
	}
	if version.CompareSemanticVersions(availableVersion, installedVersion) > 0 {
		return "outdated"
	}
	return "installed"
}

// renderPluginRows applies standard output controls to plugin storefront rows.
func renderPluginRows(stdout io.Writer, rows []map[string]string, columns []output.Column, opts globalOptions) error {
	rows = localizedPluginStorefrontRows(rows, opts.Language)
	var err error
	opts.Filter, err = output.ResolveFilterExpression(columns, opts.Filter)
	if err != nil {
		return err
	}
	opts.Sort, err = output.ResolveSortExpression(columns, opts.Sort)
	if err != nil {
		return err
	}
	rows, _ = output.FilterRows(rows, opts.Filter)
	rows, _ = output.SortRows(rows, opts.Sort)
	switch opts.Output {
	case "json":
		rendered, err := renderOutputJSON(rows)
		if err != nil {
			return err
		}
		return writeString(stdout, rendered)
	case "table":
		rendered, err := renderTableOutput(stdout, rows, columns, output.TableOptions{
			Columns:  opts.Columns,
			NoHeader: opts.NoHeader,
			Style:    opts.Table,
		})
		if err != nil {
			return err
		}
		return writeString(stdout, rendered)
	default:
		return diagnostic.New("error.unsupported_output", opts.Output)
	}
}

// localizedPluginStorefrontRows localizes display enum values before
// user-facing filters, sorts, and rendering are applied.
func localizedPluginStorefrontRows(rows []map[string]string, language string) []map[string]string {
	localized := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		next := make(map[string]string, len(row))
		maps.Copy(next, row)
		if quality := next["quality"]; quality != "" {
			next["quality"] = pluginQualityText(language, quality)
		}
		if status := next["status"]; status != "" {
			next["status"] = pluginStatusText(language, status)
		}
		localized = append(localized, next)
	}
	return localized
}

// availablePluginColumns returns localized columns for registry storefront
// output.
func availablePluginColumns(language string) []output.Column {
	return []output.Column{
		{Key: "name", Label: helpText("plugin.column.name", language)},
		{Key: "plugin", Label: helpText("plugin.column.plugin", language)},
		{Key: "product", Label: helpText("plugin.column.product", language)},
		{Key: "version", Label: helpText("plugin.column.version", language)},
		{Key: "channel", Label: helpText("plugin.column.channel", language)},
		{Key: "quality", Label: helpText("plugin.column.quality", language)},
		{Key: "status", Label: helpText("plugin.column.status", language)},
		{Key: "installed_version", Label: helpText("plugin.column.installed_version", language)},
	}
}
