/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
)

// TestNetworkProductOwnership prevents shared documentation links from duplicating commands.
func TestNetworkProductOwnership(t *testing.T) {
	owners := map[string]string{}
	for _, name := range []string{"vpc", "eip", "elb", "nat", "private-nat"} {
		data, err := os.ReadFile(repoPath(t, filepath.Join("openapi-catalogs", name, "baseline.json")))
		if err != nil {
			t.Fatal(err)
		}
		var c openapipipeline.Catalog
		if err := json.Unmarshal(data, &c); err != nil {
			t.Fatal(err)
		}
		for _, op := range c.Operations {
			if owner, ok := owners[op.APIID]; ok {
				t.Fatalf("API %s belongs to both %s and %s", op.APIID, owner, name)
			}
			owners[op.APIID] = name
		}
	}
	if len(owners) != 407 {
		t.Fatalf("network API inventory=%d, want407", len(owners))
	}
}
