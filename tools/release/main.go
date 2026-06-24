/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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

type releaseOptions struct {
	Version       string
	Channel       string
	OutDir        string
	PrivateKeyEnv string
	GenerateKey   bool
	Platforms     []string
}

type multiFlag []string

var buildBinary = defaultBuildBinary

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
	opts := releaseOptions{Channel: "stable", PrivateKeyEnv: "CTYUN_RELEASE_PRIVATE_KEY"}
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.Version, "version", "", "release version")
	fs.StringVar(&opts.Channel, "channel", opts.Channel, "release channel")
	fs.StringVar(&opts.OutDir, "out", "", "output directory")
	fs.StringVar(&opts.PrivateKeyEnv, "private-key-env", opts.PrivateKeyEnv, "private key environment variable")
	fs.BoolVar(&opts.GenerateKey, "generate-key", false, "generate a release signing key")
	fs.Var(&platforms, "platform", "GOOS/GOARCH platform")
	if err := fs.Parse(args); err != nil {
		return releaseOptions{}, err
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
	if len(opts.Platforms) == 0 {
		return fmt.Errorf("at least one --platform GOOS/GOARCH is required")
	}
	return nil
}

// generateKey writes a new base64 Ed25519 signing key pair.
func generateKey(stdout io.Writer) error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
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

// privateKeyFromEnv decodes the base64 Ed25519 private key from envName.
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
	return key, nil
}

// writeRelease builds core and plugin archives, then writes signed indexes.
func writeRelease(opts releaseOptions, privateKey ed25519.PrivateKey) error {
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}
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
			return err
		}
		index.Releases[0].Artifacts = append(index.Releases[0].Artifacts, artifact)
	}
	index, err := mergeCoreIndex(opts.OutDir, index)
	if err != nil {
		return err
	}
	indexBytes, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	indexBytes = append(indexBytes, '\n')
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
	return writePluginRegistry(opts, privateKey)
}

// buildCoreArtifact builds and archives one platform-specific core artifact.
func buildCoreArtifact(opts releaseOptions, publicKeyBase64, platform string) (corerelease.Artifact, error) {
	goos, goarch, err := splitPlatform(platform)
	if err != nil {
		return corerelease.Artifact{}, err
	}
	binaryName := archiveBinaryName(goos)
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
	sum, size, err := checksumAndSize(archivePath)
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

// archiveBinaryName returns the binary name used inside a release archive.
func archiveBinaryName(goos string) string {
	if goos == "windows" {
		return "ctyun.exe"
	}
	return "ctyun"
}

// mergeCoreIndex preserves other release channels while replacing the current
// channel pointer rebuilt by this packaging run.
func mergeCoreIndex(outDir string, current corerelease.Index) (corerelease.Index, error) {
	indexPath := filepath.Join(outDir, "core-index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return current, nil
		}
		return corerelease.Index{}, err
	}
	existing, err := corerelease.LoadIndex(raw)
	if err != nil {
		return corerelease.Index{}, err
	}
	for _, release := range current.Releases {
		existing = upsertCoreRelease(existing, release)
	}
	return existing, nil
}

// upsertCoreRelease replaces one channel entry, merging platform artifacts only
// when rebuilding the same version.
func upsertCoreRelease(index corerelease.Index, release corerelease.Release) corerelease.Index {
	for i, existing := range index.Releases {
		if existing.Channel != release.Channel {
			continue
		}
		if existing.Version != release.Version {
			index.Releases[i] = release
			return index
		}
		index.Releases[i].PublishedAt = release.PublishedAt
		index.Releases[i].NotesURL = release.NotesURL
		index.Releases[i].Artifacts = mergeCoreArtifacts(existing.Artifacts, release.Artifacts)
		return index
	}
	index.Releases = append(index.Releases, release)
	return index
}

