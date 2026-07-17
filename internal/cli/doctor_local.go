/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"strings"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/localdoctor"
)

// doctorLocalDependenciesFactory creates read-only production dependencies.
var doctorLocalDependenciesFactory = localdoctor.DefaultDependencies

// doctorLocalHelpLanguage resolves help language without requiring valid local state.
func doctorLocalHelpLanguage(cfg Config, opts globalOptions, path string, pathSource coreconfig.Source, getenv func(string) string) string {
	raw, err := loadConfigBytes(cfg.Config, path)
	if err != nil {
		raw = nil
	}
	inspection := coreconfig.Inspect(coreconfig.ResolveInput{
		Raw: raw, ConfigPath: path, ConfigPathSource: pathSource,
		ProfileOption: opts.Profile, LanguageOption: opts.Language,
		Getenv: withoutCredentials(getenv), OSLocale: detectOSLocale(getenv),
	})
	language, _ := inspection.Resolution.Setting("language")
	return language.Value
}

// withoutCredentials prevents help-only resolution from reading AK/SK values.
func withoutCredentials(getenv func(string) string) func(string) string {
	return func(key string) string {
		if key == "CTYUN_AK" || key == "CTYUN_SK" {
			return ""
		}
		return getenv(key)
	}
}

// isDoctorLocalCommand reports whether args select the local diagnostic leaf.
func isDoctorLocalCommand(args []string) bool {
	return len(args) >= 2 && args[0] == "doctor" && args[1] == "local"
}

// isDoctorLocalHelpTarget reports whether args use the explicit help command.
func isDoctorLocalHelpTarget(args []string) bool {
	return len(args) >= 3 && args[0] == "help" && args[1] == "doctor" && args[2] == "local"
}

// runDoctorLocalCommand validates the leaf boundary before running diagnostics.
func runDoctorLocalCommand(stdout io.Writer, args []string, input localdoctor.Input, opts globalOptions) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return diagnostic.New("error.unknown_option", arg)
		}
	}
	if err := validatePositionalArguments(args, nil, 0, 0); err != nil {
		return err
	}
	return runDoctorLocal(stdout, input, doctorLocalDependenciesFactory(), opts)
}

// runDoctorLocal executes and renders the complete offline diagnostic report.
func runDoctorLocal(stdout io.Writer, input localdoctor.Input, dependencies localdoctor.Dependencies, opts globalOptions) error {
	report := localdoctor.Run(input, dependencies)
	opts.Language = report.Language
	if err := renderDoctorLocalReport(stdout, report, opts); err != nil {
		return err
	}
	if report.Counts.HasFailures() {
		return silentExitError{}
	}
	return nil
}
