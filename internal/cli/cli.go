/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package cli coordinates command-line parsing, profile loading, plugin
// dispatch, and output rendering for the ctyun command.
package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// Config provides the process boundary for running the CLI from tests or main.
type Config struct {
	Args          []string
	Stdout        io.Writer
	Stderr        io.Writer
	Stdin         io.Reader
	PluginRoot    string
	Env           func(string) string
	Config        []byte
	ConfigPath    string
	HTTPTransport http.RoundTripper
}

// globalOptions captures options accepted before a core or plugin command.
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
	Help     bool
}

// globalOptionHelp describes one global flag for help and completion output.
type globalOptionHelp struct {
	Short   string
	Long    string
	Aliases []string
	Value   string
	Key     string
	Default string
}

// globalOptionsHelp is the shared global flag catalog.
var globalOptionsHelp = []globalOptionHelp{
	{Short: "-o", Long: "--output", Value: "table|json", Key: "option.output", Default: "table"},
	{Short: "-c", Long: "--cols", Value: "keys", Key: "option.cols"},
	{Short: "-H", Long: "--no-header", Key: "option.no_header"},
	{Short: "-f", Long: "--filter", Value: "key=value", Key: "option.filter"},
	{Short: "-s", Long: "--sort", Value: "key|-key", Key: "option.sort"},
	{Short: "-l", Long: "--lang", Aliases: []string{"--language"}, Value: "locale", Key: "option.lang"},
	{Short: "-y", Long: "--yes", Key: "option.yes"},
	{Short: "-w", Long: "--wait", Value: "waiter", Key: "option.wait"},
	{Short: "-t", Long: "--table", Value: "bordered|compact|plain", Key: "option.table", Default: "bordered"},
	{Short: "-T", Long: "--timeout", Value: "seconds", Key: "option.timeout"},
	{Short: "-C", Long: "--config", Value: "path", Key: "option.config"},
	{Short: "-P", Long: "--profile", Value: "name", Key: "option.profile"},
	{Short: "-d", Long: "--debug", Key: "option.debug"},
	{Short: "-h", Long: "--help", Key: "option.help"},
}

// Run executes one CLI invocation and returns a user-facing error on failure.
func Run(cfg Config) error {
	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	stdin := cfg.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	getenv := cfg.Env
	if getenv == nil {
		getenv = os.Getenv
	}

	if len(cfg.Args) > 0 && cfg.Args[0] == "__complete" {
		return runComplete(stdout, cfg.Args[1:], pluginRoot(cfg.PluginRoot))
	}

	opts, args, err := parseGlobalOptions(cfg.Args)
	if err != nil {
		return err
	}
	if opts.Output == "" {
		opts.Output = "table"
	}
	if opts.Offline && !version.IsDevelopmentBuild() {
		return diagnostic.New("error.fixture_dev_only")
	}
	resolvedConfigPath := configPath(opts.Config, cfg.ConfigPath, getenv)
	configBytes, err := loadConfigBytes(cfg.Config, resolvedConfigPath)
	if err != nil {
		return err
	}
	var profile coreconfig.Profile
	if len(args) == 0 || args[0] != "config" {
		profile, err = activeProfile(configBytes, opts.Profile)
		if err != nil {
			return err
		}
	}
	if opts.Language == "" {
		opts.Language = resolveCLILanguage(getenv, profile.Language)
	} else {
		opts.Language = i18n.ResolveLanguage(i18n.LanguageOptions{Flag: opts.Language})
	}

	if opts.Help {
		return runHelp(stdout, args, pluginRoot(cfg.PluginRoot), opts.Language)
	}

	if len(args) == 0 {
		fmt.Fprintln(stderr, missingCommandUsageLine(opts.Language))
		return diagnostic.New("error.missing_command")
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
		return runDoctor(stdout, args[1:], opts.Language)
	case "config":
		return runConfigCommand(stdout, stderr, stdin, args[1:], opts, configBytes, resolvedConfigPath)
	case "upgrade", "update":
		return runUpgrade(stdout, stderr, args[1:], getenv, cfg.HTTPTransport, opts.Language)
	case "plugin", "plugins":
		return runPluginWithOptions(stdout, stderr, stdin, pluginRoot(cfg.PluginRoot), args[1:], profile, getenv, cfg.HTTPTransport, opts)
	default:
		return runPluginCommand(stdout, stderr, stdin, opts, args, pluginRoot(cfg.PluginRoot), profile, getenv, cfg.HTTPTransport)
	}
}

