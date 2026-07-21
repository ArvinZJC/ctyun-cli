/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/openapipipeline"
	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestOceanFSReviewedMetadataPreservesPortalSemantics prevents reviewed
// OceanFS option, constraint, fixture, and table metadata from regressing.
func TestOceanFSReviewedMetadataPreservesPortalSemantics(t *testing.T) {
	context := loadStorageReviewContext(t, "oceanfs")
	bundle := context.bundle
	operations, commands := context.operations, context.commands

	t.Run("finite values", func(t *testing.T) {
		expected := map[string][]string{
			"v4.oceanfs.permission-rule.create.rwPermission":   {"rw", "ro"},
			"v4.oceanfs.permission-rule.create.userPermission": {"no_root_squash", "all_squash", "root_squash"},
			"v4.oceanfs.permission-rule.update.rwPermission":   {"rw", "ro"},
			"v4.oceanfs.permission-rule.update.userPermission": {"no_root_squash", "all_squash", "root_squash"},
			"v4.oceanfs.permission-group.create.networkType":   {"private_network"},
			"v4.oceanfs.pricing.quote-renewal.cycleType":       {"year", "month"},
			"v4.oceanfs.pricing.quote-create.sfsType":          {"massive", "massive_perf"},
			"v4.oceanfs.pricing.quote-create.cycleType":        {"year", "month"},
			"v4.oceanfs.file-system.renew.cycleType":           {"year", "month"},
			"v4.oceanfs.file-system.create.sfsType":            {"massive", "massive_perf"},
			"v4.oceanfs.file-system.create.sfsProtocol":        {"nfs", "cifs"},
			"v4.oceanfs.file-system.create.cycleType":          {"year", "month"},
		}
		assertStorageParameterEnums(t, context, expected)
	})

	t.Run("public option names", func(t *testing.T) {
		expected := map[string][2]string{
			"v4.oceanfs.file-system.show.sfsUID":        {"sfs_uid", "sfs-uid"},
			"v4.oceanfs.file-system.list-by-vpc.vpcID":  {"vpc_id", "vpc-id"},
			"v4.oceanfs.file-system-vpc.bind.subnetID":  {"subnet_id", "subnet-id"},
			"v4.oceanfs.replication.create.srcSfsUID":   {"source_sfs_uid", "source-sfs-uid"},
			"v4.oceanfs.replication.create.dstSfsUID":   {"destination_sfs_uid", "destination-sfs-uid"},
			"v4.oceanfs.replication.create.srcRegionID": {"source_region_id", "source-region-id"},
			"v4.oceanfs.replication.create.dstRegionID": {"destination_region_id", "destination-region-id"},
			"v4.oceanfs.file-system.create.vpc":         {"vpc_id", "vpc-id"},
			"v4.oceanfs.file-system.create.subnet":      {"subnet_id", "subnet-id"},
			"v4.oceanfs.pricing.quote-create.cycleCnt":  {"cycle_count", "cycle-count"},
			"v4.oceanfs.pricing.quote-renewal.cycleCnt": {"cycle_count", "cycle-count"},
		}
		assertStorageParameterNames(t, context, expected)
		for _, command := range bundle.Commands.Commands {
			for _, parameter := range command.Parameters {
				for _, unsafe := range []string{"-i-d", "_i_d", "-u-i-d", "_u_i_d", "-v-p-c", "_v_p_c"} {
					if strings.Contains(parameter.Flag, unsafe) || strings.Contains(parameter.Name, unsafe) {
						t.Errorf("%s has split acronym option %s/%s", command.ID, parameter.Name, parameter.Flag)
					}
				}
			}
		}
	})

	t.Run("localized option help", func(t *testing.T) {
		assertStorageLocalizedOptionHelp(t, context)
	})

	t.Run("conditional requirements", func(t *testing.T) {
		expected := map[string][]plugin.ConditionalRequirement{
			"v4.oceanfs.pricing.quote-create": {
				{When: plugin.ParameterCondition{Parameter: "on_demand", Equals: "false"}, Required: []string{"cycle_type", "cycle_count"}},
			},
			"v4.oceanfs.file-system-vpc.bind": {
				{When: plugin.ParameterCondition{Parameter: "is_vpce", Equals: "true"}, Required: []string{"subnet_id"}},
			},
			"v4.oceanfs.file-system.create": {
				{When: plugin.ParameterCondition{Parameter: "on_demand", Equals: "false"}, Required: []string{"cycle_type", "cycle_count"}},
				{When: plugin.ParameterCondition{Parameter: "is_vpce", Equals: "true"}, Required: []string{"subnet_id"}},
			},
		}
		for operationID, want := range expected {
			if got := operations[operationID].ConditionalRequirements; !reflect.DeepEqual(got, want) {
				t.Errorf("%s source conditional requirements = %#v, want %#v", operationID, got, want)
			}
			if got := commands[operationID].ConditionalRequirements; !reflect.DeepEqual(got, want) {
				t.Errorf("%s command conditional requirements = %#v, want %#v", operationID, got, want)
			}
		}
	})

	t.Run("constraint-complete examples", func(t *testing.T) {
		expected := map[string]string{
			"v4.oceanfs.permission-group.update": "ctyun oceanfs permission-group update --permission-group-fuid 5be3d148-12c7-59fb-9b95-be0be60f69af --permission-group-name changename2",
			"v4.oceanfs.permission-rule.list":    "ctyun oceanfs permission-rule list --permission-group-fuid 00fa1424-ba4c-5f9b-be30-65014dc21ab5 --permission-rule-fuid 3d69c5b8-699a-53c4-bd85-d70e93a658eb",
		}
		for operationID, example := range expected {
			want := []string{example}
			if got := operations[operationID].Examples; !reflect.DeepEqual(got, want) {
				t.Errorf("%s source examples = %#v, want %#v", operationID, got, want)
			}
			if got := commands[operationID].Examples; !reflect.DeepEqual(got, want) {
				t.Errorf("%s command examples = %#v, want %#v", operationID, got, want)
			}
		}
	})

	t.Run("mounted client values are visible by default", func(t *testing.T) {
		operationID := "v4.oceanfs.mounted-client.list"
		want := []string{"ip_list", "total_count", "current_count"}
		if got := operations[operationID].Response.DefaultColumns; !reflect.DeepEqual(got, want) {
			t.Errorf("source default columns = %#v, want %#v", got, want)
		}
		if got := bundle.Tables.Tables[commands[operationID].Table].DefaultColumns; !reflect.DeepEqual(got, want) {
			t.Errorf("table default columns = %#v, want %#v", got, want)
		}
	})

	t.Run("used-size charge column collisions", func(t *testing.T) {
		const notice = "是否为按实际使用量计费的文件系统（已废弃，建议使用usedSizeCharge）"
		for _, operationID := range []string{
			"v4.oceanfs.file-system.show",
			"v4.oceanfs.file-system.show-by-name",
			"v4.oceanfs.file-system.list",
			"v4.oceanfs.file-system.list-default-project",
		} {
			operation := operations[operationID]
			table := bundle.Tables.Tables[commands[operationID].Table]
			sourceByKey := make(map[string]openapipipeline.Column, len(operation.Response.Columns))
			for _, column := range operation.Response.Columns {
				if _, duplicate := sourceByKey[column.Key]; duplicate {
					t.Errorf("%s has duplicate source key %s", operationID, column.Key)
				}
				sourceByKey[column.Key] = column
			}
			legacySource, legacyFound := sourceByKey["used_size_charge_legacy"]
			currentSource, currentFound := sourceByKey["used_size_charge"]
			if !legacyFound || legacySource.Path != "used_size_charge" || !currentFound || currentSource.Path != "usedSizeCharge" {
				t.Errorf("%s source used-size columns = legacy %#v current %#v", operationID, legacySource, currentSource)
			}
			if !slices.Contains(operation.Response.DefaultColumns, "used_size_charge") || slices.Contains(operation.Response.DefaultColumns, "used_size_charge_legacy") {
				t.Errorf("%s source defaults = %#v", operationID, operation.Response.DefaultColumns)
			}

			tableByKey := make(map[string]plugin.TableColumn, len(table.Columns))
			for _, column := range table.Columns {
				if _, duplicate := tableByKey[column.Key]; duplicate {
					t.Errorf("%s has duplicate table key %s", operationID, column.Key)
				}
				tableByKey[column.Key] = column
			}
			legacy := tableByKey["used_size_charge_legacy"]
			current := tableByKey["used_size_charge"]
			if legacy.Path != "used_size_charge" || current.Path != "usedSizeCharge" {
				t.Errorf("%s table used-size columns = legacy %#v current %#v", operationID, legacy, current)
			}
			if current.Deprecation != nil {
				t.Errorf("%s current used-size column is deprecated: %#v", operationID, current.Deprecation)
			}
			if legacy.Deprecation == nil || legacy.Deprecation.Notice != notice || legacy.Deprecation.Replacement == nil {
				t.Errorf("%s legacy deprecation = %#v", operationID, legacy.Deprecation)
			} else if got, want := *legacy.Deprecation.Replacement, (plugin.Replacement{Kind: "field", Label: "usedSizeCharge"}); got != want {
				t.Errorf("%s legacy replacement = %#v, want %#v", operationID, got, want)
			}
			if !slices.Contains(table.DefaultColumns, "used_size_charge") || slices.Contains(table.DefaultColumns, "used_size_charge_legacy") {
				t.Errorf("%s table defaults = %#v", operationID, table.DefaultColumns)
			}

			content, err := os.ReadFile(repoPath(t, filepath.Join("plugins", "oceanfs", commands[operationID].FixtureResponse)))
			if err != nil {
				t.Errorf("%s fixture: %v", operationID, err)
				continue
			}
			var fixture any
			if err := json.Unmarshal(content, &fixture); err != nil {
				t.Errorf("%s fixture JSON: %v", operationID, err)
				continue
			}
			for part := range strings.SplitSeq(strings.TrimPrefix(table.RowPath, "$."), ".") {
				fixture = fixture.(map[string]any)[part]
			}
			if rows, ok := fixture.([]any); ok {
				fixture = rows[0]
			}
			row := fixture.(map[string]any)
			for _, path := range []string{"used_size_charge", "usedSizeCharge"} {
				if _, exists := row[path]; !exists {
					t.Errorf("%s fixture lacks %s", operationID, path)
				}
			}
		}

		operationID := "v4.oceanfs.file-system.list-by-vpc"
		table := bundle.Tables.Tables[commands[operationID].Table]
		count := 0
		for _, column := range table.Columns {
			if column.Path == "usedSizeCharge" && column.Key == "used_size_charge" && column.Deprecation == nil {
				count++
			}
			if column.Path == "used_size_charge" || column.Key == "used_size_charge_legacy" {
				t.Errorf("%s unexpectedly contains a legacy used-size column: %#v", operationID, column)
			}
		}
		if count != 1 {
			t.Errorf("%s current used-size column count = %d, want 1", operationID, count)
		}
	})

	t.Run("official fixtures and default fields", func(t *testing.T) {
		assertStorageFixtureDefaults(t, context, storageFixtureOptions{})
	})

	t.Run("technical acronym labels", func(t *testing.T) {
		wrongCasing := []string{"Sfs", "Uid", "Fuid", "Vpc", "Vpce", "Ip", "Az", "Nfs", "Smb", "Cifs"}
		assertStorageTechnicalAcronymLabels(t, context, wrongCasing)
	})

	t.Run("no speculative lifecycle metadata", func(t *testing.T) {
		assertStorageNoCommandLifecycle(t, context)
	})
}
