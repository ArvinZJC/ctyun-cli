package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	stdpath "path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
	"github.com/ArvinZJC/ctyun-cli/internal/waiter"
)

type Config struct {
	Args          []string
	Stdout        io.Writer
	Stderr        io.Writer
	PluginRoot    string
	Env           func(string) string
	Config        []byte
	ConfigPath    string
	HTTPTransport http.RoundTripper
}

type globalOptions struct {
	Output   string
	Columns  []string
	NoHeader bool
	Language string
	Filter   string
	Sort     string
	Offline  bool
	Yes      bool
	Waiter   string
	Table    string
	Config   string
	Profile  string
	Debug    bool
	Timeout  int
}

func Run(cfg Config) error {
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	getenv := cfg.Env
	if getenv == nil {
		getenv = os.Getenv
	}

	opts, args, err := parseGlobalOptions(cfg.Args)
	if err != nil {
		return err
	}
	if opts.Output == "" {
		opts.Output = "table"
	}
	configBytes, err := loadConfigBytes(cfg.Config, configPath(opts.Config, cfg.ConfigPath, getenv))
	if err != nil {
		return err
	}
	profile, err := activeProfile(configBytes, opts.Profile)
	if err != nil {
		return err
	}
	if opts.Language == "" {
		opts.Language = i18n.ResolveLanguage(i18n.LanguageOptions{
			Env:      getenv("CTYUN_LANGUAGE"),
			Profile:  profile.Language,
			OSLocale: getenv("LANG"),
		})
	} else {
		opts.Language = i18n.ResolveLanguage(i18n.LanguageOptions{Flag: opts.Language})
	}

	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: ctyun <command>")
		return fmt.Errorf("missing command")
	}

	switch args[0] {
	case "version":
		_, err = fmt.Fprintf(stdout, "%s %s\n", version.Name, version.Version)
		return err
	case "help":
		return runHelp(stdout, args[1:], pluginRoot(cfg.PluginRoot), opts.Language)
	case "completion":
		return runCompletion(stdout, args[1:], pluginRoot(cfg.PluginRoot))
	case "doctor":
		return runDoctor(stdout, args[1:])
	case "upgrade":
		fmt.Fprintln(stdout, "core self-upgrade is deferred; install updates through your package manager for now")
		return nil
	case "plugin":
		return runPlugin(stdout, pluginRoot(cfg.PluginRoot), args[1:], profile, getenv, cfg.HTTPTransport)
	default:
		return runPluginCommand(stdout, stderr, opts, args, pluginRoot(cfg.PluginRoot), profile, getenv, cfg.HTTPTransport)
	}
}

func Execute(cfg Config) int {
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	getenv := cfg.Env
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg.Stdout = stdout
	cfg.Stderr = stderr
	cfg.Env = getenv

	if err := Run(cfg); err != nil {
		language := errorLanguage(cfg, getenv)
		message := formatError(err, language)
		message = client.RedactHTTPDetails(message, coreconfig.Credentials{
			AccessKey: getenv("CTYUN_AK"),
			SecretKey: getenv("CTYUN_SK"),
		}, "")
		fmt.Fprintln(stderr, message)
		return 1
	}
	return 0
}

func errorLanguage(cfg Config, getenv func(string) string) string {
	opts, _, err := parseGlobalOptions(cfg.Args)
	if err != nil {
		return i18n.ResolveLanguage(i18n.LanguageOptions{Env: getenv("CTYUN_LANGUAGE"), OSLocale: getenv("LANG")})
	}
	if opts.Language != "" {
		return i18n.ResolveLanguage(i18n.LanguageOptions{Flag: opts.Language})
	}
	configBytes, err := loadConfigBytes(cfg.Config, configPath(opts.Config, cfg.ConfigPath, getenv))
	if err != nil {
		return i18n.ResolveLanguage(i18n.LanguageOptions{Env: getenv("CTYUN_LANGUAGE"), OSLocale: getenv("LANG")})
	}
	profile, _ := activeProfile(configBytes, opts.Profile)
	return i18n.ResolveLanguage(i18n.LanguageOptions{
		Env:      getenv("CTYUN_LANGUAGE"),
		Profile:  profile.Language,
		OSLocale: getenv("LANG"),
	})
}

