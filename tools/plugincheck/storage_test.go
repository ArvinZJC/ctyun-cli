/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/client"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
	"github.com/ArvinZJC/ctyun-cli/internal/version"
)

// TestStorageProjectionsRetainResults verifies sparse and mixed results against promoted metadata.
func TestStorageProjectionsRetainResults(t *testing.T) {
	// HTTP URIs in these fixtures are XML namespace identifiers, never network destinations.
	//goland:noinspection HttpUrlsUsage
	for _, tc := range []struct {
		product, command, xml, key, want string
		count                            int
	}{
		{"media-storage", "website.show", `<WebsiteConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><RedirectAllRequestsTo><HostName>example.test</HostName><Protocol>https</Protocol></RedirectAllRequestsTo></WebsiteConfiguration>`, "redirect_host", "example.test", 1},
		{"classic-object-storage", "account-summary.show", `<GetAccountSummaryResponse><GetAccountSummaryResult><SummaryMap><entry><key>Users</key><value>392</value></entry><entry><key>Groups</key><value>29</value></entry></SummaryMap></GetAccountSummaryResult></GetAccountSummaryResponse>`, "key", "Users", 2},
		{"classic-object-storage", "user.show", `<GetUserResponse><GetUserResult><User><UserName>untagged</UserName><UserId>7</UserId></User></GetUserResult></GetUserResponse>`, "user_name", "untagged", 1},
		{"classic-object-storage", "user.create", `<CreateUserResponse><CreateUserResult><User><UserName>untagged</UserName></User></CreateUserResult></CreateUserResponse>`, "user_name", "untagged", 1},
		{"classic-object-storage", "entities-for-policy.list", `<ListEntitiesForPolicyResponse><ListEntitiesForPolicyResult><PolicyGroups><member><GroupName>group-only</GroupName></member></PolicyGroups></ListEntitiesForPolicyResult></ListEntitiesForPolicyResponse>`, "group_name", "group-only", 1},
		{"classic-object-storage", "multiple-objects.delete", `<DeleteResult><Error><Key>failed</Key><Code>AccessDenied</Code></Error><Deleted><Key>done</Key></Deleted></DeleteResult>`, "code", "AccessDenied", 2},
		{"media-storage", "multiple-objects.delete", `<DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Error><Key>failed</Key><Code>AccessDenied</Code></Error><Deleted><Key>done</Key></Deleted></DeleteResult>`, "code", "AccessDenied", 2},
		{"media-storage", "versions.list", `<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><DeleteMarker><Key>deleted</Key></DeleteMarker><Version><Key>existing</Key></Version></ListVersionsResult>`, "record_type", "DeleteMarker", 2},
	} {
		t.Run(tc.product+"."+tc.command, func(t *testing.T) {
			bundle, err := plugin.LoadBundle(repoPath(t, filepath.Join("plugins", tc.product)), version.Version)
			if err != nil {
				t.Fatal(err)
			}
			for _, command := range bundle.Commands.Commands {
				if command.ID != tc.product+"."+tc.command {
					continue
				}
				root, err := client.DecodeXML([]byte(tc.xml))
				if err != nil {
					t.Fatal(err)
				}
				rows, err := client.ProjectXML(root, *bundle.Tables.Tables[command.Table].XML)
				if err != nil || len(rows) != tc.count {
					t.Fatalf("rows %#v, %v", rows, err)
				}
				if rows[0][tc.key] != tc.want {
					t.Fatalf("rows %#v", rows)
				}
				return
			}
			t.Fatal("command missing")
		})
	}
}

// TestStorageHeadFixtureExposesMetadata protects the header-only object inspection contract.
func TestStorageHeadFixtureExposesMetadata(t *testing.T) {
	dir := repoPath(t, "plugins/classic-object-storage")
	bundle, err := plugin.LoadBundle(dir, version.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range bundle.Commands.Commands {
		if command.ID != "classic-object-storage.object.head" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, command.FixtureResponse))
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.DecodeFixture(data)
		if err != nil {
			t.Fatal(err)
		}
		result, err := client.DecodeHTTPResponse(response, client.RequestSpec{Response: bundle.APIs.Operations[command.Operation].Response})
		if err != nil {
			t.Fatal(err)
		}
		headers := result.Payload["headers"].(map[string]any)
		for _, key := range []string{"content_length", "e_tag", "last_modified", "content_type"} {
			if headers[key] == nil || headers[key] == "" {
				t.Errorf("missing %s in %#v", key, headers)
			}
		}
		return
	}
	t.Fatal("command missing")
}
