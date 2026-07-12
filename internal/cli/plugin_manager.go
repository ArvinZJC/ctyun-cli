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
	"strconv"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// runPluginWithOptions dispatches plugin install, list, search, remove,
// reinstall, lint, and update commands. Lint is intentionally hidden from
// public help because it validates local bundle directories for contributor
// workflows.
func runPluginWithOptions(stdout, stderr io.Writer, stdin io.Reader, root, group string, args []string, _ coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, global globalOptions) error {
	if len(args) == 0 {
		_, err := printPluginHelp(stdout, []string{"plugin"}, global.Language)
		return err
	}
	if global.Output == "" {
		global.Output = "table"
	}
	publicKey := releasePublicKey(getenv)
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
			return installBundledPluginsWithProgress(stdout, stderr, root, opts.Names, opts.All, global.Language)
		}
		source, err := resolvePluginSource(opts.Source, getenv)
		if err != nil {
			return err
		}
		return installPluginsFromHostedSourceWithProgress(stdout, stderr, root, source, opts.Names, opts.All, opts.Channel, transport, publicKey, global.Language)
	case "reinstall":
		opts, err := parsePluginReinstallOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Bundled {
			return reinstallBundledPluginsWithProgress(stdout, stderr, root, opts.Names, opts.All, global.Language)
		}
		source, err := resolvePluginSource(opts.Source, getenv)
		if err != nil {
			return err
		}
		return reinstallPluginsFromHostedSourceWithProgress(stdout, stderr, root, source, opts.Names, opts.All, opts.Channel, transport, publicKey, global.Language)
	case "list":
		opts, err := parsePluginListOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Available {
			if opts.Bundled {
				return listBundledAvailablePlugins(stdout, root, opts.Channel, global)
			}
			source, err := resolvePluginSource(opts.Source, getenv)
			if err != nil {
				return err
			}
			return listAvailablePlugins(stdout, root, source, opts.Channel, transport, publicKey, global)
		}
		if opts.Updates {
			if opts.Bundled {
				return listBundledPluginUpdates(stdout, root, opts.Channel, global.Language)
			}
			source, err := resolvePluginSource(opts.Source, getenv)
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
		source, err := resolvePluginSource(opts.Source, getenv)
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
		if err := validatePositionalArguments(args[1:], []string{"path"}, 1, 1); err != nil {
			return err
		}
		bundle, err := plugin.LoadBundle(args[1], version.Version)
		if err != nil {
			return err
		}
		return writeLine(stdout, pluginValidMessage(global.Language, bundle.Manifest.Name, bundle.Manifest.Version))
	case "update", "upgrade":
		opts, err := parsePluginUpdateOptions(args[1:])
		if err != nil {
			return err
		}
		if opts.Bundled {
			if opts.All {
				return updateBundledPluginsWithProgress(stdout, stderr, root, nil, true, global.Language)
			}
			if opts.Name != "" {
				return updateBundledPluginsWithProgress(stdout, stderr, root, []string{opts.Name}, false, global.Language)
			}
			return diagnostic.New("error.plugin_bundled_update_target")
		}
		source, err := resolvePluginSource(opts.Source, getenv)
		if err != nil {
			return err
		}
		if opts.All {
			return updatePluginsFromHostedSourceWithProgress(stdout, stderr, root, source, nil, true, opts.Channel, transport, publicKey, global.Language)
		}
		if opts.Name != "" {
			return updatePluginsFromHostedSourceWithProgress(stdout, stderr, root, source, []string{opts.Name}, false, opts.Channel, transport, publicKey, global.Language)
		}
		return diagnostic.New("error.plugin_update_target")
	default:
		return commandBoundaryError(append([]string{group}, args...))
	}
}

