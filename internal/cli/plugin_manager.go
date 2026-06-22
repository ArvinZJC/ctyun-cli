/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// runPlugin runs plugin-manager commands with the default English language.
func runPlugin(stdout io.Writer, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper) error {
	return runPluginWithLanguage(stdout, root, args, profile, getenv, transport, "en-US")
}

// runPluginWithLanguage runs plugin-manager commands with a selected language.
func runPluginWithLanguage(stdout io.Writer, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, language string) error {
	return runPluginWithOptions(stdout, io.Discard, strings.NewReader(""), root, args, profile, getenv, transport, globalOptions{Output: "table", Language: language})
}

// runPluginWithOptions dispatches plugin install, list, search, remove, lint,
// and update commands. Lint is intentionally hidden from public help because it
// validates local bundle directories for contributor workflows.
func runPluginWithOptions(stdout, stderr io.Writer, stdin io.Reader, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, global globalOptions) error {
	if len(args) == 0 {
		return diagnostic.New("error.plugin_subcommand")
	}
	if global.Output == "" {
		global.Output = "table"
	}
	publicKey := pluginPublicKey(getenv)
	switch args[0] {
	case "install":
		opts, err := parsePluginInstallOptions(args[1:])
		if err != nil {
			return err
		}
		if !opts.All && len(opts.Names) == 0 {
			return diagnostic.New("error.plugin_install_name")
		}
		if opts.Bundled {
			return installBundledPlugins(stdout, root, opts.Names, opts.All, global.Language)
		}
		source, err := resolvePluginSource(opts.Source, getenv, profile)
		if err != nil {
			return err
		}
		return installPluginsFromHostedSource(stdout, root, source, opts.Names, opts.All, opts.Channel, transport, publicKey, global.Language)
	case "list":
		opts, err := parsePluginListOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Available {
			if opts.Bundled {
				return listBundledAvailablePlugins(stdout, root, opts.Channel, global)
			}
			source, err := resolvePluginSource(opts.Source, getenv, profile)
			if err != nil {
				return err
			}
			return listAvailablePlugins(stdout, root, source, opts.Channel, transport, publicKey, global)
		}
		if opts.Updates {
			if opts.Bundled {
				return listBundledPluginUpdates(stdout, root, opts.Channel, global.Language)
			}
			source, err := resolvePluginSource(opts.Source, getenv, profile)
			if err != nil {
				return err
			}
			return listPluginUpdates(stdout, root, source, opts.Channel, transport, publicKey, global.Language)
		}
		return listPlugins(stdout, root, global)
	case "search":
		opts, err := parsePluginSearchOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Bundled {
			return searchBundledPlugins(stdout, root, opts.Channel, opts.Query, global)
		}
		source, err := resolvePluginSource(opts.Source, getenv, profile)
		if err != nil {
			return err
		}
		return searchPlugins(stdout, root, source, opts.Channel, opts.Query, transport, publicKey, global)
	case "remove":
		opts, err := parsePluginRemoveOptions(args[1:])
		if err != nil {
			return err
		}
		return removePlugins(stdout, stderr, stdin, root, opts, global)
	case "lint":
		if !version.IsDevelopmentBuild() {
			return diagnostic.New("error.plugin_lint_dev_only")
		}
		if len(args) != 2 {
			return diagnostic.New("error.plugin_lint_path")
		}
		bundle, err := plugin.LoadBundle(args[1], version.Version)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, pluginValidMessage(global.Language, bundle.Manifest.Name, bundle.Manifest.Version))
		return nil
	case "update", "upgrade":
		opts, err := parsePluginUpdateOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Bundled {
			if opts.All {
				return updateAllBundledPlugins(stdout, root, global.Language)
			}
			if opts.Name != "" {
				return updateOneBundledPlugin(stdout, root, opts.Name, global.Language)
			}
			return diagnostic.New("error.plugin_bundled_update_target")
		}
		source, err := resolvePluginSource(opts.Source, getenv, profile)
		if err != nil {
			return err
		}
		if opts.All {
			return updateAllPlugins(stdout, root, source, opts.Channel, transport, publicKey, global.Language)
		}
		if opts.Name != "" {
			return updateOnePlugin(stdout, root, source, opts.Name, opts.Channel, transport, publicKey, global.Language)
		}
		return diagnostic.New("error.plugin_update_target")
	default:
		return diagnostic.New("error.unknown_plugin_subcommand", args[0])
	}
}