func formatError(err error, language string) string {
	prefix := "Error"
	if language == "zh-CN" {
		return fmt.Sprintf("错误：%s", localizedErrorText(err.Error(), language))
	}
	return fmt.Sprintf("%s: %s", prefix, err.Error())
}

func localizedErrorText(message, language string) string {
	if language != "zh-CN" {
		return message
	}
	if match := regexp.MustCompile(`^plugin ([^ ]+) requires ctyun (.+), current version is (.+)$`).FindStringSubmatch(message); match != nil {
		return fmt.Sprintf("插件 %s 需要 ctyun %s，当前版本是 %s", match[1], match[2], match[3])
	}
	if match := regexp.MustCompile(`^unknown command "(.+)"$`).FindStringSubmatch(message); match != nil {
		return fmt.Sprintf("未知命令 %q", match[1])
	}
	if message == "missing command" {
		return "缺少命令"
	}
	return message
}

func runCompletion(stdout io.Writer, args []string, installedRoot string) error {
	if len(args) != 1 {
		return fmt.Errorf("completion requires one shell: bash, zsh, or fish")
	}
	words := completionWords(installedRoot)
	switch args[0] {
	case "zsh":
		fmt.Fprintln(stdout, "#compdef ctyun")
		fmt.Fprintf(stdout, "_ctyun() { _arguments '*::ctyun command:((%s))' }\n", strings.Join(words, " "))
		return nil
	case "bash":
		fmt.Fprintf(stdout, "complete -W '%s' ctyun\n", strings.Join(words, " "))
		return nil
	case "fish":
		fmt.Fprintf(stdout, "complete -c ctyun -f -a '%s'\n", strings.Join(words, " "))
		return nil
	default:
		return fmt.Errorf("unsupported shell %q", args[0])
	}
}

func completionWords(installedRoot string) []string {
	seen := map[string]bool{
		"version": true, "upgrade": true, "doctor": true, "plugin": true, "completion": true, "help": true,
		"install": true, "list": true, "lint": true, "remove": true, "search": true, "update": true,
		"network":  true,
		"--output": true, "--cols": true, "--no-header": true, "--filter": true, "--sort": true,
		"--language": true, "--lang": true, "--offline": true, "--fixture": true, "--yes": true, "--wait": true,
		"--table":   true,
		"--timeout": true,
		"--config":  true, "--profile": true, "--debug": true, "--registry": true, "--channel": true,
	}
	for _, bundle := range mustLoadBundlesForCompletion(installedRoot) {
		for _, command := range bundle.Commands.Commands {
			for _, part := range command.Path {
				if strings.HasPrefix(part, "{") {
					continue
				}
				seen[part] = true
			}
			for _, alias := range command.Aliases {
				for _, part := range alias {
					if strings.HasPrefix(part, "{") {
						continue
					}
					seen[part] = true
				}
			}
			for _, parameter := range command.Parameters {
				if parameter.Flag != "" {
					seen["--"+parameter.Flag] = true
				}
			}
		}
	}
	words := make([]string, 0, len(seen))
	for word := range seen {
		words = append(words, word)
	}
	sortStrings(words)
	return words
}

func mustLoadBundlesForCompletion(installedRoot string) []plugin.Bundle {
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return nil
	}
	return bundles
}

