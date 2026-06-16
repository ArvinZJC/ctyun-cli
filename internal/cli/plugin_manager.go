/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	stdpath "path"
	"path/filepath"
	"strconv"
	"strings"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

func runPlugin(stdout io.Writer, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper) error {
	return runPluginWithLanguage(stdout, root, args, profile, getenv, transport, "en-US")
}

func runPluginWithLanguage(stdout io.Writer, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, language string) error {
	return runPluginWithOptions(stdout, root, args, profile, getenv, transport, globalOptions{Output: "table", Language: language})
}

func runPluginWithOptions(stdout io.Writer, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, global globalOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("plugin requires a subcommand")
	}
	if global.Output == "" {
		global.Output = "table"
	}
	trustedRegistryKey := registryPublicKey(getenv, profile)
	switch args[0] {
	case "install":
		opts, err := parsePluginInstallOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Source == "" {
			return fmt.Errorf("plugin install requires a bundle path or plugin name")
		}
		source := opts.Source
		opts.Registry = registryURL(opts.Registry, getenv, profile)
		if opts.Registry != "" && !pathExists(source) {
			artifact, err := findRegistryArtifact(opts.Registry, source, opts.Channel, transport, trustedRegistryKey)
			if err != nil {
				return err
			}
			artifactSource, cleanup, err := prepareRegistryArtifact(opts.Registry, artifact, transport)
			if err != nil {
				return err
			}
			defer cleanup()
			source = artifactSource
			if err := verifyArtifact(source, artifact); err != nil {
				return err
			}
		}
		installed, err := plugin.InstallVerifiedLocalBundle(source, root, version.Version)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "installed %s\n", filepath.Base(installed))
		return nil
	case "list":
		opts, err := parsePluginListOptions(args[1:])
		if err != nil {
			return err
		}
		opts.Registry = registryURL(opts.Registry, getenv, profile)
		if opts.Updates {
			return listPluginUpdates(stdout, root, opts.Registry, transport, trustedRegistryKey)
		}
		return listPlugins(stdout, root, global)
	case "search":
		opts, err := parsePluginSearchOptions(args[1:])
		if err != nil {
			return err
		}
		opts.Registry = registryURL(opts.Registry, getenv, profile)
		return searchPlugins(stdout, opts.Registry, opts.Channel, opts.Query, transport, trustedRegistryKey)
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
		opts.Registry = registryURL(opts.Registry, getenv, profile)
		if opts.All {
			return updateAllPlugins(stdout, root, opts.Registry, transport, trustedRegistryKey)
		}
		if opts.Name != "" {
			return updateOnePlugin(stdout, root, opts.Registry, opts.Name, transport, trustedRegistryKey)
		}
		return fmt.Errorf("plugin update/upgrade requires a plugin name or --all")
	default:
		return fmt.Errorf("unknown plugin subcommand %q", args[0])
	}
}

func registryURL(flag string, getenv func(string) string, profile coreconfig.Profile) string {
	if flag != "" {
		return flag
	}
	if value := getenv("CTYUN_REGISTRY_URL"); value != "" {
		return value
	}
	return profile.RegistryURL
}

func registryPublicKey(getenv func(string) string, profile coreconfig.Profile) string {
	if value := getenv("CTYUN_REGISTRY_PUBLIC_KEY"); value != "" {
		return value
	}
	return profile.RegistryPublicKey
}

type pluginInstallOptions struct {
	Source   string
	Registry string
	Channel  string
}

func parsePluginInstallOptions(args []string) (pluginInstallOptions, error) {
	var opts pluginInstallOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--registry":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--registry requires a value")
			}
			opts.Registry = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--channel requires a value")
			}
			opts.Channel = args[i]
		default:
			if opts.Source != "" {
				return opts, fmt.Errorf("plugin install accepts one source")
			}
			opts.Source = args[i]
		}
	}
	return opts, nil
}

type pluginUpdateOptions struct {
	All      bool
	Name     string
	Registry string
}

type pluginSearchOptions struct {
	Query    string
	Registry string
	Channel  string
}

func parsePluginSearchOptions(args []string) (pluginSearchOptions, error) {
	var opts pluginSearchOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--registry":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--registry requires a value")
			}
			opts.Registry = args[i]
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

func parsePluginUpdateOptions(args []string) (pluginUpdateOptions, error) {
	var opts pluginUpdateOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--registry":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--registry requires a value")
			}
			opts.Registry = args[i]
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
	return opts, nil
}

func findRegistryArtifact(registryRoot, name, channel string, transport http.RoundTripper, publicKey string) (registry.Artifact, error) {
	indexBytes, err := readRegistryIndex(registryRoot, transport, publicKey)
	if err != nil {
		return registry.Artifact{}, err
	}
	idx, err := registry.LoadIndex(indexBytes)
	if err != nil {
		return registry.Artifact{}, err
	}
	artifact, ok := idx.Find(name, channel)
	if !ok {
		return registry.Artifact{}, fmt.Errorf("plugin %s not found in registry", name)
	}
	return artifact, nil
}