// resolvePluginSource applies hosted source precedence for plugin metadata.
func resolvePluginSource(flag string, getenv func(string) string, profile coreconfig.Profile) (distribution.Source, error) {
	requested := flag
	if requested == "" && getenv != nil {
		requested = getenv("CTYUN_PLUGIN_SOURCE")
	}
	source, err := distribution.ResolveSource(distribution.SourceOptions{
		Label:            "plugin",
		Requested:        requested,
		DevelopmentBuild: version.IsDevelopmentBuild(),
		GitHubURL:        registry.GitHubPluginSource,
		GiteeURL:         registry.GiteePluginSource,
		DisableDevAuto:   true,
		Getenv:           getenv,
	})
	if err != nil {
		return distribution.Source{}, err
	}
	if source.Kind == distribution.SourceDevelopmentUnavailable {
		return distribution.Source{}, diagnostic.New("error.hosted_plugin_dev")
	}
	return source, nil
}

// pluginPublicKey returns the trusted hosted plugin index public key.
func pluginPublicKey(getenv func(string) string) string {
	return releasePublicKey(getenv)
}

// pluginInstallOptions captures plugin install flags and source.
type pluginInstallOptions struct {
	Names   []string
	All     bool
	Source  string
	Channel string
	Bundled bool
}

// parsePluginInstallOptions parses plugin install arguments.
func parsePluginInstallOptions(args []string) (pluginInstallOptions, error) {
	var opts pluginInstallOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--source":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.requires_value", args[i-1])
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.channel_requires_value")
			}
			opts.Channel = args[i]
		case "--bundled":
			opts.Bundled = true
		default:
			opts.Names = append(opts.Names, args[i])
		}
	}
	if opts.All && len(opts.Names) > 0 {
		return opts, diagnostic.New("error.plugin_install_all_or_names")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_install_source_choice")
	}
	return opts, nil
}

// pluginUpdateOptions captures plugin update or upgrade flags.
type pluginUpdateOptions struct {
	All     bool
	Name    string
	Source  string
	Channel string
	Bundled bool
}

// pluginSearchOptions captures plugin search flags and query.
type pluginSearchOptions struct {
	Query   string
	Source  string
	Channel string
	Bundled bool
}

// pluginRemoveOptions captures plugin remove flags and targets.
type pluginRemoveOptions struct {
	Names []string
	All   bool
}

// parsePluginSearchOptions parses plugin search arguments.
func parsePluginSearchOptions(args []string) (pluginSearchOptions, error) {
	var opts pluginSearchOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.requires_value", args[i-1])
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.channel_requires_value")
			}
			opts.Channel = args[i]
		case "--bundled":
			opts.Bundled = true
		default:
			if opts.Query != "" {
				return opts, diagnostic.New("error.plugin_search_one_query")
			}
			opts.Query = args[i]
		}
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_search_source_choice")
	}
	return opts, nil
}

// parsePluginRemoveOptions parses plugin remove arguments.
func parsePluginRemoveOptions(args []string) (pluginRemoveOptions, error) {
	var opts pluginRemoveOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		default:
			opts.Names = append(opts.Names, args[i])
		}
	}
	if opts.All && len(opts.Names) > 0 {
		return opts, diagnostic.New("error.plugin_remove_all_or_names")
	}
	if !opts.All && len(opts.Names) == 0 {
		return opts, diagnostic.New("error.plugin_remove_name")
	}
	for _, name := range opts.Names {
		if !plugin.ValidName(name) {
			return opts, diagnostic.New("error.plugin_name", name)
		}
	}
	return opts, nil
}

// removePlugins removes selected installed plugins after confirmation.
func removePlugins(stdout, stderr io.Writer, stdin io.Reader, root string, opts pluginRemoveOptions, global globalOptions) error {
	if err := confirmDangerousOperation(stderr, stdin, global, "plugin remove"); err != nil {
		return err
	}
	names := opts.Names
	if opts.All {
		dirs := pluginDirs(root)
		names = make([]string, 0, len(dirs))
		for _, dir := range dirs {
			names = append(names, filepath.Base(dir))
		}
	}
	for _, name := range names {
		if !plugin.ValidName(name) {
			return diagnostic.New("error.plugin_name", name)
		}
		if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
			return err
		}
		fmt.Fprintln(stdout, pluginRemovedMessage(global.Language, name))
	}
	return nil
}

