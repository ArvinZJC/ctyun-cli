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
	"unicode"

	"github.com/ArvinZJC/ctyun-cli/internal/plugin"
)

// TestSFSReviewedMetadataPreservesPortalSemantics prevents reviewed SFS
// option, constraint, fixture, table, support-boundary, and lifecycle metadata
// from regressing.
func TestSFSReviewedMetadataPreservesPortalSemantics(t *testing.T) {
	context := loadStorageReviewContext(t, "sfs")
	catalog, bundle := context.catalog, context.bundle
	operations, commands := context.operations, context.commands
	findSourceParameter := context.sourceParameter
	findCommandParameter := context.commandParameter

	t.Run("finite values and defaults", func(t *testing.T) {
		enums := map[string][]string{
			"v4.sfs.storage-type.list.sfsType":                {"capacity", "performance"},
			"v4.sfs.pricing.quote-create.cycleType":           {"year", "month"},
			"v4.sfs.pricing.quote-create.volumeType":          {"hdd", "nvme"},
			"v4.sfs.pricing.quote-renewal.cycleType":          {"year", "month"},
			"v4.sfs.permission-group.create.networkType":      {"private_network"},
			"v4.sfs.file-system.create.sfsType":               {"capacity", "performance"},
			"v4.sfs.file-system.create.sfsProtocol":           {"nfs", "cifs"},
			"v4.sfs.file-system.create.cycleType":             {"year", "month"},
			"v4.sfs.file-system.list-by-storage-type.sfsType": {"capacity", "performance"},
			"v4.sfs.file-system.list-by-protocol.sfsProtocol": {"nfs", "cifs"},
			"v4.sfs.file-system.renew.cycleType":              {"year", "month"},
			"v4.sfs.permission-rule.create.rwPermission":      {"rw", "ro"},
			"v4.sfs.permission-rule.create.userPermission":    {"no_root_squash"},
			"v4.sfs.permission-rule.update.rwPermission":      {"rw", "ro"},
			"v4.sfs.permission-rule.update.userPermission":    {"no_root_squash"},
		}
		assertStorageParameterEnums(t, context, enums)
		for _, operation := range catalog.Operations {
			for _, source := range operation.Parameters {
				if source.Name == "pageNo" && source.Default != "1" {
					t.Errorf("%s pageNo default = %q", operation.ID, source.Default)
				}
				if source.Name == "pageSize" && source.Default != "10" {
					t.Errorf("%s pageSize default = %q", operation.ID, source.Default)
				}
			}
		}
		defaults := map[string]string{
			"v4.sfs.ad-domain.join.isAnonymousAcc":                 "false",
			"v4.sfs.ad-domain.update.isAnonymousAcc":               "false",
			"v4.sfs.replication.set-protection.protectSwitch":      "true",
			"v4.sfs.pricing.quote-create.orderNum":                 "1",
			"v4.sfs.file-system.create.isEncrypt":                  "false",
			"v4.sfs.file-system.create.projectID":                  "0",
			"v4.sfs.file-system.create.onDemand":                   "true",
			"v4.sfs.file-system.create.orderNum":                   "1",
			"v4.sfs.file-system.list-by-billing.onDemand":          "true",
			"v4.sfs.permission-rule.create.permissionRulePriority": "1",
			"v4.sfs.permission-rule.update.permissionRulePriority": "1",
			"v4.sfs.file-system.list.projectID":                    "0",
		}
		for key, want := range defaults {
			operationID, target := storageParameterKey(t, key)
			if source := findSourceParameter(operationID, target); source.Default != want || findCommandParameter(operationID, target).Default != want {
				t.Errorf("%s default is not %q", key, want)
			}
		}
	})

	t.Run("region and public option names", func(t *testing.T) {
		expected := map[string][2]string{
			"v4.sfs.mount-point.delete.mountPointID": {"mount_point_id", "mount-point-id"},
			"v4.sfs.ad-domain.join.keytabMd5":        {"keytab_md5", "keytab-md5"},
			"v4.sfs.file-system.create.kmsUUID":      {"kms_uuid", "kms-uuid"},
			"v4.sfs.file-system.create.vpc":          {"vpc_id", "vpc-id"},
			"v4.sfs.file-system.create.subnet":       {"subnet_id", "subnet-id"},
			"v4.sfs.subdirectory.create.subDIR":      {"subdirectory", "subdirectory"},
			"v4.sfs.replication.create.srcSfsUID":    {"source_sfs_uid", "source-sfs-uid"},
			"v4.sfs.replication.create.dstSfsUID":    {"destination_sfs_uid", "destination-sfs-uid"},
			"v4.sfs.replication.create.srcRegionID":  {"source_region_id", "source-region-id"},
			"v4.sfs.replication.create.dstRegionID":  {"destination_region_id", "destination-region-id"},
		}
		assertStorageParameterNames(t, context, expected)
		for _, operation := range catalog.Operations {
			for _, source := range operation.Parameters {
				if source.Name != "regionID" {
					continue
				}
				if source.Profile != "region" || source.CLIName != "" || source.CLIFlag != "" {
					t.Errorf("%s region source = %#v", operation.ID, source)
				}
				parameter := findCommandParameter(operation.ID, source.Name)
				if parameter.Name != "region" || parameter.Flag != "region" || parameter.Required {
					t.Errorf("%s region command parameter = %#v", operation.ID, parameter)
				}
			}
		}
		for _, command := range bundle.Commands.Commands {
			for _, parameter := range command.Parameters {
				for _, unsafe := range []string{"-i-d", "_i_d", "-u-i-d", "_u_i_d", "-v-p-c", "_v_p_c", "-k-m-s", "_k_m_s", "-a-d", "_a_d", "-m-d5", "_m_d5", "-c-i-f-s", "_c_i_f_s", "-n-f-s", "_n_f_s"} {
					if strings.Contains(parameter.Flag, unsafe) || strings.Contains(parameter.Name, unsafe) {
						t.Errorf("%s has split acronym option %s/%s", command.ID, parameter.Name, parameter.Flag)
					}
				}
			}
		}
	})

	t.Run("localized constraint-bearing help", func(t *testing.T) {
		for _, operation := range catalog.Operations {
			command := commands[operation.ID]
			for _, source := range operation.Parameters {
				parameter := findCommandParameter(operation.ID, source.Name)
				key := "parameter." + command.ID + "." + parameter.Name + ".description"
				for _, language := range []string{"en-GB", "en-US", "zh-CN"} {
					help := strings.TrimSpace(source.HelpDescriptions[language])
					if help == "" || strings.Contains(help, "<") || bundle.I18N[language][key] != help {
						t.Errorf("%s %s help is not complete plain localized source text", key, language)
					}
				}
				if source.HelpDescriptions["en-US"] == source.Descriptions["en-US"] || source.HelpDescriptions["en-GB"] == source.Descriptions["en-GB"] {
					t.Errorf("%s retains mechanical English help", key)
				}
				if !strings.ContainsFunc(source.HelpDescriptions["zh-CN"], func(char rune) bool { return unicode.Is(unicode.Han, char) }) {
					t.Errorf("%s zh-CN help is not localized", key)
				}
			}
			for _, language := range []string{"en-GB", "en-US", "zh-CN"} {
				description := operation.Description[language]
				key := "command." + command.ID + ".description"
				if strings.Contains(description, "<") || bundle.I18N[language][key] != description {
					t.Errorf("%s %s command description does not preserve plain source text", operation.ID, language)
				}
			}
		}
	})

	t.Run("support boundaries and cross-field rules", func(t *testing.T) {
		requiredDescriptionText := map[string][3]string{
			"v4.sfs.mount-point.list":        {"only to 3.0 resource pools", "only to 3.0 resource pools", `regionVersion": "v3.0`},
			"v4.sfs.permission-group.create": {"only to 4.0 resource pools", "only to 4.0 resource pools", `regionVersion": "v4.0`},
			"v4.sfs.pricing.quote-create":    {"subscription resources", "subscription resources", "onDemand为false"},
			"v4.sfs.pricing.quote-resize":    {"subscription resources", "subscription resources", "onDemand为false"},
			"v4.sfs.pricing.quote-renewal":   {"subscription resources", "subscription resources", "onDemand为false"},
			"v4.sfs.replication.create":      {"Southwest 1 AZ1 and North China 2 AZ1", "Southwest 1 AZ1 and North China 2 AZ1", "西南1可用区1与华北2可用区 1"},
			"v4.sfs.ad-domain.join":          {"public beta", "public beta", "公测"},
			"v4.sfs.file-system.create":      {"Shanghai 7, Nanjing 3, and Chengdu 4", "Shanghai 7, Nanjing 3, and Chengdu 4", "上海7、南京3、成都4"},
			"v4.sfs.file-system.resize":      {"expansion only", "expansion only", "只支持扩容"},
			"v4.sfs.file-system.renew":       {"subscription file systems", "subscription file systems", "包年包月类型"},
			"v4.sfs.subdirectory.create":     {"East China 1", "East China 1", "仅华东1"},
			"v4.sfs.file-system.show":        {"VPCE share path", "VPCE share path", "vpceSharePath"},
		}
		languages := []string{"en-GB", "en-US", "zh-CN"}
		for operationID, values := range requiredDescriptionText {
			for index, language := range languages {
				if !strings.Contains(operations[operationID].Description[language], values[index]) {
					t.Errorf("%s %s description lacks %q: %q", operationID, language, values[index], operations[operationID].Description[language])
				}
			}
		}
		create := "v4.sfs.file-system.create"
		if subnet := findSourceParameter(create, "subnet"); subnet.Required || findCommandParameter(create, "subnet").Required {
			t.Errorf("version-dependent subnet is unconditionally required")
		}
		mount := "v4.sfs.mount-point.add"
		if subnet := findSourceParameter(mount, "subnetID"); subnet.Required || findCommandParameter(mount, "subnetID").Required {
			t.Errorf("version-dependent mount-point subnet is unconditionally required")
		}
		helpChecks := map[string][]string{
			create + ".subnet":                                   {"required in a 3.0 resource pool", "3.0 资源池中必填"},
			create + ".isEncrypt":                                {"symmetric KMS key", "对称 KMS 密钥"},
			"v4.sfs.ad-domain.update.isAnonymousAcc":             {"at least one", "至少提供一项"},
			"v4.sfs.ad-domain.update.keytab":                     {"together with the Keytab MD5", "必须与 Keytab MD5 同时提供"},
			"v4.sfs.permission-group.update.permissionGroupName": {"at least one", "至少提供一项"},
			"v4.sfs.permission-rule.list.permissionGroupFuid":    {"at least one", "至少提供一项"},
			"v4.sfs.label.update.labelList":                      {"operateType is BIND or UNBIND", "operateType 仅允许 BIND 或 UNBIND"},
		}
		for key, values := range helpChecks {
			operationID, target := storageParameterKey(t, key)
			help := findSourceParameter(operationID, target).HelpDescriptions
			if !strings.Contains(help["en-GB"], values[0]) || !strings.Contains(help["en-US"], values[0]) || !strings.Contains(help["zh-CN"], values[1]) {
				t.Errorf("%s lacks cross-field/support help: %#v", key, help)
			}
		}
		wantConditionals := []plugin.ConditionalRequirement{
			{When: plugin.ParameterCondition{Parameter: "on_demand", Equals: "false"}, Required: []string{"cycle_type", "cycle_count"}},
			{When: plugin.ParameterCondition{Parameter: "is_encrypt", Equals: "true"}, Required: []string{"kms_uuid"}},
		}
		if got := operations[create].ConditionalRequirements; !reflect.DeepEqual(got, wantConditionals) {
			t.Errorf("create source conditionals = %#v, want %#v", got, wantConditionals)
		}
		if got := commands[create].ConditionalRequirements; !reflect.DeepEqual(got, wantConditionals) {
			t.Errorf("create command conditionals = %#v, want %#v", got, wantConditionals)
		}
	})

	t.Run("complete official examples", func(t *testing.T) {
		for _, command := range bundle.Commands.Commands {
			if len(command.Examples) == 0 {
				if commandNeedsPublishedExample(command) {
					t.Errorf("%s has no official example for required command options", command.ID)
				}
				continue
			}
			for _, example := range command.Examples {
				if strings.Contains(example, "{") && !strings.Contains(example, `[{`) {
					t.Errorf("%s retains a placeholder: %q", command.ID, example)
				}
				if err := plugin.ValidateCommandExample(command, example); err != nil {
					t.Errorf("%s example %q: %v", command.ID, example, err)
				}
			}
		}
		for _, operationID := range []string{"v4.sfs.label.batch-bind", "v4.sfs.label.update"} {
			if example := strings.Join(commands[operationID].Examples, "\n"); !strings.Contains(example, `--label-list '[{`) {
				t.Errorf("%s does not preserve the official label array: %q", operationID, example)
			}
		}
		labelUpdate := strings.Join(commands["v4.sfs.label.update"].Examples, "\n")
		for _, value := range []string{`"operateType":"BIND"`, `"operateType":"UNBIND"`} {
			if !strings.Contains(labelUpdate, value) {
				t.Errorf("label update example does not preserve nested %s: %q", value, labelUpdate)
			}
		}
		updates := map[string]string{
			"v4.sfs.file-system.rename":      "--sfs-name",
			"v4.sfs.ad-domain.update":        "--is-anonymous-access",
			"v4.sfs.permission-group.update": "--permission-group-name",
			"v4.sfs.permission-rule.update":  "--auth-address",
			"v4.sfs.file-system.resize":      "--sfs-size",
		}
		for operationID, option := range updates {
			if example := strings.Join(commands[operationID].Examples, "\n"); !strings.Contains(example, option) {
				t.Errorf("%s update example lacks %s: %q", operationID, option, example)
			}
		}
	})

	t.Run("retry and safety semantics", func(t *testing.T) {
		readOnlyPOST := map[string]bool{
			"v4.sfs.pricing.quote-create":  true,
			"v4.sfs.pricing.quote-resize":  true,
			"v4.sfs.pricing.quote-renewal": true,
			"v4.sfs.ad-domain.validate":    true,
		}
		for _, operation := range catalog.Operations {
			wantRetrieve := operation.Method == "GET" || readOnlyPOST[operation.ID]
			if operation.Retryable != wantRetrieve || operation.Dangerous == wantRetrieve {
				t.Errorf("%s retryable/dangerous = %t/%t, want %t/%t", operation.ID, operation.Retryable, operation.Dangerous, wantRetrieve, !wantRetrieve)
			}
		}
	})

	t.Run("official fixtures and default fields", func(t *testing.T) {
		assertStorageFixtureDefaults(t, context, storageFixtureOptions{requireUniqueColumn: true})
	})

	t.Run("technical labels and lifecycle", func(t *testing.T) {
		wrongCasing := []string{"Sfs", "Uid", "Fuid", "Vpc", "Kms", "Ad Domain", "Ad-domain", "Cifs", "Nfs", "Md5", "Vpce", "Ip", "Uuid"}
		deprecated := map[string]map[string]string{
			"v4.sfs.file-system-vpc.list":  {"permission_group_id": "permissionGroupFuid"},
			"v4.sfs.permission-group.list": {"permission_group_id": "permissionGroupFuid"},
			"v4.sfs.permission-rule.list":  {"permission_group_id": "permissionGroupFuid", "permission_rule_id": "permissionRuleFuid"},
			"v4.sfs.subdirectory.create":   {"subdir_name": "subDIR"},
		}
		deprecatedCount := 0
		for _, operation := range catalog.Operations {
			command := commands[operation.ID]
			if command.Deprecation != nil || command.Recommendation != nil || bundle.APIs.Operations[operation.ID].Deprecation != nil {
				t.Errorf("%s has unsupported API/command lifecycle metadata", operation.ID)
			}
			table := bundle.Tables.Tables[command.Table]
			for _, column := range operation.Response.Columns {
				for _, wrong := range wrongCasing {
					if strings.Contains(column.LabelEN, wrong) {
						t.Errorf("%s source label %s has incorrect casing %q", operation.ID, column.Path, column.LabelEN)
					}
				}
			}
			for _, column := range table.Columns {
				for language, label := range column.Labels {
					for _, wrong := range wrongCasing {
						if strings.Contains(label, wrong) {
							t.Errorf("%s %s label %s has incorrect casing %q", operation.ID, language, column.Path, label)
						}
					}
				}
				if column.Deprecation == nil {
					continue
				}
				deprecatedCount++
				wantReplacement, ok := deprecated[operation.ID][column.Key]
				if !ok || column.Deprecation.Status != "deprecated" || column.Deprecation.Replacement == nil || column.Deprecation.Replacement.Kind != "field" || column.Deprecation.Replacement.Label != wantReplacement {
					t.Errorf("unsupported deprecation on %s.%s: %#v", operation.ID, column.Key, column.Deprecation)
				}
				if slices.Contains(table.DefaultColumns, column.Key) {
					t.Errorf("deprecated %s.%s remains visible by default", operation.ID, column.Key)
				}
			}
		}
		if deprecatedCount != 5 {
			t.Errorf("deprecated column count = %d, want 5", deprecatedCount)
		}
	})
}
