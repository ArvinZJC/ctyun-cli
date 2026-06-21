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
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
		fmt.Fprintln(os.Stderr, err)
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
	if opts.Channel != "stable" && opts.Channel != "beta" && opts.Channel != "edge" {
		return fmt.Errorf("--channel must be stable, beta, or edge")
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
	fmt.Fprintln(stdout, "Keep the private key secret; do not commit it.")
	fmt.Fprintf(stdout, "CTYUN_RELEASE_PRIVATE_KEY=%s\n", base64.StdEncoding.EncodeToString(privateKey))
	fmt.Fprintf(stdout, "CTYUN_RELEASE_PUBLIC_KEY=%s\n", base64.StdEncoding.EncodeToString(publicKey))
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
	return ed25519.PrivateKey(key), nil
}

// writeRelease builds archives, writes core-index.json, and signs it.
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
		goos, goarch, err := splitPlatform(platform)
		if err != nil {
			return err
		}
		binaryName := "ctyun"
		if goos == "windows" {
			binaryName = "ctyun.exe"
		}
		buildDir, err := os.MkdirTemp("", "ctyun-release-build-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(buildDir)
		binaryPath := filepath.Join(buildDir, binaryName)
		if err := buildBinary(buildOptions{Version: opts.Version, Channel: opts.Channel, ReleasePublicKey: publicKeyBase64, GOOS: goos, GOARCH: goarch, Output: binaryPath}); err != nil {
			return err
		}
		archiveName := fmt.Sprintf("ctyun_%s_%s_%s.tar.gz", opts.Version, goos, goarch)
		archivePath := filepath.Join(opts.OutDir, archiveName)
		if err := writeArchive(archivePath, binaryPath, binaryName); err != nil {
			return err
		}
		sum, size, err := checksumAndSize(archivePath)
		if err != nil {
			return err
		}
		index.Releases[0].Artifacts = append(index.Releases[0].Artifacts, corerelease.Artifact{
			OS:     goos,
			Arch:   goarch,
			URL:    archiveName,
			SHA256: sum,
			Size:   size,
		})
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
	return copyInstallerScripts(opts.OutDir)
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
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	if err := addFileToArchive(tarWriter, binaryPath, binaryName, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"README.md", "README-EN.md", "LICENCE"} {
		if path, err := projectFilePath(name); err == nil {
			if err := addFileToArchive(tarWriter, path, name, 0o644); err != nil {
				return err
			}
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
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
func checksumAndSize(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
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
