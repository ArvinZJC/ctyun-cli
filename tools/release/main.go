/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ArvinZJC/ctyun-cli/internal/dirlock"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

// buildOptions captures one platform-specific core binary build.
type buildOptions struct {
	Version          string
	Channel          string
	ReleasePublicKey string
	GOOS             string
	GOARCH           string
	Output           string
}

// releaseOptions holds packaging, signing, and mirror settings.
type releaseOptions struct {
	Version                 string
	Channel                 string
	OutDir                  string
	PrivateKeyEnv           string
	GiteePluginDownloadRoot string
	GenerateKey             bool
	Platforms               []string
}

// multiFlag collects repeated platform options.
type multiFlag []string

// buildBinary permits packaging tests to replace compilation with a fixture.
var buildBinary = defaultBuildBinary

// exitProcess is the process boundary used to verify main without terminating tests.
var exitProcess = os.Exit

// createSigningKey keeps key-provider failures observable independently of the
// operating system entropy source.
var createSigningKey = ed25519.GenerateKey

// encodeReleaseJSON is the serialization boundary for signed publication bytes.
var encodeReleaseJSON = indentedJSON

// checksumArchive binds completed archives to their index entries, with an I/O
// seam for failures occurring between archive creation and checksum reading.
var checksumArchive = checksumAndSize

// loadArchivePayload verifies published and candidate archive payloads before
// replacement; the seam permits deterministic candidate-read failure tests.
var loadArchivePayload = readArchivePayload

// workingDirectory exposes failures resolving the process directory before
// project-file discovery, independently of platform-specific Getwd caching.
var workingDirectory = os.Getwd

// main runs release packaging and reports failures to stderr.
func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		exitProcess(1)
	}
}

// run executes the release packaging tool.
func run(args []string, getenv func(string) string, stdout io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.GenerateKey {
		return generateKey(stdout)
	}
	if err := validateReleaseOptions(opts); err != nil {
		return err
	}
	privateKey, err := privateKeyFromEnv(getenv, opts.PrivateKeyEnv)
	if err != nil {
		return err
	}
	return writeRelease(opts, privateKey)
}

// parseOptions parses release tool command-line flags.
func parseOptions(args []string) (releaseOptions, error) {
	var platforms multiFlag
	opts := releaseOptions{Channel: "stable", PrivateKeyEnv: "CTYUN_RELEASE_PRIVATE_KEY", GiteePluginDownloadRoot: defaultGiteePluginDownloadRoot}
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.Version, "version", "", "release version")
	fs.StringVar(&opts.Channel, "channel", opts.Channel, "release channel")
	fs.StringVar(&opts.OutDir, "out", "", "output directory")
	fs.StringVar(&opts.PrivateKeyEnv, "private-key-env", opts.PrivateKeyEnv, "private key environment variable")
	fs.StringVar(&opts.GiteePluginDownloadRoot, "gitee-plugin-download-root", opts.GiteePluginDownloadRoot, "Gitee release download root")
	fs.BoolVar(&opts.GenerateKey, "generate-key", false, "generate a release signing key")
	fs.Var(&platforms, "platform", "GOOS/GOARCH platform")
	if err := fs.Parse(args); err != nil {
		return releaseOptions{}, err
	}
	if fs.NArg() != 0 {
		return releaseOptions{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	opts.Platforms = platforms
	return opts, nil
}

// validateReleaseOptions checks required packaging inputs.
func validateReleaseOptions(opts releaseOptions) error {
	if opts.Version == "" {
		return fmt.Errorf("--version is required")
	}
	if !coreversion.IsSemanticVersion(opts.Version) {
		return fmt.Errorf("--version must be a semantic version")
	}
	if opts.OutDir == "" {
		return fmt.Errorf("--out is required")
	}
	if opts.Channel != "stable" && opts.Channel != "beta" && opts.Channel != "alpha" {
		return fmt.Errorf("--channel must be stable, beta, or alpha")
	}
	if err := validateVersionChannel(opts.Version, opts.Channel); err != nil {
		return err
	}
	if len(opts.Platforms) == 0 {
		return fmt.Errorf("at least one --platform GOOS/GOARCH is required")
	}
	return validateGiteePluginDownloadRoot(opts.GiteePluginDownloadRoot)
}

// validateVersionChannel keeps stable releases and SemVer prereleases in their corresponding distribution channels.
func validateVersionChannel(version, channel string) error {
	precedence, _, _ := strings.Cut(version, "+")
	prerelease := strings.Contains(precedence, "-")
	if prerelease == (channel == "stable") {
		return fmt.Errorf("version %q does not match release channel %q", version, channel)
	}
	return nil
}

// generateKey writes a new base64 Ed25519 signing key pair.
func generateKey(stdout io.Writer) error {
	publicKey, privateKey, err := createSigningKey(rand.Reader)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Keep the private key secret; do not commit it."); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "CTYUN_RELEASE_PRIVATE_KEY=%s\n", base64.StdEncoding.EncodeToString(privateKey)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "CTYUN_RELEASE_PUBLIC_KEY=%s\n", base64.StdEncoding.EncodeToString(publicKey)); err != nil {
		return err
	}
	return nil
}