// mergeCoreArtifacts replaces artifacts for matching GOOS/GOARCH pairs.
func mergeCoreArtifacts(existing, current []corerelease.Artifact) []corerelease.Artifact {
	merged := slices.Clone(existing)
	for _, artifact := range current {
		replaced := false
		for i, candidate := range merged {
			if candidate.OS == artifact.OS && candidate.Arch == artifact.Arch {
				merged[i] = artifact
				replaced = true
				break
			}
		}
		if !replaced {
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
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", opts.Output, "./cmd/ctyun")
	cmd.Env = append(os.Environ(), "GOOS="+opts.GOOS, "GOARCH="+opts.GOARCH)
	return cmd.Run()
}

// writeArchive writes one platform archive with the binary and public docs.
func writeArchive(archivePath, binaryPath, binaryName string) error {
	return writeTarGzArchive(archivePath, func(tarWriter *tar.Writer) error {
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
	})
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
func writePluginRegistry(opts releaseOptions, privateKey ed25519.PrivateKey) error {
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
		archiveName := fmt.Sprintf("ctyun_plugin_%s_%s.tar.gz", bundle.Manifest.Name, bundle.Manifest.Version)
		archivePath := filepath.Join(opts.OutDir, archiveName)
		if err := writeDirectoryArchive(archivePath, bundlePath, bundle.Manifest.Name); err != nil {
			return err
		}
		sum, _, err := checksumAndSize(archivePath)
		if err != nil {
			return err
		}
		index.Plugins = append(index.Plugins, registry.Artifact{
			Name:        bundle.Manifest.Name,
			Product:     bundle.Manifest.API.Product,
			DisplayName: pluginDisplayName(bundle),
			Version:     bundle.Manifest.Version,
			Channel:     bundle.Manifest.Channel,
			Quality:     bundle.Manifest.Quality,
			URL:         archiveName,
			SHA256:      sum,
		})
	}
	index, err = mergeRegistryIndex(opts.OutDir, index)
	if err != nil {
		return err
	}
	indexBytes, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	indexBytes = append(indexBytes, '\n')
	if err := os.WriteFile(filepath.Join(opts.OutDir, "index.json"), indexBytes, 0o644); err != nil {
		return err
	}
	signature := ed25519.Sign(privateKey, indexBytes)
	return os.WriteFile(filepath.Join(opts.OutDir, "index.sig"), []byte(base64.StdEncoding.EncodeToString(signature)), 0o644)
}

// pluginDisplayName returns a stable English storefront name when plugin i18n
// metadata provides one.
func pluginDisplayName(bundle plugin.Bundle) string {
	for _, language := range []string{"en-US", "en-GB", "zh-CN"} {
		if texts := bundle.I18N[language]; texts != nil && texts["name"] != "" {
			return texts["name"]
		}
	}
	return bundle.Manifest.Name
}

// mergeRegistryIndex preserves other plugin channels while replacing the
// current plugin channel pointers rebuilt by this packaging run.
func mergeRegistryIndex(outDir string, current registry.Index) (registry.Index, error) {
	indexPath := filepath.Join(outDir, "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return current, nil
		}
		return registry.Index{}, err
	}
	existing, err := registry.LoadIndex(raw)
	if err != nil {
		return registry.Index{}, err
	}
	for _, artifact := range current.Plugins {
		existing = upsertRegistryArtifact(existing, artifact)
	}
	return existing, nil
}

// upsertRegistryArtifact replaces one plugin artifact with the same name and
// channel.
func upsertRegistryArtifact(index registry.Index, artifact registry.Artifact) registry.Index {
	for i, existing := range index.Plugins {
		if existing.Name == artifact.Name && existing.Channel == artifact.Channel {
			index.Plugins[i] = artifact
			return index
		}
	}
	index.Plugins = append(index.Plugins, artifact)
	return index
}

// writeDirectoryArchive writes a tar.gz archive containing rootName as the
// single top-level directory.
func writeDirectoryArchive(archivePath, rootPath, rootName string) error {
	return writeTarGzArchive(archivePath, func(tarWriter *tar.Writer) error {
		return filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}
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
	dir, err := os.Getwd()
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

// Set appends one repeated flag value.
func (m *multiFlag) Set(value string) error {
	if _, _, err := splitPlatform(value); err != nil {
		return err
	}
	*m = append(*m, value)
	return nil
}