// Execute runs the CLI, writes formatted errors to stderr, and returns a process
// exit code.
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
		message = client.RedactHTTPDetails(message, errorCredentials(cfg, getenv), "")
		fmt.Fprintln(stderr, message)
		return 1
	}
	return 0
}

// errorLanguage resolves the best language to use after command execution
// fails.
func errorLanguage(cfg Config, getenv func(string) string) string {
	opts, _, err := parseGlobalOptions(cfg.Args)
	if err != nil {
		return resolveCLILanguage(getenv, "")
	}
	if opts.Language != "" {
		return i18n.ResolveLanguage(i18n.LanguageOptions{Flag: opts.Language})
	}
	configBytes, err := loadConfigBytes(cfg.Config, configPath(opts.Config, cfg.ConfigPath, getenv))
	if err != nil {
		return resolveCLILanguage(getenv, "")
	}
	profile, _ := activeProfile(configBytes, opts.Profile)
	return resolveCLILanguage(getenv, profile.Language)
}

// errorCredentials resolves best-effort credentials for error redaction.
func errorCredentials(cfg Config, getenv func(string) string) coreconfig.Credentials {
	fallback := coreconfig.Credentials{
		AccessKey: getenv("CTYUN_AK"),
		SecretKey: getenv("CTYUN_SK"),
	}
	opts, _, err := parseGlobalOptions(cfg.Args)
	if err != nil {
		return fallback
	}
	configBytes, err := loadConfigBytes(cfg.Config, configPath(opts.Config, cfg.ConfigPath, getenv))
	if err != nil {
		return fallback
	}
	profile, err := activeProfile(configBytes, opts.Profile)
	if err != nil {
		return fallback
	}
	creds, err := coreconfig.ResolveCredentials(getenv, profile)
	if err != nil {
		return fallback
	}
	return creds
}

// resolveCLILanguage applies CLI language precedence from environment, profile,
// and OS locale.
func resolveCLILanguage(getenv func(string) string, profileLanguage string) string {
	return i18n.ResolveLanguage(i18n.LanguageOptions{
		Env:      getenv("CTYUN_LANGUAGE"),
		Profile:  profileLanguage,
		OSLocale: detectOSLocale(getenv),
	})
}

// detectOSLocale returns the most specific locale available from environment or
// platform helpers.
func detectOSLocale(getenv func(string) string) string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := strings.TrimSpace(getenv(key)); value != "" && !isCLocale(value) {
			return value
		}
	}
	if runtimeGOOS == "darwin" {
		return readDarwinAppleLocale()
	}
	if runtimeGOOS == "windows" {
		return readWindowsUserLocale()
	}
	return getenv("LANG")
}

// isCLocale reports whether value is the unlocalized C/POSIX locale.
func isCLocale(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if base, _, ok := strings.Cut(normalized, "."); ok {
		normalized = base
	}
	return normalized == "C" || normalized == "POSIX"
}

// readDarwinAppleLocale reads the macOS user locale when environment variables
// are not useful.
var readDarwinAppleLocale = func() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runtimeGOOS is replaceable in tests for locale branch coverage.
var runtimeGOOS = runtime.GOOS

// runtimeCaller is replaceable in tests for bundled plugin root discovery.
var runtimeCaller = runtime.Caller