func runHelp(stdout io.Writer, args []string, installedRoot, language string) error {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "usage: ctyun <command>")
		fmt.Fprintln(stdout, "core commands: completion, doctor, help, plugin, upgrade, version")
		return nil
	}
	bundle, command, _, _, ok, err := findPluginCommand(args, installedRoot, language)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unknown command %q", strings.Join(args, " "))
	}
	fmt.Fprintf(stdout, "%s\n", command.ID)
	if productName := localizedPluginText(bundle, language, "name", ""); productName != "" {
		fmt.Fprintf(stdout, "product: %s\n", productName)
	}
	if description := localizedPluginText(bundle, language, "command."+command.ID+".description", ""); description != "" {
		fmt.Fprintf(stdout, "description: %s\n", description)
	}
	fmt.Fprintf(stdout, "usage: ctyun %s\n", strings.Join(command.Path, " "))
	if command.DocsURL != "" {
		fmt.Fprintf(stdout, "docs: %s\n", command.DocsURL)
	}
	if len(command.Examples) > 0 {
		fmt.Fprintln(stdout, "examples:")
		for _, example := range command.Examples {
			fmt.Fprintf(stdout, "  %s\n", example)
		}
	}
	if len(command.Parameters) > 0 {
		fmt.Fprintln(stdout, "options:")
		for _, parameter := range command.Parameters {
			required := ""
			if parameter.Required {
				required = " (required)"
			}
			description := localizedPluginText(bundle, language, "parameter."+command.ID+"."+parameter.Name+".description", parameter.Description)
			if description != "" {
				description = ": " + description
			}
			validation := parameterValidationHint(parameter)
			fmt.Fprintf(stdout, "  --%s%s%s%s\n", parameter.Flag, required, description, validation)
		}
	}
	if table, ok := bundle.Tables.Tables[command.Table]; ok && len(table.Columns) > 0 {
		keys := make([]string, 0, len(table.Columns))
		for _, column := range table.Columns {
			keys = append(keys, column.Key)
		}
		fmt.Fprintf(stdout, "columns: %s\n", strings.Join(keys, ","))
	}
	return nil
}

func localizedPluginText(bundle plugin.Bundle, language, key, fallback string) string {
	if catalog, ok := bundle.I18N[language]; ok {
		if value := catalog[key]; value != "" {
			return value
		}
	}
	if catalog, ok := bundle.I18N["zh-CN"]; ok && language == "" {
		if value := catalog[key]; value != "" {
			return value
		}
	}
	return fallback
}

func parameterValidationHint(parameter plugin.Parameter) string {
	parts := make([]string, 0, 2)
	if len(parameter.AllowedValues) > 0 {
		parts = append(parts, "one of "+strings.Join(parameter.AllowedValues, ","))
	}
	if parameter.Pattern != "" {
		parts = append(parts, "matches "+parameter.Pattern)
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, "; ") + "]"
}

func runDoctor(stdout io.Writer, args []string) error {
	if len(args) != 1 || args[0] != "network" {
		return fmt.Errorf("doctor supports: network")
	}
	fmt.Fprintln(stdout, "registry: configurable with --registry, CTYUN_REGISTRY_URL, or profile registry.url")
	fmt.Fprintln(stdout, "mirror: supported through registry.url-compatible indexes; registry_url remains a flat alias")
	fmt.Fprintln(stdout, "live API: retrieval commands use CTYUN_AK and CTYUN_SK from the process environment only")
	return nil
}

func parseGlobalOptions(args []string) (globalOptions, []string, error) {
	var opts globalOptions
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--output":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--output requires a value")
			}
			opts.Output = args[i]
		case "--cols":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--cols requires a value")
			}
			opts.Columns = splitCSV(args[i])
		case "--no-header":
			opts.NoHeader = true
		case "--offline", "--fixture":
			opts.Offline = true
		case "--live":
			// Deprecated: live requests are the default. Keep this as a no-op
			// so early preview scripts do not fail abruptly.
		case "--debug":
			opts.Debug = true
		case "--yes":
			opts.Yes = true
		case "--wait":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--wait requires a value")
			}
			opts.Waiter = args[i]
		case "--table":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--table requires a value")
			}
			opts.Table = args[i]
		case "--timeout":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--timeout requires a value")
			}
			timeout, err := strconv.Atoi(args[i])
			if err != nil || timeout <= 0 {
				return opts, nil, fmt.Errorf("--timeout requires a positive integer number of seconds")
			}
			opts.Timeout = timeout
		case "--config":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--config requires a value")
			}
			opts.Config = args[i]
		case "--profile":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--profile requires a value")
			}
			opts.Profile = args[i]
		case "--filter":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--filter requires a value")
			}
			opts.Filter = args[i]
		case "--sort":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--sort requires a value")
			}
			opts.Sort = args[i]
		case "--language", "--lang":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Language = args[i]
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

func loadConfigBytes(injected []byte, path string) ([]byte, error) {
	if len(injected) > 0 {
		return injected, nil
	}
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func configPath(flag, configured string, getenv func(string) string) string {
	if flag != "" {
		return flag
	}
	if configured != "" {
		return configured
	}
	if value := getenv("CTYUN_CONFIG"); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".ctyun", "config.json")
	}
	return ""
}