// resolvePluginSource applies hosted source precedence for plugin metadata.
func resolvePluginSource(flag string, getenv func(string) string) (distribution.Source, error) {
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

// pluginNameSourceOptions captures common plugin target and source flags.
type pluginNameSourceOptions struct {
	Names   []string
	All     bool
	Source  string
	Channel string
	Bundled bool
}

// pluginInstallOptions captures plugin install flags and source.
type pluginInstallOptions = pluginNameSourceOptions

// parsePluginNameSourceOptions parses shared plugin target and source flags.
func parsePluginNameSourceOptions(args []string) (pluginNameSourceOptions, error) {
	var opts pluginNameSourceOptions
	options := []commandOption{
		{Name: "all"},
		{Name: "source", TakesValue: true},
		{Name: "channel", TakesValue: true},
	}
	if version.IsDevelopmentBuild() {
		options = append(options, commandOption{Name: "bundled"})
	}
	parsed, err := parseCommandTokens(args, options)
	if err != nil {
		return opts, err
	}
	opts.Names = parsed.Positionals
	opts.All = parsed.Present["all"]
	opts.Source = parsed.Options["source"]
	opts.Channel = parsed.Options["channel"]
	opts.Bundled = parsed.Present["bundled"]
	if err := validatePluginSourceOption(opts.Source); err != nil {
		return opts, err
	}
	if err := validatePluginChannelOption(opts.Channel, false); err != nil {
		return opts, err
	}
	return opts, nil
}

// parsePluginInstallOptions parses plugin install arguments.
func parsePluginInstallOptions(args []string) (pluginInstallOptions, error) {
	opts, err := parsePluginNameSourceOptions(args)
	if err != nil {
		return opts, err
	}
	if opts.All && len(opts.Names) > 0 {
		return opts, diagnostic.New("error.plugin_install_all_or_names")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_install_source_choice")
	}
	return opts, nil
}

// pluginReinstallOptions captures plugin reinstall flags and source.
type pluginReinstallOptions = pluginNameSourceOptions

// parsePluginReinstallOptions parses plugin reinstall arguments.
func parsePluginReinstallOptions(args []string) (pluginReinstallOptions, error) {
	opts, err := parsePluginNameSourceOptions(args)
	if err != nil {
		return opts, err
	}
	if opts.All && len(opts.Names) > 0 {
		return opts, diagnostic.New("error.plugin_reinstall_all_or_names")
	}
	if !opts.All && len(opts.Names) == 0 {
		return opts, diagnostic.New("error.plugin_reinstall_target")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_reinstall_source_choice")
	}
	for _, name := range opts.Names {
		if !plugin.ValidName(name) {
			return opts, diagnostic.New("error.plugin_name", name)
		}
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

// removeAll provides a test seam for independent plugin removal failures.
var removeAll = os.RemoveAll

// parsePluginSearchOptions parses plugin search arguments.
func parsePluginSearchOptions(args []string) (pluginSearchOptions, error) {
	var opts pluginSearchOptions
	options := []commandOption{
		{Name: "source", TakesValue: true},
		{Name: "channel", TakesValue: true},
	}
	if version.IsDevelopmentBuild() {
		options = append(options, commandOption{Name: "bundled"})
	}
	parsed, err := parseCommandTokens(args, options)
	if err != nil {
		return opts, err
	}
	if err := requirePositional(parsed.Positionals, 1, "query"); err != nil {
		return opts, err
	}
	if err := rejectUnexpectedPositionals(parsed.Positionals, 1); err != nil {
		return opts, err
	}
	opts.Query = parsed.Positionals[0]
	opts.Source = parsed.Options["source"]
	opts.Channel = parsed.Options["channel"]
	opts.Bundled = parsed.Present["bundled"]
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_search_source_choice")
	}
	if err := validatePluginSourceOption(opts.Source); err != nil {
		return opts, err
	}
	if err := validatePluginChannelOption(opts.Channel, true); err != nil {
		return opts, err
	}
	return opts, nil
}

// parsePluginRemoveOptions parses plugin remove arguments.
func parsePluginRemoveOptions(args []string) (pluginRemoveOptions, error) {
	var opts pluginRemoveOptions
	parsed, err := parseCommandTokens(args, []commandOption{{Name: "all"}})
	if err != nil {
		return opts, err
	}
	opts.Names = parsed.Positionals
	opts.All = parsed.Present["all"]
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
	tasks := make([]operationTask, 0, len(names))
	for _, name := range names {
		name := name
		tasks = append(tasks, operationTask{
			Target: name,
			Label:  operationProgressLabel(global.Language, "remove", name),
			Run: func() operationResult {
				if !plugin.ValidName(name) {
					return operationResult{Target: name, Err: diagnostic.New("error.plugin_name", name)}
				}
				if err := removeAll(filepath.Join(root, name)); err != nil {
					return operationResult{Target: name, Err: err}
				}
				return operationResult{Target: name, Outcome: operationChanged}
			},
		})
	}
	return executeAndReportOperationTasks(stdout, stderr, tasks, global.Language, pluginRemoveSummary)
}

// parsePluginUpdateOptions parses plugin update or upgrade arguments.
func parsePluginUpdateOptions(args []string) (pluginUpdateOptions, error) {
	var opts pluginUpdateOptions
	options := []commandOption{
		{Name: "all"},
		{Name: "source", TakesValue: true},
		{Name: "channel", TakesValue: true},
	}
	if version.IsDevelopmentBuild() {
		options = append(options, commandOption{Name: "bundled"})
	}
	parsed, err := parseCommandTokens(args, options)
	if err != nil {
		return opts, err
	}
	if err := rejectUnexpectedPositionals(parsed.Positionals, 1); err != nil {
		return opts, err
	}
	if len(parsed.Positionals) == 1 {
		opts.Name = parsed.Positionals[0]
	}
	opts.All = parsed.Present["all"]
	opts.Source = parsed.Options["source"]
	opts.Channel = parsed.Options["channel"]
	opts.Bundled = parsed.Present["bundled"]
	if opts.All && opts.Name != "" {
		return opts, diagnostic.New("error.plugin_update_all_or_one")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_update_source_choice")
	}
	if err := validatePluginSourceOption(opts.Source); err != nil {
		return opts, err
	}
	if err := validatePluginChannelOption(opts.Channel, false); err != nil {
		return opts, err
	}
	return opts, nil
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
	options := []commandOption{
		{Name: "updates"},
		{Name: "available"},
		{Name: "source", TakesValue: true},
		{Name: "channel", TakesValue: true},
	}
	if version.IsDevelopmentBuild() {
		options = append(options, commandOption{Name: "bundled"})
	}
	parsed, err := parseCommandTokens(args, options)
	if err != nil {
		return opts, err
	}
	if err := rejectUnexpectedPositionals(parsed.Positionals, 0); err != nil {
		return opts, err
	}
	opts.Updates = parsed.Present["updates"]
	opts.Available = parsed.Present["available"]
	opts.Source = parsed.Options["source"]
	opts.Channel = parsed.Options["channel"]
	opts.Bundled = parsed.Present["bundled"]
	if opts.Available && opts.Updates {
		return opts, diagnostic.New("error.plugin_list_available_updates")
	}
	if opts.Bundled && opts.Source != "" {
		return opts, diagnostic.New("error.plugin_list_source_choice")
	}
	if err := validatePluginSourceOption(opts.Source); err != nil {
		return opts, err
	}
	if err := validatePluginChannelOption(opts.Channel, true); err != nil {
		return opts, err
	}
	if opts.Channel == "all" && !opts.Available {
		return opts, diagnostic.New("error.plugin_list_all_channel_available")
	}
	return opts, nil
}

// validatePluginSourceOption checks the finite source values accepted by plugin
// manager commands before hosted source resolution.
func validatePluginSourceOption(source string) error {
	if source == "" || source == "auto" || source == "github" || source == "gitee" {
		return nil
	}
	return diagnostic.New("error.unsupported_source", "plugin", source)
}

// validatePluginChannelOption checks finite channel values accepted by plugin
// manager commands.
func validatePluginChannelOption(channel string, allowAll bool) error {
	if channel == "" || channel == "stable" || channel == "beta" || channel == "alpha" {
		return nil
	}
	if allowAll && channel == "all" {
		return nil
	}
	if allowAll {
		return diagnostic.New("error.unsupported_plugin_discovery_channel", channel)
	}
	return diagnostic.New("error.unsupported_channel", channel)
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
	return renderPluginRows(stdout, rows, pluginListColumns(opts.Language), opts)
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

// installBundledPlugins installs development bundled plugins by name or all
// bundled plugin directories.
func installBundledPlugins(stdout io.Writer, root string, names []string, all bool, language string) error {
	return installBundledPluginsWithProgress(stdout, io.Discard, root, names, all, language)
}

// installBundledPluginsWithProgress installs only absent bundled plugins and
// reports one structured batch result.
func installBundledPluginsWithProgress(stdout, stderr io.Writer, root string, names []string, all bool, language string) error {
	var targets []string
	if all {
		if !version.IsDevelopmentBuild() {
			return diagnostic.New("error.bundled_dev_only")
		}
		for _, dir := range pluginDirs(defaultPluginRoot()) {
			bundle, err := plugin.LoadBundle(dir, version.Version)
			if err != nil {
				return err
			}
			targets = append(targets, bundle.Manifest.Name)
		}
	} else {
		targets = names
	}

	tasks := make([]operationTask, 0, len(targets))
	for _, name := range targets {
		name := name
		tasks = append(tasks, operationTask{
			Target: name,
			Label:  operationProgressLabel(language, "install", name),
			Run: func() operationResult {
				_, installed, err := loadInstalledPlugin(root, name)
				if err != nil {
					return operationResult{Target: name, Err: err}
				}
				if installed {
					return operationResult{Target: name, Outcome: operationSkipped}
				}
				source, err := bundledPluginSource(name)
				if err != nil {
					return operationResult{Target: name, Err: err}
				}
				if _, err := plugin.InstallVerifiedLocalBundle(source, root, version.Version); err != nil {
					return operationResult{Target: name, Err: err}
				}
				return operationResult{Target: name, Outcome: operationChanged}
			},
		})
	}
	return executeAndReportOperationTasks(stdout, stderr, tasks, language, pluginInstallSummary)
}

// loadInstalledPlugin loads one valid installed plugin and distinguishes an
// absent target from corrupt installed state.
func loadInstalledPlugin(root, name string) (plugin.Bundle, bool, error) {
	if !plugin.ValidName(name) {
		return plugin.Bundle{}, false, diagnostic.New("error.plugin_name", name)
	}
	path := filepath.Join(root, name)
	info, err := osStat(path)
	if os.IsNotExist(err) {
		return plugin.Bundle{}, false, nil
	}
	if err != nil {
		return plugin.Bundle{}, false, err
	}
	if !info.IsDir() {
		return plugin.Bundle{}, false, diagnostic.New("error.plugin_root_not_directory", path)
	}
	bundle, err := plugin.LoadBundle(path, version.Version)
	if err != nil {
		return plugin.Bundle{}, false, err
	}
	return bundle, true, nil
}

// reinstallBundledPlugins reinstalls selected installed plugins from bundled
// development metadata without comparing versions.
func reinstallBundledPlugins(stdout io.Writer, root string, names []string, all bool, language string) error {
	return reinstallBundledPluginsWithProgress(stdout, io.Discard, root, names, all, language)
}

// reinstallBundledPluginsWithProgress replaces selected installed plugins
// from bundled development metadata and reports one batch result.
func reinstallBundledPluginsWithProgress(stdout, stderr io.Writer, root string, names []string, all bool, language string) error {
	targets := installedPluginTargets(root, names, all)
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
				source, err := bundledPluginSource(target.Name)
				if err != nil {
					return operationResult{Target: target.Name, Err: err}
				}
				if _, err := plugin.InstallVerifiedLocalBundle(source, root, version.Version); err != nil {
					return operationResult{Target: target.Name, Err: err}
				}
				return operationResult{Target: target.Name, Outcome: operationChanged}
			},
		})
	}
	return executeAndReportOperationTasks(stdout, stderr, tasks, language, pluginReinstallSummary)
}

