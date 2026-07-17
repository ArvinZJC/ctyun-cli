/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package localdoctor

import (
	"errors"
	"io/fs"
	"net/url"
	"strings"

	coreconfig "github.com/ArvinZJC/ctyun-cli/internal/config"
	"github.com/ArvinZJC/ctyun-cli/internal/doctor"
)

// configEvaluation carries one file finding and best-effort parsed context.
type configEvaluation struct {
	inspection coreconfig.Inspection
	file       Finding
	blocked    bool
}

// evaluateConfigFile reads the selected config once and classifies its file state.
func evaluateConfigFile(input Input, dependencies Dependencies) configEvaluation {
	target := input.ConfigPath
	if input.UseInjectedConfig {
		inspection := inspectConfig(input, input.InjectedConfig)
		finding := Finding{Key: CheckConfigFile, Target: target, Status: doctor.StatusPassed, DetailKey: "doctor.local.detail.config_injected"}
		if inspection.ConfigError != nil {
			finding.Status = doctor.StatusFailed
			finding.Category = FailureConfiguration
			finding.DetailKey = "doctor.local.detail.config_invalid"
		}
		return configEvaluation{inspection: inspection, file: finding}
	}
	if target == "" {
		return configEvaluation{
			inspection: inspectConfig(input, nil),
			file:       Finding{Key: CheckConfigFile, Status: doctor.StatusFailed, Category: FailureFilesystem, DetailKey: "doctor.local.detail.config_path_unavailable"},
			blocked:    true,
		}
	}
	info, err := dependencies.Stat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return configEvaluation{
			inspection: inspectConfig(input, nil),
			file:       Finding{Key: CheckConfigFile, Target: target, Status: doctor.StatusSkipped, DetailKey: "doctor.local.detail.config_missing"},
		}
	}
	if err != nil {
		return blockedConfigEvaluation(input, target, "doctor.local.detail.config_unreadable")
	}
	if !info.Mode().IsRegular() {
		return blockedConfigEvaluation(input, target, "doctor.local.detail.config_not_regular")
	}
	raw, err := dependencies.ReadFile(target)
	if err != nil {
		return blockedConfigEvaluation(input, target, "doctor.local.detail.config_unreadable")
	}
	inspection := inspectConfig(input, raw)
	finding := Finding{Key: CheckConfigFile, Target: target, Status: doctor.StatusPassed, DetailKey: "doctor.local.detail.config_passed"}
	if inspection.ConfigError != nil {
		finding.Status = doctor.StatusFailed
		finding.Category = FailureConfiguration
		finding.DetailKey = "doctor.local.detail.config_invalid"
		return configEvaluation{inspection: inspection, file: finding}
	}
	if inspection.HasStoredCredentials() && isPOSIXGOOS(dependencies.GOOS) && info.Mode().Perm()&0o077 != 0 {
		finding.Status = doctor.StatusWarning
		finding.Category = FailurePermissions
		finding.DetailKey = "doctor.local.detail.config_permissions"
	}
	return configEvaluation{inspection: inspection, file: finding}
}

// blockedConfigEvaluation returns a file failure whose content cannot be inspected.
func blockedConfigEvaluation(input Input, target, detailKey string) configEvaluation {
	return configEvaluation{
		inspection: inspectConfig(input, nil),
		file:       Finding{Key: CheckConfigFile, Target: target, Status: doctor.StatusFailed, Category: FailureFilesystem, DetailKey: detailKey},
		blocked:    true,
	}
}

// inspectConfig builds the shared best-effort config resolution input.
func inspectConfig(input Input, raw []byte) coreconfig.Inspection {
	return coreconfig.Inspect(coreconfig.ResolveInput{
		Raw: raw, ConfigPath: input.ConfigPath, ConfigPathSource: input.ConfigPathSource,
		ProfileOption: input.ProfileOption, LanguageOption: input.LanguageOption,
		Getenv: input.Getenv, OSLocale: input.OSLocale,
	})
}

// configFindings derives profile, credential, region, and endpoint findings.
func configFindings(evaluation configEvaluation) []Finding {
	inspection := evaluation.inspection
	findings := []Finding{evaluation.file}
	profileBlocked := evaluation.blocked || inspection.ConfigError != nil
	profileFinding := Finding{Key: CheckProfile, Target: inspection.Resolution.ProfileName()}
	switch {
	case profileBlocked:
		profileFinding.Status = doctor.StatusSkipped
		profileFinding.DependsOn = CheckConfigFile
		profileFinding.DetailKey = "doctor.local.detail.dependency"
	case inspection.ProfileError != nil:
		profileFinding.Status = doctor.StatusFailed
		profileFinding.Category = FailureConfiguration
		profileFinding.DetailKey = "doctor.local.detail.profile_invalid"
	case inspection.Resolution.ProfileName() == "":
		profileFinding.Status = doctor.StatusSkipped
		profileFinding.DetailKey = "doctor.local.detail.profile_missing"
	default:
		profileFinding.Status = doctor.StatusPassed
		profileFinding.DetailKey = "doctor.local.detail.profile_passed"
	}
	findings = append(findings, profileFinding)
	findings = append(findings, credentialFinding(inspection, profileBlocked))
	findings = append(findings, regionFinding(inspection, profileBlocked))
	findings = append(findings, endpointFinding(inspection, profileBlocked))
	return findings
}