// privateKeyFromEnv decodes the Ed25519 private key and checks that its
// embedded public key matches the seed before packaging uses it.
func privateKeyFromEnv(getenv func(string) string, envName string) (ed25519.PrivateKey, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	raw := strings.TrimSpace(getenv(envName))
	if raw == "" {
		return nil, fmt.Errorf("%s is required", envName)
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", envName, err)
	}
	if len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("%s has length %d, want %d", envName, len(key), ed25519.PrivateKeySize)
	}
	derived := ed25519.NewKeyFromSeed(key[:ed25519.SeedSize])
	if !derived.Equal(ed25519.PrivateKey(key)) {
		return nil, fmt.Errorf("%s contains an inconsistent Ed25519 private key", envName)
	}
	return derived, nil
}

// writeRelease prepares a complete set of artifacts and signed metadata before replacing release files.
func writeRelease(opts releaseOptions, privateKey ed25519.PrivateKey) (err error) {
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}
	output := opts.OutDir
	stage, err := createReleaseStage(output, ".ctyun-release-*")
	if err != nil {
		return err
	}
	defer func() { err = closeWithError(err, func() error { return os.RemoveAll(stage) }) }()
	opts.OutDir = stage
	index, err := buildCoreRelease(opts, privateKey)
	if err != nil {
		return err
	}
	return dirlock.With(output, true, func() error {
		if err := writeReleaseMetadata(opts, privateKey, output, index); err != nil {
			return err
		}
		publicKey := base64.StdEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey))
		plan, err := planArchiveRetention(stage, output, publicKey)
		if err != nil {
			return err
		}
		return publishReleaseFiles(stage, output, plan)
	})
}

// buildCoreRelease compiles the independent platform archives before locking the shared publication directory.
func buildCoreRelease(opts releaseOptions, privateKey ed25519.PrivateKey) (corerelease.Index, error) {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)
	publishedAt := time.Now().UTC().Format(time.RFC3339)
	index := corerelease.Index{Schema: 1, Releases: []corerelease.Release{{
		Version:     opts.Version,
		Channel:     opts.Channel,
		PublishedAt: publishedAt,
		Artifacts:   make([]corerelease.Artifact, 0, len(opts.Platforms)),
	}}}
	for _, platform := range opts.Platforms {
		artifact, err := buildCoreArtifact(opts, publicKeyBase64, platform)
		if err != nil {
			return corerelease.Index{}, err
		}
		index.Releases[0].Artifacts = append(index.Releases[0].Artifacts, artifact)
	}
	return index, nil
}

