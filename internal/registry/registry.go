package registry

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

type Index struct {
	Plugins []Artifact `json:"plugins"`
}

type Artifact struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Channel string `json:"channel"`
	Quality string `json:"quality"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

func LoadIndex(raw []byte) (Index, error) {
	var idx Index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return Index{}, fmt.Errorf("parse registry index: %w", err)
	}
	if err := validateIndex(idx); err != nil {
		return Index{}, err
	}
	return idx, nil
}

func validateIndex(idx Index) error {
	for i, artifact := range idx.Plugins {
		prefix := fmt.Sprintf("registry plugin %d", i)
		if artifact.Name == "" {
			return fmt.Errorf("%s is missing name", prefix)
		}
		if !plugin.ValidName(artifact.Name) {
			return fmt.Errorf("%s has invalid plugin name %q", prefix, artifact.Name)
		}
		if artifact.Version == "" {
			return fmt.Errorf("%s %s is missing version", prefix, artifact.Name)
		}
		if !oneOf(artifact.Channel, "stable", "beta", "edge") {
			return fmt.Errorf("%s %s has unsupported channel %q", prefix, artifact.Name, artifact.Channel)
		}
		if !oneOf(artifact.Quality, "generated", "reviewed", "curated") {
			return fmt.Errorf("%s %s has unsupported quality %q", prefix, artifact.Name, artifact.Quality)
		}
		if artifact.URL == "" {
			return fmt.Errorf("%s %s is missing url", prefix, artifact.Name)
		}
		if !validArtifactURL(artifact.URL) {
			return fmt.Errorf("%s %s has invalid artifact url %q", prefix, artifact.Name, artifact.URL)
		}
	}
	return nil
}

func validArtifactURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Scheme != "" {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "\\") {
		return false
	}
	if strings.Contains(raw, "\\") {
		return false
	}
	clean := path.Clean(raw)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "." || part == ".." || part == "" {
			return false
		}
	}
	return true
}

func (i Index) Find(name, channel string) (Artifact, bool) {
	if channel == "" {
		channel = "stable"
	}

	candidates := make([]Artifact, 0)
	for _, artifact := range i.Plugins {
		if artifact.Name != name || artifact.Channel != channel {
			continue
		}
		if channel == "stable" && artifact.Quality != "reviewed" && artifact.Quality != "curated" {
			continue
		}
		candidates = append(candidates, artifact)
	}
	if len(candidates) == 0 {
		return Artifact{}, false
	}

	slices.SortFunc(candidates, func(left, right Artifact) int {
		return compareVersion(right.Version, left.Version)
	})
	return candidates[0], true
}

func (i Index) Search(query, channel string) []Artifact {
	if channel == "" {
		channel = "stable"
	}
	query = strings.ToLower(query)

	latestByName := make(map[string]Artifact)
	for _, artifact := range i.Plugins {
		if artifact.Channel != channel {
			continue
		}
		if channel == "stable" && artifact.Quality != "reviewed" && artifact.Quality != "curated" {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(artifact.Name), query) {
			continue
		}
		current, ok := latestByName[artifact.Name]
		if !ok || compareVersion(artifact.Version, current.Version) > 0 {
			latestByName[artifact.Name] = artifact
		}
	}

	results := make([]Artifact, 0, len(latestByName))
	for _, artifact := range latestByName {
		results = append(results, artifact)
	}
	slices.SortFunc(results, func(left, right Artifact) int {
		if left.Name < right.Name {
			return -1
		}
		if left.Name > right.Name {
			return 1
		}
		return compareVersion(right.Version, left.Version)
	})
	return results
}

func VerifySHA256(path, want string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != want {
		return fmt.Errorf("sha256 mismatch for %s: got %s, want %s", path, got, want)
	}
	return nil
}

func VerifyIndexSignature(index, signature []byte, publicKey string) error {
	if publicKey == "" {
		return fmt.Errorf("HTTP registry index requires a trusted public key")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return fmt.Errorf("decode registry public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("registry public key has length %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil {
		return fmt.Errorf("decode registry signature: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(keyBytes), index, signatureBytes) {
		return fmt.Errorf("registry index signature verification failed")
	}
	return nil
}

func compareVersion(left, right string) int {
	leftParts := parseVersion(left)
	rightParts := parseVersion(right)
	for i := 0; i < len(leftParts); i++ {
		if leftParts[i] < rightParts[i] {
			return -1
		}
		if leftParts[i] > rightParts[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(version string) [3]int {
	var parsed [3]int
	parts := strings.Split(version, ".")
	for i := 0; i < len(parsed) && i < len(parts); i++ {
		value, _ := strconv.Atoi(parts[i])
		parsed[i] = value
	}
	return parsed
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
