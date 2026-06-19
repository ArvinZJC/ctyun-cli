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
	return runPluginWithOptions(stdout, root, args, profile, getenv, transport, globalOptions{Output: "table", Language: language})
}

// runPluginWithOptions dispatches plugin install, list, search, remove, lint,
// and update commands.
func runPluginWithOptions(stdout io.Writer, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, global globalOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("plugin requires a subcommand")
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
		if opts.Name == "" {
			return fmt.Errorf("plugin install requires a plugin name")
		}
		if opts.Bundled {
			source, err := bundledPluginSource(opts.Name)
			if err != nil {
				return err
			}
			if _, err := installPluginSource(source, root); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "installed %s\n", opts.Name)
			return nil
		}
		source, err := resolvePluginSource(opts.Source, getenv, profile)
		if err != nil {
			return err
		}
		return installPluginFromHostedSource(stdout, root, source, opts.Name, opts.Channel, transport, publicKey)
	case "list":
		opts, err := parsePluginListOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Updates {
			source, err := resolvePluginSource(opts.Source, getenv, profile)
			if err != nil {
				return err
			}
			return listPluginUpdates(stdout, root, source, transport, publicKey)
		}
		return listPlugins(stdout, root, global)
	case "search":
		opts, err := parsePluginSearchOptions(args[1:])
		if err != nil {
			return err
		}
		source, err := resolvePluginSource(opts.Source, getenv, profile)
		if err != nil {
			return err
		}
		return searchPlugins(stdout, source, opts.Channel, opts.Query, transport, publicKey)
	case "remove":
		if len(args) != 2 {
			return fmt.Errorf("plugin remove requires a plugin name")
		}
		if !plugin.ValidName(args[1]) {
			return fmt.Errorf("invalid plugin name %q", args[1])
		}
		if err := os.RemoveAll(filepath.Join(root, args[1])); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "removed %s\n", args[1])
		return nil
	case "lint":
		if len(args) != 2 {
			return fmt.Errorf("plugin lint requires a bundle path")
		}
		bundle, err := plugin.LoadBundle(args[1], version.Version)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "valid %s %s\n", bundle.Manifest.Name, bundle.Manifest.Version)
		return nil
	case "update", "upgrade":
		opts, err := parsePluginUpdateOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Bundled {
			if opts.All {
				return updateAllBundledPlugins(stdout, root)
			}
			if opts.Name != "" {
				return updateOneBundledPlugin(stdout, root, opts.Name)
			}
			return fmt.Errorf("plugin update/upgrade --bundled requires a plugin name or --all")
		}
		source, err := resolvePluginSource(opts.Source, getenv, profile)
		if err != nil {
			return err
		}
		if opts.All {
			return updateAllPlugins(stdout, root, source, transport, publicKey)
		}
		if opts.Name != "" {
			return updateOnePlugin(stdout, root, source, opts.Name, transport, publicKey)
		}
		return fmt.Errorf("plugin update/upgrade requires a plugin name or --all")
	default:
		return fmt.Errorf("unknown plugin subcommand %q", args[0])
	}
}

// resolvePluginSource applies hosted source precedence for plugin metadata.
func resolvePluginSource(flag string, getenv func(string) string, profile coreconfig.Profile) (distribution.Source, error) {
	requested := flag
	if requested == "" && getenv != nil {
		requested = getenv("CTYUN_PLUGIN_SOURCE")
	}
	source, err := distribution.ResolveSource(distribution.SourceOptions{
		Label:          "plugin",
		Requested:      requested,
		CurrentVersion: version.Version,
		GitHubURL:      registry.GitHubPluginSource,
		GiteeURL:       registry.GiteePluginSource,
		DisableDevAuto: true,
		Getenv:         getenv,
	})
	if err != nil {
		return distribution.Source{}, err
	}
	if source.Kind == distribution.SourceDevelopmentUnavailable {
		return distribution.Source{}, fmt.Errorf("hosted plugin updates are unavailable for development builds; use --bundled")
	}
	return source, nil
}

// pluginPublicKey returns the trusted hosted plugin index public key.
func pluginPublicKey(getenv func(string) string) string {
	return releasePublicKey(getenv)
}

