/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"strings"
	"testing"
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
			"v4.evs.volume.create":                     "default is false and is supported only in East China 1 and North China 2",
			"v4.evs.volume.delete":                     "must be true in regions that support snapshots",
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
}
