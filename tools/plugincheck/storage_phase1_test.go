/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"strings"
	"testing"
)

// TestStoragePhaseOnePluginsMatchCatalogs pins the approved EVS and VBS
// identities, URI scopes, and exact portal API inventories.
func TestStoragePhaseOnePluginsMatchCatalogs(t *testing.T) {
	cases := []openAPIPluginExpectation{
		{
			name: "evs", version: "0.1.0-beta.1", displayNameEN: "Elastic Volume Service", productID: 35,
			revision: "45", endpoint: "https://ebs-global.ctapi.ctyun.cn",
			scope:  []string{"/v4/ebs/", "/v4/ebs_snapshot/"},
			apiIDs: strings.Fields("12523 7907 7332 7338 7908 12570 7337 7333 7335 7909 7336 12567 12571 7910 7334 12568 7911 7339 7340 12569 7912 7913 7341 12572 13411 14067 21547 12574 23281 18938"),
		},
		{
			name: "vbs", version: "0.1.0-beta.1", displayNameEN: "Volume Backup Service", productID: 15,
			revision: "93", endpoint: "https://ebsbackup-global.ctapi.ctyun.cn",
			scope:  []string{"/v4/ebs-backup/"},
			apiIDs: strings.Fields("15298 7323 4755 7321 5451 7104 4753 4758 4756 4757 5442 5461 5464 4798 7322 7319 4754 7320 7318 7317 7316 7315 15297 5455 7106 7105 5453 5444 5438 5434 5458 5467 6868 4799 6869 6871 6870"),
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assertOpenAPIPluginMatchesCatalog(t, testCase)
		})
	}
}
