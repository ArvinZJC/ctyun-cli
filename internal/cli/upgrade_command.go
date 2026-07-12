/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"io"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/ArvinZJC/ctyun-cli/internal/diagnostic"
	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/release"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// upgradeOptions captures core self-upgrade command flags.
type upgradeOptions struct {
	Check   bool
	Source  string
	Channel string
}

// runUpgrade checks or applies updates for the core ctyun binary.
func runUpgrade(stdout, stderr io.Writer, args []string, getenv func(string) string, transport http.RoundTripper, language string, currentExecutable func() (string, error)) error {
	opts, err := parseUpgradeOptions(args)
	if err != nil {
		return err
	}

	source, err := release.ResolveSource(release.SourceOptions{
		Requested:        opts.Source,
		DevelopmentBuild: version.IsDevelopmentBuild(),
		Getenv:           getenv,
	})
	if err != nil {
		return err
	}
	if source.Kind == release.SourceDevelopmentUnavailable {
		for _, message := range upgradeDevelopmentMessages(language) {
			if err := writeLine(stdout, message); err != nil {
				return err
			}
		}
		return nil
	}

	publicKey := releasePublicKey(getenv)
	selectedSource, indexBytes, err := readUpgradeIndex(source, publicKey, transport)
	if err != nil {
		return err
	}
	index, err := release.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	channel := upgradeChannel(opts.Channel)
	rel, artifact, ok := index.FindLatest(channel, runtime.GOOS, runtime.GOARCH)
	if !ok {
		return diagnostic.New("error.release_not_found", runtime.GOOS, runtime.GOARCH, channel)
	}
	if !release.VersionNewer(rel.Version, version.Version) {
		return writeLine(stdout, upgradeCurrentMessage(language, version.Version, channel))
	}
	if !opts.Check {
		display := operationProgressFactory(stderr)
		if err := display.Start(3); err != nil {
			return err
		}
		result := operationResult{
			Target:     version.Name,
			Outcome:    operationChanged,
			OldVersion: version.Version,
			NewVersion: rel.Version,
		}
		if err := updateOperationProgress(display, 0, operationProgressLabel(language, "download", version.Name)); err != nil {
			return err
		}
		artifactPath, cleanup, err := release.PrepareArtifact(selectedSource.URL, artifact, transport)
		if err != nil {
			return reportCoreUpgradeFailure(stdout, stderr, display, result, err, language)
		}
		defer cleanup()
		if err := updateOperationProgress(display, 1, operationProgressLabel(language, "verify", version.Name)); err != nil {
			return err
		}
		if err := distribution.VerifySHA256(artifactPath, artifact.SHA256); err != nil {
			return reportCoreUpgradeFailure(stdout, stderr, display, result, err, language)
		}
		if err := updateOperationProgress(display, 2, operationProgressLabel(language, "core_install", version.Name)); err != nil {
			return err
		}
		executable, err := currentExecutable()
		if err != nil {
			return reportCoreUpgradeFailure(stdout, stderr, display, result, err, language)
		}
		if err := release.InstallArtifact(release.InstallOptions{
			CurrentExecutable: executable,
			ArchivePath:       artifactPath,
			BinaryName:        upgradeBinaryName(executable),
		}); err != nil {
			return reportCoreUpgradeFailure(stdout, stderr, display, result, err, language)
		}
		if err := updateOperationProgress(display, 3, operationProgressLabel(language, "core_install", version.Name)); err != nil {
			return err
		}
		if err := display.Clear(); err != nil {
			return err
		}
		return reportOperationResults(stdout, stderr, []operationResult{result}, language, coreUpgradeSummary)
	}
	return writeLine(stdout, upgradeAvailableMessage(language, rel.Version, selectedSource.Name, artifact.OS, artifact.Arch, artifact.URL))
}

// updateOperationProgress clears the display before propagating an update
// write failure.
func updateOperationProgress(display operationDisplay, completed int, label string) error {
	if err := display.Update(completed, label); err != nil {
		_ = display.Clear()
		return err
	}
	return nil
}

// reportCoreUpgradeFailure clears progress and reports one structured failed
// core update result.
func reportCoreUpgradeFailure(stdout, stderr io.Writer, display operationDisplay, result operationResult, cause error, language string) error {
	_ = display.Clear()
	result.Outcome = operationFailed
	result.Err = cause
	return reportOperationResults(stdout, stderr, []operationResult{result}, language, coreUpgradeSummary)
}

// releasePublicKey applies release public-key precedence.
func releasePublicKey(getenv func(string) string) string {
	if value := getenv("CTYUN_RELEASE_PUBLIC_KEY"); value != "" {
		return value
	}
	return version.ReleasePublicKey
}

// readUpgradeIndex reads the primary source and then any mirror fallbacks.
func readUpgradeIndex(source release.Source, publicKey string, transport http.RoundTripper) (release.Source, []byte, error) {
	sources := append([]release.Source{source}, source.Fallbacks...)
	var lastErr error
	for _, candidate := range sources {
		indexBytes, err := release.ReadSignedIndex(candidate.URL, publicKey, transport)
		if err == nil {
			return candidate, indexBytes, nil
		}
		lastErr = err
	}
	return release.Source{}, nil, lastErr
}

// parseUpgradeOptions parses core upgrade arguments.
func parseUpgradeOptions(args []string) (upgradeOptions, error) {
	var opts upgradeOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--check":
			opts.Check = true
		case "--source":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.source_requires_value")
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, diagnostic.New("error.channel_requires_value")
			}
			opts.Channel = args[i]
		default:
			return opts, diagnostic.New("error.upgrade_option", args[i])
		}
	}
	if opts.Source != "" && opts.Source != "auto" && opts.Source != "github" && opts.Source != "gitee" {
		return opts, diagnostic.New("error.unsupported_source", "core", opts.Source)
	}
	if opts.Channel != "" && opts.Channel != "stable" && opts.Channel != "beta" && opts.Channel != "alpha" {
		return opts, diagnostic.New("error.unsupported_channel", opts.Channel)
	}
	return opts, nil
}

// upgradeChannel returns the default core release channel when value is empty.
func upgradeChannel(value string) string {
	if value != "" {
		return value
	}
	if version.Channel == "stable" || version.Channel == "beta" || version.Channel == "alpha" {
		return version.Channel
	}
	return "stable"
}

// upgradeBinaryName returns the binary name expected inside release archives.
func upgradeBinaryName(executable string) string {
	return upgradeBinaryNameForGOOS(executable, runtime.GOOS)
}

// upgradeBinaryNameForGOOS returns the release archive binary name for goos.
func upgradeBinaryNameForGOOS(executable, goos string) string {
	if goos == "windows" {
		return "ctyun.exe"
	}
	return filepath.Base(executable)
}
