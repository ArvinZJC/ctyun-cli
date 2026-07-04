/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
)

// credentialMask is the fixed middle segment for displayed AK/SK values.
const credentialMask = "*****"

// configSecretKeys returns credential keys accepted by set-secret.
func configSecretKeys() []string {
	return []string{"ak", "sk"}
}

// validConfigSecretKey reports whether key is accepted by set-secret.
func validConfigSecretKey(key string) bool {
	for _, candidate := range configSecretKeys() {
		if key == candidate {
			return true
		}
	}
	return false
}

// runConfigCommand executes non-interactive configuration management commands.
func runConfigCommand(stdout, stderr io.Writer, stdin io.Reader, args []string, opts globalOptions, raw []byte, path string) error {
	if len(args) == 0 {
		_, err := printConfigHelp(stdout, []string{"config"}, opts.Language)
		return err
	}
	switch args[0] {
	case "path":
		return runConfigPath(stdout, path)
	case "show":
		return runConfigShow(stdout, args[1:], opts, raw)
	case "set":
		return runConfigSet(stdout, raw, path, opts.Profile, args[1:], opts.Language)
	case "unset":
		return runConfigUnset(stdout, raw, path, opts.Profile, args[1:], opts.Language)
	case "profile", "profiles":
		return runConfigProfile(stdout, stderr, stdin, raw, path, opts, args[1:])
	case "reset":
		return runConfigReset(stdout, stderr, stdin, path, opts)
	default:
		return diagnostic.New("error.unknown_config_subcommand", args[0])
	}
}

// configSubcommandHelp describes one config command subcommand.
type configSubcommandHelp struct {
	Name           string
	Aliases        []string
	DescriptionKey string
	Usage          string
	Options        []pluginOptionSummary
}

// printConfigHelp writes structured help for config commands.
func printConfigHelp(stdout io.Writer, args []string, language string) (bool, error) {
	if len(args) == 1 {
		writer := newOutputWriter(stdout)
		writer.Line(helpPageText("config.description", language))
		writer.Format("\n%s:\n", helpText("usage.heading", language))
		writer.Lines(
			"  ctyun config <subcommand> [options]",
			"  ctyun help config <subcommand>",
		)
		writeConfigSubcommandList(writer, configSubcommandSummaries(), language)
		return true, writer.Err()
	}
	if len(args) < 2 {
		return false, nil
	}
	if args[1] == "profile" || args[1] == "profiles" {
		return printConfigProfileHelp(stdout, args, language)
	}
	if len(args) != 2 {
		return false, nil
	}
	for _, command := range configSubcommandSummaries() {
		if configSubcommandMatches(command, args[1]) {
			return true, printConfigSubcommandHelp(stdout, command, language)
		}
	}
	return false, nil
}

// printConfigProfileHelp writes structured help for profile config commands.
func printConfigProfileHelp(stdout io.Writer, args []string, language string) (bool, error) {
	if len(args) == 2 {
		writer := newOutputWriter(stdout)
		writer.Line(helpPageText("config.profile.description", language))
		writer.Format("\n%s:\n", helpText("usage.heading", language))
		writer.Lines(
			"  ctyun config profile <subcommand> [options]",
			"  ctyun config profiles <subcommand> [options]",
			"  ctyun help config profile <subcommand>",
		)
		writeConfigSubcommandList(writer, configProfileSubcommandSummaries(), language)
		return true, writer.Err()
	}
	if len(args) != 3 {
		return false, nil
	}
	for _, command := range configProfileSubcommandSummaries() {
		if configSubcommandMatches(command, args[2]) {
			return true, printConfigSubcommandHelp(stdout, command, language)
		}
	}
	return false, nil
}

// writeConfigSubcommandList writes aligned config subcommand help rows.
func writeConfigSubcommandList(writer *outputWriter, commands []configSubcommandHelp, language string) {
	writer.Format("\n%s:\n", helpText("subcommands.heading", language))
	maxNameWidth := 0
	for _, command := range commands {
		if len(configSubcommandNames(command)) > maxNameWidth {
			maxNameWidth = len(configSubcommandNames(command))
		}
	}
	for _, command := range commands {
		writer.Format("  %-*s  %s\n", maxNameWidth, configSubcommandNames(command), helpText(command.DescriptionKey, language))
	}
}

