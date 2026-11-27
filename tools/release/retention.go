/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArvinZJC/ctyun-cli/internal/distribution"
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
)

// archiveRetention identifies current upload inputs and formerly indexed local files eligible for removal.
type archiveRetention struct {
	current    map[string]bool
	superseded []string
}

// planArchiveRetention runs under the output lock. Only a previously signed local reference
// establishes ownership of an existing archive. Missing historical indexes or signatures supply no candidates.
func planArchiveRetention(stage, output, publicKey string) (archiveRetention, error) {
	keep := make(map[string]bool)
	previous := make(map[string]string)
	for _, stem := range []string{"core-index", "index", "gitee/index"} {
		archives, err := signedArchives(stage, stem, publicKey)
		if err != nil {
			return archiveRetention{}, err
		}
		for raw := range archives {
			// The mirror manifest uses the canonical URL's basename as its local upload filename.
			keep[distribution.ArtifactBase(raw)] = true
		}
		if stem == "gitee/index" {
			continue
		}
		archives, err = signedArchives(output, stem, publicKey)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return archiveRetention{}, err
		}
		maps.Copy(previous, archives)
	}
	// The packager creates its stage inside the output directory. Its known
	// lowercase index name lets us query the volume without a separate probe file.
	_, probeErr := os.Lstat(filepath.Join(stage, "CORE-INDEX.JSON"))
	foldCase := !errors.Is(probeErr, fs.ErrNotExist)
	return localArchiveRetention(stage, output, keep, previous, foldCase)
}

// localArchiveRetention matches exact archive names for uploads and accounts for the
// output filesystem's case policy when deciding which existing entries can be retired.
func localArchiveRetention(stage, output string, keep map[string]bool, previous map[string]string, foldCase bool) (archiveRetention, error) {
	lookup := func(name string) string {
		if foldCase {
			return strings.ToLower(name)
		}
		return name
	}
	wanted := make(map[string]string, len(keep))
	for name := range keep {
		key := lookup(name)
		if other, exists := wanted[key]; exists && other != name {
			return archiveRetention{}, fmt.Errorf("release archive names %q and %q collide on this filesystem", other, name)
		}
		wanted[key] = name
	}
	stagedFiles, err := os.ReadDir(stage)
	if err != nil {
		return archiveRetention{}, err
	}
	staged := make(map[string]bool, len(stagedFiles))
	for _, file := range stagedFiles {
		staged[file.Name()] = true
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		return archiveRetention{}, err
	}
	plan := archiveRetention{current: keep}
	for _, entry := range entries {
		name := entry.Name()
		if keep[name] || !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		alias := wanted[lookup(name)]
		if alias != "" && !staged[alias] {
			continue // The current upload reuses this archive in place.
		}
		digest := previous[name]
		if digest != "" && entry.Type().IsRegular() {
			sum, _, err := checksumArchive(filepath.Join(output, name))
			if err != nil {
				return archiveRetention{}, err
			}
			if sum == digest {
				plan.superseded = append(plan.superseded, name)
				continue
			}
		}
		if alias != "" {
			return archiveRetention{}, fmt.Errorf("cannot replace local file %q with indexed archive %q", name, alias)
		}
	}
	return plan, nil
}

// signedArchives reads the archive URLs and checksums authenticated by a local publication index.
func signedArchives(root, stem, publicKey string) (map[string]string, error) {
	base := filepath.Join(root, filepath.FromSlash(stem))
	data, err := os.ReadFile(base + ".json")
	if err != nil {
		return nil, err
	}
	sig, err := os.ReadFile(base + ".sig")
	if err != nil {
		return nil, err
	}
	if err := distribution.VerifyIndexSignature(data, sig, publicKey, stem); err != nil {
		return nil, fmt.Errorf("verify %s.json: %w", stem, err)
	}
	archives := make(map[string]string)
	if stem == "core-index" {
		index, err := corerelease.LoadIndex(data)
		if err != nil {
			return nil, err
		}
		for _, rel := range index.Releases {
			for _, art := range rel.Artifacts {
				archives[art.URL] = art.SHA256
			}
		}
	} else {
		index, err := registry.LoadIndex(data)
		if err != nil {
			return nil, err
		}
		for _, art := range index.Plugins {
			archives[art.URL] = art.SHA256
		}
	}
	return archives, nil
}
