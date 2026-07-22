/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestVBSReviewedMetadataPreservesPortalSemantics prevents the reviewed VBS
// enum, option, lifecycle, conditional-input, and table-label metadata from
// regressing after promotion.
func TestVBSReviewedMetadataPreservesPortalSemantics(t *testing.T) {
	context := loadStorageReviewContext(t, "vbs")
	catalog, bundle := context.catalog, context.bundle
	operations, commands := context.operations, context.commands

	t.Run("finite values", func(t *testing.T) {
		expected := map[string][]string{
			"v4.vbs.backup.legacy-list.status":         {"available", "error", "restoring", "creating", "deleting", "merging_backup", "frozen"},
			"v4.vbs.backup.list.backupStatus":          {"available", "error", "restoring", "creating", "deleting", "merging_backup", "frozen"},
			"v4.vbs.repository.legacy-list.status":     {"active", "master_order_creating", "freezing", "expired"},
			"v4.vbs.repository.list.status":            {"active", "master_order_creating", "freezing", "expired"},
			"v4.vbs.policy.list-tasks.sort":            {"createdTime", "completedTime"},
			"v4.vbs.policy.list-tasks.taskStatus":      {"-1", "0", "1"},
			"v4.vbs.task.list.taskStatus":              {"running", "success", "failed", "canceled", "canceling"},
			"v4.vbs.task.list.taskType":                {"1", "2", "3"},
			"v4.vbs.policy.create.status":              {"0", "1"},
			"v4.vbs.policy.create.cycleType":           {"day", "week"},
			"v4.vbs.policy.create.retentionType":       {"num", "date", "all"},
			"v4.vbs.policy.update.cycleType":           {"day", "week"},
			"v4.vbs.policy.update.retentionType":       {"num", "date", "all"},
			"v4.vbs.repository.create.cycleType":       {"MONTH", "YEAR"},
			"v4.vbs.repository.create.autoRenewStatus": {"0", "1"},
			"v4.vbs.repository.renew.cycleType":        {"MONTH", "YEAR"},
		}
		assertStorageParameterEnums(t, context, expected)
	})

	t.Run("public option names", func(t *testing.T) {
		expected := map[string][2]string{
			"v4.vbs.policy.delete.policyIDs":                {"policy_ids", "policy-ids"},
			"v4.vbs.policy.bind-repository.policyIDs":       {"policy_ids", "policy-ids"},
			"v4.vbs.policy.unbind-repository.policyIDs":     {"policy_ids", "policy-ids"},
			"v4.vbs.policy.legacy-bind-volumes.volumeIDs":   {"volume_ids", "volume-ids"},
			"v4.vbs.policy.legacy-unbind-volumes.volumeIDs": {"volume_ids", "volume-ids"},
			"v4.vbs.policy.bind-volumes.diskIDs":            {"disk_ids", "disk-ids"},
			"v4.vbs.policy.unbind-volumes.diskIDs":          {"disk_ids", "disk-ids"},
			"v4.vbs.policy.list-tasks.sort":                 {"resource_sort", "resource-sort"},
			"v4.vbs.repository.list.sort":                   {"resource_sort", "resource-sort"},
			"v4.vbs.repository.legacy-list.sort":            {"resource_sort", "resource-sort"},
		}
		assertStorageParameterNames(t, context, expected)
		for _, command := range bundle.Commands.Commands {
			for _, parameter := range command.Parameters {
				if strings.Contains(parameter.Flag, "-i-d") || strings.Contains(parameter.Name, "_i_d") || parameter.Flag == "sort" {
					t.Errorf("%s has unsafe option name %s/%s", command.ID, parameter.Name, parameter.Flag)
				}
			}
		}
	})

	t.Run("localized option help", func(t *testing.T) {
		assertStorageLocalizedOptionHelp(t, context)
	})

	t.Run("legacy lifecycle", func(t *testing.T) {
		legacyAPIIDs := map[string]bool{"4753": true, "4754": true, "4755": true, "4756": true, "4757": true, "4758": true, "4798": true, "5442": true, "5461": true, "5464": true}
		checked := 0
		for _, operation := range catalog.Operations {
			if !legacyAPIIDs[operation.APIID] {
				continue
			}
			checked++
			if !strings.Contains(operation.Description["zh-CN"], "退役") {
				t.Errorf("%s source description omits portal retirement notice", operation.ID)
			}
			deprecation := bundle.APIs.Operations[operation.ID].Deprecation
			if !deprecation.Active() {
				t.Errorf("%s generated API is not deprecated", operation.ID)
			}
			if deprecation.Replacement != nil {
				t.Errorf("%s generated speculative replacement %#v", operation.ID, deprecation.Replacement)
			}
		}
		if checked != len(legacyAPIIDs) {
			t.Errorf("checked %d legacy operations, want %d", checked, len(legacyAPIIDs))
		}
	})

	t.Run("conditional requirements", func(t *testing.T) {
		for _, operationID := range []string{"v4.vbs.policy.create", "v4.vbs.policy.update", "v4.vbs.repository.create"} {
			if len(operations[operationID].ConditionalRequirements) == 0 {
				t.Errorf("%s source conditional requirements are missing", operationID)
			}
			if got, want := commands[operationID].ConditionalRequirements, operations[operationID].ConditionalRequirements; !reflect.DeepEqual(got, want) {
				t.Errorf("%s conditional requirements = %#v, want %#v", operationID, got, want)
			}
		}
	})

	t.Run("repository billing example", func(t *testing.T) {
		const example = "ctyun vbs repository create --client-token 4cf2962d-e92c-4c00-9181-cfbb2218636c --region 81f7728662dd11ec810800155d307d5b --repository-name test-repo --size 100 --cycle-type MONTH --cycle-count 6 --auto-renew-status 1"
		want := []string{example}
		if got := operations["v4.vbs.repository.create"].Examples; !reflect.DeepEqual(got, want) {
			t.Errorf("repository-create source examples = %#v, want %#v", got, want)
		}
		if got := commands["v4.vbs.repository.create"].Examples; !reflect.DeepEqual(got, want) {
			t.Errorf("repository-create command examples = %#v, want %#v", got, want)
		}
	})

	t.Run("response quirks", func(t *testing.T) {
		expectedColumns := map[string]map[string]string{
			"v4.vbs.backup.list":        {"backup_status": "status"},
			"v4.vbs.policy.legacy-list": {"created_date": "createDate", "resource_ids": "resourceIDs"},
			"v4.vbs.policy.list":        {"binded_disk_ids": "bindedDiskIDs"},
		}
		for operationID, expected := range expectedColumns {
			operation := operations[operationID]
			table := bundle.Tables.Tables[commands[operationID].Table]
			for key, path := range expected {
				sourceFound := false
				for _, column := range operation.Response.Columns {
					if column.Key == key && column.Path == path {
						sourceFound = true
					}
				}
				if !sourceFound {
					t.Errorf("%s source column %s/%s is missing", operationID, key, path)
				}
				tableFound := false
				for _, column := range table.Columns {
					if column.Key == key && column.Path == path {
						tableFound = true
					}
				}
				if !tableFound {
					t.Errorf("%s table column %s/%s is missing", operationID, key, path)
				}
			}
		}
		for _, operation := range catalog.Operations {
			for _, column := range operation.Response.Columns {
				if strings.Contains(column.Key, "_i_ds") {
					t.Errorf("%s retains split acronym table key %s", operation.ID, column.Key)
				}
			}
		}
		for operationID, rowPath := range map[string]string{
			"v4.vbs.backup.legacy-list":     "returnObj.backupList",
			"v4.vbs.policy.legacy-list":     "returnObj.policyList",
			"v4.vbs.repository.legacy-list": "returnObj.repositoryList",
		} {
			if got := operations[operationID].Response.RowPath; got != rowPath {
				t.Errorf("%s row path = %q, want %q", operationID, got, rowPath)
			}
		}
	})

	t.Run("default fixture fields", func(t *testing.T) {
		assertStorageFixtureDefaults(t, context, storageFixtureOptions{skipRootTables: true})
	})

	t.Run("Chinese technical table labels", func(t *testing.T) {
		for _, operation := range catalog.Operations {
			table := bundle.Tables.Tables[commands[operation.ID].Table]
			for _, source := range operation.Response.Columns {
				if !strings.ContainsFunc(source.LabelZH, func(char rune) bool { return unicode.Is(unicode.Han, char) }) {
					continue
				}
				found := false
				for _, column := range table.Columns {
					if column.Path != source.Path {
						continue
					}
					found = true
					label := column.Labels["zh-CN"]
					if !strings.ContainsFunc(label, func(char rune) bool { return unicode.Is(unicode.Han, char) }) {
						t.Errorf("%s table label %s lost Chinese text: %q", operation.ID, source.Path, label)
					}
					for _, token := range []string{"ID", "UUID"} {
						if strings.Contains(source.LabelZH, token) && !strings.Contains(label, token) {
							t.Errorf("%s table label %s lost %s: %q", operation.ID, source.Path, token, label)
						}
					}
				}
				if !found {
					t.Errorf("%s table column %s is missing", operation.ID, source.Path)
				}
			}
		}
	})

	t.Run("English technical table labels", func(t *testing.T) {
		wantLabels := map[string]string{"paas": "PaaS", "pass": "PASS"}
		foundLabels := make(map[string]int, len(wantLabels))
		for _, operation := range catalog.Operations {
			table := bundle.Tables.Tables[commands[operation.ID].Table]
			for _, source := range operation.Response.Columns {
				want, tracked := wantLabels[source.Key]
				if !tracked {
					continue
				}
				foundLabels[source.Key]++
				if source.LabelEN != want {
					t.Errorf("%s source label %s = %q, want %q", operation.ID, source.Path, source.LabelEN, want)
				}
				for _, column := range table.Columns {
					if column.Path == source.Path && column.Labels["en-US"] != want {
						t.Errorf("%s generated en-US label %s = %q, want %q", operation.ID, source.Path, column.Labels["en-US"], want)
					}
				}
			}
		}
		for key := range wantLabels {
			if foundLabels[key] == 0 {
				t.Errorf("VBS source has no %s table columns", key)
			}
		}

		for _, test := range []struct {
			operationID string
			columnKey   string
			want        string
			unwanted    string
		}{
			{operationID: "v4.vbs.backup.list", columnKey: "paas", want: "PaaS", unwanted: "Paas"},
			{operationID: "v4.vbs.backup.legacy-create", columnKey: "pass", want: "PASS", unwanted: "Pass"},
		} {
			command := commands[test.operationID]
			args := []string{"--lang", "en-US", "--table", "plain", "--cols", test.columnKey}
			if command.Dangerous.Confirm != "" {
				args = append(args, "--yes")
			}
			args = append(args, commandSmokeArgs(t, command)...)
			args = append(args, "--offline")
			var stdout, stderr bytes.Buffer
			if err := cli.Run(cli.Config{Args: args, Stdout: &stdout, Stderr: &stderr, PluginRoot: t.TempDir()}); err != nil {
				t.Fatalf("offline command %q returned error: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
			}
			output := stdout.String()
			if !strings.Contains(output, test.want) {
				t.Errorf("offline output for %s omits %q:\n%s", test.operationID, test.want, output)
			}
			if strings.Contains(output, test.unwanted) {
				t.Errorf("offline output for %s contains %q:\n%s", test.operationID, test.unwanted, output)
			}
		}
	})
}