// printConfigSubcommandHelp writes usage and options for one config subcommand.
func printConfigSubcommandHelp(stdout io.Writer, command configSubcommandHelp, language string) error {
	writer := newOutputWriter(stdout)
	writer.Line(helpPageText(command.DescriptionKey, language))
	writer.Format("\n%s:\n", helpText("usage.heading", language))
	writer.Format("  %s\n", command.Usage)
	if len(command.Options) == 0 {
		return writer.Err()
	}
	writer.Format("\n%s:\n", helpText("command.heading", language))
	maxNameWidth := 0
	for _, option := range command.Options {
		if len(option.Name) > maxNameWidth {
			maxNameWidth = len(option.Name)
		}
	}
	for _, option := range command.Options {
		writer.Format("  %-*s  %s\n", maxNameWidth, option.Name, optionHelpText(option.Key, option.Default, language))
	}
	return writer.Err()
}

// configSubcommandSummaries returns help definitions for config subcommands.
func configSubcommandSummaries() []configSubcommandHelp {
	return []configSubcommandHelp{
		{Name: "path", DescriptionKey: "config.path.description", Usage: "ctyun config path"},
		{Name: "show", DescriptionKey: "config.show.description", Usage: "ctyun config show [--profile name]"},
		{Name: "set", DescriptionKey: "config.set.description", Usage: "ctyun config set <key> <value> [--profile name]"},
		{Name: "unset", DescriptionKey: "config.unset.description", Usage: "ctyun config unset <key> [--profile name]"},
		{Name: "profile", Aliases: []string{"profiles"}, DescriptionKey: "config.profile.description", Usage: "ctyun config profile <subcommand> [options]"},
		{Name: "reset", DescriptionKey: "config.reset.description", Usage: "ctyun config reset"},
	}
}

// configProfileSubcommandSummaries returns help definitions for profile config
// subcommands.
func configProfileSubcommandSummaries() []configSubcommandHelp {
	return []configSubcommandHelp{
		{Name: "list", DescriptionKey: "config.profile.list.description", Usage: "ctyun config profile list"},
		{Name: "use", DescriptionKey: "config.profile.use.description", Usage: "ctyun config profile use <name>"},
		{Name: "set", DescriptionKey: "config.profile.set.description", Usage: "ctyun config profile set <name> <key=value|key value>"},
		{Name: "unset", DescriptionKey: "config.profile.unset.description", Usage: "ctyun config profile unset <name> <key>"},
		{
			Name:           "set-secret",
			DescriptionKey: "config.profile.set_secret.description",
			Usage:          "ctyun config profile set-secret <name> <ak|sk> --from-stdin",
			Options: []pluginOptionSummary{
				{Name: "--from-stdin", Key: "config.option.from_stdin"},
			},
		},
		{Name: "reset", DescriptionKey: "config.profile.reset.description", Usage: "ctyun config profile reset <name>"},
	}
}

// configSubcommandNames joins a config command and aliases for display.
func configSubcommandNames(command configSubcommandHelp) string {
	names := append([]string{command.Name}, command.Aliases...)
	return strings.Join(names, "|")
}

// configSubcommandMatches reports whether name selects command or its alias.
func configSubcommandMatches(command configSubcommandHelp, name string) bool {
	if command.Name == name {
		return true
	}
	for _, alias := range command.Aliases {
		if alias == name {
			return true
		}
	}
	return false
}

// runConfigPath prints the resolved config path.
func runConfigPath(stdout io.Writer, path string) error {
	if path == "" {
		return diagnostic.New("error.config_path_unavailable")
	}
	_, err := fmt.Fprintln(stdout, path)
	return err
}

// runConfigShow prints redacted config JSON.
func runConfigShow(stdout io.Writer, args []string, opts globalOptions, raw []byte) error {
	if len(args) != 0 {
		return diagnostic.New("usage.config.show")
	}
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	if opts.Profile != "" {
		profile, ok := cfg.Profiles[opts.Profile]
		if !ok {
			return diagnostic.New("error.profile_not_found", opts.Profile)
		}
		return writeJSON(stdout, redactProfile(profile), true)
	}
	return writeJSON(stdout, redactConfig(cfg), true)
}

// runConfigSet writes one global or profile-scoped config key.
func runConfigSet(stdout io.Writer, raw []byte, path, profileName string, args []string, language string) error {
	if len(args) != 2 {
		return diagnostic.New("usage.config.set")
	}
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	if profileName != "" {
		if err := setProfileValue(&cfg, profileName, args[0], args[1]); err != nil {
			return err
		}
	} else if err := setGlobalConfigValue(&cfg, args[0], args[1]); err != nil {
		return err
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, configUpdatedMessage(language, args[0]))
	return err
}