// writeReleaseMetadata merges the current output indexes and reuses unchanged plugin archives under the publisher lock.
func writeReleaseMetadata(opts releaseOptions, privateKey ed25519.PrivateKey, previousDir string, index corerelease.Index) error {
	index, err := mergeCoreIndex(previousDir, index)
	if err != nil {
		return err
	}
	// Bootstrap installers take the first match for their channel and platform.
	slices.SortStableFunc(index.Releases, func(left, right corerelease.Release) int {
		return coreversion.CompareSemanticVersions(right.Version, left.Version)
	})
	indexBytes, err := encodeReleaseJSON(coreIndexDocument(index))
	if err != nil {
		return err
	}
	indexPath := filepath.Join(opts.OutDir, "core-index.json")
	if err := os.WriteFile(indexPath, indexBytes, 0o644); err != nil {
		return err
	}
	signature := ed25519.Sign(privateKey, indexBytes)
	if err := os.WriteFile(filepath.Join(opts.OutDir, "core-index.sig"), []byte(base64.StdEncoding.EncodeToString(signature)), 0o644); err != nil {
		return err
	}
	if err := copyInstallerScripts(opts.OutDir); err != nil {
		return err
	}
	if err := writePluginRegistry(opts, privateKey, previousDir); err != nil {
		return err
	}
	return writeGiteePluginRegistry(opts.OutDir, opts.GiteePluginDownloadRoot, privateKey)
}

// buildCoreArtifact builds and archives one platform-specific core artifact.
func buildCoreArtifact(opts releaseOptions, publicKeyBase64, platform string) (corerelease.Artifact, error) {
	goos, goarch, err := splitPlatform(platform)
	if err != nil {
		return corerelease.Artifact{}, err
	}
	binaryName := corerelease.ArchiveBinaryName(goos)
	buildDir, err := os.MkdirTemp("", "ctyun-release-build-*")
	if err != nil {
		return corerelease.Artifact{}, err
	}
	defer func() {
		_ = os.RemoveAll(buildDir)
	}()
	binaryPath := filepath.Join(buildDir, binaryName)
	if err := buildBinary(buildOptions{Version: opts.Version, Channel: opts.Channel, ReleasePublicKey: publicKeyBase64, GOOS: goos, GOARCH: goarch, Output: binaryPath}); err != nil {
		return corerelease.Artifact{}, err
	}
	archiveName := fmt.Sprintf("ctyun_%s_%s_%s.tar.gz", opts.Version, goos, goarch)
	archivePath := filepath.Join(opts.OutDir, archiveName)
	if err := writeArchive(archivePath, binaryPath, binaryName); err != nil {
		return corerelease.Artifact{}, err
	}
	sum, size, err := checksumArchive(archivePath)
	if err != nil {
		return corerelease.Artifact{}, err
	}
	return corerelease.Artifact{
		OS:     goos,
		Arch:   goarch,
		URL:    archiveName,
		SHA256: sum,
		Size:   size,
	}, nil
}

// mergeCoreIndex checks tag ownership before merging platform builds into existing channels.
func mergeCoreIndex(outDir string, current corerelease.Index) (corerelease.Index, error) {
	indexPath := filepath.Join(outDir, "core-index.json")
	existing := corerelease.Index{Schema: current.Schema}
	raw, err := os.ReadFile(indexPath)
	if err == nil {
		existing, err = corerelease.LoadIndex(raw)
	}
	if err != nil && !os.IsNotExist(err) {
		return corerelease.Index{}, err
	}
	if err := validateTagChannels(slices.Concat(existing.Releases, current.Releases), func(rel corerelease.Release) (string, string) {
		return "releases/core/" + rel.Version, rel.Channel
	}); err != nil {
		return corerelease.Index{}, err
	}
	for _, release := range current.Releases {
		existing = upsertCoreRelease(existing, release)
	}
	return existing, nil
}

// upsertCoreRelease advances each platform independently. An indexed historical
// version can be rebuilt without displacing newer builds on the same channel.
func upsertCoreRelease(index corerelease.Index, release corerelease.Release) corerelease.Index {
	slot := slices.IndexFunc(index.Releases, func(old corerelease.Release) bool {
		return old.Version == release.Version && old.Channel == release.Channel
	})
	if slot >= 0 {
		release.Artifacts = mergeCoreArtifacts(index.Releases[slot].Artifacts, release.Artifacts)
		index.Releases[slot] = release
	} else {
		// An out-of-order first build can supply targets with no newer release.
		for _, old := range index.Releases {
			if old.Channel == release.Channel && coreversion.CompareSemanticVersions(old.Version, release.Version) > 0 {
				release.Artifacts = withoutCorePlatforms(release.Artifacts, old.Artifacts)
			}
		}
		if len(release.Artifacts) == 0 {
			return index
		}
		// Put newly published build metadata ahead of equally ranked alternatives.
		slot = slices.IndexFunc(index.Releases, func(old corerelease.Release) bool { return old.Channel == release.Channel })
		if slot < 0 {
			slot = len(index.Releases)
		}
		index.Releases = slices.Insert(index.Releases, slot, release)
	}
	for i, old := range index.Releases {
		if i != slot && old.Channel == release.Channel && coreversion.CompareSemanticVersions(old.Version, release.Version) <= 0 {
			index.Releases[i].Artifacts = withoutCorePlatforms(old.Artifacts, release.Artifacts)
		}
	}
	index.Releases = slices.DeleteFunc(index.Releases, func(old corerelease.Release) bool {
		return old.Channel == release.Channel && len(old.Artifacts) == 0
	})
	return index
}