func activeProfile(raw []byte, profileName string) (coreconfig.Profile, error) {
	if len(raw) == 0 {
		return coreconfig.Profile{}, nil
	}
	cfg, err := coreconfig.Load(raw)
	if err != nil {
		return coreconfig.Profile{}, err
	}
	if profileName != "" {
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			return coreconfig.Profile{}, fmt.Errorf("profile %q not found", profileName)
		}
		return profile, nil
	}
	profile, ok := cfg.ActiveProfile()
	if !ok && len(cfg.Profiles) > 0 {
		return coreconfig.Profile{}, fmt.Errorf("config with multiple profiles requires active_profile or --profile")
	}
	return profile, nil
}

func runPlugin(stdout io.Writer, root string, args []string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper) error {
	if len(args) == 0 {
		return fmt.Errorf("plugin requires a subcommand")
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
		entries, err := os.ReadDir(root)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				fmt.Fprintln(stdout, entry.Name())
			}
		}
		return nil
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
	case "update":
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
		return fmt.Errorf("plugin update requires a plugin name or --all")
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
	tmp, err := os.CreateTemp("", "ctyun-plugin-*.tar.gz")
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

func runPluginCommand(stdout, stderr io.Writer, opts globalOptions, args []string, installedRoot string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper) error {
	bundle, command, commandArgs, parameterValues, ok, err := findPluginCommand(args, installedRoot, opts.Language)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unknown command %q", strings.Join(args, " "))
	}
	if command.Dangerous.Confirm != "" && !opts.Yes {
		message := command.Dangerous.Message
		if message == "" {
			message = command.ID
		}
		return localizedConfirmationRequired(message, opts.Language)
	}

	table := bundle.Tables.Tables[command.Table]
	loadResponse := func() (map[string]any, error) {
		return loadCommandResponse(bundle, command, commandArgs, parameterValues, opts, profile, getenv, transport, debugWriter(opts, stderr))
	}
	payload, err := loadResponse()
	if err != nil {
		return err
	}

	switch opts.Output {
	case "json":
		rendered, err := output.RenderJSON(payload)
		if err != nil {
			return err
		}
		if _, err = io.WriteString(stdout, rendered); err != nil {
			return err
		}
		return renderWaiter(stderr, bundle, opts.Waiter, payload, loadResponse)
	case "table":
		rows, err := rowsFromPayload(payload, table)
		if err != nil {
			return err
		}
		rows = filterRowsByParameters(rows, table, command.Parameters, parameterValues)
		if err := validateFilterSortKeys(table, opts.Filter, opts.Sort); err != nil {
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
		columns := tableColumns(table, opts.Language)
		rendered, err := output.RenderTable(rows, columns, output.TableOptions{
			Columns:  opts.Columns,
			NoHeader: opts.NoHeader,
			Style:    opts.Table,
		})
		if err != nil {
			return err
		}
		if _, err = io.WriteString(stdout, rendered); err != nil {
			return err
		}
		return renderWaiter(stdout, bundle, opts.Waiter, payload, loadResponse)
	default:
		return fmt.Errorf("unsupported output %q", opts.Output)
	}
}