// runConfigUnset clears one global or profile-scoped config key.
func runConfigUnset(stdout io.Writer, raw []byte, path, profileName string, args []string, language string) error {
	if len(args) != 1 {
		return diagnostic.New("usage.config.unset")
	}
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	if profileName != "" {
		if err := unsetProfileValue(&cfg, profileName, args[0]); err != nil {
			return err
		}
	} else if err := unsetGlobalConfigValue(&cfg, args[0]); err != nil {
		return err
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, configUnsetMessage(language, args[0]))
	return err
}

// runConfigProfile executes profile management subcommands.
func runConfigProfile(stdout, stderr io.Writer, stdin io.Reader, raw []byte, path string, opts globalOptions, args []string) error {
	if len(args) == 0 {
		return diagnostic.New("usage.config.profile")
	}
	switch args[0] {
	case "list":
		return runConfigProfileList(stdout, raw)
	case "use":
		return runConfigProfileUse(stdout, raw, path, args[1:], opts.Language)
	case "set":
		return runConfigProfileSet(stdout, raw, path, args[1:], opts.Language)
	case "unset":
		return runConfigProfileUnset(stdout, raw, path, args[1:], opts.Language)
	case "set-secret":
		return runConfigProfileSetSecret(stdout, stdin, raw, path, args[1:], opts.Language)
	case "reset":
		return runConfigProfileReset(stdout, stderr, stdin, raw, path, opts, args[1:])
	default:
		return diagnostic.New("error.unknown_config_profile_subcommand", args[0])
	}
}

// runConfigProfileList prints configured profiles with the active one marked.
func runConfigProfileList(stdout io.Writer, raw []byte) error {
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		marker := " "
		if name == cfg.ActiveProfileName {
			marker = "*"
		}
		if _, err := fmt.Fprintf(stdout, "%s %s\n", marker, name); err != nil {
			return err
		}
	}
	return nil
}

// runConfigProfileUse persists the active profile name.
func runConfigProfileUse(stdout io.Writer, raw []byte, path string, args []string, language string) error {
	if len(args) != 1 {
		return diagnostic.New("usage.config.profile_use")
	}
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[args[0]]; !ok {
		return diagnostic.New("error.profile_not_found", args[0])
	}
	cfg.ActiveProfileName = args[0]
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, configActiveProfileMessage(language, args[0]))
	return err
}

// runConfigProfileSet writes a profile key using either key=value or key value.
func runConfigProfileSet(stdout io.Writer, raw []byte, path string, args []string, language string) error {
	if len(args) != 2 && len(args) != 3 {
		return diagnostic.New("usage.config.profile_set")
	}
	name := args[0]
	key, value, err := profileSetPair(args[1:])
	if err != nil {
		return err
	}
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	if err := setProfileValue(&cfg, name, key, value); err != nil {
		return err
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, configProfileUpdatedMessage(language, name, key))
	return err
}

// runConfigProfileUnset clears one profile key.
func runConfigProfileUnset(stdout io.Writer, raw []byte, path string, args []string, language string) error {
	if len(args) != 2 {
		return diagnostic.New("usage.config.profile_unset")
	}
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	if err := unsetProfileValue(&cfg, args[0], args[1]); err != nil {
		return err
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, configProfileUnsetMessage(language, args[0], args[1]))
	return err
}

// runConfigProfileSetSecret writes a profile AK/SK read from stdin.
func runConfigProfileSetSecret(stdout io.Writer, stdin io.Reader, raw []byte, path string, args []string, language string) error {
	if len(args) != 3 || args[2] != "--from-stdin" {
		return diagnostic.New("usage.config.profile_set_secret")
	}
	if !validConfigSecretKey(args[1]) {
		return diagnostic.New("error.unsupported_secret_key", args[1])
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return err
	}
	value := strings.TrimRight(string(data), "\r\n")
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	_ = setProfileValue(&cfg, args[0], args[1], value)
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, configProfileUpdatedMessage(language, args[0], args[1]))
	return err
}

// runConfigProfileReset deletes one profile after explicit confirmation.
func runConfigProfileReset(stdout, stderr io.Writer, stdin io.Reader, raw []byte, path string, opts globalOptions, args []string) error {
	if len(args) != 1 {
		return diagnostic.New("usage.config.profile_reset")
	}
	if err := confirmDangerousOperation(stderr, stdin, opts, "config profile reset"); err != nil {
		return err
	}
	cfg, err := mutableConfig(raw)
	if err != nil {
		return err
	}
	delete(cfg.Profiles, args[0])
	if cfg.ActiveProfileName == args[0] {
		cfg.ActiveProfileName = ""
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, configProfileResetMessage(opts.Language, args[0]))
	return err
}