// withoutCorePlatforms excludes targets already supplied by another release.
func withoutCorePlatforms(artifacts, supplied []corerelease.Artifact) []corerelease.Artifact {
	return slices.DeleteFunc(artifacts, func(artifact corerelease.Artifact) bool {
		return slices.ContainsFunc(supplied, func(other corerelease.Artifact) bool {
			return artifact.OS == other.OS && artifact.Arch == other.Arch
		})
	})
}

// mergeCoreArtifacts keeps the first indexed entry for each retained platform
// and replaces rebuilt platforms in place, preserving platform order.
func mergeCoreArtifacts(existing, current []corerelease.Artifact) []corerelease.Artifact {
	merged := make([]corerelease.Artifact, 0, len(existing)+len(current))
	positions := make(map[[2]string]int)
	for _, artifact := range existing {
		key := [2]string{artifact.OS, artifact.Arch}
		if _, exists := positions[key]; !exists {
			positions[key] = len(merged)
			merged = append(merged, artifact)
		}
	}
	for _, artifact := range current {
		key := [2]string{artifact.OS, artifact.Arch}
		if index, exists := positions[key]; exists {
			merged[index] = artifact
		} else {
			positions[key] = len(merged)
			merged = append(merged, artifact)
		}
	}
	return merged
}

// splitPlatform parses GOOS/GOARCH.
func splitPlatform(platform string) (string, string, error) {
	goos, goarch, ok := strings.Cut(platform, "/")
	if !ok || goos == "" || goarch == "" {
		return "", "", fmt.Errorf("platform %q must be GOOS/GOARCH", platform)
	}
	return goos, goarch, nil
}

// defaultBuildBinary builds the ctyun command for one target platform.
func defaultBuildBinary(opts buildOptions) error {
	ldflags := strings.Join([]string{
		"-X github.com/ArvinZJC/ctyun-cli/internal/version.Version=" + opts.Version,
		"-X github.com/ArvinZJC/ctyun-cli/internal/version.Channel=" + opts.Channel,
		"-X github.com/ArvinZJC/ctyun-cli/internal/version.ReleasePublicKey=" + opts.ReleasePublicKey,
	}, " ")
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", opts.Output, "./cmd/ctyun")
	cmd.Env = append(os.Environ(), "GOOS="+opts.GOOS, "GOARCH="+opts.GOARCH)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build ctyun for %s/%s: %w%s", opts.GOOS, opts.GOARCH, err, strings.TrimRight("\n"+string(output), "\r\n"))
	}
	return nil
}

