/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ArvinZJC/ctyun-cli/internal/release"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// upgradeOptions captures core self-upgrade command flags.
type upgradeOptions struct {
	Check   bool
	Source  string
	Channel string
}

var currentExecutable = os.Executable

// runUpgrade checks or applies updates for the core ctyun binary.
func runUpgrade(stdout, _ io.Writer, args []string, getenv func(string) string, transport http.RoundTripper, language string) error {
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
			fmt.Fprintln(stdout, message)
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
		return fmt.Errorf("no ctyun release found for %s/%s on channel %s", runtime.GOOS, runtime.GOARCH, channel)
	}
	if !release.VersionNewer(rel.Version, version.Version) {
		fmt.Fprintln(stdout, upgradeCurrentMessage(language, version.Version, channel))
		return nil
	}
	if !opts.Check {
		artifactPath, cleanup, err := release.PrepareArtifact(selectedSource.URL, artifact, transport)
		if err != nil {
			return err
		}
		defer cleanup()
		if err := release.VerifySHA256(artifactPath, artifact.SHA256); err != nil {
			return err
		}
		executable, err := currentExecutable()
		if err != nil {
			return err
		}
		if err := release.InstallArtifact(release.InstallOptions{
			CurrentExecutable: executable,
			ArchivePath:       artifactPath,
			BinaryName:        upgradeBinaryName(executable),
		}); err != nil {
			return err
		}
		fmt.Fprintln(stdout, upgradeInstalledMessage(language, version.Name, version.Version, rel.Version))
		return nil
	}
	fmt.Fprintln(stdout, upgradeAvailableMessage(language, rel.Version, selectedSource.Name, artifact.OS, artifact.Arch, artifact.URL))
	return nil
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
				return opts, fmt.Errorf("--source requires a value")
			}
			opts.Source = args[i]
		case "--channel":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--channel requires a value")
			}
			opts.Channel = args[i]
		default:
			return opts, fmt.Errorf("unknown upgrade option %q", args[i])
		}
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
	if runtime.GOOS == "windows" {
		return "ctyun.exe"
	}
	return filepath.Base(executable)
}