// parsePluginUpdateOptions parses plugin update or upgrade arguments.
func parsePluginUpdateOptions(args []string) (pluginUpdateOptions, error) {
	var opts pluginUpdateOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--source":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.requires_value", args[i-1])
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.channel_requires_value")
			}
			opts.Channel = args[i]
		case "--bundled":
			opts.Bundled = true
		default:
			if opts.Name != "" {
				return opts, diagnostic.New("error.plugin_update_one_name")
			}
			opts.Name = args[i]
		}
	}
	if opts.All && opts.Name != "" {
		return opts, diagnostic.New("error.plugin_update_all_or_one")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_update_source_choice")
	}
	return opts, nil
}

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
		if _, err := installPluginSource(artifactSource, root); err != nil {
			cleanup()
			return err
		}
		cleanup()
		fmt.Fprintln(stdout, pluginUpdatedMessage(language, bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version))
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
		fmt.Fprintln(stdout, pluginCurrentMessage(language, bundle.Manifest.Name))
		return nil
	}
	artifactSource, cleanup, err := prepareRegistryArtifact(selectedSource.URL, artifact, transport)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := verifyArtifact(artifactSource, artifact); err != nil {
		return err
	}
	if _, err := installPluginSource(artifactSource, root); err != nil {
		return err
	}
	fmt.Fprintln(stdout, pluginUpdatedMessage(language, bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version))
	return nil
}

// pluginListOptions captures plugin list flags.
type pluginListOptions struct {
	Updates   bool
	Available bool
	Source    string
	Channel   string
	Bundled   bool
}

// parsePluginListOptions parses plugin list arguments.
func parsePluginListOptions(args []string) (pluginListOptions, error) {
	var opts pluginListOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--updates":
			opts.Updates = true
		case "--available":
			opts.Available = true
		case "--source":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.requires_value", args[i-1])
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.channel_requires_value")
			}
			opts.Channel = args[i]
		case "--bundled":
			opts.Bundled = true
		default:
			return opts, diagnostic.New("error.unknown_plugin_list_option", args[i])
		}
	}
	if opts.Available && opts.Updates {
		return opts, diagnostic.New("error.plugin_list_available_updates")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_list_source_choice")
	}
	return opts, nil
}

// listPlugins renders installed plugin metadata.
func listPlugins(stdout io.Writer, root string, opts globalOptions) error {
	if err := validatePluginRootForList(root); err != nil {
		return err
	}
	bundles, err := loadInstalledBundles(root)
	if err != nil {
		return err
	}
	rows := make([]map[string]string, 0, len(bundles))
	for _, bundle := range bundles {
		rows = append(rows, map[string]string{
			"name":       localizedPluginText(bundle, opts.Language, "name", bundle.Manifest.Name),
			"plugin":     bundle.Manifest.Name,
			"product":    bundle.Manifest.API.Product,
			"version":    bundle.Manifest.Version,
			"channel":    bundle.Manifest.Channel,
			"quality":    bundle.Manifest.Quality,
			"commands":   strconv.Itoa(len(bundle.Commands.Commands)),
			"operations": strconv.Itoa(len(bundle.APIs.Operations)),
		})
	}
	columns := pluginListColumns(opts.Language)
	rows = localizedPluginStorefrontRows(rows, opts.Language)
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
		fmt.Fprint(stdout, rendered)
		return nil
	case "table":
		rendered, err := renderOutputTable(rows, columns, output.TableOptions{
			Columns:  opts.Columns,
			NoHeader: opts.NoHeader,
			Style:    opts.Table,
		})
		if err != nil {
			return err
		}
		fmt.Fprint(stdout, rendered)
		return nil
	default:
		return diagnostic.New("error.unsupported_output", opts.Output)
	}
}