// installedPluginTarget records one resolved reinstall or update target.
type installedPluginTarget struct {
	Name   string
	Bundle plugin.Bundle
	Err    error
}

// installedPluginTargets resolves explicit names or every installed directory
// while preserving explicit order and stable directory order.
func installedPluginTargets(root string, names []string, all bool) []installedPluginTarget {
	if all {
		names = make([]string, 0)
		for _, dir := range pluginDirs(root) {
			names = append(names, filepath.Base(dir))
		}
	}
	targets := make([]installedPluginTarget, 0, len(names))
	for _, name := range names {
		bundle, installed, err := loadInstalledPlugin(root, name)
		if err != nil {
			targets = append(targets, installedPluginTarget{Name: name, Err: err})
			continue
		}
		if !installed {
			targets = append(targets, installedPluginTarget{Name: name, Err: diagnostic.New("error.plugin_not_installed", name)})
			continue
		}
		targets = append(targets, installedPluginTarget{Name: bundle.Manifest.Name, Bundle: bundle})
	}
	return targets
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
	return updateBundledPluginsWithProgress(stdout, io.Discard, root, []string{name}, false, language)
}

// updateAllBundledPlugins updates installed plugins from bundled metadata.
func updateAllBundledPlugins(stdout io.Writer, root string, language string) error {
	return updateBundledPluginsWithProgress(stdout, io.Discard, root, nil, true, language)
}