// runConfigReset removes the config file after backing it up.
func runConfigReset(stdout, stderr io.Writer, stdin io.Reader, path string, opts globalOptions) error {
	if err := confirmDangerousOperation(stderr, stdin, opts, "config reset"); err != nil {
		return err
	}
	if path == "" {
		return diagnostic.New("error.config_path_unavailable")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, writeErr := fmt.Fprintln(stdout, configAlreadyResetMessage(opts.Language))
		return writeErr
	} else if err != nil {
		return err
	}
	backupPath, err := backupConfigFile(path)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err == nil {
		_, err = fmt.Fprintln(stdout, configResetMessage(opts.Language, backupPath))
	}
	return err
}

// mutableConfig loads config bytes or returns an empty mutable config.
func mutableConfig(raw []byte) (coreconfig.Config, error) {
	if len(raw) == 0 {
		return coreconfig.Config{Profiles: make(map[string]coreconfig.Profile)}, nil
	}
	return coreconfig.Load(raw)
}

// writeConfigFile persists config JSON with owner-only default permissions.
func writeConfigFile(path string, cfg coreconfig.Config) error {
	if path == "" {
		return diagnostic.New("error.config_path_unavailable")
	}
	data, err := validatedConfigJSON(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// validatedConfigJSON marshals config and verifies it can be loaded before
// persistence.
func validatedConfigJSON(cfg coreconfig.Config) ([]byte, error) {
	return validatedConfigJSONWith(cfg, validateConfigBytes)
}

// validatedConfigJSONWith marshals config and applies validate() to the resulting
// bytes.
func validatedConfigJSONWith(cfg coreconfig.Config, validate func([]byte) error) ([]byte, error) {
	if err := validateConfigForWrite(cfg); err != nil {
		return nil, err
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	data = append(data, '\n')
	if err := validate(data); err != nil {
		return nil, err
	}
	return data, nil
}

// validateConfigBytes verifies serialized config is accepted by the config
// loader.
func validateConfigBytes(data []byte) error {
	if _, err := coreconfig.Load(data); err != nil {
		return diagnostic.Wrap("error.validate_config", err)
	}
	return nil
}

// validateConfigForWrite checks semantic config constraints before writing.
func validateConfigForWrite(cfg coreconfig.Config) error {
	if cfg.ActiveProfileName != "" {
		if _, ok := cfg.Profiles[cfg.ActiveProfileName]; !ok {
			return diagnostic.New("error.active_profile_missing", cfg.ActiveProfileName)
		}
	}
	for name, profile := range cfg.Profiles {
		if strings.TrimSpace(name) == "" {
			return diagnostic.New("error.profile_name_empty")
		}
		if profile.TimeoutSeconds < 0 {
			return diagnostic.New("error.profile_timeout_negative", name)
		}
		if profile.Language != "" && !validConfigLanguage(profile.Language) {
			return diagnostic.New("error.profile_language_unsupported", name, profile.Language)
		}
	}
	return nil
}

// validConfigLanguage reports whether language is accepted in config files.
func validConfigLanguage(language string) bool {
	switch language {
	case "zh-CN", "en-US", "en-GB":
		return true
	default:
		return false
	}
}

// writeJSON writes compact or indented JSON followed by a newline.
func writeJSON(stdout io.Writer, value any, indent bool) error {
	var (
		data []byte
		err  error
	)
	if indent {
		data, err = json.MarshalIndent(value, "", "  ")
	} else {
		data, err = json.Marshal(value)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "%s\n", data)
	return err
}

// redactConfig returns a display copy with credential values redacted.
func redactConfig(cfg coreconfig.Config) coreconfig.Config {
	if cfg.AccessKey != "" {
		cfg.AccessKey = maskCredentialValue(cfg.AccessKey)
	}
	if cfg.SecretKey != "" {
		cfg.SecretKey = maskCredentialValue(cfg.SecretKey)
	}
	for name, profile := range cfg.Profiles {
		cfg.Profiles[name] = redactProfile(profile)
	}
	return cfg
}

// redactProfile returns a display copy with credential values redacted.
func redactProfile(profile coreconfig.Profile) coreconfig.Profile {
	if profile.AccessKey != "" {
		profile.AccessKey = maskCredentialValue(profile.AccessKey)
	}
	if profile.SecretKey != "" {
		profile.SecretKey = maskCredentialValue(profile.SecretKey)
	}
	return profile
}

// maskCredentialValue keeps enough shape to show a credential is configured
// without printing the full value.
func maskCredentialValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return credentialMask
	}
	return value[:2] + credentialMask + value[len(value)-2:]
}

// setGlobalConfigValue writes one top-level config value.
func setGlobalConfigValue(cfg *coreconfig.Config, key, value string) error {
	switch key {
	case "active_profile":
		if value != "" {
			if _, ok := cfg.Profiles[value]; !ok {
				return diagnostic.New("error.profile_not_found", value)
			}
		}
		cfg.ActiveProfileName = value
	case "ak":
		cfg.AccessKey = value
	case "sk":
		cfg.SecretKey = value
	case "warn_config_credentials":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return diagnostic.Wrap("error.parse_warn_config_credentials", err)
		}
		cfg.WarnConfigCredentials = &parsed
	default:
		return diagnostic.New("error.unsupported_global_config_key", key)
	}
	return nil
}

