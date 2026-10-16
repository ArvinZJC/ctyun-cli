/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package main

import (
	"github.com/ArvinZJC/ctyun-cli/internal/registry"
	corerelease "github.com/ArvinZJC/ctyun-cli/internal/release"
	coreversion "github.com/ArvinZJC/ctyun-cli/internal/version"
)

// corePublicationIndex adds channel-keyed badge versions to the signed index.
// Runtime readers continue to select artifacts from the embedded release array.
type corePublicationIndex struct {
	corerelease.Index
	Versions map[string]string `json:"versions"`
}

// registryPublicationIndex adds plugin- and channel-keyed badge versions to
// the signed registry without changing runtime artifact selection.
type registryPublicationIndex struct {
	registry.Index
	Versions map[string]map[string]string `json:"versions"`
}

// coreIndexDocument derives the highest semantic version for each channel
// from the final merged releases, independently of their array order.
func coreIndexDocument(index corerelease.Index) corePublicationIndex {
	versions := make(map[string]string)
	for _, rel := range index.Releases {
		if current, ok := versions[rel.Channel]; !ok || coreversion.CompareSemanticVersions(rel.Version, current) > 0 {
			versions[rel.Channel] = rel.Version
		}
	}
	return corePublicationIndex{Index: index, Versions: versions}
}

// registryIndexDocument derives each plugin's highest semantic version per
// channel from the final merged artifacts, including retained channels.
func registryIndexDocument(index registry.Index) registryPublicationIndex {
	versions := make(map[string]map[string]string)
	for _, artifact := range index.Plugins {
		channels := versions[artifact.Name]
		if channels == nil {
			channels = make(map[string]string)
			versions[artifact.Name] = channels
		}
		if current, ok := channels[artifact.Channel]; !ok || coreversion.CompareSemanticVersions(artifact.Version, current) > 0 {
			channels[artifact.Channel] = artifact.Version
		}
	}
	return registryPublicationIndex{Index: index, Versions: versions}
}
