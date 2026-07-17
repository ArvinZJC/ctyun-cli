/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestComputePhaseOnePluginsMatchCatalogs pins the five approved first-class
// product identities and portal API inventories.
func TestComputePhaseOnePluginsMatchCatalogs(t *testing.T) {
	cases := []computePluginExpectation{
		{name: "acs", displayNameEN: "Application Cloud Server", productID: 189, revision: "312", endpoint: "https://ctlite-global.ctapi.ctyun.cn", apiIDs: []string{"18915", "18926", "18917", "18928", "23410", "23413", "23411", "23412"}},
		{name: "cbr", productID: 1, revision: "124", endpoint: "https://ctcbr-global.ctapi.ctyun.cn", apiIDs: []string{"7956", "7958", "7967", "7966", "7965", "7964", "7963", "7962", "7961", "7960", "7959", "14057", "7957", "7955", "7954", "7953", "7952", "7951", "7950", "7949", "7948", "7947"}},
		{name: "ehpc", productID: 8, revision: "100", endpoint: "https://cthpc-global.ctapi.ctyun.cn", apiIDs: []string{"5620", "5623", "13103", "5621", "5624", "6033", "6035", "6034", "6036", "5625", "13093", "13094", "5626", "13087", "13095", "13088", "13097", "13089", "13099", "13091", "13101", "13102", "13086", "5622"}},
		{name: "ims", productID: 23, revision: "83", endpoint: "https://ctimage-global.ctapi.ctyun.cn", apiIDs: []string{"4763", "4764", "4765", "4766", "4767", "5085", "5114", "5225", "5227", "5229", "5230", "5306", "5305", "5585", "5584", "6764", "7488", "7489", "18060", "19685", "22125", "21002", "21418", "22124", "18059", "18058", "18057"}},
		{name: "cdr", productID: 132, revision: "266", endpoint: "https://ctcdr-global.ctapi.ctyun.cn", apiIDs: []string{"13879", "13880", "13881", "13882", "13883", "13884", "13885", "13886", "13887", "13878", "12830", "12848", "12847", "12846", "12845", "12844", "12843", "12842", "12841", "12840", "12839", "12969", "12831", "12832", "12833", "12834", "12835", "12836", "12837", "12838", "13877"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assertComputePluginMatchesCatalog(t, testCase)
		})
	}
}

// TestCDRServiceEnableRendersOfficialResourceRows protects the API 12830
// fixture from regressing to a blank summary row when the response keeps
// order metadata around its resources array.
func TestCDRServiceEnableRendersOfficialResourceRows(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := cli.Run(cli.Config{
		Args: []string{
			"--lang", "en-US", "--table", "plain", "--yes",
			"cdr", "service", "enable",
			"--region", "81f7728662dd11ec810800155d307d5b", "--offline",
		},
		Stdout:     &stdout,
		Stderr:     &stderr,
		PluginRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("run CDR service enable fixture: %v\nstderr:\n%s", err, stderr.String())
	}

	output := stdout.String()
	var officialRow string
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "245e965a91194fc2962bc5edde8993d3") && strings.Contains(line, "4b65909021224618938c33924ff2d2f0") {
			officialRow = line
			break
		}
	}
	if officialRow == "" {
		t.Fatalf("offline table does not contain the official resource row:\n%s", output)
	}
	for _, field := range []string{"| 2 ", "| true ", "| CT_CDR_CLIENT "} {
		if !strings.Contains(officialRow, field) {
			t.Errorf("official resource row does not resolve default-column field %q:\n%s", field, officialRow)
		}
	}
}
