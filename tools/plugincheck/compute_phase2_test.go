/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestComputePhaseTwoPluginsMatchCatalogs pins the three approved first-class
// product identities and portal API inventories.
func TestComputePhaseTwoPluginsMatchCatalogs(t *testing.T) {
	cases := []openAPIPluginExpectation{
		{name: "cf", version: "0.1.0-beta.2", productID: 53, revision: "40", endpoint: "https://cf-global.ctapi.ctyun.cn", scope: []string{"/openapi/v1/"}, apiIDs: []string{"16013", "16014", "16015", "16006", "16017", "16039", "16038", "16043", "16044", "16005", "16018", "16019", "16021", "16022", "16023", "16024", "16025", "16026", "16027", "15971", "15968", "15969", "15970", "15995", "15999", "15997", "15998", "16032", "16033", "16034", "16523", "16035", "16036", "16037", "16028", "16029", "16030", "16031", "16040", "16041", "16042", "16016", "16000", "16001", "16002", "16003", "16004", "16009", "19963", "19964", "19965", "19966", "19967", "19968", "19969", "19970", "19971", "19972", "19973", "19974", "19975", "19976"}},
		{name: "as", version: "0.1.0-beta.2", productID: 19, revision: "87", endpoint: "https://scaling-global.ctapi.ctyun.cn", scope: []string{"/v4/scaling/"}, apiIDs: []string{"12710", "4991", "5007", "5070", "5092", "5079", "5078", "5080", "5091", "5138", "5073", "5000", "5093", "5094", "4990", "4992", "5067", "12651", "12648", "12649", "12650", "5068", "4995", "4994", "4184", "5005", "5006", "7103", "13237", "13238", "13239", "4773", "5095", "4996", "5081", "5072", "4997", "4999", "4998", "4774", "4775", "5084", "5001", "5002", "5083", "5074", "5082", "5003", "4761", "12652", "5008", "4772", "5077", "5071", "5136", "5135", "4776", "4762", "4989", "5004", "4993", "5069"}},
		{name: "dps", version: "0.1.0-beta.2", productID: 16, revision: "91", endpoint: "https://ebm-global.ctapi.ctyun.cn", scope: []string{"/v4/ebm/"}, apiIDs: []string{"6940", "6942", "4908", "6941", "4574", "4910", "4912", "4575", "4579", "4576", "4580", "4577", "4581", "5117", "5308", "4582", "4584", "5116", "5178", "5118", "5179", "5119", "5880", "5120", "5121", "5881", "5122", "5123", "5307", "9885", "10243", "9886", "10244", "9888", "9884", "9887", "22338", "22339", "21944", "18905", "18906", "18907", "18908", "19687", "20164", "19688", "20165", "19690", "23200", "16489", "17282", "21940", "21939", "15568", "22761", "23495", "23494"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assertOpenAPIPluginMatchesCatalog(t, testCase)
		})
	}
}

// TestDPSAutoRenewCommandsUseGroupedPaths keeps the visible command surface
// aligned with the repository's noun-group/action convention.
func TestDPSAutoRenewCommandsUseGroupedPaths(t *testing.T) {
	context := loadStorageReviewContext(t, "dps")
	wants := map[string][]string{
		"v4.dps.instance.get-auto-renew-config":    {"dps", "instance", "auto-renew", "show", "{instance_uuid}"},
		"v4.dps.instance.update-auto-renew-config": {"dps", "instance", "auto-renew", "update"},
	}
	for operationID, want := range wants {
		if got := context.commands[operationID].Path; !slices.Equal(got, want) {
			t.Errorf("%s command path = %#v, want %#v", operationID, got, want)
		}
	}
}

// TestDPSOptionHelpUsesSharedDefaultFormatting verifies source prose does not
// repeat the default value rendered from command metadata.
func TestDPSOptionHelpUsesSharedDefaultFormatting(t *testing.T) {
	var stdout bytes.Buffer
	if err := cli.Run(cli.Config{
		Args:       []string{"--lang", "en-US", "help", "dps", "super-pod", "stock", "list"},
		Stdout:     &stdout,
		PluginRoot: t.TempDir(),
	}); err != nil {
		t.Fatalf("render DPS option help: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Required stock count (default: 1)") {
		t.Fatalf("DPS default help is not shared-formatted:\n%s", got)
	}
	if strings.Contains(strings.ToLower(got), "default is 1") {
		t.Fatalf("DPS option prose repeats its default:\n%s", got)
	}
}