// unsetGlobalConfigValue clears one top-level config value.
func unsetGlobalConfigValue(cfg *coreconfig.Config, key string) error {
	switch key {
	case "active_profile":
		cfg.ActiveProfileName = ""
	case "ak":
		cfg.AccessKey = ""
	case "sk":
		cfg.SecretKey = ""
	case "warn_config_credentials":
		cfg.WarnConfigCredentials = nil
	default:
		return diagnostic.New("error.unsupported_global_config_key", key)
	}
	return nil
}

// setProfileValue writes one profile value, creating the profile when needed.
func setProfileValue(cfg *coreconfig.Config, name, key, value string) error {
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]coreconfig.Profile)
	}
	profile := cfg.Profiles[name]
	if err := applyProfileValue(&profile, key, value); err != nil {
		return err
	}
	cfg.Profiles[name] = profile
	return nil
}

// unsetProfileValue clears one profile value.
func unsetProfileValue(cfg *coreconfig.Config, name, key string) error {
	profile, ok := cfg.Profiles[name]
	if !ok {
		return diagnostic.New("error.profile_not_found", name)
	}
	if err := clearProfileValue(&profile, key); err != nil {
		return err
	}
	cfg.Profiles[name] = profile
	return nil
}

// applyProfileValue writes one supported profile field.
func applyProfileValue(profile *coreconfig.Profile, key, value string) error {
	switch key {
	case "region":
		profile.Region = value
	case "language":
		profile.Language = value
	case "registry_url":
		profile.RegistryURL = value
	case "registry_public_key":
		profile.RegistryPublicKey = value
	case "registry.url":
		profile.Registry.URL = value
		profile.RegistryURL = value
	case "registry.public_key":
		profile.Registry.PublicKey = value
		profile.RegistryPublicKey = value
	case "endpoint_url":
		profile.EndpointURL = value
	case "timeout_seconds":
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return diagnostic.Wrap("error.parse_timeout_seconds", err)
		}
		profile.TimeoutSeconds = seconds
	case "ak":
		profile.AccessKey = value
	case "sk":
		profile.SecretKey = value
	case "warn_config_credentials":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return diagnostic.Wrap("error.parse_warn_config_credentials", err)
		}
		profile.WarnConfigCredentials = &parsed
	default:
		return diagnostic.New("error.unsupported_profile_config_key", key)
	}
	return nil
}

// clearProfileValue clears one supported profile field.
func clearProfileValue(profile *coreconfig.Profile, key string) error {
	switch key {
	case "region":
		profile.Region = ""
	case "language":
		profile.Language = ""
	case "registry_url":
		profile.RegistryURL = ""
	case "registry_public_key":
		profile.RegistryPublicKey = ""
	case "registry.url":
		profile.Registry.URL = ""
		profile.RegistryURL = ""
	case "registry.public_key":
		profile.Registry.PublicKey = ""
		profile.RegistryPublicKey = ""
	case "endpoint_url":
		profile.EndpointURL = ""
	case "timeout_seconds":
		profile.TimeoutSeconds = 0
	case "ak":
		profile.AccessKey = ""
	case "sk":
		profile.SecretKey = ""
	case "warn_config_credentials":
		profile.WarnConfigCredentials = nil
	default:
		return diagnostic.New("error.unsupported_profile_config_key", key)
	}
	return nil
}

// profileSetPair parses profile set arguments.
func profileSetPair(args []string) (string, string, error) {
	if len(args) == 1 {
		key, value, ok := strings.Cut(args[0], "=")
		if !ok || key == "" {
			return "", "", diagnostic.New("usage.config.profile_set")
		}
		return key, value, nil
	}
	return args[0], args[1], nil
}

// backupConfigFile copies the config file to a non-overwriting backup path.
func backupConfigFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	backupPath := path + ".bak"
	for i := 2; ; i++ {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = fmt.Sprintf("%s.bak.%d", path, i)
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return "", err
	}
	return backupPath, nil
}