func filterRowsByParameters(rows []map[string]string, table plugin.Table, parameters []plugin.Parameter, values map[string]string) []map[string]string {
	if len(rows) == 0 || len(parameters) == 0 || len(values) == 0 {
		return rows
	}
	columnKeyByPath := make(map[string]string, len(table.Columns))
	for _, column := range table.Columns {
		columnKeyByPath[column.Path] = column.Key
	}
	targetByName := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		targetByName[parameter.Name] = columnKeyByPath[parameter.Target]
	}
	filtered := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		matches := true
		for name, value := range values {
			if value == "" {
				continue
			}
			target := targetByName[name]
			if target == "" {
				continue
			}
			if row[target] != value {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func validateFilterSortKeys(table plugin.Table, filter, sort string) error {
	keys := make(map[string]bool, len(table.Columns))
	for _, column := range table.Columns {
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

func filterKey(expression string) string {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return ""
	}
	parts := strings.SplitN(expression, "!=", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(expression, "=", 2)
	}
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func sortKey(expression string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(expression), "-"))
}

func renderWaiter(stdout io.Writer, bundle plugin.Bundle, waiterID string, payload map[string]any, loadResponse func() (map[string]any, error)) error {
	if waiterID == "" {
		return nil
	}
	spec, ok := bundle.Waiters.Waiters[waiterID]
	if !ok {
		return fmt.Errorf("unknown waiter %q", waiterID)
	}
	attempts := spec.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}
	var state waiter.State
	for attempt := 1; attempt <= attempts; attempt++ {
		var err error
		state, err = waiter.Evaluate(waiter.Spec{
			Path:    spec.Path,
			Success: spec.Success,
			Failure: spec.Failure,
		}, payload)
		if err != nil {
			return err
		}
		if state != waiter.Pending {
			break
		}
		if attempt == attempts {
			state = waiter.Timeout
			break
		}
		if spec.IntervalSeconds > 0 {
			time.Sleep(time.Duration(spec.IntervalSeconds) * time.Second)
		}
		payload, err = loadResponse()
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "waiter %s: %s\n", waiterID, state)
	return nil
}

func findPluginCommand(args []string, installedRoot, language string) (plugin.Bundle, plugin.Command, map[string]string, map[string]string, bool, error) {
	bundles, err := loadBundles(installedRoot)
	if err != nil {
		return plugin.Bundle{}, plugin.Command{}, nil, nil, false, err
	}
	for _, bundle := range bundles {
		command, commandArgs, rest, ok := plugin.FindCommandPrefixWithArgs(bundle, args)
		if ok {
			parameterValues, err := parseCommandParameters(command, rest, language)
			if err != nil {
				return plugin.Bundle{}, plugin.Command{}, nil, nil, false, err
			}
			return bundle, command, commandArgs, parameterValues, true, nil
		}
	}
	return plugin.Bundle{}, plugin.Command{}, nil, nil, false, nil
}

func parseCommandParameters(command plugin.Command, args []string, language string) (map[string]string, error) {
	byFlag := make(map[string]plugin.Parameter, len(command.Parameters))
	for _, parameter := range command.Parameters {
		byFlag[parameter.Flag] = parameter
	}

	values := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			return nil, localizedUnexpectedArgument(arg, command.ID, language)
		}
		flag := strings.TrimPrefix(arg, "--")
		value := ""
		if name, inline, ok := strings.Cut(flag, "="); ok {
			flag = name
			value = inline
		} else {
			i++
			if i >= len(args) {
				return nil, localizedFlagRequiresValue(flag, language)
			}
			value = args[i]
		}
		parameter, ok := byFlag[flag]
		if !ok {
			return nil, localizedUnknownOption(flag, command.ID, language)
		}
		if err := validateParameterValue(command, parameter, value, language); err != nil {
			return nil, err
		}
		values[parameter.Name] = value
	}

	for _, parameter := range command.Parameters {
		if parameter.Required && values[parameter.Name] == "" {
			return nil, localizedMissingRequiredFlag(command.ID, parameter.Flag, language)
		}
	}
	return values, nil
}

func validateParameterValue(command plugin.Command, parameter plugin.Parameter, value, language string) error {
	if len(parameter.AllowedValues) > 0 && !slices.Contains(parameter.AllowedValues, value) {
		return localizedAllowedValuesError(command.ID, parameter.Flag, strings.Join(parameter.AllowedValues, ","), language)
	}
	if parameter.Pattern != "" {
		matched, err := regexp.MatchString(parameter.Pattern, value)
		if err != nil {
			return localizedInvalidPattern(command.ID, parameter.Flag, err, language)
		}
		if !matched {
			return localizedPatternMismatch(command.ID, parameter.Flag, parameter.Pattern, language)
		}
	}
	return nil
}

func localizedConfirmationRequired(message, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 需要确认：请使用 --yes 重新执行", message)
	}
	return fmt.Errorf("confirmation required for %s: rerun with --yes", message)
}

func localizedUnexpectedArgument(arg, commandID, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 不支持参数 %q", commandID, arg)
	}
	return fmt.Errorf("unexpected argument %q for %s", arg, commandID)
}