// pluginInstallOptions captures plugin install flags and source.
type pluginInstallOptions struct {
	Name    string
	Source  string
	Channel string
	Bundled bool
}

// parsePluginInstallOptions parses plugin install arguments.
func parsePluginInstallOptions(args []string) (pluginInstallOptions, error) {
	var opts pluginInstallOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("%s requires a value", args[i-1])
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--channel requires a value")
			}
			opts.Channel = args[i]
		case "--bundled":
			opts.Bundled = true
		default:
			if opts.Name != "" {
				return opts, fmt.Errorf("plugin install accepts one plugin name")
			}
			opts.Name = args[i]
		}
	}
	if opts.Bundled && opts.Source != "" {
		return opts, fmt.Errorf("plugin install accepts either --bundled or --source")
	}
	return opts, nil
}

// pluginUpdateOptions captures plugin update or upgrade flags.
type pluginUpdateOptions struct {
	All     bool
	Name    string
	Source  string
	Bundled bool
}

// pluginSearchOptions captures plugin search flags and query.
type pluginSearchOptions struct {
	Query   string
	Source  string
	Channel string
}

// parsePluginSearchOptions parses plugin search arguments.
func parsePluginSearchOptions(args []string) (pluginSearchOptions, error) {
	var opts pluginSearchOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("%s requires a value", args[i-1])
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--channel requires a value")
			}
			opts.Channel = args[i]
		default:
			if opts.Query != "" {
				return opts, fmt.Errorf("plugin search accepts one query")
			}
			opts.Query = args[i]
		}
	}
	return opts, nil
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
				return opts, fmt.Errorf("%s requires a value", args[i-1])
			}
			opts.Source = args[i]
		case "--bundled":
			opts.Bundled = true
		default:
			if opts.Name != "" {
				return opts, fmt.Errorf("plugin update accepts one plugin name")
			}
			opts.Name = args[i]
		}
	}
	if opts.All && opts.Name != "" {
		return opts, fmt.Errorf("plugin update accepts either --all or one plugin name")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, fmt.Errorf("plugin update accepts either --bundled or --source")
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
		return distribution.Source{}, registry.Artifact{}, fmt.Errorf("plugin %s not found in registry", name)
	}
	return selectedSource, artifact, nil
}

// searchPlugins prints matching artifacts from a registry.
func searchPlugins(stdout io.Writer, source any, channel, query string, transport http.RoundTripper, publicKey string) error {
	_, indexBytes, err := readRegistryIndex(source, transport, publicKey)
	if err != nil {
		return err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	for _, artifact := range idx.Search(query, channel) {
		fmt.Fprintf(stdout, "%s %s %s %s\n", artifact.Name, artifact.Version, artifact.Channel, artifact.Quality)
	}
	return nil
}

// updateAllPlugins installs newer registry artifacts for every installed plugin.
func updateAllPlugins(stdout io.Writer, root string, source any, transport http.RoundTripper, publicKey string) error {
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
		artifact, ok := idx.Find(bundle.Manifest.Name, "")
		if !ok || compareVersion(artifact.Version, bundle.Manifest.Version) <= 0 {
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
		fmt.Fprintf(stdout, "updated %s %s -> %s\n", bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version)
	}
	return nil
}

// updateOnePlugin installs a newer registry artifact for one plugin.
func updateOnePlugin(stdout io.Writer, root string, source any, name string, transport http.RoundTripper, publicKey string) error {
	bundle, err := plugin.LoadBundle(filepath.Join(root, name), version.Version)
	if err != nil {
		return err
	}
	selectedSource, artifact, err := findRegistryArtifact(source, bundle.Manifest.Name, "", transport, publicKey)
	if err != nil {
		return err
	}
	if compareVersion(artifact.Version, bundle.Manifest.Version) <= 0 {
		fmt.Fprintf(stdout, "%s is up to date\n", bundle.Manifest.Name)
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
	fmt.Fprintf(stdout, "updated %s %s -> %s\n", bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version)
	return nil
}

// pluginListOptions captures plugin list flags.
type pluginListOptions struct {
	Updates bool
	Source  string
}

// parsePluginListOptions parses plugin list arguments.
func parsePluginListOptions(args []string) (pluginListOptions, error) {
	var opts pluginListOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--updates":
			opts.Updates = true
		case "--source":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("%s requires a value", args[i-1])
			}
			opts.Source = args[i]
		default:
			return opts, fmt.Errorf("unknown plugin list option %q", args[i])
		}
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
	if err := validateOutputControlKeys(columns, opts.Filter, opts.Sort); err != nil {
		return err
	}
	rows, err = output.FilterRows(rows, opts.Filter)
	if err != nil {
		return err
	}
	rows, err = output.SortRows(rows, opts.Sort)
	if err != nil {
		return err
	}
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
		return fmt.Errorf("unsupported output %q", opts.Output)
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
		return fmt.Errorf("plugin root %s is not a directory", root)
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

// validateOutputControlKeys checks list filters and sorts against stable keys.
func validateOutputControlKeys(columns []output.Column, filter, sort string) error {
	keys := make(map[string]bool, len(columns))
	for _, column := range columns {
		keys[column.Key] = true
	}
	if key := filterKey(filter); key != "" && !keys[key] {
		return fmt.Errorf("unknown filter key %q; use stable column keys", key)
	}
	if key := sortKey(sort); key != "" && !keys[key] {
		return fmt.Errorf("unknown sort key %q; use stable column keys", key)
	}
	return nil
}

// listPluginUpdates prints available updates for installed plugins.
func listPluginUpdates(stdout io.Writer, root string, source any, transport http.RoundTripper, publicKey string) error {
	_, indexBytes, err := readRegistryIndex(source, transport, publicKey)
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
		bundle, err := plugin.LoadBundle(filepath.Join(root, entry.Name()), version.Version)
		if err != nil {
			return err
		}
		artifact, ok := idx.Find(bundle.Manifest.Name, "")
		if !ok || compareVersion(artifact.Version, bundle.Manifest.Version) <= 0 {
			continue
		}
		fmt.Fprintf(stdout, "%s %s -> %s\n", bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version)
	}
	return nil
}

