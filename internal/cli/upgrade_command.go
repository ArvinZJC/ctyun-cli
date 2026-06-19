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
func runUpgrade(stdout, _ io.Writer, args []string, getenv func(string) string, transport http.RoundTripper) error {
	opts, err := parseUpgradeOptions(args)
	if err != nil {
		return err
	}

	source, err := release.ResolveSource(release.SourceOptions{
		Requested:      opts.Source,
		CurrentVersion: version.Version,
		Getenv:         getenv,
	})
	if err != nil {
		return err
	}
	if source.Kind == release.SourceDevelopmentUnavailable {
		fmt.Fprintln(stdout, "self-upgrade is unavailable for development builds without an explicit release source")
		fmt.Fprintln(stdout, "use ctyun upgrade --check --source <path-or-url> to test signed release metadata")
		return nil
	}

	publicKey := getenv("CTYUN_RELEASE_PUBLIC_KEY")
	indexBytes, err := release.ReadSignedIndex(source.URL, publicKey, transport)
	if err != nil {
		return err
	}
	index, err := release.LoadIndex(indexBytes)
	if err != nil {
		return err
	}
	rel, artifact, ok := index.FindLatest(opts.Channel, runtime.GOOS, runtime.GOARCH)
	if !ok {
		return fmt.Errorf("no ctyun release found for %s/%s on channel %s", runtime.GOOS, runtime.GOARCH, upgradeChannel(opts.Channel))
	}
	if !release.VersionNewer(rel.Version, version.Version) {
		fmt.Fprintf(stdout, "ctyun %s is already up to date on channel %s\n", version.Version, upgradeChannel(opts.Channel))
		return nil
	}
	if !opts.Check {
		artifactPath, cleanup, err := release.PrepareArtifact(source.URL, artifact, transport)
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
		fmt.Fprintf(stdout, "upgraded %s %s -> %s\n", version.Name, version.Version, rel.Version)
		return nil
	}
	fmt.Fprintf(stdout, "ctyun %s is available from %s for %s/%s (%s)\n", rel.Version, source.Name, artifact.OS, artifact.Arch, artifact.URL)
	return nil
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
	if value == "" {
		return "stable"
	}
	return value
}

// upgradeBinaryName returns the binary name expected inside release archives.
func upgradeBinaryName(executable string) string {
	if runtime.GOOS == "windows" {
		return "ctyun.exe"
	}
	if base := filepath.Base(executable); base != "" {
		return base
	}
	return version.Name
}
