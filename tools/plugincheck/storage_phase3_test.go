/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"strings"
	"testing"
)

// TestStoragePhaseThreePluginMatchesCatalog pins the approved ZOS identity,
// two-prefix URI scope, and exact portal API inventory.
func TestStoragePhaseThreePluginMatchesCatalog(t *testing.T) {
	assertOpenAPIPluginMatchesCatalog(t, openAPIPluginExpectation{
		name: "zos", version: "0.1.0-beta.1", displayNameEN: "ZOS", productID: 9, revision: "99",
		endpoint: "https://zos-global.ctapi.ctyun.cn",
		scope:    []string{"/v4/oss/", "/v4/zms/"},
		apiIDs:   strings.Fields("6246 9262 9025 9024 9023 9022 9021 9016 9013 7906 7904 7903 7902 7358 7116 7115 7114 6925 6304 6301 6299 6298 6292 6291 6290 6288 6287 6286 6282 6281 6278 6277 6275 6273 6271 6268 6267 6266 6265 6262 6259 6258 6255 6254 6253 6251 6249 6247 5579 5575 5572 5569 5566 5560 5558 5555 5547 5542 5538 5534 5531 5524 7357 5523 5522 6293 5525 6302 14102 6263 5520 5518 5516 5515 9026 7905 6284 17820 17821 17822 17823 17824 17825 17826 17827 17828 17829 17831 17833 17836 17837 17838 17882 17881 16485 5514 7359 7355 7356 9845 19872 19873 22332 22333 22334 22335"),
	})
}
