/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

// menuInventory contains the reviewed ordinary APIs and cross-references grouped by upstream catalog.
type menuInventory struct {
	Products []struct {
		ProductID int `json:"product_id"`
		Services  []struct {
			Versions []struct {
				VersionID  int `json:"version_id"`
				Operations []struct {
					ID   string `json:"api_id"`
					Kind string `json:"kind"`
				} `json:"operations"`
			} `json:"versions"`
		} `json:"services"`
	} `json:"products"`
}

// readCatalogRecord loads the durable review evidence used by the release checks.
func readCatalogRecord(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// TestCatalogMenuReview reconciles retained operations against every published menu,
// including APIs owned by a different product and APIs exclusive to a secondary catalog.
func TestCatalogMenuReview(t *testing.T) {
	root := repoPath(t, "openapi-catalogs")
	var inventory menuInventory
	readCatalogRecord(t, filepath.Join(root, "product-coverage.json"), &inventory)
	var menuCount, entryCount int
	published := make(map[string]bool)
	kafkaMenus := make(map[int]map[string]bool)
	for _, product := range inventory.Products {
		for _, service := range product.Services {
			for _, version := range service.Versions {
				menuCount++
				if version.Operations == nil {
					t.Fatalf("missing menu for product %d catalog %d", product.ProductID, version.VersionID)
				}
				ids := make(map[string]bool)
				for _, op := range version.Operations {
					if op.Kind == "api" {
						entryCount++
						ids[op.ID] = true
						published[op.ID] = true
					}
				}
				if product.ProductID == 48 {
					kafkaMenus[version.VersionID] = ids
				}
			}
		}
	}

	var kafka struct {
		Published int      `json:"published_api_count"`
		Unlisted  []string `json:"retained_unlisted_api_ids"`
		Versions  []struct {
			ID    int `json:"version_id"`
			Count int `json:"count"`
		} `json:"published_versions"`
	}
	readCatalogRecord(t, filepath.Join(root, "kafka", "coverage.json"), &kafka)
	counts := make(map[int]int)
	kafkaIDs := make(map[string]bool)
	for id, menu := range kafkaMenus {
		counts[id] = len(menu)
		for api := range menu {
			kafkaIDs[api] = true
		}
	}
	reportedCounts := make(map[int]int)
	for _, version := range kafka.Versions {
		reportedCounts[version.ID] = version.Count
	}
	if !reflect.DeepEqual(reportedCounts, counts) || kafka.Published != len(kafkaIDs) {
		t.Errorf("Kafka coverage reports %v and %d distinct APIs, menus contain %v and %d", reportedCounts, kafka.Published, counts, len(kafkaIDs))
	}
	type operation struct {
		Plugin string `json:"plugin"`
		ID     string `json:"api_id"`
	}
	var report struct {
		Capture struct {
			Versions  int `json:"versions"`
			Entries   int `json:"ordinary_api_menu_entries"`
			Documents int `json:"unique_ordinary_api_documents"`
		} `json:"capture_counts"`
		Unlisted []operation `json:"retained_unlisted_operations"`
	}
	readCatalogRecord(t, filepath.Join(root, "official-review.json"), &report)
	if menuCount != report.Capture.Versions || entryCount != report.Capture.Entries || len(published) != report.Capture.Documents {
		t.Fatalf("inventory has %d menus, %d ordinary entries and %d distinct APIs; captured counts are %d, %d and %d", menuCount, entryCount, len(published), report.Capture.Versions, report.Capture.Entries, report.Capture.Documents)
	}
	reviewed := make(map[operation]bool)
	for _, op := range report.Unlisted {
		reviewed[op] = true
	}
	kafkaUnlisted := make(map[string]bool)
	for _, id := range kafka.Unlisted {
		kafkaUnlisted[id] = true
	}
	paths, err := filepath.Glob(filepath.Join(root, "*", "baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		name := filepath.Base(filepath.Dir(path))
		var baseline struct {
			Operations []struct {
				ID string `json:"api_id"`
			} `json:"operations"`
		}
		readCatalogRecord(t, path, &baseline)
		for _, api := range baseline.Operations {
			// Native operations use nonnumeric IDs.
			if _, err := strconv.ParseUint(api.ID, 10, 64); err != nil {
				continue
			}
			op := operation{name, api.ID}
			if reviewed[op] == published[api.ID] {
				t.Errorf("%s API %s: listed=%t, recorded as unlisted=%t", name, api.ID, published[api.ID], reviewed[op])
			}
			delete(reviewed, op)
			if name == "kafka" {
				if kafkaUnlisted[api.ID] == kafkaIDs[api.ID] {
					t.Errorf("Kafka coverage misclassifies API %s", api.ID)
				}
				delete(kafkaUnlisted, api.ID)
			}
		}
	}
	if len(reviewed)+len(kafkaUnlisted) != 0 {
		t.Errorf("unlisted entries without promoted operations: %v %v", reviewed, kafkaUnlisted)
	}
}