// osStat is replaceable in tests for filesystem branch coverage.
var osStat = os.Stat

// renderOutputTable is replaceable in tests for output error paths.
var renderOutputTable = output.RenderTable

// renderOutputJSON is replaceable in tests for output error paths.
var renderOutputJSON = output.RenderJSON

// tempArtifactFile is the minimal temporary-file contract used for registry
// downloads.
type tempArtifactFile interface {
	Name() string
	Write([]byte) (int, error)
	Close() error
}

// createTempArtifactFile creates a temporary plugin artifact download target.
var createTempArtifactFile = func() (tempArtifactFile, error) {
	return os.CreateTemp("", "ctyun-plugin-*.tar.gz")
}

// formatError applies language-specific CLI error prefixes and translations.
func formatError(err error, language string) string {
	return messagef("error.prefix", language, localizedError(err, language))
}

// exactErrorMessageKeys maps internal stable errors to localized catalog keys.
var exactErrorMessageKeys = map[string]string{
	"missing command":                                                             "error.missing_command",
	"doctor supports: network":                                                    "error.doctor_supports",
	"plugin requires a subcommand":                                                "error.plugin_subcommand",
	"plugin install requires a plugin name":                                       "error.plugin_install_name",
	"plugin remove requires a plugin name":                                        "error.plugin_remove_name",
	"plugin lint requires a bundle path":                                          "error.plugin_lint_path",
	"plugin lint is only available in development builds":                         "error.plugin_lint_dev_only",
	"plugin update/upgrade --bundled requires a plugin name or --all":             "error.plugin_bundled_update_target",
	"plugin update/upgrade requires a plugin name or --all":                       "error.plugin_update_target",
	"hosted plugin metadata is unavailable for development builds; use --bundled": "error.hosted_plugin_dev",
	"plugin install accepts one plugin name":                                      "error.plugin_install_one_name",
	"plugin install accepts either --all or plugin names":                         "error.plugin_install_all_or_names",
	"plugin install accepts either --bundled or --source":                         "error.plugin_install_source_choice",
	"plugin search accepts one query":                                             "error.plugin_search_one_query",
	"plugin search accepts either --bundled or --source":                          "error.plugin_search_source_choice",
	"plugin list accepts either --available or --updates":                         "error.plugin_list_available_updates",
	"plugin list accepts either --bundled or --source":                            "error.plugin_list_source_choice",
	"plugin remove accepts either --all or plugin names":                          "error.plugin_remove_all_or_names",
	"confirmation required for plugin remove: rerun with --yes":                   "error.plugin_remove_confirm",
	"plugin remove confirmation required":                                         "error.plugin_remove_confirm",
	"plugin update accepts one plugin name":                                       "error.plugin_update_one_name",
	"plugin update accepts either --all or one plugin name":                       "error.plugin_update_all_or_one",
	"plugin update accepts either --bundled or --source":                          "error.plugin_update_source_choice",
	"plugin source is empty":                                                      "error.plugin_source_empty",
	"--source requires a value":                                                   "error.source_requires_value",
	"--channel requires a value":                                                  "error.channel_requires_value",
	"--bundled is only available in development builds":                           "error.bundled_dev_only",
	"fixture mode is only available in development builds":                        "error.fixture_dev_only",
	"completion requires one shell: bash, zsh, fish, or powershell":               "error.completion_shell_required",
	"config path is unavailable":                                                  "error.config_path_unavailable",
	"Usage: ctyun config show [--profile name]":                                   "usage.config.show",
	"Usage: ctyun config set <key> <value> [--profile name]":                      "usage.config.set",
	"Usage: ctyun config unset <key> [--profile name]":                            "usage.config.unset",
	"Usage: ctyun config profile <list|use|set|unset|set-secret|reset>":           "usage.config.profile",
	"Usage: ctyun config profile use <name>":                                      "usage.config.profile_use",
	"Usage: ctyun config profile set <name> <key=value|key value>":                "usage.config.profile_set",
	"Usage: ctyun config profile unset <name> <key>":                              "usage.config.profile_unset",
	"Usage: ctyun config profile set-secret <name> <ak|sk> --from-stdin":          "usage.config.profile_set_secret",
	"Usage: ctyun config profile reset <name>":                                    "usage.config.profile_reset",
	"config profile reset requires --yes":                                         "error.config_profile_reset_confirm",
	"config reset requires --yes":                                                 "error.config_reset_confirm",
	"config profile reset confirmation required":                                  "error.config_profile_reset_confirm",
	"config reset confirmation required":                                          "error.config_reset_confirm",
	"profile name cannot be empty":                                                "error.profile_name_empty",
	"ctyun API request failed":                                                    "error.api_request_failed",
}

