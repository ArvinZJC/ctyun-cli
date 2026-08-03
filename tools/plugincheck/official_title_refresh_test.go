/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestECSPluginMatchesCurrentCatalog pins the current public ECS API inventory
// after the legacy API group disappeared from the official documentation.
func TestECSPluginMatchesCurrentCatalog(t *testing.T) {
	assertOpenAPIPluginMatchesCatalog(t, openAPIPluginExpectation{
		name:      "ecs",
		version:   "0.1.0-beta.4",
		productID: 25,
		revision:  "81",
		endpoint:  "https://ctecs-global.ctapi.ctyun.cn",
		scope:     []string{"/v4/ecs/", "/global-trust-authority/"},
		apiIDs: strings.Fields(
			"8315 8312 8316 8324 8320 8321 8313 13721 8314 8336 8332 8133 8330 8354 8353 8329 8343 8339 6919 6920 6922 6914 6916 6924 7974 6921 6923 6918 6910 6907 6909 6908 6912 16484 16080 22250 22252 22254 22255 22249 22248 22251 22337 22256 22253 22267 14312 15066 6201 6198 21468 8327 8325 13078 21469 16070 8326 21976 4826 5076 5613 6199 5614 5616 8282 5548 20054 21938 17816 8305 8306 8288 13850 8303 8304 8286 8134 19684 21413 8281 17817 18910 9268 15572 8307 21414 8308 8309 13079 6138 17814 23282 5543 21519 8298 8302 18629 8287 8283 13849 8310 5607 5611 23416 23415 8296 6133 8297 22331 22328 8284 8285 21415 13241 21516 8293 8292 8295 6186 6178 6188 6182 6187 6180 6189 6184 9271 8341 8344 5605 8340 8345 8342 20327 21467 17815 9265 8323 8319 8322 9607 5323 5321 5325 5318 5319 5326 5315 5317 5324 5322 5320 15581 8350 8352 8132 8349 8346 12852 8347 8351 9594 9588 9692 9579 9591 9605 9604 9600 9603 9598 9597 9549 15754 20200 20209 20203 20206 20207 20211 22327 8311 8289 11993 11960 11994 11991 11992 8290 12225 8291 11979 5557 5564 5562 5561 5574 5571 5573 5567 5565 5559 5570 5568 22092 22093 22095 22094 22091"),
	})
}

// TestOfficialPluginTitlesMatchCurrentDocumentation pins public page titles
// separately from concise generated command descriptions.
func TestOfficialPluginTitlesMatchCurrentDocumentation(t *testing.T) {
	cases := []struct {
		plugin    string
		apiID     string
		wantTitle string
	}{
		{plugin: "region", apiID: "5851", wantTitle: "资源池列表查询"},
		{plugin: "region", apiID: "5860", wantTitle: "资源池概况信息查询"},
		{plugin: "region", apiID: "5855", wantTitle: "资源池可用区查询"},
		{plugin: "region", apiID: "5853", wantTitle: "资源池产品信息查询"},
		{plugin: "region", apiID: "7083", wantTitle: "资源池产品可售状态查询"},
		{plugin: "ehpc", apiID: "13103", wantTitle: "批量修改用户密码"},
		{plugin: "ims", apiID: "5585", wantTitle: "弃用私有镜像"},
		{plugin: "ims", apiID: "5584", wantTitle: "取消弃用私有镜像"},
	}

	for _, testCase := range cases {
		t.Run(testCase.plugin+"/"+testCase.apiID, func(t *testing.T) {
			catalog := readProductCatalog(t, repoPath(t, filepath.Join("openapi-catalogs", testCase.plugin, "baseline.json")))
			for _, operation := range catalog.Operations {
				if operation.APIID != testCase.apiID {
					continue
				}
				if operation.Title != testCase.wantTitle {
					t.Fatalf("title = %q, want %q", operation.Title, testCase.wantTitle)
				}
				return
			}
			t.Fatalf("API %s is missing", testCase.apiID)
		})
	}

	imsCatalog := readProductCatalog(t, repoPath(t, filepath.Join("openapi-catalogs", "ims", "baseline.json")))
	imsBundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", "ims")), version.Version)
	if err != nil {
		t.Fatalf("load IMS plugin: %v", err)
	}
	for _, apiID := range []string{"5585", "5584"} {
		var operationID string
		for _, operation := range imsCatalog.Operations {
			if operation.APIID == apiID {
				operationID = operation.ID
				break
			}
		}
		if operationID == "" {
			t.Fatalf("IMS API %s is missing", apiID)
		}
		if deprecation := imsBundle.APIs.Operations[operationID].Deprecation; deprecation != nil {
			t.Errorf("IMS API %s operation deprecation = %#v", apiID, deprecation)
		}
		for _, command := range imsBundle.Commands.Commands {
			if command.Operation == operationID && command.Deprecation != nil {
				t.Errorf("IMS API %s command deprecation = %#v", apiID, command.Deprecation)
			}
		}
	}
}