// installPluginFromHostedSource installs one plugin selected from a hosted
// source.
func installPluginFromHostedSource(stdout io.Writer, root string, source distribution.Source, name, channel string, transport http.RoundTripper, publicKey string) error {
	selectedSource, artifact, err := findRegistryArtifact(source, name, channel, transport, publicKey)
	if err != nil {
		return err
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
	fmt.Fprintf(stdout, "installed %s\n", artifact.Name)
	return nil
}

// installPluginSource installs a verified local plugin bundle or archive.
func installPluginSource(source, root string) (string, error) {
	return plugin.InstallVerifiedLocalBundle(source, root, version.Version)
}

// bundledPluginSource resolves a bundled plugin name for development installs.
func bundledPluginSource(name string) (string, error) {
	if !strings.HasSuffix(version.Version, "-dev") {
		return "", fmt.Errorf("--bundled is only available in development builds")
	}
	if !plugin.ValidName(name) {
		return "", fmt.Errorf("invalid plugin name %q", name)
	}
	source := filepath.Join(defaultPluginRoot(), name)
	if _, err := os.Stat(source); err != nil {
		return "", err
	}
	return source, nil
}

// updateOneBundledPlugin updates one installed plugin from bundled metadata.
func updateOneBundledPlugin(stdout io.Writer, root, name string) error {
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
	if compareVersion(bundled.Manifest.Version, bundle.Manifest.Version) <= 0 {
		fmt.Fprintf(stdout, "%s is up to date\n", bundle.Manifest.Name)
		return nil
	}
	if _, err := installPluginSource(source, root); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "updated %s %s -> %s\n", bundle.Manifest.Name, bundle.Manifest.Version, bundled.Manifest.Version)
	return nil
}

// updateAllBundledPlugins updates installed plugins from bundled metadata.
func updateAllBundledPlugins(stdout io.Writer, root string) error {
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
		if err := updateOneBundledPlugin(stdout, root, entry.Name()); err != nil {
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
			return distribution.Source{}, fmt.Errorf("plugin source is empty")
		}
		return value, nil
	case string:
		if value == "" {
			return distribution.Source{}, fmt.Errorf("plugin source is empty")
		}
		return distribution.Source{Name: "test", URL: value, Kind: distribution.SourceReady}, nil
	default:
		return distribution.Source{}, fmt.Errorf("unsupported plugin source %T", source)
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