// localizedErrorText translates selected internal error strings for users.
func localizedErrorText(message, language string) string {
	if key := exactErrorMessageKeys[message]; key != "" {
		return messageText(key, language)
	}
	if match := regexp.MustCompile(`^(.+) requires a value$`).FindStringSubmatch(message); match != nil {
		return messagef("error.requires_value", language, match[1])
	}
	if match := regexp.MustCompile(`^unknown upgrade option "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.upgrade_option", language, match[1])
	}
	if match := regexp.MustCompile(`^unknown plugin subcommand "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unknown_plugin_subcommand", language, match[1])
	}
	if match := regexp.MustCompile(`^invalid plugin name "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.plugin_name", language, match[1])
	}
	if match := regexp.MustCompile(`^plugin ([^ ]+) requires ctyun (.+), current version is (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.plugin_version", language, match[1], match[2], match[3])
	}
	if match := regexp.MustCompile(`^unknown command "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unknown_command", language, match[1])
	}
	if match := regexp.MustCompile(`^unsupported output "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unsupported_output", language, match[1])
	}
	if match := regexp.MustCompile(`^unknown filter key "(.+)"; use (?:stable column keys|a visible column label or stable key)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unknown_filter_key", language, match[1])
	}
	if match := regexp.MustCompile(`^unknown sort key "(.+)"; use (?:stable column keys|a visible column label or stable key)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unknown_sort_key", language, match[1])
	}
	if match := regexp.MustCompile(`^unknown waiter "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unknown_waiter", language, match[1])
	}
	if match := regexp.MustCompile(`^plugin root (.+) is not a directory$`).FindStringSubmatch(message); match != nil {
		return messagef("error.plugin_root_not_directory", language, match[1])
	}
	if match := regexp.MustCompile(`^plugin ([^ ]+) not found in registry$`).FindStringSubmatch(message); match != nil {
		return messagef("error.plugin_not_found_registry", language, match[1])
	}
	if match := regexp.MustCompile(`^unsupported shell "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unsupported_shell", language, match[1])
	}
	if match := regexp.MustCompile(`^unknown config subcommand "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unknown_config_subcommand", language, match[1])
	}
	if match := regexp.MustCompile(`^profile "(.+)" not found$`).FindStringSubmatch(message); match != nil {
		return messagef("error.profile_not_found", language, match[1])
	}
	if match := regexp.MustCompile(`^unsupported secret key "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unsupported_secret_key", language, match[1])
	}
	if match := regexp.MustCompile(`^active_profile "(.+)" does not exist$`).FindStringSubmatch(message); match != nil {
		return messagef("error.active_profile_missing", language, match[1])
	}
	if match := regexp.MustCompile(`^profile "(.+)" timeout_seconds cannot be negative$`).FindStringSubmatch(message); match != nil {
		return messagef("error.profile_timeout_negative", language, match[1])
	}
	if match := regexp.MustCompile(`^profile "(.+)" language "(.+)" is not supported$`).FindStringSubmatch(message); match != nil {
		return messagef("error.profile_language_unsupported", language, match[1], match[2])
	}
	if match := regexp.MustCompile(`^unsupported global config key "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unsupported_global_config_key", language, match[1])
	}
	if match := regexp.MustCompile(`^unsupported profile config key "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unsupported_profile_config_key", language, match[1])
	}
	if match := regexp.MustCompile(`^parse timeout_seconds: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.parse_timeout_seconds", language, match[1])
	}
	if match := regexp.MustCompile(`^parse warn_config_credentials: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.parse_warn_config_credentials", language, match[1])
	}
	if match := regexp.MustCompile(`^validate config: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.validate_config", language, localizedErrorText(match[1], language))
	}
	if match := regexp.MustCompile(`^parse response JSON: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.parse_response_json", language, localizedErrorText(match[1], language))
	}
	if match := regexp.MustCompile(`^ctyun API returned HTTP ([0-9]+): (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.api_http", language, match[1], match[2])
	}
	if match := regexp.MustCompile(`^no ctyun release found for ([^/]+)/([^ ]+) on channel (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_not_found", language, match[1], match[2], match[3])
	}
	if match := regexp.MustCompile(`^unsupported release index schema ([0-9]+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.unsupported_release_schema", language, match[1])
	}
	if match := regexp.MustCompile(`^(.+) is missing version$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_missing_version", language, match[1])
	}
	if match := regexp.MustCompile(`^(.+) has invalid version "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_invalid_version", language, match[1], match[2])
	}
	if match := regexp.MustCompile(`^(.+) has unsupported channel "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_unsupported_channel", language, match[1], match[2])
	}
	if match := regexp.MustCompile(`^(.+) has no artifacts$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_no_artifacts", language, match[1])
	}
	if match := regexp.MustCompile(`^(.+) is missing os$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_missing_os", language, match[1])
	}
	if match := regexp.MustCompile(`^(.+) is missing arch$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_missing_arch", language, match[1])
	}
	if match := regexp.MustCompile(`^(.+) is missing url$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_missing_url", language, match[1])
	}
	if match := regexp.MustCompile(`^(.+) has invalid artifact url "(.+)"$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_invalid_artifact_url", language, match[1], match[2])
	}
	if match := regexp.MustCompile(`^(.+) has invalid sha256$`).FindStringSubmatch(message); match != nil {
		return messagef("error.release_invalid_sha256", language, match[1])
	}
	if match := regexp.MustCompile(`^read (.+) index: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.read_index", language, match[1], localizedErrorText(match[2], language))
	}
	if match := regexp.MustCompile(`^read (.+) index signature: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.read_index_signature", language, match[1], localizedErrorText(match[2], language))
	}
	if match := regexp.MustCompile(`^(.+) index signature: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.index_signature", language, match[1], localizedErrorText(match[2], language))
	}
	if match := regexp.MustCompile(`^(.+) index requires a trusted public key$`).FindStringSubmatch(message); match != nil {
		return messagef("error.index_public_key_required", language, match[1])
	}
	if match := regexp.MustCompile(`^decode (.+) public key: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.decode_public_key", language, match[1], localizedErrorText(match[2], language))
	}
	if match := regexp.MustCompile(`^(.+) public key has length ([0-9]+), want ([0-9]+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.public_key_length", language, match[1], match[2], match[3])
	}
	if match := regexp.MustCompile(`^decode (.+) signature: (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.decode_signature", language, match[1], localizedErrorText(match[2], language))
	}
	if match := regexp.MustCompile(`^(.+) index signature verification failed$`).FindStringSubmatch(message); match != nil {
		return messagef("error.signature_failed", language, match[1])
	}
	if match := regexp.MustCompile(`^GET (.+) returned (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.http_get_status", language, match[1], match[2])
	}
	if match := regexp.MustCompile(`^(.+) requires sha256$`).FindStringSubmatch(message); match != nil {
		return messagef("error.artifact_requires_sha256", language, match[1])
	}
	if match := regexp.MustCompile(`^sha256 mismatch for (.+): got (.+), want (.+)$`).FindStringSubmatch(message); match != nil {
		return messagef("error.sha256_mismatch", language, match[1], match[2], match[3])
	}
	return message
}

// localizedError translates structured diagnostics and selected legacy error
// strings for users.
func localizedError(err error, language string) string {
	var diagnosticErr interface {
		MessageKey() string
		MessageArgs() []any
		Unwrap() error
	}
	if errors.As(err, &diagnosticErr) {
		args := slices.Clone(diagnosticErr.MessageArgs())
		if cause := diagnosticErr.Unwrap(); cause != nil {
			args = append(args, localizedError(cause, language))
		}
		return messagef(diagnosticErr.MessageKey(), language, args...)
	}
	return localizedErrorText(err.Error(), language)
}

// runDoctor prints local diagnostic hints for supported doctor topics.
func runDoctor(stdout io.Writer, args []string, language string) error {
	if len(args) != 1 || args[0] != "network" {
		return diagnostic.New("error.doctor_supports")
	}
	for _, message := range doctorNetworkMessages(language) {
		fmt.Fprintln(stdout, message)
	}
	return nil
}

// parseGlobalOptions separates leading global options from command arguments.
func parseGlobalOptions(args []string) (globalOptions, []string, error) {
	var opts globalOptions
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--output", "-o":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Output = args[i]
		case "--cols", "-c":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Columns = splitCSV(args[i])
		case "--no-header", "-H":
			opts.NoHeader = true
		case "--offline", "--fixture", "-O":
			opts.Offline = true
		case "--debug", "-d":
			opts.Debug = true
		case "--yes", "-y":
			opts.Yes = true
		case "--wait", "-w":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Waiter = args[i]
		case "--table", "-t":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Table = args[i]
		case "--timeout", "-T":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			timeout, err := strconv.Atoi(args[i])
			if err != nil || timeout <= 0 {
				return opts, nil, diagnostic.New("error.requires_positive_seconds", arg)
			}
			opts.Timeout = timeout
		case "--config", "-C":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Config = args[i]
		case "--profile", "-P":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Profile = args[i]
		case "--filter", "-f":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Filter = args[i]
		case "--sort", "-s":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Sort = args[i]
		case "--language", "--lang", "-l":
			i++
			if i >= len(args) {
				return opts, nil, diagnostic.New("error.requires_value", arg)
			}
			opts.Language = args[i]
		case "--help", "-h":
			opts.Help = true
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

// loadConfigBytes returns injected test config or reads an optional config file.
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

// configPath applies config path precedence from flag, embedded config, env,
// and user home.
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

// activeProfile resolves the selected profile from raw config bytes.
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
			return coreconfig.Profile{}, diagnostic.New("error.profile_not_found", profileName)
		}
		return cfg.ApplyProfileDefaults(profile), nil
	}
	profile, ok := cfg.ActiveProfile()
	if !ok && len(cfg.Profiles) > 0 {
		return coreconfig.Profile{}, diagnostic.New("error.config_multiple_profiles")
	}
	return cfg.ApplyProfileDefaults(profile), nil
}

// pluginRoot returns the user plugin root from config or the default home path.
func pluginRoot(configured string) string {
	if configured != "" {
		return configured
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".ctyun", "plugins")
	}
	return ".ctyun/plugins"
}

// defaultPluginRoot locates the bundled plugin directory for development and
// tests.
func defaultPluginRoot() string {
	relative := "plugins"
	if _, err := os.Stat(relative); err == nil {
		return relative
	}
	_, file, _, ok := runtimeCaller(0)
	if !ok {
		return relative
	}
	// Tests and installed binaries may run outside the repo root; walk upward
	// from this source file so bundled plugin fixtures remain discoverable.
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

// splitCSV parses comma-separated CLI values while dropping empty items.
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

// sortStrings sorts values in place.
func sortStrings(values []string) {
	slices.Sort(values)
}
