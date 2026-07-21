/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"strings"
	"testing"
)

// TestStoragePhaseTwoPluginsMatchCatalogs pins the approved file-storage
// identities, URI scopes, and exact portal API inventories.
func TestStoragePhaseTwoPluginsMatchCatalogs(t *testing.T) {
	cases := []openAPIPluginExpectation{
		{name: "oceanfs", version: "0.1.0-beta.1", displayNameEN: "OceanFS", productID: 145, revision: "279", endpoint: "https://oceanfs-global.ctapi.ctyun.cn", scope: []string{"/v4/oceanfs/"}, apiIDs: strings.Fields("22432 13429 13430 22431 13428 22433 13431 13432 22435 22434 13435 13442 13455 20842 13453 13452 13445 13444 13443 13441 13439 13438 22438 22437 22436 18706 13437 13440 13454 13436 13434 18708 18705 18707")},
		{name: "hpfs", version: "0.1.0-beta.1", displayNameEN: "High Performance File Storage", productID: 124, revision: "260", endpoint: "https://hpfs-global.ctapi.ctyun.cn", scope: []string{"/v4/hpfs/"}, apiIDs: strings.Fields("19869 17104 17105 17107 17109 19862 19863 19864 19865 19867 19868 17103 20757 20758 20759 20760 20761 21734 21735 21736 21737 17113 11312 11265 11311 11313 11314 14085 14094 17112 17114 17115 17116 17117 17119 17118 17120 17121 17122 11315")},
		{name: "sfs", version: "0.1.0-beta.1", displayNameEN: "Scalable File Service", productID: 37, revision: "15", endpoint: "https://sfs-global.ctapi.ctyun.cn", scope: []string{"/v4/sfs/"}, apiIDs: strings.Fields("12520 8035 8027 9134 16353 16355 8048 9133 8028 12584 8032 16350 16358 8029 8012 16361 12577 12547 8033 16352 16356 8030 8013 12583 8034 16351 8018 8019 16357 8031 8020 8016 12566 8021 8015 12575 12563 8022 8017 12564 8043 8023 12576 8041 8024 8042 8025 15800 12565 8026 12573 16362 16360 13730 16354 8014")},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assertOpenAPIPluginMatchesCatalog(t, testCase)
		})
	}
}