// writeArchive writes one platform archive with the binary and public docs.
func writeArchive(archivePath, binaryPath, binaryName string) (err error) {
	stage, err := os.MkdirTemp(filepath.Dir(archivePath), ".ctyun-core-*")
	if err != nil {
		return err
	}
	defer func() { err = closeWithError(err, func() error { return os.RemoveAll(stage) }) }()
	candidate := filepath.Join(stage, "core.tar.gz")
	if err := writeTarGzArchive(candidate, func(tarWriter *tar.Writer) error {
		if err := addFileToArchive(tarWriter, binaryPath, binaryName, 0o755); err != nil {
			return err
		}
		for _, name := range []string{"README.md", "README-EN.md", "LICENCE"} {
			if path, err := projectFilePath(name); err == nil {
				if err := addFileToArchive(tarWriter, path, name, 0o644); err != nil {
					return err
				}
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return os.Rename(candidate, archivePath)
}

// writeTarGzArchive creates archivePath and passes its tar writer to write.
func writeTarGzArchive(archivePath string, write func(*tar.Writer) error) (err error) {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer func() {
		err = closeWithError(err, file.Close)
	}()
	gzipWriter := gzip.NewWriter(file)
	defer func() {
		err = closeWithError(err, gzipWriter.Close)
	}()
	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		err = closeWithError(err, tarWriter.Close)
	}()
	return write(tarWriter)
}

// writePluginRegistry writes bundled plugin archives and a signed registry
// index.
func writePluginRegistry(opts releaseOptions, privateKey ed25519.PrivateKey, previousDir string) error {
	pluginsRoot, err := projectFilePath("plugins")
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(pluginsRoot)
	if err != nil {
		return err
	}
	index := registry.Index{Plugins: make([]registry.Artifact, 0, len(entries))}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bundlePath := filepath.Join(pluginsRoot, entry.Name())
		bundle, err := plugin.LoadBundle(bundlePath, opts.Version)
		if err != nil {
			return err
		}
		if err := validateVersionChannel(bundle.Manifest.Version, bundle.Manifest.Channel); err != nil {
			return fmt.Errorf("plugin %q: %w", bundle.Manifest.Name, err)
		}
		archiveName := fmt.Sprintf("ctyun_plugin_%s_%s.tar.gz", bundle.Manifest.Name, bundle.Manifest.Version)
		archivePath := filepath.Join(opts.OutDir, archiveName)
		chosen, err := writeDirectoryArchive(archivePath, bundlePath, bundle.Manifest.Name, filepath.Join(previousDir, archiveName))
		if err != nil {
			return err
		}
		sum, _, err := checksumArchive(chosen)
		if err != nil {
			return err
		}
		artifact := registry.ArtifactFromBundle(bundle)
		artifact.URL = archiveName
		artifact.SHA256 = sum
		index.Plugins = append(index.Plugins, artifact)
	}
	index, err = mergeRegistryIndex(previousDir, index)
	if err != nil {
		return err
	}
	indexBytes, err := encodeReleaseJSON(registryIndexDocument(index))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opts.OutDir, "index.json"), indexBytes, 0o644); err != nil {
		return err
	}
	signature := ed25519.Sign(privateKey, indexBytes)
	return os.WriteFile(filepath.Join(opts.OutDir, "index.sig"), []byte(base64.StdEncoding.EncodeToString(signature)), 0o644)
}

// mergeRegistryIndex checks tag ownership before refreshing the selected plugin channel entries.
func mergeRegistryIndex(outDir string, current registry.Index) (registry.Index, error) {
	indexPath := filepath.Join(outDir, "index.json")
	existing := registry.Index{Plugins: []registry.Artifact{}}
	raw, err := os.ReadFile(indexPath)
	if err == nil {
		existing, err = registry.LoadIndex(raw)
	}
	if err != nil && !os.IsNotExist(err) {
		return registry.Index{}, err
	}
	if err := validateTagChannels(slices.Concat(existing.Plugins, current.Plugins), func(artifact registry.Artifact) (string, string) {
		return pluginReleaseTag(artifact), artifact.Channel
	}); err != nil {
		return registry.Index{}, err
	}
	for _, artifact := range current.Plugins {
		existing = upsertRegistryArtifact(existing, artifact)
	}
	return existing, nil
}

// upsertRegistryArtifact refreshes a known plugin version or advances its channel pointer for a new version.
func upsertRegistryArtifact(index registry.Index, artifact registry.Artifact) registry.Index {
	channelSlot := -1
	for i, existing := range index.Plugins {
		if existing.Name != artifact.Name || existing.Channel != artifact.Channel {
			continue
		}
		if channelSlot == -1 || coreversion.CompareSemanticVersions(existing.Version, index.Plugins[channelSlot].Version) > 0 {
			channelSlot = i
		}
		if existing.Version == artifact.Version {
			channelSlot = i
			break
		}
	}
	if channelSlot == -1 {
		index.Plugins = append(index.Plugins, artifact)
	} else if coreversion.CompareSemanticVersions(artifact.Version, index.Plugins[channelSlot].Version) >= 0 {
		index.Plugins[channelSlot] = artifact
	}
	return index
}