// updateBundledPluginsWithProgress updates installed plugins only when bundled
// metadata has a strictly newer semantic version.
func updateBundledPluginsWithProgress(stdout, stderr io.Writer, root string, names []string, all bool, language string) error {
	if all {
		if err := validatePluginRootForList(root); err != nil {
			return err
		}
	}
	targets := installedPluginTargets(root, names, all)
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
				source, err := bundledPluginSource(target.Name)
				if err != nil {
					return operationResult{Target: target.Name, Err: err}
				}
				bundled, err := plugin.LoadBundle(source, version.Version)
				if err != nil {
					return operationResult{Target: target.Name, Err: err}
				}
				if version.CompareSemanticVersions(bundled.Manifest.Version, target.Bundle.Manifest.Version) <= 0 {
					return operationResult{Target: target.Name, Outcome: operationUnchanged, OldVersion: target.Bundle.Manifest.Version, NewVersion: bundled.Manifest.Version}
				}
				if _, err := plugin.InstallVerifiedLocalBundle(source, root, version.Version); err != nil {
					return operationResult{Target: target.Name, Err: err}
				}
				return operationResult{Target: target.Name, Outcome: operationChanged, OldVersion: target.Bundle.Manifest.Version, NewVersion: bundled.Manifest.Version}
			},
		})
	}
	return executeAndReportOperationTasks(stdout, stderr, tasks, language, pluginUpdateSummary)
}
