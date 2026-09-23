/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/cli"
)

// TestEVSReviewedMetadataPreservesPublicOptionsAndLabels prevents reviewed EVS
// enum, flag, localized-help, and IOPS-label metadata from regressing.
func TestEVSReviewedMetadataPreservesPublicOptionsAndLabels(t *testing.T) {
	context := loadStorageReviewContext(t, "evs")
	catalog, bundle := context.catalog, context.bundle
	commands := context.commands
	findSourceParameter := context.sourceParameter

	t.Run("finite values", func(t *testing.T) {
		expected := map[string][]string{
			"v4.evs.volume.create.diskMode":                  {"VBD", "ISCSI", "FCSAN"},
			"v4.evs.volume.create.diskType":                  {"SATA", "SAS", "SSD", "FAST-SSD", "XSSD-0", "XSSD-1", "XSSD-2", "XSSD-3"},
			"v4.evs.volume.create.cycleType":                 {"year", "month"},
			"v4.evs.volume.renew.cycleType":                  {"year", "month"},
			"v4.evs.volume.list.diskType":                    {"SATA", "SAS", "SSD", "FAST-SSD", "XSSD-0", "XSSD-1", "XSSD-2", "XSSD-3"},
			"v4.evs.volume.list.diskMode":                    {"VBD", "ISCSI", "FCSAN"},
			"v4.evs.volume.list.diskStatus":                  {"in-use", "available", "diskAttaching", "detaching", "creating", "expired", "freezing"},
			"v4.evs.volume.list.multiAttach":                 {"true", "false"},
			"v4.evs.volume.list.isSystemVolume":              {"true", "false"},
			"v4.evs.volume.list.isEncrypt":                   {"true", "false"},
			"v4.evs.volume.list.deleteDiskWithInstance":      {"true", "false"},
			"v4.evs.snapshot.create.retentionPolicy":         {"custom", "forever"},
			"v4.evs.snapshot.list.snapshotStatus":            {"available", "freezing", "creating", "deleting", "rollbacking", "cloning", "error"},
			"v4.evs.snapshot.list.snapshotType":              {"manu", "timer"},
			"v4.evs.snapshot.list.volumeAttr":                {"data", "system"},
			"v4.evs.snapshot.list.retentionPolicy":           {"forever", "custom"},
			"v4.evs.snapshot.create-volume.diskMode":         {"VBD"},
			"v4.evs.snapshot.create-volume.cycleType":        {"year", "month"},
			"v4.evs.snapshot-policy.set-status.targetStatus": {"activated", "nonactivated"},
		}
		assertStorageParameterEnums(t, context, expected)
	})

	t.Run("acronym flags", func(t *testing.T) {
		expected := map[string][2]string{
			"v4.evs.snapshot.delete.snapshotIDs":                        {"snapshot_ids", "snapshot-ids"},
			"v4.evs.snapshot-policy.associate-volumes.targetDiskIDs":    {"target_disk_ids", "target-disk-ids"},
			"v4.evs.snapshot-policy.disassociate-volumes.targetDiskIDs": {"target_disk_ids", "target-disk-ids"},
			"v4.evs.snapshot-policy.set-status.snapshotPolicyIDs":       {"snapshot_policy_ids", "snapshot-policy-ids"},
		}
		assertStorageParameterNames(t, context, expected)
		for _, command := range bundle.Commands.Commands {
			for _, parameter := range command.Parameters {
				if strings.Contains(parameter.Flag, "-i-d") || strings.Contains(parameter.Name, "_i_d") {
					t.Errorf("%s retains split acronym parameter %s/%s", command.ID, parameter.Name, parameter.Flag)
				}
			}
		}
	})

	t.Run("localized option help", func(t *testing.T) {
		assertStorageLocalizedOptionHelp(t, context)
		constraints := map[string]string{
			"v4.evs.volume.create":                     "official volume-release policy feature matrix",
			"v4.evs.volume.delete":                     "set true when supported snapshots exist",
			"v4.evs.volume.update":                     "default is false",
			"v4.evs.volume.set-snapshot-delete-policy": "whether snapshots are deleted with the volume",
		}
		for operationID, text := range constraints {
			if help := findSourceParameter(operationID, "deleteSnapWithEbs").HelpDescriptions["en-US"]; !strings.Contains(help, text) {
				t.Errorf("%s deleteSnapWithEbs help = %q, want %q", operationID, help, text)
			}
		}
	})

	t.Run("IOPS labels", func(t *testing.T) {
		expected := map[string]string{
			"provisionedIops": "预配置 IOPS",
			"iops":            "云硬盘 IOPS",
			"maxIops":         "最大 IOPS",
			"baselineIops":    "初始 IOPS",
			"iopsPerGb":       "单位容量 IOPS",
		}
		count := 0
		for _, operation := range catalog.Operations {
			command := commands[operation.ID]
			table := bundle.Tables.Tables[command.Table]
			for _, source := range operation.Response.Columns {
				want, ok := expected[source.Path]
				if !ok {
					continue
				}
				count++
				if source.LabelZH != want {
					t.Errorf("%s source label %s = %q, want %q", operation.ID, source.Path, source.LabelZH, want)
				}
				for _, column := range table.Columns {
					if column.Path == source.Path && column.Labels["zh-CN"] != want {
						t.Errorf("%s table label %s = %q, want %q", operation.ID, source.Path, column.Labels["zh-CN"], want)
					}
				}
			}
		}
		if count != 11 {
			t.Fatalf("IOPS column count = %d, want 11", count)
		}
	})

	t.Run("auto-renew show example matches fixture", func(t *testing.T) {
		source := findSourceParameter("v4.evs.volume.auto-renew.show", "diskID")
		var parameterExample string
		if err := json.Unmarshal(source.Example, &parameterExample); err != nil {
			t.Fatalf("decode official diskID parameter example: %v", err)
		}
		if parameterExample != "d5673536-6c77-8ac8-5b73-19a96fd41dca" {
			t.Fatalf("official diskID parameter example = %q", parameterExample)
		}
		command := commands["v4.evs.volume.auto-renew.show"]
		if len(command.Examples) != 1 || !strings.Contains(command.Examples[0], "5dc14c28-3f14-a80c-38d0-e8eb988d4369") {
			t.Fatalf("auto-renew show command example = %#v, want captured response disk ID", command.Examples)
		}
		args := append([]string{"--lang", "en-US", "--table", "plain"}, commandSmokeArgs(t, command)...)
		args = append(args, "--offline")
		var stdout, stderr bytes.Buffer
		if err := cli.Run(cli.Config{Args: args, Stdout: &stdout, Stderr: &stderr, PluginRoot: t.TempDir()}); err != nil {
			t.Fatalf("offline command %q returned error: %v\nstderr:\n%s", strings.Join(args, " "), err, stderr.String())
		}
		if !strings.Contains(stdout.String(), "5dc14c28-3f14-a80c-38d0-e8eb988d4369") {
			t.Fatalf("auto-renew show example does not select the official fixture row:\n%s", stdout.String())
		}
	})
}