// credentialFinding evaluates only completeness and safe source kinds.
func credentialFinding(inspection coreconfig.Inspection, profileBlocked bool) Finding {
	finding := Finding{Key: CheckCredentials, Category: FailureCredentials}
	ak, _ := inspection.Resolution.Setting("ak")
	sk, _ := inspection.Resolution.Setting("sk")
	if profileBlocked || inspection.ProfileError != nil {
		if ak.Configured && sk.Configured && ak.Source.Kind == coreconfig.SourceEnvironment && sk.Source.Kind == coreconfig.SourceEnvironment {
			finding.Status = doctor.StatusPassed
			finding.DetailKey = "doctor.local.detail.credentials_environment"
			return finding
		}
		finding.Status = doctor.StatusSkipped
		finding.DependsOn = CheckProfile
		finding.DetailKey = "doctor.local.detail.dependency"
		return finding
	}
	if !ak.Configured || !sk.Configured {
		finding.Status = doctor.StatusFailed
		finding.DetailKey = "doctor.local.detail.credentials_missing"
		return finding
	}
	akStored := credentialStored(ak)
	skStored := credentialStored(sk)
	if akStored && skStored && ak.Source.Kind == sk.Source.Kind {
		finding.Status = doctor.StatusWarning
		finding.DetailKey = "doctor.local.detail.credentials_stored"
		finding.DetailArgs = []any{string(ak.Source.Kind)}
		return finding
	}
	if akStored || skStored {
		finding.Status = doctor.StatusWarning
		finding.DetailKey = "doctor.local.detail.credentials_mixed"
		finding.DetailArgs = []any{string(ak.Source.Kind), string(sk.Source.Kind)}
		return finding
	}
	finding.Status = doctor.StatusPassed
	finding.DetailKey = "doctor.local.detail.credentials_environment"
	return finding
}

// credentialStored reports whether one credential resolved from persistent config.
func credentialStored(setting coreconfig.Setting) bool {
	return setting.Source.Kind == coreconfig.SourceProfile || setting.Source.Kind == coreconfig.SourceConfig
}

// regionFinding evaluates whether the selected base context provides a region.
func regionFinding(inspection coreconfig.Inspection, profileBlocked bool) Finding {
	finding := Finding{Key: CheckRegion}
	if profileBlocked || inspection.ProfileError != nil {
		finding.Status = doctor.StatusSkipped
		finding.DependsOn = CheckProfile
		finding.DetailKey = "doctor.local.detail.dependency"
		return finding
	}
	setting, _ := inspection.Resolution.Setting("region")
	finding.Target = setting.Value
	if !setting.Configured {
		finding.Status = doctor.StatusWarning
		finding.Category = FailureConfiguration
		finding.DetailKey = "doctor.local.detail.region_missing"
		return finding
	}
	finding.Status = doctor.StatusPassed
	finding.DetailKey = "doctor.local.detail.region_passed"
	return finding
}

// endpointFinding validates only the syntax of an optional endpoint override.
func endpointFinding(inspection coreconfig.Inspection, profileBlocked bool) Finding {
	finding := Finding{Key: CheckEndpoint}
	if profileBlocked || inspection.ProfileError != nil {
		finding.Status = doctor.StatusSkipped
		finding.DependsOn = CheckProfile
		finding.DetailKey = "doctor.local.detail.dependency"
		return finding
	}
	setting, _ := inspection.Resolution.Setting("endpoint_url")
	finding.Target = setting.Value
	if !setting.Configured {
		finding.Status = doctor.StatusSkipped
		finding.DetailKey = "doctor.local.detail.endpoint_missing"
		return finding
	}
	parsed, err := url.Parse(setting.Value)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
		finding.Status = doctor.StatusFailed
		finding.Category = FailureEndpoint
		finding.DetailKey = "doctor.local.detail.endpoint_invalid"
		return finding
	}
	finding.Status = doctor.StatusPassed
	finding.DetailKey = "doctor.local.detail.endpoint_passed"
	return finding
}

// isPOSIXGOOS reports platforms where Go permission bits describe POSIX access.
func isPOSIXGOOS(goos string) bool {
	switch goos {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "illumos", "ios", "linux", "netbsd", "openbsd", "solaris":
		return true
	default:
		return false
	}
}
