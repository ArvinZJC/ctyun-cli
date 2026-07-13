/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

// Package localdoctor performs read-only offline checks of local CLI state.
package localdoctor

import (
	"io/fs"
	"os"
	"runtime"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// CheckKey identifies one stable local diagnostic check.
type CheckKey string

const (
	// CheckConfigFile inspects the selected config path and content.
	CheckConfigFile CheckKey = "config_file"
	// CheckProfile inspects profile selection.
	CheckProfile CheckKey = "profile"
	// CheckCredentials inspects credential completeness and storage source.
	CheckCredentials CheckKey = "credentials"
	// CheckRegion inspects the resolved profile region.
	CheckRegion CheckKey = "region"
	// CheckEndpoint inspects the optional API endpoint override.
	CheckEndpoint CheckKey = "endpoint"
	// CheckPluginDirectory inspects the installed-plugin root.
	CheckPluginDirectory CheckKey = "plugin_directory"
	// CheckPlugin inspects one installed plugin bundle.
	CheckPlugin CheckKey = "plugin"
)

// FailureCategory identifies the local subsystem responsible for a finding.
type FailureCategory string

const (
	// FailureConfiguration identifies config parsing or selection problems.
	FailureConfiguration FailureCategory = "configuration"
	// FailurePermissions identifies observable unsafe filesystem permissions.
	FailurePermissions FailureCategory = "permissions"
	// FailureCredentials identifies missing or stored credentials.
	FailureCredentials FailureCategory = "credentials"
	// FailureEndpoint identifies an invalid endpoint override.
	FailureEndpoint FailureCategory = "endpoint"
	// FailureFilesystem identifies local path inspection problems.
	FailureFilesystem FailureCategory = "filesystem"
	// FailurePlugin identifies an invalid installed plugin bundle.
	FailurePlugin FailureCategory = "plugin"
)

// Finding records one safe, localizable local diagnostic result.
type Finding struct {
	Key        CheckKey
	Target     string
	Status     doctor.Status
	Category   FailureCategory
	DetailKey  string
	DetailArgs []any
	DependsOn  CheckKey
}

// Report contains deterministic findings, complete counts, and the resolved language.
type Report struct {
	Findings []Finding
	Counts   doctor.Counts
	Language string
}

// Input contains local paths and already-collected process config inputs.
type Input struct {
	ConfigPath        string
	ConfigPathSource  coreconfig.Source
	InjectedConfig    []byte
	UseInjectedConfig bool
	ProfileOption     string
	LanguageOption    string
	Getenv            func(string) string
	OSLocale          string
	PluginRoot        string
	CoreVersion       string
}

// Dependencies contains only read-only filesystem operations used by local checks.
type Dependencies struct {
	Stat       func(string) (fs.FileInfo, error)
	ReadFile   func(string) ([]byte, error)
	GOOS       string
	ReadDir    func(string) ([]os.DirEntry, error)
	LoadBundle func(string, string) (plugin.Bundle, error)
}

// DefaultDependencies returns production read-only filesystem dependencies.
func DefaultDependencies() Dependencies {
	return Dependencies{Stat: os.Stat, ReadFile: os.ReadFile, GOOS: runtime.GOOS, ReadDir: os.ReadDir, LoadBundle: plugin.LoadBundle}
}