func localizedFlagRequiresValue(flag, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("--%s 需要一个值", flag)
	}
	return fmt.Errorf("--%s requires a value", flag)
}

func localizedUnknownOption(flag, commandID, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 不支持选项 --%s", commandID, flag)
	}
	return fmt.Errorf("unknown option --%s for %s", flag, commandID)
}

func localizedMissingRequiredFlag(commandID, flag, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s 需要 --%s", commandID, flag)
	}
	return fmt.Errorf("%s requires --%s", commandID, flag)
}

func localizedAllowedValuesError(commandID, flag, allowed, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s --%s 必须是以下值之一 %s", commandID, flag, allowed)
	}
	return fmt.Errorf("%s --%s must be one of %s", commandID, flag, allowed)
}

func localizedInvalidPattern(commandID, flag string, err error, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s --%s 的校验表达式无效: %w", commandID, flag, err)
	}
	return fmt.Errorf("%s --%s has invalid validation pattern: %w", commandID, flag, err)
}

func localizedPatternMismatch(commandID, flag, pattern, language string) error {
	if language == "zh-CN" {
		return fmt.Errorf("%s --%s 不匹配 %s", commandID, flag, pattern)
	}
	return fmt.Errorf("%s --%s does not match %s", commandID, flag, pattern)
}

func loadBundles(installedRoot string) ([]plugin.Bundle, error) {
	dirs := append(pluginDirs(installedRoot), pluginDirs(defaultPluginRoot())...)
	bundles := make([]plugin.Bundle, 0, len(dirs))
	seen := make(map[string]bool, len(dirs))

	for _, dir := range dirs {
		bundle, err := plugin.LoadBundle(dir, version.Version)
		if err != nil {
			return nil, err
		}
		if seen[bundle.Manifest.Name] {
			continue
		}
		seen[bundle.Manifest.Name] = true
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}

func pluginDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return nil
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(root, entry.Name()))
		}
	}
	return dirs
}

func loadCommandResponse(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, opts globalOptions, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, debug io.Writer) (map[string]any, error) {
	if !opts.Offline {
		if opts.Timeout > 0 {
			profile.TimeoutSeconds = opts.Timeout
		}
		return executeAPICommand(bundle, command, commandArgs, parameterValues, profile, getenv, transport, debug)
	}
	if command.FixtureResponse == "" {
		return nil, fmt.Errorf("command %s has no fixture response for offline mode", command.ID)
	}

	data, err := os.ReadFile(filepath.Join(bundle.Dir, command.FixtureResponse))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse fixture response for %s: %w", command.ID, err)
	}
	return payload, nil
}

func executeAPICommand(bundle plugin.Bundle, command plugin.Command, commandArgs, parameterValues map[string]string, profile coreconfig.Profile, getenv func(string) string, transport http.RoundTripper, debug io.Writer) (map[string]any, error) {
	operation, ok := bundle.APIs.Operations[command.Operation]
	if !ok {
		return nil, fmt.Errorf("command %s references missing operation %s", command.ID, command.Operation)
	}
	endpointURL := profile.EndpointURL
	if endpointURL == "" {
		endpointURL = bundle.Manifest.API.EndpointURL
	}
	if endpointURL == "" {
		return nil, fmt.Errorf("command %s requires plugin api.endpoint_url or profile endpoint_url for live API execution", command.ID)
	}
	creds, err := coreconfig.LoadCredentialsFromEnv(getenv)
	if err != nil {
		return nil, err
	}

	bodyMap := resolveMap(operation.Body, profile, commandArgs, parameterValues, command.Parameters, len(operation.Body) > 0)
	var body []byte
	if len(bodyMap) > 0 {
		var err error
		body, err = json.Marshal(bodyMap)
		if err != nil {
			return nil, err
		}
	}
	query := encodeQuery(resolveMap(operation.Query, profile, commandArgs, parameterValues, command.Parameters, false))
	headers := resolveMap(operation.Headers, profile, commandArgs, parameterValues, command.Parameters, false)
	contentType := operation.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	retries := 0
	if operation.Retryable {
		retries = 2
	}
	var timeout time.Duration
	if profile.TimeoutSeconds > 0 {
		timeout = time.Duration(profile.TimeoutSeconds) * time.Second
	}
	return client.DoJSON(transport, client.RequestSpec{
		Method:      operation.Method,
		BaseURL:     endpointURL,
		Path:        operation.Path,
		Query:       query,
		ContentType: contentType,
		Body:        body,
		Headers:     headers,
		Credentials: creds,
		Timeout:     timeout,
		Retries:     retries,
		Debug:       debug,
	})
}

