/*
 * Copyright (c) 2026 IsArvin.
 * This file is part of ctyun-cli. Please refer to the LICENCE file for licence information.
 */

package plugincheck

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestHPFSReviewedMetadataPreservesPortalSemantics prevents reviewed HPFS
// option, constraint, fixture, table, and lifecycle metadata from regressing.
func TestHPFSReviewedMetadataPreservesPortalSemantics(t *testing.T) {
	context := loadStorageReviewContext(t, "hpfs")
	catalog, bundle := context.catalog, context.bundle
	operations, commands := context.operations, context.commands
	findSourceParameter := context.sourceParameter
	findCommandParameter := context.commandParameter

	t.Run("finite values", func(t *testing.T) {
		expected := map[string][]string{
			"v4.hpfs.dataflow.create.importDataType":              {"data", "metadata"},
			"v4.hpfs.dataflow.create.exportDataType":              {"data"},
			"v4.hpfs.dataflow.create.importTrigger":               {"new"},
			"v4.hpfs.dataflow.create.exportTrigger":               {"new", "changed"},
			"v4.hpfs.dataflow-task.create.taskType":               {"import_data", "import_metadata", "export_data"},
			"v4.hpfs.dataflow-task.list.taskStatus":               {"creating", "executing", "completed", "canceling", "fail"},
			"v4.hpfs.dataflow-task.list.taskType":                 {"import_data", "import_metadata", "export_data"},
			"v4.hpfs.protocol-service.create.protocolSpec":        {"general"},
			"v4.hpfs.protocol-service.create.protocolType":        {"nfs"},
			"v4.hpfs.protocol-service.create.ipVersion":           {"0", "1", "2"},
			"v4.hpfs.protocol-service.list.protocolServiceStatus": {"creating", "available", "deleting", "create_fail", "agent_err"},
			"v4.hpfs.protocol-service.list.protocolSpec":          {"general"},
			"v4.hpfs.protocol-service.list.protocolType":          {"nfs"},
			"v4.hpfs.fileset.list.filesetStatus":                  {"available", "unusable"},
			"v4.hpfs.file-system.create.sfsType":                  {"hpfs_perf"},
			"v4.hpfs.file-system.create.sfsProtocol":              {"hpfs"},
			"v4.hpfs.file-system.create.cycleType":                {"year", "month"},
			"v4.hpfs.cluster.list.sfsType":                        {"hpfs_perf"},
			"v4.hpfs.baseline.list.sfsType":                       {"hpfs_perf"},
			"v4.hpfs.file-system.list-by-storage-type.sfsType":    {"hpfs_perf"},
			"v4.hpfs.file-system.list.sfsStatus":                  {"creating", "available", "expired", "unusable"},
			"v4.hpfs.file-system.list.sfsProtocol":                {"hpfs"},
		}
		assertStorageParameterEnums(t, context, expected)
	})

	t.Run("documented defaults", func(t *testing.T) {
		expected := map[string]string{
			"v4.hpfs.protocol-service.create.isVpce":             "false",
			"v4.hpfs.protocol-service.create.ipVersion":          "0",
			"v4.hpfs.file-system.create.projectID":               "0",
			"v4.hpfs.file-system.create.onDemand":                "true",
			"v4.hpfs.file-system.create.orderNum":                "1",
			"v4.hpfs.directory.create.sfsDirectoryMode":          "755",
			"v4.hpfs.directory.create.sfsDirectoryUID":           "0",
			"v4.hpfs.directory.create.sfsDirectoryGID":           "0",
			"v4.hpfs.file-system.list-by-cluster.projectID":      "0",
			"v4.hpfs.file-system.list-by-storage-type.projectID": "0",
		}
		for key, value := range expected {
			operationID, target := storageParameterKey(t, key)
			if got := findSourceParameter(operationID, target).Default; got != value {
				t.Errorf("%s source default = %q, want %q", key, got, value)
			}
			if got := findCommandParameter(operationID, target).Default; got != value {
				t.Errorf("%s command default = %q, want %q", key, got, value)
			}
		}
		for _, operation := range catalog.Operations {
			for _, source := range operation.Parameters {
				if source.Name != "pageNo" && source.Name != "pageSize" {
					continue
				}
				want := map[string]string{"pageNo": "1", "pageSize": "10"}[source.Name]
				if source.Default != want || findCommandParameter(operation.ID, source.Name).Default != want {
					t.Errorf("%s.%s does not preserve page default %s", operation.ID, source.Name, want)
				}
			}
		}
	})

	t.Run("public option names", func(t *testing.T) {
		expected := map[string][2]string{
			"v4.hpfs.protocol-service.show.protocolServiceID": {"protocol_service_id", "protocol-service-id"},
			"v4.hpfs.dataflow.show.dataflowID":                {"dataflow_id", "dataflow-id"},
			"v4.hpfs.dataflow-task.show.taskID":               {"task_id", "task-id"},
			"v4.hpfs.fileset.show.filesetID":                  {"fileset_id", "fileset-id"},
			"v4.hpfs.cluster.list.ebmDeviceType":              {"ebm_device_type", "ebm-device-type"},
			"v4.hpfs.file-system.create.vpc":                  {"vpc_id", "vpc-id"},
			"v4.hpfs.file-system.create.subnet":               {"subnet_id", "subnet-id"},
			"v4.hpfs.directory.create.sfsDirectory":           {"directory", "directory"},
			"v4.hpfs.directory.create.sfsDirectoryUID":        {"directory_uid", "directory-uid"},
			"v4.hpfs.directory.create.sfsDirectoryGID":        {"directory_gid", "directory-gid"},
		}
		assertStorageParameterNames(t, context, expected)
		for _, command := range bundle.Commands.Commands {
			for _, parameter := range command.Parameters {
				for _, unsafe := range []string{"-i-d", "_i_d", "-u-i-d", "_u_i_d", "-v-p-c", "_v_p_c", "-h-p-f-s", "_h_p_f_s"} {
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

	t.Run("cross-field and support-boundary help", func(t *testing.T) {
		const dataflowOperation = "v4.hpfs.dataflow.create"
		exactlyOne := map[string]string{
			"en-GB": "Exactly one of automatic import and automatic export must be enabled; enabling both or neither is invalid.",
			"en-US": "Exactly one of automatic import and automatic export must be enabled; enabling both or neither is invalid.",
			"zh-CN": "自动导入和自动导出必须且只能开启一个；同时开启或同时关闭均无效。",
		}
		for _, target := range []string{"autoImport", "autoExport"} {
			source := findSourceParameter(dataflowOperation, target)
			parameter := findCommandParameter(dataflowOperation, target)
			key := "parameter." + commands[dataflowOperation].ID + "." + parameter.Name + ".description"
			for language, requiredText := range exactlyOne {
				if help := source.HelpDescriptions[language]; !strings.Contains(help, requiredText) {
					t.Errorf("%s source %s help does not preserve exactly-one rule: %q", target, language, help)
				}
				if help := bundle.I18N[language][key]; !strings.Contains(help, requiredText) {
					t.Errorf("%s generated %s help does not preserve exactly-one rule: %q", target, language, help)
				}
			}
		}

		const protocolOperation = "v4.hpfs.protocol-service.create"
		ipVersion := findSourceParameter(protocolOperation, "ipVersion")
		ipParameter := findCommandParameter(protocolOperation, "ipVersion")
		ipKey := "parameter." + commands[protocolOperation].ID + "." + ipParameter.Name + ".description"
		unsupported := map[string]string{
			"en-GB": "Pure IPv6 and dual-stack endpoint creation are currently unsupported.",
			"en-US": "Pure IPv6 and dual-stack endpoint creation are currently unsupported.",
			"zh-CN": "当前不支持创建纯 IPv6 或双栈终端节点。",
		}
		for language, requiredText := range unsupported {
			if help := ipVersion.HelpDescriptions[language]; !strings.Contains(help, requiredText) {
				t.Errorf("ipVersion source %s help lacks support boundary: %q", language, help)
			}
			if help := bundle.I18N[language][ipKey]; !strings.Contains(help, requiredText) {
				t.Errorf("ipVersion generated %s help lacks support boundary: %q", language, help)
			}
		}

		isVpce := findSourceParameter(protocolOperation, "isVpce")
		isVpceParameter := findCommandParameter(protocolOperation, "isVpce")
		isVpceKey := "parameter." + commands[protocolOperation].ID + "." + isVpceParameter.Name + ".description"
		for _, language := range []string{"en-GB", "en-US"} {
			const requiredText = "This option applies only to 4.0 resource pools."
			if help := isVpce.HelpDescriptions[language]; !strings.Contains(help, requiredText) {
				t.Errorf("isVpce source %s help lacks 4.0 applicability: %q", language, help)
			}
			if help := bundle.I18N[language][isVpceKey]; !strings.Contains(help, requiredText) {
				t.Errorf("isVpce generated %s help lacks 4.0 applicability: %q", language, help)
			}
		}
		for _, help := range []string{isVpce.HelpDescriptions["zh-CN"], bundle.I18N["zh-CN"][isVpceKey]} {
			if !strings.Contains(help, "仅4.0资源池生效") {
				t.Errorf("isVpce zh-CN help lacks 4.0 applicability: %q", help)
			}
		}
	})

	t.Run("conditional requirements", func(t *testing.T) {
		expected := map[string][]plugin.ConditionalRequirement{
			"v4.hpfs.dataflow.create": {
				{When: plugin.ParameterCondition{Parameter: "auto_import", Equals: "true"}, Required: []string{"import_data_type", "import_trigger"}},
				{When: plugin.ParameterCondition{Parameter: "auto_export", Equals: "true"}, Required: []string{"export_data_type", "export_trigger"}},
			},
			"v4.hpfs.protocol-service.create": {
				{When: plugin.ParameterCondition{Parameter: "is_vpce", Equals: "true"}, Required: []string{"subnet_id"}},
			},
			"v4.hpfs.file-system.create": {
				{When: plugin.ParameterCondition{Parameter: "on_demand", Equals: "false"}, Required: []string{"cycle_type", "cycle_count"}},
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

	t.Run("complete official examples", func(t *testing.T) {
		for _, command := range bundle.Commands.Commands {
			if len(command.Examples) == 0 {
				if commandNeedsPublishedExample(command) {
					t.Errorf("%s has no generated official example for required command options", command.ID)
				}
				continue
			}
			for _, example := range command.Examples {
				if strings.Contains(example, "{") && !strings.Contains(example, `[{`) {
					t.Errorf("%s example retains a placeholder: %q", command.ID, example)
				}
				if err := plugin.ValidateCommandExample(command, example); err != nil {
					t.Errorf("%s example %q: %v", command.ID, example, err)
				}
			}
		}
		for _, operationID := range []string{"v4.hpfs.label.update", "v4.hpfs.file-system.list-by-label", "v4.hpfs.file-system.create"} {
			example := strings.Join(commands[operationID].Examples, "\n")
			if !strings.Contains(example, `--label-list '[{`) {
				t.Errorf("%s does not preserve its captured label array: %q", operationID, example)
			}
		}
		expectedUpdates := map[string]string{
			"v4.hpfs.dataflow.update": "ctyun hpfs dataflow update --region 81f7728662dd11ec810800155d307d5b --dataflow-id dataflow-j9Kn2B --auto-sync true --dataflow-description 'this is the test dataflow strategy'",
			"v4.hpfs.fileset.update":  "ctyun hpfs fileset update --region 81f7728662dd11ec810800155d307d5b --fileset-id fset-nkh6qmv3iaf --capacity-quota 50",
		}
		for operationID, example := range expectedUpdates {
			want := []string{example}
			if got := operations[operationID].Examples; !reflect.DeepEqual(got, want) {
				t.Errorf("%s source examples = %#v, want %#v", operationID, got, want)
			}
			if got := commands[operationID].Examples; !reflect.DeepEqual(got, want) {
				t.Errorf("%s command examples = %#v, want %#v", operationID, got, want)
			}
			if err := plugin.ValidateCommandExample(commands[operationID], example); err != nil {
				t.Errorf("%s reviewed update example: %v", operationID, err)
			}
		}
	})

	t.Run("region retry and safety semantics", func(t *testing.T) {
		for _, operation := range catalog.Operations {
			for _, parameter := range operation.Parameters {
				if parameter.Name == "regionID" && (parameter.Profile != "region" || parameter.CLIName != "" || parameter.CLIFlag != "") {
					t.Errorf("%s region mapping = profile %q name %q flag %q", operation.ID, parameter.Profile, parameter.CLIName, parameter.CLIFlag)
				}
			}
			wantRetrieve := operation.Method == "GET" || operation.ID == "v4.hpfs.file-system.list-by-label"
			if operation.Retryable != wantRetrieve || operation.Dangerous == wantRetrieve {
				t.Errorf("%s retryable/dangerous = %t/%t, want %t/%t", operation.ID, operation.Retryable, operation.Dangerous, wantRetrieve, !wantRetrieve)
			}
		}
	})

	t.Run("official fixtures and default fields", func(t *testing.T) {
		assertStorageFixtureDefaults(t, context, storageFixtureOptions{requireUniqueColumn: true})
	})

	t.Run("technical acronym labels", func(t *testing.T) {
		wrongCasing := []string{"Hpfs", "Hpc", "Fileset", "Sfs", "Uid", "Gid", "Vpc", "Vpce", "Ip", "Ipv4", "Ipv6", "Ebm", "Kms", "Uuid"}
		assertStorageTechnicalAcronymLabels(t, context, wrongCasing)
	})

	t.Run("evidence-backed lifecycle metadata", func(t *testing.T) {
		assertStorageNoCommandLifecycle(t, context)
		deprecatedColumns := 0
		for _, command := range bundle.Commands.Commands {
			for _, column := range bundle.Tables.Tables[command.Table].Columns {
				if column.Deprecation == nil {
					continue
				}
				deprecatedColumns++
				if command.Operation != "v4.hpfs.file-system.list" || column.Key != "mount_count" || column.Path != "mountCount" {
					t.Errorf("unsupported deprecated field on %s: %#v", command.Operation, column)
				} else if column.Deprecation.Status != "deprecated" || column.Deprecation.Notice != "挂载点数量(废弃)" || column.Deprecation.Replacement != nil {
					t.Errorf("mount-count deprecation = %#v", column.Deprecation)
				}
			}
		}
		if deprecatedColumns != 1 {
			t.Errorf("deprecated column count = %d, want 1", deprecatedColumns)
		}
		if defaults := operations["v4.hpfs.file-system.list"].Response.DefaultColumns; slices.Contains(defaults, "mount_count") {
			t.Errorf("deprecated mount_count is visible by default: %#v", defaults)
		}
	})
}