func searchPlugins(stdout io.Writer, registryRoot, channel, query string, transport http.RoundTripper, publicKey string) error {
	if registryRoot == "" {
		return fmt.Errorf("plugin search requires --registry for now")
	}
	indexBytes, err := readRegistryIndex(registryRoot, transport, publicKey)
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

func updateAllPlugins(stdout io.Writer, root, registryRoot string, transport http.RoundTripper, publicKey string) error {
	if registryRoot == "" {
		return fmt.Errorf("plugin update --all requires --registry for now")
	}
	indexBytes, err := readRegistryIndex(registryRoot, transport, publicKey)
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
		source, cleanup, err := prepareRegistryArtifact(registryRoot, artifact, transport)
		if err != nil {
			return err
		}
		if err := verifyArtifact(source, artifact); err != nil {
			cleanup()
			return err
		}
		if _, err := plugin.InstallVerifiedLocalBundle(source, root, version.Version); err != nil {
			cleanup()
			return err
		}
		cleanup()
		fmt.Fprintf(stdout, "updated %s %s -> %s\n", bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version)
	}
	return nil
}

func updateOnePlugin(stdout io.Writer, root, registryRoot, name string, transport http.RoundTripper, publicKey string) error {
	if registryRoot == "" {
		return fmt.Errorf("plugin update %s requires --registry for now", name)
	}
	bundle, err := plugin.LoadBundle(filepath.Join(root, name), version.Version)
	if err != nil {
		return err
	}
	artifact, err := findRegistryArtifact(registryRoot, bundle.Manifest.Name, "", transport, publicKey)
	if err != nil {
		return err
	}
	if compareVersion(artifact.Version, bundle.Manifest.Version) <= 0 {
		fmt.Fprintf(stdout, "%s is up to date\n", bundle.Manifest.Name)
		return nil
	}
	source, cleanup, err := prepareRegistryArtifact(registryRoot, artifact, transport)
	if err != nil {
		return err
	}
	defer cleanup()
	if err := verifyArtifact(source, artifact); err != nil {
		return err
	}
	if _, err := plugin.InstallVerifiedLocalBundle(source, root, version.Version); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "updated %s %s -> %s\n", bundle.Manifest.Name, bundle.Manifest.Version, artifact.Version)
	return nil
}

type pluginListOptions struct {
	Updates  bool
	Registry string
}

func parsePluginListOptions(args []string) (pluginListOptions, error) {
	var opts pluginListOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--updates":
			opts.Updates = true
		case "--registry":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--registry requires a value")
			}
			opts.Registry = args[i]
		default:
			return opts, fmt.Errorf("unknown plugin list option %q", args[i])
		}
	}
	return opts, nil
}

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

func listPluginUpdates(stdout io.Writer, root, registryRoot string, transport http.RoundTripper, publicKey string) error {
	if registryRoot == "" {
		return fmt.Errorf("plugin list --updates requires --registry for now")
	}
	indexBytes, err := readRegistryIndex(registryRoot, transport, publicKey)
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

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readRegistryIndex(registryRoot string, transport http.RoundTripper, publicKey string) ([]byte, error) {
	if !isHTTPURL(registryRoot) {
		return os.ReadFile(filepath.Join(registryRoot, "index.json"))
	}
	// HTTP registries cross a trust boundary, so require an adjacent detached
	// signature before accepting the index.
	index, err := httpGetBytes(joinRegistryURL(registryRoot, "index.json"), transport)
	if err != nil {
		return nil, err
	}
	signature, err := httpGetBytes(joinRegistryURL(registryRoot, "index.sig"), transport)
	if err != nil {
		return nil, fmt.Errorf("read registry signature: %w", err)
	}
	if err := registry.VerifyIndexSignature(index, signature, publicKey); err != nil {
		return nil, err
	}
	return index, nil
}

func prepareRegistryArtifact(registryRoot string, artifact registry.Artifact, transport http.RoundTripper) (string, func(), error) {
	artifactURL := artifact.URL
	if isHTTPURL(artifactURL) {
		if artifact.SHA256 == "" {
			return "", func() {}, fmt.Errorf("HTTP registry artifact %s requires sha256", artifact.Name)
		}
		return downloadRegistryArtifact(artifactURL, transport)
	}
	if !isHTTPURL(registryRoot) {
		return filepath.Join(registryRoot, artifact.URL), func() {}, nil
	}
	if artifact.SHA256 == "" {
		return "", func() {}, fmt.Errorf("HTTP registry artifact %s requires sha256", artifact.Name)
	}
	// Relative artifact URLs inherit the registry origin so mirror indexes can
	// stay portable between hosts.
	artifactURL = artifact.URL
	if !isHTTPURL(artifactURL) {
		artifactURL = joinRegistryURL(registryRoot, artifactURL)
	}
	return downloadRegistryArtifact(artifactURL, transport)
}

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

func httpGetBytes(rawURL string, transport http.RoundTripper) ([]byte, error) {
	if transport == nil {
		transport = http.DefaultTransport
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s returned %s", rawURL, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func joinRegistryURL(root, name string) string {
	parsed, err := url.Parse(root)
	if err != nil {
		return root + "/" + name
	}
	parsed.Path = stdpath.Join(parsed.Path, name)
	return parsed.String()
}

func isHTTPURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

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