func debugWriter(opts globalOptions, stderr io.Writer) io.Writer {
	if !opts.Debug {
		return nil
	}
	return stderr
}

func resolveMap(values map[string]string, profile coreconfig.Profile, commandArgs, parameterValues map[string]string, parameters []plugin.Parameter, includeParameterTargets bool) map[string]string {
	resolved := make(map[string]string, len(values)+len(parameterValues))
	for key, value := range values {
		switch value {
		case "$profile.region":
			resolved[key] = profile.Region
		default:
			if strings.HasPrefix(value, "$arg.") {
				resolved[key] = commandArgs[strings.TrimPrefix(value, "$arg.")]
			} else if strings.HasPrefix(value, "$param.") {
				if parameterValue := parameterValues[strings.TrimPrefix(value, "$param.")]; parameterValue != "" {
					resolved[key] = parameterValue
				}
			} else {
				resolved[key] = value
			}
		}
	}
	if includeParameterTargets {
		for _, parameter := range parameters {
			if value, ok := parameterValues[parameter.Name]; ok && value != "" {
				resolved[parameter.Target] = value
			}
		}
	}
	return resolved
}

func encodeQuery(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	query := url.Values{}
	for key, value := range values {
		if value != "" {
			query.Set(key, value)
		}
	}
	return query.Encode()
}

func rowsFromPayload(payload map[string]any, table plugin.Table) ([]map[string]string, error) {
	rawRows, err := valueAtPath(payload, table.RowPath)
	if err != nil {
		return nil, err
	}
	rowValues, ok := rawRows.([]any)
	if !ok {
		return nil, fmt.Errorf("row path %q is not an array", table.RowPath)
	}

	rows := make([]map[string]string, 0, len(rowValues))
	for _, rawRow := range rowValues {
		rowMap, ok := rawRow.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("row path %q contains a non-object row", table.RowPath)
		}
		row := make(map[string]string, len(table.Columns))
		for _, column := range table.Columns {
			value, err := valueAtPath(rowMap, column.Path)
			if err == nil {
				row[column.Key] = fmt.Sprint(value)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func tableColumns(table plugin.Table, language string) []output.Column {
	columns := make([]output.Column, 0, len(table.Columns))
	for _, column := range table.Columns {
		catalog := i18n.Catalog{column.Key: column.Labels}
		columns = append(columns, output.Column{
			Key:   column.Key,
			Label: catalog.Text(column.Key, language),
		})
	}
	return columns
}

func valueAtPath(value any, path string) (any, error) {
	current := value
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path %q cannot read %q", path, part)
		}
		current, ok = object[part]
		if !ok {
			return nil, fmt.Errorf("path %q is missing %q", path, part)
		}
	}
	return current, nil
}

func pluginRoot(configured string) string {
	if configured != "" {
		return configured
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".ctyun", "plugins")
	}
	return ".ctyun/plugins"
}

func defaultPluginRoot() string {
	relative := "plugins"
	if _, err := os.Stat(relative); err == nil {
		return relative
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return relative
	}
	dir := filepath.Dir(file)
	for {
		candidate := filepath.Join(dir, relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return relative
		}
		dir = parent
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func compareVersion(left, right string) int {
	leftParts := parseVersion(left)
	rightParts := parseVersion(right)
	for i := 0; i < len(leftParts); i++ {
		if leftParts[i] < rightParts[i] {
			return -1
		}
		if leftParts[i] > rightParts[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(value string) [3]int {
	var result [3]int
	parts := strings.Split(value, ".")
	for i := 0; i < len(result) && i < len(parts); i++ {
		n, _ := strconv.Atoi(parts[i])
		result[i] = n
	}
	return result
}

func sortStrings(values []string) {
	slices.Sort(values)
}