// writeDirectoryArchive preserves existing compressed bytes when the generated
// tar payload is unchanged. It returns the actual file to hash, either the retained or staged archive.
func writeDirectoryArchive(archivePath, rootPath, rootName, previousPath string) (chosen string, err error) {
	temporaryDir, err := os.MkdirTemp(filepath.Dir(archivePath), ".ctyun-plugin-*")
	if err != nil {
		return "", err
	}
	defer func() {
		err = closeWithError(err, func() error { return os.RemoveAll(temporaryDir) })
	}()
	candidate := filepath.Join(temporaryDir, "plugin.tar.gz")
	if err := writeDirectoryArchiveFile(candidate, rootPath, rootName); err != nil {
		return "", err
	}
	previous, err := loadArchivePayload(previousPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err == nil {
		generated, err := loadArchivePayload(candidate)
		if err != nil {
			return "", err
		}
		if bytes.Equal(previous, generated) {
			return previousPath, nil
		}
	}
	return archivePath, os.Rename(candidate, archivePath)
}

// readArchivePayload decodes an archive without extracting its entries.
func readArchivePayload(path string) (payload []byte, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		err = closeWithError(err, reader.Close)
	}()
	return io.ReadAll(reader)
}

// writeDirectoryArchiveFile writes a plugin directory as a .tar.gz archive with
// a single top-level directory.
func writeDirectoryArchiveFile(archivePath, rootPath, rootName string) error {
	return writeTarGzArchive(archivePath, func(tarWriter *tar.Writer) error {
		return filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// WalkDir supplies rootPath itself or its lexical descendants, so
			// Rel cannot cross volumes or mix absolute and relative paths.
			rel, _ := filepath.Rel(rootPath, path)
			name := rootName
			if rel != "." {
				name = filepath.ToSlash(filepath.Join(rootName, rel))
			}
			if entry.IsDir() {
				header := &tar.Header{Name: name + "/", Mode: 0o755, Typeflag: tar.TypeDir}
				return tarWriter.WriteHeader(header)
			}
			if entry.Type() != 0 {
				return fmt.Errorf("unsupported plugin archive entry %s", rel)
			}
			return addFileToArchive(tarWriter, path, name, 0o644)
		})
	})
}

// addFileToArchive appends one regular file to a tar archive.
func addFileToArchive(tarWriter *tar.Writer, path, name string, mode int64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	header := &tar.Header{Name: name, Mode: mode, Size: int64(len(data)), Typeflag: tar.TypeReg}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err = tarWriter.Write(data)
	return err
}

// copyInstallerScripts copies first-install bootstrap scripts into outDir.
func copyInstallerScripts(outDir string) error {
	for _, name := range []string{"install.sh", "install.ps1"} {
		path, err := projectFilePath(name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(name, ".sh") {
			mode = 0o755
		}
		if err := os.WriteFile(filepath.Join(outDir, name), data, mode); err != nil {
			return err
		}
	}
	return nil
}

// projectFilePath finds a repository file from the current directory or a parent.
func projectFilePath(name string) (string, error) {
	dir, err := workingDirectory()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// checksumAndSize returns the SHA-256 digest and size of path.
func checksumAndSize(path string) (sum string, size int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		err = closeWithError(err, file.Close)
	}()
	hash := sha256.New()
	size, err = io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

// closeWithError reports close failures without discarding the primary error.
func closeWithError(err error, close func() error) error {
	if closeErr := close(); closeErr != nil {
		if err != nil {
			return errors.Join(err, closeErr)
		}
		return closeErr
	}
	return err
}

// String returns a comma-separated representation for flag help.
func (m *multiFlag) String() string {
	if m == nil {
		return ""
	}
	return strings.Join(*m, ",")
}

// Set schedules each valid build target once in first-requested order.
func (m *multiFlag) Set(value string) error {
	if _, _, err := splitPlatform(value); err != nil {
		return err
	}
	if !slices.Contains(*m, value) {
		*m = append(*m, value)
	}
	return nil
}