// loadInstalledBundles loads every plugin bundle in the user plugin root.
func loadInstalledBundles(root string) ([]plugin.Bundle, error) {
	dirs := pluginDirs(root)
	bundles := make([]plugin.Bundle, 0, len(dirs))
	for _, dir := range dirs {
		bundle, err := plugin.LoadBundle(dir, version.Version)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}

// validatePluginRootForList verifies that an existing plugin root is a
// directory.
func validatePluginRootForList(root string) error {
	info, err := osStat(root)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if !info.IsDir() {
		return diagnostic.New("error.plugin_root_not_directory", root)
	}
	return nil
}

// pluginListColumns returns localized columns for plugin list output.
func pluginListColumns(language string) []output.Column {
	return []output.Column{
		{Key: "name", Label: helpText("plugin.column.name", language)},
		{Key: "plugin", Label: helpText("plugin.column.plugin", language)},
		{Key: "product", Label: helpText("plugin.column.product", language)},
		{Key: "version", Label: helpText("plugin.column.version", language)},
		{Key: "channel", Label: helpText("plugin.column.channel", language)},
		{Key: "quality", Label: helpText("plugin.column.quality", language)},
		{Key: "commands", Label: helpText("plugin.column.commands", language)},
		{Key: "operations", Label: helpText("plugin.column.operations", language)},
	}
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
		fmt.Fprintln(stdout, pluginUpdateAvailableMessage(language, bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version))
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
	artifactSource, cleanup, err := prepareRegistryArtifact(selectedSource.URL, artifact, transport)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := verifyArtifact(artifactSource, artifact); err != nil {
		return err
	}
	if _, err := installPluginSource(artifactSource, root); err != nil {
		return err
	}
	fmt.Fprintln(stdout, pluginInstalledMessage(language, artifact.Name))
	return nil
}

// installBundledPlugins installs development bundled plugins by name or all
// bundled plugin directories.
func installBundledPlugins(stdout io.Writer, root string, names []string, all bool, language string) error {
	if all {
		if !version.IsDevelopmentBuild() {
			return diagnostic.New("error.bundled_dev_only")
		}
		for _, dir := range pluginDirs(defaultPluginRoot()) {
			bundle, err := plugin.LoadBundle(dir, version.Version)
			if err != nil {
				return err
			}
			if _, err := installPluginSource(dir, root); err != nil {
				return err
			}
			fmt.Fprintln(stdout, pluginInstalledMessage(language, bundle.Manifest.Name))
		}
		return nil
	}
	for _, name := range names {
		source, err := bundledPluginSource(name)
		if err != nil {
			return err
		}
		if _, err := installPluginSource(source, root); err != nil {
			return err
		}
		fmt.Fprintln(stdout, pluginInstalledMessage(language, name))
	}
	return nil
}

// installPluginSource installs a verified local plugin bundle or archive.
func installPluginSource(source, root string) (string, error) {
	return plugin.InstallVerifiedLocalBundle(source, root, version.Version)
}

// bundledPluginSource resolves a bundled plugin name for development installs.
func bundledPluginSource(name string) (string, error) {
	if !version.IsDevelopmentBuild() {
		return "", diagnostic.New("error.bundled_dev_only")
	}
	if !plugin.ValidName(name) {
		return "", diagnostic.New("error.plugin_name", name)
	}
	source := filepath.Join(defaultPluginRoot(), name)
	if _, err := os.Stat(source); err != nil {
		return "", err
	}
	return source, nil
}

// updateOneBundledPlugin updates one installed plugin from bundled metadata.
func updateOneBundledPlugin(stdout io.Writer, root, name string, language string) error {
	bundle, err := plugin.LoadBundle(filepath.Join(root, name), version.Version)
	if err != nil {
		return err
	}
	source, err := bundledPluginSource(bundle.Manifest.Name)
	if err != nil {
		return err
	}
	bundled, err := plugin.LoadBundle(source, version.Version)
	if err != nil {
		return err
	}
	if version.CompareSemanticVersions(bundled.Manifest.Version, bundle.Manifest.Version) <= 0 {
		fmt.Fprintln(stdout, pluginCurrentMessage(language, bundle.Manifest.Name))
		return nil
	}
	if _, err := installPluginSource(source, root); err != nil {
		return err
	}
	fmt.Fprintln(stdout, pluginUpdatedMessage(language, bundle.Manifest.Name, bundle.Manifest.Version, bundled.Manifest.Version))
	return nil
}

// updateAllBundledPlugins updates installed plugins from bundled metadata.
func updateAllBundledPlugins(stdout io.Writer, root string, language string) error {
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
		if err := updateOneBundledPlugin(stdout, root, entry.Name(), language); err != nil {
			return err
		}
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

// downloadRegistryArtifact downloads an artifact to a temporary file.
func downloadRegistryArtifact(artifactURL string, transport http.RoundTripper) (string, func(), error) {
	data, err := httpGetBytes(artifactURL, transport)
	if err != nil {
		return "", func() {}, err
	}
	tmp, err := createTempArtifactFile()
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmp.Name(), cleanup, nil
}

// httpGetBytes fetches an HTTP URL and returns successful response bytes.
func httpGetBytes(rawURL string, transport http.RoundTripper) ([]byte, error) {
	return distribution.HTTPGetBytes(rawURL, transport)
}

// joinRegistryURL appends name to the path portion of a registry URL.
func joinRegistryURL(root, name string) string {
	return distribution.JoinURL(root, name)
}

// isHTTPURL reports whether value has an HTTP or HTTPS scheme.
func isHTTPURL(value string) bool {
	return distribution.IsHTTPURL(value)
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
		return registry.VerifySHA256(filepath.Join(path, "plugin.json"), artifact.SHA256)
	}
	return registry.VerifySHA256(path, artifact.SHA256)
}
