/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package openapipipeline

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// catalogFingerprint returns a stable hash of normalized OpenAPI evidence.
func catalogFingerprint(catalog Catalog) string {
	normalized := catalog
	normalized.Product.SourceRevision = ""
	normalized.Product.SourceURL = ""
	normalized.Product.DisplayNamePolicy = DisplayNamePolicy{}
	data, _ := json.Marshal(normalized)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}
