/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package cli coordinates command-line parsing, profile loading, plugin
// dispatch, and output rendering for the ctyun command.
package cli

import (
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
	"github.com/ArvinZJC/ctyun-cli/internal/i18n"
	"github.com/ArvinZJC/ctyun-cli/internal/output"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// Config provides the process boundary for running the CLI from tests or main.
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
	Help     bool
}

type globalOptionHelp struct {
	Short   string
	Long    string
	Aliases []string
	Value   string
	Key     string
}

var globalOptionsHelp = []globalOptionHelp{
	{Short: "-o", Long: "--output", Value: "table|json", Key: "option.output"},
	{Short: "-c", Long: "--cols", Value: "keys", Key: "option.cols"},
	{Short: "-H", Long: "--no-header", Key: "option.no_header"},
	{Short: "-f", Long: "--filter", Value: "key=value", Key: "option.filter"},
	{Short: "-s", Long: "--sort", Value: "key|-key", Key: "option.sort"},
	{Short: "-l", Long: "--lang", Aliases: []string{"--language"}, Value: "locale", Key: "option.lang"},
	{Short: "-y", Long: "--yes", Key: "option.yes"},
	{Short: "-w", Long: "--wait", Value: "waiter", Key: "option.wait"},
	{Short: "-t", Long: "--table", Value: "bordered|compact|plain", Key: "option.table"},
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
		opts.Language = resolveCLILanguage(getenv, profile.Language)
	} else {
		opts.Language = i18n.ResolveLanguage(i18n.LanguageOptions{Flag: opts.Language})
	}

	if opts.Help {
		return runHelp(stdout, args, pluginRoot(cfg.PluginRoot), opts.Language)
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
	case "upgrade", "update":
		fmt.Fprintln(stdout, "updating or upgrading the core ctyun binary is deferred; install core updates through your package manager for now")
		fmt.Fprintln(stdout, "for plugin updates, run ctyun plugin|plugins update|upgrade")
		return nil
	case "plugin", "plugins":
		return runPluginWithOptions(stdout, pluginRoot(cfg.PluginRoot), args[1:], profile, getenv, cfg.HTTPTransport, opts)
	default:
		return runPluginCommand(stdout, stderr, opts, args, pluginRoot(cfg.PluginRoot), profile, getenv, cfg.HTTPTransport)
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

func resolveCLILanguage(getenv func(string) string, profileLanguage string) string {
	return i18n.ResolveLanguage(i18n.LanguageOptions{
		Env:      getenv("CTYUN_LANGUAGE"),
		Profile:  profileLanguage,
		OSLocale: detectOSLocale(getenv),
	})
}

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

func isCLocale(value string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if base, _, ok := strings.Cut(normalized, "."); ok {
		normalized = base
	}
	return normalized == "C" || normalized == "POSIX"
}

var readDarwinAppleLocale = func() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var runtimeGOOS = runtime.GOOS
var runtimeCaller = runtime.Caller
var osStat = os.Stat
var renderOutputTable = output.RenderTable
var renderOutputJSON = output.RenderJSON

type tempArtifactFile interface {
	Name() string
	Write([]byte) (int, error)
	Close() error
}

var createTempArtifactFile = func() (tempArtifactFile, error) {
	return os.CreateTemp("", "ctyun-plugin-*.tar.gz")
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
		case "--output", "-o":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Output = args[i]
		case "--cols", "-c":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
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
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Waiter = args[i]
		case "--table", "-t":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Table = args[i]
		case "--timeout", "-T":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			timeout, err := strconv.Atoi(args[i])
			if err != nil || timeout <= 0 {
				return opts, nil, fmt.Errorf("%s requires a positive integer number of seconds", arg)
			}
			opts.Timeout = timeout
		case "--config", "-C":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Config = args[i]
		case "--profile", "-P":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Profile = args[i]
		case "--filter", "-f":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Filter = args[i]
		case "--sort", "-s":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.Sort = args[i]
		case "--language", "--lang", "-l":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
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
